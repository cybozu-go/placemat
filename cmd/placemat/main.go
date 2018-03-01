package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cybozu-go/placemat"

	yaml "gopkg.in/yaml.v2"
)

func loadResources(data []byte) (*placemat.Cluster, error) {
	yamls := bytes.Split(data, []byte("---\n"))

	var c baseConfig
	var cluster placemat.Cluster
	for _, text := range yamls {
		err := yaml.Unmarshal([]byte(text), &c)
		if err != nil {
			return &cluster, err
		}
		switch c.Kind {
		case "Node":
			r, err := loadNodeResource(text)
			if err != nil {
				return &cluster, err
			}
			cluster.Nodes = append(cluster.Nodes, r)

		}

	}
	return &cluster, nil
}

func loadResourcesFromFile(args []string) (*placemat.Cluster, error) {
	var cluster placemat.Cluster
	for _, file := range args {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}
		c, err := loadResources(data)
		if err != nil {
			return nil, err
		}

		cluster.Nodes = append(cluster.Nodes, c.Nodes...)
	}
	return &cluster, nil
}

func run(args []string) error {
	qemu := placemat.QemuProvider{
		BaseDir: ".",
	}

	cluster, err := loadResourcesFromFile(args)
	if err != nil {
		return err
	}

	return placemat.Run(context.Background(), qemu, cluster)
}

func main() {
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
