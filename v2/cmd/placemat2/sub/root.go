package sub

import (
	"fmt"
	"os"

	v2 "github.com/cybozu-go/placemat/v2"
	"github.com/spf13/cobra"
)

const (
	defaultRunPath    = "/tmp"
	defaultCacheDir   = ""
	defaultDataDir    = "/var/scratch/placemat"
	defaultListenAddr = "127.0.0.1:10808"
)

var config struct {
	runDir     string
	cacheDir   string
	dataDir    string
	sharedDir  string
	listenAddr string
	graphic    bool
	debug      bool
	force      bool
}

var rootCmd = &cobra.Command{
	Use:   "placemat2",
	Short: "Virtual data center build tool",
	Long: `Placemat2 is a CLI tool for Go that empowers automated test for distributes system.

This application is a tool to build a virtual data center as the given settings.
Prepare a data directory before running placemat. /var/scratch is default.`,
	Version: v2.Version(),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return subMain(args)
	},
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
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&config.runDir, "run-dir", defaultRunPath, "run directory")
	pf.StringVar(&config.cacheDir, "cache-dir", defaultCacheDir, "directory for cache data")
	pf.StringVar(&config.dataDir, "data-dir", defaultDataDir, "directory to store data")
	pf.StringVar(&config.listenAddr, "listen-addr", defaultListenAddr, "listen address")
	pf.BoolVar(&config.graphic, "graphic", false, "run QEMU with graphical console")
	pf.BoolVar(&config.debug, "debug", false, "show QEMU's stdout and stderr")
	pf.BoolVar(&config.force, "force", false, "force run with removal of garbage")
}
