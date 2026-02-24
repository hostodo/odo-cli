package resolver

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hostodo/hostodo-cli/pkg/api"
	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
	"github.com/spf13/cobra"
)

// MatchType describes how an instance was resolved
type MatchType string

const (
	MatchExact  MatchType = "exact"
	MatchPrefix MatchType = "prefix"
	MatchID     MatchType = "id"
)

// ResolveResult contains the resolved instance and how it was matched
type ResolveResult struct {
	Instance  *api.Instance
	MatchType MatchType
}

// Instance cache for shell completions
var (
	cachedInstances []api.Instance
	cacheExpiry     time.Time
	cacheMutex      sync.RWMutex
	cacheTTL        = 3 * time.Second
)

// ResolveInstance resolves a hostname, prefix, or instance ID to an instance
func ResolveInstance(client *api.Client, identifier string) (*ResolveResult, error) {
	// Fetch all instances
	instancesResp, err := client.ListInstances(1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch instances: %w", err)
	}

	instances := instancesResp.Results

	// Try exact hostname match (case-sensitive)
	for i := range instances {
		if instances[i].Hostname == identifier {
			return &ResolveResult{
				Instance:  &instances[i],
				MatchType: MatchExact,
			}, nil
		}
	}

	// Try unambiguous prefix match
	var matches []api.Instance
	for i := range instances {
		if strings.HasPrefix(instances[i].Hostname, identifier) {
			matches = append(matches, instances[i])
		}
	}

	if len(matches) == 1 {
		return &ResolveResult{
			Instance:  &matches[0],
			MatchType: MatchPrefix,
		}, nil
	} else if len(matches) > 1 {
		// Build list of matching hostnames
		var hostnames []string
		for _, m := range matches {
			hostnames = append(hostnames, m.Hostname)
		}
		return nil, fmt.Errorf("ambiguous hostname prefix '%s' — matches: %s", identifier, strings.Join(hostnames, ", "))
	}

	// Try instance ID match (exact string match)
	for i := range instances {
		if instances[i].InstanceID == identifier {
			return &ResolveResult{
				Instance:  &instances[i],
				MatchType: MatchID,
			}, nil
		}
	}

	// No matches found
	return nil, fmt.Errorf("no instance found matching '%s'", identifier)
}

// GetInstancesCached returns cached instances or fetches fresh ones if cache expired
func GetInstancesCached(client *api.Client) ([]api.Instance, error) {
	// Fast path: check if cache is valid (read lock)
	cacheMutex.RLock()
	if time.Now().Before(cacheExpiry) && cachedInstances != nil {
		instances := cachedInstances
		cacheMutex.RUnlock()
		return instances, nil
	}
	cacheMutex.RUnlock()

	// Slow path: refresh cache (write lock)
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// Double-check: another goroutine might have refreshed while we waited
	if time.Now().Before(cacheExpiry) && cachedInstances != nil {
		return cachedInstances, nil
	}

	// Fetch fresh instances
	instancesResp, err := client.ListInstances(1000, 0)
	if err != nil {
		return nil, err
	}

	// Update cache
	cachedInstances = instancesResp.Results
	cacheExpiry = time.Now().Add(cacheTTL)

	return cachedInstances, nil
}

// InvalidateCache clears the instance cache
func InvalidateCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	cacheExpiry = time.Time{}
}

// CompleteHostname provides shell completion for hostnames
func CompleteHostname(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Only complete if we don't have a hostname yet
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Check authentication
	if !auth.IsAuthenticated() {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Create API client
	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Fetch instances (cached)
	instances, err := GetInstancesCached(client)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Filter hostnames matching the prefix
	var completions []string
	for _, instance := range instances {
		if strings.HasPrefix(instance.Hostname, toComplete) {
			completions = append(completions, instance.Hostname)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
