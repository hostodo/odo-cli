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
	// JSON mode validation
	if jsonFlag {
		if osFlag == "" || regionFlag == "" || planFlag == "" {
			return fmt.Errorf("JSON mode requires --os, --region, and --plan flags")
		}
	}

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

	// Select OS template
	var selectedTemplate *api.Template
	if osFlag != "" {
		selectedTemplate = findTemplate(templates, osFlag)
		if selectedTemplate == nil {
			templateNames := make([]string, len(templates))
			for i, t := range templates {
				templateNames[i] = t.Name
			}
			return fmt.Errorf("no OS template matching '%s'. Available: %s", osFlag, strings.Join(templateNames, ", "))
		}
	} else {
		if jsonFlag {
			return fmt.Errorf("JSON mode requires --os flag")
		}
		templateNames := make([]string, len(templates))
		for i, t := range templates {
			templateNames[i] = t.Name
		}
		var selectedOS string
		prompt := &survey.Select{
			Message:  "Choose an operating system:",
			Options:  templateNames,
			PageSize: 15,
		}
		if err := survey.AskOne(prompt, &selectedOS); err != nil {
			return err
		}
		selectedTemplate = findTemplate(templates, selectedOS)
	}

	// Select region
	var selectedRegion *api.Region
	if regionFlag != "" {
		selectedRegion = findRegion(regions, regionFlag)
		if selectedRegion == nil {
			regionNames := make([]string, len(regions))
			for i, r := range regions {
				regionNames[i] = r.Name
			}
			return fmt.Errorf("no region matching '%s'. Available: %s", regionFlag, strings.Join(regionNames, ", "))
		}
	} else {
		if jsonFlag {
			return fmt.Errorf("JSON mode requires --region flag")
		}
		regionNames := make([]string, len(regions))
		for i, r := range regions {
			regionNames[i] = r.Name
		}
		var selectedRegionName string
		prompt := &survey.Select{
			Message:  "Choose a region:",
			Options:  regionNames,
			PageSize: 15,
		}
		if err := survey.AskOne(prompt, &selectedRegionName); err != nil {
			return err
		}
		selectedRegion = findRegion(regions, selectedRegionName)
	}

	// Select plan
	var selectedPlan *api.Plan
	if planFlag != "" {
		selectedPlan = findPlan(plans, planFlag)
		if selectedPlan == nil {
			planNames := make([]string, len(plans))
			for i, p := range plans {
				planNames[i] = p.Name
			}
			return fmt.Errorf("no plan matching '%s'. Available: %s", planFlag, strings.Join(planNames, ", "))
		}
	} else {
		if jsonFlag {
			return fmt.Errorf("JSON mode requires --plan flag")
		}
		planOptions := make([]string, len(plans))
		for i, p := range plans {
			planOptions[i] = fmt.Sprintf("%-12s $%s/mo   %d vCPU, %sGB RAM, %dGB SSD, %sTB BW",
				p.Name, p.PriceMonthly, p.VCPU, formatRAM(p.RAM), p.Disk, formatBW(p.Bandwidth))
		}
		var selectedPlanOption string
		prompt := &survey.Select{
			Message:  "Choose a plan:",
			Options:  planOptions,
			PageSize: 15,
		}
		if err := survey.AskOne(prompt, &selectedPlanOption); err != nil {
			return err
		}
		// Extract plan name from formatted option (first field)
		planName := strings.Fields(selectedPlanOption)[0]
		selectedPlan = findPlan(plans, planName)
	}

	// Determine hostname
	var hostname string
	if hostnameFlag != "" {
		if err := deploy.Validate(hostnameFlag); err != nil {
			return fmt.Errorf("invalid hostname: %w", err)
		}
		hostname = hostnameFlag
	} else {
		hostname, err = deploy.Generate(client.CheckHostnameExists)
		if err != nil {
			return fmt.Errorf("failed to generate hostname: %w", err)
		}
	}

	// SSH key selection
	var selectedSSHKeyName string
	if sshKeyFlag != "" {
		// Use flag value
		selectedSSHKeyName = sshKeyFlag
	} else {
		// Fetch SSH keys (non-fatal - skip on error)
		sshKeys, err := client.ListSSHKeys()
		if err == nil && len(sshKeys) > 0 {
			if len(sshKeys) == 1 {
				// Auto-select single key
				selectedSSHKeyName = sshKeys[0].Name
				if !jsonFlag {
					fmt.Printf("Using SSH key: %s\n", sshKeys[0].Name)
				}
			} else {
				// Multiple keys - show picker (not in JSON mode)
				if jsonFlag {
					return fmt.Errorf("multiple SSH keys found. Use --ssh-key flag to specify which one")
				}
				keyOptions := make([]string, len(sshKeys))
				for i, key := range sshKeys {
					fingerprint, err := utils.CalculateSSHFingerprint(key.PublicKey)
					if err != nil {
						fingerprint = "(error)"
					}
					keyOptions[i] = fmt.Sprintf("%s (%s)", key.Name, fingerprint)
				}
				var selectedOption string
				prompt := &survey.Select{
					Message:  "Choose an SSH key:",
					Options:  keyOptions,
					PageSize: 10,
				}
				if err := survey.AskOne(prompt, &selectedOption); err != nil {
					return err
				}
				// Extract key name from option
				selectedSSHKeyName = strings.Split(selectedOption, " (")[0]
			}
		}
		// If no keys or error fetching, skip silently (deploy with password auth)
	}

	// Get quote
	quote, err := client.GetQuote(api.QuoteRequest{
		Plan:         selectedPlan.Name,
		BillingCycle: "monthly",
		Quantity:     1,
	})
	if err != nil {
		return fmt.Errorf("failed to get price quote: %w", err)
	}

	// Get payment method
	paymentMethod, err := client.GetDefaultPaymentMethod()
	if err != nil {
		return fmt.Errorf("failed to get payment method: %w", err)
	}
	if paymentMethod == nil {
		errorMsg := ui.ErrorStyle.Render("No payment method on file. Add one at https://panel.hostodo.com/billing")
		fmt.Println(errorMsg)
		return fmt.Errorf("no payment method configured")
	}

	// Confirmation prompt (unless --yes or --json)
	if !yesFlag && !jsonFlag {
		summary := fmt.Sprintf(`Deploy Summary:
  OS:       %s
  Region:   %s
  Plan:     %s (%d vCPU, %sGB RAM, %dGB SSD)
  Hostname: %s
  Price:    $%s/mo
  Payment:  %s ****%s`,
			selectedTemplate.Name,
			selectedRegion.Name,
			selectedPlan.Name,
			selectedPlan.VCPU,
			formatRAM(selectedPlan.RAM),
			selectedPlan.Disk,
			hostname,
			quote.AmountDue,
			paymentMethod.CardType,
			paymentMethod.LastFour)

		fmt.Println("\n" + summary + "\n")

		confirmMsg := fmt.Sprintf("Deploy? (charges $%s to %s ****%s)",
			quote.AmountDue, paymentMethod.CardType, paymentMethod.LastFour)
		var confirmed bool
		prompt := &survey.Confirm{
			Message: confirmMsg,
			Default: true,
		}
		if err := survey.AskOne(prompt, &confirmed); err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Deployment cancelled.")
			return nil
		}
	}

	// Provisioning with progress display
	var deployResp *api.DeployResponse
	var instance *api.Instance

	if !jsonFlag {
		// Stage 1 - Creating order
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = " Creating order..."
		s.Start()

		deployResp, err = client.CreateDeployOrder(api.DeployRequest{
			Hostname:     hostname,
			Region:       selectedRegion.Name,
			Template:     selectedTemplate.Name,
			Plan:         selectedPlan.Name,
			BillingCycle: "monthly",
			SSHKey:       selectedSSHKeyName,
			Quantity:     1,
		})

		s.Stop()
		if err != nil {
			fmt.Println(ui.ErrorStyle.Render("✗ Order creation failed: " + err.Error()))
			return err
		}
		fmt.Println(ui.SuccessStyle.Render("✓ Order created"))

		// Stage 2 - Payment processing
		s = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = " Processing payment..."
		s.Start()

		// Check invoice status
		if deployResp.Invoice.Status != "paid" {
			s.Stop()
			errorMsg := fmt.Sprintf("✗ Payment failed: %s", deployResp.Invoice.Status)
			fmt.Println(ui.ErrorStyle.Render(errorMsg))
			return fmt.Errorf("payment failed: %s. No instance was created", deployResp.Invoice.Status)
		}

		s.Stop()
		fmt.Println(ui.SuccessStyle.Render("✓ Payment processed"))

		// Stage 3 - Provisioning server with granular updates
		instance, err = pollForProvisioningWithProgress(client, hostname, 12*time.Minute)

		if err != nil {
			fmt.Println(ui.ErrorStyle.Render("✗ Provisioning timed out"))
			fmt.Println("Instance may still be provisioning. Check the web panel.")
			return err
		}
		fmt.Println(ui.SuccessStyle.Render("✓ Server online"))
	} else {
		// JSON mode - no progress display
		deployResp, err = client.CreateDeployOrder(api.DeployRequest{
			Hostname:     hostname,
			Region:       selectedRegion.Name,
			Template:     selectedTemplate.Name,
			Plan:         selectedPlan.Name,
			BillingCycle: "monthly",
			SSHKey:       selectedSSHKeyName,
			Quantity:     1,
		})
		if err != nil {
			return err
		}

		// Wait for provisioning
		instance, err = pollForProvisioning(client, hostname, 12*time.Minute)
		if err != nil {
			return err
		}
	}

	// JSON output mode
	if jsonFlag {
		output := map[string]string{
			"hostname": instance.Hostname,
			"ip":       instance.MainIP,
			"region":   selectedRegion.Name,
			"plan":     selectedPlan.Name,
			"status":   instance.Status,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Determine SSH user for display
	sshUser := instance.Template.DefaultUsername
	if sshUser == "" {
		sshUser = "root"
	}

	// Determine password display
	passwordDisplay := instance.DefaultPassword
	if passwordDisplay == "" {
		passwordDisplay = "(check email for password)"
	}

	// Boxed result card
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
		selectedRegion.Name,
		selectedPlan.Name,
		sshUser,
		instance.MainIP)

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2)

	fmt.Println("\n" + cardStyle.Render(cardContent) + "\n")

	// SSH prompt
	var connectNow bool
	prompt := &survey.Confirm{
		Message: "Connect now?",
		Default: true,
	}
	if err := survey.AskOne(prompt, &connectNow); err != nil {
		return err
	}

	if connectNow {
		// Execute SSH command (runSSH uses Run not RunE, call directly)
		runSSH(sshCmd, []string{hostname})
	}

	return nil
}

// Helper functions

func formatRAM(mb int) string {
	return fmt.Sprintf("%.0f", float64(mb)/1024.0)
}

func formatBW(gb int) string {
	return fmt.Sprintf("%.0f", float64(gb)/1000.0)
}

func findTemplate(templates []api.Template, name string) *api.Template {
	lowerName := strings.ToLower(name)
	for i := range templates {
		if strings.Contains(strings.ToLower(templates[i].Name), lowerName) {
			return &templates[i]
		}
	}
	return nil
}

func findRegion(regions []api.Region, name string) *api.Region {
	lowerName := strings.ToLower(name)
	for i := range regions {
		if strings.Contains(strings.ToLower(regions[i].Name), lowerName) {
			return &regions[i]
		}
	}
	return nil
}

func findPlan(plans []api.Plan, name string) *api.Plan {
	lowerName := strings.ToLower(name)
	for i := range plans {
		if strings.Contains(strings.ToLower(plans[i].Name), lowerName) {
			return &plans[i]
		}
	}
	return nil
}

func pollForProvisioning(client *api.Client, hostname string, timeout time.Duration) (*api.Instance, error) {
	startTime := time.Now()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Find instance by hostname in the list
			instancesResp, err := client.ListInstances(1000, 0)
			if err != nil {
				continue
			}

			for _, inst := range instancesResp.Results {
				if inst.Hostname == hostname {
					// Found the instance — check live power status
					powerStatus, err := client.GetInstancePowerStatus(inst.InstanceID)
					if err == nil && powerStatus == "running" {
						inst.PowerStatus = powerStatus
						return &inst, nil
					}
					break
				}
			}

			// Check timeout
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
			// Step 1: Find instance by hostname
			if !instanceFound {
				instancesResp, err := client.ListInstances(1000, 0)
				if err != nil {
					continue
				}
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
				// Events come newest-first, reverse to print in order
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

			// Step 3: Check if IP appeared (in case it wasn't there initially)
			if !ipPrinted {
				instancesResp, err := client.ListInstances(1000, 0)
				if err == nil {
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
			}

			// Step 4: Check live power status
			powerStatus, err := client.GetInstancePowerStatus(instanceID)
			if err == nil && powerStatus == "running" {
				// Fetch final instance details
				inst, err := client.GetInstance(instanceID)
				if err != nil {
					// Fallback: return what we can
					return &api.Instance{
						InstanceID:  instanceID,
						Hostname:    hostname,
						PowerStatus: "running",
					}, nil
				}
				inst.PowerStatus = "running"
				return inst, nil
			}

			// Check timeout
			if time.Since(startTime) > timeout {
				return nil, fmt.Errorf("provisioning timeout exceeded")
			}
		}
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
