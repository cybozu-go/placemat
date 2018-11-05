package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat/web"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// nodeShowCmd represents the nodeShow command
var nodeShowCmd = &cobra.Command{
	Use:   "show NODE",
	Short: "show node info",
	Long:  `show node info`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("node name not specified")
		} else if len(args) > 1 {
			return errors.New("too many arguments")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			node := args[0]
			var status web.NodeStatus
			getJSON(ctx, "/nodes/"+node, nil, &status)
			return json.NewEncoder(os.Stdout).Encode(status)
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	nodeCmd.AddCommand(nodeShowCmd)
}
