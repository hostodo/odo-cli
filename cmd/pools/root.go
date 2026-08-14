package pools

import (
	"fmt"

	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/auth"
	"github.com/hostodo/odo-cli/v2/pkg/config"
	"github.com/spf13/cobra"
)

// PoolsCmd is the parent command for Hostodo pool operations.
var PoolsCmd = &cobra.Command{
	Use:     "pools",
	Aliases: []string{"pool", "capacity"},
	Short:   "Manage Hostodo resource pools",
	Long: `Manage Hostodo capacity pools: list quota, buy or upgrade a pool, and create $0 VMs inside it.

Examples:
  odo pools                    # List your pools
  odo pools show pool::abc     # Show pool quota and member VMs
  odo pools options            # List available pool tiers
  odo pools buy                # Buy a Hostodo pool
  odo pools upgrade            # Upgrade an existing pool
  odo pools vm                 # Create a VM inside a pool`,
}

func init() {
	PoolsCmd.AddCommand(listCmd)
	PoolsCmd.AddCommand(showCmd)
	PoolsCmd.AddCommand(optionsCmd)
	PoolsCmd.AddCommand(buyCmd)
	PoolsCmd.AddCommand(upgradeCmd)
	PoolsCmd.AddCommand(vmCmd)

	PoolsCmd.Flags().BoolVar(&poolsJSONFlag, "json", false, "Output as JSON")
	PoolsCmd.RunE = runList
}

func newAuthenticatedClient() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if !auth.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated. Run 'odo login' first")
	}
	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	return client, nil
}

func completePoolID(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	client, err := newAuthenticatedClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	pools, err := client.ListResourcePools()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ids := make([]string, 0, len(pools))
	for _, pool := range pools {
		ids = append(ids, pool.PoolID)
	}
	return ids, cobra.ShellCompDirectiveNoFileComp
}

func printJSON(v interface{}) error {
	data, err := marshalIndent(v)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
