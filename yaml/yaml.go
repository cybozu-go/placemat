package yaml

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/cybozu-go/placemat"
	k8sYaml "github.com/kubernetes/apimachinery/pkg/util/yaml"
	yaml "gopkg.in/yaml.v2"
)

type baseConfig struct {
	Kind string `yaml:"kind"`
}

// NodeVolumeSpec represents a Node's Volume specification in YAML
type NodeVolumeSpec struct {
	Image         string `yaml:"image,omitempty"`
	UserData      string `yaml:"user-data,omitempty"`
	NetworkConfig string `yaml:"network-config,omitempty"`
	Size          string `yaml:"size,omitempty"`
	Folder        string `yaml:"folder,omitempty"`
	CopyOnWrite   bool   `yaml:"copy-on-write,omitempty"`
}

// NodeVolumeConfig represents a Node's Volume definition in YAML
type NodeVolumeConfig struct {
	Kind           string         `yaml:"kind"`
	Name           string         `yaml:"name"`
	RecreatePolicy string         `yaml:"recreatePolicy,omitempty"`
	Spec           NodeVolumeSpec `yaml:"spec"`
}

// NodeResourceConfig represents a Node's Resource definition in YAML
type NodeResourceConfig struct {
	CPU    string `yaml:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

// SMBIOSConfig represents a Node's SMBIOS definition in YAML
type SMBIOSConfig struct {
	Manufacturer string `yaml:"manufacturer,omitempty"`
	ProductName  string `yaml:"product,omitempty"`
	SerialNumber string `yaml:"serial,omitempty"`
}

// NodeSpec represents a Node specification in YAML
type NodeSpec struct {
	Interfaces   []string           `yaml:"interfaces,omitempty"`
	Volumes      []NodeVolumeConfig `yaml:"volumes,omitempty"`
	IgnitionFile string             `yaml:"ignition,omitempty"`
	Resources    NodeResourceConfig `yaml:"resources,omitempty"`
	BIOS         string             `yaml:"bios,omitempty"`
	SMBIOS       SMBIOSConfig       `yaml:"smbios,omitempty"`
}

// NodeConfig represents a Node definition in YAML
type NodeConfig struct {
	Kind string   `yaml:"kind"`
	Name string   `yaml:"name"`
	Spec NodeSpec `yaml:"spec"`
}

// NodeSetSpec represents a NodeSet specification in YAML
type NodeSetSpec struct {
	Replicas int      `yaml:"replicas"`
	Template NodeSpec `yaml:"template"`
}

// NodeSetConfig represents a NodeSet definition in YAML
type NodeSetConfig struct {
	Kind string      `yaml:"kind"`
	Name string      `yaml:"name"`
	Spec NodeSetSpec `yaml:"spec"`
}

// PodInterfaceConfig represents a Pod's Interface definition in YAML
type PodInterfaceConfig struct {
	Network   string   `yaml:"network"`
	Addresses []string `yaml:"addresses,omitempty"`
}

// PodVolumeConfig represents a Pod's Volume definition in YAML
type PodVolumeConfig struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	Folder   string `yaml:"folder,omitempty"`
	ReadOnly bool   `yaml:"readonly"`
	Mode     string `yaml:"mode,omitempty"`
	UID      string `yaml:"uid,omitempty"`
	GID      string `yaml:"gid,omitempty"`
}

// PodAppMountConfig represents a App's Mount definition in YAML
type PodAppMountConfig struct {
	Volume string `yaml:"volume"`
	Target string `yaml:"target"`
}

// PodAppConfig represents a Pod's App definition in YAML
type PodAppConfig struct {
	Name           string              `yaml:"name"`
	Image          string              `yaml:"image"`
	ReadOnlyRootfs bool                `yaml:"readonly-rootfs"`
	User           string              `yaml:"user,omitempty"`
	Group          string              `yaml:"group,omitempty"`
	Exec           string              `yaml:"exec,omitempty"`
	Args           []string            `yaml:"args,omitempty"`
	Env            map[string]string   `yaml:"env,omitempty"`
	CapsRetain     []string            `yaml:"caps-retain,omitempty"`
	Mount          []PodAppMountConfig `yaml:"mount,omitempty"`
}

// PodSpec represents a Pod specification in YAML
type PodSpec struct {
	InitScripts []string             `yaml:"init-scripts,omitempty"`
	Interfaces  []PodInterfaceConfig `yaml:"interfaces,omitempty"`
	Volumes     []PodVolumeConfig    `yaml:"volumes,omitempty"`
	Apps        []PodAppConfig       `yaml:"apps"`
}

// PodConfig represents a Pod definition in YAML
type PodConfig struct {
	Kind string  `yaml:"kind"`
	Name string  `yaml:"name"`
	Spec PodSpec `yaml:"spec"`
}

// NetworkSpec represents a Network specification in YAML
type NetworkSpec struct {
	Type      string   `yaml:"type"`
	UseNAT    bool     `yaml:"use-nat"`
	Addresses []string `yaml:"addresses,omitempty"`
}

// NetworkConfig represents a Network definition in YAML
type NetworkConfig struct {
	Kind string      `yaml:"kind"`
	Name string      `yaml:"name"`
	Spec NetworkSpec `yaml:"spec"`
}

// ImageSpec represents a Image specification in YAML
type ImageSpec struct {
	URL               string `yaml:"url,omitempty"`
	File              string `yaml:"file,omitempty"`
	CompressionMethod string `yaml:"compression,omitempty"`
}

// ImageConfig represents a Image definition in YAML
type ImageConfig struct {
	Kind string    `yaml:"kind"`
	Name string    `yaml:"name"`
	Spec ImageSpec `yaml:"spec"`
}

// DataFolderFileConfig represents a DataFolder's File definition in YAML
type DataFolderFileConfig struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url,omitempty"`
	File string `yaml:"file,omitempty"`
}

// DataFolderSpec represents a DataFolder specification in YAML
type DataFolderSpec struct {
	Dir   string                 `yaml:"dir,omitempty"`
	Files []DataFolderFileConfig `yaml:"files,omitempty"`
}

// DataFolderConfig represents a DataFolder definition in YAML
type DataFolderConfig struct {
	Kind string         `yaml:"kind"`
	Name string         `yaml:"name"`
	Spec DataFolderSpec `yaml:"spec"`
}

var recreatePolicyConfig = map[string]placemat.VolumeRecreatePolicy{
	"":             placemat.RecreateIfNotPresent,
	"IfNotPresent": placemat.RecreateIfNotPresent,
	"Always":       placemat.RecreateAlways,
	"Never":        placemat.RecreateNever,
}

var biosConfig = map[string]placemat.BIOSMode{
	"":       placemat.LegacyBIOS,
	"legacy": placemat.LegacyBIOS,
	"uefi":   placemat.UEFI,
}

func unmarshalNode(data []byte) (*placemat.Node, error) {
	var n NodeConfig
	err := yaml.Unmarshal(data, &n)
	if err != nil {
		return nil, err
	}
	if n.Name == "" {
		return nil, errors.New("node name is empty")
	}

	var node placemat.Node
	node.Name = n.Name
	s, err := constructNodeSpec(n.Spec)
	if err != nil {
		return nil, err
	}
	node.Spec = s

	return &node, nil
}

func unmarshalNodeSet(data []byte) (*placemat.NodeSet, error) {
	var nsc NodeSetConfig
	err := yaml.Unmarshal(data, &nsc)
	if err != nil {
		return nil, err
	}
	if nsc.Name == "" {
		return nil, errors.New("nodeSet name is empty")
	}

	var nodeSet placemat.NodeSet
	nodeSet.Name = nsc.Name
	nodeSet.Spec.Replicas = nsc.Spec.Replicas
	nodeSet.Spec.Template, err = constructNodeSpec(nsc.Spec.Template)

	return &nodeSet, err
}

func constructNodeSpec(ns NodeSpec) (placemat.NodeSpec, error) {
	var res placemat.NodeSpec
	var ok bool
	res.Interfaces = ns.Interfaces
	if ns.Interfaces == nil {
		res.Interfaces = []string{}
	}
	res.Volumes = make([]placemat.Volume, len(ns.Volumes))
	for i, v := range ns.Volumes {
		policy, ok := recreatePolicyConfig[v.RecreatePolicy]
		if !ok {
			return placemat.NodeSpec{}, fmt.Errorf("invalid RecreatePolicy: " + v.RecreatePolicy)
		}

		var dst placemat.Volume

		switch v.Kind {
		case "image":
			if v.Spec.Image == "" {
				return placemat.NodeSpec{}, errors.New("image volume must specify an image name")
			}
			dst = placemat.NewImageVolume(v.Name, policy, v.Spec.Image, v.Spec.CopyOnWrite)
		case "localds":
			if v.Spec.UserData == "" {
				return placemat.NodeSpec{}, errors.New("localds volume must specify user-data")
			}
			dst = placemat.NewLocalDSVolume(v.Name, policy, v.Spec.UserData, v.Spec.NetworkConfig)
		case "raw":
			if v.Spec.Size == "" {
				return placemat.NodeSpec{}, errors.New("raw volume must specify size")
			}
			dst = placemat.NewRawVolume(v.Name, policy, v.Spec.Size)
		case "vvfat":
			if v.Spec.Folder == "" {
				return placemat.NodeSpec{}, errors.New("VVFAT volume must specify a folder name")
			}
			dst = placemat.NewVVFATVolume(v.Name, policy, v.Spec.Folder)
		default:
			return placemat.NodeSpec{}, errors.New("unknown volume kind: " + v.Kind)
		}

		res.Volumes[i] = dst
	}
	res.IgnitionFile = ns.IgnitionFile
	res.Resources.CPU = ns.Resources.CPU
	res.Resources.Memory = ns.Resources.Memory
	res.BIOS, ok = biosConfig[ns.BIOS]
	if !ok {
		return placemat.NodeSpec{}, fmt.Errorf("invalid BIOS: " + ns.BIOS)
	}
	res.SMBIOS.Manufacturer = ns.SMBIOS.Manufacturer
	res.SMBIOS.Product = ns.SMBIOS.ProductName
	res.SMBIOS.Serial = ns.SMBIOS.SerialNumber

	return res, nil
}

func unmarshalNetwork(data []byte) (*placemat.Network, error) {
	var n NetworkConfig
	err := yaml.Unmarshal(data, &n)
	if err != nil {
		return nil, err
	}
	if n.Name == "" {
		return nil, errors.New("network name is empty")
	}
	if !(n.Spec.Type == "internal" || n.Spec.Type == "external" || n.Spec.Type == "bmc") {
		return nil, errors.New("unknown network type")
	}
	if n.Spec.Type == "internal" && (n.Spec.UseNAT || len(n.Spec.Addresses) > 0) {
		return nil, errors.New("'use-nat' and 'addresses' are meaningless for internal network")
	}
	if n.Spec.Type != "internal" && len(n.Spec.Addresses) == 0 {
		return nil, errors.New("addresses is empty for non-internal network")
	}

	var network placemat.Network
	network.Name = n.Name
	switch n.Spec.Type {
	case "internal":
		network.Spec.Type = placemat.NetworkInternal
	case "external":
		network.Spec.Type = placemat.NetworkExternal
	case "bmc":
		network.Spec.Type = placemat.NetworkBMC
	}
	network.Spec.UseNAT = n.Spec.UseNAT
	network.Spec.Addresses = n.Spec.Addresses
	return &network, nil

}

func unmarshalImage(data []byte) (*placemat.Image, error) {
	var dto ImageConfig
	err := yaml.Unmarshal(data, &dto)
	if err != nil {
		return nil, err
	}
	if dto.Name == "" {
		return nil, errors.New("image name is empty")
	}

	if dto.Spec.URL == "" && dto.Spec.File == "" {
		return nil, errors.New("either image.spec.url or image.spec.file must be specified")
	}
	if dto.Spec.URL != "" && dto.Spec.File != "" {
		return nil, errors.New("only one of image.spec.url or image.spec.file can be specified")
	}

	var image placemat.Image

	image.Name = dto.Name
	if dto.Spec.URL != "" {
		image.Spec.URL, err = url.Parse(dto.Spec.URL)
		if err != nil {
			return nil, err
		}
	}
	image.Spec.File = dto.Spec.File

	decompressor, err := placemat.NewDecompressor(dto.Spec.CompressionMethod)
	if err != nil {
		return nil, err
	}
	image.Spec.Decompressor = decompressor

	return &image, nil
}

func unmarshalDataFolder(data []byte) (*placemat.DataFolder, error) {
	var dto DataFolderConfig
	err := yaml.Unmarshal(data, &dto)
	if err != nil {
		return nil, err
	}
	if dto.Name == "" {
		return nil, errors.New("data folder name is empty")
	}

	if dto.Spec.Dir == "" && len(dto.Spec.Files) == 0 {
		return nil, errors.New("either datafolder.spec.dir or datafolder.spec.files must be specified")
	}
	if dto.Spec.Dir != "" && len(dto.Spec.Files) > 0 {
		return nil, errors.New("only one of datafolder.spec.dir or datafolder.spec.files can be specified")
	}

	var dataFolder placemat.DataFolder

	dataFolder.Name = dto.Name
	dataFolder.Spec.Dir = dto.Spec.Dir
	for _, file := range dto.Spec.Files {
		dataFolderFile := placemat.DataFolderFile{}

		if file.Name == "" {
			return nil, errors.New("element of datafolder.spec.files must have name")
		}
		dataFolderFile.Name = file.Name

		if file.URL == "" && file.File == "" {
			return nil, errors.New("element of datafolder.spec.files must have either url or file")
		}
		if file.URL != "" && file.File != "" {
			return nil, errors.New("element of datafolder.spec.files can have only one of url or file")
		}
		if file.URL != "" {
			dataFolderFile.URL, err = url.Parse(file.URL)
			if err != nil {
				return nil, err
			}
		}
		dataFolderFile.File = file.File

		dataFolder.Spec.Files = append(dataFolder.Spec.Files, dataFolderFile)
	}

	return &dataFolder, nil
}

func unmarshalPod(data []byte) (*placemat.Pod, error) {
	var n PodConfig
	err := yaml.Unmarshal(data, &n)
	if err != nil {
		return nil, err
	}

	var pod placemat.Pod

	if len(n.Name) == 0 {
		return nil, errors.New("pod name is empty")
	}
	pod.Name = n.Name

	for _, script := range n.Spec.InitScripts {
		script, err = filepath.Abs(script)
		if err != nil {
			return nil, err
		}
		_, err := os.Stat(script)
		if err != nil {
			return nil, err
		}
		pod.InitScripts = append(pod.InitScripts, script)
	}

	for _, iface := range n.Spec.Interfaces {
		if len(iface.Network) == 0 {
			return nil, errors.New("empty network name in pod " + n.Name)
		}
		pod.Interfaces = append(pod.Interfaces, struct {
			NetworkName string
			Addresses   []string
		}{
			iface.Network,
			iface.Addresses,
		})
	}

	for _, v := range n.Spec.Volumes {
		pv, err := placemat.NewPodVolume(v.Name, v.Kind, v.Folder, v.Mode, v.UID, v.GID, v.ReadOnly)
		if err != nil {
			return nil, err
		}
		pod.Volumes = append(pod.Volumes, pv)
	}

	if len(n.Spec.Apps) == 0 {
		return nil, errors.New("no app for pod " + n.Name)
	}

	for _, a := range n.Spec.Apps {
		var app placemat.PodApp
		if len(a.Name) == 0 {
			return nil, errors.New("empty app name in pod " + n.Name)
		}
		app.Name = a.Name
		if len(a.Image) == 0 {
			return nil, errors.New("no container image for app " + a.Name)
		}
		app.Image = a.Image
		app.ReadOnlyRootfs = a.ReadOnlyRootfs
		app.User = a.User
		app.Group = a.Group
		app.Exec = a.Exec
		app.Args = a.Args
		app.Env = a.Env
		app.CapsRetain = a.CapsRetain
		for _, m := range a.Mount {
			app.MountPoints = append(app.MountPoints, struct {
				VolumeName string
				Target     string
			}{
				m.Volume,
				m.Target,
			})
		}
		pod.Apps = append(pod.Apps, &app)
	}

	return &pod, nil
}

// ReadYaml reads a yaml file and constructs placemat.Cluster
func ReadYaml(r *bufio.Reader) (*placemat.Cluster, error) {
	var c baseConfig
	var cluster placemat.Cluster
	var y = k8sYaml.NewYAMLReader(r)
	for {
		data, err := y.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		err = yaml.Unmarshal(data, &c)
		if err != nil {
			return &cluster, err
		}

		switch c.Kind {
		case "Network":
			r, err := unmarshalNetwork(data)
			if err != nil {
				return nil, err
			}
			cluster.Networks = append(cluster.Networks, r)
		case "Image":
			r, err := unmarshalImage(data)
			if err != nil {
				return nil, err
			}
			cluster.Images = append(cluster.Images, r)
		case "DataFolder":
			r, err := unmarshalDataFolder(data)
			if err != nil {
				return nil, err
			}
			cluster.DataFolders = append(cluster.DataFolders, r)
		case "Node":
			r, err := unmarshalNode(data)
			if err != nil {
				return nil, err
			}
			cluster.Nodes = append(cluster.Nodes, r)
		case "NodeSet":
			r, err := unmarshalNodeSet(data)
			if err != nil {
				return &cluster, err
			}
			cluster.NodeSets = append(cluster.NodeSets, r)
		case "Pod":
			r, err := unmarshalPod(data)
			if err != nil {
				return nil, err
			}
			cluster.Pods = append(cluster.Pods, r)
		default:
			return nil, errors.New("unknown resource: " + c.Kind)
		}
	}
	return &cluster, nil
}
