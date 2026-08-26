package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"time"
)

// ListResourcePools retrieves Hostodo pools for the authenticated user.
func (c *Client) ListResourcePools() ([]ResourcePool, error) {
	resp, err := c.Get("/client/resource-pools/?limit=200")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, body)
	}

	var paginated ResourcePoolsResponse
	if err := json.Unmarshal(body, &paginated); err == nil && paginated.Results != nil {
		return paginated.Results, nil
	}

	var pools []ResourcePool
	if err := json.Unmarshal(body, &pools); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return pools, nil
}

// GetResourcePool retrieves one Hostodo pool, including member VMs.
func (c *Client) GetResourcePool(poolID string) (*ResourcePoolDetail, error) {
	path := fmt.Sprintf("/client/resource-pools/%s/", url.PathEscape(poolID))
	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var pool ResourcePoolDetail
	if err := parseResponse(resp, &pool); err != nil {
		return nil, err
	}
	return &pool, nil
}

// ListPoolOptions retrieves pool tiers and the caller's current pool, if any.
func (c *Client) ListPoolOptions() (*PoolOptionsResponse, error) {
	resp, err := c.Get("/client/resource-pools/options/")
	if err != nil {
		return nil, err
	}

	var options PoolOptionsResponse
	if err := parseResponse(resp, &options); err != nil {
		return nil, err
	}
	return &options, nil
}

// QuotePoolCheckout quotes a pool purchase or upgrade without creating an order.
func (c *Client) QuotePoolCheckout(req PoolCheckoutRequest) (*PoolQuote, error) {
	req.QuoteOnly = true
	resp, err := c.Post("/client/resource-pools/checkout/", req)
	if err != nil {
		return nil, err
	}

	var quote PoolQuote
	if err := parseResponse(resp, &quote); err != nil {
		return nil, err
	}
	return &quote, nil
}

// CheckoutResourcePool buys a Hostodo pool (or upgrades if one already exists).
func (c *Client) CheckoutResourcePool(req PoolCheckoutRequest) (*PoolCheckoutResponse, error) {
	resp, err := c.Post("/client/resource-pools/checkout/", req)
	if err != nil {
		return nil, err
	}

	var result PoolCheckoutResponse
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpgradeResourcePool upgrades an existing Hostodo pool.
func (c *Client) UpgradeResourcePool(poolID string, req PoolCheckoutRequest) (*PoolCheckoutResponse, error) {
	if req.TargetPlanID == 0 {
		req.TargetPlanID = req.PlanID
	}
	path := fmt.Sprintf("/client/resource-pools/%s/upgrade/", url.PathEscape(poolID))
	resp, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var result PoolCheckoutResponse
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	if result.PoolID == "" {
		result.PoolID = poolID
	}
	return &result, nil
}

// CreatePoolVM provisions a $0 gen2 VM inside a Hostodo pool.
// The API creates the VM synchronously, so this uses a long timeout.
func (c *Client) CreatePoolVM(req CreatePoolVMRequest) (*CreatePoolVMResponse, error) {
	resp, err := c.doRequestWithTimeout("POST", "/client/instances/create_in_pool/", req, 10*time.Minute)
	if err != nil {
		return nil, err
	}

	var result CreatePoolVMResponse
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
