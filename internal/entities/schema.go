// Package entities provides object-oriented interfaces for managing ee entities.
// Each entity type has its own module with storage operations and business logic.
package entities

import (
	"fmt"

	"github.com/n1rna/ee-cli/internal/config"
)

// SchemaManager handles all schema-related operations
type SchemaManager struct {
	storage *baseStorage
}

// NewSchemaManager creates a new schema manager
func NewSchemaManager(cfg *config.Config) (*SchemaManager, error) {
	storage, err := newBaseStorage(cfg, "schemas")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize schema storage: %w", err)
	}

	return &SchemaManager{
		storage: storage,
	}, nil
}

// Create creates a new schema
func (sm *SchemaManager) Create(name, description string, variables []Variable, extends []string) (*Schema, error) {
	// Check if schema with same name already exists
	if _, err := sm.GetByName(name); err == nil {
		return nil, fmt.Errorf("schema with name '%s' already exists", name)
	}

	// Create new schema
	s := NewSchema(name, description, variables, extends)

	// Save schema
	if err := sm.Save(s); err != nil {
		return nil, fmt.Errorf("failed to save schema: %w", err)
	}

	return s, nil
}

// Save saves a schema to storage
func (sm *SchemaManager) Save(s *Schema) error {
	// Save entity file
	if err := sm.storage.saveEntity(s.ID, s); err != nil {
		return fmt.Errorf("failed to save schema file: %w", err)
	}

	// Update index
	if err := sm.storage.updateIndex(s.Entity); err != nil {
		return fmt.Errorf("failed to update schema index: %w", err)
	}

	return nil
}

// GetByID loads a schema by UUID
func (sm *SchemaManager) GetByID(uuid string) (*Schema, error) {
	var s Schema
	if err := sm.storage.loadEntity(uuid, &s); err != nil {
		return nil, fmt.Errorf("failed to load schema %s: %w", uuid, err)
	}
	return &s, nil
}

// GetByName loads a schema by name
func (sm *SchemaManager) GetByName(name string) (*Schema, error) {
	uuid, err := sm.storage.resolveUUID(name)
	if err != nil {
		return nil, fmt.Errorf("schema '%s' not found: %w", name, err)
	}
	return sm.GetByID(uuid)
}

// Get loads a schema by name or UUID
func (sm *SchemaManager) Get(nameOrUUID string) (*Schema, error) {
	// Try UUID first
	if s, err := sm.GetByID(nameOrUUID); err == nil {
		return s, nil
	}
	// Try name
	return sm.GetByName(nameOrUUID)
}

// Delete removes a schema
func (sm *SchemaManager) Delete(nameOrUUID string) error {
	// Resolve to UUID
	uuid, err := sm.storage.resolveUUID(nameOrUUID)
	if err != nil {
		return fmt.Errorf("schema '%s' not found: %w", nameOrUUID, err)
	}

	// Remove entity file
	if err := sm.storage.removeEntity(uuid); err != nil {
		return fmt.Errorf("failed to remove schema file: %w", err)
	}

	// Remove from index
	if err := sm.storage.removeFromIndex(nameOrUUID); err != nil {
		return fmt.Errorf("failed to remove from schema index: %w", err)
	}

	return nil
}

// List returns all schema summaries
func (sm *SchemaManager) List() ([]EntitySummary, error) {
	return sm.storage.listSummaries()
}

// Update updates an existing schema
func (sm *SchemaManager) Update(nameOrUUID string, updater func(*Schema) error) (*Schema, error) {
	// Load existing schema
	s, err := sm.Get(nameOrUUID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if err := updater(s); err != nil {
		return nil, fmt.Errorf("failed to apply updates: %w", err)
	}

	// Save updated schema
	if err := sm.Save(s); err != nil {
		return nil, fmt.Errorf("failed to save updated schema: %w", err)
	}

	return s, nil
}

