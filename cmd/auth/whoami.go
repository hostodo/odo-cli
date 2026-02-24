package auth

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display current logged-in user",
	Long: `Show information about the currently authenticated user.

Example:
  hostodo whoami`,
	Run: runWhoami,
}

func init() {
	AuthCmd.AddCommand(whoamiCmd)
}

var (
	whoamiLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280"))

	whoamiValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#10B981")).
				Bold(true)

	whoamiErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#EF4444"))

	whoamiInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))
)

func runWhoami(cmd *cobra.Command, args []string) {
	// Check if authenticated locally first
	if !auth.IsAuthenticated() {
		fmt.Println(whoamiInfoStyle.Render("Not logged in."))
		fmt.Println()
		fmt.Println(whoamiInfoStyle.Render("Run 'hostodo login' to authenticate."))
		return
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(whoamiErrorStyle.Render("Error: ") + "Failed to load config: " + err.Error())
		os.Exit(1)
	}

	// Create API client and get current user
	client, err := api.NewClient(cfg)
	if err != nil {
		fmt.Println(whoamiErrorStyle.Render("Error: ") + "Failed to create API client: " + err.Error())
		os.Exit(1)
	}

	user, err := client.GetCurrentUser()
	if err != nil {
		// Check if it's an auth error
		if errors.Is(err, api.ErrNotAuthenticated) || errors.Is(err, api.ErrTokenExpired) {
			fmt.Println(whoamiInfoStyle.Render("Session expired or invalid."))
			fmt.Println()
			fmt.Println(whoamiInfoStyle.Render("Run 'hostodo login' to authenticate."))
			// Clear the invalid token
			auth.DeleteToken()
			return
		}
		fmt.Println(whoamiErrorStyle.Render("Error: ") + "Failed to get user info: " + err.Error())
		os.Exit(1)
	}

	// Display user info
	fmt.Println()
	fmt.Printf("%s %s\n", whoamiLabelStyle.Render("Logged in as:"), whoamiValueStyle.Render(user.Email))

	if user.FirstName != "" || user.LastName != "" {
		name := user.FirstName
		if user.LastName != "" {
			if name != "" {
				name += " "
			}
			name += user.LastName
		}
		fmt.Printf("%s %s\n", whoamiLabelStyle.Render("Name:        "), name)
	}
	fmt.Println()
}
