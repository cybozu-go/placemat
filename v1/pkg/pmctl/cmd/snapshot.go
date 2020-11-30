package cmd

import (
	"github.com/spf13/cobra"
)

// snapshotCmd represents the node command
var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "snapshot subcommand",
	Long:  `snapshot subcommand is the parent of commands that control snapshots`,
}

func init() {
	rootCmd.AddCommand(snapshotCmd)
}
