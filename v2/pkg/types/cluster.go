package types

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/yaml"
)

// Cluster represents a set of resources for a virtual data center.
type Cluster struct {
	Networks []*NetworkSpec
	NetNSs   []*NetNSSpec
}

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

func (n *NetworkSpec) validate() error {
	if len(n.Name) > maxNetworkNameLen {
		return errors.New("too long name: " + n.Name)
	}

	switch n.Type {
	case NetworkInternal:
		if n.UseNAT {
			return errors.New("useNAT must be false for internal network")
		}
		if len(n.Address) > 0 {
			return errors.New("address cannot be specified for internal network")
		}
	case NetworkExternal:
		if len(n.Address) == 0 {
			return errors.New("address must be specified for external network")
		}
	case NetworkBMC:
		if n.UseNAT {
			return errors.New("useNAT must be false for BMC network")
		}
		if len(n.Address) == 0 {
			return errors.New("address must be specified for BMC network")
		}
	default:
		return errors.New("unknown type: " + n.Type)
	}

	return nil
}

// NetNSSpec represents a NetworkNamespace specification in YAML
type NetNSSpec struct {
	Kind        string                `json:"kind"`
	Name        string                `json:"name"`
	Interfaces  []*NetNSInterfaceSpec `json:"interfaces"`
	Apps        []*NetNSAppSpec       `json:"apps,omitempty"`
	InitScripts []string              `json:"init-scripts,omitempty"`
}

func (n *NetNSSpec) validate() error {
	if len(n.Name) == 0 {
		return errors.New("network namespace is empty")
	}

	if len(n.Interfaces) == 0 {
		return fmt.Errorf("no interface for Network Namespace %s", n.Name)
	}

	for _, app := range n.Apps {
		if len(app.Command) == 0 {
			return fmt.Errorf("no command for app %s", app.Name)
		}
	}
	return nil
}

// NetNSInterfaceSpec represents a NetworkNamespace's Interface definition in YAML
type NetNSInterfaceSpec struct {
	Network   string   `json:"network"`
	Addresses []string `json:"addresses,omitempty"`
}

// NetNSAppSpec represents a NetworkNamespace's App definition in YAML
type NetNSAppSpec struct {
	Name    string   `json:"name"`
	Command []string `json:"command"`
}

type baseConfig struct {
	Kind string `json:"kind"`
}

// Parse reads a yaml document and create Cluster
func Parse(r io.Reader) (*Cluster, error) {
	cluster := &Cluster{}
	f := json.YAMLFramer.NewFrameReader(ioutil.NopCloser(r))
	for {
		y, err := readSingleYamlDoc(f)
		if err == io.EOF {
			break
		}
		b := &baseConfig{}
		if err := yaml.Unmarshal([]byte(y), b); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the yaml document %s: %w", y, err)
		}

		switch b.Kind {
		case "Network":
			n := &NetworkSpec{}
			if err := yaml.Unmarshal([]byte(y), n); err != nil {
				return nil, fmt.Errorf("failed to unmarshal the Network yaml document %s: %w", y, err)
			}
			if err := n.validate(); err != nil {
				return nil, fmt.Errorf("invalid Network resource: %w", err)
			}
			cluster.Networks = append(cluster.Networks, n)
		case "NetworkNamespace":
			n := &NetNSSpec{}
			if err := yaml.Unmarshal([]byte(y), n); err != nil {
				return nil, fmt.Errorf("failed to unmarshal the NetworkNamespace yaml document %s: %w", y, err)
			}
			if err := n.validate(); err != nil {
				return nil, fmt.Errorf("invalid NetworkNamespace resource: %w", err)
			}
			cluster.NetNSs = append(cluster.NetNSs, n)
		default:
			return nil, errors.New("unknown resource: " + b.Kind)
		}
	}
	return cluster, nil
}

func readSingleYamlDoc(reader io.Reader) (string, error) {
	buf := make([]byte, 1024)
	maxBytes := 16 * 1024 * 1024
	base := 0
	for {
		n, err := reader.Read(buf[base:])
		if err == io.ErrShortBuffer {
			if n == 0 {
				return "", fmt.Errorf("got short buffer with n=0, base=%d, cap=%d", base, cap(buf))
			}
			if len(buf) < maxBytes {
				base += n
				buf = append(buf, make([]byte, len(buf))...)
				continue
			}
			return "", errors.New("yaml document is too large")
		}
		if err != nil {
			return "", err
		}
		base += n
		return string(buf[:base]), nil
	}
}
