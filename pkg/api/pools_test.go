package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func samplePool() ResourcePool {
	return ResourcePool{
		PoolID:        "pool::abc123",
		DisplayName:   "Lab",
		Status:        "active",
		PlanID:        12,
		BillingAmount: "20.00",
		BillingCycle:  "monthly",
		Quota:         PoolQuota{Instances: 4, VCPU: 4, RAMMB: 8192, DiskGB: 80, BandwidthGB: 8192, IPs: 4, MaxVCPUPerInstance: 2},
		Usage:         PoolQuota{Instances: 1, VCPU: 1, RAMMB: 1024, DiskGB: 20, BandwidthGB: 1024, IPs: 1},
		Remaining:     PoolQuota{Instances: 3, VCPU: 3, RAMMB: 7168, DiskGB: 60, BandwidthGB: 7168, IPs: 3},
	}
}

func TestListResourcePools_Paginated(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/client/resource-pools/") {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, 200, ResourcePoolsResponse{Count: 1, Results: []ResourcePool{samplePool()}})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	pools, err := client.ListResourcePools()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pools) != 1 || pools[0].PoolID != "pool::abc123" {
		t.Fatalf("unexpected pools: %+v", pools)
	}
	if pools[0].Label() != "Lab" {
		t.Errorf("expected label Lab, got %s", pools[0].Label())
	}
}

func TestGetResourcePool_IncludesMembers(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/client/resource-pools/pool::abc123/" {
			http.NotFound(w, r)
			return
		}
		detail := ResourcePoolDetail{
			ResourcePool: samplePool(),
			Members: []ResourcePoolMember{
				{InstanceID: "ins::1", Hostname: "brave-tiger", MainIP: "1.2.3.4", Status: "active", VCPU: 1, RAM: 1024, Disk: 20, Region: "DET01"},
			},
		}
		writeJSON(w, 200, detail)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	pool, err := client.GetResourcePool("pool::abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool.PoolID != "pool::abc123" || len(pool.Members) != 1 || pool.Members[0].Hostname != "brave-tiger" {
		t.Fatalf("unexpected pool: %+v", pool)
	}
}

func TestListPoolOptions(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/client/resource-pools/options/" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, 200, PoolOptionsResponse{
			CurrentPoolID: "pool::abc123",
			BillingCycles: []string{"monthly", "annually"},
			Tiers: []PoolTier{
				{ID: 12, Name: "Hostodo Nano", PriceMonthly: "20.00", RAMMB: 8192, TotalVCPU: 4, Flag: "current"},
				{ID: 13, Name: "Hostodo Micro", PriceMonthly: "40.00", RAMMB: 16384, TotalVCPU: 8, Flag: "upgrade"},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	options, err := client.ListPoolOptions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.CurrentPoolID != "pool::abc123" || len(options.Tiers) != 2 {
		t.Fatalf("unexpected options: %+v", options)
	}
}

func TestQuotePoolCheckout_SendsQuoteOnly(t *testing.T) {
	injectToken(t)

	var captured PoolCheckoutRequest
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/client/resource-pools/checkout/" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&captured)
		writeJSON(w, 200, PoolQuote{PlanID: 12, PlanName: "Hostodo Nano", Mode: "purchase", AmountDueAfterCredit: "20.00"})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	quote, err := client.QuotePoolCheckout(PoolCheckoutRequest{PlanID: 12, BillingCycle: "monthly"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !captured.QuoteOnly {
		t.Fatal("expected quote_only=true")
	}
	if quote.PlanName != "Hostodo Nano" || quote.AmountDueAfterCredit.String() != "20.00" {
		t.Errorf("unexpected quote: %+v", quote)
	}
}

func TestCheckoutResourcePool(t *testing.T) {
	injectToken(t)

	var captured PoolCheckoutRequest
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		writeJSON(w, 201, PoolCheckoutResponse{
			Mode:          "purchase",
			OrderNumber:   "ORD-1",
			InvoiceNumber: "INV-1",
			AmountDue:     "20.00",
			PlanName:      "Hostodo Nano",
			CheckoutURL:   "https://checkout.stripe.com/c/pay/cs_test",
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	result, err := client.CheckoutResourcePool(PoolCheckoutRequest{
		PlanID:         12,
		BillingCycle:   "monthly",
		PaymentMethod:  "saved_card",
		IdempotencyKey: "abc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.QuoteOnly {
		t.Fatal("checkout must not send quote_only")
	}
	if result.OrderNumber != "ORD-1" || result.CheckoutURL == "" {
		t.Errorf("unexpected checkout: %+v", result)
	}
}

func TestUpgradeResourcePool_UsesUpgradePath(t *testing.T) {
	injectToken(t)

	var path string
	var captured PoolCheckoutRequest
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		json.NewDecoder(r.Body).Decode(&captured)
		writeJSON(w, 200, PoolCheckoutResponse{Mode: "upgrade", PlanName: "Hostodo Micro", PoolID: "pool::abc123"})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	result, err := client.UpgradeResourcePool("pool::abc123", PoolCheckoutRequest{PlanID: 13, BillingCycle: "monthly"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/client/resource-pools/pool::abc123/upgrade/" {
		t.Errorf("wrong path: %s", path)
	}
	if captured.TargetPlanID != 13 {
		t.Errorf("expected target_plan_id=13, got %d", captured.TargetPlanID)
	}
	if result.Mode != "upgrade" || result.PoolID != "pool::abc123" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestCreatePoolVM_RejectsPaymentFieldsByNotSendingThem(t *testing.T) {
	injectToken(t)

	var captured map[string]interface{}
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/client/instances/create_in_pool/" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&captured)
		writeJSON(w, 201, CreatePoolVMResponse{})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	_, err := client.CreatePoolVM(CreatePoolVMRequest{
		PoolID:      "pool::abc123",
		Hostname:    "brave-tiger",
		RegionID:    1,
		TemplateID:  2,
		VCPU:        1,
		RAMMB:       1024,
		DiskGB:      20,
		BandwidthGB: 1024,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, banned := range []string{"payment_method", "payment_method_id", "billing_cycle"} {
		if _, ok := captured[banned]; ok {
			t.Errorf("create_in_pool body must not include %s", banned)
		}
	}
	if captured["pool_id"] != "pool::abc123" || captured["hostname"] != "brave-tiger" {
		t.Errorf("unexpected body: %+v", captured)
	}
}

func TestCreatePoolVM_QuotaErrorMessage(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 400, map[string]string{
			"code":    "quota_exceeded",
			"message": "Not enough RAM remaining in this pool",
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	_, err := client.CreatePoolVM(CreatePoolVMRequest{PoolID: "pool::abc123", Hostname: "x", RegionID: 1, TemplateID: 2})
	if err == nil {
		t.Fatal("expected quota error")
	}
	if !strings.Contains(err.Error(), "Not enough RAM remaining in this pool") {
		t.Errorf("expected pool error message, got: %v", err)
	}
}

func TestExtractAPIErrorMessage(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{name: "detail", body: `{"detail":"Promocode not found"}`, want: "Promocode not found"},
		{name: "message", body: `{"code":"quota_exceeded","message":"Not enough RAM"}`, want: "Not enough RAM"},
		{name: "nested", body: `{"detail":{"message":"pool inactive"}}`, want: "pool inactive"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractAPIErrorMessage([]byte(tc.body))
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResourcePoolLabelFallback(t *testing.T) {
	p := ResourcePool{PoolID: "pool::xyz", DisplayName: "  "}
	if p.Label() != "pool::xyz" {
		t.Errorf("expected pool id fallback, got %q", p.Label())
	}
}
