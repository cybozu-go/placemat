package cmd

import (
	"github.com/spf13/cobra"
)

// podCmd represents the pod command
var podCmd = &cobra.Command{
	Use:   "pod",
	Short: "pod subcommand",
	Long:  `pod subcommand is the parent of commands that control pods`,
}

func init() {
	rootCmd.AddCommand(podCmd)
}
