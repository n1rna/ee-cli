// Package entities provides object-oriented interfaces for managing ee entities.
// Each entity type has its own module with storage operations and business logic.
package entities

import (
	"fmt"
	"strings"

	"github.com/n1rna/ee-cli/internal/config"
	"github.com/n1rna/ee-cli/internal/storage"
)

// SchemaManager handles all schema-related operations
type SchemaManager struct {
	storage *storage.BaseStorage
}

// NewSchemaManager creates a new schema manager
func NewSchemaManager(cfg *config.Config) (*SchemaManager, error) {
	storage, err := storage.NewBaseStorage(cfg, "schemas")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize schema storage: %w", err)
	}

	return &SchemaManager{
		storage: storage,
	}, nil
}

// Create creates a new schema
func (sm *SchemaManager) Create(
	name, description string,
	variables []Variable,
	extends []string,
) (*Schema, error) {
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
	if err := sm.storage.SaveEntity(s.ID, s); err != nil {
		return fmt.Errorf("failed to save schema file: %w", err)
	}

	// Update index
	if err := sm.storage.UpdateIndex(s.Entity); err != nil {
		return fmt.Errorf("failed to update schema index: %w", err)
	}

	return nil
}

// GetByID loads a schema by UUID
func (sm *SchemaManager) GetByID(uuid string) (*Schema, error) {
	var s Schema
	if err := sm.storage.LoadEntity(uuid, &s); err != nil {
		return nil, fmt.Errorf("failed to load schema %s: %w", uuid, err)
	}
	return &s, nil
}

// GetByName loads a schema by name
func (sm *SchemaManager) GetByName(name string) (*Schema, error) {
	uuid, err := sm.storage.ResolveUUID(name)
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
	uuid, err := sm.storage.ResolveUUID(nameOrUUID)
	if err != nil {
		return fmt.Errorf("schema '%s' not found: %w", nameOrUUID, err)
	}

	// Remove entity file
	if err := sm.storage.RemoveEntity(uuid); err != nil {
		return fmt.Errorf("failed to remove schema file: %w", err)
	}

	// Remove from index
	if err := sm.storage.RemoveFromIndex(nameOrUUID); err != nil {
		return fmt.Errorf("failed to remove from schema index: %w", err)
	}

	return nil
}

// List returns all schema summaries
func (sm *SchemaManager) List() ([]storage.EntitySummary, error) {
	return sm.storage.ListSummaries()
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

// GetByReference loads a schema by reference, handling local:// and remote:// prefixes
func (sm *SchemaManager) GetByReference(schemaRef string) (*Schema, error) {
	switch {
	case strings.HasPrefix(schemaRef, "local://"):
		// Local schema reference: local://schema-name
		schemaName := strings.TrimPrefix(schemaRef, "local://")
		return sm.Get(schemaName)

	case strings.HasPrefix(schemaRef, "remote://"):
		// Remote schema reference: not yet implemented
		return nil, fmt.Errorf("remote schema references not yet implemented: %s", schemaRef)

	default:
		// Assume it's a local schema name without prefix
		return sm.Get(schemaRef)
	}
}
