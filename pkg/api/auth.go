package api

import (
	"fmt"
)

// NOTE: Login now uses OAuth device flow - see pkg/auth/oauth.go
// The old JWT-based login is deprecated and removed

// RevokeSession revokes the current CLI session on the server
// Note: This is a best-effort call - we still clear local token even if server call fails
func (c *Client) RevokeSession() error {
	// Call DELETE /v1/cli-sessions/current/ to revoke server-side session
	// The backend identifies the session from the Bearer token
	resp, err := c.Delete("/v1/cli-sessions/current/")
	if err != nil {
		// Log but don't fail - local cleanup will still happen
		return fmt.Errorf("server revocation failed: %w", err)
	}
	defer resp.Body.Close()

	// 204 No Content is success
	// 404 means session already revoked or expired (also fine)
	if resp.StatusCode != 204 && resp.StatusCode != 404 {
		return fmt.Errorf("server returned error %d", resp.StatusCode)
	}

	return nil
}

// GetCurrentUser retrieves the authenticated user's information
func (c *Client) GetCurrentUser() (*User, error) {
	resp, err := c.Get("/client/user/")
	if err != nil {
		return nil, err
	}

	var user User
	if err := parseResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// ValidateSession checks if the current session is valid
func (c *Client) ValidateSession() (*User, error) {
	resp, err := c.Get("/v1/auth/")
	if err != nil {
		return nil, err
	}

	var user User
	if err := parseResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}
