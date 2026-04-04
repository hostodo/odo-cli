package instances

import (
	"fmt"

	"github.com/hostodo/hostodo-cli/pkg/resolver"
	"github.com/spf13/cobra"
)

// ReinstallCmd is a stub for the reinstall command
var ReinstallCmd = &cobra.Command{
	Use:               "reinstall <hostname>",
	Short:             "Reinstall the OS on an instance",
	ValidArgsFunction: resolver.CompleteHostname,
	Long: `Reinstall the operating system on a VPS instance.

This will wipe the instance and reinstall from scratch.

Examples:
  odo instances reinstall mybox`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("reinstall: not yet implemented")
		return nil
	},
}
