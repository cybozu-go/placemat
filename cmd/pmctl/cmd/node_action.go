package cmd

import (
	"context"
	"errors"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// nodeRestartCmd represents the nodeRestart command
var nodeRestartCmd = &cobra.Command{
	Use:   "action ACTION NODE",
	Short: "control nodes",
	Long: `control nodes

ACTION
  * start: power on the target node
  * stop: power off the target node 
  * restart: restart the target node`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("action name not specified")
		} else if len(args) == 1 {
			return errors.New("node name not specified")
		} else if len(args) > 2 {
			return errors.New("too many arguments")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		action := args[0]
		node := args[1]
		well.Go(func(ctx context.Context) error {
			err := postAction(ctx, "/nodes/"+node+"/"+action, nil)
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
	nodeCmd.AddCommand(nodeRestartCmd)
}
