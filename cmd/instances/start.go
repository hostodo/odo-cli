package instances

import (
	"fmt"
	"time"

	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/hostodo/hostodo-cli/pkg/resolver"
	"github.com/spf13/cobra"
)

// StartCmd represents the start command
var StartCmd = &cobra.Command{
	Use:               "start <hostname>",
	Short:             "Start a stopped instance",
	ValidArgsFunction: resolver.CompleteHostname,
	Long: `Start a stopped VPS instance.

This command will power on the instance. The instance must be in a stopped state.
You can specify the instance by hostname, hostname prefix, or instance ID.

Examples:
  odo instances start mybox              # Start instance with hostname "mybox"
  odo instances start my                 # Start if "my" is an unambiguous prefix
  odo instances start abc123             # Start by instance ID (fallback)`,
	Args: cobra.ExactArgs(1),
	Run:  runStart,
}

func runStart(cmd *cobra.Command, args []string) {
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

	// Start instance
	fmt.Printf("Starting instance %s (%s)...\n", instance.Hostname, instance.MainIP)
	err = client.StartInstance(instance.InstanceID)
	if err != nil {
		exitWithError("Failed to start instance: %v", err)
	}

	fmt.Println("✓ Instance start command sent successfully")
	fmt.Println("  The instance is now booting up...")
	fmt.Print("\nWaiting for instance to start")

	// Poll for status (up to 30 seconds)
	for i := 0; i < 30; i++ {
		fmt.Print(".")
		time.Sleep(1 * time.Second)

		status, err := client.GetInstancePowerStatus(instance.InstanceID)
		if err == nil && status == "running" {
			fmt.Println()
			fmt.Println("✓ Instance is now running")
			return
		}
	}

	fmt.Println()
	fmt.Println("⚠ Instance is starting (this may take a few moments)")
}

