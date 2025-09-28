// Package parser provides .ee project configuration file support.
package parser

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/n1rna/ee-cli/internal/entities"
)

// ProjectConfig represents the structure of a .ee project configuration file
type ProjectConfig struct {
	Project      string                           `json:"project"`          // Project UUID or name
	Remote       string                           `json:"remote,omitempty"` // Remote API endpoint
	Schema       ProjectConfigSchema              `json:"schema"`           // Schema definition or reference
	Environments map[string]EnvironmentDefinition `json:"environments"`     // Environment configurations
}

// ProjectConfigSchema defines the schema for the project, either inline or by reference
type ProjectConfigSchema struct {
	Ref       string                       `json:"ref,omitempty"`       // Reference to remote/local schema
	Extends   []string                     `json:"extends,omitempty"`   // Schema inheritance
	Variables map[string]entities.Variable `json:"variables,omitempty"` // Inline variable definitions
}

// EnvironmentDefinition defines how an environment is configured
type EnvironmentDefinition struct {
	Sheet  string        `json:"sheet,omitempty"`  // Single config sheet reference
	Sheets []interface{} `json:"sheets,omitempty"` // Multiple config sheets (mixed types)
}

// LoadProjectConfig loads and parses a .ee file from the current directory
func LoadProjectConfig() (*ProjectConfig, error) {
	return LoadProjectConfigFromPath(".ee")
}

// LoadProjectConfigFromPath loads and parses a .ee file from the specified path
func LoadProjectConfigFromPath(path string) (*ProjectConfig, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf(".ee file not found: %s", path)
	}

	// Read file content
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read .ee file: %w", err)
	}

	// Parse JSON
	var config ProjectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse .ee file: %w", err)
	}

	// Validate required fields
	if config.Project == "" {
		return nil, fmt.Errorf(".ee file missing required 'project' field")
	}

	if len(config.Environments) == 0 {
		return nil, fmt.Errorf(".ee file missing required 'environments' field")
	}

	return &config, nil
}

// SaveProjectConfig saves a project configuration to a .ee file
func SaveProjectConfig(config *ProjectConfig, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write .ee file: %w", err)
	}

	return nil
}

// IsProjectDirectory checks if the current directory contains a .ee file
func IsProjectDirectory() bool {
	_, err := os.Stat(".ee")
	return err == nil
}

// GetEnvironmentNames returns a list of all environment names defined in the project
func (pc *ProjectConfig) GetEnvironmentNames() []string {
	names := make([]string, 0, len(pc.Environments))
	for name := range pc.Environments {
		names = append(names, name)
	}
	return names
}

// HasEnvironment checks if the project has the specified environment
func (pc *ProjectConfig) HasEnvironment(name string) bool {
	_, exists := pc.Environments[name]
	return exists
}

// GetEnvironment returns the environment definition for the specified name
func (pc *ProjectConfig) GetEnvironment(name string) (EnvironmentDefinition, error) {
	env, exists := pc.Environments[name]
	if !exists {
		return EnvironmentDefinition{}, fmt.Errorf("environment '%s' not found", name)
	}
	return env, nil
}
