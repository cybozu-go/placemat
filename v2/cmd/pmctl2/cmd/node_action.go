package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

type nodeAction string

const (
	nodeActionStart   = nodeAction("start")
	nodeActionStop    = nodeAction("stop")
	nodeActionRestart = nodeAction("restart")
)

func (n nodeAction) valid() error {
	switch n {
	case nodeActionStart, nodeActionStop, nodeActionRestart:
		return nil
	default:
		return fmt.Errorf("invalid node action: %s: valid actions are [%s|%s|%s]", n, nodeActionStart, nodeActionStop, nodeActionRestart)
	}
}

// nodeActionCmd represents the nodeAction command
var nodeActionCmd = &cobra.Command{
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
		if err := (nodeAction(action)).valid(); err != nil {
			log.ErrorExit(err)
		}

		well.Go(func(ctx context.Context) error {
			err := postAction(ctx, fmt.Sprintf("/nodes/%s/%s", node, action), nil)
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
	nodeCmd.AddCommand(nodeActionCmd)
}
