package api

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// ListInstances retrieves all instances for the authenticated user
func (c *Client) ListInstances(limit, offset int) (*InstancesResponse, error) {
	path := fmt.Sprintf("/client/instances/?limit=%d&offset=%d&sort=updated_at&order=asc", limit, offset)

	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var instancesResp InstancesResponse
	if err := parseResponse(resp, &instancesResp); err != nil {
		return nil, err
	}

	return &instancesResp, nil
}

// GetInstance retrieves details for a specific instance
func (c *Client) GetInstance(instanceID string) (*Instance, error) {
	path := fmt.Sprintf("/client/instances/%s/", instanceID)

	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read body once, try both unmarshal strategies
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Try wrapped format first: {"instance": {...}}
	var wrappedResp InstanceDetailResponse
	if err := json.Unmarshal(body, &wrappedResp); err == nil && wrappedResp.Instance.InstanceID != "" {
		return &wrappedResp.Instance, nil
	}

	// Try direct format: {...}
	var instance Instance
	if err := json.Unmarshal(body, &instance); err != nil {
		return nil, fmt.Errorf("failed to parse instance response: %w", err)
	}

	return &instance, nil
}

// GetInstancePowerStatus retrieves the power status for an instance
func (c *Client) GetInstancePowerStatus(instanceID string) (string, error) {
	path := fmt.Sprintf("/client/instances/%s/power_status/", instanceID)

	resp, err := c.Get(path)
	if err != nil {
		return "", err
	}

	// API returns {"instance": {..., "power_status": "running"}}
	var wrappedResp InstanceDetailResponse
	if err := parseResponse(resp, &wrappedResp); err != nil {
		return "", err
	}

	return wrappedResp.Instance.PowerStatus, nil
}

// StartInstance starts a stopped instance
func (c *Client) StartInstance(instanceID string) error {
	path := fmt.Sprintf("/client/instances/%s/start/", instanceID)
	resp, err := c.Post(path, nil)
	if err != nil {
		return err
	}
	return parseResponse(resp, nil)
}

// StopInstance stops a running instance. If force is true, performs an immediate shutdown.
func (c *Client) StopInstance(instanceID string, force bool) error {
	path := fmt.Sprintf("/client/instances/%s/stop/", instanceID)
	var body interface{}
	if force {
		body = map[string]bool{"force": true}
	}
	resp, err := c.Post(path, body)
	if err != nil {
		return err
	}
	return parseResponse(resp, nil)
}

// ListInstanceEvents retrieves provisioning events for an instance
func (c *Client) ListInstanceEvents(instanceID string) ([]EventLog, error) {
	path := fmt.Sprintf("/client/instances/%s/events/", instanceID)

	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var eventsResp EventsResponse
	if err := parseResponse(resp, &eventsResp); err != nil {
		return nil, err
	}

	return eventsResp.Events, nil
}

// RenameInstance renames an instance by updating its hostname.
func (c *Client) RenameInstance(instanceID, newHostname string) error {
	path := fmt.Sprintf("/client/instances/%s/update_info/", instanceID)
	body := map[string]string{"hostname": newHostname}
	resp, err := c.Patch(path, body)
	if err != nil {
		return err
	}
	return parseResponse(resp, nil)
}

// RebootInstance reboots an instance. If force is true, performs an immediate reboot.
// Uses a longer timeout since the backend performs stop+start synchronously.
func (c *Client) RebootInstance(instanceID string, force bool) error {
	path := fmt.Sprintf("/client/instances/%s/reboot/", instanceID)
	var body interface{}
	if force {
		body = map[string]bool{"force": true}
	}
	resp, err := c.doRequestWithTimeout("POST", path, body, 120*time.Second)
	if err != nil {
		return err
	}
	return parseResponse(resp, nil)
}
