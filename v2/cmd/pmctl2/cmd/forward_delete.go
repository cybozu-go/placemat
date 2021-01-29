package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// forwardDeleteCmd represents the `forward delete` command
var forwardDeleteCmd = &cobra.Command{
	Use:   "delete LOCAL_PORT",
	Short: "delete forward setting",
	Long:  `delete forward setting`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("wrong number of arguments")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			localPort, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}

			service := fmt.Sprintf("pmctl-forward-%d.service", localPort)

			c := exec.CommandContext(ctx, "systemctl", "stop", service)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			err = c.Run()
			if err != nil {
				return err
			}

			c = exec.CommandContext(ctx, "systemctl", "disable", service)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			err = c.Run()
			if err != nil {
				return err
			}

			err = os.Remove(filepath.Join("/run/systemd/transient", service))
			if err != nil {
				return err
			}

			c = exec.CommandContext(ctx, "systemctl", "daemon-reload")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			err = c.Run()
			if err != nil {
				return err
			}

			c = exec.CommandContext(ctx, "systemctl", "reset-failed", service)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	forwardCmd.AddCommand(forwardDeleteCmd)
}
