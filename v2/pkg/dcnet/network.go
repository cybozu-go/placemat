package dcnet

import (
	"errors"
	"fmt"

	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"
)

const (
	maxNetworkNameLen = 15
	v4ForwardKey      = "net.ipv4.ip_forward"
	v6ForwardKey      = "net.ipv6.conf.all.forwarding"
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
	name        string
	typ         string
	useNAT      bool
	addr        *netlink.Addr
	v4forwarded bool
	v6forwarded bool
}

// NewNetwork creates *Network from spec.
func NewNetwork(spec *NetworkSpec) (*Network, error) {
	err := validate(spec)
	if err != nil {
		return nil, err
	}

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

	return n, nil
}

func validate(spec *NetworkSpec) error {
	if len(spec.Name) > maxNetworkNameLen {
		return errors.New("too long name: " + spec.Name)
	}

	switch spec.Type {
	case NetworkInternal:
		if spec.UseNAT {
			return errors.New("useNAT must be false for internal network")
		}
		if len(spec.Address) > 0 {
			return errors.New("address cannot be specified for internal network")
		}
	case NetworkExternal:
		if len(spec.Address) == 0 {
			return errors.New("address must be specified for external network")
		}
	case NetworkBMC:
		if spec.UseNAT {
			return errors.New("useNAT must be false for BMC network")
		}
		if len(spec.Address) == 0 {
			return errors.New("address must be specified for BMC network")
		}
	default:
		return errors.New("unknown type: " + spec.Type)
	}

	return nil
}

// Create creates a virtual L2 switch using Linux bridge.
func (n *Network) Create(mtu int) error {
	la := netlink.NewLinkAttrs()
	la.Name = n.name
	la.MTU = mtu
	bridge := &netlink.Bridge{LinkAttrs: la}
	err := netlink.LinkAdd(bridge)
	if err != nil {
		return fmt.Errorf("failed to add the bridge %s: %w", n.name, err)
	}
	if n.addr != nil {
		err = netlink.AddrAdd(bridge, n.addr)
		if err != nil {
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

	if !isForwarding(v4ForwardKey) {
		err = setForwarding(v4ForwardKey, true)
		if err != nil {
			return fmt.Errorf("failed to set %s: %w", v4ForwardKey, err)
		}
		n.v4forwarded = true
	}

	if !isForwarding(v6ForwardKey) {
		err = setForwarding(v6ForwardKey, true)
		if err != nil {
			return fmt.Errorf("failed to set %s: %w", v6ForwardKey, err)
		}
		n.v6forwarded = true
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

func isForwarding(name string) bool {
	val, err := sysctl.Sysctl(name)
	if err != nil {
		return false
	}
	return len(val) > 0 && val[0] != '0'
}

func setForwarding(name string, flag bool) error {
	val := "1"
	if !flag {
		val = "0"
	}
	_, err := sysctl.Sysctl(name, val)
	return err
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

	if n.v4forwarded {
		setForwarding(v4ForwardKey, false)
	}
	if n.v6forwarded {
		setForwarding(v6ForwardKey, false)
	}

	return nil
}
