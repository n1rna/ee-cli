// Package entities provides object-oriented interfaces for managing ee entities.
package entities

import (
	"fmt"

	"github.com/n1rna/ee-cli/internal/config"
)

// ConfigSheetManager handles all config sheet-related operations
type ConfigSheetManager struct {
	storage *baseStorage
}

// NewConfigSheetManager creates a new config sheet manager
func NewConfigSheetManager(cfg *config.Config) (*ConfigSheetManager, error) {
	storage, err := newBaseStorage(cfg, "sheets")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config sheet storage: %w", err)
	}

	return &ConfigSheetManager{
		storage: storage,
	}, nil
}

// Create creates a new standalone config sheet
func (csm *ConfigSheetManager) Create(name, description string, schemaRef SchemaReference, values map[string]string) (*ConfigSheet, error) {
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

// CreateForProject creates a new config sheet for a project environment
func (csm *ConfigSheetManager) CreateForProject(name, description string, schemaRef SchemaReference, projectUUID, envName string, values map[string]string) (*ConfigSheet, error) {
	// Check if config sheet with same name already exists
	if _, err := csm.GetByName(name); err == nil {
		return nil, fmt.Errorf("config sheet with name '%s' already exists", name)
	}

	// Create new config sheet for project
	cs := NewConfigSheetForProject(name, description, schemaRef, projectUUID, envName, values)


	// Save config sheet
	if err := csm.Save(cs); err != nil {
		return nil, fmt.Errorf("failed to save config sheet: %w", err)
	}

	return cs, nil
}

// Save saves a config sheet to storage
func (csm *ConfigSheetManager) Save(cs *ConfigSheet) error {
	// Save entity file
	if err := csm.storage.saveEntity(cs.ID, cs); err != nil {
		return fmt.Errorf("failed to save config sheet file: %w", err)
	}

	// Update index
	if err := csm.storage.updateIndex(cs.Entity); err != nil {
		return fmt.Errorf("failed to update config sheet index: %w", err)
	}

	return nil
}

// GetByID loads a config sheet by UUID
func (csm *ConfigSheetManager) GetByID(uuid string) (*ConfigSheet, error) {
	var cs ConfigSheet
	if err := csm.storage.loadEntity(uuid, &cs); err != nil {
		return nil, fmt.Errorf("failed to load config sheet %s: %w", uuid, err)
	}
	return &cs, nil
}

// GetByName loads a config sheet by name
func (csm *ConfigSheetManager) GetByName(name string) (*ConfigSheet, error) {
	uuid, err := csm.storage.resolveUUID(name)
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
	uuid, err := csm.storage.resolveUUID(nameOrUUID)
	if err != nil {
		return fmt.Errorf("config sheet '%s' not found: %w", nameOrUUID, err)
	}

	// Remove entity file
	if err := csm.storage.removeEntity(uuid); err != nil {
		return fmt.Errorf("failed to remove config sheet file: %w", err)
	}

	// Remove from index
	if err := csm.storage.removeFromIndex(nameOrUUID); err != nil {
		return fmt.Errorf("failed to remove from config sheet index: %w", err)
	}

	return nil
}

// List returns all config sheet summaries
func (csm *ConfigSheetManager) List() ([]EntitySummary, error) {
	return csm.storage.listSummaries()
}

// ListWithFilters returns config sheet summaries matching the given filters
func (csm *ConfigSheetManager) ListWithFilters(filters map[string]string) ([]EntitySummary, error) {
	allSummaries, err := csm.List()
	if err != nil {
		return nil, err
	}

	// If no filters, return all
	if len(filters) == 0 {
		return allSummaries, nil
	}

	var filtered []EntitySummary

	for _, summary := range allSummaries {
		// Load full config sheet to check filters
		cs, err := csm.GetByID(summary.Name) // Note: summary.Name is actually UUID in the index
		if err != nil {
			continue // Skip if we can't load it
		}

		matches := true

		// Check project filter
		if projectFilter, ok := filters["project"]; ok {
			if cs.Project != projectFilter {
				matches = false
			}
		}

		// Check environment filter
		if envFilter, ok := filters["environment"]; ok {
			if cs.Environment != envFilter {
				matches = false
			}
		}

		// Check standalone filter
		if standaloneFilter, ok := filters["standalone"]; ok {
			isStandalone := cs.IsStandalone()
			if (standaloneFilter == "true" && !isStandalone) || (standaloneFilter == "false" && isStandalone) {
				matches = false
			}
		}

		if matches {
			filtered = append(filtered, summary)
		}
	}

	return filtered, nil
}

// ListStandalone returns only standalone config sheets (not associated with projects)
func (csm *ConfigSheetManager) ListStandalone() ([]EntitySummary, error) {
	return csm.ListWithFilters(map[string]string{"standalone": "true"})
}

// ListByProject returns config sheets associated with a specific project
func (csm *ConfigSheetManager) ListByProject(projectUUID string) ([]EntitySummary, error) {
	return csm.ListWithFilters(map[string]string{"project": projectUUID})
}

// Update updates an existing config sheet
func (csm *ConfigSheetManager) Update(nameOrUUID string, updater func(*ConfigSheet) error) (*ConfigSheet, error) {
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

