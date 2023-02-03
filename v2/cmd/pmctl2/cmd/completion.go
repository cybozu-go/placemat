package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generates shell completion scripts",
	Long: `To load completion run

Bash:

$ source <(pmctl2 completion bash)

# To load completions for each session, execute once:
Linux:
  $ pmctl2 completion bash > /etc/bash_completion.d/pmctl2
MacOS:
  $ pmctl2 completion bash > /usr/local/etc/bash_completion.d/pmctl2

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ pmctl2 completion zsh > "${fpath[1]}/_pmctl2"

# You will need to start a new shell for this setup to take effect.

Fish:

$ pmctl2 completion fish | source

# To load completions for each session, execute once:
$ pmctl2 completion fish > ~/.config/fish/completions/pmctl2.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletion(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
