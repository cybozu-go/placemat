package cmd

import (
	"context"
	"errors"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// netClearCmd represents the netClear command
var netClearCmd = &cobra.Command{
	Use:   "action ACTION DEVICE",
	Short: "control network of nodes and pods",
	Long: `control network of nodes and pods

ACTION
  * up: change state of the device to UP
  * down:  change state of the device to DOWN
  * delay: add delay to the packet through the device
  * loss: randomly lose packets through the device
  * clear: clear the effect by "delay" and "loss" action`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("action name not specified")
		} else if len(args) == 1 {
			return errors.New("device name not specified")
		} else if len(args) > 2 {
			return errors.New("too many arguments")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		action := args[0]
		device := args[1]
		well.Go(func(ctx context.Context) error {
			err := postAction(ctx, "/networks/"+device+"/"+action, nil)
			return err
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	netCmd.AddCommand(netClearCmd)
}
