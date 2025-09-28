// Package manager provides a unified manager for all entity types.
package manager

import (
	"fmt"

	"github.com/n1rna/ee-cli/internal/config"
	"github.com/n1rna/ee-cli/internal/entities"
)

// Manager provides a unified interface to all entity managers
type Manager struct {
	Schemas      *entities.SchemaManager
	ConfigSheets *entities.ConfigSheetManager
	config       *config.Config
}

// NewManager creates a new unified entity manager
func NewManager(cfg *config.Config) (*Manager, error) {
	schemas, err := entities.NewSchemaManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create schema manager: %w", err)
	}

	configSheets, err := entities.NewConfigSheetManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create config sheet manager: %w", err)
	}

	return &Manager{
		Schemas:      schemas,
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
