package virtualbmc

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/cybozu-go/log"
	"github.com/vishvananda/netlink"
)

// IPMIServer represents IPMI Server
type IPMIServer struct {
	bridges []*Bridge
}

// VM defines the interface to manipulate VM
type VM interface {
	IsRunning() bool
	PowerOn() error
	PowerOff() error
}

// Bridge represents bridge information
type Bridge struct {
	Name  string
	ipNet *net.IPNet
}

// NewIPMIServer creates an IPMIServer
func NewIPMIServer(bridges []*Bridge) (*IPMIServer, error) {
	return &IPMIServer{
		bridges: bridges,
	}, nil
}

func (s *IPMIServer) listen(ctx context.Context, addr string, port int, vm VM) error {
	if err := s.addPort(addr); err != nil {
		return fmt.Errorf("failed to add port %s", addr)
	}

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
		res, err := HandleRMCPRequest(bytebuf, vm, session, bmcUser)
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

func (s *IPMIServer) addPort(addr string) error {
	bridge, err := s.findBridge(addr)
	if err != nil {
		return err
	}

	prefixLen, _ := bridge.ipNet.Mask.Size()
	addrWithMask, err := netlink.ParseAddr(addr + "/" + strconv.Itoa(prefixLen))
	if err != nil {
		return fmt.Errorf("failed to parse the address: %w", err)
	}

	log.Info("creating BMC port", map[string]interface{}{
		"bmc_address": addrWithMask.String(),
		"bridge":      bridge.Name,
	})

	link, err := netlink.LinkByName(bridge.Name)
	if err != nil {
		return fmt.Errorf("failed to find the bridge %s: %w", bridge.Name, err)
	}
	if err := netlink.AddrAdd(link, addrWithMask); err != nil {
		return fmt.Errorf("failed to add the address %s: %w", addrWithMask.String(), err)
	}

	return nil
}

func (s *IPMIServer) findBridge(addr string) (*Bridge, error) {
	ip := net.ParseIP(addr)
	for _, bridge := range s.bridges {
		if bridge.ipNet.Contains(ip) {
			return bridge, nil
		}
	}

	return nil, fmt.Errorf("BMC address is not in the range %s", addr)
}
