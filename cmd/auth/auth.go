package auth

import (
	"github.com/spf13/cobra"
)

// AuthCmd is the parent command for authentication operations
var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long: `Manage authentication for the Hostodo CLI.

Commands:
  login     Authenticate with Hostodo
  logout    Sign out and remove stored credentials
  whoami    Display current logged-in user
  sessions  List your active CLI sessions

Example:
  hostodo auth login
  hostodo auth logout
  hostodo auth whoami
  hostodo auth sessions`,
}

func init() {
	// Subcommands will be added by their respective files
}
