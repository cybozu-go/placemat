package placemat

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/cybozu-go/placemat"
	k8sYaml "github.com/kubernetes/apimachinery/pkg/util/yaml"
	yaml "gopkg.in/yaml.v2"
)

type baseConfig struct {
	Kind string `yaml:"kind"`
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
