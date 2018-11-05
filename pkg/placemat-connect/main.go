package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cybozu-go/well"
)

const (
	defaultRunPath = "/tmp"
)

var (
	runDir = flag.String("run-dir", defaultRunPath, "run directory")
)

func socketPath(host string) string {
	return filepath.Join(*runDir, host+".socket")
}

func ptyPath(host string) string {
	return filepath.Join("/tmp", "placemat_"+host)
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("host not specified")
	} else if len(args) > 1 {
		return errors.New("too many arguments")
	}

	host := args[0]
	sock := socketPath(host)
	pty := ptyPath(host)

	_, err := os.Stat(sock)
	if os.IsNotExist(err) {
		return errors.New(`unable to connect to "` + host + `"`)
	}

	well.Go(func(ctx context.Context) error {
		defer os.Remove(pty)
		return exec.CommandContext(ctx, "socat", "UNIX-CONNECT:"+sock, "PTY,link="+pty).Run()
	})
	well.Go(func(ctx context.Context) error {
		time.Sleep(1 * time.Second)

		cmd := exec.CommandContext(ctx, "picocom", "-e", "q", pty)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return err
		}
		return context.Canceled
	})
	well.Stop()
	return well.Wait()
}

func main() {
	flag.Parse()
	err := run(flag.Args())
	if err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
