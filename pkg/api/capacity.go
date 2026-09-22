package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ResourcePoolCapacity describes a Capacity subscription's quota or usage.
type ResourcePoolCapacity struct {
	Instances          int `json:"instances"`
	VCPU               int `json:"vcpu"`
	RAMMB              int `json:"ram_mb"`
	DiskGB             int `json:"disk_gb"`
	BandwidthGB        int `json:"bandwidth_gb"`
	IPs                int `json:"ips"`
	MaxVCPUPerInstance int `json:"max_vcpu_per_instance,omitempty"`
}

// ResourcePoolMember is a VPS assigned to a Capacity subscription.
type ResourcePoolMember struct {
	InstanceID string `json:"instance_id"`
	Hostname   string `json:"hostname"`
	Status     string `json:"status"`
	MainIP     string `json:"main_ip"`
	VCPU       int    `json:"vcpu"`
	RAMMB      int    `json:"ram"`
	DiskGB     int    `json:"disk"`
	Bandwidth  int    `json:"bandwidth"`
	Region     string `json:"region"`
	PlanName   string `json:"plan_name"`
}

// ResourcePool represents a customer Capacity subscription.
type ResourcePool struct {
	PoolID             string               `json:"pool_id"`
	DisplayName        string               `json:"display_name"`
	Status             string               `json:"status"`
	Enforcement        string               `json:"enforcement"`
	PlanID             int                  `json:"plan_id"`
	BillingAmount      json.Number          `json:"billing_amount"`
	BillingCycle       string               `json:"billing_cycle"`
	NextDueDate        string               `json:"next_due_date"`
	AutorenewalEnabled bool                 `json:"autorenewal_enabled"`
	Quota              ResourcePoolCapacity `json:"quota"`
	Usage              ResourcePoolCapacity `json:"usage"`
	Remaining          ResourcePoolCapacity `json:"remaining"`
	Members            []ResourcePoolMember `json:"members,omitempty"`
	Regions            []ResourcePoolRegion `json:"regions,omitempty"`
	DowngradeBlockers  []interface{}        `json:"downgrade_blockers,omitempty"`
	CreatedAt          string               `json:"created_at"`
	UpdatedAt          string               `json:"updated_at"`
}

// ResourcePoolRegion reports the number of pool members in a region.
type ResourcePoolRegion struct {
	Region string `json:"region"`
	Count  int    `json:"count"`
}

// ResourcePoolsResponse is the paginated Capacity list response.
type ResourcePoolsResponse struct {
	Count    int            `json:"count"`
	Next     *string        `json:"next"`
	Previous *string        `json:"previous"`
	Results  []ResourcePool `json:"results"`
}

// ResourcePoolTier is a purchasable Capacity plan.
type ResourcePoolTier struct {
	ID                 int         `json:"id"`
	Name               string      `json:"name"`
	PriceMonthly       json.Number `json:"price_monthly"`
	PriceAnnually      json.Number `json:"price_annually"`
	PriceSemiannually  json.Number `json:"price_semiannually"`
	PriceBiennially    json.Number `json:"price_biennially"`
	PriceTriennially   json.Number `json:"price_triennially"`
	RAMMB              int         `json:"ram_mb"`
	TotalVCPU          int         `json:"total_vcpu"`
	MaxVCPUPerInstance int         `json:"max_vcpu_per_instance"`
	DiskGB             int         `json:"disk_gb"`
	BandwidthGB        int         `json:"bandwidth_gb"`
	MaxInstances       int         `json:"max_instances"`
	MaxIPs             int         `json:"max_ips"`
	DollarsPerGB       *float64    `json:"dollars_per_gb"`
	SelfServe          bool        `json:"self_serve"`
	Flag               string      `json:"flag"`
}

// ResourcePoolOptions contains billing cycles and available Capacity tiers.
type ResourcePoolOptions struct {
	BillingCycles []string           `json:"billing_cycles"`
	CurrentPoolID string             `json:"current_pool_id"`
	Tiers         []ResourcePoolTier `json:"tiers"`
}

// ResourcePoolCheckoutRequest is used for both a fresh quote and checkout.
type ResourcePoolCheckoutRequest struct {
	PlanID          int                        `json:"plan_id"`
	BillingCycle    string                     `json:"billing_cycle"`
	PaymentMethod   string                     `json:"payment_method,omitempty"`
	PaymentMethodID string                     `json:"payment_method_id,omitempty"`
	Promocode       string                     `json:"promocode,omitempty"`
	IdempotencyKey  string                     `json:"idempotency_key,omitempty"`
	QuoteOnly       bool                       `json:"quote_only"`
	ExpectedQuote   *ResourcePoolExpectedQuote `json:"expected_quote,omitempty"`
}

// ResourcePoolExpectedQuote binds checkout to the fresh quote the customer confirmed.
type ResourcePoolExpectedQuote struct {
	Mode                 string      `json:"mode"`
	ExistingPoolID       *string     `json:"existing_pool_id"`
	UnitPrice            json.Number `json:"unit_price"`
	RecurringAmount      json.Number `json:"recurring_amount"`
	AmountDueAfterCredit json.Number `json:"amount_due_after_credit"`
}

// ResourcePoolCheckoutResponse represents either a quote or completed checkout.
type ResourcePoolCheckoutResponse struct {
	PlanID               int                  `json:"plan_id"`
	PlanName             string               `json:"plan_name"`
	BillingCycle         string               `json:"billing_cycle"`
	Mode                 string               `json:"mode"`
	ExistingPoolID       *string              `json:"existing_pool_id"`
	UnitPrice            json.Number          `json:"unit_price"`
	Subtotal             json.Number          `json:"subtotal"`
	RecurringAmount      json.Number          `json:"recurring_amount"`
	CreditsAvailable     json.Number          `json:"credits_available"`
	CreditsApplied       json.Number          `json:"credits_applied_if_created"`
	AmountDueAfterCredit json.Number          `json:"amount_due_after_credit"`
	PromocodeApplied     bool                 `json:"promocode_applied"`
	InvoiceDate          string               `json:"invoice_date"`
	NextDueDate          string               `json:"next_due_date"`
	Proration            interface{}          `json:"proration"`
	Quota                ResourcePoolCapacity `json:"quota"`
	OrderNumber          string               `json:"order_number"`
	InvoiceNumber        string               `json:"invoice_number"`
	AmountDue            json.Number          `json:"amount_due"`
	CheckoutURL          string               `json:"checkout_url"`
	Checkout             interface{}          `json:"checkout"`
	PaymentMethod        string               `json:"payment_method"`
}

// ResourcePoolUpdateRequest changes mutable Capacity fields. Pointer fields
// distinguish omitted values from an explicit empty name or false autorenewal.
type ResourcePoolUpdateRequest struct {
	DisplayName        *string `json:"display_name,omitempty"`
	AutorenewalEnabled *bool   `json:"autorenewal_enabled,omitempty"`
}

// ResourcePoolCancelRequest confirms cancellation and optionally records why.
type ResourcePoolCancelRequest struct {
	Confirm bool   `json:"confirm"`
	Reason  string `json:"reason,omitempty"`
}

// ResourcePoolCancelResponse reports the cancelled pool and affected instances.
type ResourcePoolCancelResponse struct {
	PoolID           string   `json:"pool_id"`
	Status           string   `json:"status"`
	CancelledMembers []string `json:"cancelled_members"`
}

// parseResponsePreservingRaw delegates status/error handling and decoding to
// parseResponse while retaining the exact successful server payload for --json.
func parseResponsePreservingRaw(resp *http.Response, value interface{}) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	if err := parseResponse(resp, value); err != nil {
		return nil, err
	}
	return body, nil
}

func resourcePoolPath(poolID string) (string, error) {
	poolID = strings.TrimSpace(poolID)
	if poolID == "" {
		return "", fmt.Errorf("pool ID is required")
	}
	return "/client/resource-pools/" + url.PathEscape(poolID) + "/", nil
}

// ListResourcePools lists the authenticated customer's Capacity subscriptions.
func (c *Client) ListResourcePools() (*ResourcePoolsResponse, []byte, error) {
	resp, err := c.Get("/client/resource-pools/")
	if err != nil {
		return nil, nil, err
	}
	var result ResourcePoolsResponse
	raw, err := parseResponsePreservingRaw(resp, &result)
	return &result, raw, err
}

// GetResourcePool retrieves one Capacity subscription by public pool ID.
func (c *Client) GetResourcePool(poolID string) (*ResourcePool, []byte, error) {
	path, err := resourcePoolPath(poolID)
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.Get(path)
	if err != nil {
		return nil, nil, err
	}
	var result ResourcePool
	raw, err := parseResponsePreservingRaw(resp, &result)
	return &result, raw, err
}

// GetResourcePoolOptions retrieves available Capacity tiers and billing cycles.
func (c *Client) GetResourcePoolOptions() (*ResourcePoolOptions, []byte, error) {
	resp, err := c.Get("/client/resource-pools/options/")
	if err != nil {
		return nil, nil, err
	}
	var result ResourcePoolOptions
	raw, err := parseResponsePreservingRaw(resp, &result)
	return &result, raw, err
}

// CheckoutResourcePool requests a fresh quote or creates/upgrades Capacity.
func (c *Client) CheckoutResourcePool(req ResourcePoolCheckoutRequest) (*ResourcePoolCheckoutResponse, []byte, error) {
	resp, err := c.Post("/client/resource-pools/checkout/", req)
	if err != nil {
		return nil, nil, err
	}
	var result ResourcePoolCheckoutResponse
	raw, err := parseResponsePreservingRaw(resp, &result)
	return &result, raw, err
}

// UpdateResourcePool changes a Capacity display name and/or autorenewal state.
func (c *Client) UpdateResourcePool(poolID string, req ResourcePoolUpdateRequest) (*ResourcePool, []byte, error) {
	path, err := resourcePoolPath(poolID)
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.Patch(path, req)
	if err != nil {
		return nil, nil, err
	}
	var result ResourcePool
	raw, err := parseResponsePreservingRaw(resp, &result)
	return &result, raw, err
}

// CancelResourcePool permanently cancels a Capacity subscription.
func (c *Client) CancelResourcePool(poolID string, req ResourcePoolCancelRequest) (*ResourcePoolCancelResponse, []byte, error) {
	path, err := resourcePoolPath(poolID)
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.Post(path+"cancel/", req)
	if err != nil {
		return nil, nil, err
	}
	var result ResourcePoolCancelResponse
	raw, err := parseResponsePreservingRaw(resp, &result)
	return &result, raw, err
}
