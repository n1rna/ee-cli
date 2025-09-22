// Package entities provides object-oriented interfaces for managing ee entities.
package entities

import (
	"fmt"

	"github.com/n1rna/ee-cli/internal/config"
)

// ProjectManager handles all project-related operations
type ProjectManager struct {
	storage *baseStorage
}

// NewProjectManager creates a new project manager
func NewProjectManager(cfg *config.Config) (*ProjectManager, error) {
	storage, err := newBaseStorage(cfg, "projects")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize project storage: %w", err)
	}

	return &ProjectManager{
		storage: storage,
	}, nil
}

// Create creates a new project
func (pm *ProjectManager) Create(name, description, schemaID string) (*Project, error) {
	// Check if project with same name already exists
	if _, err := pm.GetByName(name); err == nil {
		return nil, fmt.Errorf("project with name '%s' already exists", name)
	}

	// Create new project
	p := NewProject(name, description, schemaID)

	// Save project
	if err := pm.Save(p); err != nil {
		return nil, fmt.Errorf("failed to save project: %w", err)
	}

	return p, nil
}

// Save saves a project to storage
func (pm *ProjectManager) Save(p *Project) error {
	// Save entity file
	if err := pm.storage.saveEntity(p.ID, p); err != nil {
		return fmt.Errorf("failed to save project file: %w", err)
	}

	// Update index
	if err := pm.storage.updateIndex(p.Entity); err != nil {
		return fmt.Errorf("failed to update project index: %w", err)
	}

	return nil
}

// GetByID loads a project by UUID
func (pm *ProjectManager) GetByID(uuid string) (*Project, error) {
	var p Project
	if err := pm.storage.loadEntity(uuid, &p); err != nil {
		return nil, fmt.Errorf("failed to load project %s: %w", uuid, err)
	}
	return &p, nil
}

// GetByName loads a project by name
func (pm *ProjectManager) GetByName(name string) (*Project, error) {
	uuid, err := pm.storage.resolveUUID(name)
	if err != nil {
		return nil, fmt.Errorf("project '%s' not found: %w", name, err)
	}
	return pm.GetByID(uuid)
}

// Get loads a project by name or UUID
func (pm *ProjectManager) Get(nameOrUUID string) (*Project, error) {
	// Try UUID first
	if p, err := pm.GetByID(nameOrUUID); err == nil {
		return p, nil
	}
	// Try name
	return pm.GetByName(nameOrUUID)
}

// Delete removes a project
func (pm *ProjectManager) Delete(nameOrUUID string) error {
	// Resolve to UUID
	uuid, err := pm.storage.resolveUUID(nameOrUUID)
	if err != nil {
		return fmt.Errorf("project '%s' not found: %w", nameOrUUID, err)
	}

	// Remove entity file
	if err := pm.storage.removeEntity(uuid); err != nil {
		return fmt.Errorf("failed to remove project file: %w", err)
	}

	// Remove from index
	if err := pm.storage.removeFromIndex(nameOrUUID); err != nil {
		return fmt.Errorf("failed to remove from project index: %w", err)
	}

	return nil
}

// List returns all project summaries
func (pm *ProjectManager) List() ([]EntitySummary, error) {
	return pm.storage.listSummaries()
}

// Update updates an existing project
func (pm *ProjectManager) Update(nameOrUUID string, updater func(*Project) error) (*Project, error) {
	// Load existing project
	p, err := pm.Get(nameOrUUID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if err := updater(p); err != nil {
		return nil, fmt.Errorf("failed to apply updates: %w", err)
	}

	// Save updated project
	if err := pm.Save(p); err != nil {
		return nil, fmt.Errorf("failed to save updated project: %w", err)
	}

	return p, nil
}

// AddEnvironment adds an environment to a project
func (pm *ProjectManager) AddEnvironment(projectNameOrUUID, envName string) (*Project, error) {
	return pm.Update(projectNameOrUUID, func(p *Project) error {
		p.AddEnvironment(envName)
		return nil
	})
}

// RemoveEnvironment removes an environment from a project
func (pm *ProjectManager) RemoveEnvironment(projectNameOrUUID, envName string) (*Project, error) {
	return pm.Update(projectNameOrUUID, func(p *Project) error {
		p.RemoveEnvironment(envName)
		return nil
	})
}

