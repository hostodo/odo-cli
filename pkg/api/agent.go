package api

import (
	"fmt"
)

// AgentToken represents an agent token for an instance
type AgentToken struct {
	ID           int    `json:"id"`
	InstanceID   string `json:"instance_id"`
	Hostname     string `json:"hostname"`
	Status       string `json:"status"` // "active" or "revoked"
	CreatedAt    string `json:"created_at"`
	LastUsedAt   string `json:"last_used_at"`
	UsageCount   int    `json:"usage_count"`
}

// AgentTokensResponse represents the paginated response for agent tokens
type AgentTokensResponse struct {
	Results []AgentToken `json:"results"`
	Count   int          `json:"count"`
}

// AgentSettings represents account-level agent settings
type AgentSettings struct {
	Enabled      bool   `json:"enabled"`
	UseOwnKey    bool   `json:"use_own_key"`
	MonthlyLimit int    `json:"monthly_limit"`
	TokensUsed   int    `json:"tokens_used"`
}

// GetAgentTokens retrieves all agent tokens for the authenticated user
func (c *Client) GetAgentTokens() (*AgentTokensResponse, error) {
	path := "/client/agent-tokens/"

	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var tokensResp AgentTokensResponse
	if err := parseResponse(resp, &tokensResp); err != nil {
		return nil, err
	}

	return &tokensResp, nil
}

// GetAgentToken retrieves a specific agent token by instance ID
func (c *Client) GetAgentToken(instanceID string) (*AgentToken, error) {
	path := fmt.Sprintf("/client/agent-tokens/%s/", instanceID)

	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var token AgentToken
	if err := parseResponse(resp, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

// GetAgentSettings retrieves account-level agent settings
func (c *Client) GetAgentSettings() (*AgentSettings, error) {
	path := "/client/agent-settings/"

	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var settings AgentSettings
	if err := parseResponse(resp, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// RevokeResponse represents the response from a revoke operation
type RevokeResponse struct {
	Message string `json:"message"`
	Count   int    `json:"count"` // Used for revoke-all
}

// RevokeAgentToken revokes the agent token for a specific instance
func (c *Client) RevokeAgentToken(instanceID string) error {
	path := fmt.Sprintf("/client/agent-tokens/%s/revoke/", instanceID)

	resp, err := c.Post(path, nil)
	if err != nil {
		return err
	}

	if err := parseResponse(resp, nil); err != nil {
		return err
	}

	return nil
}

// RevokeAllAgentTokens revokes agent tokens for all instances
func (c *Client) RevokeAllAgentTokens() (*RevokeResponse, error) {
	path := "/client/agent-tokens/revoke-all/"

	resp, err := c.Post(path, nil)
	if err != nil {
		return nil, err
	}

	var revokeResp RevokeResponse
	if err := parseResponse(resp, &revokeResp); err != nil {
		return nil, err
	}

	return &revokeResp, nil
}

// RegenerateAgentToken regenerates a revoked agent token
func (c *Client) RegenerateAgentToken(instanceID string) (*AgentToken, error) {
	path := fmt.Sprintf("/client/agent-tokens/%s/regenerate/", instanceID)

	resp, err := c.Post(path, nil)
	if err != nil {
		return nil, err
	}

	var token AgentToken
	if err := parseResponse(resp, &token); err != nil {
		return nil, err
	}

	return &token, nil
}
