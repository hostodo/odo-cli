package ui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hostodo/hostodo-cli/pkg/api"
)

func sampleInstance() api.Instance {
	return api.Instance{
		InstanceID:   "inst-abc123",
		Hostname:     "web-server-01",
		MainIP:       "192.168.1.10",
		Status:       "active",
		PowerStatus:  "running",
		RAM:          2048,
		VCPU:         2,
		Disk:         40,
		Bandwidth:    1000,
		BandwidthUsage: 250.5,
		IPs:          []string{"192.168.1.10"},
		BillingCycle:  "monthly",
		BillingAmount: "5.99",
		NextDueDate:   "2025-06-01",
		Plan:         api.Plan{Name: "VPS-1G"},
		Template:     api.Template{Name: "Ubuntu 22.04"},
		Node:         api.Node{Region: "Las Vegas, NV"},
	}
}

func suspendedInstance() api.Instance {
	inst := sampleInstance()
	inst.InstanceID = "inst-sus456"
	inst.Hostname = "suspended-box"
	inst.IsSuspended = true
	inst.SuspensionReason = "non-payment"
	inst.Status = "suspended"
	return inst
}

// --- FormatInstancesJSON ---

func TestFormatInstancesJSON_WithInstances(t *testing.T) {
	instances := []api.Instance{sampleInstance()}
	result, err := FormatInstancesJSON(instances)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must be valid JSON
	var parsed []api.Instance
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(parsed))
	}
	if parsed[0].Hostname != "web-server-01" {
		t.Errorf("expected hostname 'web-server-01', got %q", parsed[0].Hostname)
	}
	if parsed[0].MainIP != "192.168.1.10" {
		t.Errorf("expected main_ip '192.168.1.10', got %q", parsed[0].MainIP)
	}
}

func TestFormatInstancesJSON_EmptySlice(t *testing.T) {
	result, err := FormatInstancesJSON([]api.Instance{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "[]" {
		t.Errorf("expected '[]', got %q", result)
	}
}

// --- FormatInstancesSimpleTable ---

func TestFormatInstancesSimpleTable_Empty(t *testing.T) {
	result := FormatInstancesSimpleTable(nil)
	if result != "No instances found" {
		t.Errorf("expected 'No instances found', got %q", result)
	}
}

func TestFormatInstancesSimpleTable_WithInstances(t *testing.T) {
	instances := []api.Instance{sampleInstance()}
	result := FormatInstancesSimpleTable(instances)

	for _, expected := range []string{"HOSTNAME", "IP ADDRESS", "STATUS", "web-server-01", "192.168.1.10", "active"} {
		if !strings.Contains(result, expected) {
			t.Errorf("expected output to contain %q, got:\n%s", expected, result)
		}
	}
}

// --- FormatInstancesDetailedTable ---

func TestFormatInstancesDetailedTable_Empty(t *testing.T) {
	result := FormatInstancesDetailedTable(nil)
	if result != "No instances found" {
		t.Errorf("expected 'No instances found', got %q", result)
	}
}

func TestFormatInstancesDetailedTable_WithInstances(t *testing.T) {
	instances := []api.Instance{sampleInstance()}
	result := FormatInstancesDetailedTable(instances)

	for _, expected := range []string{"Instance:", "Hostname:", "IP Address:", "Resources:"} {
		if !strings.Contains(result, expected) {
			t.Errorf("expected output to contain %q, got:\n%s", expected, result)
		}
	}
}

func TestFormatInstancesDetailedTable_Suspended(t *testing.T) {
	instances := []api.Instance{suspendedInstance()}
	result := FormatInstancesDetailedTable(instances)

	if !strings.Contains(result, "Suspended:") {
		t.Errorf("expected output to contain 'Suspended:', got:\n%s", result)
	}
	if !strings.Contains(result, "non-payment") {
		t.Errorf("expected output to contain 'non-payment', got:\n%s", result)
	}
}

// --- FormatInvoicesTable ---

func TestFormatInvoicesTable_Empty(t *testing.T) {
	result := FormatInvoicesTable(nil)
	if result != "No invoices found" {
		t.Errorf("expected 'No invoices found', got %q", result)
	}
}

func TestFormatInvoicesTable_WithInvoices(t *testing.T) {
	invoices := []api.Invoice{
		{
			InvoiceNumber: "INV-001",
			Status:        "paid",
			Subtotal:      "9.99",
			DueDate:       "2025-07-01",
			Instances: []struct {
				Hostname string `json:"hostname"`
				MainIP   string `json:"main_ip"`
			}{
				{Hostname: "my-vps", MainIP: "10.0.0.1"},
			},
		},
	}
	result := FormatInvoicesTable(invoices)

	for _, expected := range []string{"INVOICE #", "AMOUNT", "STATUS", "INV-001", "$9.99", "paid"} {
		if !strings.Contains(result, expected) {
			t.Errorf("expected output to contain %q, got:\n%s", expected, result)
		}
	}
}

func TestFormatInvoicesTable_Overdue(t *testing.T) {
	invoices := []api.Invoice{
		{
			InvoiceNumber: "INV-LATE",
			Status:        "unpaid",
			Subtotal:      "4.99",
			DueDate:       "2020-01-01",
		},
	}
	result := FormatInvoicesTable(invoices)

	if !strings.Contains(result, "Overdue") {
		t.Errorf("expected output to contain 'Overdue' for past-due unpaid invoice, got:\n%s", result)
	}
}

func TestFormatInvoicesTable_MultipleInstances(t *testing.T) {
	invoices := []api.Invoice{
		{
			InvoiceNumber: "INV-MULTI",
			Status:        "paid",
			Subtotal:      "19.99",
			DueDate:       "2025-08-01",
			Instances: []struct {
				Hostname string `json:"hostname"`
				MainIP   string `json:"main_ip"`
			}{
				{Hostname: "host-a", MainIP: "10.0.0.1"},
				{Hostname: "host-b", MainIP: "10.0.0.2"},
			},
		},
	}
	result := FormatInvoicesTable(invoices)

	// Comma-separated hostnames (may be truncated, but short enough to fit)
	if !strings.Contains(result, "host-a") || !strings.Contains(result, "host-b") {
		t.Errorf("expected output to contain both hostnames, got:\n%s", result)
	}
}

// --- FormatSSHKeysTable ---

func TestFormatSSHKeysTable_Empty(t *testing.T) {
	result := FormatSSHKeysTable(nil)
	if result != "No SSH keys found" {
		t.Errorf("expected 'No SSH keys found', got %q", result)
	}
}

func TestFormatSSHKeysTable_WithKeys(t *testing.T) {
	keys := []SSHKeyDisplay{
		{
			Name:        "my-laptop",
			Fingerprint: "SHA256:abcdef1234567890",
			CreatedAt:   "2025-01-15",
		},
	}
	result := FormatSSHKeysTable(keys)

	for _, expected := range []string{"NAME", "FINGERPRINT", "DATE ADDED", "my-laptop", "SHA256:abcdef1234567890", "2025-01-15"} {
		if !strings.Contains(result, expected) {
			t.Errorf("expected output to contain %q, got:\n%s", expected, result)
		}
	}
}

// --- truncate ---

func TestTruncate_WithinLimit(t *testing.T) {
	result := truncate("hello", 10)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestTruncate_ExactLimit(t *testing.T) {
	result := truncate("hello", 5)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestTruncate_ExceedsLimit(t *testing.T) {
	result := truncate("hello world", 8)
	if result != "hello..." {
		t.Errorf("expected 'hello...', got %q", result)
	}
}

func TestTruncate_VeryShortLimit(t *testing.T) {
	result := truncate("hello", 3)
	if result != "hel" {
		t.Errorf("expected 'hel', got %q", result)
	}
}

func TestTruncate_LimitTwo(t *testing.T) {
	result := truncate("hello", 2)
	if result != "he" {
		t.Errorf("expected 'he', got %q", result)
	}
}
