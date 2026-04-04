package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion scripts for odo CLI.

Hostname completions are dynamic — they fetch your instance list and
auto-complete hostnames as you type.

To install completions:

Bash:
  odo completion bash > ~/.local/share/bash-completion/completions/odo
  # Or on macOS with Homebrew:
  odo completion bash > $(brew --prefix)/etc/bash_completion.d/odo

Zsh:
  odo completion zsh > "${fpath[1]}/_odo"
  # Or manually:
  mkdir -p ~/.zsh_completions
  odo completion zsh > ~/.zsh_completions/_odo
  # Add to ~/.zshrc: fpath=(~/.zsh_completions $fpath); autoload -Uz compinit && compinit

Fish:
  odo completion fish > ~/.config/fish/completions/odo.fish`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeShellType,
	Run:               runCompletion,
}

func completeShellType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return []string{"bash", "zsh", "fish"}, cobra.ShellCompDirectiveNoFileComp
}

func runCompletion(cmd *cobra.Command, args []string) {
	shell := args[0]

	switch shell {
	case "bash":
		err := cmd.Root().GenBashCompletionV2(os.Stdout, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating bash completion: %v\n", err)
			os.Exit(1)
		}
	case "zsh":
		err := cmd.Root().GenZshCompletion(os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating zsh completion: %v\n", err)
			os.Exit(1)
		}
	case "fish":
		err := cmd.Root().GenFishCompletion(os.Stdout, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating fish completion: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unsupported shell: %s\nSupported: bash, zsh, fish\n", shell)
		os.Exit(1)
	}
}
