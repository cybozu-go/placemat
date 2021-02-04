package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var forwardListParams struct {
	JSON bool
}

// forwardListCmd represents the `forward list` command
var forwardListCmd = &cobra.Command{
	Use:   "list",
	Short: "show forward list",
	Long:  `show forward list`,
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			output, err := exec.CommandContext(ctx,
				"systemctl", "show", "pmctl-forward-*.service",
				"--property", "Description", "--value").Output()
			if err != nil {
				return err
			}

			var forwards []*ForwardSetting
			scanner := bufio.NewScanner(bytes.NewReader(output))
			for scanner.Scan() {
				line := scanner.Bytes()
				if len(line) == 0 {
					continue
				}

				forward := new(ForwardSetting)
				err := json.Unmarshal(line, forward)
				if err != nil {
					return err
				}
				forwards = append(forwards, forward)
			}

			if forwardListParams.JSON {
				return json.NewEncoder(os.Stdout).Encode(forwards)
			}

			for _, forward := range forwards {
				fmt.Printf("%d %s:%s:%d\n", forward.LocalPort, forward.PodName, forward.RemoteHost, forward.RemotePort)
			}
			return nil
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	forwardCmd.AddCommand(forwardListCmd)
	forwardListCmd.Flags().BoolVar(&forwardListParams.JSON, "json", false, "show in JSON")
}
