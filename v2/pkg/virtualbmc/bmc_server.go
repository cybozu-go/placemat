package virtualbmc

import (
	"bytes"
	"context"
	"net"

	"github.com/cybozu-go/log"
)

// Machine defines the interface to manipulate Machine
type Machine interface {
	IsRunning() bool
	PowerOn() error
	PowerOff() error
}

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
