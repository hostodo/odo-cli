package instances

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/auth"
	"github.com/hostodo/odo-cli/v2/pkg/config"
	"github.com/hostodo/odo-cli/v2/pkg/resolver"
	"github.com/hostodo/odo-cli/v2/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	reinstallOSFlag     string
	reinstallSSHKeyFlag string
	reinstallYesFlag    bool
)

// ReinstallCmd reinstalls the OS on an instance
var ReinstallCmd = &cobra.Command{
	Use:               "reinstall <hostname>",
	Short:             "Reinstall the OS on an instance",
	ValidArgsFunction: resolver.CompleteHostname,
	Long: `Wipe and reinstall the operating system on a VPS instance.

All data on the instance will be lost. Access after reinstall depends on the SSH key and your VM console permissions.

Examples:
  odo instances reinstall mybox
  odo instances reinstall mybox --os "Ubuntu 22.04"
  odo instances reinstall mybox --os "Debian 12" --yes`,
	Args: cobra.ExactArgs(1),
	RunE: runReinstall,
}

func init() {
	ReinstallCmd.Flags().StringVar(&reinstallOSFlag, "os", "", "OS template to install (skips prompt)")
	ReinstallCmd.Flags().StringVar(&reinstallSSHKeyFlag, "ssh-key", "", "SSH key name to inject")
	ReinstallCmd.Flags().BoolVarP(&reinstallYesFlag, "yes", "y", false, "Skip confirmation prompt")
}

func runReinstall(cmd *cobra.Command, args []string) error {
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

	// Resolve instance
	result, err := resolver.ResolveInstance(client, args[0])
	if err != nil {
		return err
	}
	instance := result.Instance

	if instance.IsSuspended {
		return fmt.Errorf("instance %s is suspended and cannot be reinstalled", instance.Hostname)
	}

	// Load templates
	templates, err := client.ListTemplates()
	if err != nil {
		return fmt.Errorf("failed to load OS templates: %w", err)
	}

	// Select template
	selectedTemplate, err := selectTemplate(templates, reinstallOSFlag, false)
	if err != nil {
		return err
	}

	// Select SSH key
	sshKeyName, err := selectSSHKey(client, reinstallSSHKeyFlag, false)
	if err != nil {
		return err
	}
	var sshKeyID int
	if sshKeyName != "" {
		keys, err := client.ListSSHKeys()
		if err != nil {
			return fmt.Errorf("failed to resolve SSH key %q: %w", sshKeyName, err)
		}
		for _, key := range keys {
			if key.Name == sshKeyName {
				sshKeyID = key.ID
				break
			}
		}
		if sshKeyID <= 0 {
			return fmt.Errorf("SSH key %q could not be found", sshKeyName)
		}
	}

	// Confirmation
	if !reinstallYesFlag {
		access := "Access will require permission to view the VM console."
		if sshKeyID > 0 {
			access = fmt.Sprintf("The SSH key %q will be used.", sshKeyName)
		}
		warning := fmt.Sprintf(
			"Reinstall %s (%s) with %s?\n  All data will be erased. %s",
			instance.Hostname, instance.MainIP, selectedTemplate.Name, access,
		)
		confirmed := false
		err := huh.NewConfirm().
			Title(warning).
			Value(&confirmed).
			Run()
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Reinstall cancelled.")
			return nil
		}
	}

	// Execute reinstall
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Reinstalling %s with %s (this takes a few minutes)...", instance.Hostname, selectedTemplate.Name)
	s.Start()

	reinstallResp, err := client.ReinstallInstance(instance.InstanceID, selectedTemplate.ID, sshKeyID)

	s.Stop()
	if err != nil {
		fmt.Println(ui.ErrorStyle.Render("✗ Reinstall failed: " + err.Error()))
		return err
	}
	fmt.Println(ui.SuccessStyle.Render("✓ Reinstall complete"))

	displayReinstallResult(reinstallResp, instance, sshKeyName)
	return nil
}

func displayReinstallResult(r *api.ReinstallResponse, instance *api.Instance, sshKeyName string) {
	sshUser := r.DefaultUser
	if sshUser == "" {
		sshUser = r.Instance.Template.DefaultUsername
	}
	if sshUser == "" {
		sshUser = "root"
	}

	access := "Access:          Open the VM console in Hostodo."
	sshCommand := fmt.Sprintf("ssh %s@%s", sshUser, instance.MainIP)
	switch {
	case r.RootPW != "":
		access = "Root Password:  " + r.RootPW
	case sshKeyName != "":
		access = fmt.Sprintf("SSH Key:         %q", sshKeyName)
		sshCommand = fmt.Sprintf("ssh -i <private-key-path> %s@%s", sshUser, instance.MainIP)
	}

	cardContent := fmt.Sprintf(`OS Reinstalled Successfully!

Hostname:       %s
IP Address:     %s
OS:             %s
%s

SSH:            %s`,
		instance.Hostname,
		instance.MainIP,
		r.Instance.Template.Name,
		access,
		sshCommand,
	)

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2)

	fmt.Println("\n" + cardStyle.Render(cardContent) + "\n")
}
