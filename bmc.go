package placemat

import (
	"context"
	"net"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
	"github.com/pkg/errors"
)

type bmcServer struct {
	nodeCh        chan bmcInfo
	nodeProcesses map[string]nodeProcess // key: serial
	nodeSerials   map[string]string      // key: address

	tng      nameGenerator
	networks map[string][]*net.IPNet
}

func newBMCServer() *bmcServer {
	return &bmcServer{
		nodeCh:        make(chan bmcInfo),
		nodeProcesses: make(map[string]nodeProcess),
		nodeSerials:   make(map[string]string),
		networks:      make(map[string][]*net.IPNet),
	}
}

func (s *bmcServer) setup(networks []*Network) error {
	for _, n := range networks {
		if n.Spec.Type == NetworkBMC {
			s.networks[n.Name] = make([]*net.IPNet, len(n.Spec.Addresses))
			for i, address := range n.Spec.Addresses {
				_, ipnet, err := net.ParseCIDR(address)
				if err != nil {
					return err
				}
				s.networks[n.Name][i] = ipnet
			}
		}
	}

	s.tng.prefix = "bmctap"

	return nil
}

func (s *bmcServer) start(ctx context.Context) error {
	for {
		select {
		case info := <-s.nodeCh:
			err := s.addTap(ctx, info)
			if err != nil {
				log.Warn("adding tap failed", map[string]interface{}{
					log.FnError:   err,
					"serial":      info.serial,
					"bmc_address": info.bmcAddress,
				})
			}
		case <-ctx.Done():
			s.deleteTaps(ctx)
			return nil
		}
	}
}

func (s *bmcServer) addTap(ctx context.Context, info bmcInfo) error {
	s.nodeSerials[info.bmcAddress] = info.serial

	tap := s.tng.New()
	br, err := s.findBridge(info.bmcAddress)
	if err != nil {
		return err
	}

	log.Info("creating BMC tap", map[string]interface{}{
		"serial":      info.serial,
		"bmc_address": info.bmcAddress,
		"tap":         tap,
		"bridge":      br,
	})

	err = createTap(ctx, tap, br)
	if err != nil {
		return err
	}

	c := cmd.CommandContext(ctx, "ip", "addr", "add", info.bmcAddress, "dev", tap)
	c.Severity = log.LvDebug
	return c.Run()
}

func (s *bmcServer) findBridge(address string) (string, error) {
	ip := net.ParseIP(address)

	for name, network := range s.networks {
		for _, ipnet := range network {
			if ipnet.Contains(ip) {
				return name, nil
			}
		}
	}

	return "", errors.New("BMC address not in range of BMC networks: " + address)
}

func (s *bmcServer) deleteTaps(ctx context.Context) {
	for _, tap := range s.tng.GeneratedNames() {
		err := deleteTap(ctx, tap)
		if err != nil {
			log.Error("failed to delete a TAP", map[string]interface{}{
				"name":      tap,
				log.FnError: err,
			})
		}
	}
}
