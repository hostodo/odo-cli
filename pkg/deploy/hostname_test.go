package deploy

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		wantErr  bool
		errMsg   string
	}{
		{"valid simple", "mybox", false, ""},
		{"valid with hyphen", "my-box", false, ""},
		{"valid with numbers", "box123", false, ""},
		{"valid adjective-noun", "swift-falcon", false, ""},
		{"valid single char", "a", false, ""},
		{"empty", "", true, "hostname cannot be empty"},
		{"starts with hyphen", "-mybox", true, "cannot start or end with a hyphen"},
		{"ends with hyphen", "mybox-", true, "cannot start or end with a hyphen"},
		{"too long", strings.Repeat("a", 64), true, "63 characters or less"},
		{"max length", strings.Repeat("a", 63), false, ""},
		{"contains underscore", "my_box", true, "letters, numbers, and hyphens"},
		{"contains dot", "my.box", true, "letters, numbers, and hyphens"},
		{"contains space", "my box", true, "letters, numbers, and hyphens"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.hostname)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate(%q) = nil, want error containing %q", tt.hostname, tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate(%q) error = %q, want error containing %q", tt.hostname, err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate(%q) = %v, want nil", tt.hostname, err)
				}
			}
		})
	}
}

func TestGenerate_NoCollisions(t *testing.T) {
	// existsCheck always returns false (no collisions)
	hostname, err := Generate(func(h string) (bool, error) {
		return false, nil
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should be adjective-noun format
	parts := strings.Split(hostname, "-")
	if len(parts) != 2 {
		t.Errorf("Generate() = %q, want adjective-noun format (2 parts)", hostname)
	}

	// Should pass validation
	if err := Validate(hostname); err != nil {
		t.Errorf("Generated hostname %q fails validation: %v", hostname, err)
	}
}

func TestGenerate_WithCollisions(t *testing.T) {
	callCount := 0
	// First 10 calls return true (collision), then false
	hostname, err := Generate(func(h string) (bool, error) {
		callCount++
		if callCount <= 10 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// After 10 retries, should fall back to adjective-noun-NNNN format
	parts := strings.Split(hostname, "-")
	if len(parts) != 3 {
		t.Errorf("Generate() = %q, want adjective-noun-NNNN format (3 parts) after max retries", hostname)
	}

	if err := Validate(hostname); err != nil {
		t.Errorf("Generated hostname %q fails validation: %v", hostname, err)
	}
}

func TestGenerate_CheckError(t *testing.T) {
	_, err := Generate(func(h string) (bool, error) {
		return false, fmt.Errorf("network error")
	})
	if err == nil {
		t.Fatal("Generate() = nil error, want error")
	}
	if !strings.Contains(err.Error(), "network error") {
		t.Errorf("Generate() error = %q, want error containing 'network error'", err.Error())
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	// Generate multiple hostnames and check they use valid words
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		hostname, err := Generate(func(h string) (bool, error) {
			return false, nil
		})
		if err != nil {
			t.Fatalf("Generate() iteration %d error = %v", i, err)
		}
		seen[hostname] = true
	}

	// With 77 adjectives * 72 nouns = 5544 combos, 50 samples should have some variety
	if len(seen) < 10 {
		t.Errorf("Generated only %d unique hostnames out of 50, expected more variety", len(seen))
	}
}
