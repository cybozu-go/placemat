package vm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat/v2/pkg/dcnet"
	"github.com/cybozu-go/placemat/v2/pkg/types"
	"github.com/cybozu-go/placemat/v2/pkg/virtualbmc"
	"github.com/cybozu-go/well"
)

type BMCServer interface {
	// Start runs BMC Server that servers
	Start(ctx context.Context) error
}

type bmcServer struct {
	nodeCh   <-chan BMCInfo
	networks []dcnet.Network
	vms      map[string]VM // key: serial
	tempDir  string
}

// NewBMCServer creates a BMCServer instance
func NewBMCServer(vms map[string]VM, networks []dcnet.Network, ch <-chan BMCInfo, tempDir string) BMCServer {
	s := &bmcServer{
		nodeCh:  ch,
		vms:     vms,
		tempDir: tempDir,
	}
	for _, n := range networks {
		if n.IsType(types.NetworkBMC) {
			s.networks = append(s.networks, n)
		}
	}

	return s
}

func (s *bmcServer) Start(ctx context.Context) error {
	env := well.NewEnvironment(ctx)

OUTER:
	for {
		select {
		case info := <-s.nodeCh:
			// Configure network
			err := s.addBMCAddrToNetwork(info)
			if err != nil {
				log.Error("failed to add BMC port", map[string]interface{}{
					log.FnError:   err,
					"serial":      info.serial,
					"bmc_address": info.bmcAddress,
				})
			}

			// Start IPMI server
			serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", info.bmcAddress, 623))
			if err != nil {
				log.Error("failed to resolve UDP address", map[string]interface{}{
					log.FnError: err,
					"address":   info.bmcAddress,
				})
			}
			conn, err := net.ListenUDP("udp", serverAddr)
			if err != nil {
				log.Error("failed to listen UDP address", map[string]interface{}{
					log.FnError: err,
					"address":   info.bmcAddress,
				})
			}
			env.Go(func(ctx context.Context) error {
				return virtualbmc.StartIPMIServer(ctx, conn, s.vms[info.serial])
			})

			// Start Redfish server
			addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", info.bmcAddress, 443))
			if err != nil {
				log.Error("failed to resolve TCP address", map[string]interface{}{
					log.FnError: err,
					"address":   info.bmcAddress,
				})
			}
			listener, err := net.ListenTCP("tcp", addr)
			if err != nil {
				log.Error("failed to listen TCPk address", map[string]interface{}{
					log.FnError: err,
					"address":   info.bmcAddress,
				})
			}
			env.Go(func(ctx context.Context) error {
				return virtualbmc.StartRedfishServer(ctx, listener, s.tempDir, s.vms[info.serial])
			})

		case <-ctx.Done():
			break OUTER
		}
	}

	env.Cancel(nil)
	return env.Wait()
}

func (s *bmcServer) addBMCAddrToNetwork(info BMCInfo) error {
	br, err := s.findBridge(info.bmcAddress)
	if err != nil {
		return err
	}

	log.Info("creating BMC port", map[string]interface{}{
		"serial":      info.serial,
		"bmc_address": info.bmcAddress,
	})

	if err := br.AddAddr(info.bmcAddress); err != nil {
		return fmt.Errorf("failed to add IP Address: %s: %w", info.bmcAddress, err)
	}

	return nil
}

func (s *bmcServer) findBridge(address string) (dcnet.Network, error) {
	ip := net.ParseIP(address)

	for _, n := range s.networks {
		if n.Contains(ip) {
			return n, nil
		}
	}

	return nil, fmt.Errorf("BMC address not in range of BMC networks: %s", address)
}

type BMCInfo struct {
	serial     string
	bmcAddress string
}

type guestConnection struct {
	serial string
	sent   bool
	guest  net.Conn
	ch     chan<- BMCInfo
}

func (g *guestConnection) handle() {
	bufr := bufio.NewReader(g.guest)
	for {
		line, err := bufr.ReadBytes('\n')
		if err != nil {
			return
		}

		if g.sent {
			continue
		}

		bmcAddress := string(bytes.TrimSpace(line))
		g.ch <- BMCInfo{
			serial:     g.serial,
			bmcAddress: bmcAddress,
		}
		g.sent = true
	}
}
