package config

import (
	"os"
	"path/filepath"
	"testing"
)

// -------------------------------------------------------------------
// SetAPIURLOverride — the --api-url flag
// -------------------------------------------------------------------

func TestSetAPIURLOverride_Valid(t *testing.T) {
	t.Cleanup(func() { SetAPIURLOverride("") })

	if err := SetAPIURLOverride("https://uat-api.hostodo.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url := GetDefaultAPIURL(); url != "https://uat-api.hostodo.com" {
		t.Errorf("expected override URL, got %s", url)
	}
}

func TestSetAPIURLOverride_BeatsEnvVar(t *testing.T) {
	t.Setenv("HOSTODO_API_URL", "https://env.hostodo.com")
	t.Cleanup(func() { SetAPIURLOverride("") })

	if err := SetAPIURLOverride("https://flag.hostodo.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url := GetDefaultAPIURL(); url != "https://flag.hostodo.com" {
		t.Errorf("expected flag URL to beat env var, got %s", url)
	}
}

func TestSetAPIURLOverride_BeatsConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Cleanup(func() { SetAPIURLOverride("") })

	configDirPath := filepath.Join(home, configDir)
	if err := os.MkdirAll(configDirPath, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	data := []byte(`{"api_url": "https://file.hostodo.com", "device_id": "abc"}`)
	if err := os.WriteFile(filepath.Join(configDirPath, configFile), data, 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if err := SetAPIURLOverride("https://flag.hostodo.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIURL != "https://flag.hostodo.com" {
		t.Errorf("expected flag URL to beat config file, got %s", cfg.APIURL)
	}
}

func TestSetAPIURLOverride_HTTPRejected(t *testing.T) {
	if err := SetAPIURLOverride("http://evil.example.com"); err == nil {
		t.Error("expected error for http:// URL, got nil")
	}
	if url := GetDefaultAPIURL(); url != "https://api.hostodo.com" {
		t.Errorf("rejected override must not stick, got %s", url)
	}
}

func TestSetAPIURLOverride_InvalidURLRejected(t *testing.T) {
	if err := SetAPIURLOverride("not-a-url"); err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestSetAPIURLOverride_MissingHostRejected(t *testing.T) {
	if err := SetAPIURLOverride("https://"); err == nil {
		t.Error("expected error for URL without host, got nil")
	}
}

func TestSetAPIURLOverride_HTTPAllowedWithForce(t *testing.T) {
	t.Cleanup(func() {
		SetAllowHTTPAPIURL(false)
		SetAPIURLOverride("")
	})

	SetAllowHTTPAPIURL(true)
	if err := SetAPIURLOverride("http://localdev.hostodo.com:8000"); err != nil {
		t.Fatalf("unexpected error with force-http: %v", err)
	}
	if url := GetDefaultAPIURL(); url != "http://localdev.hostodo.com:8000" {
		t.Errorf("expected http override URL, got %s", url)
	}
}

func TestSetAPIURLOverride_NonHTTPSchemeRejectedWithForce(t *testing.T) {
	t.Cleanup(func() { SetAllowHTTPAPIURL(false) })

	SetAllowHTTPAPIURL(true)
	if err := SetAPIURLOverride("ftp://example.com"); err == nil {
		t.Error("expected error for ftp:// URL even with force-http, got nil")
	}
}

func TestGetDefaultAPIURL_HTTPEnvVarAllowedWithForce(t *testing.T) {
	t.Setenv("HOSTODO_API_URL", "http://localdev.hostodo.com:8000")
	t.Cleanup(func() { SetAllowHTTPAPIURL(false) })

	SetAllowHTTPAPIURL(true)
	if url := GetDefaultAPIURL(); url != "http://localdev.hostodo.com:8000" {
		t.Errorf("expected http env URL with force-http, got %s", url)
	}
}

func TestSetAPIURLOverride_EmptyClears(t *testing.T) {
	if err := SetAPIURLOverride("https://flag.hostodo.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := SetAPIURLOverride(""); err != nil {
		t.Fatalf("unexpected error clearing override: %v", err)
	}
	if url := GetDefaultAPIURL(); url != "https://api.hostodo.com" {
		t.Errorf("expected prod URL after clearing override, got %s", url)
	}
}
