package placemat

import (
	"context"
	"net"
	"strconv"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
	"github.com/pkg/errors"
)

type bmcServer struct {
	nodeCh        chan bmcInfo
	nodeProcesses map[string]nodeProcess // key: serial
	nodeSerials   map[string]string      // key: address

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

	return nil
}

func (s *bmcServer) start(ctx context.Context) error {
	for {
		select {
		case info := <-s.nodeCh:
			err := s.addPort(ctx, info)
			if err != nil {
				log.Warn("failed to add BMC port", map[string]interface{}{
					log.FnError:   err,
					"serial":      info.serial,
					"bmc_address": info.bmcAddress,
				})
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *bmcServer) addPort(ctx context.Context, info bmcInfo) error {
	s.nodeSerials[info.bmcAddress] = info.serial

	br, network, err := s.findBridge(info.bmcAddress)
	if err != nil {
		return err
	}

	prefixLen, _ := network.Mask.Size()
	address := info.bmcAddress + "/" + strconv.Itoa(prefixLen)

	log.Info("creating BMC port", map[string]interface{}{
		"serial":      info.serial,
		"bmc_address": address,
		"bridge":      br,
	})

	c := cmd.CommandContext(ctx, "ip", "addr", "add", address, "dev", br)
	c.Severity = log.LvDebug
	return c.Run()
}

func (s *bmcServer) findBridge(address string) (string, *net.IPNet, error) {
	ip := net.ParseIP(address)

	for name, network := range s.networks {
		for _, ipnet := range network {
			if ipnet.Contains(ip) {
				return name, ipnet, nil
			}
		}
	}

	return "", nil, errors.New("BMC address not in range of BMC networks: " + address)
}
