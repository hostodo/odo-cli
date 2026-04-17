package instances

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/hostodo/hostodo-cli/pkg/deploy"
	"github.com/hostodo/hostodo-cli/pkg/resolver"
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
		output, err := ui.FormatInstancesJSON(instancesResp.Results)
		if err != nil {
			exitWithError("Failed to format JSON: %v", err)
		}
		fmt.Println(output)
	} else if instanceListSimple {
		fmt.Println()
		output := ui.FormatInstancesSimpleTable(instancesResp.Results)
		fmt.Println(output)
		fmt.Printf("\nTotal: %d instances\n", instancesResp.Count)
	} else if instanceListDetails {
		fmt.Println()
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
		tm, ok := finalModel.(ui.TableModel)
		if !ok {
			return
		}
		// Handle actions requested from within the TUI
		if tm.SSHHostname != "" {
			RunSSH(SSHCmd, []string{tm.SSHHostname})
			return
		}
		if tm.APIAction != "" && tm.ActionInstance != nil {
			runTUIAction(client, tm)
		}
	}
}

// runTUIAction executes an instance action that was triggered from the TUI detail view.
func runTUIAction(client *api.Client, tm ui.TableModel) {
	instance := tm.ActionInstance
	fmt.Println()

	switch tm.APIAction {
	case "start":
		fmt.Printf("Starting instance %s (%s)...\n", instance.Hostname, instance.MainIP)
		if err := client.StartInstance(instance.InstanceID); err != nil {
			exitWithError("Failed to start instance: %v", err)
		}
		fmt.Println("✓ Start command sent")
		fmt.Print("\nWaiting for instance to start")
		pollPower(client, instance.InstanceID, "running", 30)

	case "stop":
		fmt.Printf("Stopping instance %s (%s)...\n", instance.Hostname, instance.MainIP)
		if err := client.StopInstance(instance.InstanceID, false); err != nil {
			exitWithError("Failed to stop instance: %v", err)
		}
		fmt.Println("✓ Stop command sent")
		fmt.Print("\nWaiting for instance to stop")
		pollPower(client, instance.InstanceID, "stopped", 30)

	case "force-stop":
		fmt.Printf("Force stopping instance %s (%s)...\n", instance.Hostname, instance.MainIP)
		if err := client.StopInstance(instance.InstanceID, true); err != nil {
			exitWithError("Failed to force stop instance: %v", err)
		}
		fmt.Println("✓ Force stop command sent")

	case "restart":
		fmt.Printf("Restarting instance %s (%s)...\n", instance.Hostname, instance.MainIP)
		if err := client.RebootInstance(instance.InstanceID, false); err != nil {
			exitWithError("Failed to restart instance: %v", err)
		}
		fmt.Println("✓ Restart command sent")
		fmt.Print("\nWaiting for instance to restart")
		pollRestart(client, instance.InstanceID, 60)

	case "force-restart":
		fmt.Printf("Force restarting instance %s (%s)...\n", instance.Hostname, instance.MainIP)
		if err := client.RebootInstance(instance.InstanceID, true); err != nil {
			exitWithError("Failed to force restart instance: %v", err)
		}
		fmt.Println("✓ Force restart command sent")

	case "rename":
		newHostname := tm.RenameTarget
		if err := deploy.Validate(newHostname); err != nil {
			exitWithError("Invalid hostname %q: %v", newHostname, err)
		}
		fmt.Printf("Renaming instance %s to %s...\n", instance.Hostname, newHostname)
		if err := client.RenameInstance(instance.InstanceID, newHostname); err != nil {
			exitWithError("Failed to rename instance: %v", err)
		}
		resolver.InvalidateCache()
		fmt.Printf("✓ Instance renamed to %s\n", newHostname)

	case "reinstall":
		if err := ReinstallCmd.RunE(ReinstallCmd, []string{instance.Hostname}); err != nil {
			exitWithError("Reinstall failed: %v", err)
		}
	}
}

// pollPower polls until the instance reaches targetStatus or timeout.
func pollPower(client *api.Client, instanceID, targetStatus string, seconds int) {
	for i := 0; i < seconds; i++ {
		fmt.Print(".")
		time.Sleep(1 * time.Second)
		status, err := client.GetInstancePowerStatus(instanceID)
		if err == nil && status == targetStatus {
			fmt.Println()
			fmt.Printf("✓ Instance is now %s\n", targetStatus)
			return
		}
	}
	fmt.Println()
	fmt.Println("⚠ Operation in progress (this may take a few moments)")
}

// pollRestart polls for running state after a reboot, allowing for a stop transition.
func pollRestart(client *api.Client, instanceID string, seconds int) {
	sawStopped := false
	for i := 0; i < seconds; i++ {
		fmt.Print(".")
		time.Sleep(1 * time.Second)
		status, err := client.GetInstancePowerStatus(instanceID)
		if err == nil {
			if status == "stopped" {
				sawStopped = true
			} else if status == "running" && (sawStopped || i >= 5) {
				fmt.Println()
				fmt.Println("✓ Instance has restarted and is now running")
				return
			}
		}
	}
	fmt.Println()
	fmt.Println("⚠ Instance is restarting (this may take a few moments)")
}
