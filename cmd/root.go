package cmd

import (
	"fmt"
	"os"

	"github.com/hostodo/hostodo-cli/cmd/agent"
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
	Use:   "hostodo",
	Short: "Hostodo CLI - Manage your VPS instances from the command line",
	Long: `Hostodo CLI is a beautiful, interactive command-line interface for managing
your Hostodo VPS instances.

Features:
  - Interactive TUI with Bubble Tea
  - List and manage instances
  - Control instance power (start/stop/reboot)
  - Multiple output formats (interactive, JSON, simple table)
  - Secure credential storage in system keychain

Authentication:
  hostodo login                    # Authenticate with your account
  hostodo logout                   # Sign out
  hostodo whoami                   # Show current user

Instance Management:
  hostodo instances list           # List all your instances
  hostodo instances get <id>       # Get details about an instance
  hostodo instances start <id>     # Start an instance
  hostodo instances stop <id>      # Stop an instance
  hostodo instances reboot <id>    # Reboot an instance`,
	Version: Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Set version template
	rootCmd.SetVersionTemplate(fmt.Sprintf("hostodo version %s (commit: %s, built: %s)\n", Version, Commit, Date))

	// Add subcommands
	rootCmd.AddCommand(agent.AgentCmd)
	rootCmd.AddCommand(auth.AuthCmd)
	rootCmd.AddCommand(instances.InstancesCmd)

	// Root-level aliases for common auth commands
	rootCmd.AddCommand(loginAliasCmd)
	rootCmd.AddCommand(logoutAliasCmd)
	rootCmd.AddCommand(whoamiAliasCmd)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.hostodo/config.json)")
	rootCmd.PersistentFlags().String("api-url", "", "API URL (default is https://console.hostodo.com or $HOSTODO_API_URL)")
}

// loginAliasCmd is a convenience alias for 'auth login'
var loginAliasCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Hostodo (alias for 'auth login')",
	Long: `Authenticate with your Hostodo account using device flow.

This is a convenience alias for 'hostodo auth login'.

Example:
  hostodo login`,
	Run: func(cmd *cobra.Command, args []string) {
		// Find and execute the login subcommand directly
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

This is a convenience alias for 'hostodo auth logout'.

Example:
  hostodo logout`,
	Run: func(cmd *cobra.Command, args []string) {
		// Find and execute the logout subcommand directly
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

This is a convenience alias for 'hostodo auth whoami'.

Example:
  hostodo whoami`,
	Run: func(cmd *cobra.Command, args []string) {
		// Find and execute the whoami subcommand directly
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
