package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/hostodo/hostodo-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var payYesFlag bool

var payCmd = &cobra.Command{
	Use:   "pay [invoice-number]",
	Short: "Pay an invoice",
	Long: `Pay an invoice using your default payment method.

Example:
  hostodo pay INV-12345
  hostodo pay INV-12345 --yes    # Skip confirmation`,
	Args: cobra.ExactArgs(1),
	RunE: runPay,
}

func init() {
	payCmd.Flags().BoolVarP(&payYesFlag, "yes", "y", false, "Skip confirmation prompt")
}

func runPay(cmd *cobra.Command, args []string) error {
	invoiceNumber := args[0]

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

	// Confirmation prompt (unless --yes)
	if !payYesFlag {
		confirmMsg := fmt.Sprintf("Pay invoice %s using your default payment method?", invoiceNumber)
		var confirmed bool
		prompt := &survey.Confirm{
			Message: confirmMsg,
			Default: false,
		}
		if err := survey.AskOne(prompt, &confirmed); err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Payment cancelled.")
			return nil
		}
	}

	// Pay invoice
	paymentResp, err := client.PayInvoice(invoiceNumber)
	if err != nil {
		// Payment failed - show error with dashboard link
		errorMsg := ui.ErrorStyle.Render(fmt.Sprintf("Payment failed: %s", err.Error()))
		fmt.Println(errorMsg)
		fmt.Println("\nUpdate your payment method at https://console.hostodo.com/billing")
		return err
	}

	// Payment successful - format payment method display
	paymentMethodDisplay := paymentResp.BillingIntegration
	if paymentResp.BillingIntegration == "" {
		paymentMethodDisplay = "Default payment method"
	}

	// Check if it's a checkout URL (Stripe/PayPal redirect)
	if paymentResp.StripeCheckoutURL != "" {
		fmt.Println(ui.SuccessStyle.Render("Payment initiated!"))
		fmt.Printf("\nComplete payment at: %s\n", paymentResp.StripeCheckoutURL)
		return nil
	}

	// Direct payment (credit or stored card) - show receipt
	receipt := ui.FormatPaymentReceipt(
		invoiceNumber,
		paymentResp.Amount,
		paymentMethodDisplay,
		paymentResp.TransactionID,
	)
	fmt.Println(receipt)

	return nil
}
