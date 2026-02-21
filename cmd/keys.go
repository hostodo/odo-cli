package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/hostodo/hostodo-cli/pkg/ui"
	"github.com/hostodo/hostodo-cli/pkg/utils"
	"github.com/spf13/cobra"
)

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage SSH keys",
	Long: `Manage SSH keys for VPS deployments.

SSH keys allow you to authenticate to your VPS instances without passwords.
Keys can be associated with new instances during deployment.

Examples:
  hostodo keys list
  hostodo keys add mykey "ssh-rsa AAAAB3..."
  hostodo keys remove mykey`,
}

var keysListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all SSH keys",
	Aliases: []string{"ls"},
	RunE:    runKeysList,
}

var keysAddCmd = &cobra.Command{
	Use:   "add [name] [public-key]",
	Short: "Add a new SSH key",
	Long: `Add a new SSH public key.

The public key can be provided as an argument or read from a file using --file flag.

Examples:
  # Add key inline
  hostodo keys add mykey "ssh-rsa AAAAB3NzaC1yc2EAAA... user@host"

  # Add key from file
  hostodo keys add mykey --file ~/.ssh/id_rsa.pub`,
	Args: cobra.MinimumNArgs(1),
	RunE: runKeysAdd,
}

var keysRemoveCmd = &cobra.Command{
	Use:     "remove [name]",
	Short:   "Remove an SSH key",
	Aliases: []string{"rm", "delete"},
	Args:    cobra.ExactArgs(1),
	RunE:    runKeysRemove,
}

var keyFileFlag string

func init() {
	keysCmd.AddCommand(keysListCmd, keysAddCmd, keysRemoveCmd)
	keysAddCmd.Flags().StringVar(&keyFileFlag, "file", "", "Read public key from file")
}

func runKeysList(cmd *cobra.Command, args []string) error {
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

	// Fetch SSH keys
	keys, err := client.ListSSHKeys()
	if err != nil {
		return fmt.Errorf("failed to list SSH keys: %w", err)
	}

	if len(keys) == 0 {
		fmt.Println("No SSH keys found.")
		fmt.Println("Add a key with: hostodo keys add <name> <public-key>")
		return nil
	}

	// Calculate fingerprints for each key
	displayKeys := make([]ui.SSHKeyDisplay, len(keys))
	for i, key := range keys {
		fingerprint, err := utils.CalculateSSHFingerprint(key.PublicKey)
		if err != nil {
			fingerprint = "(error calculating)"
		}

		// Format created_at as date only
		createdAt := key.CreatedAt
		if t, err := time.Parse(time.RFC3339, key.CreatedAt); err == nil {
			createdAt = t.Format("2006-01-02")
		}

		displayKeys[i] = ui.SSHKeyDisplay{
			Name:        key.Name,
			Fingerprint: fingerprint,
			CreatedAt:   createdAt,
		}
	}

	// Format table
	output := ui.FormatSSHKeysTable(displayKeys)
	fmt.Println(output)

	return nil
}

func runKeysAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	var publicKey string

	// Get public key from file or arguments
	if keyFileFlag != "" {
		// Read from file
		data, err := os.ReadFile(keyFileFlag)
		if err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}
		publicKey = strings.TrimSpace(string(data))
	} else {
		// Join remaining args as public key
		if len(args) < 2 {
			return fmt.Errorf("public key required. Provide inline or use --file flag")
		}
		publicKey = strings.Join(args[1:], " ")
	}

	// Validate key format
	if !strings.Contains(publicKey, " ") {
		return fmt.Errorf("invalid SSH public key format")
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

	// Add SSH key
	key, err := client.AddSSHKey(name, publicKey)
	if err != nil {
		return fmt.Errorf("failed to add SSH key: %w", err)
	}

	// Calculate fingerprint
	fingerprint, err := utils.CalculateSSHFingerprint(key.PublicKey)
	if err != nil {
		fingerprint = "(error calculating)"
	}

	// Show confirmation
	fmt.Println(ui.SuccessStyle.Render("✓ SSH key added: " + name))
	fmt.Println("Fingerprint: " + fingerprint)

	return nil
}

func runKeysRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

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

	// Fetch all keys to find by name
	keys, err := client.ListSSHKeys()
	if err != nil {
		return fmt.Errorf("failed to list SSH keys: %w", err)
	}

	// Find matching keys
	var matchingKeys []api.SSHKey
	for _, key := range keys {
		if key.Name == name {
			matchingKeys = append(matchingKeys, key)
		}
	}

	if len(matchingKeys) == 0 {
		return fmt.Errorf("no SSH key found with name: %s", name)
	}

	// If multiple keys with same name, ask which one
	var keyToDelete *api.SSHKey
	if len(matchingKeys) > 1 {
		options := make([]string, len(matchingKeys))
		for i, key := range matchingKeys {
			fingerprint, _ := utils.CalculateSSHFingerprint(key.PublicKey)
			options[i] = fmt.Sprintf("%d. %s (%s)", i+1, key.Name, fingerprint)
		}

		var selectedOption string
		prompt := &survey.Select{
			Message: "Multiple keys found. Select which to delete:",
			Options: options,
		}
		if err := survey.AskOne(prompt, &selectedOption); err != nil {
			return err
		}

		// Parse selection
		var selectedIndex int
		fmt.Sscanf(selectedOption, "%d.", &selectedIndex)
		keyToDelete = &matchingKeys[selectedIndex-1]
	} else {
		keyToDelete = &matchingKeys[0]
	}

	// Calculate fingerprint for confirmation
	fingerprint, err := utils.CalculateSSHFingerprint(keyToDelete.PublicKey)
	if err != nil {
		fingerprint = "(error calculating)"
	}

	// Confirmation prompt
	fmt.Printf("Remove SSH key: %s\n", keyToDelete.Name)
	fmt.Printf("Fingerprint: %s\n", fingerprint)

	var confirmed bool
	prompt := &survey.Confirm{
		Message: "Are you sure?",
		Default: false,
	}
	if err := survey.AskOne(prompt, &confirmed); err != nil {
		return err
	}

	if !confirmed {
		fmt.Println("Cancelled.")
		return nil
	}

	// Delete key
	if err := client.DeleteSSHKey(keyToDelete.ID); err != nil {
		return fmt.Errorf("failed to delete SSH key: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render("✓ SSH key removed successfully"))

	return nil
}
