package instances

import (
	"fmt"

	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/hostodo/hostodo-cli/pkg/resolver"
	"github.com/hostodo/hostodo-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var instanceStatusJSON bool

// StatusCmd represents the status command
var StatusCmd = &cobra.Command{
	Use:               "status <hostname>",
	Short:             "Show detailed instance information",
	ValidArgsFunction: resolver.CompleteHostname,
	Long: `Get detailed information about a specific VPS instance.

Displays comprehensive information including:
  - Basic information (ID, hostname, status)
  - Network configuration (IPs, MAC address)
  - Resource allocation (RAM, CPU, Disk, Bandwidth)
  - Plan and template details
  - Billing information
  - Timeline (created, updated)

You can specify the instance by hostname, hostname prefix, or instance ID.

Examples:
  odo instances status mybox              # Show status for instance "mybox"
  odo instances status my                 # Show status if "my" is an unambiguous prefix
  odo instances status mybox --json       # JSON output
  odo instances status abc123             # Show status by instance ID (fallback)`,
	Args: cobra.ExactArgs(1),
	Run:  runStatus,
}

func init() {
	StatusCmd.Flags().BoolVar(&instanceStatusJSON, "json", false, "Output as JSON")
}

func runStatus(cmd *cobra.Command, args []string) {
	identifier := args[0]

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

	// Resolve hostname to instance
	result, err := resolver.ResolveInstance(client, identifier)
	if err != nil {
		exitWithError("%v", err)
	}

	instance := result.Instance

	// Fetch full instance details
	fullInstance, err := client.GetInstance(instance.InstanceID)
	if err != nil {
		exitWithError("Failed to fetch instance details: %v", err)
	}

	// Get power status
	powerStatus, err := client.GetInstancePowerStatus(instance.InstanceID)
	if err == nil {
		fullInstance.PowerStatus = powerStatus
	}

	// Display
	if instanceStatusJSON {
		output, err := ui.FormatInstancesJSON([]api.Instance{*fullInstance})
		if err != nil {
			exitWithError("Failed to format JSON: %v", err)
		}
		fmt.Println(output)
	} else {
		fmt.Println(ui.FormatInstanceDetail(fullInstance))
	}
}
