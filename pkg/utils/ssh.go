package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// CalculateSSHFingerprint calculates the SHA256 fingerprint of an SSH public key
// Returns fingerprint in OpenSSH format: "SHA256:<base64-hash-without-padding>"
func CalculateSSHFingerprint(publicKey string) (string, error) {
	// Parse SSH public key format: "ssh-rsa AAAAB3... [optional comment]"
	parts := strings.Fields(publicKey)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid SSH public key format: expected at least 2 fields")
	}

	// The actual key data is in the second field (base64-encoded)
	keyData := parts[1]

	// Decode the base64 key data
	decoded, err := base64.StdEncoding.DecodeString(keyData)
	if err != nil {
		return "", fmt.Errorf("failed to decode SSH key: %w", err)
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(decoded)

	// Encode as base64 without padding (RawStdEncoding) to match OpenSSH format
	fingerprint := "SHA256:" + base64.RawStdEncoding.EncodeToString(hash[:])

	return fingerprint, nil
}
