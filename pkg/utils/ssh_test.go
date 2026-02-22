package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

func TestCalculateSSHFingerprint(t *testing.T) {
	// Use a minimal valid base64 blob as a test key
	keyData := []byte("test-key-data-for-fingerprint")
	b64Key := base64.StdEncoding.EncodeToString(keyData)
	pubKey := "ssh-rsa " + b64Key + " user@host"

	// Expected: SHA256 of the decoded key data, base64-encoded without padding
	hash := sha256.Sum256(keyData)
	expected := "SHA256:" + base64.RawStdEncoding.EncodeToString(hash[:])

	fingerprint, err := CalculateSSHFingerprint(pubKey)
	if err != nil {
		t.Fatalf("CalculateSSHFingerprint() error = %v", err)
	}

	if fingerprint != expected {
		t.Errorf("CalculateSSHFingerprint() = %q, want %q", fingerprint, expected)
	}
}

func TestCalculateSSHFingerprint_Format(t *testing.T) {
	keyData := []byte("another-test-key")
	b64Key := base64.StdEncoding.EncodeToString(keyData)
	pubKey := "ssh-ed25519 " + b64Key

	fingerprint, err := CalculateSSHFingerprint(pubKey)
	if err != nil {
		t.Fatalf("CalculateSSHFingerprint() error = %v", err)
	}

	// Should start with SHA256: prefix
	if !strings.HasPrefix(fingerprint, "SHA256:") {
		t.Errorf("fingerprint %q doesn't start with SHA256:", fingerprint)
	}

	// Should NOT have padding (uses RawStdEncoding)
	if strings.HasSuffix(fingerprint, "=") {
		t.Errorf("fingerprint %q has padding, expected no padding", fingerprint)
	}
}

func TestCalculateSSHFingerprint_NoComment(t *testing.T) {
	// Key without a comment (just type + key)
	keyData := []byte("no-comment-key")
	b64Key := base64.StdEncoding.EncodeToString(keyData)
	pubKey := "ssh-rsa " + b64Key

	_, err := CalculateSSHFingerprint(pubKey)
	if err != nil {
		t.Fatalf("CalculateSSHFingerprint() error = %v, want nil", err)
	}
}

func TestCalculateSSHFingerprint_InvalidFormat(t *testing.T) {
	tests := []struct {
		name   string
		pubKey string
	}{
		{"empty string", ""},
		{"single field", "ssh-rsa"},
		{"invalid base64", "ssh-rsa not-valid-base64!!!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateSSHFingerprint(tt.pubKey)
			if err == nil {
				t.Errorf("CalculateSSHFingerprint(%q) = nil error, want error", tt.pubKey)
			}
		})
	}
}
