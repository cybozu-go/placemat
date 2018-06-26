package placemat

import (
	"context"
	"errors"
	"net"

	"github.com/cybozu-go/cmd"
)

const maxNetworkNameLen = 15

// NetworkType represents a network type.
type NetworkType int

// Network types.
const (
	NetworkInternal NetworkType = iota
	NetworkExternal
	NetworkBMC
)

// NetworkSpec represents a Network specification in YAML
type NetworkSpec struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"`
	UseNAT  bool   `yaml:"use-nat"`
	Address string `yaml:"address,omitempty"`
}

// Network represents a network configuration
type Network struct {
	*NetworkSpec
	Type NetworkType

	ip        net.IP
	ipNet     *net.IPNet
	tapNames  []string
	vethNames []string
	ng        *nameGenerator
}

func NewNetwork(spec *NetworkSpec) (*Network, error) {
	n := &Network{
		NetworkSpec: spec,
	}

	if len(spec.Name) > maxNetworkNameLen {
		return nil, errors.New("too long name: " + spec.Name)
	}

	switch spec.Type {
	case "internal":
		n.Type = NetworkInternal
		if spec.UseNAT {
			return nil, errors.New("UseNAT must be false for internal network")
		}
		if len(spec.Address) > 0 {
			return nil, errors.New("Address cannot be specified for internal network")
		}
	case "external":
		n.Type = NetworkExternal
		if len(spec.Address) == 0 {
			return nil, errors.New("Address must be specified for external network")
		}
	case "bmc":
		n.Type = NetworkBMC
		if spec.UseNAT {
			return nil, errors.New("UseNAT must be false for BMC network")
		}
		if len(spec.Address) == 0 {
			return nil, errors.New("Address must be specified for BMC network")
		}
	default:
		return nil, errors.New("unknown type: " + spec.Type)
	}

	if len(spec.Address) > 0 {
		ip, ipNet, err := net.ParseCIDR(spec.Address)
		if err != nil {
			return nil, err
		}
		n.ip = ip
		n.ipNet = ipNet
	}

	return n, nil
}

func iptables(ip net.IP) string {
	if ip.To4() != nil {
		return "iptables"
	}
	return "ip6tables"
}

func (n *Network) Create(ng *nameGenerator) error {
	n.ng = ng

	cmds := [][]string{
		{"ip", "link", "add", n.Name, "type", "bridge"},
		{"ip", "link", "set", n.Name, "up"},
	}
	if len(n.Address) > 0 {
		cmds = append(cmds,
			[]string{"ip", "addr", "add", n.Address, "dev", n.Name},
		)
	}

	err := execCommands(context.Background(), cmds)
	if err != nil {
		return err
	}

	if !n.UseNAT {
		return nil
	}

	cmds = [][]string{
		[]string{"iptables", "-t", "filter", "-A", "PLACEMAT", "-i", n.Name, "-j", "ACCEPT"},
		[]string{"iptables", "-t", "filter", "-A", "PLACEMAT", "-o", n.Name, "-j", "ACCEPT"},
		[]string{"ip6tables", "-t", "filter", "-A", "PLACEMAT", "-i", n.Name, "-j", "ACCEPT"},
		[]string{"ip6tables", "-t", "filter", "-A", "PLACEMAT", "-o", n.Name, "-j", "ACCEPT"},
		[]string{iptables(n.ip), "-t", "nat", "-A", "PLACEMAT", "-j", "MASQUERADE",
			"--source", n.ipNet.String(), "!", "--destination", n.ipNet.String()},
	}
	return execCommands(context.Background(), cmds)
}

func (n *Network) CreateTap() (string, error) {
	name := n.ng.New()

	cmds := [][]string{
		{"ip", "tuntap", "add", name, "mode", "tap"},
		{"ip", "link", "set", name, "master", n.Name},
		{"ip", "link", "set", name, "up"},
	}
	err := execCommands(context.Background(), cmds)
	if err != nil {
		return "", err
	}

	n.tapNames = append(n.tapNames, name)
	return name, nil
}

func (n *Network) CreateVeth() (string, error) {
	name := n.ng.New()
	nameInNS := name + "_"

	cmds := [][]string{
		{"ip", "link", "add", name, "type", "veth", "peer", "name", nameInNS},
		{"ip", "link", "set", name, "master", n.Name, "up"},
	}
	err := execCommands(context.Background(), cmds)
	if err != nil {
		return "", err
	}

	n.vethNames = append(n.vethNames, name)
	return nameInNS, nil
}

func (n *Network) Destroy() error {
	ctx := context.Background()

	for _, name := range n.tapNames {
		cmd.CommandContext(ctx, "ip", "tuntap", "delete", name, "mode", "tap").Run()
	}
	for _, name := range n.vethNames {
		cmd.CommandContext(ctx, "ip", "link", "delete", name).Run()
	}

	return cmd.CommandContext(ctx, "ip", "link", "delete", n.Name, "type", "bridge").Run()
}
