package placemat

import (
	"bufio"
	"errors"
	"io"

	k8sYaml "github.com/kubernetes/apimachinery/pkg/util/yaml"
	yaml "gopkg.in/yaml.v2"
)

type baseConfig struct {
	Kind string `yaml:"kind"`
}

// ReadYaml reads a yaml file and constructs Cluster
func ReadYaml(r *bufio.Reader) (*Cluster, error) {
	var c baseConfig
	var cluster Cluster
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
