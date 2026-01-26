package api

import (
	"fmt"
)

// ListCLISessions retrieves all active CLI sessions for the current user
func (c *Client) ListCLISessions() (*CLISessionsResponse, error) {
	resp, err := c.Get("/v1/cli-sessions/")
	if err != nil {
		return nil, err
	}

	var sessions CLISessionsResponse
	if err := parseResponse(resp, &sessions); err != nil {
		return nil, err
	}

	return &sessions, nil
}

// RevokeCLISession revokes a specific CLI session by ID
func (c *Client) RevokeCLISession(sessionID int) error {
	resp, err := c.Delete(fmt.Sprintf("/v1/cli-sessions/%d/", sessionID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("session not found")
	}
	if resp.StatusCode == 403 {
		return fmt.Errorf("not authorized to revoke this session")
	}
	if resp.StatusCode != 204 {
		return fmt.Errorf("failed to revoke session: status %d", resp.StatusCode)
	}

	return nil
}
