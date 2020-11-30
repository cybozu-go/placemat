package cmd

import (
	"context"
	"errors"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// snapshotLoadCmd represents the snapshotSave command
var snapshotLoadCmd = &cobra.Command{
	Use:   "load TAG",
	Short: "load a snapshot",
	Long:  `load a snapshot`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("tag name not specified")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		tag := args[0]
		well.Go(func(ctx context.Context) error {
			err := postAction(ctx, "/snapshots/load/"+tag, nil)
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
	snapshotCmd.AddCommand(snapshotLoadCmd)
}
