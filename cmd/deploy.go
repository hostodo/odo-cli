package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/briandowns/spinner"
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
	osFlag       string
	regionFlag   string
	planFlag     string
	hostnameFlag string
	sshKeyFlag   string
	yesFlag      bool
	jsonFlag     bool
)

var deployCmd = &cobra.Command{
	Use:     "deploy",
	Short:   "Deploy a new VPS instance",
	Aliases: []string{"new", "create"},
	Long: `Deploy a new VPS instance with interactive prompts or flags.

This command guides you through selecting an OS, region, and plan, then
provisions a new VPS instance. You can use flags to skip interactive prompts.

Examples:
  # Interactive mode (guided prompts)
  hostodo deploy

  # Skip prompts with flags
  hostodo deploy --os "Ubuntu 22.04" --region "Los Angeles" --plan KVM-2G

  # Custom hostname
  hostodo deploy --hostname my-server

  # Skip confirmation
  hostodo deploy --yes

  # JSON output (requires all selection flags)
  hostodo deploy --os "Ubuntu 22.04" --region "Los Angeles" --plan KVM-2G --json`,
	RunE: runDeploy,
}

func init() {
	deployCmd.Flags().StringVar(&osFlag, "os", "", "OS template name (skips OS prompt)")
	deployCmd.Flags().StringVar(&regionFlag, "region", "", "Region name (skips region prompt)")
	deployCmd.Flags().StringVar(&planFlag, "plan", "", "Plan name (skips plan prompt)")
	deployCmd.Flags().StringVar(&hostnameFlag, "hostname", "", "Custom hostname (skips auto-generation)")
	deployCmd.Flags().StringVar(&sshKeyFlag, "ssh-key", "", "SSH key name to use for authentication")
	deployCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Skip confirmation prompt")
	deployCmd.Flags().BoolVar(&jsonFlag, "json", false, "JSON output mode (requires --os, --region, --plan)")
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
		return fmt.Errorf("not authenticated. Run 'hostodo login' first")
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
	selectedPlan, err := selectPlan(plans, planFlag, jsonFlag)
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
		Plan:         selectedPlan.Name,
		BillingCycle: "monthly",
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
		confirmed, err := confirmDeploy(selectedTemplate, selectedRegion, selectedPlan, hostname, quote, paymentMethod)
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
		Hostname:     hostname,
		Region:       selectedRegion.Name,
		Template:     selectedTemplate.Name,
		Plan:         selectedPlan.Name,
		BillingCycle: "monthly",
		SSHKey:       sshKeyName,
		Quantity:     1,
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
	prompt := &survey.Select{
		Message:  "Choose an OS:",
		Options:  osOptions,
		PageSize: 15,
	}
	if err := survey.AskOne(prompt, &selected); err != nil {
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
	prompt := &survey.Select{
		Message:  "Choose a region:",
		Options:  regionOptions,
		PageSize: 15,
	}
	if err := survey.AskOne(prompt, &selected); err != nil {
		return nil, err
	}
	region, _ := findRegion(regions, selected)
	return region, nil
}

func selectPlan(plans []api.Plan, flag string, jsonMode bool) (*api.Plan, error) {
	if flag != "" {
		plan := findPlan(plans, flag)
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
	planOptions := make([]string, len(plans))
	for i, p := range plans {
		planOptions[i] = fmt.Sprintf("%-12s $%s/mo   %d vCPU, %sGB RAM, %dGB SSD, %sTB BW",
			p.Name, p.PriceMonthly, p.VCPU, formatRAM(p.RAM), p.Disk, formatBW(p.Bandwidth))
	}
	var selected string
	prompt := &survey.Select{
		Message:  "Choose a plan:",
		Options:  planOptions,
		PageSize: 15,
	}
	if err := survey.AskOne(prompt, &selected); err != nil {
		return nil, err
	}
	planName := strings.Fields(selected)[0]
	return findPlan(plans, planName), nil
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
	prompt := &survey.Select{
		Message:  "Choose an SSH key:",
		Options:  keyOptions,
		PageSize: 10,
	}
	if err := survey.AskOne(prompt, &selected); err != nil {
		return "", err
	}
	return strings.Split(selected, " (")[0], nil
}

// --- Deploy execution ---

func confirmDeploy(tmpl *api.Template, region *api.Region, plan *api.Plan, hostname string, quote *api.QuoteResponse, pm *api.PaymentMethod) (bool, error) {
	summary := fmt.Sprintf(`Deploy Summary:
  OS:       %s
  Region:   %s
  Plan:     %s (%d vCPU, %sGB RAM, %dGB SSD)
  Hostname: %s
  Price:    $%s/mo
  Payment:  %s ****%s`,
		tmpl.Name,
		region.Name,
		plan.Name,
		plan.VCPU,
		formatRAM(plan.RAM),
		plan.Disk,
		hostname,
		quote.AmountDue,
		pm.CardType,
		pm.LastFour)

	fmt.Println("\n" + summary + "\n")

	confirmMsg := fmt.Sprintf("Deploy? (charges $%s to %s ****%s)",
		quote.AmountDue, pm.CardType, pm.LastFour)
	var confirmed bool
	prompt := &survey.Confirm{
		Message: confirmMsg,
		Default: true,
	}
	if err := survey.AskOne(prompt, &confirmed); err != nil {
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

	// Stage 2 - Payment processing
	s = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Processing payment..."
	s.Start()

	if deployResp.Invoice.Status != "paid" {
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
	var connectNow bool
	prompt := &survey.Confirm{
		Message: "Connect now?",
		Default: true,
	}
	if err := survey.AskOne(prompt, &connectNow); err != nil {
		return
	}
	if connectNow {
		runSSH(sshCmd, []string{hostname})
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

func findPlan(plans []api.Plan, name string) *api.Plan {
	for i := range plans {
		if strings.EqualFold(plans[i].Name, name) {
			return &plans[i]
		}
	}
	return nil
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
