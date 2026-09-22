package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hostodo/odo-cli/v2/pkg/api"
)

func TestPoolCheckoutReplayOutput(t *testing.T) {
	for _, tt := range []struct {
		phase, status, invoiceStatus, want string
	}{
		{"preparing", "processing", "", "processing; payment outcome is not yet confirmed"},
		{"order_linked", "", "unpaid", "processing; payment outcome is not yet confirmed"},
		{"provider_started", "", "unpaid", "processing; payment outcome is not yet confirmed"},
		{"unknown", "", "unpaid", "unknown outcome"},
		{"", "", "", "unknown outcome"},
		{"completed", "", "unpaid", "completed; invoice unpaid"},
		{"completed", "", "paid", "completed; invoice paid"},
	} {
		for _, jsonMode := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%s/json=%t", tt.phase, tt.invoiceStatus, jsonMode), func(t *testing.T) {
				body := fmt.Sprintf(`{"idempotent_replay":true,"phase":%q,"status":%q,"invoice_status":%q,"checkout_url":"https://provider.example/stale-secret","checkout":{"client_secret":"secret"},"provider_token":"token","order_number":"ORD-1","order_status":"pending","invoice_number":"INV-1","invoice_url":"https://panel.example/billing/invoices/INV-1?provider_secret=invoice-secret","amount_due":"5.0001","unknown":9007199254740993}`, tt.phase, tt.status, tt.invoiceStatus)
				calls := 0
				client := poolTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					calls++
					var req api.ResourcePoolCheckoutRequest
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Error(err)
					}
					if req.QuoteOnly {
						fmt.Fprint(w, `{"confirmation":"PURCHASE CAPACITY","mode":"purchase","unit_price":"5.0001","recurring_amount":"5.0001","amount_due_after_credit":"5.0001"}`)
						return
					}
					fmt.Fprint(w, body)
				})
				req, _ := buildPoolCheckoutRequest(42, "monthly", "stripe_checkout", "", "", "replay-key", false)
				cmd, out, diagnostics := poolTestCommand()
				if err := runPoolCheckout(cmd, client, req, true, jsonMode); err != nil {
					t.Fatal(err)
				}
				if calls != 2 {
					t.Fatalf("calls=%d", calls)
				}
				for name, stream := range map[string]string{"stdout": out.String(), "stderr": diagnostics.String()} {
					for _, secret := range []string{"stale-secret", "client_secret", "provider_token", "invoice-secret"} {
						if strings.Contains(stream, secret) {
							t.Fatalf("%s exposed replay secret %q: %s", name, secret, stream)
						}
					}
				}
				if jsonMode {
					var safe map[string]any
					if err := json.Unmarshal(out.Bytes(), &safe); err != nil {
						t.Fatalf("stdout=%s err=%v", out, err)
					}
					for _, forbidden := range []string{"checkout_url", "checkout", "client_secret", "provider_token", "invoice_url", "unknown"} {
						if _, ok := safe[forbidden]; ok {
							t.Fatalf("unsafe replay field %q in %s", forbidden, out)
						}
					}
					if safe["order_number"] != "ORD-1" || !strings.Contains(diagnostics.String(), "Capacity checkout replay:") {
						t.Fatalf("stdout=%s stderr=%s", out, diagnostics)
					}
					return
				}
				if out.Len() != 0 || strings.Contains(diagnostics.String(), "checkout submitted") || strings.Contains(diagnostics.String(), "Invoice URL") {
					t.Fatalf("stdout=%s stderr=%s", out, diagnostics)
				}
				for _, want := range []string{tt.want, "Order: ORD-1", "Order status: pending", "Invoice: INV-1", "Amount due: $5.0001"} {
					if !strings.Contains(diagnostics.String(), want) {
						t.Errorf("stderr missing %q: %s", want, diagnostics)
					}
				}
				if tt.phase != "" && !strings.Contains(diagnostics.String(), "Phase: "+tt.phase) || tt.status != "" && !strings.Contains(diagnostics.String(), "Status: "+tt.status) {
					t.Errorf("missing replay state: %s", diagnostics)
				}
			})
		}
	}
}

func TestPoolReplayJSONOmitsUnknownAmount(t *testing.T) {
	data, err := marshalPoolReplay(api.ResourcePoolCheckoutResponse{
		IdempotentReplay: true,
		Phase:            "preparing",
		Status:           "processing",
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded["amount_due"]; ok {
		t.Fatalf("unknown amount rendered as %v", decoded["amount_due"])
	}
}
