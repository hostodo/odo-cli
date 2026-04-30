package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/hostodo/hostodo-cli/pkg/auth"
	"github.com/hostodo/hostodo-cli/pkg/config"
)

// Client represents the API client
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	config     *config.Config
}

// ErrNotAuthenticated indicates user needs to login
var ErrNotAuthenticated = fmt.Errorf("not authenticated - run 'hostodo login'")

// ErrTokenExpired indicates token is invalid and user needs to re-login
var ErrTokenExpired = fmt.Errorf("session expired - run 'hostodo login'")

// ErrSessionRevoked indicates session was revoked and user needs to re-login
var ErrSessionRevoked = fmt.Errorf("session revoked - run 'hostodo login' to authenticate again")

// NewClient creates a new API client
func NewClient(cfg *config.Config) (*Client, error) {
	// Create cookie jar for session management
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}

	client := &Client{
		BaseURL:    cfg.APIURL,
		HTTPClient: httpClient,
		config:     cfg,
	}

	return client, nil
}

// doRequestWithTimeout performs an HTTP request with a custom timeout.
// It temporarily adjusts the HTTP client timeout for this request.
func (c *Client) doRequestWithTimeout(method, path string, body interface{}, timeout time.Duration) (*http.Response, error) {
	origTimeout := c.HTTPClient.Timeout
	c.HTTPClient.Timeout = timeout
	defer func() { c.HTTPClient.Timeout = origTimeout }()
	return c.doRequest(method, path, body)
}

// doRequest performs an HTTP request with token from keychain
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	// Get token from keychain
	token, err := auth.GetToken()
	if err != nil {
		return nil, ErrNotAuthenticated
	}

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := c.BaseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Use Bearer token from keychain
	req.Header.Set("Authorization", "Bearer "+token)

	// Make request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Check for invalid/expired/revoked token (401 Unauthorized)
	if resp.StatusCode == 401 {
		// Try to parse error detail for distinct revoked vs expired messages
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var errResp struct {
			Detail string `json:"detail"`
		}
		if json.Unmarshal(body, &errResp) == nil {
			if strings.Contains(errResp.Detail, "revoked") {
				return nil, ErrSessionRevoked
			}
		}
		return nil, ErrTokenExpired
	}

	return resp, nil
}

// Get performs a GET request
func (c *Client) Get(path string) (*http.Response, error) {
	return c.doRequest("GET", path, nil)
}

// Post performs a POST request
func (c *Client) Post(path string, body interface{}) (*http.Response, error) {
	return c.doRequest("POST", path, body)
}

// Patch performs a PATCH request
func (c *Client) Patch(path string, body interface{}) (*http.Response, error) {
	return c.doRequest("PATCH", path, body)
}

// Delete performs a DELETE request
func (c *Client) Delete(path string) (*http.Response, error) {
	return c.doRequest("DELETE", path, nil)
}

// parseResponse reads and unmarshals the response body
func parseResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, errorResp.Detail)
		}
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	if v != nil {
		if err := json.Unmarshal(body, v); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}
