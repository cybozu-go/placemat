package cmd

import (
	"github.com/spf13/cobra"
)

// nodeCmd represents the node command
var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "node subcommand",
	Long:  `node subcommand is the parent of commands that control nodes`,
}

func init() {
	rootCmd.AddCommand(nodeCmd)
}
