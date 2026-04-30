package auth

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	serviceName    = "odo-cli"
	oldServiceName = "hostodo-cli"
	accountName    = "access-token"
)

// TokenStore manages CLI token storage
type TokenStore struct {
	fallbackPath    string
	oldFallbackPath string
}

// NewTokenStore creates a new token store
func NewTokenStore() *TokenStore {
	home, _ := os.UserHomeDir()
	return &TokenStore{
		fallbackPath:    filepath.Join(home, ".odo", "token"),
		oldFallbackPath: filepath.Join(home, ".hostodo", "token.enc"),
	}
}

// Save stores a token in keychain, falling back to a 0600 file
func (s *TokenStore) Save(token string) error {
	err := keyring.Set(serviceName, accountName, token)
	if err == nil {
		// Also delete any fallback files if keychain succeeds
		os.Remove(s.fallbackPath)
		os.Remove(s.oldFallbackPath)
		return nil
	}

	// Fallback to plain file with strict permissions.
	// The file is owner-read-only (0600); on a single-user machine this is
	// equivalent to the old hostname-derived encryption which provided no
	// meaningful protection beyond filesystem permissions.
	fmt.Println("Warning: System keychain unavailable, using file-based token storage")
	return s.saveToFile(token)
}

// Get retrieves token from keychain or fallback file
func (s *TokenStore) Get() (string, error) {
	// Try new service name first
	token, err := keyring.Get(serviceName, accountName)
	if err == nil {
		return token, nil
	}

	// Try old service name (migration fallback)
	token, err = keyring.Get(oldServiceName, accountName)
	if err == nil {
		return token, nil
	}

	// Try new fallback file path
	token, err = s.getFromFile()
	if err == nil {
		return token, nil
	}

	// Try old fallback file path (legacy encrypted format — best-effort)
	return s.getFromOldFile()
}

// Delete removes token from keychain and fallback file
func (s *TokenStore) Delete() error {
	// Delete from both keychain service names (ignore errors if not found)
	keyring.Delete(serviceName, accountName)
	keyring.Delete(oldServiceName, accountName)
	// Delete fallback files (ignore errors if not found)
	os.Remove(s.fallbackPath)
	os.Remove(s.oldFallbackPath)
	return nil
}

// saveToFile writes the token to a 0600 file
func (s *TokenStore) saveToFile(token string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.fallbackPath), 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	// Write with restrictive permissions (owner read/write only)
	if err := os.WriteFile(s.fallbackPath, []byte(token), 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}
	return nil
}

// getFromFile reads the token from the new fallback file
func (s *TokenStore) getFromFile() (string, error) {
	data, err := os.ReadFile(s.fallbackPath)
	if err != nil {
		return "", fmt.Errorf("not authenticated: %w", err)
	}
	return string(data), nil
}

// getFromOldFile attempts to read the old encrypted token file.
// The old format used AES-256-GCM with a hostname-derived key; since that
// key was not secret we just try the keychain migration path instead and
// return an error so the user is prompted to re-login.
func (s *TokenStore) getFromOldFile() (string, error) {
	if _, err := os.Stat(s.oldFallbackPath); os.IsNotExist(err) {
		return "", fmt.Errorf("not authenticated")
	}
	// Old encrypted file exists but we no longer carry the decryption logic.
	// Tell user to re-login which will write a clean token.
	return "", fmt.Errorf("legacy token format detected — please run 'odo login' to re-authenticate")
}

// Helper functions for package-level access
var defaultStore = NewTokenStore()

// ResetDefaultStore re-initialises the package-level store from the current
// HOME environment variable. Useful in tests that redirect HOME to a temp dir.
func ResetDefaultStore() {
	defaultStore = NewTokenStore()
}

// GetToken retrieves the stored access token
func GetToken() (string, error) {
	return defaultStore.Get()
}

// SaveToken stores an access token
func SaveToken(token string) error {
	return defaultStore.Save(token)
}

// DeleteToken removes the stored token
func DeleteToken() error {
	return defaultStore.Delete()
}

// IsAuthenticated checks if a token exists
func IsAuthenticated() bool {
	token, err := GetToken()
	return err == nil && token != ""
}
