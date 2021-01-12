package virtualbmc

import (
	"bytes"
	"context"
	"fmt"
	"net"

	"github.com/cybozu-go/log"
)

// BMCServer represents IPMI Server
type BMCServer struct {
}

// Machine defines the interface to manipulate Machine
type Machine interface {
	IsRunning() bool
	PowerOn() error
	PowerOff() error
}

// NewBMCServer creates an BMCServer
func NewBMCServer() (*BMCServer, error) {
	return &BMCServer{}, nil
}

func (s *BMCServer) listen(ctx context.Context, addr string, port int, machine Machine) error {
	serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return err
	}

	server, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		server.Close()
	}()

	session := NewRMCPPlusSessionHolder()
	bmcUser := NewBMCUserHolder()
	bmcUser.AddBMCUser("cybozu", "cybozu")

	buf := make([]byte, 1024)
	for {
		_, addr, err := server.ReadFromUDP(buf)
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
		_, err = server.WriteToUDP(res, addr)
		if err != nil {
			log.Warn("failed to write to UDP", map[string]interface{}{
				log.FnError: err,
			})
			continue
		}
	}
}
