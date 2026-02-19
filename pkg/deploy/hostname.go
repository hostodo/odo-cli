package deploy

import (
	"fmt"
	"math/rand"
	"regexp"
	"time"
)

// Word lists for hostname generation - mixed tone (neutral/techy + fun/playful)
var adjectives = []string{
	"bold", "brave", "calm", "clever", "cosmic", "crisp", "deft", "eager",
	"fast", "fierce", "grand", "keen", "live", "lucid", "merry", "noble",
	"prime", "quick", "rapid", "sharp", "sleek", "smart", "snug", "solid",
	"stout", "swift", "terse", "vast", "vivid", "warm", "wise", "agile",
	"azure", "blaze", "bright", "brisk", "chief", "clear", "coral", "cyber",
	"delta", "ember", "fern", "fleet", "forge", "frost", "gleam", "haze",
	"iron", "jade", "laser", "lunar", "maple", "neon", "onyx", "orbit",
	"pearl", "pixel", "polar", "prism", "pulse", "quartz", "raven", "ridge",
	"ruby", "sage", "sigma", "slate", "solar", "spark", "steel", "storm",
	"surge", "terra", "titan", "ultra", "vapor", "wave", "xenon", "zeal", "zen",
}

var nouns = []string{
	"tiger", "falcon", "otter", "panda", "wolf", "hawk", "lynx", "crane",
	"raven", "drake", "bear", "hound", "stag", "fox", "pike", "bass",
	"wren", "dove", "lark", "finch", "cedar", "oak", "elm", "pine",
	"birch", "aspen", "maple", "fern", "sage", "iris", "atlas", "prism",
	"nexus", "forge", "vault", "tower", "beacon", "bridge", "summit", "crest",
	"ridge", "reef", "mesa", "peak", "cliff", "grove", "haven", "shore",
	"bluff", "delta", "orbit", "comet", "nova", "quasar", "pulsar", "nebula",
	"photon", "boson", "meson", "gluon", "cipher", "vector", "matrix", "tensor",
	"kernel", "socket", "proxy", "router", "cache", "stack", "queue", "node",
	"shard", "codec", "modem", "pixel", "voxel", "waffle", "muffin", "pickle",
}

var hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// Generate creates a unique hostname using adjective-noun pattern with collision detection
func Generate(existsCheck func(string) (bool, error)) (string, error) {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		// Generate random adjective-noun combination
		adj := adjectives[rand.Intn(len(adjectives))]
		noun := nouns[rand.Intn(len(nouns))]
		hostname := fmt.Sprintf("%s-%s", adj, noun)

		// Check if hostname already exists
		exists, err := existsCheck(hostname)
		if err != nil {
			return "", fmt.Errorf("failed to check hostname existence: %w", err)
		}

		if !exists {
			return hostname, nil
		}

		// Collision detected, retry
	}

	// After max retries, fall back to adding a random 4-digit suffix
	adj := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	suffix := rand.Intn(10000)
	hostname := fmt.Sprintf("%s-%s-%04d", adj, noun, suffix)

	exists, err := existsCheck(hostname)
	if err != nil {
		return "", fmt.Errorf("failed to check hostname existence: %w", err)
	}
	if exists {
		return "", fmt.Errorf("could not generate a unique hostname after %d retries", maxRetries+1)
	}
	return hostname, nil
}

// Validate checks if a custom hostname meets the requirements
func Validate(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname cannot be empty")
	}

	// Check if hostname starts or ends with hyphen
	if hostname[0] == '-' || hostname[len(hostname)-1] == '-' {
		return fmt.Errorf("hostname cannot start or end with a hyphen")
	}

	// Check if hostname matches pattern (letters, numbers, hyphens only)
	if !hostnameRegex.MatchString(hostname) {
		return fmt.Errorf("hostname can only contain letters, numbers, and hyphens")
	}

	return nil
}
