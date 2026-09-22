package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/config"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

func poolTestClient(t *testing.T, handler http.HandlerFunc) *api.Client {
	t.Helper()
	testHome := t.TempDir()
	t.Setenv("HOME", testHome)
	t.Setenv("USERPROFILE", testHome)
	keyring.MockInit()
	if err := keyring.Set("odo-cli", "access-token", "capacity-test-token"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(keyring.MockInit)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/auth/" {
			id := 123
			if r.Header.Get("Authorization") == "Bearer different-login" {
				id = 456
			}
			fmt.Fprintf(w, `{"user_id":%d,"email":"capacity@example.test","is_email_verified":true}`, id)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	return &api.Client{BaseURL: server.URL, HTTPClient: server.Client()}
}

func poolTestCommand() (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	cmd := &cobra.Command{}
	out, diagnostics := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(diagnostics)
	cmd.SetIn(strings.NewReader("PURCHASE\nCANCEL\n"))
	return cmd, out, diagnostics
}

func TestPoolCheckoutWorkflow(t *testing.T) {
	for _, tt := range []struct {
		name, method, methodID, key, mode string
		jsonMode, quoteOnly, yes          bool
		quoteStatus                       int
		quoteBody, wantError              string
	}{
		{name: "hosted checkout", method: "stripe_checkout", key: "retry-123", yes: true},
		{name: "PayPal confirmation", method: "paypal", yes: true},
		{name: "Alipay confirmation", method: "alipay", yes: true},
		{name: "crypto confirmation", method: "crypto", yes: true},
		{name: "credit confirmation", method: "credit", yes: true},
		{name: "saved card", method: "saved_card", methodID: "pm::123", yes: true},
		{name: "raw purchase JSON", method: "stripe_checkout", yes: true, jsonMode: true},
		{name: "saved card upgrade JSON", method: "saved_card", methodID: "pm::123", key: "retry-card-upgrade", yes: true, jsonMode: true},
		{name: "new hosted purchase", mode: "purchase", method: "stripe_checkout", key: "retry-new", yes: true},
		{name: "new hosted purchase JSON", mode: "purchase", method: "stripe_checkout", yes: true, jsonMode: true},
		{name: "new saved card purchase", mode: "purchase", method: "saved_card", methodID: "pm::123", yes: true},
		{name: "new saved card purchase JSON", mode: "purchase", method: "saved_card", methodID: "pm::123", key: "retry-card-new", yes: true, jsonMode: true},
		{name: "quote only", quoteOnly: true},
		{name: "raw quote JSON", quoteOnly: true, jsonMode: true},
		{name: "noninteractive purchase", method: "stripe_checkout", wantError: "--yes"},
		{name: "JSON is not consent", method: "saved_card", methodID: "pm::123", jsonMode: true, wantError: "--yes"},
		{name: "quote fails", method: "stripe_checkout", yes: true, quoteStatus: 400, quoteBody: `{"detail":"tier unavailable"}`, wantError: "tier unavailable"},
		{name: "missing price", method: "stripe_checkout", yes: true, quoteBody: `{"plan_id":42}`, wantError: "missing amount_due_after_credit"},
		{name: "zero due", method: "saved_card", methodID: "pm::123", yes: true, quoteBody: `{"confirmation":"PURCHASE CAPACITY","mode":"upgrade","existing_pool_id":"pool::zero","unit_price":"7.5000","amount_due_after_credit":"0.00","recurring_amount":"30.00"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd, out, diagnostics := poolTestCommand()
			quoteBody := tt.quoteBody
			if quoteBody == "" {
				mode, poolID := tt.mode, `null`
				if mode == "" {
					mode, poolID = "upgrade", `"pool::existing"`
				}
				quoteBody = fmt.Sprintf(`{"confirmation":"PURCHASE CAPACITY","plan_id":42,"plan_name":"Capacity 16G","mode":%q,"existing_pool_id":%s,"unit_price":"7.5000000000000001","billing_cycle":"annually","amount_due_after_credit":"5.0001","recurring_amount":"30.00","unknown":9007199254740993}`, mode, poolID)
			}
			if tt.method == "saved_card" && tt.quoteStatus == 0 {
				quoteBody = strings.ReplaceAll(quoteBody, "5.0001", "5.00")
				amount := "5.00"
				if tt.name == "zero due" {
					amount = "0.00"
				}
				quoteBody = strings.TrimSuffix(quoteBody, "}") + fmt.Sprintf(`,"payment_confirmation":"CHARGE SAVED CARD %s ENDING 4242 FOR %s"}`, tt.methodID, amount)
			}
			checkoutBody := `{"plan_id":42,"plan_name":"Capacity 16G","order_number":"ORD-1","invoice_number":"INV-1","amount_due":"5.0001","payment_method":"` + tt.method + `","unknown":9007199254740993`
			if tt.method == "stripe_checkout" {
				checkoutBody += `,"checkout_url":"https://pay.example/checkout"`
			}
			checkoutBody += "}"
			calls := 0
			var purchased api.ResourcePoolCheckoutRequest
			client := poolTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodPost || r.URL.Path != "/client/resource-pools/checkout/" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				var body map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if calls == 1 {
					if string(body["quote_only"]) != "true" || string(body["plan_id"]) != "42" || string(body["billing_cycle"]) != `"annually"` || string(body["promocode"]) != `"SAVE"` {
						t.Errorf("unexpected quote: %s", body)
					}
					for _, field := range []string{"idempotency_key", "expected_quote", "confirmation", "payment_confirmation", "approved_charge_amount"} {
						if _, ok := body[field]; ok {
							t.Errorf("quote contains %s", field)
						}
					}
					if tt.method == "saved_card" {
						if string(body["payment_method"]) != `"saved_card"` || string(body["payment_method_id"]) != fmt.Sprintf("%q", tt.methodID) {
							t.Errorf("saved-card quote missing selected card: %s", body)
						}
					} else if string(body["payment_method"]) != fmt.Sprintf("%q", reqPaymentMethod(tt.method)) || body["payment_method_id"] != nil {
						t.Errorf("quote contains unexpected payment routing: %s", body)
					}
					if tt.quoteStatus != 0 {
						w.WriteHeader(tt.quoteStatus)
					}
					fmt.Fprint(w, quoteBody)
					return
				}
				if !strings.Contains(diagnostics.String(), "Capacity quote") || !strings.Contains(diagnostics.String(), "Idempotency key:") {
					t.Error("purchase sent before displaying quote and idempotency key")
				}
				encoded, _ := json.Marshal(body)
				if err := json.Unmarshal(encoded, &purchased); err != nil {
					t.Error(err)
				}
				if purchased.Confirmation != "PURCHASE CAPACITY" {
					t.Errorf("missing generic confirmation: %+v", purchased)
				}
				if tt.method == "saved_card" {
					var quoted struct {
						PaymentConfirmation string      `json:"payment_confirmation"`
						Amount              json.Number `json:"amount_due_after_credit"`
					}
					if err := json.Unmarshal([]byte(quoteBody), &quoted); err != nil {
						t.Error(err)
					}
					if string(body["payment_confirmation"]) != fmt.Sprintf("%q", quoted.PaymentConfirmation) || string(body["approved_charge_amount"]) != quoted.Amount.String() {
						t.Errorf("incorrect backend approval fields: %s", encoded)
					}
				}
				var snapshot map[string]json.RawMessage
				if err := json.Unmarshal(body["expected_quote"], &snapshot); err != nil {
					t.Errorf("missing or invalid expected_quote: %s (%v)", body["expected_quote"], err)
				}
				if len(snapshot) != 5 {
					t.Errorf("expected exactly five snapshot fields, got %s", body["expected_quote"])
				}
				if _, ok := snapshot["existing_pool_id"]; !ok {
					t.Error("expected_quote must include existing_pool_id, including null for a new purchase")
				}
				w.WriteHeader(http.StatusCreated)
				fmt.Fprint(w, checkoutBody)
			})
			req, err := buildPoolCheckoutRequest(42, "annually", tt.method, tt.methodID, "SAVE", tt.key, tt.quoteOnly)
			if err != nil {
				t.Fatal(err)
			}
			err = runPoolCheckout(cmd, client, req, tt.yes, tt.jsonMode)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) || calls != 1 || out.Len() != 0 {
					t.Fatalf("err=%v calls=%d stdout=%q", err, calls, out.String())
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.quoteOnly {
				if calls != 1 || diagnostics.Len() != 0 {
					t.Fatalf("quote calls=%d stderr=%q", calls, diagnostics.String())
				}
			} else {
				if calls != 2 || purchased.QuoteOnly || purchased.PlanID != 42 || purchased.BillingCycle != "annually" || purchased.Promocode != "SAVE" || purchased.PaymentMethod != tt.method || purchased.PaymentMethodID != tt.methodID {
					t.Fatalf("calls=%d purchase=%+v", calls, purchased)
				}
				var expectedQuote api.ResourcePoolExpectedQuote
				if err := json.Unmarshal([]byte(quoteBody), &expectedQuote); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(purchased.ExpectedQuote, &expectedQuote) {
					t.Fatalf("expected_quote = %+v, want exact fresh quote %+v", purchased.ExpectedQuote, expectedQuote)
				}
				if !strings.Contains(diagnostics.String(), "Idempotency key: "+purchased.IdempotencyKey+"\n") {
					t.Fatalf("purchase key not printed to stderr: %q", diagnostics.String())
				}
				if tt.key != "" {
					if purchased.IdempotencyKey != tt.key {
						t.Fatalf("idempotency key = %q", purchased.IdempotencyKey)
					}
				} else if _, err := uuid.Parse(purchased.IdempotencyKey); err != nil {
					t.Fatalf("generated idempotency key = %q", purchased.IdempotencyKey)
				}
				if tt.method == "saved_card" {
					want, err := poolCardConfirmation(tt.methodID, expectedQuote.AmountDueAfterCredit, "4242")
					if err != nil || purchased.PaymentConfirmation != want || purchased.ApprovedChargeAmount != expectedQuote.AmountDueAfterCredit || !strings.Contains(diagnostics.String(), want) {
						t.Fatalf("missing exact saved-card approval: %+v err=%v stderr=%s", purchased, err, diagnostics)
					}
				}
				if tt.method != "saved_card" && tt.quoteBody == "" && !strings.Contains(diagnostics.String(), "$5.0001") {
					t.Fatalf("fresh amount lost precision: %q", diagnostics.String())
				}
			}
			if tt.jsonMode {
				want := checkoutBody
				if tt.quoteOnly {
					want = quoteBody
				}
				var compact bytes.Buffer
				if err := json.Compact(&compact, out.Bytes()); err != nil || compact.String() != want {
					t.Fatalf("raw JSON not preserved: %q (err=%v)", out.String(), err)
				}
			} else if tt.quoteOnly {
				if !strings.Contains(out.String(), "Capacity quote") {
					t.Fatalf("quote output = %q", out.String())
				}
			} else if tt.method == "stripe_checkout" {
				if out.String() != "https://pay.example/checkout\n" {
					t.Fatalf("expected bare checkout URL, got %q", out.String())
				}
			} else if out.Len() != 0 || !strings.Contains(diagnostics.String(), "ORD-1") || !strings.Contains(diagnostics.String(), "INV-1") {
				t.Fatalf("stdout=%q stderr=%q", out.String(), diagnostics.String())
			}
		})
	}
}

func TestPoolCheckoutRetryPreservesIdempotencyKey(t *testing.T) {
	cmd, out, diagnostics := poolTestCommand()
	var purchases []api.ResourcePoolCheckoutRequest
	quotes := 0
	client := poolTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req api.ResourcePoolCheckoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		if req.QuoteOnly {
			quotes++
			fmt.Fprint(w, `{"confirmation":"PURCHASE CAPACITY","mode":"upgrade","existing_pool_id":"pool::existing","unit_price":"7.50","recurring_amount":"30.00","amount_due_after_credit":"5.00","payment_confirmation":"CHARGE SAVED CARD pm::123 ENDING 4242 FOR 5.00"}`)
			return
		}
		purchases = append(purchases, req)
		if len(purchases) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprint(w, `{"detail":"checkout unavailable"}`)
			return
		}
		fmt.Fprint(w, `{"order_number":"ORD-1","invoice_number":"INV-1","payment_method":"saved_card","amount_due":"5.00"}`)
	})
	req, err := buildPoolCheckoutRequest(42, "monthly", "saved_card", "pm::123", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	err = runPoolCheckout(cmd, client, req, true, false)
	if len(purchases) != 1 || err == nil || out.Len() != 0 {
		t.Fatalf("purchases=%+v err=%v stdout=%q", purchases, err, out.String())
	}
	key := purchases[0].IdempotencyKey
	if _, parseErr := uuid.Parse(key); parseErr != nil || !strings.Contains(err.Error(), key) || !strings.Contains(diagnostics.String(), key) {
		t.Fatalf("generated retry key not preserved: key=%q err=%v stderr=%q", key, err, diagnostics.String())
	}
	req.IdempotencyKey = key
	if err := runPoolCheckout(cmd, client, req, true, false); err != nil {
		t.Fatal(err)
	}
	if quotes != 1 || len(purchases) != 2 {
		t.Fatalf("quotes=%d purchases=%d", quotes, len(purchases))
	}
	if purchases[0].ExpectedQuote == nil || !reflect.DeepEqual(purchases[0], purchases[1]) {
		t.Fatalf("retry changed checkout request: first=%+v retry=%+v", purchases[0], purchases[1])
	}
}

func TestPoolCancelWorkflow(t *testing.T) {
	for _, jsonMode := range []bool{false, true} {
		for _, yes := range []bool{false, true} {
			t.Run(fmt.Sprintf("json=%t/yes=%t", jsonMode, yes), func(t *testing.T) {
				cmd, out, _ := poolTestCommand()
				calls := 0
				client := poolTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.Method != http.MethodPost || r.URL.Path != "/client/resource-pools/pool::abc/cancel/" {
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					}
					var req api.ResourcePoolCancelRequest
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !req.Confirm || req.Reason != "Moving" {
						t.Errorf("cancel request=%+v err=%v", req, err)
					}
					fmt.Fprint(w, `{"pool_id":"pool::abc","status":"cancelled","cancelled_members":["ins::1"],"unknown":9007199254740993}`)
				})
				err := runPoolCancel(cmd, client, "pool::abc", "Moving", yes, jsonMode)
				if !yes {
					if err == nil || !strings.Contains(err.Error(), "--yes") || calls != 0 {
						t.Fatalf("unconfirmed cancellation: calls=%d err=%v", calls, err)
					}
					return
				}
				if err != nil || calls != 1 {
					t.Fatalf("calls=%d err=%v", calls, err)
				}
				if jsonMode {
					if !json.Valid(out.Bytes()) || !strings.Contains(out.String(), "9007199254740993") {
						t.Fatalf("raw JSON = %q", out.String())
					}
				} else if !strings.Contains(out.String(), "pool::abc: cancelled") || !strings.Contains(out.String(), "instances: 1") {
					t.Fatalf("cancel output = %q", out.String())
				}
			})
		}
	}
}

func TestPoolExactConfirmation(t *testing.T) {
	for _, input := range []string{"yes\n", "cancel\n", " CANCEL\n", "CANCEL \n", "CANCEL", ""} {
		if err := confirmAction(strings.NewReader(input), &bytes.Buffer{}, false, true, "Cancel?", "CANCEL"); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
	if err := confirmAction(strings.NewReader("CANCEL\r\n"), &bytes.Buffer{}, false, true, "Cancel?", "CANCEL"); err != nil {
		t.Fatal(err)
	}
}

func TestPoolUpdateExplicitFields(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		req, err := buildPoolUpdateRequest("", enabled, true, true)
		if err != nil {
			t.Fatal(err)
		}
		body, err := json.Marshal(req)
		want := fmt.Sprintf(`{"display_name":"","autorenewal_enabled":%t}`, enabled)
		if err != nil || string(body) != want {
			t.Fatalf("body=%s err=%v", body, err)
		}
	}
}

func TestPoolUpdateCobraExplicitAutorenewFalse(t *testing.T) {
	calls := 0
	client := poolTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPatch || r.URL.Path != "/client/resource-pools/pool::abc/" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body) != 1 || string(body["autorenewal_enabled"]) != "false" {
			t.Errorf("request body = %s, want only autorenewal_enabled:false", body)
		}
		fmt.Fprint(w, `{"pool_id":"pool::abc","autorenewal_enabled":false}`)
	})
	config.SetAllowHTTPAPIURL(true)
	t.Cleanup(func() {
		config.SetAPIURLOverride("")
		config.SetAllowHTTPAPIURL(false)
	})
	if err := config.SetAPIURLOverride(client.BaseURL); err != nil {
		t.Fatal(err)
	}

	root, out, _ := poolTestCommand()
	root.Use = "odo"
	pools := &cobra.Command{Use: "pools"}
	pools.AddCommand(newPoolsUpdateCommand())
	root.AddCommand(pools)
	root.SetArgs([]string{"pools", "update", "pool::abc", "--autorenew=false"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !strings.Contains(out.String(), "Autorenew: off") {
		t.Fatalf("requests = %d, output = %q", calls, out.String())
	}
}

func TestPoolSummaryCompatibility(t *testing.T) {
	for _, tt := range []struct {
		name, body, plan string
	}{
		{"legacy nested plan", `{"id":"legacy-id","plan_id":41,"plan":{"name":"Capacity 8G"},"used_ram_mb":1024,"total_ram_mb":8192,"used_disk_gb":20,"total_disk_gb":160,"used_instances":1,"max_instances":4,"used_ips":1,"max_ips":4}`, "Capacity 8G"},
		{"legacy flat plan", `{"id":"legacy-id","plan_id":41,"plan_name":"Capacity 8G","used_ram_mb":1024,"total_ram_mb":8192,"used_disk_gb":20,"total_disk_gb":160,"used_instances":1,"max_instances":4,"used_ips":1,"max_ips":4}`, "Capacity 8G"},
		{"current serializer", `{"pool_id":"pool::abc","plan_id":41,"quota":{"ram_mb":8192,"disk_gb":160,"instances":4,"ips":4},"usage":{"ram_mb":1024,"disk_gb":20,"instances":1,"ips":1},"display_name":"Production","autorenewal_enabled":false}`, "Plan #41"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var pool poolSummary
			if err := json.Unmarshal([]byte(tt.body), &pool); err != nil {
				t.Fatal(err)
			}
			if pool.UsedRAMMB != 1024 || pool.TotalRAMMB != 8192 || pool.UsedDiskGB != 20 || pool.TotalDiskGB != 160 || pool.UsedInstances != 1 || pool.MaxInstances != 4 || pool.UsedIPv4 != 1 || pool.MaxIPv4 != 4 || poolPlanName(pool) != tt.plan {
				t.Fatalf("decoded pool = %+v", pool)
			}
			simple, details := formatPoolsSimple([]poolSummary{pool}), formatPoolsDetails([]poolSummary{pool})
			if !strings.Contains(simple, "1024/8192") || !strings.Contains(details, "1024 / 8192 MB") {
				t.Fatal("quota/usage missing from list/show output")
			}
			if !strings.Contains(simple, tt.plan) || !strings.Contains(details, tt.plan) {
				t.Fatalf("plan missing from list/show output: simple=%q details=%q", simple, details)
			}
			if poolIdentifier(pool) == "-" {
				t.Fatal("pool identifier missing")
			}
		})
	}
}

func reqPaymentMethod(method string) string {
	if method == "" {
		return "stripe_checkout"
	}
	return method
}
