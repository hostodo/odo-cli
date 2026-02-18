package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient creates a Client pointed at a test server with a static token.
func newTestClient(server *httptest.Server) *Client {
	return &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		TokenFunc:  func() (string, error) { return "test-token", nil },
	}
}

// ---------------------------------------------------------------------------
// Instances
// ---------------------------------------------------------------------------

func TestListInstances(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/client/instances/") {
			http.Error(w, "not found", 404)
			return
		}
		json.NewEncoder(w).Encode(InstancesResponse{
			Count: 2,
			Results: []Instance{
				{InstanceID: "i-1", Hostname: "host-1"},
				{InstanceID: "i-2", Hostname: "host-2"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.ListInstances(10, 0)
	if err != nil {
		t.Fatalf("ListInstances: unexpected error: %v", err)
	}
	if resp.Count != 2 {
		t.Errorf("expected count 2, got %d", resp.Count)
	}
	if len(resp.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(resp.Results))
	}
}

func TestListInstances_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(InstancesResponse{Count: 0, Results: []Instance{}})
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.ListInstances(10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Count != 0 || len(resp.Results) != 0 {
		t.Errorf("expected empty response, got count=%d results=%d", resp.Count, len(resp.Results))
	}
}

func TestGetInstance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return the wrapped format that GetInstance tries first
		json.NewEncoder(w).Encode(InstanceDetailResponse{
			Instance: Instance{InstanceID: "abc-123", Hostname: "my-vps"},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	inst, err := client.GetInstance("abc-123")
	if err != nil {
		t.Fatalf("GetInstance: unexpected error: %v", err)
	}
	if inst.InstanceID != "abc-123" {
		t.Errorf("expected instance_id abc-123, got %s", inst.InstanceID)
	}
	if inst.Hostname != "my-vps" {
		t.Errorf("expected hostname my-vps, got %s", inst.Hostname)
	}
}

func TestGetInstance_DirectFormat(t *testing.T) {
	// Simulate a server that returns direct Instance JSON (not wrapped).
	// GetInstance's first parse attempt will produce an empty InstanceID,
	// so it will make a second request. We return direct format both times.
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(Instance{InstanceID: "direct-1", Hostname: "direct-host"})
	}))
	defer server.Close()

	client := newTestClient(server)
	inst, err := client.GetInstance("direct-1")
	if err != nil {
		t.Fatalf("GetInstance direct format: unexpected error: %v", err)
	}
	if inst.InstanceID != "direct-1" {
		t.Errorf("expected instance_id direct-1, got %s", inst.InstanceID)
	}
	if callCount < 2 {
		t.Errorf("expected at least 2 requests for direct format fallback, got %d", callCount)
	}
}

func TestGetInstancePowerStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(InstanceDetailResponse{
			Instance: Instance{InstanceID: "abc", PowerStatus: "running"},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	status, err := client.GetInstancePowerStatus("abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "running" {
		t.Errorf("expected 'running', got %q", status)
	}
}

func TestStartInstance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/start/") {
			t.Errorf("expected path ending with /start/, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	client := newTestClient(server)
	if err := client.StartInstance("abc"); err != nil {
		t.Fatalf("StartInstance: unexpected error: %v", err)
	}
}

func TestStopInstance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	client := newTestClient(server)
	if err := client.StopInstance("abc"); err != nil {
		t.Fatalf("StopInstance: unexpected error: %v", err)
	}
}

func TestRebootInstance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/reboot/") {
			t.Errorf("expected path ending with /reboot/, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	client := newTestClient(server)
	if err := client.RebootInstance("abc"); err != nil {
		t.Fatalf("RebootInstance: unexpected error: %v", err)
	}
}

func TestListInstanceEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(EventsResponse{
			Events: []EventLog{
				{ID: 1, ClientEventMessage: "Provisioning started", Status: "in_progress"},
				{ID: 2, ClientEventMessage: "Provisioning complete", Status: "completed"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	events, err := client.ListInstanceEvents("abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].ClientEventMessage != "Provisioning started" {
		t.Errorf("unexpected first event message: %s", events[0].ClientEventMessage)
	}
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

func TestValidateSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/" {
			http.Error(w, "not found", 404)
			return
		}
		// Verify authorization header
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %q", auth)
		}
		json.NewEncoder(w).Encode(User{ID: 1, Email: "user@example.com", FirstName: "Test", LastName: "User"})
	}))
	defer server.Close()

	client := newTestClient(server)
	user, err := client.ValidateSession()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Email != "user@example.com" {
		t.Errorf("expected email user@example.com, got %s", user.Email)
	}
	if user.ID != 1 {
		t.Errorf("expected id 1, got %d", user.ID)
	}
}

func TestValidateSession_401Expired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		fmt.Fprint(w, `{"detail":"token is expired"}`)
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.ValidateSession()
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestValidateSession_401Revoked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		fmt.Fprint(w, `{"detail":"session has been revoked"}`)
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.ValidateSession()
	if err == nil {
		t.Fatal("expected error for 401 revoked, got nil")
	}
	if err != ErrSessionRevoked {
		t.Errorf("expected ErrSessionRevoked, got %v", err)
	}
}

func TestRevokeSession_204(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(204)
	}))
	defer server.Close()

	client := newTestClient(server)
	if err := client.RevokeSession(); err != nil {
		t.Fatalf("RevokeSession 204: unexpected error: %v", err)
	}
}

func TestRevokeSession_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := newTestClient(server)
	if err := client.RevokeSession(); err != nil {
		t.Fatalf("RevokeSession 404 (already revoked): unexpected error: %v", err)
	}
}

func TestRevokeSession_500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	client := newTestClient(server)
	err := client.RevokeSession()
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error mentioning 500, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Invoices
// ---------------------------------------------------------------------------

func TestListInvoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(InvoicesResponse{
			Count: 1,
			Results: []Invoice{
				{InvoiceNumber: "INV-001", Status: "paid", Subtotal: "9.99"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	invoices, err := client.ListInvoices("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invoices) != 1 {
		t.Fatalf("expected 1 invoice, got %d", len(invoices))
	}
	if invoices[0].InvoiceNumber != "INV-001" {
		t.Errorf("expected INV-001, got %s", invoices[0].InvoiceNumber)
	}
}

func TestListInvoices_FilteredByStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		if status != "unpaid" {
			t.Errorf("expected status=unpaid query param, got %q", status)
		}
		json.NewEncoder(w).Encode(InvoicesResponse{
			Count: 1,
			Results: []Invoice{
				{InvoiceNumber: "INV-002", Status: "unpaid", Subtotal: "19.99"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	invoices, err := client.ListInvoices("unpaid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invoices) != 1 || invoices[0].Status != "unpaid" {
		t.Errorf("expected 1 unpaid invoice, got %+v", invoices)
	}
}

func TestPayInvoice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/INV-001/pay/") {
			t.Errorf("expected path containing /INV-001/pay/, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(PaymentResponse{
			TransactionID: "tx-abc",
			Amount:        "9.99",
			Status:        "completed",
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.PayInvoice("INV-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TransactionID != "tx-abc" {
		t.Errorf("expected tx-abc, got %s", resp.TransactionID)
	}
	if resp.Status != "completed" {
		t.Errorf("expected completed, got %s", resp.Status)
	}
}

// ---------------------------------------------------------------------------
// SSH Keys
// ---------------------------------------------------------------------------

func TestListSSHKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ListSSHKeys unmarshals directly into []SSHKey
		json.NewEncoder(w).Encode([]SSHKey{
			{ID: 1, Name: "my-key", Fingerprint: "SHA256:abc"},
			{ID: 2, Name: "work-key", Fingerprint: "SHA256:def"},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	keys, err := client.ListSSHKeys()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].Name != "my-key" {
		t.Errorf("expected my-key, got %s", keys[0].Name)
	}
}

func TestAddSSHKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		json.Unmarshal(body, &payload)
		if payload["name"] != "test-key" {
			t.Errorf("expected name test-key, got %s", payload["name"])
		}
		if payload["public_key"] != "ssh-ed25519 AAAA..." {
			t.Errorf("expected public_key ssh-ed25519 AAAA..., got %s", payload["public_key"])
		}
		json.NewEncoder(w).Encode(SSHKey{
			ID: 3, Name: "test-key", PublicKey: "ssh-ed25519 AAAA...", Fingerprint: "SHA256:xyz",
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	key, err := client.AddSSHKey("test-key", "ssh-ed25519 AAAA...")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.ID != 3 {
		t.Errorf("expected id 3, got %d", key.ID)
	}
	if key.Fingerprint != "SHA256:xyz" {
		t.Errorf("expected fingerprint SHA256:xyz, got %s", key.Fingerprint)
	}
}

func TestDeleteSSHKey_204(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(204)
	}))
	defer server.Close()

	client := newTestClient(server)
	if err := client.DeleteSSHKey(1); err != nil {
		t.Fatalf("DeleteSSHKey 204: unexpected error: %v", err)
	}
}

func TestDeleteSSHKey_200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := newTestClient(server)
	if err := client.DeleteSSHKey(1); err != nil {
		t.Fatalf("DeleteSSHKey 200: unexpected error: %v", err)
	}
}

func TestDeleteSSHKey_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
	}))
	defer server.Close()

	client := newTestClient(server)
	err := client.DeleteSSHKey(1)
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected error mentioning 403, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Deploy
// ---------------------------------------------------------------------------

func TestListPlans_FiltersDisabledAndOutOfStock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(PlansResponse{
			Results: []Plan{
				{ID: 1, Name: "Plan A", Enabled: true, OutOfStock: false},
				{ID: 2, Name: "Plan B (disabled)", Enabled: false, OutOfStock: false},
				{ID: 3, Name: "Plan C (out of stock)", Enabled: true, OutOfStock: true},
				{ID: 4, Name: "Plan D", Enabled: true, OutOfStock: false},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	plans, err := client.ListPlans()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("expected 2 available plans, got %d", len(plans))
	}
	if plans[0].Name != "Plan A" || plans[1].Name != "Plan D" {
		t.Errorf("unexpected plans: %v", plans)
	}
}

func TestListRegions_FiltersOutOfStock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(RegionsResponse{
			Results: []Region{
				{ID: 1, Name: "US-West", OutOfStock: false},
				{ID: 2, Name: "EU-East", OutOfStock: true},
				{ID: 3, Name: "US-East", OutOfStock: false},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	regions, err := client.ListRegions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(regions) != 2 {
		t.Fatalf("expected 2 available regions, got %d", len(regions))
	}
	if regions[0].Name != "US-West" || regions[1].Name != "US-East" {
		t.Errorf("unexpected regions: %v", regions)
	}
}

func TestListTemplates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(TemplatesResponse{
			Results: []Template{
				{ID: 1, Name: "Ubuntu 22.04", DefaultUsername: "root"},
				{ID: 2, Name: "Debian 12", DefaultUsername: "root"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	templates, err := client.ListTemplates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}
	if templates[0].Name != "Ubuntu 22.04" {
		t.Errorf("expected Ubuntu 22.04, got %s", templates[0].Name)
	}
}

func TestGetQuote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req QuoteRequest
		json.Unmarshal(body, &req)
		if req.Plan != "plan-1" {
			t.Errorf("expected plan plan-1, got %s", req.Plan)
		}
		fmt.Fprint(w, `{"amount_due":"9.99","unit_price":"9.99","quantity":1}`)
	}))
	defer server.Close()

	client := newTestClient(server)
	quote, err := client.GetQuote(QuoteRequest{Plan: "plan-1", BillingCycle: "monthly", Quantity: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quote.AmountDue.String() != "9.99" {
		t.Errorf("expected amount_due 9.99, got %s", quote.AmountDue)
	}
}

func TestCreateDeployOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		fmt.Fprint(w, `{
			"order": {"order_number":"ORD-1","status":"pending","billing_amount":"9.99","hostname":"my-vps"},
			"invoice": {"invoice_number":"INV-1","status":"unpaid","subtotal":"9.99"},
			"checkout_url":"https://pay.example.com"
		}`)
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.CreateDeployOrder(DeployRequest{
		Hostname:     "my-vps",
		Region:       "us-west",
		Template:     "ubuntu-22",
		Plan:         "plan-1",
		BillingCycle: "monthly",
		Quantity:     1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Order.OrderNumber != "ORD-1" {
		t.Errorf("expected order ORD-1, got %s", resp.Order.OrderNumber)
	}
	if resp.CheckoutURL != "https://pay.example.com" {
		t.Errorf("expected checkout URL, got %s", resp.CheckoutURL)
	}
}

func TestCheckHostnameExists_Found(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(InstancesResponse{
			Count: 2,
			Results: []Instance{
				{InstanceID: "i-1", Hostname: "taken-host"},
				{InstanceID: "i-2", Hostname: "other-host"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	exists, err := client.CheckHostnameExists("taken-host")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected hostname to exist")
	}
}

func TestCheckHostnameExists_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(InstancesResponse{
			Count: 1,
			Results: []Instance{
				{InstanceID: "i-1", Hostname: "other-host"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	exists, err := client.CheckHostnameExists("new-host")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected hostname to not exist")
	}
}

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

func TestListCLISessions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/cli-sessions/" {
			http.Error(w, "not found", 404)
			return
		}
		json.NewEncoder(w).Encode(CLISessionsResponse{
			Count: 2,
			Results: []CLISession{
				{ID: 1, DeviceName: "macbook", LoginIP: "1.2.3.4"},
				{ID: 2, DeviceName: "linux-box", LoginIP: "5.6.7.8"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	sessions, err := client.ListCLISessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sessions.Count != 2 {
		t.Errorf("expected count 2, got %d", sessions.Count)
	}
	if len(sessions.Results) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions.Results))
	}
}

func TestRevokeCLISession_204(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(204)
	}))
	defer server.Close()

	client := newTestClient(server)
	if err := client.RevokeCLISession(42); err != nil {
		t.Fatalf("RevokeCLISession 204: unexpected error: %v", err)
	}
}

func TestRevokeCLISession_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := newTestClient(server)
	err := client.RevokeCLISession(42)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if err.Error() != "session not found" {
		t.Errorf("expected 'session not found', got %q", err.Error())
	}
}

func TestRevokeCLISession_403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
	}))
	defer server.Close()

	client := newTestClient(server)
	err := client.RevokeCLISession(42)
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	if err.Error() != "not authorized to revoke this session" {
		t.Errorf("expected 'not authorized to revoke this session', got %q", err.Error())
	}
}

func TestRevokeCLISession_500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	client := newTestClient(server)
	err := client.RevokeCLISession(42)
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error mentioning 500, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Error handling — parseResponse
// ---------------------------------------------------------------------------

func TestParseResponse_JSONError(t *testing.T) {
	resp := &http.Response{
		StatusCode: 400,
		Body:       io.NopCloser(strings.NewReader(`{"detail":"bad request","code":400}`)),
	}
	err := parseResponse(resp, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "bad request") {
		t.Errorf("expected error containing 'bad request', got %v", err)
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected error containing '400', got %v", err)
	}
}

func TestParseResponse_NonJSONError(t *testing.T) {
	resp := &http.Response{
		StatusCode: 502,
		Body:       io.NopCloser(strings.NewReader(`Bad Gateway`)),
	}
	err := parseResponse(resp, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Bad Gateway") {
		t.Errorf("expected error containing 'Bad Gateway', got %v", err)
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("expected error containing '502', got %v", err)
	}
}

func TestParseResponse_Success(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"id":1,"email":"test@example.com"}`)),
	}
	var user User
	err := parseResponse(resp, &user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 1 || user.Email != "test@example.com" {
		t.Errorf("unexpected user: %+v", user)
	}
}

func TestParseResponse_SuccessNilTarget(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
	}
	err := parseResponse(resp, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Error handling — doRequest 401
// ---------------------------------------------------------------------------

func TestDoRequest_401Expired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		fmt.Fprint(w, `{"detail":"token has expired","code":"token_not_valid"}`)
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Get("/any-path")
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestDoRequest_401Revoked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		fmt.Fprint(w, `{"detail":"session has been revoked"}`)
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Get("/any-path")
	if err != ErrSessionRevoked {
		t.Errorf("expected ErrSessionRevoked, got %v", err)
	}
}

func TestDoRequest_TokenFuncError(t *testing.T) {
	client := &Client{
		BaseURL:    "http://localhost",
		HTTPClient: &http.Client{},
		TokenFunc:  func() (string, error) { return "", fmt.Errorf("no token") },
	}
	_, err := client.Get("/any-path")
	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestDoRequest_SetsHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}
		if accept := r.Header.Get("Accept"); accept != "application/json" {
			t.Errorf("expected Accept application/json, got %q", accept)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("expected Authorization Bearer test-token, got %q", auth)
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.Get("/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
}
