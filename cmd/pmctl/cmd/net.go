package cmd

import (
	"github.com/spf13/cobra"
)

// netCmd represents the net command
var netCmd = &cobra.Command{
	Use:   "net",
	Short: "net subcommand",
	Long:  `net subcommand is the parent of commands that control networks`,
}

func init() {
	rootCmd.AddCommand(netCmd)
}
