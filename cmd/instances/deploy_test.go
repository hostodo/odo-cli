package instances

import (
	"testing"

	"github.com/hostodo/hostodo-cli/pkg/api"
)

var testTemplates = []api.Template{
	{ID: 1, Name: "Ubuntu 22.04"},
	{ID: 2, Name: "Ubuntu 24.04"},
	{ID: 3, Name: "Debian 12"},
	{ID: 4, Name: "AlmaLinux 9"},
}

var testRegions = []api.Region{
	{ID: 1, Name: "Los Angeles"},
	{ID: 2, Name: "Las Vegas"},
	{ID: 3, Name: "New York"},
	{ID: 4, Name: "New Jersey"},
}

var testPlans = []api.Plan{
	{ID: 1, Name: "KVM-1G", RAM: 1024, VCPU: 1, Disk: 25, Bandwidth: 2000, PriceMonthly: "5.99"},
	{ID: 2, Name: "KVM-2G", RAM: 2048, VCPU: 2, Disk: 50, Bandwidth: 3000, PriceMonthly: "9.99"},
	{ID: 3, Name: "KVM-4G", RAM: 4096, VCPU: 4, Disk: 80, Bandwidth: 4000, PriceMonthly: "19.99"},
}

func TestFindTemplate_ExactMatch(t *testing.T) {
	tmpl, err := findTemplate(testTemplates, "Ubuntu 22.04")
	if err != nil {
		t.Fatalf("findTemplate() error = %v", err)
	}
	if tmpl == nil || tmpl.Name != "Ubuntu 22.04" {
		t.Errorf("findTemplate() = %v, want Ubuntu 22.04", tmpl)
	}
}

func TestFindTemplate_CaseInsensitive(t *testing.T) {
	tmpl, err := findTemplate(testTemplates, "ubuntu 22.04")
	if err != nil {
		t.Fatalf("findTemplate() error = %v", err)
	}
	if tmpl == nil || tmpl.Name != "Ubuntu 22.04" {
		t.Errorf("findTemplate() = %v, want Ubuntu 22.04", tmpl)
	}
}

func TestFindTemplate_SubstringUnique(t *testing.T) {
	tmpl, err := findTemplate(testTemplates, "Debian")
	if err != nil {
		t.Fatalf("findTemplate() error = %v", err)
	}
	if tmpl == nil || tmpl.Name != "Debian 12" {
		t.Errorf("findTemplate() = %v, want Debian 12", tmpl)
	}
}

func TestFindTemplate_SubstringAmbiguous(t *testing.T) {
	_, err := findTemplate(testTemplates, "Ubuntu")
	if err == nil {
		t.Fatal("findTemplate() = nil error, want ambiguous error")
	}
	if want := "ambiguous"; !contains(err.Error(), want) {
		t.Errorf("findTemplate() error = %q, want error containing %q", err.Error(), want)
	}
}

func TestFindTemplate_NoMatch(t *testing.T) {
	tmpl, err := findTemplate(testTemplates, "Windows 11")
	if err != nil {
		t.Fatalf("findTemplate() error = %v", err)
	}
	if tmpl != nil {
		t.Errorf("findTemplate() = %v, want nil", tmpl)
	}
}

func TestFindRegion_ExactMatch(t *testing.T) {
	region, err := findRegion(testRegions, "New York")
	if err != nil {
		t.Fatalf("findRegion() error = %v", err)
	}
	if region == nil || region.Name != "New York" {
		t.Errorf("findRegion() = %v, want New York", region)
	}
}

func TestFindRegion_SubstringAmbiguous(t *testing.T) {
	// "New" matches both "New York" and "New Jersey"
	_, err := findRegion(testRegions, "New")
	if err == nil {
		t.Fatal("findRegion() = nil error, want ambiguous error")
	}
	if want := "ambiguous"; !contains(err.Error(), want) {
		t.Errorf("findRegion() error = %q, want error containing %q", err.Error(), want)
	}
}

func TestFindRegion_SubstringUnique(t *testing.T) {
	region, err := findRegion(testRegions, "York")
	if err != nil {
		t.Fatalf("findRegion() error = %v", err)
	}
	if region == nil || region.Name != "New York" {
		t.Errorf("findRegion() = %v, want New York", region)
	}
}

func TestFindPlan_ExactMatch(t *testing.T) {
	plan, err := findPlan(testPlans, "KVM-2G")
	if err != nil {
		t.Fatalf("findPlan() error = %v", err)
	}
	if plan == nil || plan.Name != "KVM-2G" {
		t.Errorf("findPlan() = %v, want KVM-2G", plan)
	}
}

func TestFindPlan_CaseInsensitive(t *testing.T) {
	plan, err := findPlan(testPlans, "kvm-2g")
	if err != nil {
		t.Fatalf("findPlan() error = %v", err)
	}
	if plan == nil || plan.Name != "KVM-2G" {
		t.Errorf("findPlan() = %v, want KVM-2G", plan)
	}
}

func TestFindPlan_SubstringMatch(t *testing.T) {
	plan, err := findPlan(testPlans, "2G")
	if err != nil {
		t.Fatalf("findPlan() error = %v", err)
	}
	if plan == nil || plan.Name != "KVM-2G" {
		t.Errorf("findPlan() = %v, want KVM-2G", plan)
	}
}

func TestFindPlan_NoMatch(t *testing.T) {
	plan, err := findPlan(testPlans, "KVM-8G")
	if err != nil {
		t.Fatalf("findPlan() unexpected error = %v", err)
	}
	if plan != nil {
		t.Errorf("findPlan() = %v, want nil", plan)
	}
}

func TestMapEventMessage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Cloning template 123", "Installing OS"},
		{"cloning disk", "Installing OS"},
		{"Configuring instance settings", "Configuring server"},
		{"Setting up cloud init", "Configuring network"},
		{"Applying cloud-init config", "Configuring network"},
		{"Password changed successfully", "Root password set"},
		{"Root password reset", "Root password set"},
		{"Configuring firewall rules", "Configuring firewall"},
		{"Setting DNS records", "Setting up DNS"},
		{"Resizing disk to 50G", "Resizing disk"},
		{"Disk expansion complete", "Resizing disk"},
		{"Instance created", "Starting server"},
		{"VM started successfully", "Starting server"},
		{"Some custom message", "Some custom message"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapEventMessage(tt.input)
			if got != tt.want {
				t.Errorf("mapEventMessage(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatRAM(t *testing.T) {
	tests := []struct {
		mb   int
		want string
	}{
		{1024, "1"},
		{2048, "2"},
		{4096, "4"},
		{512, "0"}, // 0.5 rounds to 0 (banker's rounding)
		{8192, "8"},
	}

	for _, tt := range tests {
		got := formatRAM(tt.mb)
		if got != tt.want {
			t.Errorf("formatRAM(%d) = %q, want %q", tt.mb, got, tt.want)
		}
	}
}

func TestFormatBW(t *testing.T) {
	tests := []struct {
		gb   int
		want string
	}{
		{1000, "1"},
		{2000, "2"},
		{3000, "3"},
		{500, "0"}, // 0.5 rounds to 0 (banker's rounding)
		{10000, "10"},
	}

	for _, tt := range tests {
		got := formatBW(tt.gb)
		if got != tt.want {
			t.Errorf("formatBW(%d) = %q, want %q", tt.gb, got, tt.want)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
