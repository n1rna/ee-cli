// Package config handles configuration management for ee.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config holds global configuration settings
type Config struct {
	// BaseDir is the root directory for ee storage
	BaseDir string

	// API settings
	API APIConfig
}

// APIConfig holds API-related configuration
type APIConfig struct {
	// Enabled indicates if API integration is enabled
	Enabled bool
	// BaseURL is the API endpoint URL
	BaseURL string
	// APIKey is the authentication key for the API
	APIKey string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		BaseDir: getDefaultBaseDir(),
		API: APIConfig{
			Enabled: false,
			BaseURL: "http://127.0.0.1:8000",
			APIKey:  "",
		},
	}
}

// getDefaultBaseDir returns the default base directory path
func getDefaultBaseDir() string {
	// Check for environment variable first
	if envDir := os.Getenv("EE_HOME"); envDir != "" {
		return envDir
	}

	// Fallback to default location in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// If we can't get the home directory, use current directory
		return ".ee"
	}
	return filepath.Join(homeDir, ".ee")
}

// LoadConfig loads configuration from environment and validates it
func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	// Override with environment variables if present
	if envDir := os.Getenv("EE_HOME"); envDir != "" {
		cfg.BaseDir = envDir
	}

	// Override API settings from environment
	if apiURL := os.Getenv("EE_API_URL"); apiURL != "" {
		cfg.API.BaseURL = apiURL
	}
	if apiKey := os.Getenv("EE_API_KEY"); apiKey != "" {
		cfg.API.APIKey = apiKey
		cfg.API.Enabled = true
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.BaseDir == "" {
		return fmt.Errorf("base directory cannot be empty")
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(c.BaseDir)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	c.BaseDir = absPath

	return nil
}

// EnsureDirectories creates necessary directories if they don't exist
func (c *Config) EnsureDirectories() error {
	dirs := []string{
		c.BaseDir,
		filepath.Join(c.BaseDir, "schemas"),
		filepath.Join(c.BaseDir, "projects"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
