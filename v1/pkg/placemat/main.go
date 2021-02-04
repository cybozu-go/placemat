package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat"
	"github.com/cybozu-go/well"
)

const (
	defaultRunPath    = "/tmp"
	defaultCacheDir   = ""
	defaultDataDir    = "/var/scratch/placemat"
	defaultSharedPath = "/mnt/placemat"
	defaultListenAddr = "127.0.0.1:10808"
)

var (
	flgRunDir       = flag.String("run-dir", defaultRunPath, "run directory")
	flgCacheDir     = flag.String("cache-dir", defaultCacheDir, "directory for cache data")
	flgDataDir      = flag.String("data-dir", defaultDataDir, "directory to store data")
	flgSharedDir    = flag.String("shared-dir", defaultSharedPath, "shared directory")
	flgListenAddr   = flag.String("listen-addr", defaultListenAddr, "listen address")
	flgGraphic      = flag.Bool("graphic", false, "run QEMU with graphical console")
	flgDebug        = flag.Bool("debug", false, "show QEMU's and Pod's stdout and stderr")
	flgForce        = flag.Bool("force", false, "force run with removal of garbage")
	flgEnableVirtFS = flag.Bool("enable-virtfs", false, "enable VirtFS to share files between guest and host OS.")
	flgBMCCert      = flag.String("bmc-cert", "", "Certificate file for BMC HTTPS servers.")
	flgBMCKey       = flag.String("bmc-key", "", "Key file for BMC HTTPS servers.")
)

func loadClusterFromFile(p string) (*placemat.Cluster, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return placemat.ReadYaml(bufio.NewReader(f))
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

func runChildProcess() error {
	placemat, err := os.Readlink("/proc/self/exe")
	if err != nil {
		return err
	}
	c := exec.Command(placemat, os.Args[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.SysProcAttr = &syscall.SysProcAttr{
		Unshareflags: syscall.CLONE_NEWNS,
	}
	c.Env = append(os.Environ(), "UNSHARED_NAMESPACE=true")

	done := make(chan error, 1)
	err = c.Start()
	if err != nil {
		return err
	}
	go func() {
		done <- c.Wait()
	}()

	well.Go(func(ctx context.Context) error {
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			c.Process.Signal(syscall.SIGTERM)
			select {
			case <-done:
				return nil
			case <-time.After(30 * time.Second):
				log.Warn("could not stop child process.", nil)
				return nil
			}
		}
	})
	well.Stop()
	return well.Wait()
}

func run(yamls []string) error {
	if os.Getenv("UNSHARED_NAMESPACE") != "true" {
		return runChildProcess()
	}

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

	if *flgCacheDir == "" {
		if os.Getenv("SUDO_USER") != "" {
			*flgCacheDir = "/home/${SUDO_USER}/placemat_data"
		} else {
			*flgCacheDir = *flgDataDir
		}
	}
	runDir := os.ExpandEnv(*flgRunDir)
	dataDir := os.ExpandEnv(*flgDataDir)
	cacheDir := os.ExpandEnv(*flgCacheDir)
	sharedDir := os.ExpandEnv(*flgSharedDir)
	r, err := placemat.NewRuntime(*flgForce, *flgGraphic, *flgEnableVirtFS, runDir, dataDir, cacheDir, sharedDir, *flgListenAddr, *flgBMCCert, *flgBMCKey)
	if err != nil {
		return err
	}

	cluster, err := loadClusterFromFiles(yamls)
	if err != nil {
		return err
	}
	err = cluster.Resolve()
	if err != nil {
		return err
	}

	well.Go(func(ctx context.Context) error {
		return cluster.Start(ctx, r)
	})
	well.Stop()
	return well.Wait()
}

func main() {
	flag.Parse()
	err := well.LogConfig{}.Apply()
	if err != nil {
		log.ErrorExit(err)
	}

	err = run(flag.Args())
	if err != nil {
		log.ErrorExit(err)
	}
}
