package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hostodo/odo-cli/v2/pkg/auth"
	"github.com/hostodo/odo-cli/v2/pkg/config"
)

// newTestClient creates a Client pointed at a test server.
func newTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	cfg := &config.Config{APIURL: serverURL}
	client := &Client{
		BaseURL:    serverURL,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		config:     cfg,
	}
	return client
}

// writeJSON is a helper to write a JSON response in test handlers.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// authMiddleware wraps a handler and checks for a Bearer token.
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") == "" {
			writeJSON(w, 401, ErrorResponse{Detail: "no token"})
			return
		}
		next(w, r)
	}
}

func TestClient_SendsCLIAttributionHeaders(t *testing.T) {
	injectToken(t)

	deviceID := "11111111-1111-1111-1111-111111111111"
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Hostodo-Client") != "cli" {
			t.Errorf("expected X-Hostodo-Client=cli, got %q", r.Header.Get("X-Hostodo-Client"))
		}
		if r.Header.Get("X-Hostodo-Device-Id") != deviceID {
			t.Errorf("expected device id header, got %q", r.Header.Get("X-Hostodo-Device-Id"))
		}
		if !strings.Contains(r.Header.Get("User-Agent"), "odo-cli") {
			t.Errorf("expected odo-cli user agent, got %q", r.Header.Get("User-Agent"))
		}
		writeJSON(w, 200, PlansResponse{Results: []Plan{}})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	client.config.DeviceID = deviceID
	if _, err := client.ListPlans(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// injectToken sets HOME to a temp dir and writes a fake token file so
// auth.GetToken() succeeds without a real OS keychain.
func injectToken(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	// Override home so auth.NewTokenStore picks up our temp dir.
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows
	// Write the token file where auth.TokenStore expects it (~/.odo/token).
	if err := os.MkdirAll(dir+"/.odo", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/.odo/token", []byte("test-bearer-token"), 0600); err != nil {
		t.Fatal(err)
	}
	// Re-initialise the package-level auth store so it reads from the new HOME.
	auth.ResetDefaultStore()
	t.Cleanup(func() { auth.ResetDefaultStore() })
}

// -------------------------------------------------------------------
// GetQuote
// -------------------------------------------------------------------

func TestGetQuote_WithPromo_ReturnsDiscount(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/client/orders/price/" {
			http.NotFound(w, r)
			return
		}
		var req QuoteRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Promocode != "LETCLI" {
			writeJSON(w, 400, ErrorResponse{Detail: "Promocode not found"})
			return
		}
		writeJSON(w, 200, QuoteResponse{
			AmountDue: "3.40",
			UnitPrice: "4.00",
			Quantity:  1,
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	quote, err := client.GetQuote(QuoteRequest{
		PlanID:       23,
		BillingCycle: "monthly",
		Quantity:     1,
		Promocode:    "LETCLI",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quote.AmountDue.String() != "3.40" {
		t.Errorf("expected amount_due=3.40, got %s", quote.AmountDue)
	}
	if quote.UnitPrice.String() != "4.00" {
		t.Errorf("expected unit_price=4.00, got %s", quote.UnitPrice)
	}
}

func TestGetQuote_NoPromo_ReturnsFullPrice(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		var req QuoteRequest
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, 200, QuoteResponse{
			AmountDue: "4.00",
			UnitPrice: "4.00",
			Quantity:  1,
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	quote, err := client.GetQuote(QuoteRequest{
		PlanID:       23,
		BillingCycle: "monthly",
		Quantity:     1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quote.AmountDue.String() != "4.00" {
		t.Errorf("expected 4.00, got %s", quote.AmountDue)
	}
}

func TestGetQuote_InvalidPromo_ReturnsError(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		var req QuoteRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Promocode != "" {
			writeJSON(w, 400, ErrorResponse{Detail: "Promocode not found"})
			return
		}
		writeJSON(w, 200, QuoteResponse{AmountDue: "4.00", UnitPrice: "4.00", Quantity: 1})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	_, err := client.GetQuote(QuoteRequest{
		PlanID:       23,
		BillingCycle: "monthly",
		Quantity:     1,
		Promocode:    "BADCODE",
	})
	if err == nil {
		t.Fatal("expected error for invalid promo, got nil")
	}
	if !strings.Contains(err.Error(), "Promocode not found") {
		t.Errorf("expected 'Promocode not found' in error, got: %v", err)
	}
}

func TestGetQuote_PromoSentInRequestBody(t *testing.T) {
	injectToken(t)

	var capturedBody QuoteRequest
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		writeJSON(w, 200, QuoteResponse{AmountDue: "3.40", UnitPrice: "4.00", Quantity: 1})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	client.GetQuote(QuoteRequest{
		PlanID:       23,
		BillingCycle: "monthly",
		Quantity:     1,
		Promocode:    "LETCLI",
	})

	if capturedBody.Promocode != "LETCLI" {
		t.Errorf("promo not sent in request body: got %q", capturedBody.Promocode)
	}
	if capturedBody.PlanID != 23 {
		t.Errorf("wrong plan_id in body: got %d", capturedBody.PlanID)
	}
}

// -------------------------------------------------------------------
// ListPlans
// -------------------------------------------------------------------

func TestListPlans_FiltersDisabledAndOutOfStock(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, PlansResponse{Results: []Plan{
			{ID: 1, Name: "EPYC-1G", Enabled: true, OutOfStock: false},
			{ID: 2, Name: "EPYC-2G", Enabled: true, OutOfStock: true},   // filtered: out of stock
			{ID: 3, Name: "EPYC-4G", Enabled: false, OutOfStock: false}, // filtered: disabled
			{ID: 4, Name: "EPYC-8G", Enabled: true, OutOfStock: false},
		}})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	plans, err := client.ListPlans()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("expected 2 available plans, got %d", len(plans))
	}
	if plans[0].Name != "EPYC-1G" || plans[1].Name != "EPYC-8G" {
		t.Errorf("wrong plans returned: %+v", plans)
	}
}

func TestListPlans_EmptyResults(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, PlansResponse{Results: []Plan{}})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	plans, err := client.ListPlans()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("expected 0 plans, got %d", len(plans))
	}
}

// -------------------------------------------------------------------
// ListRegions
// -------------------------------------------------------------------

func TestListRegions_FiltersOutOfStock(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, RegionsResponse{Results: []Region{
			{ID: 1, Name: "DET01", OutOfStock: false},
			{ID: 2, Name: "LV01", OutOfStock: true}, // filtered
			{ID: 3, Name: "TPA01", OutOfStock: false},
		}})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	regions, err := client.ListRegions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(regions) != 2 {
		t.Fatalf("expected 2 regions, got %d", len(regions))
	}
	if regions[0].Name != "DET01" || regions[1].Name != "TPA01" {
		t.Errorf("wrong regions: %+v", regions)
	}
}

// -------------------------------------------------------------------
// ListTemplates
// -------------------------------------------------------------------

func TestListTemplates_ReturnsAll(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, TemplatesResponse{Results: []Template{
			{ID: 1, Name: "Ubuntu 22.04", DefaultUsername: "ubuntu"},
			{ID: 2, Name: "Debian 12", DefaultUsername: "debian"},
		}})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	templates, err := client.ListTemplates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}
	if templates[0].Name != "Ubuntu 22.04" {
		t.Errorf("wrong template name: %s", templates[0].Name)
	}
}

// -------------------------------------------------------------------
// CreateDeployOrder
// -------------------------------------------------------------------

func TestCreateDeployOrder_WithPromo_SendsPromoInBody(t *testing.T) {
	injectToken(t)

	var capturedReq DeployRequest
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedReq)
		writeJSON(w, 200, DeployResponse{
			Order: struct {
				OrderNumber   string `json:"order_number"`
				Status        string `json:"status"`
				BillingAmount string `json:"billing_amount"`
				Hostname      string `json:"hostname"`
			}{OrderNumber: "ORD-001", Status: "pending", Hostname: "brave-tiger"},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	req := DeployRequest{
		Hostname:     "brave-tiger",
		Region:       "DET01",
		Template:     "Ubuntu 22.04",
		Plan:         "EPYC-2G1C32GN",
		BillingCycle: "monthly",
		Promocode:    "LETCLI",
		Quantity:     1,
	}
	resp, err := client.CreateDeployOrder(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Order.OrderNumber != "ORD-001" {
		t.Errorf("expected order ORD-001, got %s", resp.Order.OrderNumber)
	}
	if capturedReq.Promocode != "LETCLI" {
		t.Errorf("promo not sent in deploy body: got %q", capturedReq.Promocode)
	}
	if capturedReq.Hostname != "brave-tiger" {
		t.Errorf("wrong hostname in body: %s", capturedReq.Hostname)
	}
}

func TestCreateDeployOrder_NoPromo_OmitsPromoField(t *testing.T) {
	injectToken(t)

	var raw map[string]interface{}
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&raw)
		writeJSON(w, 200, DeployResponse{})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	client.CreateDeployOrder(DeployRequest{
		Hostname:     "quiet-lake",
		Region:       "LV01",
		Template:     "Debian 12",
		Plan:         "EPYC-1G1C16GN",
		BillingCycle: "monthly",
		Quantity:     1,
		// No Promocode
	})

	if _, ok := raw["promocode"]; ok {
		t.Error("promocode field should be omitted when empty (omitempty), but was present in JSON")
	}
}

// -------------------------------------------------------------------
// Auth / token errors
// -------------------------------------------------------------------

func TestClient_401_ReturnsTokenExpiredError(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 401, ErrorResponse{Detail: "token expired"})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	_, err := client.ListPlans()
	if err == nil {
		t.Fatal("expected error on 401, got nil")
	}
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got: %v", err)
	}
}

func TestClient_401_Revoked_ReturnsSessionRevokedError(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 401, ErrorResponse{Detail: "session has been revoked"})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	_, err := client.ListPlans()
	if err != ErrSessionRevoked {
		t.Errorf("expected ErrSessionRevoked, got: %v (type: %T)", err, err)
	}
}

// -------------------------------------------------------------------
// GetDefaultPaymentMethod
// -------------------------------------------------------------------

func TestGetDefaultPaymentMethod_ReturnsDefault(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, PaymentMethodsResponse{Results: []PaymentMethod{
			{PaymentMethodID: "pm_111", LastFour: "1234", CardType: "Visa", CustomerDefault: false},
			{PaymentMethodID: "pm_222", LastFour: "5678", CardType: "Mastercard", CustomerDefault: true},
		}})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	pm, err := client.GetDefaultPaymentMethod()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pm == nil {
		t.Fatal("expected a payment method, got nil")
	}
	if pm.PaymentMethodID != "pm_222" {
		t.Errorf("wrong default payment method: %s", pm.PaymentMethodID)
	}
}

func TestGetDefaultPaymentMethod_NoneDefault_ReturnsNil(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, PaymentMethodsResponse{Results: []PaymentMethod{
			{PaymentMethodID: "pm_111", CustomerDefault: false},
		}})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	pm, err := client.GetDefaultPaymentMethod()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pm != nil {
		t.Errorf("expected nil when no default, got %+v", pm)
	}
}

// -------------------------------------------------------------------
// GetInstance — dual format (wrapped vs direct)
// -------------------------------------------------------------------

func TestGetInstance_WrappedFormat(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, InstanceDetailResponse{
			Instance: Instance{InstanceID: "ins::abc123", Hostname: "brave-tiger", MainIP: "1.2.3.4"},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	inst, err := client.GetInstance("ins::abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.InstanceID != "ins::abc123" {
		t.Errorf("wrong instance ID: %s", inst.InstanceID)
	}
	if inst.MainIP != "1.2.3.4" {
		t.Errorf("wrong IP: %s", inst.MainIP)
	}
}

func TestGetInstance_DirectFormat(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, Instance{
			InstanceID: "ins::xyz789",
			Hostname:   "quiet-lake",
			MainIP:     "5.6.7.8",
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	inst, err := client.GetInstance("ins::xyz789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.InstanceID != "ins::xyz789" {
		t.Errorf("wrong instance ID: %s", inst.InstanceID)
	}
}

// -------------------------------------------------------------------
// Helpdesk tickets
// -------------------------------------------------------------------

func TestListDepartments_ReturnsEnabledDepartments(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/helpdesk/departments/" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, 200, DepartmentsResponse{Results: []Department{
			{ID: 1, Name: "Technical Support", Description: "VPS and server-related issues", Enabled: true},
			{ID: 2, Name: "Billing Support", Description: "Payment and billing inquiries", Enabled: true},
		}})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	departments, err := client.ListDepartments()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(departments) != 2 {
		t.Fatalf("expected 2 departments, got %d", len(departments))
	}
	if departments[0].Name != "Technical Support" {
		t.Errorf("wrong first department: %+v", departments[0])
	}
}

func TestCreateTicket_SendsHelpdeskPayload(t *testing.T) {
	injectToken(t)

	var captured TicketCreateRequest
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/helpdesk/tickets/" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&captured)
		writeJSON(w, 201, Ticket{ID: 123, TicketID: "TCK-123", Subject: captured.Subject, Content: captured.Content, Status: "open", Priority: captured.Priority})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	ticket, err := client.CreateTicket(TicketCreateRequest{Subject: "Need rDNS", Content: "Please set rDNS", DepartmentID: 1, Priority: "medium"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Subject != "Need rDNS" || captured.Content != "Please set rDNS" || captured.DepartmentID != 1 || captured.Priority != "medium" {
		t.Errorf("wrong request body: %+v", captured)
	}
	if ticket.TicketID != "TCK-123" || ticket.Status != "open" {
		t.Errorf("wrong ticket response: %+v", ticket)
	}
}

func TestReplyToTicket_PostsReplyToTicketEndpoint(t *testing.T) {
	injectToken(t)

	var captured TicketReplyRequest
	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/helpdesk/tickets/TCK-123/reply/" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&captured)
		writeJSON(w, 201, TicketReply{ID: 456, Content: captured.Content, InternalNote: captured.InternalNote})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	reply, err := client.ReplyToTicket("TCK-123", TicketReplyRequest{Content: "Thanks", InternalNote: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Content != "Thanks" || captured.InternalNote {
		t.Errorf("wrong reply body: %+v", captured)
	}
	if reply.ID != 456 || reply.Content != "Thanks" {
		t.Errorf("wrong reply response: %+v", reply)
	}
}

func TestGetTicket_UsesEscapedTicketID(t *testing.T) {
	injectToken(t)

	srv := httptest.NewServer(authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/helpdesk/tickets/TCK-123/" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, 200, Ticket{ID: 123, TicketID: "TCK-123", Subject: "Need help", Replies: []TicketReply{{ID: 1, Content: "Reply"}}})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	ticket, err := client.GetTicket("TCK-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.TicketID != "TCK-123" || len(ticket.Replies) != 1 {
		t.Errorf("wrong ticket response: %+v", ticket)
	}
}

// -------------------------------------------------------------------
// config.GetDefaultAPIURL — HTTPS enforcement
// -------------------------------------------------------------------

func TestGetDefaultAPIURL_HTTPSEnvVar_Accepted(t *testing.T) {
	t.Setenv("HOSTODO_API_URL", "https://custom.hostodo.com")
	url := config.GetDefaultAPIURL()
	if url != "https://custom.hostodo.com" {
		t.Errorf("expected custom URL, got %s", url)
	}
}

func TestGetDefaultAPIURL_HTTPEnvVar_Rejected(t *testing.T) {
	t.Setenv("HOSTODO_API_URL", "http://evil.example.com")
	url := config.GetDefaultAPIURL()
	if url != "https://api.hostodo.com" {
		t.Errorf("expected fallback to prod URL, got %s", url)
	}
}

func TestGetDefaultAPIURL_NoEnvVar_ReturnsProd(t *testing.T) {
	t.Setenv("HOSTODO_API_URL", "")
	url := config.GetDefaultAPIURL()
	if url != "https://api.hostodo.com" {
		t.Errorf("expected prod URL, got %s", url)
	}
}

func TestGetDefaultAPIURL_InvalidURL_Rejected(t *testing.T) {
	t.Setenv("HOSTODO_API_URL", "not-a-url")
	url := config.GetDefaultAPIURL()
	if url != "https://api.hostodo.com" {
		t.Errorf("expected fallback to prod URL, got %s", url)
	}
}

func TestGetDefaultAPIURL_MissingHost_Rejected(t *testing.T) {
	t.Setenv("HOSTODO_API_URL", "https://")
	url := config.GetDefaultAPIURL()
	if url != "https://api.hostodo.com" {
		t.Errorf("expected fallback to prod URL, got %s", url)
	}
}
