package virtualbmc

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/gin-gonic/gin"
)

// Machine defines the interface to manipulate Machine
type Machine interface {
	PowerStatus() (PowerStatus, error)
	PowerOn() error
	PowerOff() error
}

type PowerStatus string

const (
	PowerStatusOn          = PowerStatus("On")
	PowerStatusPoweringOn  = PowerStatus("PoweringOn")
	PowerStatusOff         = PowerStatus("Off")
	PowerStatusPoweringOff = PowerStatus("PoweringOff")
	PowerStatusUnknown     = PowerStatus("Unknown")
)

// StartIPMIServer starts an ipmi server that handles RMCP requests
func StartIPMIServer(ctx context.Context, conn net.PacketConn, machine Machine) error {
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	session := newRMCPPlusSessionHolder()
	bmcUser := newBMCUserHolder()
	bmcUser.addBMCUser("cybozu", "cybozu")

	buf := make([]byte, 1024)
	for {
		_, addr, err := conn.ReadFrom(buf)
		if err != nil {
			return err
		}

		bytebuf := bytes.NewBuffer(buf)
		res, err := handleRMCPRequest(bytebuf, machine, session, bmcUser)
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

// StartRedfishServer starts a redfish server
func StartRedfishServer(ctx context.Context, listener net.Listener, machine Machine) error {
	serv := &well.HTTPServer{
		Server: &http.Server{
			Handler: prepareRouter(machine),
		},
	}

	certPem, keyPem, err := generateCertificate("placemat.com", 36500*24*time.Hour)
	if err != nil {
		return fmt.Errorf("failed to generate certificate: %w", err)
	}

	go func() {
		<-ctx.Done()
		listener.Close()
		serv.Close()
	}()

	cert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		return err
	}

	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	tlsListener := tls.NewListener(listener, cfg)
	if err := serv.Server.Serve(tlsListener); err != nil {
		log.Error("failed to serve TLS", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	return nil
}

func prepareRouter(machine Machine) http.Handler {
	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, nil)
	})
	router.GET("redfish/v1", handleServiceRoot)
	router.GET("redfish/v1/", handleServiceRoot)

	authorized := router.Group("/", gin.BasicAuth(gin.Accounts{
		"cybozu": "cybozu",
	}))
	redfish := newRedfishServer(machine)
	authorized.GET("redfish/v1/Chassis", handleChassisCollection)
	authorized.GET("redfish/v1/Chassis/:id", redfish.handleChassis)
	authorized.POST("redfish/v1/Chassis/:id/Actions/Chassis.Reset", redfish.handleChassisActionsReset)
	authorized.GET("redfish/v1/Systems", handleComputerSystemCollection)
	authorized.GET("redfish/v1/Systems/:id", redfish.handleComputerSystem)
	authorized.POST("redfish/v1/Systems/:id/Actions/ComputerSystem.Reset", redfish.handleComputerSystemActionsReset)

	return router
}
