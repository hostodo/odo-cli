package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const (
	configDir  = ".hostodo"
	configFile = "config.json"
)

// Config represents the CLI configuration
type Config struct {
	APIURL   string `json:"api_url"`
	DeviceID string `json:"device_id"` // Persistent UUID for device identification
}

// GetConfigPath returns the full path to the config file
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	configPath := filepath.Join(home, configDir, configFile)
	return configPath, nil
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDirPath := filepath.Join(home, configDir)

	// Create directory with 0700 permissions (owner read/write/execute only)
	if err := os.MkdirAll(configDirPath, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return nil
}

// Load reads the configuration from disk
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return GetDefaultConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// If API URL is not set in config, use default
	if config.APIURL == "" {
		config.APIURL = GetDefaultAPIURL()
	}

	return &config, nil
}

// Save writes the configuration to disk
func Save(config *Config) error {
	// Ensure config directory exists
	if err := EnsureConfigDir(); err != nil {
		return err
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file with 0600 permissions (owner read/write only)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Clear removes the configuration file
func Clear() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Remove config file
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config file: %w", err)
	}

	return nil
}

// GetDefaultAPIURL returns the default API URL
func GetDefaultAPIURL() string {
	// Check environment variable first
	if apiURL := os.Getenv("HOSTODO_API_URL"); apiURL != "" {
		return apiURL
	}
	// Default to production API
	return "https://api.hostodo.com"
}

// GetDefaultConfig returns a default configuration
func GetDefaultConfig() *Config {
	return &Config{
		APIURL: GetDefaultAPIURL(),
	}
}

// GetOrCreateDeviceID returns the device ID from config, or generates and saves a new one
func GetOrCreateDeviceID(config *Config) (string, error) {
	// If device ID already exists, return it
	if config.DeviceID != "" {
		return config.DeviceID, nil
	}

	// Generate new UUID
	deviceID := uuid.New().String()
	config.DeviceID = deviceID

	// Save config with new device ID
	if err := Save(config); err != nil {
		return "", fmt.Errorf("failed to save device ID: %w", err)
	}

	return deviceID, nil
}
