package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat/web"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var podListParams struct {
	JSON bool
}

// podListCmd represents the podList command
var podListCmd = &cobra.Command{
	Use:   "list",
	Short: "show pod list",
	Long:  `show pod list`,
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			var status []web.PodStatus
			getJSON(ctx, "/pods", nil, &status)
			if podListParams.JSON {
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
	podCmd.AddCommand(podListCmd)
	podListCmd.Flags().BoolVar(&podListParams.JSON, "json", false, "show in JSON")
}
