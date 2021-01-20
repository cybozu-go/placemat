package virtualbmc

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/gin-gonic/gin"
)

// Machine defines the interface to manipulate Machine
type Machine interface {
	PowerStatus() PowerStatus
	PowerOn() error
	PowerOff() error
}

type PowerStatus string

const (
	PowerStatusOn          = PowerStatus("On")
	PowerStatusPoweringOn  = PowerStatus("PoweringOn")
	PowerStatusOff         = PowerStatus("Off")
	PowerStatusPoweringOff = PowerStatus("PoweringOff")
)

// StartIPMIServer starts an IPMI server that handles RMCP requests
func StartIPMIServer(ctx context.Context, conn net.PacketConn, machine Machine) error {
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	session := NewRMCPPlusSessionHolder()
	bmcUser := NewBMCUserHolder()
	bmcUser.AddBMCUser("cybozu", "cybozu")

	buf := make([]byte, 1024)
	for {
		_, addr, err := conn.ReadFrom(buf)
		if err != nil {
			return err
		}

		bytebuf := bytes.NewBuffer(buf)
		res, err := HandleRMCPRequest(bytebuf, machine, session, bmcUser)
		if err != nil {
			log.Warn("failed to handle RMCP request", map[string]interface{}{
				log.FnError: err,
			})
			continue
		}
		_, err = conn.WriteTo(res, addr)
		if err != nil {
			log.Warn("failed to write to UDP", map[string]interface{}{
				log.FnError: err,
			})
			continue
		}
	}
}

// StartRedfishServer starts a Redfish server
func StartRedfishServer(ctx context.Context, listener net.Listener, outDir string, machine Machine) error {
	serv := &well.HTTPServer{
		Server: &http.Server{
			Handler: prepareRouter(machine),
		},
	}

	cert, key, err := GenerateCertificate("placemat.com", outDir, 36500*24*time.Hour)
	if err != nil {
		return fmt.Errorf("failed to generate certificate: %w", err)
	}

	go func() {
		<-ctx.Done()
		listener.Close()
		serv.Close()
		os.Remove(cert)
		os.Remove(key)
	}()

	if err := serv.ServeTLS(listener, cert, key); err != nil {
		return err
	}

	return nil
}

func prepareRouter(machine Machine) http.Handler {
	router := gin.Default()
	router.GET("redfish/v1", handleServiceRoot)
	router.GET("redfish/v1/", handleServiceRoot)

	authorized := router.Group("/", gin.BasicAuth(gin.Accounts{
		"cybozu": "cybozu",
	}))
	redfish := NewRedfish(machine)
	authorized.GET("redfish/v1/Chassis", handleChassisCollection)
	authorized.GET("redfish/v1/Chassis/:id", redfish.handleChassis)
	authorized.POST("redfish/v1/Chassis/:id/Actions/Chassis.Reset", redfish.handleChassisActionsReset)
	authorized.GET("redfish/v1/Systems", handleComputerSystemCollection)
	authorized.GET("redfish/v1/Systems/:id", redfish.handleComputerSystem)
	authorized.POST("redfish/v1/Systems/:id/Actions/ComputerSystem.Reset", redfish.handleComputerSystemActionsReset)

	return router
}
