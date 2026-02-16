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

var restartForce bool

// restartCmd represents the restart command
var restartCmd = &cobra.Command{
	Use:               "restart <hostname>",
	Short:             "Restart an instance",
	ValidArgsFunction: resolver.CompleteHostname,
	Long: `Restart a VPS instance.

This command will gracefully restart the instance. Use --force for immediate restart.
You can specify the instance by hostname, hostname prefix, or instance ID.

Examples:
  hostodo restart mybox              # Restart instance with hostname "mybox"
  hostodo restart my                 # Restart if "my" is an unambiguous prefix
  hostodo restart mybox --force      # Force immediate restart
  hostodo restart abc123             # Restart by instance ID (fallback)`,
	Args: cobra.ExactArgs(1),
	Run:  runRestart,
}

func init() {
	restartCmd.Flags().BoolVarP(&restartForce, "force", "f", false, "Force immediate restart")
}

func runRestart(cmd *cobra.Command, args []string) {
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

	// Restart instance
	if restartForce {
		fmt.Printf("Force restarting instance %s (%s)...\n", instance.Hostname, instance.MainIP)
	} else {
		fmt.Printf("Restarting instance %s (%s)...\n", instance.Hostname, instance.MainIP)
	}

	err = client.RebootInstance(instance.InstanceID)
	if err != nil {
		exitWithError("Failed to restart instance: %v", err)
	}

	fmt.Println("✓ Instance restart command sent successfully")
	fmt.Println("  The instance is now rebooting...")
	fmt.Print("\nWaiting for instance to restart")

	// Poll for status (up to 90 seconds)
	sawStopped := false
	for i := 0; i < 90; i++ {
		fmt.Print(".")
		time.Sleep(1 * time.Second)

		status, err := client.GetInstancePowerStatus(instance.InstanceID)
		if err == nil {
			if status == "stopped" {
				sawStopped = true
			} else if status == "running" && sawStopped {
				fmt.Println()
				fmt.Println("✓ Instance has restarted and is now running")
				return
			}
		}
	}

	fmt.Println()
	fmt.Println("⚠ Instance is restarting (this may take a few moments)")
}
