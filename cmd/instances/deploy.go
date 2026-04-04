package instances

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/hostodo/hostodo-cli/pkg/deploy"
	"github.com/hostodo/hostodo-cli/pkg/ui"
	"github.com/hostodo/hostodo-cli/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	osFlag           string
	regionFlag       string
	planFlag         string
	hostnameFlag     string
	sshKeyFlag       string
	billingCycleFlag string
	yesFlag          bool
	jsonFlag         bool
)

// DeployCmd represents the deploy command
var DeployCmd = &cobra.Command{
	Use:     "deploy",
	Short:   "Deploy a new VPS instance",
	Aliases: []string{"new", "create"},
	Long: `Deploy a new VPS instance with interactive prompts or flags.

This command guides you through selecting an OS, region, and plan, then
provisions a new VPS instance. You can use flags to skip interactive prompts.

Examples:
  # Interactive mode (guided prompts)
  odo instances deploy

  # Skip prompts with flags
  odo instances deploy --os "Ubuntu 22.04" --region "Los Angeles" --plan KVM-2G

  # Custom hostname
  odo instances deploy --hostname my-server

  # Skip confirmation
  odo instances deploy --yes

  # JSON output (requires all selection flags)
  odo instances deploy --os "Ubuntu 22.04" --region "Los Angeles" --plan KVM-2G --json`,
	RunE: runDeploy,
}

func init() {
	DeployCmd.Flags().StringVar(&osFlag, "os", "", "OS template name (skips OS prompt)")
	DeployCmd.Flags().StringVar(&regionFlag, "region", "", "Region name (skips region prompt)")
	DeployCmd.Flags().StringVar(&planFlag, "plan", "", "Plan name (skips plan prompt)")
	DeployCmd.Flags().StringVar(&hostnameFlag, "hostname", "", "Custom hostname (skips auto-generation)")
	DeployCmd.Flags().StringVar(&sshKeyFlag, "ssh-key", "", "SSH key name to use for authentication")
	DeployCmd.Flags().StringVar(&billingCycleFlag, "billing-cycle", "", "Billing cycle (monthly, annually, semiannually, biennially, triennially)")
	DeployCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Skip confirmation prompt")
	DeployCmd.Flags().BoolVar(&jsonFlag, "json", false, "JSON output mode (requires --os, --region, --plan)")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	if jsonFlag && (osFlag == "" || regionFlag == "" || planFlag == "") {
		return fmt.Errorf("JSON mode requires --os, --region, and --plan flags")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if !auth.IsAuthenticated() {
		return fmt.Errorf("not authenticated. Run 'odo login' first")
	}
	client, err := api.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Fetch available options
	if !jsonFlag {
		fmt.Println("Loading available options...")
	}
	templates, err := client.ListTemplates()
	if err != nil {
		return fmt.Errorf("failed to load OS templates: %w", err)
	}
	regions, err := client.ListRegions()
	if err != nil {
		return fmt.Errorf("failed to load regions: %w", err)
	}
	plans, err := client.ListPlans()
	if err != nil {
		return fmt.Errorf("failed to load plans: %w", err)
	}

	// Selections
	selectedTemplate, err := selectTemplate(templates, osFlag, jsonFlag)
	if err != nil {
		return err
	}
	selectedRegion, err := selectRegion(regions, regionFlag, jsonFlag)
	if err != nil {
		return err
	}
	// Filter plans by selected region's plan categories
	regionCategoryIDs := make(map[int]bool)
	for _, cat := range selectedRegion.PlanCategories {
		regionCategoryIDs[cat.ID] = true
	}
	var regionPlans []api.Plan
	for _, plan := range plans {
		if regionCategoryIDs[plan.PlanCategoryID] {
			regionPlans = append(regionPlans, plan)
		}
	}
	if len(regionPlans) == 0 {
		return fmt.Errorf("no plans available for region %s", selectedRegion.Name)
	}

	selectedCycle, err := selectBillingCycle(regionPlans, billingCycleFlag, jsonFlag)
	if err != nil {
		return err
	}
	// Filter plans to only those with pricing for the selected billing cycle
	var filteredPlans []api.Plan
	for _, plan := range regionPlans {
		if planHasPricing(plan, selectedCycle) {
			filteredPlans = append(filteredPlans, plan)
		}
	}
	if len(filteredPlans) == 0 {
		return fmt.Errorf("no plans available for %s billing", selectedCycle)
	}
	selectedPlan, err := selectPlan(filteredPlans, planFlag, jsonFlag, selectedCycle)
	if err != nil {
		return err
	}
	hostname, err := resolveHostname(client, hostnameFlag)
	if err != nil {
		return err
	}
	sshKeyName, err := selectSSHKey(client, sshKeyFlag, jsonFlag)
	if err != nil {
		return err
	}

	// Quote and payment
	quote, err := client.GetQuote(api.QuoteRequest{
		PlanID:       selectedPlan.ID,
		BillingCycle: selectedCycle,
		Quantity:     1,
	})
	if err != nil {
		return fmt.Errorf("failed to get price quote: %w", err)
	}
	paymentMethod, err := client.GetDefaultPaymentMethod()
	if err != nil {
		return fmt.Errorf("failed to get payment method: %w", err)
	}
	if paymentMethod == nil {
		fmt.Println(ui.ErrorStyle.Render("No payment method on file. Add one at https://panel.hostodo.com/billing"))
		return fmt.Errorf("no payment method configured")
	}

	// Confirmation
	if !yesFlag && !jsonFlag {
		confirmed, err := confirmDeploy(selectedTemplate, selectedRegion, selectedPlan, hostname, quote, paymentMethod, selectedCycle)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Deployment cancelled.")
			return nil
		}
	}

	// Build request and deploy
	deployReq := api.DeployRequest{
		Hostname:        hostname,
		Region:          selectedRegion.Name,
		Template:        selectedTemplate.Name,
		Plan:            selectedPlan.Name,
		BillingCycle:    selectedCycle,
		SSHKey:          sshKeyName,
		PaymentMethodID: paymentMethod.PaymentMethodID,
		Quantity:        1,
	}

	instance, err := executeDeploy(client, deployReq, hostname, jsonFlag)
	if err != nil {
		return err
	}

	// Output
	if jsonFlag {
		return printJSONResult(instance, selectedRegion, selectedPlan)
	}
	displayDeployResult(instance, selectedRegion, selectedPlan)
	promptSSHConnect(hostname)
	return nil
}

// --- Selection helpers ---

func selectTemplate(templates []api.Template, flag string, jsonMode bool) (*api.Template, error) {
	if flag != "" {
		tmpl, err := findTemplate(templates, flag)
		if err != nil {
			return nil, err
		}
		if tmpl == nil {
			names := make([]string, len(templates))
			for i, t := range templates {
				names[i] = t.Name
			}
			return nil, fmt.Errorf("no OS template matching '%s'. Available: %s", flag, strings.Join(names, ", "))
		}
		return tmpl, nil
	}
	if jsonMode {
		return nil, fmt.Errorf("JSON mode requires --os flag")
	}
	osOptions := make([]string, len(templates))
	for i, t := range templates {
		osOptions[i] = t.Name
	}
	var selected string
	err := huh.NewSelect[string]().
		Title("Choose an OS:").
		Options(huh.NewOptions(osOptions...)...).
		Value(&selected).
		Height(15).
		Run()
	if err != nil {
		return nil, err
	}
	tmpl, _ := findTemplate(templates, selected)
	return tmpl, nil
}

func selectRegion(regions []api.Region, flag string, jsonMode bool) (*api.Region, error) {
	if flag != "" {
		region, err := findRegion(regions, flag)
		if err != nil {
			return nil, err
		}
		if region == nil {
			names := make([]string, len(regions))
			for i, r := range regions {
				names[i] = r.Name
			}
			return nil, fmt.Errorf("no region matching '%s'. Available: %s", flag, strings.Join(names, ", "))
		}
		return region, nil
	}
	if jsonMode {
		return nil, fmt.Errorf("JSON mode requires --region flag")
	}
	regionOptions := make([]string, len(regions))
	for i, r := range regions {
		regionOptions[i] = r.Name
	}
	var selected string
	err := huh.NewSelect[string]().
		Title("Choose a region:").
		Options(huh.NewOptions(regionOptions...)...).
		Value(&selected).
		Height(15).
		Run()
	if err != nil {
		return nil, err
	}
	region, _ := findRegion(regions, selected)
	return region, nil
}

func selectPlan(plans []api.Plan, flag string, jsonMode bool, billingCycle string) (*api.Plan, error) {
	if flag != "" {
		plan, err := findPlan(plans, flag)
		if err != nil {
			return nil, err
		}
		if plan == nil {
			names := make([]string, len(plans))
			for i, p := range plans {
				names[i] = p.Name
			}
			return nil, fmt.Errorf("no plan matching '%s'. Available: %s", flag, strings.Join(names, ", "))
		}
		return plan, nil
	}
	if jsonMode {
		return nil, fmt.Errorf("JSON mode requires --plan flag")
	}
	suffix := billingCycleSuffix(billingCycle)
	planOptions := make([]string, len(plans))
	for i, p := range plans {
		price := getPriceForCycle(p, billingCycle)
		planOptions[i] = fmt.Sprintf("%-12s $%s%s   %d vCPU, %sGB RAM, %dGB SSD, %sTB BW",
			p.Name, price, suffix, p.VCPU, formatRAM(p.RAM), p.Disk, formatBW(p.Bandwidth))
	}
	var selected string
	err := huh.NewSelect[string]().
		Title("Choose a plan:").
		Options(huh.NewOptions(planOptions...)...).
		Value(&selected).
		Height(15).
		Run()
	if err != nil {
		return nil, err
	}
	planName := strings.Fields(selected)[0]
	plan, _ := findPlan(plans, planName)
	return plan, nil
}

func resolveHostname(client *api.Client, flag string) (string, error) {
	if flag != "" {
		if err := deploy.Validate(flag); err != nil {
			return "", fmt.Errorf("invalid hostname: %w", err)
		}
		return flag, nil
	}
	hostname, err := deploy.Generate(client.CheckHostnameExists)
	if err != nil {
		return "", fmt.Errorf("failed to generate hostname: %w", err)
	}
	return hostname, nil
}

func selectSSHKey(client *api.Client, flag string, jsonMode bool) (string, error) {
	if flag != "" {
		return flag, nil
	}
	sshKeys, err := client.ListSSHKeys()
	if err != nil || len(sshKeys) == 0 {
		return "", nil
	}
	if len(sshKeys) == 1 {
		if !jsonMode {
			fmt.Printf("Using SSH key: %s\n", sshKeys[0].Name)
		}
		return sshKeys[0].Name, nil
	}
	// Multiple keys
	if jsonMode {
		return "", fmt.Errorf("multiple SSH keys found. Use --ssh-key flag to specify which one")
	}
	keyOptions := make([]string, len(sshKeys))
	for i, key := range sshKeys {
		fingerprint, err := utils.CalculateSSHFingerprint(key.PublicKey)
		if err != nil {
			fingerprint = "(error)"
		}
		keyOptions[i] = fmt.Sprintf("%s (%s)", key.Name, fingerprint)
	}
	var selected string
	err = huh.NewSelect[string]().
		Title("Choose an SSH key:").
		Options(huh.NewOptions(keyOptions...)...).
		Value(&selected).
		Height(10).
		Run()
	if err != nil {
		return "", err
	}
	return strings.Split(selected, " (")[0], nil
}

// --- Deploy execution ---

func confirmDeploy(tmpl *api.Template, region *api.Region, plan *api.Plan, hostname string, quote *api.QuoteResponse, pm *api.PaymentMethod, billingCycle string) (bool, error) {
	suffix := billingCycleSuffix(billingCycle)
	summary := fmt.Sprintf(`Deploy Summary:
  OS:       %s
  Region:   %s
  Plan:     %s (%d vCPU, %sGB RAM, %dGB SSD)
  Billing:  %s
  Hostname: %s
  Price:    $%s%s
  Payment:  %s ****%s`,
		tmpl.Name,
		region.Name,
		plan.Name,
		plan.VCPU,
		formatRAM(plan.RAM),
		plan.Disk,
		billingCycleLabel(billingCycle),
		hostname,
		quote.AmountDue,
		suffix,
		pm.CardType,
		pm.LastFour)

	fmt.Println("\n" + summary + "\n")

	confirmMsg := fmt.Sprintf("Deploy? (charges $%s to %s ****%s)",
		quote.AmountDue, pm.CardType, pm.LastFour)
	confirmed := true
	err := huh.NewConfirm().
		Title(confirmMsg).
		Value(&confirmed).
		Run()
	if err != nil {
		return false, err
	}
	return confirmed, nil
}

func executeDeploy(client *api.Client, req api.DeployRequest, hostname string, jsonMode bool) (*api.Instance, error) {
	if jsonMode {
		_, err := client.CreateDeployOrder(req)
		if err != nil {
			return nil, err
		}
		return pollForProvisioning(client, hostname, 12*time.Minute)
	}

	// Stage 1 - Creating order
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Creating order..."
	s.Start()

	deployResp, err := client.CreateDeployOrder(req)

	s.Stop()
	if err != nil {
		fmt.Println(ui.ErrorStyle.Render("✗ Order creation failed: " + err.Error()))
		return nil, err
	}
	fmt.Println(ui.SuccessStyle.Render("✓ Order created"))

	// Stage 2 - Payment processing (poll for Stripe webhook)
	s = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Processing payment..."
	s.Start()

	paid := deployResp.Invoice.Status == "paid"
	if !paid {
		invoiceNum := deployResp.Invoice.InvoiceNumber
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) && !paid {
			time.Sleep(2 * time.Second)
			invoices, err := client.ListInvoices("")
			if err != nil {
				continue
			}
			for _, inv := range invoices {
				if inv.InvoiceNumber == invoiceNum && inv.Status == "paid" {
					paid = true
					break
				}
			}
		}
	}

	if !paid {
		s.Stop()
		errorMsg := fmt.Sprintf("✗ Payment failed: %s", deployResp.Invoice.Status)
		fmt.Println(ui.ErrorStyle.Render(errorMsg))
		return nil, fmt.Errorf("payment failed: %s. No instance was created", deployResp.Invoice.Status)
	}

	s.Stop()
	fmt.Println(ui.SuccessStyle.Render("✓ Payment processed"))

	// Stage 3 - Provisioning with progress
	instance, err := pollForProvisioningWithProgress(client, hostname, 12*time.Minute)
	if err != nil {
		fmt.Println(ui.ErrorStyle.Render("✗ Provisioning timed out"))
		fmt.Println("Instance may still be provisioning. Check the web panel.")
		return nil, err
	}
	fmt.Println(ui.SuccessStyle.Render("✓ Server online"))
	return instance, nil
}

// --- Output helpers ---

func printJSONResult(instance *api.Instance, region *api.Region, plan *api.Plan) error {
	output := map[string]string{
		"hostname": instance.Hostname,
		"ip":       instance.MainIP,
		"region":   region.Name,
		"plan":     plan.Name,
		"status":   instance.Status,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func displayDeployResult(instance *api.Instance, region *api.Region, plan *api.Plan) {
	sshUser := instance.Template.DefaultUsername
	if sshUser == "" {
		sshUser = "root"
	}
	passwordDisplay := instance.DefaultPassword
	if passwordDisplay == "" {
		passwordDisplay = "(check email for password)"
	}

	cardContent := fmt.Sprintf(`Instance Deployed Successfully!

Hostname:       %s
IP Address:     %s
Root Password:  %s
Region:         %s
Plan:           %s

SSH:            ssh %s@%s`,
		instance.Hostname,
		instance.MainIP,
		passwordDisplay,
		region.Name,
		plan.Name,
		sshUser,
		instance.MainIP)

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2)

	fmt.Println("\n" + cardStyle.Render(cardContent) + "\n")
}

func promptSSHConnect(hostname string) {
	connectNow := true
	err := huh.NewConfirm().
		Title("Connect now?").
		Value(&connectNow).
		Run()
	if err != nil {
		return
	}
	if connectNow {
		// Re-exec as subprocess so os.Exit in RunSSH doesn't bypass our deferred cleanup
		binary, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		cmd := exec.Command(binary, "ssh", hostname)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}
}

// --- Search helpers ---

func findTemplate(templates []api.Template, name string) (*api.Template, error) {
	// Try exact match first (case-insensitive)
	for i := range templates {
		if strings.EqualFold(templates[i].Name, name) {
			return &templates[i], nil
		}
	}
	// Fall back to substring match — error if ambiguous
	lowerName := strings.ToLower(name)
	var matches []*api.Template
	for i := range templates {
		if strings.Contains(strings.ToLower(templates[i].Name), lowerName) {
			matches = append(matches, &templates[i])
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
		return nil, fmt.Errorf("ambiguous OS template '%s' — matches: %s", name, strings.Join(names, ", "))
	}
	return nil, nil
}

func findRegion(regions []api.Region, name string) (*api.Region, error) {
	// Try exact match first (case-insensitive)
	for i := range regions {
		if strings.EqualFold(regions[i].Name, name) {
			return &regions[i], nil
		}
	}
	// Fall back to substring match — error if ambiguous
	lowerName := strings.ToLower(name)
	var matches []*api.Region
	for i := range regions {
		if strings.Contains(strings.ToLower(regions[i].Name), lowerName) {
			matches = append(matches, &regions[i])
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
		return nil, fmt.Errorf("ambiguous region '%s' — matches: %s", name, strings.Join(names, ", "))
	}
	return nil, nil
}

func findPlan(plans []api.Plan, name string) (*api.Plan, error) {
	// Try exact match first (case-insensitive)
	for i := range plans {
		if strings.EqualFold(plans[i].Name, name) {
			return &plans[i], nil
		}
	}
	// Fall back to substring match — error if ambiguous
	lowerName := strings.ToLower(name)
	var matches []*api.Plan
	for i := range plans {
		if strings.Contains(strings.ToLower(plans[i].Name), lowerName) {
			matches = append(matches, &plans[i])
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
		return nil, fmt.Errorf("ambiguous plan '%s' — matches: %s", name, strings.Join(names, ", "))
	}
	return nil, nil
}

// --- Polling ---

func pollForProvisioning(client *api.Client, hostname string, timeout time.Duration) (*api.Instance, error) {
	startTime := time.Now()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			instancesResp, err := client.ListInstances(1000, 0)
			if err != nil {
				continue
			}

			for _, inst := range instancesResp.Results {
				if inst.Hostname == hostname {
					powerStatus, err := client.GetInstancePowerStatus(inst.InstanceID)
					if err == nil && powerStatus == "running" {
						inst.PowerStatus = powerStatus
						return &inst, nil
					}
					break
				}
			}

			if time.Since(startTime) > timeout {
				return nil, fmt.Errorf("provisioning timeout exceeded")
			}
		}
	}
}

func pollForProvisioningWithProgress(client *api.Client, hostname string, timeout time.Duration) (*api.Instance, error) {
	startTime := time.Now()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Waiting for instance..."
	s.Start()
	defer s.Stop()

	instanceFound := false
	var instanceID string
	seenEvents := make(map[int]bool)
	ipPrinted := false

	for {
		select {
		case <-ticker.C:
			instancesResp, err := client.ListInstances(1000, 0)
			if err != nil {
				continue
			}

			// Step 1: Find instance by hostname
			if !instanceFound {
				for _, inst := range instancesResp.Results {
					if inst.Hostname == hostname {
						instanceFound = true
						instanceID = inst.InstanceID

						s.Stop()
						fmt.Println(ui.SuccessStyle.Render("✓ Hostname set: " + hostname))

						if inst.MainIP != "" && !ipPrinted {
							fmt.Println(ui.SuccessStyle.Render("✓ Assigned IPv4: " + inst.MainIP))
							ipPrinted = true
						}

						s.Suffix = " Provisioning server..."
						s.Start()
						break
					}
				}
				if !instanceFound {
					if time.Since(startTime) > timeout {
						return nil, fmt.Errorf("provisioning timeout exceeded")
					}
					continue
				}
			}

			// Step 2: Poll events for progress updates
			events, err := client.ListInstanceEvents(instanceID)
			if err == nil {
				for i := len(events) - 1; i >= 0; i-- {
					ev := events[i]
					if seenEvents[ev.ID] {
						continue
					}
					seenEvents[ev.ID] = true

					msg := mapEventMessage(ev.ClientEventMessage)
					if msg != "" {
						s.Stop()
						if ev.Status == "failed" {
							fmt.Println(ui.ErrorStyle.Render("✗ " + msg))
						} else {
							fmt.Println(ui.SuccessStyle.Render("✓ " + msg))
						}
						s.Suffix = " Provisioning server..."
						s.Start()
					}
				}
			}

			// Step 3: Check if IP appeared
			if !ipPrinted {
				for _, inst := range instancesResp.Results {
					if inst.InstanceID == instanceID && inst.MainIP != "" {
						s.Stop()
						fmt.Println(ui.SuccessStyle.Render("✓ Assigned IPv4: " + inst.MainIP))
						ipPrinted = true
						s.Suffix = " Provisioning server..."
						s.Start()
						break
					}
				}
			}

			// Step 4: Check live power status
			powerStatus, err := client.GetInstancePowerStatus(instanceID)
			if err == nil && powerStatus == "running" {
				inst, err := client.GetInstance(instanceID)
				if err != nil {
					return &api.Instance{
						InstanceID:  instanceID,
						Hostname:    hostname,
						PowerStatus: "running",
					}, nil
				}
				inst.PowerStatus = "running"
				return inst, nil
			}

			if time.Since(startTime) > timeout {
				return nil, fmt.Errorf("provisioning timeout exceeded")
			}
		}
	}
}

// --- Utilities ---

func formatRAM(mb int) string {
	return fmt.Sprintf("%.0f", float64(mb)/1024.0)
}

func formatBW(gb int) string {
	return fmt.Sprintf("%.0f", float64(gb)/1000.0)
}

// --- Billing cycle helpers ---

func selectBillingCycle(plans []api.Plan, flag string, jsonMode bool) (string, error) {
	allCycles := []string{"monthly", "annually", "semiannually", "biennially", "triennially"}
	var availableCycles []string
	for _, cycle := range allCycles {
		for _, plan := range plans {
			if planHasPricing(plan, cycle) {
				availableCycles = append(availableCycles, cycle)
				break
			}
		}
	}
	if len(availableCycles) == 0 {
		return "", fmt.Errorf("no billing cycles available for any plan")
	}

	if flag != "" {
		for _, c := range availableCycles {
			if strings.EqualFold(c, flag) {
				return c, nil
			}
		}
		return "", fmt.Errorf("invalid billing cycle '%s'. Available: %s", flag, strings.Join(availableCycles, ", "))
	}

	if len(availableCycles) == 1 {
		if !jsonMode {
			fmt.Printf("Billing cycle: %s\n", billingCycleLabel(availableCycles[0]))
		}
		return availableCycles[0], nil
	}

	if jsonMode {
		return "monthly", nil
	}

	options := make([]string, len(availableCycles))
	for i, c := range availableCycles {
		options[i] = billingCycleLabel(c)
	}
	var selected string
	err := huh.NewSelect[string]().
		Title("Choose a billing cycle:").
		Options(huh.NewOptions(options...)...).
		Value(&selected).
		Height(10).
		Run()
	if err != nil {
		return "", err
	}
	for _, c := range availableCycles {
		if billingCycleLabel(c) == selected {
			return c, nil
		}
	}
	return availableCycles[0], nil
}

func getPriceForCycle(plan api.Plan, cycle string) string {
	switch cycle {
	case "monthly":
		return plan.PriceMonthly
	case "annually":
		return plan.PriceAnnually
	case "semiannually":
		return plan.PriceSemiannually
	case "biennially":
		return plan.PriceBiennially
	case "triennially":
		return plan.PriceTriennially
	default:
		return plan.PriceMonthly
	}
}

func planHasPricing(plan api.Plan, cycle string) bool {
	price := getPriceForCycle(plan, cycle)
	return price != "" && price != "0.00" && price != "0"
}

func billingCycleSuffix(cycle string) string {
	switch cycle {
	case "monthly":
		return "/mo"
	case "annually":
		return "/yr"
	case "semiannually":
		return "/6mo"
	case "biennially":
		return "/2yr"
	case "triennially":
		return "/3yr"
	default:
		return "/mo"
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

func mapEventMessage(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "cloning"):
		return "Installing OS"
	case strings.Contains(lower, "configuring instance"):
		return "Configuring server"
	case strings.Contains(lower, "cloud init") || strings.Contains(lower, "cloud-init"):
		return "Configuring network"
	case strings.Contains(lower, "password changed") || strings.Contains(lower, "root password"):
		return "Root password set"
	case strings.Contains(lower, "firewall"):
		return "Configuring firewall"
	case strings.Contains(lower, "dns"):
		return "Setting up DNS"
	case strings.Contains(lower, "resize") || strings.Contains(lower, "disk"):
		return "Resizing disk"
	case strings.Contains(lower, "instance created") || strings.Contains(lower, "started"):
		return "Starting server"
	default:
		if msg != "" {
			return msg
		}
		return ""
	}
}
