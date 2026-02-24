package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/hostodo/hostodo-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	listJSONOutput    bool
	listSimpleOutput  bool
	listDetailsOutput bool
	listLimit         int
	listOffset        int
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "ps"},
	Short:   "List all your instances",
	Long: `List all your VPS instances with various output formats.

Output Formats:
  • Interactive TUI (default) - Beautiful, scrollable table with keyboard navigation
  • JSON (--json)              - JSON format for scripting and automation
  • Simple (--simple)          - Static ASCII table for quick viewing
  • Details (--details)        - Detailed view with all information

Examples:
  hostodo list                    # Interactive TUI
  hostodo ls                      # Same as list (Docker-style alias)
  hostodo ps                      # Same as list (Docker-style alias)
  hostodo list --json             # JSON output
  hostodo list --simple           # Simple table
  hostodo list --details          # Detailed view
  hostodo list --limit 50         # Show 50 instances`,
	Run: runList,
}

func init() {
	listCmd.Flags().BoolVar(&listJSONOutput, "json", false, "Output as JSON")
	listCmd.Flags().BoolVar(&listSimpleOutput, "simple", false, "Output as simple table")
	listCmd.Flags().BoolVar(&listDetailsOutput, "details", false, "Show detailed information")
	listCmd.Flags().IntVar(&listLimit, "limit", 100, "Maximum number of instances to fetch")
	listCmd.Flags().IntVar(&listOffset, "offset", 0, "Offset for pagination")
}

func runList(cmd *cobra.Command, args []string) {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		exitWithError("Failed to load config: %v", err)
	}

	// Check authentication
	if !auth.IsAuthenticated() {
		exitWithError("You are not logged in. Please run: hostodo login")
	}

	// Create API client
	client, err := api.NewClient(cfg)
	if err != nil {
		exitWithError("Failed to create API client: %v", err)
	}

	// Fetch instances
	instancesResp, err := client.ListInstances(listLimit, listOffset)
	if err != nil {
		exitWithError("Failed to fetch instances: %v", err)
	}

	if len(instancesResp.Results) == 0 {
		fmt.Println("No instances found.")
		fmt.Println("\nYou don't have any VPS instances yet.")
		fmt.Println("Visit https://console.hostodo.com to deploy your first instance!")
		return
	}

	// Display based on output format
	if listJSONOutput {
		// JSON output
		output, err := ui.FormatInstancesJSON(instancesResp.Results)
		if err != nil {
			exitWithError("Failed to format JSON: %v", err)
		}
		fmt.Println(output)
	} else if listSimpleOutput {
		// Simple table output
		fmt.Println() // Add spacing
		output := ui.FormatInstancesSimpleTable(instancesResp.Results)
		fmt.Println(output)
		fmt.Printf("\nTotal: %d instances\n", instancesResp.Count)
	} else if listDetailsOutput {
		// Detailed output
		fmt.Println() // Add spacing
		output := ui.FormatInstancesDetailedTable(instancesResp.Results)
		fmt.Println(output)
		fmt.Printf("\nTotal: %d instances\n", instancesResp.Count)
	} else {
		// Interactive TUI (default)
		p := tea.NewProgram(ui.NewTableModel(instancesResp.Results, client.GetInstancePowerStatus))
		finalModel, err := p.Run()
		if err != nil {
			exitWithError("Failed to run interactive table: %v", err)
		}
		// Check if user requested SSH from detail view
		if tm, ok := finalModel.(ui.TableModel); ok && tm.SSHHostname != "" {
			runSSH(sshCmd, []string{tm.SSHHostname})
		}
	}
}

