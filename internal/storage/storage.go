// Package entities provides the base storage functionality for all entity types.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/n1rna/ee-cli/internal/config"
)

const (
	indexFile     = "index.json"
	fileExtension = ".json"
)

// BaseStorage provides common storage operations for all entity types
type BaseStorage struct {
	config     *config.Config
	entityType string // "schemas", "projects", "sheets"
	baseDir    string
	indexPath  string
}

// NewBaseStorage creates a new base storage instance for an entity type
func NewBaseStorage(cfg *config.Config, entityType string) (*BaseStorage, error) {
	baseDir := filepath.Join(cfg.BaseDir, entityType)

	// Ensure directory exists
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", baseDir, err)
	}

	return &BaseStorage{
		config:     cfg,
		entityType: entityType,
		baseDir:    baseDir,
		indexPath:  filepath.Join(baseDir, indexFile),
	}, nil
}

// SaveEntity saves an entity to a JSON file
func (bs *BaseStorage) SaveEntity(uuid string, entity interface{}) error {
	filePath := filepath.Join(bs.baseDir, uuid+fileExtension)

	data, err := json.MarshalIndent(entity, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entity: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write entity file: %w", err)
	}

	return nil
}

// LoadEntity loads an entity from a JSON file
func (bs *BaseStorage) LoadEntity(uuid string, entity interface{}) error {
	filePath := filepath.Join(bs.baseDir, uuid+fileExtension)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read entity file: %w", err)
	}

	if err := json.Unmarshal(data, entity); err != nil {
		return fmt.Errorf("failed to unmarshal entity: %w", err)
	}

	return nil
}

// RemoveEntity removes an entity file
func (bs *BaseStorage) RemoveEntity(uuid string) error {
	filePath := filepath.Join(bs.baseDir, uuid+fileExtension)

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove entity file: %w", err)
	}

	return nil
}

// LoadIndex loads the index for this entity type
func (bs *BaseStorage) LoadIndex() (*Index, error) {
	index := NewIndex()

	data, err := os.ReadFile(bs.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Index doesn't exist yet, return empty index
			return index, nil
		}
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}

	if err := json.Unmarshal(data, index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}

	return index, nil
}

// SaveIndex saves the index for this entity type
func (bs *BaseStorage) SaveIndex(index *Index) error {
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	if err := os.WriteFile(bs.indexPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}

// UpdateIndex adds or updates an entity in the index
func (bs *BaseStorage) UpdateIndex(entity Entity) error {
	index, err := bs.LoadIndex()
	if err != nil {
		return fmt.Errorf("failed to load index: %w", err)
	}

	index.AddEntity(entity)

	if err := bs.SaveIndex(index); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	return nil
}

// RemoveFromIndex removes an entity from the index
func (bs *BaseStorage) RemoveFromIndex(nameOrUUID string) error {
	index, err := bs.LoadIndex()
	if err != nil {
		return fmt.Errorf("failed to load index: %w", err)
	}

	index.RemoveEntity(nameOrUUID)

	if err := bs.SaveIndex(index); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	return nil
}

// ResolveUUID resolves a name or UUID to a UUID using the index
func (bs *BaseStorage) ResolveUUID(nameOrUUID string) (string, error) {
	index, err := bs.LoadIndex()
	if err != nil {
		return "", fmt.Errorf("failed to load index: %w", err)
	}

	uuid, found := index.ResolveUUID(nameOrUUID)
	if !found {
		return "", fmt.Errorf("entity '%s' not found", nameOrUUID)
	}

	return uuid, nil
}

// ListSummaries returns all entity summaries from the index
func (bs *BaseStorage) ListSummaries() ([]EntitySummary, error) {
	index, err := bs.LoadIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	return index.ListSummaries(), nil
}
