package cmd

import (
	"fmt"
	"os"

	"github.com/hostodo/hostodo-cli/cmd/auth"
	"github.com/hostodo/hostodo-cli/cmd/instances"
	"github.com/spf13/cobra"
)

var (
	// Version information (will be set during build via ldflags)
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	cfgFile string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "odo",
	Short: "Hostodo CLI - Manage your VPS instances from the command line",
	Long: `odo is the official CLI for managing Hostodo VPS instances.

Authentication:
  odo login                        # Authenticate with your account
  odo logout                       # Sign out
  odo whoami                       # Show current user

Instances:
  odo instances                    # List all instances
  odo instances deploy             # Deploy a new VPS instance
  odo instances ssh <hostname>     # SSH to an instance
  odo instances start <hostname>   # Start an instance
  odo instances stop <hostname>    # Stop an instance
  odo instances restart <hostname> # Restart an instance
  odo instances status <hostname>  # Show instance details
  odo instances rename <h> <new>   # Rename an instance
  odo instances reinstall <h>      # Reinstall OS

Billing:
  odo invoices                     # List your invoices
  odo pay <invoice-id>             # Pay an invoice

SSH Keys:
  odo keys list                    # List your SSH keys
  odo keys add <name> <key>        # Add a new SSH key
  odo keys remove <name>           # Remove an SSH key`,
	Version: Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Set version template
	rootCmd.SetVersionTemplate(fmt.Sprintf("odo version %s (commit: %s, built: %s)\n", Version, Commit, Date))

	// Auth subcommand group
	rootCmd.AddCommand(auth.AuthCmd)

	// Instances namespace (primary)
	rootCmd.AddCommand(instances.InstancesCmd)

	// Billing commands
	rootCmd.AddCommand(invoicesCmd)
	rootCmd.AddCommand(payCmd)

	// SSH key management
	rootCmd.AddCommand(keysCmd)

	// Utility commands
	rootCmd.AddCommand(completionCmd)

	// Root-level auth aliases (visible)
	rootCmd.AddCommand(loginAliasCmd)
	rootCmd.AddCommand(logoutAliasCmd)
	rootCmd.AddCommand(whoamiAliasCmd)

	// Hidden root-level shortcuts for instance commands (backward compat)
	rootCmd.AddCommand(makeHiddenShortcut("list", "ls", "ps", "List all your instances", instances.ListCmd))
	rootCmd.AddCommand(makeHiddenShortcut("status", "", "", "Show detailed instance information", instances.StatusCmd))
	rootCmd.AddCommand(makeHiddenShortcut("start", "", "", "Start a stopped instance", instances.StartCmd))
	rootCmd.AddCommand(makeHiddenShortcut("stop", "", "", "Stop a running instance", instances.StopCmd))
	rootCmd.AddCommand(makeHiddenShortcut("restart", "", "", "Restart an instance", instances.RestartCmd))
	rootCmd.AddCommand(makeHiddenShortcut("ssh", "", "", "Connect to an instance via SSH", instances.SSHCmd))
	rootCmd.AddCommand(makeHiddenShortcut("rename", "", "", "Rename an instance", instances.RenameCmd))
	rootCmd.AddCommand(makeHiddenShortcut("deploy", "new", "create", "Deploy a new VPS instance", instances.DeployCmd))

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.odo/config.json)")
	rootCmd.PersistentFlags().String("api-url", "", "API URL (default is https://api.hostodo.com or $HOSTODO_API_URL)")
	rootCmd.PersistentFlags().MarkHidden("api-url")
}

// makeHiddenShortcut creates a hidden root-level shortcut that delegates to the given subcommand.
// alias1 and alias2 are optional additional aliases (pass "" to skip).
func makeHiddenShortcut(use, alias1, alias2, short string, target *cobra.Command) *cobra.Command {
	aliases := []string{}
	if alias1 != "" {
		aliases = append(aliases, alias1)
	}
	if alias2 != "" {
		aliases = append(aliases, alias2)
	}
	shortcut := &cobra.Command{
		Use:                use,
		Aliases:            aliases,
		Short:              short,
		Hidden:             true,
		DisableFlagParsing: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Delegate to the instances subcommand's RunE or Run
			if target.RunE != nil {
				return target.RunE(cmd, args)
			}
			if target.Run != nil {
				target.Run(cmd, args)
				return nil
			}
			return nil
		},
		ValidArgsFunction: target.ValidArgsFunction,
		Args:              target.Args,
	}
	// Copy flags from target so --json, --force, etc. work at root level too
	shortcut.Flags().AddFlagSet(target.Flags())
	return shortcut
}

// loginAliasCmd is a convenience alias for 'auth login'
var loginAliasCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Hostodo (alias for 'auth login')",
	Long: `Authenticate with your Hostodo account using device flow.

This is a convenience alias for 'odo auth login'.

Example:
  odo login`,
	Run: func(cmd *cobra.Command, args []string) {
		loginCmd, _, err := auth.AuthCmd.Find([]string{"login"})
		if err != nil || loginCmd == nil {
			fmt.Fprintf(os.Stderr, "Error: login command not found\n")
			os.Exit(1)
		}
		loginCmd.Run(cmd, args)
	},
}

// logoutAliasCmd is a convenience alias for 'auth logout'
var logoutAliasCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out from Hostodo (alias for 'auth logout')",
	Long: `Sign out from your Hostodo account.

This is a convenience alias for 'odo auth logout'.

Example:
  odo logout`,
	Run: func(cmd *cobra.Command, args []string) {
		logoutCmd, _, err := auth.AuthCmd.Find([]string{"logout"})
		if err != nil || logoutCmd == nil {
			fmt.Fprintf(os.Stderr, "Error: logout command not found\n")
			os.Exit(1)
		}
		logoutCmd.Run(cmd, args)
	},
}

// whoamiAliasCmd is a convenience alias for 'auth whoami'
var whoamiAliasCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display current logged-in user (alias for 'auth whoami')",
	Long: `Show information about the currently authenticated user.

This is a convenience alias for 'odo auth whoami'.

Example:
  odo whoami`,
	Run: func(cmd *cobra.Command, args []string) {
		whoamiCmd, _, err := auth.AuthCmd.Find([]string{"whoami"})
		if err != nil || whoamiCmd == nil {
			fmt.Fprintf(os.Stderr, "Error: whoami command not found\n")
			os.Exit(1)
		}
		whoamiCmd.Run(cmd, args)
	},
}

func initConfig() {
	// Configuration is loaded on-demand by each command
	// This allows flexibility for different authentication states
}

// checkAuth verifies that the user is authenticated
func checkAuth() error {
	// This will be called by commands that require authentication
	// Implemented in each command as needed
	return nil
}

func exitWithError(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+msg+"\n", args...)
	os.Exit(1)
}
