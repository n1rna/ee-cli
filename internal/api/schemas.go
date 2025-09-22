// Package api provides schema-related API methods
package api

import (
	"fmt"
	"os"
)

// ListSchemas retrieves all schemas from the API
func (c *Client) ListSchemas() ([]Schema, error) {
	resp, err := c.doRequest("GET", "/schemas", nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	var schemas []Schema
	if err := c.parseResponse(resp, &schemas); err != nil {
		return nil, err
	}

	return schemas, nil
}

// GetSchema retrieves a specific schema by GUID
func (c *Client) GetSchema(guid string) (*Schema, error) {
	path := fmt.Sprintf("/schemas/%s", guid)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	var schema Schema
	if err := c.parseResponse(resp, &schema); err != nil {
		return nil, err
	}

	return &schema, nil
}

// CreateSchema creates a new schema
func (c *Client) CreateSchema(schema *Schema) (*Schema, error) {
	resp, err := c.doRequest("POST", "/schemas", schema)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	var createdSchema Schema
	if err := c.parseResponse(resp, &createdSchema); err != nil {
		return nil, err
	}

	return &createdSchema, nil
}

// UpdateSchema updates an existing schema
func (c *Client) UpdateSchema(guid string, schema *Schema) (*Schema, error) {
	path := fmt.Sprintf("/schemas/%s", guid)
	resp, err := c.doRequest("PUT", path, schema)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	var updatedSchema Schema
	if err := c.parseResponse(resp, &updatedSchema); err != nil {
		return nil, err
	}

	return &updatedSchema, nil
}

// DeleteSchema deletes a schema by GUID
func (c *Client) DeleteSchema(guid string) error {
	path := fmt.Sprintf("/schemas/%s", guid)
	resp, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	return c.parseResponse(resp, nil)
}

// PushSchema creates or updates a schema based on whether it exists remotely
func (c *Client) PushSchema(schema *Schema) (*Schema, error) {
	// First try to get existing schema by GUID
	if existingSchema, err := c.GetSchema(schema.GUID); err == nil {
		// Schema exists, update it
		return c.UpdateSchema(existingSchema.GUID, schema)
	}

	// Schema doesn't exist, create it
	return c.CreateSchema(schema)
}
