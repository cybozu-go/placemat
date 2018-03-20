package main

import (
	"bufio"
	"context"
	"flag"
	"math/rand"
	"os"
	"time"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat"
)

const (
	defaultRunPath = "/tmp"
)

var (
	runDir    = flag.String("run-dir", defaultRunPath, "run directory")
	nographic = flag.Bool("nographic", false, "run QEMU with no graphic")
)

func loadClusterFromFile(p string) (*placemat.Cluster, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readYaml(bufio.NewReader(f))
}

func loadClusterFromFiles(args []string) (*placemat.Cluster, error) {
	var cluster placemat.Cluster
	for _, p := range args {
		c, err := loadClusterFromFile(p)
		if err != nil {
			return nil, err
		}

		cluster.Append(c)
	}
	return &cluster, nil
}

func run(args []string) error {
	qemu := placemat.QemuProvider{
		BaseDir:   ".",
		NoGraphic: *nographic,
		RunDir:    *runDir,
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
	rand.Seed(time.Now().UnixNano())

	flag.Parse()
	cmd.LogConfig{}.Apply()

	err := run(flag.Args())
	if err != nil {
		log.ErrorExit(err)
	}
}
