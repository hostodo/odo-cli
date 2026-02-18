package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

func computeExpectedFingerprint(keyData string) string {
	decoded, _ := base64.StdEncoding.DecodeString(keyData)
	hash := sha256.Sum256(decoded)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(hash[:])
}

func TestCalculateSSHFingerprint_ValidEd25519(t *testing.T) {
	pubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com"
	keyData := "AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"
	expected := computeExpectedFingerprint(keyData)

	result, err := CalculateSSHFingerprint(pubKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("got %s, want %s", result, expected)
	}
}

func TestCalculateSSHFingerprint_ValidRSA(t *testing.T) {
	pubKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDyWc2/XFvlbAKPRBydPyDXpXb/JHkpWXhG+Ly2IIFVjzc4LdCAf0gIZRraC0lQuFQ0K3bDwGik8u66YCH29aaR2h/3ujmP6TZkrrsIWyWnji34nmwaM3WX8cJcp6Fa+UIIaDqk+2ttipBbHt3OaIGhjqzqbIVlMTULfIVreMP/zOzImQHasnn+QAsJCdrnJqNrHYprTc7pB0suQB9hL7sNFQSE7QMJnz6uPC4KrqMAWRPXNhj8eA/wpIG32wmrKYTZmWq6XmwieC0quGBbmd4dIaOkbyPc7oKYtqQhD2Br5Z591ehe3BIkHcW8nC33Mx4Wm3ioJ237dNv/g8VQVCOP test@example.com"
	keyData := "AAAAB3NzaC1yc2EAAAADAQABAAABAQDyWc2/XFvlbAKPRBydPyDXpXb/JHkpWXhG+Ly2IIFVjzc4LdCAf0gIZRraC0lQuFQ0K3bDwGik8u66YCH29aaR2h/3ujmP6TZkrrsIWyWnji34nmwaM3WX8cJcp6Fa+UIIaDqk+2ttipBbHt3OaIGhjqzqbIVlMTULfIVreMP/zOzImQHasnn+QAsJCdrnJqNrHYprTc7pB0suQB9hL7sNFQSE7QMJnz6uPC4KrqMAWRPXNhj8eA/wpIG32wmrKYTZmWq6XmwieC0quGBbmd4dIaOkbyPc7oKYtqQhD2Br5Z591ehe3BIkHcW8nC33Mx4Wm3ioJ237dNv/g8VQVCOP"
	expected := computeExpectedFingerprint(keyData)

	result, err := CalculateSSHFingerprint(pubKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("got %s, want %s", result, expected)
	}
}

func TestCalculateSSHFingerprint_InvalidFormat(t *testing.T) {
	_, err := CalculateSSHFingerprint("invalidkeydata")
	if err == nil {
		t.Fatal("expected error for invalid key format")
	}
	if !strings.Contains(err.Error(), "invalid SSH public key format") {
		t.Errorf("error %q should contain 'invalid SSH public key format'", err.Error())
	}
}

func TestCalculateSSHFingerprint_SingleField(t *testing.T) {
	_, err := CalculateSSHFingerprint("ssh-rsa")
	if err == nil {
		t.Fatal("expected error for single field input")
	}
	if !strings.Contains(err.Error(), "invalid SSH public key format") {
		t.Errorf("error %q should contain 'invalid SSH public key format'", err.Error())
	}
}

func TestCalculateSSHFingerprint_InvalidBase64(t *testing.T) {
	_, err := CalculateSSHFingerprint("ssh-rsa not-valid-base64!!! user@host")
	if err == nil {
		t.Fatal("expected error for invalid base64 data")
	}
	if !strings.Contains(err.Error(), "failed to decode SSH key") {
		t.Errorf("error %q should contain 'failed to decode SSH key'", err.Error())
	}
}
