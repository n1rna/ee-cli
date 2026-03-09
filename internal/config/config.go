// Package config handles configuration management for ee.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// ProjectConfigFileName is the name of the project configuration file
	ProjectConfigFileName = ".ee"
)

// Config holds global configuration settings
type Config struct {
	// BaseDir is the root directory for ee storage
	BaseDir string

	// ConfigFile is an optional path to the project config file (default: .ee in cwd)
	ConfigFile string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		BaseDir: getDefaultBaseDir(),
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
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
