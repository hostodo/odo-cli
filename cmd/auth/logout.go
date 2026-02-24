package auth

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out and remove stored credentials",
	Long: `Sign out from your Hostodo account.

This will:
1. Revoke your session on the server
2. Remove your stored credentials locally

Example:
  hostodo auth logout`,
	Run: runLogout,
}

func init() {
	AuthCmd.AddCommand(logoutCmd)
}

var (
	logoutSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#10B981")).
				Bold(true)

	logoutWarningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F59E0B"))

	logoutInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))
)

func runLogout(cmd *cobra.Command, args []string) {
	// Check if authenticated
	if !auth.IsAuthenticated() {
		fmt.Println(logoutInfoStyle.Render("You are not logged in."))
		return
	}

	// Try to revoke session on server
	cfg, err := config.Load()
	if err == nil {
		client, err := api.NewClient(cfg)
		if err == nil {
			if err := client.RevokeSession(); err != nil {
				// Warn but continue with local cleanup
				fmt.Println(logoutWarningStyle.Render("Warning: ") + "Could not revoke server session: " + err.Error())
			}
		}
	}

	// Always clear local token
	if err := auth.DeleteToken(); err != nil {
		fmt.Println(logoutWarningStyle.Render("Warning: ") + "Could not remove local credentials: " + err.Error())
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(logoutSuccessStyle.Render("✓ Successfully logged out"))
	fmt.Println()
	fmt.Println(logoutInfoStyle.Render("  Your session has been revoked."))
	fmt.Println(logoutInfoStyle.Render("  Run 'hostodo login' to authenticate again."))
	fmt.Println()
}
