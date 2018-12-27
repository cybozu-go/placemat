package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat/web"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

func ptyPath(host string) string {
	return filepath.Join("/tmp", "placemat_"+host)
}

// nodeEnterCmd represents the nodeEnter command
var nodeEnterCmd = &cobra.Command{
	Use:   "enter NODE",
	Short: "connect to a VM with serial console",
	Long:  `connect to a VM with serial console`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("node name not specified")
		} else if len(args) > 1 {
			return errors.New("too many arguments")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		node := args[0]

		sockCh := make(chan string)
		well.Go(func(ctx context.Context) error {
			var status web.NodeStatus
			err := getJSON(ctx, "/nodes/"+node, nil, &status)
			if err != nil {
				return err
			}
			sock := status.SocketPath
			sockCh <- sock
			return nil
		})

		ptyCh := make(chan string)
		well.Go(func(ctx context.Context) error {
			var sock string
			select {
			case sock = <-sockCh:
			case <-ctx.Done():
				return ctx.Err()
			}
			pty := ptyPath(node)
			_, err := os.Stat(sock)
			if os.IsNotExist(err) {
				return errors.New(`unable to connect to "` + node + `"`)
			}
			defer os.Remove(pty)
			ptyCh <- pty
			return well.CommandContext(ctx, "socat", "UNIX-CONNECT:"+sock, "PTY,link="+pty).Run()
		})

		well.Go(func(ctx context.Context) error {
			var pty string
			select {
			case pty = <-ptyCh:
			case <-ctx.Done():
				return ctx.Err()
			}
			time.Sleep(1 * time.Second) // wait for creating a pty file

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
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	nodeCmd.AddCommand(nodeEnterCmd)
}
