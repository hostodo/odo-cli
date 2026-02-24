package api

import "fmt"

// ListSSHKeys retrieves all SSH keys for the authenticated user
func (c *Client) ListSSHKeys() ([]SSHKey, error) {
	path := "/client/ssh-keys/"

	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var sshKeys []SSHKey
	if err := parseResponse(resp, &sshKeys); err != nil {
		return nil, err
	}

	return sshKeys, nil
}

// AddSSHKey adds a new SSH key for the authenticated user
func (c *Client) AddSSHKey(name, publicKey string) (*SSHKey, error) {
	path := "/client/ssh-keys/"

	body := map[string]string{
		"name":       name,
		"public_key": publicKey,
	}

	resp, err := c.Post(path, body)
	if err != nil {
		return nil, err
	}

	var sshKey SSHKey
	if err := parseResponse(resp, &sshKey); err != nil {
		return nil, err
	}

	return &sshKey, nil
}

// DeleteSSHKey deletes an SSH key by ID
func (c *Client) DeleteSSHKey(id int) error {
	path := fmt.Sprintf("/client/ssh-keys/%d/", id)

	resp, err := c.Delete(path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 && resp.StatusCode != 200 {
		return fmt.Errorf("failed to delete SSH key (status %d)", resp.StatusCode)
	}

	return nil
}
