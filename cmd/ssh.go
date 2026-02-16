package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/hostodo/hostodo-cli/pkg/resolver"
	"github.com/spf13/cobra"
)

var sshUser string

var sshCmd = &cobra.Command{
	Use:   "ssh <hostname>",
	Short: "Connect to an instance via SSH",
	Long: `Connect to an instance via SSH using the system ssh binary.

The SSH command resolves the hostname, checks if the instance is running,
auto-detects the SSH user from the instance template, and then executes
the system ssh binary with proper passthrough.

This allows you to use all your existing SSH configuration (~/.ssh/config),
agent forwarding, ProxyJump, and any other ssh features.

Examples:
  # Connect to an instance (auto-detects user from template)
  hostodo ssh mybox

  # Connect as a specific user
  hostodo ssh mybox --user ubuntu
  hostodo ssh mybox -u root

  # Pass additional flags to ssh (after --)
  hostodo ssh mybox -- -L 8080:localhost:8080
  hostodo ssh mybox -- -D 1080 -N
  hostodo ssh mybox -- -A -v

Note: Everything after -- is passed directly to the ssh binary.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: resolver.CompleteHostname,
	Run:               runSSH,
}

func init() {
	sshCmd.Flags().StringVarP(&sshUser, "user", "u", "", "SSH user (default: auto-detect from template)")
}

func runSSH(cmd *cobra.Command, args []string) {
	hostname := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Check authentication
	if !auth.IsAuthenticated() {
		fmt.Fprintf(os.Stderr, "Error: not authenticated. Run 'hostodo login' first.\n")
		os.Exit(1)
	}

	// Create API client
	client, err := api.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create API client: %v\n", err)
		os.Exit(1)
	}

	// Resolve hostname
	result, err := resolver.ResolveInstance(client, hostname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	instance := result.Instance

	// Check power status
	powerStatus, err := client.GetInstancePowerStatus(instance.InstanceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to check power status: %v\n", err)
		os.Exit(1)
	}

	if powerStatus != "running" {
		fmt.Fprintf(os.Stderr, "Error: Instance '%s' is stopped. Run 'hostodo start %s' first.\n", hostname, hostname)
		os.Exit(1)
	}

	// Determine SSH user
	effectiveSshUser := sshUser
	if effectiveSshUser == "" {
		// Auto-detect from template
		effectiveSshUser = instance.Template.DefaultUsername
	}
	if effectiveSshUser == "" {
		// Final fallback
		effectiveSshUser = "root"
	}

	// Build SSH arguments
	sshTarget := fmt.Sprintf("%s@%s", effectiveSshUser, instance.MainIP)
	sshArgs := []string{sshTarget}

	// Scan for -- separator in os.Args
	for i, arg := range os.Args {
		if arg == "--" {
			// Everything after -- is passed to ssh
			sshArgs = append(sshArgs, os.Args[i+1:]...)
			break
		}
	}

	// Find ssh binary
	sshBinary, err := exec.LookPath("ssh")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: ssh binary not found in PATH\n")
		os.Exit(1)
	}

	// Print connecting message (to stderr to not interfere with piping)
	fmt.Fprintf(os.Stderr, "Connecting to %s (%s) as %s...\n", instance.Hostname, instance.MainIP, effectiveSshUser)

	// Execute SSH with passthrough
	sshCmd := exec.Command(sshBinary, sshArgs...)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	err = sshCmd.Run()
	if err != nil {
		// Try to get the exit code from the ssh process
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		// Fallback to generic error
		fmt.Fprintf(os.Stderr, "Error: ssh command failed: %v\n", err)
		os.Exit(1)
	}
}
