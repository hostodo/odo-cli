package pools

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	poolPlanFlag         string
	poolBillingCycleFlag string
	poolPromoFlag        string
	poolYesFlag          bool
	poolCheckoutJSONFlag bool
)

var buyCmd = &cobra.Command{
	Use:     "buy",
	Aliases: []string{"create", "new", "subscribe"},
	Short:   "Buy a Hostodo pool",
	Long: `Buy a Hostodo capacity pool. If you already have an active pool this becomes an upgrade.

Examples:
  odo pools buy
  odo pools buy --plan "Hostodo Nano" --billing-cycle monthly --yes
  odo pools buy --plan 12 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheckout(cmd, args, false)
	},
}

var upgradeCmd = &cobra.Command{
	Use:               "upgrade [pool-id]",
	Short:             "Upgrade a Hostodo pool",
	ValidArgsFunction: completePoolID,
	Args:              cobra.MaximumNArgs(1),
	Long: `Upgrade an existing Hostodo pool to a larger tier.

Examples:
  odo pools upgrade
  odo pools upgrade pool::abc --plan "Hostodo Micro" --yes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheckout(cmd, args, true)
	},
}

func init() {
	for _, cmd := range []*cobra.Command{buyCmd, upgradeCmd} {
		cmd.Flags().StringVar(&poolPlanFlag, "plan", "", "Pool plan name or ID")
		cmd.Flags().StringVar(&poolBillingCycleFlag, "billing-cycle", "", "Billing cycle (monthly, annually, semiannually, biennially, triennially)")
		cmd.Flags().StringVar(&poolPromoFlag, "promo", "", "Promo code")
		cmd.Flags().BoolVarP(&poolYesFlag, "yes", "y", false, "Skip confirmation")
		cmd.Flags().BoolVar(&poolCheckoutJSONFlag, "json", false, "JSON output (requires --plan)")
	}
}

func runCheckout(cmd *cobra.Command, args []string, upgrade bool) error {
	if poolCheckoutJSONFlag && poolPlanFlag == "" {
		return fmt.Errorf("JSON mode requires --plan")
	}

	client, err := newAuthenticatedClient()
	if err != nil {
		return err
	}

	options, err := client.ListPoolOptions()
	if err != nil {
		return fmt.Errorf("failed to list pool options: %w", err)
	}
	if len(options.Tiers) == 0 {
		return fmt.Errorf("no Hostodo pool tiers available")
	}

	poolID := ""
	if len(args) == 1 {
		pool, err := resolvePool(client, args[0])
		if err != nil {
			return err
		}
		poolID = pool.PoolID
	} else if options.CurrentPoolID != "" {
		poolID = options.CurrentPoolID
	}

	if upgrade && poolID == "" {
		return fmt.Errorf("no active Hostodo pool to upgrade. Buy one with: odo pools buy")
	}

	tiers := options.Tiers
	if upgrade {
		var upgradeTiers []api.PoolTier
		for _, tier := range options.Tiers {
			if tier.Flag == "current" || tier.IsCurrent {
				continue
			}
			upgradeTiers = append(upgradeTiers, tier)
		}
		if len(upgradeTiers) == 0 {
			return fmt.Errorf("no upgrade tiers available")
		}
		tiers = upgradeTiers
	}

	selected, err := selectPoolTier(tiers, poolPlanFlag, poolCheckoutJSONFlag)
	if err != nil {
		return err
	}

	cycle, err := selectPoolBillingCycle(options.BillingCycles, selected, poolBillingCycleFlag, poolCheckoutJSONFlag)
	if err != nil {
		return err
	}

	if poolPromoFlag == "" && !poolCheckoutJSONFlag && !poolYesFlag {
		poolPromoFlag, err = promptPoolPromo()
		if err != nil {
			return err
		}
	}

	quote, err := client.QuotePoolCheckout(api.PoolCheckoutRequest{
		PlanID:       selected.ID,
		BillingCycle: cycle,
		Promocode:    poolPromoFlag,
	})
	if err != nil {
		return fmt.Errorf("failed to quote pool: %w", err)
	}

	paymentMethod, err := client.GetDefaultPaymentMethod()
	if err != nil {
		return fmt.Errorf("failed to get payment method: %w", err)
	}

	if !poolYesFlag && !poolCheckoutJSONFlag {
		confirmed, err := confirmPoolCheckout(upgrade, selected, cycle, quote, paymentMethod, poolPromoFlag)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	req := api.PoolCheckoutRequest{
		PlanID:         selected.ID,
		TargetPlanID:   selected.ID,
		BillingCycle:   cycle,
		Promocode:      poolPromoFlag,
		IdempotencyKey: uuid.NewString(),
		Confirm:        true,
	}
	if paymentMethod != nil {
		req.PaymentMethod = "saved_card"
		req.PaymentMethodID = paymentMethod.PaymentMethodID
	} else {
		req.PaymentMethod = "stripe_checkout"
	}

	var result *api.PoolCheckoutResponse
	if upgrade {
		result, err = client.UpgradeResourcePool(poolID, req)
	} else {
		result, err = client.CheckoutResourcePool(req)
	}
	if err != nil {
		return fmt.Errorf("failed to %s pool: %w", checkoutVerb(upgrade), err)
	}

	if poolCheckoutJSONFlag {
		return printJSON(result)
	}

	mode := result.Mode
	if mode == "" {
		if upgrade {
			mode = "upgrade"
		} else {
			mode = "purchase"
		}
	}
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✓ Pool %s order created", mode)))
	if result.PlanName != "" {
		fmt.Printf("Plan:     %s\n", result.PlanName)
	}
	if result.OrderNumber != "" {
		fmt.Printf("Order:    %s\n", result.OrderNumber)
	}
	if result.InvoiceNumber != "" {
		fmt.Printf("Invoice:  %s\n", result.InvoiceNumber)
	}
	if result.AmountDue != "" {
		fmt.Printf("Due:      $%s\n", result.AmountDue)
	}
	checkoutURL := result.CheckoutURL
	if checkoutURL == "" && result.Checkout != nil {
		if url, ok := result.Checkout["url"].(string); ok {
			checkoutURL = url
		}
	}
	if checkoutURL != "" {
		fmt.Printf("Checkout: %s\n", checkoutURL)
	}
	fmt.Println("Create a VM with: odo pools vm")
	return nil
}

func checkoutVerb(upgrade bool) string {
	if upgrade {
		return "upgrade"
	}
	return "buy"
}

func selectPoolTier(tiers []api.PoolTier, flag string, jsonMode bool) (*api.PoolTier, error) {
	if flag != "" {
		tier, err := findPoolTier(tiers, flag)
		if err != nil {
			return nil, err
		}
		if tier == nil {
			names := make([]string, len(tiers))
			for i, t := range tiers {
				names[i] = t.Name
			}
			return nil, fmt.Errorf("no pool plan matching %q. Available: %s", flag, strings.Join(names, ", "))
		}
		return tier, nil
	}
	if jsonMode {
		return nil, fmt.Errorf("JSON mode requires --plan")
	}

	options := make([]string, len(tiers))
	indexByOption := map[string]int{}
	for i, t := range tiers {
		flagLabel := t.Flag
		if flagLabel == "" && t.IsCurrent {
			flagLabel = "current"
		}
		if flagLabel != "" {
			flagLabel = " [" + flagLabel + "]"
		}
		options[i] = fmt.Sprintf("[%d] %s  $%s/mo   %d vCPU, %s RAM, %d GB disk%s",
			t.ID, t.Name, t.PriceMonthly, t.TotalVCPU, formatRAMGB(t.RAMMB), t.DiskGB, flagLabel)
		indexByOption[options[i]] = i
	}
	var selected string
	err := huh.NewSelect[string]().
		Title("Choose a Hostodo pool:").
		Options(huh.NewOptions(options...)...).
		Value(&selected).
		Height(15).
		Run()
	if err != nil {
		return nil, err
	}
	idx, ok := indexByOption[selected]
	if !ok {
		return nil, fmt.Errorf("invalid pool selection")
	}
	return &tiers[idx], nil
}

func findPoolTier(tiers []api.PoolTier, name string) (*api.PoolTier, error) {
	if id, err := strconv.Atoi(name); err == nil {
		for i := range tiers {
			if tiers[i].ID == id {
				return &tiers[i], nil
			}
		}
	}
	for i := range tiers {
		if strings.EqualFold(tiers[i].Name, name) {
			return &tiers[i], nil
		}
	}
	lower := strings.ToLower(name)
	var matches []*api.PoolTier
	for i := range tiers {
		if strings.Contains(strings.ToLower(tiers[i].Name), lower) {
			matches = append(matches, &tiers[i])
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = m.Name
		}
		return nil, fmt.Errorf("ambiguous pool plan %q — matches: %s", name, strings.Join(names, ", "))
	}
	return nil, nil
}

func selectPoolBillingCycle(cycles []string, tier *api.PoolTier, flag string, jsonMode bool) (string, error) {
	available := filterPricedCycles(cycles, tier)
	if len(available) == 0 {
		available = []string{"monthly"}
	}
	if flag != "" {
		for _, c := range available {
			if strings.EqualFold(c, flag) {
				return c, nil
			}
		}
		return "", fmt.Errorf("invalid billing cycle %q. Available: %s", flag, strings.Join(available, ", "))
	}
	if jsonMode || len(available) == 1 {
		return available[0], nil
	}
	labels := make([]string, len(available))
	for i, c := range available {
		labels[i] = billingCycleLabel(c)
	}
	var selected string
	err := huh.NewSelect[string]().
		Title("Choose a billing cycle:").
		Options(huh.NewOptions(labels...)...).
		Value(&selected).
		Height(10).
		Run()
	if err != nil {
		return "", err
	}
	for _, c := range available {
		if billingCycleLabel(c) == selected {
			return c, nil
		}
	}
	return available[0], nil
}

func filterPricedCycles(cycles []string, tier *api.PoolTier) []string {
	if len(cycles) == 0 {
		cycles = []string{"monthly", "semiannually", "annually", "biennially", "triennially"}
	}
	var available []string
	for _, cycle := range cycles {
		if poolTierHasPricing(tier, cycle) {
			available = append(available, cycle)
		}
	}
	return available
}

func poolTierHasPricing(tier *api.PoolTier, cycle string) bool {
	price := poolPriceForCycle(tier, cycle)
	return price != "" && price != "0.00" && price != "0"
}

func poolPriceForCycle(tier *api.PoolTier, cycle string) string {
	if tier == nil {
		return ""
	}
	switch cycle {
	case "monthly":
		return tier.PriceMonthly
	case "annually":
		return tier.PriceAnnually
	case "semiannually":
		return tier.PriceSemiannually
	case "biennially":
		return tier.PriceBiennially
	case "triennially":
		return tier.PriceTriennially
	default:
		return tier.PriceMonthly
	}
}

func billingCycleLabel(cycle string) string {
	switch cycle {
	case "monthly":
		return "Monthly"
	case "annually":
		return "Annually"
	case "semiannually":
		return "Semi-Annually"
	case "biennially":
		return "Biennially"
	case "triennially":
		return "Triennially"
	default:
		return cycle
	}
}

func promptPoolPromo() (string, error) {
	var code string
	err := huh.NewInput().
		Title("Promo code (leave blank to skip):").
		Value(&code).
		Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

func confirmPoolCheckout(upgrade bool, tier *api.PoolTier, cycle string, quote *api.PoolQuote, pm *api.PaymentMethod, promo string) (bool, error) {
	action := "Buy"
	if upgrade || quote.Mode == "upgrade" {
		action = "Upgrade"
	}
	promoLine := ""
	if promo != "" {
		promoLine = fmt.Sprintf("\n  Promo:     %s", promo)
	}
	paymentLine := "Stripe Checkout"
	if pm != nil {
		paymentLine = fmt.Sprintf("%s ****%s", pm.CardType, pm.LastFour)
	}
	amount := quote.AmountDueAfterCredit.String()
	if amount == "" {
		amount = quote.UnitPrice.String()
	}
	recurring := quote.RecurringAmount.String()
	if recurring == "" {
		recurring = poolPriceForCycle(tier, cycle)
	}

	fmt.Printf(`
%s summary:
  Plan:      %s
  RAM:       %s
  vCPU:      %d (max %d per VM)
  Disk:      %d GB
  VMs:       %d
  Billing:   %s
  Due today: $%s
  Recurring: $%s
  Payment:   %s%s

`, action, tier.Name, formatRAMGB(tier.RAMMB), tier.TotalVCPU, tier.MaxVCPUPerInstance, tier.DiskGB, tier.MaxInstances, billingCycleLabel(cycle), amount, recurring, paymentLine, promoLine)

	confirmed := true
	err := huh.NewConfirm().
		Title(fmt.Sprintf("%s this pool for $%s?", action, amount)).
		Value(&confirmed).
		Run()
	if err != nil {
		return false, err
	}
	return confirmed, nil
}

func formatRAMGB(mb int) string {
	if mb <= 0 {
		return "0 GB"
	}
	if mb%1024 == 0 {
		return fmt.Sprintf("%d GB", mb/1024)
	}
	return fmt.Sprintf("%.1f GB", float64(mb)/1024.0)
}
