package deploy

import (
	"fmt"
	"regexp"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		hostname    string
		wantErr     bool
		errContains string
	}{
		{"valid simple", "my-server", false, ""},
		{"valid alphanumeric", "web1", false, ""},
		{"valid single char", "a", false, ""},
		{"valid complex", "test-123-abc", false, ""},
		{"empty string", "", true, "cannot be empty"},
		{"starts with hyphen", "-server", true, "cannot start or end with a hyphen"},
		{"ends with hyphen", "server-", true, "cannot start or end with a hyphen"},
		{"underscore", "my_server", true, "can only contain letters, numbers, and hyphens"},
		{"dot", "my.server", true, "can only contain letters, numbers, and hyphens"},
		{"space", "my server", true, "can only contain letters, numbers, and hyphens"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.hostname)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Validate(%q) = nil, want error containing %q", tt.hostname, tt.errContains)
				}
				if tt.errContains != "" {
					if got := err.Error(); !contains(got, tt.errContains) {
						t.Errorf("Validate(%q) error = %q, want it to contain %q", tt.hostname, got, tt.errContains)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("Validate(%q) = %v, want nil", tt.hostname, err)
				}
			}
		})
	}
}

func TestGenerate_NoCollisions(t *testing.T) {
	pattern := regexp.MustCompile(`^[a-z]+-[a-z]+$`)

	hostname, err := Generate(func(string) (bool, error) {
		return false, nil
	})
	if err != nil {
		t.Fatalf("Generate() error = %v, want nil", err)
	}
	if !pattern.MatchString(hostname) {
		t.Errorf("Generate() = %q, want match for %s", hostname, pattern.String())
	}
}

func TestGenerate_AlwaysCollides(t *testing.T) {
	pattern := regexp.MustCompile(`^[a-z]+-[a-z]+-\d{4}$`)

	hostname, err := Generate(func(string) (bool, error) {
		return true, nil
	})
	if err != nil {
		t.Fatalf("Generate() error = %v, want nil", err)
	}
	if !pattern.MatchString(hostname) {
		t.Errorf("Generate() = %q, want match for %s", hostname, pattern.String())
	}
}

func TestGenerate_ExistsCheckError(t *testing.T) {
	_, err := Generate(func(string) (bool, error) {
		return false, fmt.Errorf("connection refused")
	})
	if err == nil {
		t.Fatal("Generate() = nil, want error")
	}
	if got := err.Error(); !contains(got, "failed to check hostname existence") {
		t.Errorf("Generate() error = %q, want it to contain %q", got, "failed to check hostname existence")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
