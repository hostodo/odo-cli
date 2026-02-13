package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/spf13/cobra"
)

var jsonOutput bool

var statusCmd = &cobra.Command{
	Use:   "status [instance_id]",
	Short: "View agent token status for your instances",
	Long: `Display agent token status for all your VPS instances or a specific instance.

Without arguments, lists all instance tokens with their status.
With an instance ID, shows detailed status for that specific instance.

Examples:
  hostodo agent status                  # List all tokens
  hostodo agent status inst_abc123      # Status for specific instance
  hostodo agent status --json           # JSON output`,
	Run: runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	AgentCmd.AddCommand(statusCmd)
}

var (
	headerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
		Bold(true)

	activeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981"))

	revokedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444"))

	infoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))
)

func runStatus(cmd *cobra.Command, args []string) {
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

	if len(args) > 0 {
		// Show status for specific instance
		showSingleStatus(client, args[0])
	} else {
		// List all tokens
		listAllTokens(client)
	}
}

func listAllTokens(client *api.Client) {
	tokensResp, err := client.GetAgentTokens()
	if err != nil {
		fmt.Printf("Error: Failed to fetch agent tokens: %v\n", err)
		os.Exit(1)
	}

	if len(tokensResp.Results) == 0 {
		fmt.Println(infoStyle.Render("No agent tokens found."))
		fmt.Println(infoStyle.Render("\nAgent tokens are created when you deploy instances with AI agent enabled."))
		return
	}

	if jsonOutput {
		output, err := json.MarshalIndent(tokensResp.Results, "", "  ")
		if err != nil {
			fmt.Printf("Error: Failed to format JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(output))
		return
	}

	// Print header
	fmt.Println()
	fmt.Println(headerStyle.Render("Agent Token Status"))
	fmt.Println()

	// Print table
	output := formatTokensTable(tokensResp.Results)
	fmt.Println(output)
	fmt.Printf("\nTotal: %d token(s)\n", tokensResp.Count)
}

func showSingleStatus(client *api.Client, instanceID string) {
	token, err := client.GetAgentToken(instanceID)
	if err != nil {
		fmt.Printf("Error: Failed to fetch agent token: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		output, err := json.MarshalIndent(token, "", "  ")
		if err != nil {
			fmt.Printf("Error: Failed to format JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(output))
		return
	}

	// Print detailed view
	fmt.Println()
	fmt.Println(headerStyle.Render("Agent Token Details"))
	fmt.Println()

	statusText := token.Status
	if token.Status == "active" {
		statusText = activeStyle.Render("active")
	} else if token.Status == "revoked" {
		statusText = revokedStyle.Render("revoked")
	}

	fmt.Printf("Instance ID:    %s\n", token.InstanceID)
	fmt.Printf("Hostname:       %s\n", token.Hostname)
	fmt.Printf("Status:         %s\n", statusText)
	fmt.Printf("Created:        %s\n", formatDateTime(token.CreatedAt))
	fmt.Printf("Last Used:      %s\n", formatLastUsed(token.LastUsedAt))
	fmt.Printf("Usage Count:    %d\n", token.UsageCount)
}

func formatTokensTable(tokens []api.AgentToken) string {
	const (
		instanceWidth = 14
		hostnameWidth = 20
		statusWidth   = 10
		lastUsedWidth = 15
	)

	var sb strings.Builder

	// Header
	header := fmt.Sprintf(
		"%-*s  %-*s  %-*s  %-*s",
		instanceWidth, "INSTANCE",
		hostnameWidth, "HOSTNAME",
		statusWidth, "STATUS",
		lastUsedWidth, "LAST USED",
	)
	sb.WriteString(header + "\n")
	sb.WriteString(strings.Repeat("-", len(header)) + "\n")

	// Rows
	for _, token := range tokens {
		statusText := token.Status
		if token.Status == "active" {
			statusText = activeStyle.Render("active")
		} else if token.Status == "revoked" {
			statusText = revokedStyle.Render("revoked")
		}

		row := fmt.Sprintf(
			"%-*s  %-*s  %-*s  %-*s",
			instanceWidth, truncate(token.InstanceID, instanceWidth),
			hostnameWidth, truncate(token.Hostname, hostnameWidth),
			statusWidth, statusText,
			lastUsedWidth, formatLastUsed(token.LastUsedAt),
		)
		sb.WriteString(row + "\n")
	}

	return sb.String()
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	if length <= 3 {
		return s[:length]
	}
	return s[:length-3] + "..."
}

func formatDateTime(dateStr string) string {
	if dateStr == "" {
		return "never"
	}
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000000Z", dateStr)
		if err != nil {
			return dateStr
		}
	}
	return t.Format("Jan 02, 2006 15:04")
}

func formatLastUsed(dateStr string) string {
	if dateStr == "" {
		return "never"
	}
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000000Z", dateStr)
		if err != nil {
			return dateStr
		}
	}

	// Calculate relative time
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 02, 2006")
	}
}
