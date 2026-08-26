package pools

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/deploy"
	"github.com/hostodo/odo-cli/v2/pkg/ui"
	"github.com/hostodo/odo-cli/v2/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	vmPoolFlag      string
	vmOSFlag        string
	vmRegionFlag    string
	vmHostnameFlag  string
	vmSSHKeyFlag    string
	vmPlanFlag      string
	vmVCPUFlag      int
	vmRAMFlag       int
	vmDiskFlag      int
	vmBandwidthFlag int
	vmYesFlag       bool
	vmJSONFlag      bool
)

var vmCmd = &cobra.Command{
	Use:     "vm",
	Aliases: []string{"deploy", "create-vm", "new-vm"},
	Short:   "Create a VM inside a Hostodo pool",
	Long: `Create a gen2 VM inside a Hostodo pool for $0 (quota check only).

Size with --vcpu/--ram/--disk/--bandwidth, or pass a catalog --plan as a shape.
Never accepts payment fields.

Examples:
  odo pools vm
  odo pools vm --os "Ubuntu 22.04" --region DET01 --vcpu 1 --ram 1024 --disk 20 --yes
  odo pools vm --pool pool::abc --os Debian --region TPA01 --plan EPYC-2G1C32GN --json`,
	RunE: runCreatePoolVM,
}

func init() {
	vmCmd.Flags().StringVar(&vmPoolFlag, "pool", "", "Pool ID (defaults to the active pool)")
	vmCmd.Flags().StringVar(&vmOSFlag, "os", "", "OS template name")
	vmCmd.Flags().StringVar(&vmRegionFlag, "region", "", "Region name")
	vmCmd.Flags().StringVar(&vmHostnameFlag, "hostname", "", "Custom hostname")
	vmCmd.Flags().StringVar(&vmSSHKeyFlag, "ssh-key", "", "SSH key name")
	vmCmd.Flags().StringVar(&vmPlanFlag, "plan", "", "Catalog instance plan name/ID used as a shape")
	vmCmd.Flags().IntVar(&vmVCPUFlag, "vcpu", 0, "vCPU count")
	vmCmd.Flags().IntVar(&vmRAMFlag, "ram", 0, "RAM in MB")
	vmCmd.Flags().IntVar(&vmDiskFlag, "disk", 0, "Disk in GB")
	vmCmd.Flags().IntVar(&vmBandwidthFlag, "bandwidth", 0, "Bandwidth in GB")
	vmCmd.Flags().BoolVarP(&vmYesFlag, "yes", "y", false, "Skip confirmation")
	vmCmd.Flags().BoolVar(&vmJSONFlag, "json", false, "JSON output (requires --os, --region, and size or --plan)")
}

func runCreatePoolVM(cmd *cobra.Command, args []string) error {
	if vmJSONFlag && (vmOSFlag == "" || vmRegionFlag == "") {
		return fmt.Errorf("JSON mode requires --os and --region")
	}
	if vmJSONFlag && vmPlanFlag == "" && (vmVCPUFlag == 0 || vmRAMFlag == 0 || vmDiskFlag == 0) {
		return fmt.Errorf("JSON mode requires --vcpu, --ram, and --disk (or --plan)")
	}

	client, err := newAuthenticatedClient()
	if err != nil {
		return err
	}

	pool, err := selectPool(client, vmPoolFlag, vmJSONFlag)
	if err != nil {
		return err
	}
	if pool.Status != "active" {
		return fmt.Errorf("pool %s is %s; only an active pool can create VMs", pool.PoolID, pool.Status)
	}

	templates, err := client.ListTemplates()
	if err != nil {
		return fmt.Errorf("failed to load OS templates: %w", err)
	}
	regions, err := client.ListRegions()
	if err != nil {
		return fmt.Errorf("failed to load regions: %w", err)
	}

	selectedTemplate, err := selectNamedTemplate(templates, vmOSFlag, vmJSONFlag)
	if err != nil {
		return err
	}
	selectedRegion, err := selectNamedRegion(regions, vmRegionFlag, vmJSONFlag)
	if err != nil {
		return err
	}

	req := api.CreatePoolVMRequest{
		PoolID:     pool.PoolID,
		TemplateID: selectedTemplate.ID,
		RegionID:   selectedRegion.ID,
	}

	if vmPlanFlag != "" {
		plans, err := client.ListPlans()
		if err != nil {
			return fmt.Errorf("failed to load plans: %w", err)
		}
		plan, err := findInstancePlan(plans, vmPlanFlag)
		if err != nil {
			return err
		}
		req.PlanID = plan.ID
	} else {
		size, err := resolveVMSize(pool, vmJSONFlag)
		if err != nil {
			return err
		}
		req.VCPU = size.vcpu
		req.RAMMB = size.ramMB
		req.DiskGB = size.diskGB
		req.BandwidthGB = size.bandwidthGB
	}

	hostname, err := resolvePoolHostname(client, vmHostnameFlag)
	if err != nil {
		return err
	}
	req.Hostname = hostname

	sshKeyID, err := selectSSHKeyID(client, vmSSHKeyFlag, vmJSONFlag)
	if err != nil {
		return err
	}
	req.SSHKeyID = sshKeyID

	if !vmYesFlag && !vmJSONFlag {
		confirmed, err := confirmPoolVM(pool, selectedTemplate, selectedRegion, req)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	result, err := client.CreatePoolVM(req)
	if err != nil {
		return fmt.Errorf("failed to create pool VM: %w", err)
	}

	if vmJSONFlag {
		return printJSON(result)
	}

	displayPoolVMResult(result, selectedRegion)
	return nil
}

type vmSize struct {
	vcpu        int
	ramMB       int
	diskGB      int
	bandwidthGB int
}

func selectPool(client *api.Client, flag string, jsonMode bool) (*api.ResourcePoolDetail, error) {
	if flag != "" {
		return resolvePool(client, flag)
	}

	pools, err := client.ListResourcePools()
	if err != nil {
		return nil, fmt.Errorf("failed to list pools: %w", err)
	}
	var active []api.ResourcePool
	for _, p := range pools {
		if p.Status == "active" {
			active = append(active, p)
		}
	}
	if len(active) == 0 {
		return nil, fmt.Errorf("no active Hostodo pool. Buy one with: odo pools buy")
	}
	if len(active) == 1 {
		return client.GetResourcePool(active[0].PoolID)
	}
	if jsonMode {
		return nil, fmt.Errorf("multiple pools found. Use --pool to specify which one")
	}

	options := make([]string, len(active))
	for i, p := range active {
		options[i] = fmt.Sprintf("%s (%s)  %s RAM remaining", p.PoolID, p.Label(), formatRAMGB(p.Remaining.RAMMB))
	}
	var selected string
	err = huh.NewSelect[string]().
		Title("Choose a pool:").
		Options(huh.NewOptions(options...)...).
		Value(&selected).
		Height(10).
		Run()
	if err != nil {
		return nil, err
	}
	poolID := strings.Fields(selected)[0]
	return client.GetResourcePool(poolID)
}

func resolveVMSize(pool *api.ResourcePoolDetail, jsonMode bool) (*vmSize, error) {
	if vmVCPUFlag != 0 || vmRAMFlag != 0 || vmDiskFlag != 0 || vmBandwidthFlag != 0 {
		if vmVCPUFlag == 0 || vmRAMFlag == 0 || vmDiskFlag == 0 {
			return nil, fmt.Errorf("--vcpu, --ram, and --disk are required together")
		}
		bw := vmBandwidthFlag
		if bw == 0 {
			bw = 1024
		}
		return &vmSize{vcpu: vmVCPUFlag, ramMB: vmRAMFlag, diskGB: vmDiskFlag, bandwidthGB: bw}, nil
	}
	if jsonMode {
		return nil, fmt.Errorf("JSON mode requires --vcpu, --ram, and --disk (or --plan)")
	}

	maxVCPU := max(1, min(pool.Quota.MaxVCPUPerInstance, pool.Remaining.VCPU))
	if pool.Quota.MaxVCPUPerInstance == 0 {
		maxVCPU = max(1, pool.Remaining.VCPU)
	}
	maxRAM := max(512, pool.Remaining.RAMMB)
	maxDisk := max(10, pool.Remaining.DiskGB)
	maxBW := max(1, pool.Remaining.BandwidthGB)

	vcpu := min(1, maxVCPU)
	ramMB := min(1024, maxRAM)
	diskGB := min(20, maxDisk)
	bandwidthGB := min(1024, maxBW)

	vcpuStr := strconv.Itoa(vcpu)
	ramGBStr := strconv.Itoa(max(1, ramMB/1024))
	diskStr := strconv.Itoa(diskGB)
	bwStr := strconv.Itoa(bandwidthGB)

	err := huh.NewInput().
		Title(fmt.Sprintf("vCPU (remaining %d, max %d per VM)", pool.Remaining.VCPU, maxVCPU)).
		Value(&vcpuStr).
		Validate(func(s string) error {
			n, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || n < 1 {
				return fmt.Errorf("must be an integer >= 1")
			}
			return nil
		}).
		Run()
	if err != nil {
		return nil, err
	}
	err = huh.NewInput().
		Title(fmt.Sprintf("RAM in GB (remaining %s)", formatRAMGB(pool.Remaining.RAMMB))).
		Value(&ramGBStr).
		Validate(func(s string) error {
			n, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || n < 1 {
				return fmt.Errorf("must be an integer >= 1")
			}
			return nil
		}).
		Run()
	if err != nil {
		return nil, err
	}
	err = huh.NewInput().
		Title(fmt.Sprintf("Disk in GB (remaining %d GB)", pool.Remaining.DiskGB)).
		Value(&diskStr).
		Validate(func(s string) error {
			n, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || n < 10 {
				return fmt.Errorf("must be an integer >= 10")
			}
			return nil
		}).
		Run()
	if err != nil {
		return nil, err
	}
	err = huh.NewInput().
		Title(fmt.Sprintf("Bandwidth in GB (remaining %d GB)", pool.Remaining.BandwidthGB)).
		Value(&bwStr).
		Validate(func(s string) error {
			n, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || n < 1 {
				return fmt.Errorf("must be an integer >= 1")
			}
			return nil
		}).
		Run()
	if err != nil {
		return nil, err
	}

	vcpu, _ = strconv.Atoi(strings.TrimSpace(vcpuStr))
	ramGB, _ := strconv.Atoi(strings.TrimSpace(ramGBStr))
	diskGB, _ = strconv.Atoi(strings.TrimSpace(diskStr))
	bandwidthGB, _ = strconv.Atoi(strings.TrimSpace(bwStr))
	return &vmSize{vcpu: vcpu, ramMB: ramGB * 1024, diskGB: diskGB, bandwidthGB: bandwidthGB}, nil
}

func resolvePoolHostname(client *api.Client, flag string) (string, error) {
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

func selectSSHKeyID(client *api.Client, flag string, jsonMode bool) (int, error) {
	keys, err := client.ListSSHKeys()
	if err != nil || len(keys) == 0 {
		return 0, nil
	}
	if flag != "" {
		for _, key := range keys {
			if strings.EqualFold(key.Name, flag) {
				return key.ID, nil
			}
		}
		return 0, fmt.Errorf("SSH key %q not found", flag)
	}
	if len(keys) == 1 {
		if !jsonMode {
			fmt.Printf("Using SSH key: %s\n", keys[0].Name)
		}
		return keys[0].ID, nil
	}
	if jsonMode {
		return 0, fmt.Errorf("multiple SSH keys found. Use --ssh-key to specify which one")
	}
	options := make([]string, len(keys)+1)
	options[0] = "None"
	for i, key := range keys {
		fingerprint, ferr := utils.CalculateSSHFingerprint(key.PublicKey)
		if ferr != nil {
			fingerprint = "(error)"
		}
		options[i+1] = fmt.Sprintf("%s (%s)", key.Name, fingerprint)
	}
	var selected string
	err = huh.NewSelect[string]().
		Title("Choose an SSH key:").
		Options(huh.NewOptions(options...)...).
		Value(&selected).
		Height(10).
		Run()
	if err != nil {
		return 0, err
	}
	if selected == "None" {
		return 0, nil
	}
	name := strings.Split(selected, " (")[0]
	for _, key := range keys {
		if key.Name == name {
			return key.ID, nil
		}
	}
	return 0, nil
}

func selectNamedTemplate(templates []api.Template, flag string, jsonMode bool) (*api.Template, error) {
	if flag != "" {
		tmpl, err := findNamed(templates, flag, func(t api.Template) string { return t.Name }, "OS template")
		if err != nil {
			return nil, err
		}
		return tmpl, nil
	}
	if jsonMode {
		return nil, fmt.Errorf("JSON mode requires --os")
	}
	names := make([]string, len(templates))
	for i, t := range templates {
		names[i] = t.Name
	}
	var selected string
	err := huh.NewSelect[string]().
		Title("Choose an OS:").
		Options(huh.NewOptions(names...)...).
		Value(&selected).
		Height(15).
		Run()
	if err != nil {
		return nil, err
	}
	tmpl, _ := findNamed(templates, selected, func(t api.Template) string { return t.Name }, "OS template")
	return tmpl, nil
}

func selectNamedRegion(regions []api.Region, flag string, jsonMode bool) (*api.Region, error) {
	if flag != "" {
		region, err := findNamed(regions, flag, func(r api.Region) string { return r.Name }, "region")
		if err != nil {
			return nil, err
		}
		return region, nil
	}
	if jsonMode {
		return nil, fmt.Errorf("JSON mode requires --region")
	}
	names := make([]string, len(regions))
	for i, r := range regions {
		names[i] = r.Name
	}
	var selected string
	err := huh.NewSelect[string]().
		Title("Choose a region:").
		Options(huh.NewOptions(names...)...).
		Value(&selected).
		Height(15).
		Run()
	if err != nil {
		return nil, err
	}
	region, _ := findNamed(regions, selected, func(r api.Region) string { return r.Name }, "region")
	return region, nil
}

func findNamed[T any](items []T, name string, label func(T) string, kind string) (*T, error) {
	for i := range items {
		if strings.EqualFold(label(items[i]), name) {
			return &items[i], nil
		}
	}
	lower := strings.ToLower(name)
	var matches []*T
	for i := range items {
		if strings.Contains(strings.ToLower(label(items[i])), lower) {
			matches = append(matches, &items[i])
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = label(*m)
		}
		return nil, fmt.Errorf("ambiguous %s %q — matches: %s", kind, name, strings.Join(names, ", "))
	}
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = label(item)
	}
	return nil, fmt.Errorf("no %s matching %q. Available: %s", kind, name, strings.Join(names, ", "))
}

func findInstancePlan(plans []api.Plan, name string) (*api.Plan, error) {
	if id, err := strconv.Atoi(name); err == nil {
		for i := range plans {
			if plans[i].ID == id {
				return &plans[i], nil
			}
		}
	}
	plan, err := findNamed(plans, name, func(p api.Plan) string { return p.Name }, "plan")
	if err != nil {
		return nil, err
	}
	return plan, nil
}

func confirmPoolVM(pool *api.ResourcePoolDetail, tmpl *api.Template, region *api.Region, req api.CreatePoolVMRequest) (bool, error) {
	sizeLine := ""
	if req.PlanID != 0 {
		sizeLine = fmt.Sprintf("  Plan:      #%d\n", req.PlanID)
	} else {
		sizeLine = fmt.Sprintf("  Size:      %d vCPU, %s RAM, %d GB disk, %d GB BW\n", req.VCPU, formatRAMGB(req.RAMMB), req.DiskGB, req.BandwidthGB)
	}
	fmt.Printf(`
Create pool VM:
  Pool:      %s (%s)
  OS:        %s
  Region:    %s
  Hostname:  %s
%s  Charge:    $0 (uses pool quota)

`, pool.PoolID, pool.Label(), tmpl.Name, region.Name, req.Hostname, sizeLine)

	confirmed := true
	err := huh.NewConfirm().
		Title("Create this VM in the pool?").
		Value(&confirmed).
		Run()
	if err != nil {
		return false, err
	}
	return confirmed, nil
}

func displayPoolVMResult(result *api.CreatePoolVMResponse, region *api.Region) {
	inst := result.Instance
	content := fmt.Sprintf(`Pool VM created

Hostname:   %s
IP Address: %s
Status:     %s
Region:     %s
Pool:       %s

Quota remaining: %s RAM, %d vCPU, %d GB disk, %d VMs`,
		inst.Hostname,
		inst.MainIP,
		inst.Status,
		region.Name,
		inst.PoolID,
		formatRAMGB(result.Quota.Remaining.RAMMB),
		result.Quota.Remaining.VCPU,
		result.Quota.Remaining.DiskGB,
		result.Quota.Remaining.Instances,
	)
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2)
	fmt.Println("\n" + ui.SuccessStyle.Render("✓ VM created") + "\n" + card.Render(content) + "\n")
	if inst.Hostname != "" {
		fmt.Printf("SSH: odo ssh %s\n", inst.Hostname)
	}
}
