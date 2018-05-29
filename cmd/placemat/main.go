package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat"
	"github.com/cybozu-go/placemat/yaml"
)

const (
	defaultRunPath  = "/tmp"
	defaultCacheDir = ""
	defaultDataDir  = "/var/scratch/placemat"
)

var (
	flgRunDir   = flag.String("run-dir", defaultRunPath, "run directory")
	flgCacheDir = flag.String("cache-dir", defaultCacheDir, "directory for cache data")
	flgDataDir  = flag.String("data-dir", defaultDataDir, "directory to store data")
	flgGraphic  = flag.Bool("graphic", false, "run QEMU with graphical console")
	flgDebug    = flag.Bool("debug", false, "show QEMU's and Pod's stdout and stderr")
)

func loadClusterFromFile(p string) (*placemat.Cluster, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return yaml.ReadYaml(bufio.NewReader(f))
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

func run(yamls []string) error {
	if len(yamls) == 0 {
		return errors.New("no YAML files specified")
	}

	// make all YAML paths absolute
	for i, p := range yamls {
		abs, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		yamls[i] = abs
	}

	err := os.Chdir(filepath.Dir(yamls[0]))
	if err != nil {
		log.Warn("cannot chdir to YAML directory", map[string]interface{}{
			log.FnError: err.Error(),
		})
	}

	qemu := &placemat.QemuProvider{
		NoGraphic: !*flgGraphic,
		Debug:     *flgDebug,
		RunDir:    *flgRunDir,
	}

	if *flgCacheDir == "" {
		if os.Getenv("SUDO_USER") != "" {
			*flgCacheDir = "/home/${SUDO_USER}/placemat_data"
		} else {
			*flgCacheDir = *flgDataDir
		}
	}

	err = qemu.Setup(os.ExpandEnv(*flgDataDir), os.ExpandEnv(*flgCacheDir))
	if err != nil {
		return err
	}

	cluster, err := loadClusterFromFiles(yamls)
	if err != nil {
		return err
	}
	err = cluster.Resolve(qemu)
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
