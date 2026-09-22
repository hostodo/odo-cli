package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hostodo/odo-cli/v2/pkg/api"
)

func TestBuildPoolCheckoutRequestRequiresPlan(t *testing.T) {
	_, err := buildPoolCheckoutRequest(0, "monthly", "stripe_checkout", "", "", "", true)
	if err == nil || !strings.Contains(err.Error(), "plan") {
		t.Fatalf("error = %v, want required plan error", err)
	}
}

func TestBuildPoolCheckoutRequestRejectsInvalidBillingCycle(t *testing.T) {
	_, err := buildPoolCheckoutRequest(42, "weekly", "stripe_checkout", "", "", "", true)
	if err == nil || !strings.Contains(err.Error(), "billing cycle") {
		t.Fatalf("error = %v, want billing cycle error", err)
	}
}

func TestBuildPoolCheckoutRequestRequiresPaymentMethodIDForSavedCard(t *testing.T) {
	_, err := buildPoolCheckoutRequest(42, "monthly", "saved_card", "", "", "", false)
	if err == nil || !strings.Contains(err.Error(), "payment-method-id") {
		t.Fatalf("error = %v, want payment method id error", err)
	}
}

func TestBuildPoolCheckoutRequestQuoteOmitsPaymentFields(t *testing.T) {
	got, err := buildPoolCheckoutRequest(42, "annually", "stripe_checkout", "pm::ignored", "SAVE", "idem-ignored", true)
	if err != nil {
		t.Fatal(err)
	}
	want := api.ResourcePoolCheckoutRequest{PlanID: 42, BillingCycle: "annually", Promocode: "SAVE", QuoteOnly: true}
	if got != want {
		t.Fatalf("request = %+v, want %+v", got, want)
	}
}

func TestBuildPoolUpdateRequestRequiresChangedFlag(t *testing.T) {
	_, err := buildPoolUpdateRequest("", false, false, false)
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("error = %v, want at least one field error", err)
	}
}

func TestBuildPoolUpdateRequestPreservesExplicitAutorenewOff(t *testing.T) {
	got, err := buildPoolUpdateRequest("", false, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.AutorenewalEnabled == nil || *got.AutorenewalEnabled {
		t.Fatalf("autorenewal = %v, want explicit false pointer", got.AutorenewalEnabled)
	}
	if got.DisplayName != nil {
		t.Fatalf("display name = %v, want omitted", got.DisplayName)
	}
}

func TestConfirmActionYesIsExplicitNoninteractiveAcknowledgement(t *testing.T) {
	var out bytes.Buffer
	if err := confirmAction(strings.NewReader(""), &out, true, false, "Purchase Capacity for $5.00?", "PURCHASE"); err != nil {
		t.Fatalf("--yes confirmation returned error: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("--yes should not prompt, got %q", out.String())
	}
}

func TestConfirmActionNoninteractiveRequiresYes(t *testing.T) {
	var out bytes.Buffer
	err := confirmAction(strings.NewReader("PURCHASE\n"), &out, false, false, "Purchase Capacity for $5.00?", "PURCHASE")
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("error = %v, want --yes guidance", err)
	}
}

func TestConfirmActionInteractiveRequiresExactAcknowledgement(t *testing.T) {
	var out bytes.Buffer
	err := confirmAction(strings.NewReader("yes\n"), &out, false, true, "Purchase Capacity for $5.00?", "PURCHASE")
	if err == nil || !strings.Contains(err.Error(), "aborted") {
		t.Fatalf("error = %v, want aborted", err)
	}
	out.Reset()
	if err := confirmAction(strings.NewReader("PURCHASE\n"), &out, false, true, "Purchase Capacity for $5.00?", "PURCHASE"); err != nil {
		t.Fatalf("exact acknowledgement failed: %v", err)
	}
}

func TestFormatPoolQuoteIncludesFreshAmountAndMode(t *testing.T) {
	quote := api.ResourcePoolCheckoutResponse{
		Mode: "upgrade", PlanName: "Capacity 16G", BillingCycle: "monthly",
		AmountDueAfterCredit: "5.00", RecurringAmount: "30.00",
	}
	got := formatPoolQuote(quote)
	for _, want := range []string{"Capacity quote", "Upgrade", "$5.00", "$30.00", "Capacity 16G"} {
		if !strings.Contains(got, want) {
			t.Errorf("formatPoolQuote() missing %q:\n%s", want, got)
		}
	}
}

func TestPoolsHelpListsCapacityCRUDCommands(t *testing.T) {
	for _, name := range []string{"options", "quote", "purchase", "update", "cancel", "list", "show"} {
		command, _, err := poolsCmd.Find([]string{name})
		if err != nil || command == nil || command.Name() != name {
			t.Errorf("pools command %q is not registered", name)
		}
	}
}

func TestPoolsOutputFlagScope(t *testing.T) {
	for _, command := range poolsCmd.Commands() {
		t.Run(command.Name(), func(t *testing.T) {
			inherited := command.InheritedFlags()
			if inherited.Lookup("json") == nil {
				t.Fatal("--json must be available on every pools command")
			}
			for _, name := range []string{"simple", "details"} {
				if inherited.Lookup(name) != nil {
					t.Errorf("--%s must not be inherited", name)
				}
				want := command.Name() == "list" || command.Name() == "show"
				if got := command.LocalFlags().Lookup(name) != nil; got != want {
					t.Errorf("local --%s present = %t, want %t", name, got, want)
				}
			}
		})
	}
}
