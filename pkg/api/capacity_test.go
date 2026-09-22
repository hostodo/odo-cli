package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestListResourcePoolsContract(t *testing.T) {
	injectToken(t)
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/client/resource-pools/" {
			t.Fatalf("request = %s %s, want GET /client/resource-pools/", r.Method, r.URL.Path)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"count": 1,
			"results": []interface{}{map[string]interface{}{
				"pool_id": "pool::abc", "display_name": "Production", "status": "active",
				"plan_id": 41, "billing_amount": "20.00", "billing_cycle": "monthly",
				"autorenewal_enabled": true,
				"quota":               map[string]interface{}{"ram_mb": 8192, "disk_gb": 160, "instances": 4, "ips": 4},
				"usage":               map[string]interface{}{"ram_mb": 1024, "disk_gb": 20, "instances": 1, "ips": 1},
			}},
		})
	}))
	defer srv.Close()

	got, raw, err := newTestClient(t, srv.URL).ListResourcePools()
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 1 || len(got.Results) != 1 || got.Results[0].PoolID != "pool::abc" {
		t.Fatalf("unexpected response: %+v", got)
	}
	if got.Results[0].Quota.RAMMB != 8192 || got.Results[0].Usage.Instances != 1 {
		t.Fatalf("nested capacity fields not decoded: %+v", got.Results[0])
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil || payload["count"].(float64) != 1 {
		t.Fatalf("raw payload not preserved: %s (%v)", raw, err)
	}
}

func TestGetResourcePoolContract(t *testing.T) {
	injectToken(t)
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/client/resource-pools/pool::abc/" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"pool_id": "pool::abc", "members": []interface{}{}})
	}))
	defer srv.Close()

	got, _, err := newTestClient(t, srv.URL).GetResourcePool("pool::abc")
	if err != nil {
		t.Fatal(err)
	}
	if got.PoolID != "pool::abc" {
		t.Fatalf("pool id = %q", got.PoolID)
	}
}

func TestGetResourcePoolOptionsContract(t *testing.T) {
	injectToken(t)
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/client/resource-pools/options/" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"billing_cycles":  []string{"monthly", "annually"},
			"current_pool_id": "pool::abc",
			"tiers": []interface{}{map[string]interface{}{
				"id": 42, "name": "Capacity 16G", "price_monthly": "30.00", "ram_mb": 16384,
				"max_instances": 8, "flag": "upgrade", "self_serve": true,
			}},
		})
	}))
	defer srv.Close()

	got, _, err := newTestClient(t, srv.URL).GetResourcePoolOptions()
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentPoolID != "pool::abc" || got.Tiers[0].Flag != "upgrade" || got.Tiers[0].RAMMB != 16384 {
		t.Fatalf("unexpected options: %+v", got)
	}
}

func TestCheckoutResourcePoolQuoteContract(t *testing.T) {
	injectToken(t)
	var captured map[string]interface{}
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/client/resource-pools/checkout/" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"plan_id": 42, "plan_name": "Capacity 16G", "mode": "upgrade",
			"unit_price": "7.50", "amount_due_after_credit": "5.00", "recurring_amount": "30.00",
		})
	}))
	defer srv.Close()

	req := ResourcePoolCheckoutRequest{PlanID: 42, BillingCycle: "annually", Promocode: "SAVE", QuoteOnly: true}
	got, _, err := newTestClient(t, srv.URL).CheckoutResourcePool(req)
	if err != nil {
		t.Fatal(err)
	}
	wantBody := map[string]interface{}{"plan_id": float64(42), "billing_cycle": "annually", "promocode": "SAVE", "quote_only": true}
	if !reflect.DeepEqual(captured, wantBody) {
		t.Fatalf("body = %#v, want %#v", captured, wantBody)
	}
	if got.Mode != "upgrade" || got.AmountDueAfterCredit.String() != "5.00" {
		t.Fatalf("unexpected quote: %+v", got)
	}
}

func TestCheckoutResourcePoolPurchaseContract(t *testing.T) {
	injectToken(t)
	var captured ResourcePoolCheckoutRequest
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/client/resource-pools/checkout/" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"mode": "purchase", "order_number": "ORD-1", "invoice_number": "INV-1",
			"amount_due": "20.00", "plan_id": 41, "plan_name": "Capacity 8G",
			"checkout_url": "https://pay.example/1", "payment_method": "stripe_checkout",
		})
	}))
	defer srv.Close()

	req := ResourcePoolCheckoutRequest{
		PlanID: 41, BillingCycle: "monthly", PaymentMethod: "saved_card",
		PaymentMethodID: "pm::123", IdempotencyKey: "idem-1", QuoteOnly: false,
		ExpectedQuote: &ResourcePoolExpectedQuote{
			Mode: "purchase", UnitPrice: "20.0000000000000001",
			RecurringAmount: "20.00", AmountDueAfterCredit: "5.0001",
		},
	}
	got, _, err := newTestClient(t, srv.URL).CheckoutResourcePool(req)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(captured, req) {
		t.Fatalf("unexpected request: %+v", captured)
	}
	if got.OrderNumber != "ORD-1" || got.CheckoutURL == "" {
		t.Fatalf("unexpected checkout: %+v", got)
	}
}

func TestResourcePoolExpectedQuoteJSON(t *testing.T) {
	for _, tt := range []struct {
		mode, want string
		poolID     *string
	}{
		{mode: "purchase", want: `{"mode":"purchase","existing_pool_id":null,"unit_price":20.0000000000000001,"recurring_amount":30.00,"amount_due_after_credit":0.00}`},
		{mode: "upgrade", poolID: strPtr("pool::existing"), want: `{"mode":"upgrade","existing_pool_id":"pool::existing","unit_price":20.0000000000000001,"recurring_amount":30.00,"amount_due_after_credit":0.00}`},
	} {
		t.Run(tt.mode, func(t *testing.T) {
			body, err := json.Marshal(ResourcePoolCheckoutRequest{
				PlanID: 42, BillingCycle: "monthly", PaymentMethod: "saved_card", PaymentMethodID: "pm::123",
				ExpectedQuote: &ResourcePoolExpectedQuote{
					Mode: tt.mode, ExistingPoolID: tt.poolID, UnitPrice: "20.0000000000000001",
					RecurringAmount: "30.00", AmountDueAfterCredit: "0.00",
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]json.RawMessage
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatal(err)
			}
			if string(payload["expected_quote"]) != tt.want {
				t.Fatalf("expected_quote = %s, want %s", payload["expected_quote"], tt.want)
			}
		})
	}
}

func TestUpdateResourcePoolContract(t *testing.T) {
	injectToken(t)
	var captured map[string]interface{}
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/client/resource-pools/pool::abc/" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"pool_id": "pool::abc", "display_name": "Edge", "autorenewal_enabled": false})
	}))
	defer srv.Close()

	autorenew := false
	got, _, err := newTestClient(t, srv.URL).UpdateResourcePool("pool::abc", ResourcePoolUpdateRequest{DisplayName: strPtr("Edge"), AutorenewalEnabled: &autorenew})
	if err != nil {
		t.Fatal(err)
	}
	if captured["display_name"] != "Edge" || captured["autorenewal_enabled"] != false || got.DisplayName != "Edge" {
		t.Fatalf("unexpected update: body=%#v response=%+v", captured, got)
	}
}

func TestCancelResourcePoolContract(t *testing.T) {
	injectToken(t)
	var captured ResourcePoolCancelRequest
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/client/resource-pools/pool::abc/cancel/" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"pool_id": "pool::abc", "status": "cancelled", "cancelled_members": []string{"ins::1"}})
	}))
	defer srv.Close()

	got, _, err := newTestClient(t, srv.URL).CancelResourcePool("pool::abc", ResourcePoolCancelRequest{Confirm: true, Reason: "Moving"})
	if err != nil {
		t.Fatal(err)
	}
	if !captured.Confirm || captured.Reason != "Moving" || got.Status != "cancelled" || len(got.CancelledMembers) != 1 {
		t.Fatalf("unexpected cancel: request=%+v response=%+v", captured, got)
	}
}

func strPtr(value string) *string { return &value }
