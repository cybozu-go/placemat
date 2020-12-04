package dcnet

import (
	"errors"
	"fmt"
	"net"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"
)

const (
	maxNetworkNameLen = 15
)

// Network types.
const (
	NetworkInternal = "internal"
	NetworkExternal = "external"
	NetworkBMC      = "bmc"
)

// NetworkSpec represents a Network specification in YAML
type NetworkSpec struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	UseNAT  bool   `json:"use-nat"`
	Address string `json:"address,omitempty"`
}

// Network represents a network configuration
type Network struct {
	name   string
	typ    string
	useNAT bool
	addr   *netlink.Addr
}

// NewNetwork creates *Network from spec.
func NewNetwork(spec *NetworkSpec) (*Network, error) {
	n := &Network{
		name:   spec.Name,
		typ:    spec.Type,
		useNAT: spec.UseNAT,
	}
	if len(spec.Address) > 0 {
		addr, err := netlink.ParseAddr(spec.Address)
		if err != nil {
			return nil, err
		}
		n.addr = addr
	}

	if err := n.validate(); err != nil {
		return nil, err
	}

	return n, nil
}

func (n *Network) validate() error {
	if len(n.name) > maxNetworkNameLen {
		return errors.New("too long name: " + n.name)
	}

	switch n.typ {
	case NetworkInternal:
		if n.useNAT {
			return errors.New("useNAT must be false for internal network")
		}
		if n.addr != nil {
			return errors.New("address cannot be specified for internal network")
		}
	case NetworkExternal:
		if n.addr == nil {
			return errors.New("address must be specified for external network")
		}
	case NetworkBMC:
		if n.useNAT {
			return errors.New("useNAT must be false for BMC network")
		}
		if n.addr == nil {
			return errors.New("address must be specified for BMC network")
		}
	default:
		return errors.New("unknown type: " + n.typ)
	}

	return nil
}

// Create creates a virtual L2 switch using Linux bridge.
func (n *Network) Create(mtu int) error {
	la := netlink.NewLinkAttrs()
	la.Name = n.name
	la.MTU = mtu
	la.Flags = net.FlagUp
	bridge := &netlink.Bridge{LinkAttrs: la}
	if err := netlink.LinkAdd(bridge); err != nil {
		return fmt.Errorf("failed to add the bridge %s: %w", n.name, err)
	}
	if n.addr != nil {
		if err := netlink.AddrAdd(bridge, n.addr); err != nil {
			return fmt.Errorf("failed to add the address %s: %w", n.addr.String(), err)
		}
	}

	ipt4, ipt6, err := newIptables()
	if err != nil {
		return err
	}

	if !n.useNAT {
		if n.typ == NetworkInternal {
			err := appendAcceptRule([]*iptables.IPTables{ipt4, ipt6}, n.name)
			if err != nil {
				return err
			}
		}
		return nil
	}

	if err := ip.EnableIP4Forward(); err != nil {
		return fmt.Errorf("failed to enable IPv4 forwarding: %w", err)
	}
	if err := ip.EnableIP6Forward(); err != nil {
		return fmt.Errorf("failed to enable IPv6 forwarding: %w", err)
	}
	var ipt *iptables.IPTables
	if n.addr.IP.To4() != nil {
		ipt = ipt4
	} else {
		ipt = ipt6
	}
	err = appendMasqueradeRule(ipt, n.addr.IPNet.String())
	if err != nil {
		return fmt.Errorf("failed to append append masquerade rule: %w", err)
	}

	return nil
}

func appendAcceptRule(ipts []*iptables.IPTables, ifName string) error {
	for _, ipt := range ipts {
		err := ipt.Append("filter", "PLACEMAT", "-i", ifName, "-j", "ACCEPT")
		if err != nil {
			return fmt.Errorf("failed to append the accept rule to input interface %s: %w", ifName, err)
		}
		err = ipt.Append("filter", "PLACEMAT", "-o", ifName, "-j", "ACCEPT")
		if err != nil {
			return fmt.Errorf("failed to append the accept rule to output interface %s: %w", ifName, err)
		}
	}
	return nil
}

func appendMasqueradeRule(ipt *iptables.IPTables, ipNet string) error {
	err := ipt.Append("nat", "PLACEMAT", "-s", ipNet, "!", "--destination", ipNet, "-j", "MASQUERADE")
	if err != nil {
		return err
	}
	return nil
}

// Cleanup deletes all the created bridges and restores all the modified configs.
func (n *Network) Cleanup() error {
	link, err := netlink.LinkByName(n.name)
	if err != nil {
		return err
	}
	err = netlink.LinkDel(link)
	if err != nil {
		return err
	}
	return nil
}
