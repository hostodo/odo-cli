package auth

import (
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

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List your active CLI sessions",
	Long: `Display all active CLI sessions across your devices.

Each session represents a device where you've logged in with 'hostodo login'.
Sessions expire after 90 days of inactivity.

Example:
  hostodo auth sessions`,
	Run: runSessions,
}

func init() {
	AuthCmd.AddCommand(sessionsCmd)
}

var (
	sessionHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#10B981")).
				Bold(true)

	sessionInfoStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280"))
)

func runSessions(cmd *cobra.Command, args []string) {
	// Check authentication
	if !auth.IsAuthenticated() {
		fmt.Println(sessionInfoStyle.Render("You are not logged in. Run 'hostodo login' first."))
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

	// Fetch sessions
	sessionsResp, err := client.ListCLISessions()
	if err != nil {
		fmt.Printf("Error: Failed to fetch sessions: %v\n", err)
		os.Exit(1)
	}

	if len(sessionsResp.Results) == 0 {
		fmt.Println(sessionInfoStyle.Render("No active sessions found."))
		return
	}

	// Print header
	fmt.Println()
	fmt.Println(sessionHeaderStyle.Render("Active CLI Sessions"))
	fmt.Println()

	// Print table
	output := formatSessionsTable(sessionsResp.Results)
	fmt.Println(output)
	fmt.Printf("\nTotal: %d session(s)\n", sessionsResp.Count)
}

func formatSessionsTable(sessions []api.CLISession) string {
	const (
		idWidth       = 6
		deviceWidth   = 25
		ipWidth       = 16
		createdWidth  = 12
		lastUsedWidth = 12
	)

	var sb strings.Builder

	// Header
	header := fmt.Sprintf(
		"%-*s  %-*s  %-*s  %-*s  %-*s",
		idWidth, "ID",
		deviceWidth, "DEVICE",
		ipWidth, "IP ADDRESS",
		createdWidth, "CREATED",
		lastUsedWidth, "LAST USED",
	)
	sb.WriteString(header + "\n")
	sb.WriteString(strings.Repeat("-", len(header)) + "\n")

	// Rows
	for _, session := range sessions {
		row := fmt.Sprintf(
			"%-*d  %-*s  %-*s  %-*s  %-*s",
			idWidth, session.ID,
			deviceWidth, truncateSession(session.DeviceName, deviceWidth),
			ipWidth, truncateSession(session.LoginIP, ipWidth),
			createdWidth, formatDate(session.CreatedAt),
			lastUsedWidth, formatDate(session.LastUsedAt),
		)
		sb.WriteString(row + "\n")
	}

	return sb.String()
}

func truncateSession(s string, length int) string {
	if len(s) <= length {
		return s
	}
	if length <= 3 {
		return s[:length]
	}
	return s[:length-3] + "..."
}

func formatDate(dateStr string) string {
	// Parse ISO format and return short date
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		// Try alternative format
		t, err = time.Parse("2006-01-02T15:04:05.000000Z", dateStr)
		if err != nil {
			return dateStr[:10] // Fallback to first 10 chars
		}
	}
	return t.Format("Jan 02 2006")
}
