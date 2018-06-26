package placemat

import "errors"

// SMBIOSConfig represents a Node's SMBIOS definition in YAML
type SMBIOSConfig struct {
	Manufacturer string `yaml:"manufacturer,omitempty"`
	Product      string `yaml:"product,omitempty"`
	Serial       string `yaml:"serial,omitempty"`
}

// NodeSpec represents a Node specification in YAML
type NodeSpec struct {
	Name         string           `yaml:"name"`
	Interfaces   []string         `yaml:"interfaces,omitempty"`
	Volumes      []NodeVolumeSpec `yaml:"volumes,omitempty"`
	IgnitionFile string           `yaml:"ignition,omitempty"`
	CPU          int              `yaml:"cpu,omitempty"`
	Memory       string           `yaml:"memory,omitempty"`
	UEFI         bool             `yaml:"uefi,omitempty"`
	SMBIOS       SMBIOSConfig     `yaml:"smbios,omitempty"`
}

type Node struct {
	*NodeSpec
	volumes []NodeVolume
	params  []string
}

func createNodeVolume(spec NodeVolumeSpec) (NodeVolume, error) {
	switch spec.Kind {
	case "image":
		if spec.Image == "" {
			return nil, errors.New("image volume must specify an image name")
		}
		return NewImageVolume(spec.Name, spec.Image, spec.CopyOnWrite), nil
	case "localds":
		if spec.UserData == "" {
			return nil, errors.New("localds volume must specify user-data")
		}
		return NewLocalDSVolume(spec.Name, spec.UserData, spec.NetworkConfig), nil
	case "raw":
		if spec.Size == "" {
			return nil, errors.New("raw volume must specify size")
		}
		return NewRawVolume(spec.Name, spec.Size), nil
	case "vvfat":
		if spec.Folder == "" {
			return nil, errors.New("VVFAT volume must specify a folder name")
		}
		return NewVVFATVolume(spec.Name, spec.Folder), nil
	default:
		return nil, errors.New("unknown volume kind: " + spec.Kind)
	}
}

func NewNode(spec *NodeSpec) (*Node, error) {
	n := &Node{
		NodeSpec: spec,
	}
	if spec.Name == "" {
		return nil, errors.New("node name is empty")
	}

	for _, v := range spec.Volumes {
		vol, err := createNodeVolume(v)
		if err != nil {
			return nil, err
		}
		n.volumes = append(n.volumes, vol)
	}
	return n, nil
}
