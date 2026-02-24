package cmd

import (
	"fmt"
	"time"

	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/hostodo/hostodo-cli/pkg/resolver"
	"github.com/spf13/cobra"
)

var stopForce bool

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:               "stop <hostname>",
	Short:             "Stop a running instance",
	ValidArgsFunction: resolver.CompleteHostname,
	Long: `Stop a running VPS instance.

This command will gracefully shut down the instance. Use --force for immediate shutdown.
You can specify the instance by hostname, hostname prefix, or instance ID.

Examples:
  hostodo stop mybox              # Stop instance with hostname "mybox"
  hostodo stop my                 # Stop if "my" is an unambiguous prefix
  hostodo stop mybox --force      # Force immediate shutdown
  hostodo stop abc123             # Stop by instance ID (fallback)`,
	Args: cobra.ExactArgs(1),
	Run:  runStop,
}

func init() {
	stopCmd.Flags().BoolVarP(&stopForce, "force", "f", false, "Force immediate shutdown")
}

func runStop(cmd *cobra.Command, args []string) {
	identifier := args[0]

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

	// Resolve hostname to instance
	result, err := resolver.ResolveInstance(client, identifier)
	if err != nil {
		exitWithError("%v", err)
	}

	instance := result.Instance

	// Stop instance
	if stopForce {
		fmt.Printf("Force stopping instance %s (%s)...\n", instance.Hostname, instance.MainIP)
	} else {
		fmt.Printf("Stopping instance %s (%s)...\n", instance.Hostname, instance.MainIP)
	}

	err = client.StopInstance(instance.InstanceID, stopForce)
	if err != nil {
		exitWithError("Failed to stop instance: %v", err)
	}

	fmt.Println("✓ Instance stop command sent successfully")
	fmt.Println("  The instance is now shutting down...")
	fmt.Print("\nWaiting for instance to stop")

	// Poll for status (up to 60 seconds)
	for i := 0; i < 60; i++ {
		fmt.Print(".")
		time.Sleep(1 * time.Second)

		status, err := client.GetInstancePowerStatus(instance.InstanceID)
		if err == nil && status == "stopped" {
			fmt.Println()
			fmt.Println("✓ Instance is now stopped")
			return
		}
	}

	fmt.Println()
	fmt.Println("⚠ Instance is stopping (this may take a few moments)")
}
