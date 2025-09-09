// Package api provides project-related API methods
package api

import (
	"fmt"
	"os"
)

// ListProjects retrieves all projects from the API
func (c *Client) ListProjects() ([]Project, error) {
	resp, err := c.doRequest("GET", "/projects", nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	var projects []Project
	if err := c.parseResponse(resp, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

// GetProject retrieves a specific project by GUID
func (c *Client) GetProject(guid string) (*Project, error) {
	path := fmt.Sprintf("/projects/%s", guid)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	var project Project
	if err := c.parseResponse(resp, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// CreateProject creates a new project
func (c *Client) CreateProject(project *Project) (*Project, error) {
	resp, err := c.doRequest("POST", "/projects", project)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	var createdProject Project
	if err := c.parseResponse(resp, &createdProject); err != nil {
		return nil, err
	}

	return &createdProject, nil
}

// UpdateProject updates an existing project
func (c *Client) UpdateProject(guid string, project *Project) (*Project, error) {
	path := fmt.Sprintf("/projects/%s", guid)
	resp, err := c.doRequest("PUT", path, project)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	var updatedProject Project
	if err := c.parseResponse(resp, &updatedProject); err != nil {
		return nil, err
	}

	return &updatedProject, nil
}
