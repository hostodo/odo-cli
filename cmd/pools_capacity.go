package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/x/term"
	"github.com/google/uuid"
	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/terminaltext"
	"github.com/spf13/cobra"
)

func init() {
	poolsCmd.AddCommand(newPoolsOptionsCommand(), newPoolsCheckoutCommand(true), newPoolsCheckoutCommand(false), newPoolsUpdateCommand(), newPoolsCancelCommand())
}

func newPoolsOptionsCommand() *cobra.Command {
	return &cobra.Command{
		Use: "options", Short: "List available Capacity plans and billing cycles", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (returnErr error) {
			defer func() { returnErr = poolHumanError(returnErr) }()
			client, err := poolsClient()
			if err != nil {
				return err
			}
			options, raw, err := client.GetResourcePoolOptions()
			if err != nil {
				return err
			}
			if jsonMode, _ := cmd.Flags().GetBool("json"); jsonMode {
				return printPrettyJSONTo(cmd.OutOrStdout(), raw)
			}
			out := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintf(out, "Billing cycles: %s\n", terminaltext.Clean(strings.Join(options.BillingCycles, ", ")))
			if options.CurrentPoolID != "" {
				fmt.Fprintf(out, "Current Capacity: %s\n", terminaltext.Clean(options.CurrentPoolID))
			}
			if len(options.Tiers) == 0 {
				fmt.Fprintln(out, "No Capacity plans available.")
			} else {
				fmt.Fprintln(out, "PLAN ID\tNAME\tMONTHLY\t6 MONTHS\tANNUALLY\t2 YEARS\t3 YEARS\tRAM MB\tVCPU\tDISK GB\tVMS\tIPS\tSTATUS\tSELF SERVE")
				for _, tier := range options.Tiers {
					fmt.Fprintf(out, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%s\t%t\n",
						tier.ID, terminaltext.Clean(tier.Name), poolMoney(tier.PriceMonthly.String()), poolMoney(tier.PriceSemiannually.String()),
						poolMoney(tier.PriceAnnually.String()), poolMoney(tier.PriceBiennially.String()), poolMoney(tier.PriceTriennially.String()),
						tier.RAMMB, tier.TotalVCPU, tier.DiskGB, tier.MaxInstances, tier.MaxIPs, terminaltext.Clean(tier.Flag), tier.SelfServe)
				}
			}
			return out.Flush()
		},
	}
}

func newPoolsCheckoutCommand(quoteOnly bool) *cobra.Command {
	var planID int
	var cycle, paymentMethod, paymentMethodID, promo, idempotencyKey string
	var yes bool
	name, short := "purchase", "Purchase or change Capacity after reviewing a quote"
	if quoteOnly {
		name, short = "quote", "Get a fresh Capacity purchase or tier-change quote"
	}
	cmd := &cobra.Command{
		Use: name + " --plan-id <id>", Short: short, Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (returnErr error) {
			defer func() { returnErr = poolHumanError(returnErr) }()
			req, err := buildPoolCheckoutRequest(planID, cycle, paymentMethod, paymentMethodID, promo, idempotencyKey, quoteOnly)
			if err != nil {
				return err
			}
			client, err := poolsClient()
			if err != nil {
				return err
			}
			jsonMode, _ := cmd.Flags().GetBool("json")
			return runPoolCheckout(cmd, client, req, yes, jsonMode)
		},
	}
	cmd.Flags().IntVar(&planID, "plan-id", 0, "Capacity plan ID from odo pools options (required)")
	cmd.Flags().StringVar(&cycle, "billing-cycle", "monthly", "monthly, semiannually, annually, biennially, or triennially")
	cmd.Flags().StringVar(&promo, "promo", "", "Promotional code")
	cmd.Flags().StringVar(&paymentMethod, "payment-method", "stripe_checkout", "stripe_checkout, paypal, alipay, crypto, credit, or saved_card")
	cmd.Flags().StringVar(&paymentMethodID, "payment-method-id", "", "Saved payment method ID (requires --payment-method saved_card)")
	if !quoteOnly {
		cmd.Long = short + `. Type the exact server confirmation phrase (and separate saved-card phrase) to confirm, or use --yes.
The quote and confirmation are written to stderr. Hosted checkout prints
the bare checkout URL to stdout; --json prints the raw checkout response instead.
Save the idempotency key printed to stderr and reuse it with --idempotency-key
when retrying the same purchase with the same arguments, API origin, and account. Token rotation is supported. Confirmed quote
snapshots are saved under ~/.odo/capacity-checkouts-v2 before checkout. A cached retry
reuses the original snapshot without fetching a new quote; an unused explicit key
fetches a fresh quote. Legacy ~/.odo/capacity-checkouts records are preserved but
never replayed or migrated; reconcile any legacy checkout before using a new key.
A v2 cache directory without its authentication key marker fails closed.
Keep retry records until any unknown outcome is resolved.`
		cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Key to reuse when retrying this purchase (generated if omitted)")
		cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Explicitly confirm purchase without prompting")
	}
	return cmd
}

func buildPoolCheckoutRequest(planID int, billingCycle, paymentMethod, paymentMethodID, promo, idempotencyKey string, quoteOnly bool) (api.ResourcePoolCheckoutRequest, error) {
	req := api.ResourcePoolCheckoutRequest{PlanID: planID, BillingCycle: billingCycle, Promocode: promo, QuoteOnly: quoteOnly}
	if planID <= 0 {
		return req, fmt.Errorf("a positive --plan-id is required")
	}
	switch billingCycle {
	case "monthly", "semiannually", "annually", "biennially", "triennially":
	default:
		return req, fmt.Errorf("invalid billing cycle %q", billingCycle)
	}
	if quoteOnly && paymentMethod == "" {
		paymentMethod = "stripe_checkout"
	}
	switch paymentMethod {
	case "saved_card":
		if strings.TrimSpace(paymentMethodID) == "" {
			return req, fmt.Errorf("--payment-method-id is required for saved_card")
		}
	case "stripe_checkout", "paypal", "alipay", "crypto", "credit":
		if paymentMethodID != "" {
			return req, fmt.Errorf("--payment-method-id requires --payment-method saved_card")
		}
	default:
		return req, fmt.Errorf("unsupported payment method %q", paymentMethod)
	}
	req.PaymentMethod, req.PaymentMethodID = paymentMethod, paymentMethodID
	if !quoteOnly {
		req.IdempotencyKey = idempotencyKey
	}
	return req, nil
}

func runPoolCheckout(cmd *cobra.Command, client *api.Client, req api.ResourcePoolCheckoutRequest, yes, jsonMode bool) (returnErr error) {
	defer func() { returnErr = poolHumanError(returnErr) }()
	var cache *poolRetryCache
	var snapshot *api.ResourcePoolExpectedQuote
	if !req.QuoteOnly {
		explicitKey := strings.TrimSpace(req.IdempotencyKey) != ""
		if !explicitKey {
			key, err := uuid.NewRandom()
			if err != nil {
				return fmt.Errorf("generate idempotency key: %w", err)
			}
			req.IdempotencyKey = key.String()
		}
		var err error
		cache, err = newPoolRetryCache(client, req)
		if err != nil {
			return err
		}
		snapshot, err = cache.load()
		if err != nil {
			return err
		}
		if explicitKey && snapshot == nil {
			if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "No local quote snapshot for this idempotency key; treating it as first use and fetching a fresh quote. If this key was used elsewhere, stop and recover its original retry record before retrying."); err != nil {
				return err
			}
		}
	}
	cachedRetry := snapshot != nil
	if cachedRetry {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Retrying with the saved confirmed Capacity quote; no fresh quote fetched.\nCapacity quote (saved)\n  Mode: %s\n  Existing Capacity: %s\n  Unit price: %s\n  Recurring amount: %s\n  Amount due after credit: %s\n",
			terminaltext.Clean(snapshot.Mode), terminaltext.Clean(poolRetryExistingID(snapshot.ExistingPoolID)), poolMoney(snapshot.UnitPrice.String()), poolMoney(snapshot.RecurringAmount.String()), poolMoney(snapshot.AmountDueAfterCredit.String())); err != nil {
			return err
		}
	} else {
		quoteReq, err := buildPoolCheckoutRequest(req.PlanID, req.BillingCycle, req.PaymentMethod, req.PaymentMethodID, req.Promocode, "", true)
		if err != nil {
			return err
		}
		quote, raw, err := client.CheckoutResourcePool(quoteReq)
		if err != nil {
			return fmt.Errorf("Capacity quote failed: %w", err)
		}
		if req.QuoteOnly {
			if jsonMode {
				return printPrettyJSONTo(cmd.OutOrStdout(), raw)
			}
			_, err := fmt.Fprint(cmd.OutOrStdout(), formatPoolQuote(*quote))
			return err
		}
		snapshot = &api.ResourcePoolExpectedQuote{
			Mode:                 quote.Mode,
			ExistingPoolID:       quote.ExistingPoolID,
			UnitPrice:            quote.UnitPrice,
			RecurringAmount:      quote.RecurringAmount,
			AmountDueAfterCredit: quote.AmountDueAfterCredit,
		}
		if err := validatePoolRetryQuote(snapshot); err != nil {
			return err
		}
		if err := validatePoolConfirmation(quote.Confirmation); err != nil {
			return err
		}
		cache.confirmation = quote.Confirmation
		if req.PaymentMethod == "saved_card" {
			lastFour, err := poolCardEnding(req.PaymentMethodID, snapshot.AmountDueAfterCredit, quote.PaymentConfirmation)
			if err != nil {
				return err
			}
			cache.cardLastFour = lastFour
		}
		if _, err := fmt.Fprint(cmd.ErrOrStderr(), formatPoolQuote(*quote)); err != nil {
			return err
		}
	}
	message := fmt.Sprintf("Purchase Capacity for %s using %s?", poolMoney(snapshot.AmountDueAfterCredit.String()), req.PaymentMethod)
	if cachedRetry {
		message = fmt.Sprintf("Retry the prior Capacity checkout for %s using %s with its saved quote?", poolMoney(snapshot.AmountDueAfterCredit.String()), req.PaymentMethod)
	}
	if err := validatePoolConfirmation(cache.confirmation); err != nil {
		return cache.failure(err)
	}
	req.Confirmation = cache.confirmation
	input := bufio.NewReader(cmd.InOrStdin())
	if err := confirmPoolPurchase(input, cmd.ErrOrStderr(), yes, poolInputIsTerminal(cmd), message, req.Confirmation); err != nil {
		return err
	}
	if req.PaymentMethod == "saved_card" {
		confirmation, err := poolCardConfirmation(req.PaymentMethodID, snapshot.AmountDueAfterCredit, cache.cardLastFour)
		if err != nil {
			return cache.failure(err)
		}
		if err := confirmPoolCard(input, cmd.ErrOrStderr(), yes, poolInputIsTerminal(cmd), message, confirmation); err != nil {
			return err
		}
		req.PaymentConfirmation = confirmation
		req.ApprovedChargeAmount = snapshot.AmountDueAfterCredit
	}
	req.ExpectedQuote = snapshot
	if !cachedRetry {
		if err := cache.save(snapshot); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Confirmed quote saved for retries: %s\n", terminaltext.Clean(cache.path)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Idempotency key: %s\n", terminaltext.Clean(req.IdempotencyKey)); err != nil {
		return err
	}
	result, raw, err := client.CheckoutResourcePool(req)
	if err != nil {
		return fmt.Errorf("Capacity checkout failed (idempotency key: %s): %w", terminaltext.Clean(req.IdempotencyKey), err)
	}
	if result.IdempotentReplay {
		if _, err := fmt.Fprint(cmd.ErrOrStderr(), formatPoolReplay(*result)); err != nil {
			return err
		}
		if jsonMode {
			safe, err := marshalPoolReplay(*result)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(safe)
			return err
		}
		return nil
	}
	if jsonMode {
		return printPrettyJSONTo(cmd.OutOrStdout(), raw)
	}
	if result.CheckoutURL != "" {
		if err := validatePoolCheckoutURL(result.CheckoutURL); err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), result.CheckoutURL)
		return err
	}
	_, err = fmt.Fprintf(cmd.ErrOrStderr(), "Capacity checkout submitted\n  Plan: %s\n  Order: %s\n  Invoice: %s\n  Amount due: %s\n  Payment: %s\n",
		terminaltext.Clean(result.PlanName), terminaltext.Clean(result.OrderNumber), terminaltext.Clean(result.InvoiceNumber), poolMoney(result.AmountDue.String()), terminaltext.Clean(result.PaymentMethod))
	return err
}

func newPoolsUpdateCommand() *cobra.Command {
	var displayName string
	var autorenew bool
	cmd := &cobra.Command{
		Use: "update <pool_id>", Short: "Update a Capacity display name or Autorenew setting", Args: cobra.ExactArgs(1),
		Example: "  odo pools update pool::abc --display-name Production\n  odo pools update pool::abc --autorenew=false\n  odo pools update pool::abc --autorenew=true",
		RunE: func(cmd *cobra.Command, args []string) (returnErr error) {
			defer func() { returnErr = poolHumanError(returnErr) }()
			req, err := buildPoolUpdateRequest(displayName, autorenew, cmd.Flags().Changed("display-name"), cmd.Flags().Changed("autorenew"))
			if err != nil {
				return err
			}
			client, err := poolsClient()
			if err != nil {
				return err
			}
			pool, raw, err := client.UpdateResourcePool(args[0], req)
			if err != nil {
				return err
			}
			if jsonMode, _ := cmd.Flags().GetBool("json"); jsonMode {
				return printPrettyJSONTo(cmd.OutOrStdout(), raw)
			}
			autorenewLabel := "off"
			if pool.AutorenewalEnabled {
				autorenewLabel = "on"
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Capacity updated: %s\n  Display name: %s\n  Autorenew: %s\n", terminaltext.Clean(pool.PoolID), terminaltext.Clean(pool.DisplayName), autorenewLabel)
			return err
		},
	}
	cmd.Flags().StringVar(&displayName, "display-name", "", "Display name (use an empty string to clear)")
	cmd.Flags().BoolVar(&autorenew, "autorenew", false, "Set Autorenew explicitly with --autorenew=true or --autorenew=false")
	return cmd
}

func buildPoolUpdateRequest(displayName string, autorenew, nameChanged, autorenewChanged bool) (api.ResourcePoolUpdateRequest, error) {
	var req api.ResourcePoolUpdateRequest
	if !nameChanged && !autorenewChanged {
		return req, fmt.Errorf("at least one of --display-name or --autorenew is required")
	}
	if nameChanged {
		req.DisplayName = &displayName
	}
	if autorenewChanged {
		req.AutorenewalEnabled = &autorenew
	}
	return req, nil
}

func newPoolsCancelCommand() *cobra.Command {
	var yes bool
	var reason string
	cmd := &cobra.Command{
		Use: "cancel <pool_id>", Short: "Permanently cancel Capacity and its member instances", Args: cobra.ExactArgs(1),
		Long: "Permanently cancel Capacity and its member instances. Type CANCEL to confirm, or use --yes.",
		RunE: func(cmd *cobra.Command, args []string) (returnErr error) {
			defer func() { returnErr = poolHumanError(returnErr) }()
			client, err := poolsClient()
			if err != nil {
				return err
			}
			jsonMode, _ := cmd.Flags().GetBool("json")
			return runPoolCancel(cmd, client, args[0], reason, yes, jsonMode)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Explicitly confirm cancellation without prompting")
	cmd.Flags().StringVar(&reason, "reason", "", "Reason for cancellation")
	return cmd
}

func runPoolCancel(cmd *cobra.Command, client *api.Client, poolID, reason string, yes, jsonMode bool) (returnErr error) {
	defer func() { returnErr = poolHumanError(returnErr) }()
	message := fmt.Sprintf("Permanently cancel Capacity %s and its member instances?", poolID)
	if err := confirmAction(cmd.InOrStdin(), cmd.ErrOrStderr(), yes, poolInputIsTerminal(cmd), message, "CANCEL"); err != nil {
		return err
	}
	result, raw, err := client.CancelResourcePool(poolID, api.ResourcePoolCancelRequest{Confirm: true, Reason: reason})
	if err != nil {
		return err
	}
	if jsonMode {
		return printPrettyJSONTo(cmd.OutOrStdout(), raw)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Capacity %s: %s\nCancelled member instances: %d\n", terminaltext.Clean(result.PoolID), terminaltext.Clean(result.Status), len(result.CancelledMembers))
	return err
}

func poolInputIsTerminal(cmd *cobra.Command) bool {
	input, ok := cmd.InOrStdin().(interface{ Fd() uintptr })
	return ok && term.IsTerminal(input.Fd())
}

func confirmAction(in io.Reader, out io.Writer, yes, interactive bool, message, acknowledgement string) error {
	if yes {
		return nil
	}
	if !interactive {
		return fmt.Errorf("confirmation requires an interactive terminal; use --yes to explicitly confirm")
	}
	if _, err := fmt.Fprintf(out, "%s\nType %s to confirm: ", terminaltext.Clean(message), terminaltext.Clean(acknowledgement)); err != nil {
		return err
	}
	reader, ok := in.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(in)
	}
	answer, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("action aborted: could not read confirmation: %w", err)
	}
	answer = strings.TrimSuffix(strings.TrimSuffix(answer, "\n"), "\r")
	if answer != acknowledgement {
		return fmt.Errorf("action aborted: exact confirmation %s required", acknowledgement)
	}
	return nil
}

func formatPoolQuote(quote api.ResourcePoolCheckoutResponse) string {
	mode := quote.Mode
	switch mode {
	case "purchase":
		mode = "Purchase"
	case "upgrade":
		mode = "Upgrade"
	}
	return fmt.Sprintf("Capacity quote\n  Mode: %s\n  Plan: %s (ID %d)\n  Billing cycle: %s\n  Amount due after credit: %s\n  Recurring amount: %s\n  Credits applied: %s\n  Next due date: %s\n",
		terminaltext.Clean(mode), terminaltext.Clean(quote.PlanName), quote.PlanID, terminaltext.Clean(quote.BillingCycle), poolMoney(quote.AmountDueAfterCredit.String()),
		poolMoney(quote.RecurringAmount.String()), poolMoney(quote.CreditsApplied.String()), valueOrDash(quote.NextDueDate))
}

// Keep API decimal strings intact; converting money through float64 loses precision.
func poolMoney(amount string) string {
	if amount == "" {
		return "-"
	}
	return "$" + terminaltext.Clean(amount)
}
