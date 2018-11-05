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

// podShowCmd represents the podShow command
var podShowCmd = &cobra.Command{
	Use:   "show POD",
	Short: "show pod info",
	Long:  `show pod info`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("pod name not specified")
		} else if len(args) > 1 {
			return errors.New("too many arguments")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			pod := args[0]
			var status web.PodStatus
			getJSON(ctx, "/pods/"+pod, nil, &status)
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
	podCmd.AddCommand(podShowCmd)
}
