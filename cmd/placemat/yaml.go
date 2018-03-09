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

type nodeSpec struct {
	Interfaces []string `yaml:"interfaces"`
	Volumes    []struct {
		Name           string `yaml:"name"`
		Size           string `yaml:"size"`
		RecreatePolicy string `yaml:"recreatePolicy"`
	} `yaml:"volumes"`
}

type nodeConfig struct {
	Name string   `yaml:"name"`
	Spec nodeSpec `yaml:"spec"`
}

type nodeSetConfig struct {
	Name string `yaml:"name"`
	Spec struct {
		Replicas int      `yaml:"replicas"`
		Template nodeSpec `yaml:"template"`
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
	s, err := constructNodeSpec(n.Spec)
	if err != nil {
		return nil, err
	}
	node.Spec = *s

	return &node, err
}

func unmarshalNodeSet(data []byte) (*placemat.NodeSet, error) {
	var nsc nodeSetConfig
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

func constructNodeSpec(ns nodeSpec) (*placemat.NodeSpec, error) {
	var res placemat.NodeSpec
	res.Interfaces = ns.Interfaces
	if ns.Interfaces == nil {
		res.Interfaces = []string{}
	}
	res.Volumes = make([]*placemat.VolumeSpec, len(ns.Volumes))
	for i, v := range ns.Volumes {
		dst := &placemat.VolumeSpec{}
		res.Volumes[i] = dst

		dst.Name = v.Name
		dst.Size = v.Size
		var ok bool
		dst.RecreatePolicy, ok = recreatePolicyConfig[v.RecreatePolicy]
		if !ok {
			return nil, errors.New("Invalid RecreatePolicy: " + v.RecreatePolicy)
		}
	}

	return &res, nil
}

func readYaml(r *bufio.Reader) (*placemat.Cluster, error) {
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
		case "Node":
			r, err := unmarshalNode(data)
			if err != nil {
				return &cluster, err
			}
			cluster.Nodes = append(cluster.Nodes, r)
		case "NodeSet":
			r, err := unmarshalNodeSet(data)
			if err != nil {
				return &cluster, err
			}
			cluster.NodeSets = append(cluster.NodeSets, r)
		}
	}
	return &cluster, nil
}
