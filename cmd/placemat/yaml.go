package main

import (
	"bufio"
	"errors"
	"io"

	"github.com/cybozu-go/placemat"
	k8sYaml "github.com/kubernetes/apimachinery/pkg/util/yaml"
	yaml "gopkg.in/yaml.v2"
)

type baseConfig struct {
	Kind string `yaml:"kind"`
}

type nodeConfig struct {
	Name string `yaml:"name"`
	Spec struct {
		Interfaces []string `yaml:"interfaces"`
		Volumes    []struct {
			Name           string `yaml:"name"`
			Size           string `yaml:"size"`
			RecreatePolicy string `yaml:"recreatePolicy"`
		} `yaml:"volumes"`
	} `yaml:"spec"`
}

var recreatePolicyConfig = map[string]placemat.VolumeRecreatePolicy{
	"":             placemat.RecreateIfNotPresent,
	"IfNotPresent": placemat.RecreateIfNotPresent,
	"Always":       placemat.RecreateAlways,
	"Never":        placemat.RecreateNever,
}

func unmarshalNode(data []byte) (*placemat.Node, error) {
	var n nodeConfig
	err := yaml.Unmarshal(data, &n)
	if err != nil {
		return nil, err
	}
	if n.Name == "" {
		return nil, errors.New("node name is empty")
	}

	var node placemat.Node
	node.Name = n.Name
	node.Spec.Interfaces = n.Spec.Interfaces
	if n.Spec.Interfaces == nil {
		node.Spec.Interfaces = []string{}
	}
	node.Spec.Volumes = make([]*placemat.VolumeSpec, len(n.Spec.Volumes))
	for i, v := range n.Spec.Volumes {
		dst := &placemat.VolumeSpec{}
		node.Spec.Volumes[i] = dst

		dst.Name = v.Name
		dst.Size = v.Size
		var ok bool
		dst.RecreatePolicy, ok = recreatePolicyConfig[v.RecreatePolicy]
		if !ok {
			return nil, errors.New("Invalid RecreatePolicy: " + v.RecreatePolicy)
		}
	}

	return &node, nil
}

func readYaml(r io.Reader) (*placemat.Cluster, error) {
	var c baseConfig
	var cluster placemat.Cluster
	var y = k8sYaml.NewYAMLReader(bufio.NewReader(r))
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
		case "Node":
			r, err := unmarshalNode(data)
			if err != nil {
				return &cluster, err
			}
			cluster.Nodes = append(cluster.Nodes, r)

		}
	}
	return &cluster, nil
}
