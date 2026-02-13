package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/spf13/cobra"
)

var settingsJSONOutput bool

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "View account-level agent settings",
	Long: `Display your account-level AI agent settings.

Shows:
  - Whether AI agent is enabled for your account
  - If you're using your own API key (BYOK)
  - Monthly token limit
  - Tokens used this month

Examples:
  hostodo agent settings           # View settings
  hostodo agent settings --json    # JSON output`,
	Run: runSettings,
}

func init() {
	settingsCmd.Flags().BoolVar(&settingsJSONOutput, "json", false, "Output as JSON")
	AgentCmd.AddCommand(settingsCmd)
}

var (
	settingsHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
		Bold(true)

	labelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Width(20)

	valueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F3F4F6"))

	enabledStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981"))

	disabledStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444"))
)

func runSettings(cmd *cobra.Command, args []string) {
	// Check authentication
	if !auth.IsAuthenticated() {
		fmt.Println(infoStyle.Render("You are not logged in. Run 'hostodo login' first."))
		os.Exit(1)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error: Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Create API client
	client, err := api.NewClient(cfg)
	if err != nil {
		fmt.Printf("Error: Failed to create API client: %v\n", err)
		os.Exit(1)
	}

	// Fetch settings
	settings, err := client.GetAgentSettings()
	if err != nil {
		fmt.Printf("Error: Failed to fetch agent settings: %v\n", err)
		os.Exit(1)
	}

	if settingsJSONOutput {
		output, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			fmt.Printf("Error: Failed to format JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(output))
		return
	}

	// Print formatted output
	fmt.Println()
	fmt.Println(settingsHeaderStyle.Render("Agent Settings"))
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println()

	// Enabled
	enabledText := "No"
	if settings.Enabled {
		enabledText = enabledStyle.Render("Yes")
	} else {
		enabledText = disabledStyle.Render("No")
	}
	fmt.Printf("%s%s\n", labelStyle.Render("Enabled:"), enabledText)

	// Using own key
	ownKeyText := "No"
	if settings.UseOwnKey {
		ownKeyText = enabledStyle.Render("Yes")
	}
	fmt.Printf("%s%s\n", labelStyle.Render("Using own key:"), ownKeyText)

	// Monthly limit
	limitText := formatTokenCount(settings.MonthlyLimit)
	fmt.Printf("%s%s\n", labelStyle.Render("Monthly limit:"), valueStyle.Render(limitText))

	// Tokens used
	usedText := formatTokenCount(settings.TokensUsed)
	percentage := float64(0)
	if settings.MonthlyLimit > 0 {
		percentage = float64(settings.TokensUsed) / float64(settings.MonthlyLimit) * 100
	}
	usageDisplay := fmt.Sprintf("%s (%.1f%%)", usedText, percentage)
	fmt.Printf("%s%s\n", labelStyle.Render("Used this month:"), valueStyle.Render(usageDisplay))

	fmt.Println()
}

func formatTokenCount(count int) string {
	if count >= 1000000 {
		return fmt.Sprintf("%d,%03d,%03d tokens", count/1000000, (count%1000000)/1000, count%1000)
	} else if count >= 1000 {
		return fmt.Sprintf("%d,%03d tokens", count/1000, count%1000)
	}
	return fmt.Sprintf("%d tokens", count)
}
