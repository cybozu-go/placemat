package main

import (
	"context"
	"flag"
	"os"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat"
)

func loadClusterFromFile(p string) (*placemat.Cluster, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readYaml(f)
}

func loadClusterFromFiles(args []string) (*placemat.Cluster, error) {
	var cluster placemat.Cluster
	for _, p := range args {
		c, err := loadClusterFromFile(p)
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

	cluster, err := loadClusterFromFiles(args)
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
