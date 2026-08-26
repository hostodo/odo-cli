package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/auth"
	"github.com/hostodo/odo-cli/v2/pkg/config"
	"github.com/spf13/cobra"
)

var (
	poolsJSON   bool
	poolsSimple bool
)

type poolPlanSummary struct {
	Name string `json:"name"`
}

type poolSummary struct {
	ID            string           `json:"id"`
	PoolID        string           `json:"pool_id"`
	Status        string           `json:"status"`
	PlanName      string           `json:"plan_name"`
	Plan          *poolPlanSummary `json:"plan"`
	UsedRAMMB     int              `json:"used_ram_mb"`
	TotalRAMMB    int              `json:"total_ram_mb"`
	UsedDiskGB    int              `json:"used_disk_gb"`
	TotalDiskGB   int              `json:"total_disk_gb"`
	UsedInstances int              `json:"used_instances"`
	MaxInstances  int              `json:"max_instances"`
}

type poolsListResponse struct {
	Count   int           `json:"count"`
	Results []poolSummary `json:"results"`
}

var poolsCmd = &cobra.Command{
	Use:   "pools",
	Short: "Manage Hostodo capacity subscriptions",
	Long:  "List and inspect Hostodo capacity subscriptions (resource pools).",
}

var poolsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List capacity subscriptions",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPoolsList()
	},
}

var poolsShowCmd = &cobra.Command{
	Use:   "show <pool_id>",
	Short: "Show a capacity subscription",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPoolsShow(args[0])
	},
}

func init() {
	poolsCmd.PersistentFlags().BoolVar(&poolsJSON, "json", false, "JSON output")
	poolsCmd.PersistentFlags().BoolVar(&poolsSimple, "simple", false, "simple table output")
	poolsCmd.AddCommand(poolsListCmd)
	poolsCmd.AddCommand(poolsShowCmd)
}

func poolsClient() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if !auth.IsAuthenticated() {
		return nil, api.ErrNotAuthenticated
	}
	return api.NewClient(cfg)
}

func runPoolsList() error {
	client, err := poolsClient()
	if err != nil {
		return err
	}
	resp, err := client.Get("/client/resource-pools/")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}
	if poolsSimple && !poolsJSON {
		var payload poolsListResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return err
		}
		printPoolsSimple(payload.Results)
		return nil
	}
	return printPrettyJSON(body)
}

func runPoolsShow(poolID string) error {
	client, err := poolsClient()
	if err != nil {
		return err
	}
	resp, err := client.Get("/client/resource-pools/" + poolID + "/")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}
	if poolsSimple && !poolsJSON {
		var pool poolSummary
		if err := json.Unmarshal(body, &pool); err != nil {
			return err
		}
		printPoolsSimple([]poolSummary{pool})
		return nil
	}
	return printPrettyJSON(body)
}

func printPrettyJSON(body []byte) error {
	var out bytes.Buffer
	if err := json.Indent(&out, body, "", "  "); err != nil {
		return err
	}
	out.WriteByte('\n')
	_, err := out.WriteTo(os.Stdout)
	return err
}

func printPoolsSimple(pools []poolSummary) {
	if len(pools) == 0 {
		fmt.Println("No capacity subscriptions found.")
		return
	}
	fmt.Printf("%-18s %-12s %-14s %-12s %-12s %-8s\n", "POOL", "STATUS", "PLAN", "RAM_MB", "DISK_GB", "VMS")
	for _, pool := range pools {
		fmt.Printf("%-18s %-12s %-14s %-12s %-12s %-8s\n",
			poolIdentifier(pool),
			valueOrDash(pool.Status),
			poolPlanName(pool),
			fmt.Sprintf("%d/%d", pool.UsedRAMMB, pool.TotalRAMMB),
			fmt.Sprintf("%d/%d", pool.UsedDiskGB, pool.TotalDiskGB),
			fmt.Sprintf("%d/%d", pool.UsedInstances, pool.MaxInstances),
		)
	}
}

func poolIdentifier(pool poolSummary) string {
	if pool.PoolID != "" {
		return pool.PoolID
	}
	return valueOrDash(pool.ID)
}

func poolPlanName(pool poolSummary) string {
	if pool.PlanName != "" {
		return pool.PlanName
	}
	if pool.Plan != nil && pool.Plan.Name != "" {
		return pool.Plan.Name
	}
	return "-"
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
