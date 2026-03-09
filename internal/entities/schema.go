// Package entities provides object-oriented interfaces for managing ee entities.
// Each entity type has its own module with storage operations and business logic.
package entities

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

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

// GetByReference loads a schema by reference string.
// Supported formats:
//   - "#/schemas/{uuid}" - JSON Pointer style reference
//   - "local://schema-name" - Local schema by name
//   - "file://path/to/schema.yaml" - Schema file relative to working directory
//   - "./schema.yaml" or "../schema.yaml" - Relative file path
//   - Plain name - Local schema by name
func (sm *SchemaManager) GetByReference(
	schemaRef string,
) (*Schema, error) {
	switch {
	case strings.HasPrefix(schemaRef, "#/schemas/"):
		uuid := strings.TrimPrefix(schemaRef, "#/schemas/")
		return sm.GetByID(uuid)

	case strings.HasPrefix(schemaRef, "local://"):
		schemaName := strings.TrimPrefix(schemaRef, "local://")
		return sm.Get(schemaName)

	case strings.HasPrefix(schemaRef, "file://"):
		path := strings.TrimPrefix(schemaRef, "file://")
		return sm.LoadFromFile(path)

	case strings.HasPrefix(schemaRef, "./"),
		strings.HasPrefix(schemaRef, "../"):
		return sm.LoadFromFile(schemaRef)

	default:
		// Try as local schema name first
		if s, err := sm.Get(schemaRef); err == nil {
			return s, nil
		}
		// Try as file path if it has a known extension
		ext := strings.ToLower(filepath.Ext(schemaRef))
		if ext == ".yaml" || ext == ".yml" || ext == ".json" {
			return sm.LoadFromFile(schemaRef)
		}
		return sm.Get(schemaRef)
	}
}

// LoadFromFile loads a schema from a YAML or JSON file
func (sm *SchemaManager) LoadFromFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read schema file %s: %w", path, err,
		)
	}

	// Try to parse as a schema with variables
	var rawSchema struct {
		Name        string     `json:"name" yaml:"name"`
		Description string     `json:"description" yaml:"description"`
		Variables   []Variable `json:"variables" yaml:"variables"`
		Extends     []string   `json:"extends" yaml:"extends"`
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &rawSchema)
	case ".json":
		err = json.Unmarshal(data, &rawSchema)
	default:
		// Try YAML first, then JSON
		if err = yaml.Unmarshal(data, &rawSchema); err != nil {
			err = json.Unmarshal(data, &rawSchema)
		}
	}
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse schema file %s: %w", path, err,
		)
	}

	name := rawSchema.Name
	if name == "" {
		// Use filename without extension as name
		name = strings.TrimSuffix(
			filepath.Base(path), filepath.Ext(path),
		)
	}

	return &Schema{
		Entity:    storage.NewEntity(name, rawSchema.Description),
		Variables: rawSchema.Variables,
		Extends:   rawSchema.Extends,
	}, nil
}
