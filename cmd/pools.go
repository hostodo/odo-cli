package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/auth"
	"github.com/hostodo/odo-cli/v2/pkg/config"
	"github.com/hostodo/odo-cli/v2/pkg/terminaltext"
	"github.com/hostodo/odo-cli/v2/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	poolsJSON    bool
	poolsSimple  bool
	poolsDetails bool
)

// poolPlanSummary is the compact plan shape returned inside pool payloads.
type poolPlanSummary struct {
	Name string `json:"name"`
}

// poolSummary is the resource-pool shape needed for CLI list rendering.
type poolSummary struct {
	ID             string           `json:"id"`
	PoolID         string           `json:"pool_id"`
	Status         string           `json:"status"`
	PlanID         int              `json:"plan_id"`
	PlanName       string           `json:"plan_name"`
	Plan           *poolPlanSummary `json:"plan"`
	UsedRAMMB      int              `json:"used_ram_mb"`
	TotalRAMMB     int              `json:"total_ram_mb"`
	UsedDiskGB     int              `json:"used_disk_gb"`
	TotalDiskGB    int              `json:"total_disk_gb"`
	UsedInstances  int              `json:"used_instances"`
	MaxInstances   int              `json:"max_instances"`
	UsedIPv4       int              `json:"used_ips"`
	MaxIPv4        int              `json:"max_ips"`
	UsedBandwidth  int              `json:"used_bandwidth_gb"`
	TotalBandwidth int              `json:"total_bandwidth_gb"`
	DisplayName    string           `json:"display_name"`
	Autorenew      *bool            `json:"autorenewal_enabled"`
}

// Accept both the original flat pool payload and the current nested quotas.
func (pool *poolSummary) UnmarshalJSON(data []byte) error {
	type legacyPool poolSummary
	var payload struct {
		legacyPool
		Quota *api.ResourcePoolCapacity `json:"quota"`
		Usage *api.ResourcePoolCapacity `json:"usage"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	*pool = poolSummary(payload.legacyPool)
	if q := payload.Quota; q != nil {
		pool.TotalRAMMB, pool.TotalDiskGB, pool.MaxInstances, pool.MaxIPv4, pool.TotalBandwidth = q.RAMMB, q.DiskGB, q.Instances, q.IPs, q.BandwidthGB
	}
	if u := payload.Usage; u != nil {
		pool.UsedRAMMB, pool.UsedDiskGB, pool.UsedInstances, pool.UsedIPv4, pool.UsedBandwidth = u.RAMMB, u.DiskGB, u.Instances, u.IPs, u.BandwidthGB
	}
	return nil
}

// poolsListResponse is the paginated response from /client/resource-pools/.
type poolsListResponse struct {
	Count   int           `json:"count"`
	Results []poolSummary `json:"results"`
}

// poolsTableModel renders capacity subscriptions in a small interactive TUI.
type poolsTableModel struct {
	table table.Model
}

var poolsCmd = &cobra.Command{
	Use:   "pools",
	Short: "Manage Hostodo capacity subscriptions",
	Long: `Purchase and manage Hostodo capacity subscriptions (resource pools).

Use options to find plan IDs, quote to preview pricing, and purchase to create
or change Capacity. Use update for display names and Autorenew, or cancel to
permanently cancel Capacity and its member instances.

List/show Output Formats:
  - Interactive TUI (default) - Scrollable capacity table
  - JSON (--json)             - JSON format for scripting and automation
  - Simple (--simple)         - Static ASCII table for quick viewing
  - Details (--details)       - Detailed capacity quota and usage`,
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
	poolsCmd.PersistentFlags().BoolVar(&poolsJSON, "json", false, "Output as JSON")
	// Parent-local flags let Cobra recognize legacy flags before the subcommand;
	// list/show also accept them after it. They are not inherited by mutations.
	for _, cmd := range []*cobra.Command{poolsCmd, poolsListCmd, poolsShowCmd} {
		cmd.Flags().BoolVar(&poolsSimple, "simple", false, "Output as simple table")
		cmd.Flags().BoolVar(&poolsDetails, "details", false, "Show detailed information")
	}
	poolsCmd.AddCommand(poolsListCmd)
	poolsCmd.AddCommand(poolsShowCmd)
}

// poolsClient returns an authenticated Hostodo API client for pool commands.
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

// runPoolsList fetches and renders all capacity subscriptions for the user.
func runPoolsList() (returnErr error) {
	defer func() { returnErr = poolHumanError(returnErr) }()
	client, err := poolsClient()
	if err != nil {
		return err
	}
	_, body, err := client.ListResourcePools()
	if err != nil {
		return err
	}
	if poolsJSON {
		return printPrettyJSON(body)
	}
	var payload poolsListResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}
	return renderPools(payload.Results, payload.Count)
}

// runPoolsShow fetches and renders a single capacity subscription by pool_id.
func runPoolsShow(poolID string) (returnErr error) {
	defer func() { returnErr = poolHumanError(returnErr) }()
	client, err := poolsClient()
	if err != nil {
		return err
	}
	_, body, err := client.GetResourcePool(poolID)
	if err != nil {
		return err
	}
	if poolsJSON {
		return printPrettyJSON(body)
	}
	var pool poolSummary
	if err := json.Unmarshal(body, &pool); err != nil {
		return err
	}
	return renderPools([]poolSummary{pool}, 1)
}

// renderPools chooses the requested pool output mode.
func renderPools(pools []poolSummary, count int) error {
	if poolsSimple {
		fmt.Print(formatPoolsSimple(pools))
		if len(pools) > 0 {
			fmt.Printf("\nTotal: %d capacity subscriptions\n", count)
		}
		return nil
	}
	if poolsDetails {
		fmt.Print(formatPoolsDetails(pools))
		if len(pools) > 0 {
			fmt.Printf("\nTotal: %d capacity subscriptions\n", count)
		}
		return nil
	}
	return runPoolsTUI(pools)
}

// printPrettyJSON preserves the successful Capacity payload byte for byte.
func printPrettyJSON(body []byte) error {
	return printPrettyJSONTo(os.Stdout, body)
}

func printPrettyJSONTo(writer io.Writer, body []byte) error {
	_, err := writer.Write(body)
	return err
}

// formatPoolsSimple formats capacity subscriptions as a static table.
func formatPoolsSimple(pools []poolSummary) string {
	if len(pools) == 0 {
		return "No capacity subscriptions found.\n"
	}
	var sb strings.Builder
	header := fmt.Sprintf("%-18s %-12s %-14s %-12s %-12s %-8s\n", "POOL", "STATUS", "PLAN", "RAM_MB", "DISK_GB", "VMS")
	sb.WriteString(header)
	sb.WriteString(strings.Repeat("-", len(strings.TrimSuffix(header, "\n"))) + "\n")
	for _, pool := range pools {
		sb.WriteString(fmt.Sprintf("%-18s %-12s %-14s %-12s %-12s %-8s\n",
			truncatePool(poolIdentifier(pool), 18),
			truncatePool(valueOrDash(pool.Status), 12),
			truncatePool(poolPlanName(pool), 14),
			fmt.Sprintf("%d/%d", pool.UsedRAMMB, pool.TotalRAMMB),
			fmt.Sprintf("%d/%d", pool.UsedDiskGB, pool.TotalDiskGB),
			fmt.Sprintf("%d/%d", pool.UsedInstances, pool.MaxInstances),
		))
	}
	return sb.String()
}

// formatPoolsDetails formats capacity subscriptions with quota dimensions.
func formatPoolsDetails(pools []poolSummary) string {
	if len(pools) == 0 {
		return "No capacity subscriptions found.\n"
	}
	var sb strings.Builder
	for i, pool := range pools {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("Capacity: %s\n", poolIdentifier(pool)))
		if pool.DisplayName != "" {
			sb.WriteString(fmt.Sprintf("  Name:       %s\n", terminaltext.Clean(pool.DisplayName)))
		}
		if pool.Autorenew != nil {
			label := "off"
			if *pool.Autorenew {
				label = "on"
			}
			sb.WriteString(fmt.Sprintf("  Autorenew:  %s\n", label))
		}
		sb.WriteString(fmt.Sprintf("  Status:     %s\n", valueOrDash(pool.Status)))
		sb.WriteString(fmt.Sprintf("  Plan:       %s\n", poolPlanName(pool)))
		sb.WriteString(fmt.Sprintf("  RAM:        %d / %d MB\n", pool.UsedRAMMB, pool.TotalRAMMB))
		sb.WriteString(fmt.Sprintf("  Disk:       %d / %d GB\n", pool.UsedDiskGB, pool.TotalDiskGB))
		sb.WriteString(fmt.Sprintf("  VMs:        %d / %d\n", pool.UsedInstances, pool.MaxInstances))
		sb.WriteString(fmt.Sprintf("  IPv4:       %d / %d\n", pool.UsedIPv4, pool.MaxIPv4))
		if pool.TotalBandwidth > 0 || pool.UsedBandwidth > 0 {
			sb.WriteString(fmt.Sprintf("  Bandwidth:  %d / %d GB\n", pool.UsedBandwidth, pool.TotalBandwidth))
		}
	}
	return sb.String()
}

// runPoolsTUI launches the default interactive pool table.
func runPoolsTUI(pools []poolSummary) error {
	if len(pools) == 0 {
		fmt.Println("No capacity subscriptions found.")
		return nil
	}
	_, err := tea.NewProgram(newPoolsTableModel(pools)).Run()
	return err
}

// newPoolsTableModel builds the Bubble Tea table model for pool rows.
func newPoolsTableModel(pools []poolSummary) poolsTableModel {
	columns := []table.Column{
		{Title: "POOL", Width: 18},
		{Title: "STATUS", Width: 12},
		{Title: "PLAN", Width: 14},
		{Title: "RAM", Width: 13},
		{Title: "DISK", Width: 12},
		{Title: "VMS", Width: 8},
	}
	rows := make([]table.Row, 0, len(pools))
	for _, pool := range pools {
		rows = append(rows, table.Row{
			truncatePool(poolIdentifier(pool), 18),
			truncatePool(valueOrDash(pool.Status), 12),
			truncatePool(poolPlanName(pool), 14),
			fmt.Sprintf("%d/%d MB", pool.UsedRAMMB, pool.TotalRAMMB),
			fmt.Sprintf("%d/%d", pool.UsedDiskGB, pool.TotalDiskGB),
			fmt.Sprintf("%d/%d", pool.UsedInstances, pool.MaxInstances),
		})
	}
	t := table.New(table.WithColumns(columns), table.WithRows(rows), table.WithFocused(true), table.WithHeight(12))
	styles := table.DefaultStyles()
	styles.Header = styles.Header.Bold(true).Foreground(lipgloss.Color("#00D9FF"))
	styles.Selected = styles.Selected.Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#8B5CF6"))
	t.SetStyles(styles)
	return poolsTableModel{table: t}
}

// Init initializes the interactive pool table model.
func (m poolsTableModel) Init() tea.Cmd { return nil }

// Update handles keyboard input for the interactive pool table model.
func (m poolsTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the interactive pool table model.
func (m poolsTableModel) View() string {
	return "\n" + ui.TitleStyle.Render("Hostodo Capacity") + "\n\n" + m.table.View() + "\n\n" + ui.HelpStyle.Render("↑/↓: navigate • q: quit")
}

// poolIdentifier chooses the stable public pool identifier for display.
func poolIdentifier(pool poolSummary) string {
	if pool.PoolID != "" {
		return terminaltext.Clean(pool.PoolID)
	}
	return valueOrDash(pool.ID)
}

// poolPlanName prefers legacy plan names, falling back to the current plan ID.
func poolPlanName(pool poolSummary) string {
	if pool.PlanName != "" {
		return terminaltext.Clean(pool.PlanName)
	}
	if pool.Plan != nil && pool.Plan.Name != "" {
		return terminaltext.Clean(pool.Plan.Name)
	}
	if pool.PlanID > 0 {
		return fmt.Sprintf("Plan #%d", pool.PlanID)
	}
	return "-"
}

// valueOrDash returns '-' for empty API display fields.
func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return terminaltext.Clean(value)
}

// truncatePool shortens values to fit pool table columns.
func truncatePool(value string, width int) string {
	value = terminaltext.Clean(value)
	if len(value) <= width {
		return value
	}
	if width <= 3 {
		return value[:width]
	}
	return value[:width-3] + "..."
}
