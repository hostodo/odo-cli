package instances

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/auth"
	"github.com/hostodo/odo-cli/v2/pkg/config"
	"github.com/hostodo/odo-cli/v2/pkg/resolver"
	"github.com/spf13/cobra"
)

func init() {
	for _, enabled := range []bool{true, false} {
		name := "protect"
		if !enabled {
			name = "unprotect"
		}
		command := &cobra.Command{
			Use:               name + " <hostname-or-id>",
			Short:             name + " a VPS instance (requires vms:protection)",
			Args:              cobra.ExactArgs(1),
			ValidArgsFunction: resolver.CompleteHostname,
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := config.Load()
				if err != nil {
					return err
				}
				if !auth.IsAuthenticated() {
					return fmt.Errorf("not authenticated; run 'odo login --protection' and approve vms:protection")
				}
				client, err := api.NewClient(cfg)
				if err != nil {
					return err
				}
				result, err := resolver.ResolveInstance(client, args[0])
				if err != nil {
					return err
				}
				confirmation := ""
				if !enabled {
					yes, _ := cmd.Flags().GetBool("yes")
					confirmation, err = confirmUnprotect(cmd.InOrStdin(), cmd.OutOrStdout(), result.Instance.InstanceID, yes)
					if err != nil {
						return err
					}
				}
				instance, err := client.SetInstanceProtection(result.Instance.InstanceID, enabled, confirmation)
				if err != nil {
					return err
				}
				resolver.InvalidateCache()
				fmt.Fprintf(cmd.OutOrStdout(), "%s (%s): protection_enabled=%t\n", instance.Hostname, instance.InstanceID, instance.ProtectionEnabled)
				return nil
			},
		}
		if !enabled {
			command.Flags().BoolP("yes", "y", false, "Explicitly confirm unprotect for automation (never bypasses reinstall protection)")
		}
		InstancesCmd.AddCommand(command)
	}
}

func confirmUnprotect(in io.Reader, out io.Writer, instanceID string, yes bool) (string, error) {
	phrase := "UNPROTECT " + instanceID
	if !yes {
		fmt.Fprintf(out, "Disabling protection permits destructive changes. Type %s to confirm: ", phrase)
		line, err := bufio.NewReader(in).ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		if strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r") != phrase {
			return "", fmt.Errorf("unprotect cancelled: confirmation must exactly match %q", phrase)
		}
	}
	return phrase, nil
}
