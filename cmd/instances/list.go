package instances

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
	instanceListJSON    bool
	instanceListSimple  bool
	instanceListDetails bool
	instanceListLimit   int
	instanceListOffset  int
)

// ListCmd represents the list command
var ListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "ps"},
	Short:   "List all your instances",
	Long: `List all your VPS instances with various output formats.

Output Formats:
  - Interactive TUI (default) - Beautiful, scrollable table with keyboard navigation
  - JSON (--json)              - JSON format for scripting and automation
  - Simple (--simple)          - Static ASCII table for quick viewing
  - Details (--details)        - Detailed view with all information

Examples:
  odo instances list                    # Interactive TUI
  odo instances ls                      # Same as list (Docker-style alias)
  odo instances ps                      # Same as list (Docker-style alias)
  odo instances list --json             # JSON output
  odo instances list --simple           # Simple table
  odo instances list --details          # Detailed view
  odo instances list --limit 50         # Show 50 instances`,
	Run: runList,
}

func init() {
	ListCmd.Flags().BoolVar(&instanceListJSON, "json", false, "Output as JSON")
	ListCmd.Flags().BoolVar(&instanceListSimple, "simple", false, "Output as simple table")
	ListCmd.Flags().BoolVar(&instanceListDetails, "details", false, "Show detailed information")
	ListCmd.Flags().IntVar(&instanceListLimit, "limit", 100, "Maximum number of instances to fetch")
	ListCmd.Flags().IntVar(&instanceListOffset, "offset", 0, "Offset for pagination")
}

func runList(cmd *cobra.Command, args []string) {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		exitWithError("Failed to load config: %v", err)
	}

	// Check authentication
	if !auth.IsAuthenticated() {
		exitWithError("You are not logged in. Please run: odo login")
	}

	// Create API client
	client, err := api.NewClient(cfg)
	if err != nil {
		exitWithError("Failed to create API client: %v", err)
	}

	// Fetch instances
	instancesResp, err := client.ListInstances(instanceListLimit, instanceListOffset)
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
	if instanceListJSON {
		// JSON output
		output, err := ui.FormatInstancesJSON(instancesResp.Results)
		if err != nil {
			exitWithError("Failed to format JSON: %v", err)
		}
		fmt.Println(output)
	} else if instanceListSimple {
		// Simple table output
		fmt.Println() // Add spacing
		output := ui.FormatInstancesSimpleTable(instancesResp.Results)
		fmt.Println(output)
		fmt.Printf("\nTotal: %d instances\n", instancesResp.Count)
	} else if instanceListDetails {
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
			RunSSH(SSHCmd, []string{tm.SSHHostname})
		}
	}
}
