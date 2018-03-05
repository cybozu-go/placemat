package main

import (
	"context"
	"flag"
	"io/ioutil"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
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

	cmd.Go(func(ctx context.Context) error {
		return placemat.Run(ctx, qemu, cluster)
	})
	cmd.Stop()
	return cmd.Wait()
}

func main() {
	flag.Parse()
	cmd.LogConfig{}.Apply()

	err := run(flag.Args())
	if err != nil {
		log.ErrorExit(err)
	}
}
