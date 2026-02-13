package agent

import (
	"github.com/spf13/cobra"
)

// AgentCmd is the parent command for agent operations
var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage AI agent settings and tokens",
	Long: `Manage AI agent configuration for your VPS instances.

Commands:
  status    View agent token status for your instances
  settings  View account-level agent settings

Example:
  hostodo agent status              # List all instance tokens
  hostodo agent status <instance>   # Status for specific instance
  hostodo agent settings            # View account settings`,
}

func init() {
	// Subcommands added by their respective files (status.go, settings.go)
}
