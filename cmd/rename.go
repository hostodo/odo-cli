package cmd

import (
	"fmt"

	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/hostodo/hostodo-cli/pkg/deploy"
	"github.com/hostodo/hostodo-cli/pkg/resolver"
	"github.com/spf13/cobra"
)

// renameCmd represents the rename command
var renameCmd = &cobra.Command{
	Use:               "rename <hostname> <new-hostname>",
	Short:             "Rename an instance",
	ValidArgsFunction: resolver.CompleteHostname,
	Long: `Rename a VPS instance by changing its hostname.

You can specify the instance by hostname, hostname prefix, or instance ID.

Examples:
  hostodo rename mybox newname        # Rename "mybox" to "newname"
  hostodo rename my newname           # Rename if "my" is an unambiguous prefix
  hostodo rename abc123 newname       # Rename by instance ID (fallback)`,
	Args: cobra.ExactArgs(2),
	Run:  runRename,
}

func runRename(cmd *cobra.Command, args []string) {
	identifier := args[0]
	newHostname := args[1]

	// Validate new hostname client-side
	if err := deploy.Validate(newHostname); err != nil {
		exitWithError("Invalid hostname %q: %v", newHostname, err)
	}

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

	fmt.Printf("Renaming instance %s (%s) to %s...\n", instance.Hostname, instance.MainIP, newHostname)

	err = client.RenameInstance(instance.InstanceID, newHostname)
	if err != nil {
		exitWithError("Failed to rename instance: %v", err)
	}

	// Invalidate resolver cache since hostname changed
	resolver.InvalidateCache()

	fmt.Printf("✓ Instance renamed to %s\n", newHostname)
}
