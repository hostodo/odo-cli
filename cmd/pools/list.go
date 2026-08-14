package pools

import (
	"encoding/json"
	"fmt"

	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	poolsJSONFlag bool
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List your Hostodo pools",
	Long: `List Hostodo pools with quota and usage.

Examples:
  odo pools
  odo pools list
  odo pools list --json`,
	RunE: runList,
}

var showCmd = &cobra.Command{
	Use:               "show [pool-id]",
	Aliases:           []string{"get", "status"},
	Short:             "Show a Hostodo pool and its member VMs",
	ValidArgsFunction: completePoolID,
	Args:              cobra.ExactArgs(1),
	RunE:              runShow,
}

func init() {
	listCmd.Flags().BoolVar(&poolsJSONFlag, "json", false, "Output as JSON")
	showCmd.Flags().BoolVar(&poolsJSONFlag, "json", false, "Output as JSON")
}

func runList(cmd *cobra.Command, args []string) error {
	client, err := newAuthenticatedClient()
	if err != nil {
		return err
	}

	pools, err := client.ListResourcePools()
	if err != nil {
		return fmt.Errorf("failed to list pools: %w", err)
	}

	if len(pools) == 0 {
		fmt.Println("No Hostodo pools found.")
		fmt.Println("Buy one with: odo pools buy")
		return nil
	}

	if poolsJSONFlag {
		return printJSON(pools)
	}

	fmt.Println(ui.FormatPoolsTable(pools))
	return nil
}

func runShow(cmd *cobra.Command, args []string) error {
	client, err := newAuthenticatedClient()
	if err != nil {
		return err
	}

	pool, err := resolvePool(client, args[0])
	if err != nil {
		return err
	}

	if poolsJSONFlag {
		return printJSON(pool)
	}

	fmt.Println(ui.FormatPoolDetail(pool))
	return nil
}

func resolvePool(client *api.Client, identifier string) (*api.ResourcePoolDetail, error) {
	pool, err := client.GetResourcePool(identifier)
	if err == nil && pool.PoolID != "" {
		return pool, nil
	}

	pools, listErr := client.ListResourcePools()
	if listErr != nil {
		if err != nil {
			return nil, fmt.Errorf("failed to get pool: %w", err)
		}
		return nil, listErr
	}

	var matches []api.ResourcePool
	for _, p := range pools {
		if p.PoolID == identifier || p.Label() == identifier {
			matches = append(matches, p)
			continue
		}
		if len(identifier) >= 4 && (hasPrefixFold(p.PoolID, identifier) || hasPrefixFold(p.Label(), identifier)) {
			matches = append(matches, p)
		}
	}
	if len(matches) == 1 {
		return client.GetResourcePool(matches[0].PoolID)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous pool %q — matches %d pools; use the full pool id", identifier, len(matches))
	}
	if err != nil {
		return nil, fmt.Errorf("resource pool not found: %s", identifier)
	}
	return nil, fmt.Errorf("resource pool not found: %s", identifier)
}

func hasPrefixFold(value, prefix string) bool {
	if len(value) < len(prefix) {
		return false
	}
	return equalFold(value[:len(prefix)], prefix)
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func marshalIndent(v interface{}) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return data, nil
}
