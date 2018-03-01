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

func loadResources(data []byte) ([]*placemat.Network, []*placemat.Node, []*placemat.NodeSet, error) {
	yamls := bytes.Split(data, []byte("---\n"))

	var c baseConfig
	var nodes = make([]*placemat.Node, 0, 0)
	for _, text := range yamls {
		err := yaml.Unmarshal([]byte(text), &c)
		if err != nil {
			return nil, nil, nil, err
		}
		switch c.Kind {
		case "Node":
			r, err := loadNodeResource(text)
			if err != nil {
				return nil, nil, nil, err
			}
			nodes = append(nodes, r)

		}

	}
	return nil, nodes, nil, nil
}

func loadResourcesFromFile(args []string) ([]*placemat.Network, []*placemat.Node, []*placemat.NodeSet, error) {
	var allNodes = make([]*placemat.Node, 0, 0)
	for _, file := range args {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, nil, nil, err
		}
		_, nodes, _, err := loadResources(data)
		if err != nil {
			return nil, nil, nil, err
		}

		allNodes = append(allNodes, nodes...)
	}
	return nil, allNodes, nil, nil
}

func run(args []string) error {
	networks, nodes, nodesets, err := loadResourcesFromFile(args)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return placemat.Start(ctx, networks, nodes, nodesets)
}

func main() {
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
