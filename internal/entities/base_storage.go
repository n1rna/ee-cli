// Package entities provides the base storage functionality for all entity types.
package entities

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

// baseStorage provides common storage operations for all entity types
type baseStorage struct {
	config     *config.Config
	entityType string // "schemas", "projects", "sheets"
	baseDir    string
	indexPath  string
}

// newBaseStorage creates a new base storage instance for an entity type
func newBaseStorage(cfg *config.Config, entityType string) (*baseStorage, error) {
	baseDir := filepath.Join(cfg.BaseDir, entityType)

	// Ensure directory exists
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", baseDir, err)
	}

	return &baseStorage{
		config:     cfg,
		entityType: entityType,
		baseDir:    baseDir,
		indexPath:  filepath.Join(baseDir, indexFile),
	}, nil
}

// saveEntity saves an entity to a JSON file
func (bs *baseStorage) saveEntity(uuid string, entity interface{}) error {
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

// loadEntity loads an entity from a JSON file
func (bs *baseStorage) loadEntity(uuid string, entity interface{}) error {
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

// removeEntity removes an entity file
func (bs *baseStorage) removeEntity(uuid string) error {
	filePath := filepath.Join(bs.baseDir, uuid+fileExtension)

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove entity file: %w", err)
	}

	return nil
}

// loadIndex loads the index for this entity type
func (bs *baseStorage) loadIndex() (*Index, error) {
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

// saveIndex saves the index for this entity type
func (bs *baseStorage) saveIndex(index *Index) error {
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	if err := os.WriteFile(bs.indexPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}

// updateIndex adds or updates an entity in the index
func (bs *baseStorage) updateIndex(entity Entity) error {
	index, err := bs.loadIndex()
	if err != nil {
		return fmt.Errorf("failed to load index: %w", err)
	}

	index.AddEntity(entity)

	if err := bs.saveIndex(index); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	return nil
}

// removeFromIndex removes an entity from the index
func (bs *baseStorage) removeFromIndex(nameOrUUID string) error {
	index, err := bs.loadIndex()
	if err != nil {
		return fmt.Errorf("failed to load index: %w", err)
	}

	index.RemoveEntity(nameOrUUID)

	if err := bs.saveIndex(index); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	return nil
}

// resolveUUID resolves a name or UUID to a UUID using the index
func (bs *baseStorage) resolveUUID(nameOrUUID string) (string, error) {
	index, err := bs.loadIndex()
	if err != nil {
		return "", fmt.Errorf("failed to load index: %w", err)
	}

	uuid, found := index.ResolveUUID(nameOrUUID)
	if !found {
		return "", fmt.Errorf("entity '%s' not found", nameOrUUID)
	}

	return uuid, nil
}

// listSummaries returns all entity summaries from the index
func (bs *baseStorage) listSummaries() ([]EntitySummary, error) {
	index, err := bs.loadIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	return index.ListSummaries(), nil
}
