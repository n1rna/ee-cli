// Package entities provides object-oriented interfaces for managing ee entities.
package entities

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/n1rna/ee-cli/internal/config"
	"github.com/n1rna/ee-cli/internal/storage"
)

// ConfigSheetManager handles all config sheet-related operations
type ConfigSheetManager struct {
	storage *storage.BaseStorage
}

// NewConfigSheetManager creates a new config sheet manager
func NewConfigSheetManager(cfg *config.Config) (*ConfigSheetManager, error) {
	storage, err := storage.NewBaseStorage(cfg, "sheets")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config sheet storage: %w", err)
	}

	return &ConfigSheetManager{
		storage: storage,
	}, nil
}

// Create creates a new standalone config sheet
func (csm *ConfigSheetManager) Create(
	name, description string,
	schemaRef SchemaReference,
	values map[string]string,
) (*ConfigSheet, error) {
	// Check if config sheet with same name already exists
	if _, err := csm.GetByName(name); err == nil {
		return nil, fmt.Errorf("config sheet with name '%s' already exists", name)
	}

	// Create new config sheet
	cs := NewConfigSheet(name, description, schemaRef, values)

	// Save config sheet
	if err := csm.Save(cs); err != nil {
		return nil, fmt.Errorf("failed to save config sheet: %w", err)
	}

	return cs, nil
}

// Save saves a config sheet to storage
func (csm *ConfigSheetManager) Save(cs *ConfigSheet) error {
	// Save entity file
	if err := csm.storage.SaveEntity(cs.ID, cs); err != nil {
		return fmt.Errorf("failed to save config sheet file: %w", err)
	}

	// Update index
	if err := csm.storage.UpdateIndex(cs.Entity); err != nil {
		return fmt.Errorf("failed to update config sheet index: %w", err)
	}

	return nil
}

// GetByID loads a config sheet by UUID
func (csm *ConfigSheetManager) GetByID(uuid string) (*ConfigSheet, error) {
	var cs ConfigSheet
	if err := csm.storage.LoadEntity(uuid, &cs); err != nil {
		return nil, fmt.Errorf("failed to load config sheet %s: %w", uuid, err)
	}
	return &cs, nil
}

// GetByName loads a config sheet by name
func (csm *ConfigSheetManager) GetByName(name string) (*ConfigSheet, error) {
	uuid, err := csm.storage.ResolveUUID(name)
	if err != nil {
		return nil, fmt.Errorf("config sheet '%s' not found: %w", name, err)
	}
	return csm.GetByID(uuid)
}

// Get loads a config sheet by name or UUID
func (csm *ConfigSheetManager) Get(nameOrUUID string) (*ConfigSheet, error) {
	// Try UUID first
	if cs, err := csm.GetByID(nameOrUUID); err == nil {
		return cs, nil
	}
	// Try name
	return csm.GetByName(nameOrUUID)
}

// Delete removes a config sheet
func (csm *ConfigSheetManager) Delete(nameOrUUID string) error {
	// Resolve to UUID
	uuid, err := csm.storage.ResolveUUID(nameOrUUID)
	if err != nil {
		return fmt.Errorf("config sheet '%s' not found: %w", nameOrUUID, err)
	}

	// Remove entity file
	if err := csm.storage.RemoveEntity(uuid); err != nil {
		return fmt.Errorf("failed to remove config sheet file: %w", err)
	}

	// Remove from index
	if err := csm.storage.RemoveFromIndex(nameOrUUID); err != nil {
		return fmt.Errorf("failed to remove from config sheet index: %w", err)
	}

	return nil
}

// List returns all config sheet summaries
func (csm *ConfigSheetManager) List() ([]storage.EntitySummary, error) {
	return csm.storage.ListSummaries()
}

// ListWithFilters returns config sheet summaries matching the given filters
func (csm *ConfigSheetManager) ListWithFilters(
	filters map[string]string,
) ([]storage.EntitySummary, error) {
	allSummaries, err := csm.List()
	if err != nil {
		return nil, err
	}

	// If no filters, return all
	if len(filters) == 0 {
		return allSummaries, nil
	}

	var filtered []storage.EntitySummary

	for _, summary := range allSummaries {
		// Load full config sheet to check filters
		_, err := csm.GetByID(summary.Name) // Note: summary.Name is actually UUID in the index
		if err != nil {
			continue // Skip if we can't load it
		}

		matches := true

		if matches {
			filtered = append(filtered, summary)
		}
	}

	return filtered, nil
}

// ListStandalone returns only standalone config sheets (not associated with projects)
func (csm *ConfigSheetManager) ListStandalone() ([]storage.EntitySummary, error) {
	return csm.ListWithFilters(map[string]string{"standalone": "true"})
}

// ListByProject returns config sheets associated with a specific project
func (csm *ConfigSheetManager) ListByProject(projectUUID string) ([]storage.EntitySummary, error) {
	return csm.ListWithFilters(map[string]string{"project": projectUUID})
}

// Update updates an existing config sheet
func (csm *ConfigSheetManager) Update(
	nameOrUUID string,
	updater func(*ConfigSheet) error,
) (*ConfigSheet, error) {
	// Load existing config sheet
	cs, err := csm.Get(nameOrUUID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if err := updater(cs); err != nil {
		return nil, fmt.Errorf("failed to apply updates: %w", err)
	}

	// Save updated config sheet
	if err := csm.Save(cs); err != nil {
		return nil, fmt.Errorf("failed to save updated config sheet: %w", err)
	}

	return cs, nil
}

// SetValue sets a variable value in a config sheet
func (csm *ConfigSheetManager) SetValue(nameOrUUID, varName, value string) (*ConfigSheet, error) {
	return csm.Update(nameOrUUID, func(cs *ConfigSheet) error {
		cs.Values[varName] = value
		return nil
	})
}

// UnsetValue removes a variable value from a config sheet
func (csm *ConfigSheetManager) UnsetValue(nameOrUUID, varName string) (*ConfigSheet, error) {
	return csm.Update(nameOrUUID, func(cs *ConfigSheet) error {
		delete(cs.Values, varName)
		return nil
	})
}

// IsFilePath detects if the argument is a file path rather than a name/ID
// Returns true if the argument starts with '.', '/', '~', or contains a file extension
func IsFilePath(arg string) bool {
	// Check if it's a relative path starting with '.' or current directory
	if strings.HasPrefix(arg, ".") {
		return true
	}

	// Check if it's an absolute path starting with '/' or '~'
	if strings.HasPrefix(arg, "/") || strings.HasPrefix(arg, "~") {
		return true
	}

	// Check if it contains a file extension
	if filepath.Ext(arg) != "" {
		return true
	}

	// Check if the file actually exists
	if _, err := os.Stat(arg); err == nil {
		return true
	}

	return false
}

// GetConfigSheetByNameOrID is a helper that gets a config sheet by name or ID
// This is a convenience wrapper around the Get method
func (csm *ConfigSheetManager) GetConfigSheetByNameOrID(nameOrID string) (*ConfigSheet, error) {
	return csm.Get(nameOrID)
}
