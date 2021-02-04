package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat/v2/pkg/placemat"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var nodeListParams struct {
	JSON bool
}

// nodeListCmd represents the nodeList command
var nodeListCmd = &cobra.Command{
	Use:   "list",
	Short: "show node list",
	Long:  `show node list`,
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			var status []placemat.NodeStatus
			err := getJSON(ctx, "/nodes", nil, &status)
			if err != nil {
				return err
			}
			if nodeListParams.JSON {
				return json.NewEncoder(os.Stdout).Encode(status)
			}
			for _, s := range status {
				fmt.Println(s.Name)
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
	nodeCmd.AddCommand(nodeListCmd)
	nodeListCmd.Flags().BoolVar(&nodeListParams.JSON, "json", false, "show in JSON")
}
