package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DeviceCodeResponse represents the response from device authorization
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// TokenResponse represents a successful token exchange
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// OAuthError represents an OAuth error response
type OAuthError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// ErrAuthorizationPending indicates user hasn't approved yet
var ErrAuthorizationPending = fmt.Errorf("authorization_pending")

// ErrAccessDenied indicates user denied authorization
var ErrAccessDenied = fmt.Errorf("access_denied")

// ErrExpiredToken indicates device code expired
var ErrExpiredToken = fmt.Errorf("expired_token")

// ErrSlowDown indicates polling too fast
var ErrSlowDown = fmt.Errorf("slow_down")

// DeviceFlowClient handles OAuth device flow
type DeviceFlowClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewDeviceFlowClient creates a new device flow client
func NewDeviceFlowClient(baseURL string) *DeviceFlowClient {
	return &DeviceFlowClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// InitiateDeviceFlow starts the device authorization flow
func (c *DeviceFlowClient) InitiateDeviceFlow(deviceName, deviceID string, scopes ...string) (*DeviceCodeResponse, error) {
	url := c.BaseURL + "/v1/oauth/device/authorize"

	body := map[string]interface{}{}
	if deviceName != "" {
		body["device_name"] = deviceName
	}
	if deviceID != "" {
		body["device_id"] = deviceID
	}
	if len(scopes) > 0 {
		body["scopes"] = scopes
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "odo-cli")
	req.Header.Set("X-Hostodo-Client", "cli")
	if deviceID != "" {
		req.Header.Set("X-Hostodo-Device-Id", deviceID)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate device flow: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device authorization failed: %s", string(respBody))
	}

	var deviceCode DeviceCodeResponse
	if err := json.Unmarshal(respBody, &deviceCode); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Default interval if not provided
	if deviceCode.Interval == 0 {
		deviceCode.Interval = 5
	}

	return &deviceCode, nil
}

// PollForToken polls the token endpoint until authorization completes
func (c *DeviceFlowClient) PollForToken(ctx context.Context, deviceCode string, interval int) (*TokenResponse, error) {
	url := c.BaseURL + "/v1/oauth/token"

	body := map[string]string{
		"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		"device_code": deviceCode,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "odo-cli")
	req.Header.Set("X-Hostodo-Client", "cli")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to poll for token: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Success case
	if resp.StatusCode == http.StatusOK {
		var tokenResp TokenResponse
		if err := json.Unmarshal(respBody, &tokenResp); err != nil {
			return nil, fmt.Errorf("failed to parse token response: %w", err)
		}
		return &tokenResp, nil
	}

	// Error case - parse OAuth error
	var oauthErr OAuthError
	if err := json.Unmarshal(respBody, &oauthErr); err != nil {
		return nil, fmt.Errorf("unexpected error: %s", string(respBody))
	}

	switch oauthErr.Error {
	case "authorization_pending":
		return nil, ErrAuthorizationPending
	case "access_denied":
		return nil, ErrAccessDenied
	case "expired_token":
		return nil, ErrExpiredToken
	case "slow_down":
		return nil, ErrSlowDown
	default:
		return nil, fmt.Errorf("oauth error: %s - %s", oauthErr.Error, oauthErr.ErrorDescription)
	}
}

// GetDeviceName returns a sanitized hostname for device identification
func GetDeviceName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "Unknown Device"
	}

	// Remove .local suffix (common on macOS)
	hostname = strings.TrimSuffix(hostname, ".local")

	// Limit length for privacy
	if len(hostname) > 32 {
		hostname = hostname[:32]
	}

	return hostname
}
