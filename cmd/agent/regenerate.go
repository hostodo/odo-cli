package agent

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/spf13/cobra"
)

var (
	regenerateForce  bool
	regenerateDeploy bool
)

var regenerateCmd = &cobra.Command{
	Use:   "regenerate <instance-id>",
	Short: "Regenerate agent token to restore agent access",
	Long: `Regenerate an agent token for a specific instance.

This revokes the existing token (if any) and generates a new one.
Use this to restore agent access after revocation or suspected compromise.

The new token is displayed only once - save it immediately!

Examples:
  hostodo agent regenerate <instance-id>           # Regenerate with confirmation
  hostodo agent regenerate <instance-id> --force   # Regenerate immediately
  hostodo agent regenerate <instance-id> --deploy  # Regenerate and deploy via SSH`,
	Args: cobra.ExactArgs(1),
	RunE: runRegenerate,
}

func init() {
	regenerateCmd.Flags().BoolVarP(&regenerateForce, "force", "f", false, "Skip confirmation prompt")
	regenerateCmd.Flags().BoolVar(&regenerateDeploy, "deploy", false, "Deploy new token to instance via SSH")
	AgentCmd.AddCommand(regenerateCmd)
}

func runRegenerate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	client, err := api.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	instanceID := args[0]

	// Get instance details for confirmation message and IP (for --deploy)
	instance, err := client.GetInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Check if instance has an existing token
	tokens, err := client.GetAgentTokens()
	hasExistingToken := false
	if err == nil {
		for _, t := range tokens.Results {
			if t.InstanceID == instanceID && t.Status == "active" {
				hasExistingToken = true
				break
			}
		}
	}

	// Confirmation prompt unless --force
	if !regenerateForce && hasExistingToken {
		fmt.Printf("\n⚠️  This will invalidate the current agent token for '%s'.\n", instance.Hostname)
		fmt.Println("The agent on the VM will lose API access until the new token is deployed.")
		fmt.Print("\nContinue? (y/N): ")

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

	// Call regenerate API
	result, err := client.RegenerateAgentToken(instanceID)
	if err != nil {
		return fmt.Errorf("failed to regenerate token: %w", err)
	}

	fmt.Printf("\n✓ New token generated for %s\n\n", instanceID)
	fmt.Printf("Token: %s\n\n", result.Token)
	fmt.Println("⚠️  Save this token now - it will NOT be shown again.")

	// Deploy to instance if --deploy flag
	if regenerateDeploy {
		fmt.Printf("\nDeploying token to %s (%s)...\n", instance.Hostname, instance.MainIP)

		err = deployTokenToInstance(instance.MainIP, result.Token)
		if err != nil {
			fmt.Printf("\n⚠️  Deployment failed: %v\n", err)
			fmt.Println("You can manually deploy the token by running:")
			fmt.Printf("  ssh root@%s \"echo '%s' > /etc/hostodo/agent-token\"\n", instance.MainIP, result.Token)
			return nil // Don't fail the command, token was generated successfully
		}

		fmt.Println("✓ Token deployed successfully")
	} else {
		fmt.Println("\nTo deploy to the VM:")
		fmt.Printf("  hostodo agent regenerate %s --deploy\n", instanceID)
	}

	return nil
}

func deployTokenToInstance(ip, token string) error {
	// Use SSH to write the token to the instance
	// Ensure directory exists and write the token
	sshCmd := fmt.Sprintf("mkdir -p /etc/hostodo && echo '%s' > /etc/hostodo/agent-token && chmod 600 /etc/hostodo/agent-token", token)

	cmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=10",
		fmt.Sprintf("root@%s", ip),
		sshCmd,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
