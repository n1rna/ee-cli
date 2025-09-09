package api

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

// ListConfigSheets retrieves all config sheets with optional filtering
func (c *Client) ListConfigSheets(
	projectGUID *string,
	schemaGUID *string,
	activeOnly bool,
) ([]ConfigSheet, error) {
	params := url.Values{}
	if projectGUID != nil {
		params.Add("project_guid", *projectGUID)
	}
	if schemaGUID != nil {
		params.Add("schema_guid", *schemaGUID)
	}
	params.Add("active_only", strconv.FormatBool(activeOnly))

	path := "/config-sheets"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list config sheets: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	var configSheets []ConfigSheet
	if err := c.parseResponse(resp, &configSheets); err != nil {
		return nil, fmt.Errorf("failed to parse config sheets: %w", err)
	}

	return configSheets, nil
}

// GetConfigSheet retrieves a specific config sheet by GUID
func (c *Client) GetConfigSheet(guid string) (*ConfigSheet, error) {
	path := fmt.Sprintf("/config-sheets/%s", guid)

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get config sheet: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	var configSheet ConfigSheet
	if err := c.parseResponse(resp, &configSheet); err != nil {
		return nil, fmt.Errorf("failed to parse config sheet: %w", err)
	}

	return &configSheet, nil
}

// CreateConfigSheet creates a new config sheet
func (c *Client) CreateConfigSheet(configSheet *ConfigSheet) (*ConfigSheet, error) {
	resp, err := c.doRequest("POST", "/config-sheets", configSheet)
	if err != nil {
		return nil, fmt.Errorf("failed to create config sheet: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	var createdConfigSheet ConfigSheet
	if err := c.parseResponse(resp, &createdConfigSheet); err != nil {
		return nil, fmt.Errorf("failed to parse created config sheet: %w", err)
	}

	return &createdConfigSheet, nil
}

// UpdateConfigSheet updates an existing config sheet
func (c *Client) UpdateConfigSheet(
	guid string,
	updates map[string]interface{},
) (*ConfigSheet, error) {
	path := fmt.Sprintf("/config-sheets/%s", guid)

	resp, err := c.doRequest("PUT", path, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update config sheet: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	var updatedConfigSheet ConfigSheet
	if err := c.parseResponse(resp, &updatedConfigSheet); err != nil {
		return nil, fmt.Errorf("failed to parse updated config sheet: %w", err)
	}

	return &updatedConfigSheet, nil
}

// DeleteConfigSheet deletes (deactivates) a config sheet
func (c *Client) DeleteConfigSheet(guid string) error {
	path := fmt.Sprintf("/config-sheets/%s", guid)

	resp, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete config sheet: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	var response map[string]string
	if err := c.parseResponse(resp, &response); err != nil {
		return fmt.Errorf("failed to parse delete response: %w", err)
	}

	return nil
}

// ListConfigSheetsByProject retrieves all config sheets for a specific project
func (c *Client) ListConfigSheetsByProject(
	projectGUID string,
	activeOnly bool,
) ([]ConfigSheet, error) {
	return c.ListConfigSheets(&projectGUID, nil, activeOnly)
}
