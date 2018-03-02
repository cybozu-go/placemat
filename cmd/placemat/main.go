package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cybozu-go/placemat"
)

func loadCluster(args []string) (*placemat.Cluster, error) {
	var cluster placemat.Cluster
	for _, file := range args {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}
		c, err := unmarshalCluster(data)
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

	cluster, err := loadCluster(args)
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
