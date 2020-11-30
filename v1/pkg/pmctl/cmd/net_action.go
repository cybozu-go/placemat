package cmd

import (
	"context"
	"errors"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var netActionParams struct {
	Delay string
	Loss  string
}

// netActionCmd represents the netClear command
var netActionCmd = &cobra.Command{
	Use:   "action ACTION DEVICE",
	Short: "control network of nodes and pods",
	Long: `control network of nodes and pods

ACTION
  * up: change state of the device to UP
  * down:  change state of the device to DOWN
  * delay: add delay to the packets going out of the device
  * loss: drop packets randomly going out of the device
  * clear: clear the effect by "delay" and "loss" action`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("action name not specified")
		} else if len(args) == 1 {
			return errors.New("device name not specified")
		} else if len(args) > 2 {
			return errors.New("too many arguments")
		}
		if len(netActionParams.Delay) != 0 && args[0] != "delay" {
			return errors.New("--delay option can be used with `delay` action")
		}
		if len(netActionParams.Loss) != 0 && args[0] != "loss" {
			return errors.New("--loss option can be used with `loss` action")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		action := args[0]
		device := args[1]
		well.Go(func(ctx context.Context) error {
			params := map[string]string{}
			if len(netActionParams.Delay) != 0 {
				params["delay"] = netActionParams.Delay
			}
			if len(netActionParams.Loss) != 0 {
				params["loss"] = netActionParams.Loss
			}
			err := postAction(ctx, "/networks/"+device+"/"+action, params)
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
	netCmd.AddCommand(netActionCmd)
	netActionCmd.Flags().StringVar(&netActionParams.Delay, "delay", "", "delay")
	netActionCmd.Flags().StringVar(&netActionParams.Loss, "loss", "", "loss")
}
