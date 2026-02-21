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

If key-based authentication fails and the instance has a default password,
the command will automatically retry using sshpass for password authentication.

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

	// Collect extra args after --
	var extraArgs []string
	for i, arg := range os.Args {
		if arg == "--" {
			extraArgs = os.Args[i+1:]
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

	// Try SSH with key-based auth first
	sshArgs := buildSSHArgs(sshTarget, extraArgs)
	exitCode := runSSHCommand(sshBinary, sshArgs)

	if exitCode == 0 {
		return
	}

	// If SSH failed and we have a default password, retry with sshpass
	if instance.DefaultPassword != "" {
		sshpassBinary, err := exec.LookPath("sshpass")
		if err != nil {
			fmt.Fprintf(os.Stderr, "SSH key auth failed. Install sshpass to use password auth: brew install sshpass\n")
			os.Exit(exitCode)
		}

		fmt.Fprintf(os.Stderr, "Key auth failed, retrying with instance password...\n")

		// Use SSHPASS env var to avoid exposing password in process list
		sshpassArgs := []string{"-e", sshBinary}
		sshpassArgs = append(sshpassArgs, "-o", "PubkeyAuthentication=no")
		sshpassArgs = append(sshpassArgs, sshTarget)
		sshpassArgs = append(sshpassArgs, extraArgs...)

		passCmd := exec.Command(sshpassBinary, sshpassArgs...)
		passCmd.Env = append(os.Environ(), "SSHPASS="+instance.DefaultPassword)
		passCmd.Stdin = os.Stdin
		passCmd.Stdout = os.Stdout
		passCmd.Stderr = os.Stderr

		err = passCmd.Run()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "Error: ssh command failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	os.Exit(exitCode)
}

func buildSSHArgs(target string, extraArgs []string) []string {
	args := []string{target}
	args = append(args, extraArgs...)
	return args
}

func runSSHCommand(binary string, args []string) int {
	cmd := exec.Command(binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}
