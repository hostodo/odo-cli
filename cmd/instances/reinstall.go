package instances

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/hostodo/hostodo-cli/pkg/resolver"
	"github.com/hostodo/hostodo-cli/pkg/ui"
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

All data on the instance will be lost. A new root password will be issued.

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
		if err == nil {
			for _, k := range keys {
				if k.Name == sshKeyName {
					sshKeyID = k.ID
					break
				}
			}
		}
	}

	// Confirmation
	if !reinstallYesFlag {
		warning := fmt.Sprintf(
			"Reinstall %s (%s) with %s?\n  All data will be erased and a new root password will be issued.",
			instance.Hostname, instance.MainIP, selectedTemplate.Name,
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

	displayReinstallResult(reinstallResp, instance)
	return nil
}

func displayReinstallResult(r *api.ReinstallResponse, instance *api.Instance) {
	sshUser := r.DefaultUser
	if sshUser == "" {
		sshUser = r.Instance.Template.DefaultUsername
	}
	if sshUser == "" {
		sshUser = "root"
	}

	password := r.RootPW
	if password == "" {
		password = "(check email)"
	}

	cardContent := fmt.Sprintf(`OS Reinstalled Successfully!

Hostname:       %s
IP Address:     %s
OS:             %s
Root Password:  %s

SSH:            ssh %s@%s`,
		instance.Hostname,
		instance.MainIP,
		r.Instance.Template.Name,
		password,
		sshUser,
		instance.MainIP,
	)

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2)

	fmt.Println("\n" + cardStyle.Render(cardContent) + "\n")
}
