package agent

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/spf13/cobra"
)

var (
	revokeForce bool
	revokeAll   bool
)

var revokeCmd = &cobra.Command{
	Use:   "revoke [instance-id]",
	Short: "Revoke agent token(s) to disable agent access",
	Long: `Revoke agent tokens to disable AI agent access for your instances.

When a token is revoked, the agent on that instance will no longer be able to
use the Claude API proxy until the token is regenerated.

Examples:
  hostodo agent revoke <instance-id>         # Revoke with confirmation prompt
  hostodo agent revoke <instance-id> --force # Revoke immediately
  hostodo agent revoke --all --force         # Revoke ALL tokens (requires --force)`,
	Args: func(cmd *cobra.Command, args []string) error {
		if revokeAll {
			if len(args) > 0 {
				return fmt.Errorf("cannot specify instance-id with --all flag")
			}
			if !revokeForce {
				return fmt.Errorf("--all requires --force flag for safety")
			}
			return nil
		}
		if len(args) != 1 {
			return fmt.Errorf("requires instance-id argument (or use --all)")
		}
		return nil
	},
	RunE: runRevoke,
}

func init() {
	revokeCmd.Flags().BoolVarP(&revokeForce, "force", "f", false, "Skip confirmation prompt")
	revokeCmd.Flags().BoolVar(&revokeAll, "all", false, "Revoke tokens for ALL instances (requires --force)")
	AgentCmd.AddCommand(revokeCmd)
}

func runRevoke(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	client, err := api.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	if revokeAll {
		return revokeAllTokens(client)
	}

	instanceID := args[0]
	return revokeSingleToken(client, instanceID)
}

func revokeSingleToken(client *api.Client, instanceID string) error {
	// Get instance details for confirmation message
	instance, err := client.GetInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Confirmation prompt unless --force
	if !revokeForce {
		fmt.Printf("\n⚠️  This will revoke the agent token for instance '%s' (%s).\n", instance.Hostname, instanceID)
		fmt.Println("The agent will no longer be able to use Claude API until regenerated.")
		fmt.Print("\nAre you sure? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Call revoke API
	err = client.RevokeAgentToken(instanceID)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	fmt.Printf("✓ Token revoked for %s\n", instanceID)
	return nil
}

func revokeAllTokens(client *api.Client) error {
	// --force is already required by Args validation

	result, err := client.RevokeAllAgentTokens()
	if err != nil {
		return fmt.Errorf("failed to revoke tokens: %w", err)
	}

	fmt.Printf("✓ Revoked tokens for %d instances\n", result.Count)
	return nil
}
