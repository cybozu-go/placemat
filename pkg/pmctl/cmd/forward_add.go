package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat/web"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// forwardAddCmd represents the `forward add` command
var forwardAddCmd = &cobra.Command{
	Use:   "add LOCAL_PORT POD:REMOTE_HOST:REMOTE_PORT",
	Short: "add forward setting",
	Long:  `add forward setting`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("wrong number of arguments")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			var forward forwardSetting

			localPort, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}
			forward.LocalPort = localPort

			remotes := strings.Split(args[1], ":")
			if len(remotes) != 3 {
				return errors.New("remote spec must be POD:REMOTE_HOST:REMOTE_PORT")
			}
			forward.PodName = remotes[0]
			forward.RemoteHost = remotes[1]
			remotePort, err := strconv.Atoi(remotes[2])
			if err != nil {
				return err
			}
			forward.RemotePort = remotePort

			forwardJSON, err := json.Marshal(forward)
			if err != nil {
				return err
			}

			var status web.PodStatus
			err = getJSON(ctx, "/pods/"+forward.PodName, nil, &status)
			if err != nil {
				return err
			}

			c := exec.CommandContext(ctx,
				"systemd-run",
				fmt.Sprintf("--unit=pmctl-forward-%d.service", forward.LocalPort),
				fmt.Sprintf("--description=%s", forwardJSON),
				"socat",
				fmt.Sprintf("tcp-listen:%d,fork,reuseaddr", forward.LocalPort),
				fmt.Sprintf("exec:nsenter -n -t %d socat STDIO tcp-connect\\:%s\\:%d,nofork",
					status.PID, forward.RemoteHost, forward.RemotePort))
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
	forwardCmd.AddCommand(forwardAddCmd)
}
