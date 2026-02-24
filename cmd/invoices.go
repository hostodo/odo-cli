package cmd

import (
	"fmt"

	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/hostodo/hostodo-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var statusFlag string

var invoicesCmd = &cobra.Command{
	Use:     "invoices",
	Short:   "List your invoices",
	Aliases: []string{"bills"},
	Long: `List your invoices with optional filtering.

By default, shows all invoices. Use --status=unpaid to filter by status.

Examples:
  # List all invoices
  hostodo invoices

  # List unpaid invoices only
  hostodo invoices --status=unpaid`,
	RunE: runInvoices,
}

func init() {
	invoicesCmd.Flags().StringVar(&statusFlag, "status", "", "Filter by status (e.g., 'unpaid')")
}

func runInvoices(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check authentication
	if !auth.IsAuthenticated() {
		return fmt.Errorf("not authenticated. Run 'hostodo login' first")
	}

	// Create API client
	client, err := api.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Fetch invoices
	invoices, err := client.ListInvoices(statusFlag)
	if err != nil {
		return fmt.Errorf("failed to list invoices: %w", err)
	}

	// Format and display
	output := ui.FormatInvoicesTable(invoices)
	fmt.Println(output)

	return nil
}
