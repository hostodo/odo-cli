package pools

import (
	"fmt"

	"github.com/hostodo/odo-cli/v2/pkg/ui"
	"github.com/spf13/cobra"
)

var optionsCmd = &cobra.Command{
	Use:     "options",
	Aliases: []string{"tiers", "plans"},
	Short:   "List Hostodo pool tiers",
	Long: `List available Hostodo pool tiers (Nano→Titan) and your current pool, if any.

Examples:
  odo pools options
  odo pools options --json`,
	RunE: runOptions,
}

func init() {
	optionsCmd.Flags().BoolVar(&poolsJSONFlag, "json", false, "Output as JSON")
}

func runOptions(cmd *cobra.Command, args []string) error {
	client, err := newAuthenticatedClient()
	if err != nil {
		return err
	}

	options, err := client.ListPoolOptions()
	if err != nil {
		return fmt.Errorf("failed to list pool options: %w", err)
	}

	if poolsJSONFlag {
		return printJSON(options)
	}

	fmt.Println(ui.FormatPoolOptionsTable(options))
	return nil
}
