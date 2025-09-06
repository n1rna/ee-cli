// Package api provides HTTP client for interacting with ee cloud API
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Client represents the API client for ee cloud service
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs HTTP request with authentication
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request: %w", err)
	}

	return resp, nil
}

// parseResponse parses HTTP response into target struct
func (c *Client) parseResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiError APIError
		if json.Unmarshal(body, &apiError) == nil {
			return &apiError
		}
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if target != nil {
		if err := json.Unmarshal(body, target); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// APIError represents an API error response
type APIError struct {
	Detail string `json:"detail"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error: %s", e.Detail)
}

// Health checks API health
func (c *Client) Health() error {
	resp, err := c.doRequest("GET", "/health", nil)
	if err != nil {
		return err
	}

	var healthResp struct {
		Status string `json:"status"`
	}

	return c.parseResponse(resp, &healthResp)
}

// ClientFromRemoteURL creates an API client from a remote URL
// Supports both shorthand format (company@ee.dev/project) and full HTTP URLs
func ClientFromRemoteURL(remoteURL string) (*Client, error) {
	baseURL, err := parseRemoteURL(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote URL: %w", err)
	}

	// Get API key from environment or credential store
	apiKey, err := getAPIKey(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return NewClient(baseURL, apiKey), nil
}

// parseRemoteURL converts various remote URL formats to API base URL
func parseRemoteURL(remoteURL string) (string, error) {
	// Handle shorthand format: company@ee.dev/project -> https://company.ee.dev/api
	if !hasScheme(remoteURL) && contains(remoteURL, "@") {
		parts := splitOnce(remoteURL, "@")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid shorthand format: %s", remoteURL)
		}

		company := parts[0]
		hostAndPath := parts[1]

		// Split host and path if present
		hostParts := splitOnce(hostAndPath, "/")
		host := hostParts[0]

		return fmt.Sprintf("https://%s.%s/api", company, host), nil
	}

	// Handle full HTTP URLs
	if hasScheme(remoteURL) {
		// Return as-is for full HTTP URLs - don't automatically add /api
		// The API endpoints are at root level, not under /api
		if endsWith(remoteURL, "/") {
			return remoteURL[:len(remoteURL)-1], nil // Remove trailing slash
		}
		return remoteURL, nil
	}

	return "", fmt.Errorf("unsupported remote URL format: %s", remoteURL)
}

// getAPIKey retrieves API key from environment variables or credential store
func getAPIKey(baseURL string) (string, error) {
	// Try environment variable first
	if key := getEnv("EE_API_KEY"); key != "" {
		return key, nil
	}

	// Try host-specific environment variable
	// Extract host from baseURL for env var name
	if host := extractHost(baseURL); host != "" {
		envKey := fmt.Sprintf("EE_API_KEY_%s", sanitizeEnvVar(host))
		if key := getEnv(envKey); key != "" {
			return key, nil
		}
	}

	// TODO: Add credential store support (keyring, etc.)
	// For now, return an error asking user to set environment variable
	return "", fmt.Errorf("API key not found. Set EE_API_KEY environment variable or use host-specific EE_API_KEY_%s", sanitizeEnvVar(extractHost(baseURL)))
}

// Helper functions for string manipulation
func hasScheme(url string) bool {
	return len(url) > 7 && (url[:7] == "http://" || url[:8] == "https://")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) != -1
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func splitOnce(s, sep string) []string {
	idx := indexOf(s, sep)
	if idx == -1 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+len(sep):]}
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func getEnv(key string) string {
	return os.Getenv(key)
}

func extractHost(url string) string {
	// Extract hostname from URL for environment variable
	if hasScheme(url) {
		// Remove scheme
		url = url[indexOf(url, "://")+3:]
	}

	// Extract host part (before first slash)
	if idx := indexOf(url, "/"); idx != -1 {
		url = url[:idx]
	}

	// Remove port if present
	if idx := indexOf(url, ":"); idx != -1 {
		url = url[:idx]
	}

	return url
}

func sanitizeEnvVar(host string) string {
	// Convert hostname to valid environment variable suffix
	result := ""
	for i := 0; i < len(host); i++ {
		c := host[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result += string(c)
		} else if c == '.' || c == '-' {
			result += "_"
		}
	}
	// Convert to uppercase
	upper := ""
	for i := 0; i < len(result); i++ {
		c := result[i]
		if c >= 'a' && c <= 'z' {
			upper += string(c - 32)
		} else {
			upper += string(c)
		}
	}
	return upper
}
