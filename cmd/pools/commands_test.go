package pools

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/auth"
	"github.com/hostodo/odo-cli/v2/pkg/config"
)

func samplePool() api.ResourcePool {
	return api.ResourcePool{
		PoolID:        "pool::abc123",
		DisplayName:   "Lab",
		Status:        "active",
		PlanID:        12,
		BillingAmount: "20.00",
		BillingCycle:  "monthly",
		Quota:         api.PoolQuota{Instances: 4, VCPU: 4, RAMMB: 8192, DiskGB: 80, BandwidthGB: 8192, IPs: 4, MaxVCPUPerInstance: 2},
		Usage:         api.PoolQuota{Instances: 1, VCPU: 1, RAMMB: 1024, DiskGB: 20, BandwidthGB: 1024, IPs: 1},
		Remaining:     api.PoolQuota{Instances: 3, VCPU: 3, RAMMB: 7168, DiskGB: 60, BandwidthGB: 7168, IPs: 3},
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func injectToken(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	if err := os.MkdirAll(dir+"/.odo", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/.odo/token", []byte("test-bearer-token"), 0600); err != nil {
		t.Fatal(err)
	}
	auth.ResetDefaultStore()
	t.Cleanup(func() { auth.ResetDefaultStore() })
}

func pointAtServer(t *testing.T, serverURL string) {
	t.Helper()
	config.SetAllowHTTPAPIURL(true)
	if err := config.SetAPIURLOverride(serverURL); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = config.SetAPIURLOverride("")
		config.SetAllowHTTPAPIURL(false)
	})
}

func mockPoolAPI(t *testing.T) *httptest.Server {
	t.Helper()
	pool := samplePool()
	mux := http.NewServeMux()

	mux.HandleFunc("/client/resource-pools/options/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, api.PoolOptionsResponse{
			CurrentPoolID: pool.PoolID,
			BillingCycles: []string{"monthly", "annually"},
			Tiers: []api.PoolTier{
				{ID: 12, Name: "Hostodo Nano", PriceMonthly: "20.00", PriceAnnually: "200.00", RAMMB: 8192, TotalVCPU: 4, DiskGB: 80, MaxInstances: 4, Flag: "current"},
				{ID: 13, Name: "Hostodo Micro", PriceMonthly: "40.00", PriceAnnually: "400.00", RAMMB: 16384, TotalVCPU: 8, DiskGB: 160, MaxInstances: 8, Flag: "upgrade"},
			},
		})
	})
	mux.HandleFunc("/client/resource-pools/checkout/", func(w http.ResponseWriter, r *http.Request) {
		var req api.PoolCheckoutRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.QuoteOnly {
			writeJSON(w, 200, api.PoolQuote{
				PlanID:               req.PlanID,
				PlanName:             "Hostodo Nano",
				BillingCycle:         req.BillingCycle,
				Mode:                 "purchase",
				UnitPrice:            "20.00",
				AmountDueAfterCredit: "20.00",
				RecurringAmount:      "20.00",
			})
			return
		}
		writeJSON(w, 201, api.PoolCheckoutResponse{
			Mode:          "purchase",
			OrderNumber:   "ORD-1",
			InvoiceNumber: "INV-1",
			AmountDue:     "20.00",
			PlanID:        req.PlanID,
			PlanName:      "Hostodo Nano",
		})
	})
	mux.HandleFunc("/client/resource-pools/pool::abc123/upgrade/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, api.PoolCheckoutResponse{
			Mode:      "upgrade",
			PlanName:  "Hostodo Micro",
			PoolID:    pool.PoolID,
			AmountDue: "12.50",
		})
	})
	mux.HandleFunc("/client/resource-pools/pool::abc123/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, api.ResourcePoolDetail{
			ResourcePool: pool,
			Members: []api.ResourcePoolMember{
				{InstanceID: "ins::1", Hostname: "brave-tiger", MainIP: "1.2.3.4", Status: "active", VCPU: 1, RAM: 1024, Disk: 20, Region: "DET01"},
			},
		})
	})
	mux.HandleFunc("/client/resource-pools/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, api.ResourcePoolsResponse{Count: 1, Results: []api.ResourcePool{pool}})
	})
	mux.HandleFunc("/client/templates/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, api.TemplatesResponse{Results: []api.Template{{ID: 2, Name: "Ubuntu 22.04"}}})
	})
	mux.HandleFunc("/client/regions/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, api.RegionsResponse{Results: []api.Region{{ID: 1, Name: "DET01"}}})
	})
	mux.HandleFunc("/client/ssh-keys/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, []api.SSHKey{})
	})
	mux.HandleFunc("/client/instances/create_in_pool/", func(w http.ResponseWriter, r *http.Request) {
		var req api.CreatePoolVMRequest
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, 201, api.CreatePoolVMResponse{
			Instance: struct {
				InstanceID string `json:"instance_id"`
				PoolID     string `json:"pool_id"`
				Status     string `json:"status"`
				Hostname   string `json:"hostname"`
				MainIP     string `json:"main_ip"`
				Bandwidth  int    `json:"bandwidth"`
			}{
				InstanceID: "ins::new",
				PoolID:     req.PoolID,
				Status:     "active",
				Hostname:   req.Hostname,
				MainIP:     "5.6.7.8",
			},
			Quota: struct {
				Used      api.PoolQuota `json:"used"`
				Remaining api.PoolQuota `json:"remaining"`
			}{
				Used:      api.PoolQuota{Instances: 2, VCPU: 2, RAMMB: 2048, DiskGB: 40},
				Remaining: api.PoolQuota{Instances: 2, VCPU: 2, RAMMB: 6144, DiskGB: 40},
			},
		})
	})
	mux.HandleFunc("/v1/billing/payment-methods/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, api.PaymentMethodsResponse{Results: []api.PaymentMethod{{
			PaymentMethodID: "pm_123",
			LastFour:        "4242",
			CardType:        "Visa",
			CustomerDefault: true,
		}}})
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			writeJSON(w, 401, api.ErrorResponse{Detail: "no token"})
			return
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func executePools(t *testing.T, args ...string) (string, error) {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	PoolsCmd.SilenceUsage = true
	PoolsCmd.SilenceErrors = true
	PoolsCmd.SetArgs(args)
	runErr := PoolsCmd.Execute()

	w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String(), runErr
}

func TestPoolsList_JSON(t *testing.T) {
	injectToken(t)
	srv := mockPoolAPI(t)
	pointAtServer(t, srv.URL)

	out, err := executePools(t, "list", "--json")
	if err != nil {
		t.Fatalf("list failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "pool::abc123") || !strings.Contains(out, "Lab") {
		t.Fatalf("expected pool JSON, got:\n%s", out)
	}
}

func TestPoolsList_EmptyJSON(t *testing.T) {
	injectToken(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/client/resource-pools/" || strings.HasPrefix(r.URL.Path, "/client/resource-pools/") {
			writeJSON(w, 200, api.ResourcePoolsResponse{Count: 0, Results: []api.ResourcePool{}})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	pointAtServer(t, srv.URL)

	out, err := executePools(t, "list", "--json")
	if err != nil {
		t.Fatalf("empty list failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "No Hostodo pools found") {
		t.Fatalf("JSON mode should not print the empty-state message, got:\n%s", out)
	}
	var pools []api.ResourcePool
	if err := json.Unmarshal([]byte(out), &pools); err != nil {
		t.Fatalf("expected JSON array, got:\n%s", out)
	}
	if len(pools) != 0 {
		t.Fatalf("expected empty array, got:\n%s", out)
	}
}

func TestPoolsShow_JSONIncludesMembers(t *testing.T) {
	injectToken(t)
	srv := mockPoolAPI(t)
	pointAtServer(t, srv.URL)

	out, err := executePools(t, "show", "pool::abc123", "--json")
	if err != nil {
		t.Fatalf("show failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "brave-tiger") || !strings.Contains(out, "1.2.3.4") {
		t.Fatalf("expected member VM in show output, got:\n%s", out)
	}
}

func TestPoolsOptions_JSON(t *testing.T) {
	injectToken(t)
	srv := mockPoolAPI(t)
	pointAtServer(t, srv.URL)

	out, err := executePools(t, "options", "--json")
	if err != nil {
		t.Fatalf("options failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Hostodo Micro") || !strings.Contains(out, "current_pool_id") {
		t.Fatalf("expected tiers JSON, got:\n%s", out)
	}
}

func TestPoolsBuy_JSON(t *testing.T) {
	injectToken(t)
	srv := mockPoolAPI(t)
	pointAtServer(t, srv.URL)

	out, err := executePools(t, "buy", "--plan", "12", "--billing-cycle", "monthly", "--yes", "--json")
	if err != nil {
		t.Fatalf("buy failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "ORD-1") || !strings.Contains(out, "purchase") {
		t.Fatalf("expected checkout JSON, got:\n%s", out)
	}
}

func TestPoolsUpgrade_JSON(t *testing.T) {
	injectToken(t)
	srv := mockPoolAPI(t)
	pointAtServer(t, srv.URL)

	out, err := executePools(t, "upgrade", "pool::abc123", "--plan", "Hostodo Micro", "--billing-cycle", "monthly", "--yes", "--json")
	if err != nil {
		t.Fatalf("upgrade failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "upgrade") || !strings.Contains(out, "Hostodo Micro") {
		t.Fatalf("expected upgrade JSON, got:\n%s", out)
	}
}

func TestPoolsVM_JSON(t *testing.T) {
	injectToken(t)
	srv := mockPoolAPI(t)
	pointAtServer(t, srv.URL)

	out, err := executePools(t, "vm",
		"--pool", "pool::abc123",
		"--os", "Ubuntu 22.04",
		"--region", "DET01",
		"--vcpu", "1",
		"--ram", "1024",
		"--disk", "20",
		"--bandwidth", "1024",
		"--hostname", "pool-test-box",
		"--yes",
		"--json",
	)
	if err != nil {
		t.Fatalf("vm failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "pool-test-box") || !strings.Contains(out, "5.6.7.8") {
		t.Fatalf("expected created VM JSON, got:\n%s", out)
	}
}

func TestPoolsList_RequiresAuth(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	auth.ResetDefaultStore()
	t.Cleanup(func() { auth.ResetDefaultStore() })
	config.SetAllowHTTPAPIURL(true)
	_ = config.SetAPIURLOverride("http://127.0.0.1:1")
	t.Cleanup(func() {
		_ = config.SetAPIURLOverride("")
		config.SetAllowHTTPAPIURL(false)
	})

	_, err := executePools(t, "list", "--json")
	if err == nil {
		t.Fatal("expected auth error")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("expected not authenticated, got: %v", err)
	}
}

func TestPoolsShow_NotFound(t *testing.T) {
	injectToken(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/client/resource-pools/") && r.Method == http.MethodGet {
			if strings.Contains(r.URL.Path, "missing") {
				writeJSON(w, 404, api.ErrorResponse{Detail: "Not found."})
				return
			}
			writeJSON(w, 200, api.ResourcePoolsResponse{Results: []api.ResourcePool{}})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	pointAtServer(t, srv.URL)

	_, err := executePools(t, "show", "missing")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found, got: %v", err)
	}
}
