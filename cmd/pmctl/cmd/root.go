package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var globalParams struct {
	endpoint string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pmctl",
	Short: "control nodes, pods, and networks on placemat",
	Long:  `pmctl is a command-line tool to control nodes, pods, and networks on placemat`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&globalParams.endpoint, "endpoint", "http://localhost:10808", "API endpoint of the target placemat")
}
