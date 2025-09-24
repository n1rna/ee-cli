// Package entities provides a unified manager for all entity types.
package entities

import (
	"fmt"

	"github.com/n1rna/ee-cli/internal/config"
)

// Manager provides a unified interface to all entity managers
type Manager struct {
	Schemas      *SchemaManager
	Projects     *ProjectManager
	ConfigSheets *ConfigSheetManager
	config       *config.Config
}

// NewManager creates a new unified entity manager
func NewManager(cfg *config.Config) (*Manager, error) {
	schemas, err := NewSchemaManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create schema manager: %w", err)
	}

	projects, err := NewProjectManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create project manager: %w", err)
	}

	configSheets, err := NewConfigSheetManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create config sheet manager: %w", err)
	}

	return &Manager{
		Schemas:      schemas,
		Projects:     projects,
		ConfigSheets: configSheets,
		config:       cfg,
	}, nil
}

// GetValidator creates a validator that can work with all entity types
func (m *Manager) GetValidator() *Validator {
	return NewValidator(m)
}

// Close closes all managers (if they need cleanup in the future)
func (m *Manager) Close() error {
	// Currently no cleanup needed, but this provides a hook for future use
	return nil
}
