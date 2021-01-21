package sub

import (
	"bufio"
	"context"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat/v2/pkg/placemat"
	"github.com/cybozu-go/placemat/v2/pkg/types"
	"github.com/cybozu-go/placemat/v2/pkg/vm"
	"github.com/cybozu-go/well"
	"github.com/gin-gonic/gin"
)

func subMain(args []string) error {
	rand.Seed(time.Now().UnixNano())

	err := well.LogConfig{}.Apply()
	if err != nil {
		log.ErrorExit(err)
	}

	if !config.debug {
		gin.SetMode(gin.ReleaseMode)
	}

	return run(args)
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

	if config.cacheDir == "" {
		if os.Getenv("SUDO_USER") != "" {
			config.cacheDir = "/home/${SUDO_USER}/placemat_data"
		} else {
			config.cacheDir = config.dataDir
		}
	}
	runDir := os.ExpandEnv(config.runDir)
	dataDir := os.ExpandEnv(config.dataDir)
	cacheDir := os.ExpandEnv(config.cacheDir)
	vm.LoadModules()
	r, err := vm.NewRuntime(config.force, config.graphic, runDir, dataDir, cacheDir, config.listenAddr)
	if err != nil {
		return err
	}

	spec, err := loadClusterFromFiles(yamls)
	if err != nil {
		return err
	}

	cluster, err := placemat.NewCluster(spec)
	if err != nil {
		return err
	}

	well.Go(func(ctx context.Context) error {
		return cluster.Setup(ctx, r)
	})
	well.Stop()
	return well.Wait()
}

func loadClusterFromFile(p string) (*types.ClusterSpec, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return types.Parse(bufio.NewReader(f))
}

func loadClusterFromFiles(args []string) (*types.ClusterSpec, error) {
	var cluster types.ClusterSpec
	for _, p := range args {
		c, err := loadClusterFromFile(p)
		if err != nil {
			return nil, err
		}

		cluster.Append(c)
	}
	return &cluster, nil
}
