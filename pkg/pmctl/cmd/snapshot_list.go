package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// snapshotListCmd represents the snapshotList command
var snapshotListCmd = &cobra.Command{
	Use:   "list",
	Short: "list snapshots",
	Long:  `list snapshots`,
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			result := make(map[string]interface{})
			err := getJSON(ctx, "/snapshots", nil, &result)
			if err != nil {
				return err
			}
			return json.NewEncoder(os.Stdout).Encode(&result)
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	snapshotCmd.AddCommand(snapshotListCmd)
}
