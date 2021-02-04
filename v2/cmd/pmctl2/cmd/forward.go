package cmd

import (
	"github.com/spf13/cobra"
)

// ForwardSetting is forward setting
type ForwardSetting struct {
	LocalPort  int    `json:"local_port"`
	PodName    string `json:"pod"`
	RemoteHost string `json:"remote_host"`
	RemotePort int    `json:"remote_port"`
}

// forwardCmd represents the forward command
var forwardCmd = &cobra.Command{
	Use:   "forward",
	Short: "forward subcommand",
	Long:  `forward subcommand is the parent of commands that control port-forward settings`,
}

func init() {
	rootCmd.AddCommand(forwardCmd)
}
