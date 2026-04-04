package instances

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// InstancesCmd is the parent command for all instance operations
var InstancesCmd = &cobra.Command{
	Use:     "instances",
	Aliases: []string{"i", "ins"},
	Short:   "Manage VPS instances",
	Long: `Manage your VPS instances.

Examples:
  odo instances              # List all instances (default)
  odo instances list         # List all instances
  odo instances status mybox # Show instance details
  odo instances ssh mybox    # SSH to an instance
  odo instances start mybox  # Start an instance
  odo instances stop mybox   # Stop an instance
  odo instances restart mybox # Restart an instance
  odo instances rename mybox newname # Rename an instance
  odo instances deploy       # Deploy a new instance
  odo instances reinstall mybox # Reinstall OS on an instance`,
}

func init() {
	// Register all instance subcommands
	InstancesCmd.AddCommand(ListCmd)
	InstancesCmd.AddCommand(StatusCmd)
	InstancesCmd.AddCommand(StartCmd)
	InstancesCmd.AddCommand(StopCmd)
	InstancesCmd.AddCommand(RestartCmd)
	InstancesCmd.AddCommand(SSHCmd)
	InstancesCmd.AddCommand(RenameCmd)
	InstancesCmd.AddCommand(DeployCmd)
	InstancesCmd.AddCommand(ReinstallCmd)

	// Default to list when no subcommand given
	InstancesCmd.Run = func(cmd *cobra.Command, args []string) {
		runList(cmd, args)
	}
}

// exitWithError prints an error message to stderr and exits with status 1.
func exitWithError(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+msg+"\n", args...)
	os.Exit(1)
}
