// Package storage handles UUID-based file system operations for ee.
// This is a complete rewrite implementing the new architecture from docs/refactoring-plan.md
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/n1rna/ee-cli/internal/config"
	"github.com/n1rna/ee-cli/internal/schema"
)

const (
	schemasDir    = "schemas"
	sheetsDir     = "sheets"
	projectsDir   = "projects"
	indexFile     = "index.json"
	fileExtension = ".json"
)

// UUIDStorage implements the new UUID-based storage system with index.json files
// This replaces the old YAML-based storage with the new architecture
type UUIDStorage struct {
	config     *config.Config
	indexCache map[string]*schema.Index // Cache for loaded indices
}

// NewUUIDStorage creates a new UUID-based storage instance
func NewUUIDStorage(cfg *config.Config) (*UUIDStorage, error) {
	storage := &UUIDStorage{
		config:     cfg,
		indexCache: make(map[string]*schema.Index),
	}

	// Ensure directory structure exists
	if err := storage.initDirectories(); err != nil {
		return nil, fmt.Errorf("failed to initialize directories: %w", err)
	}

	return storage, nil
}

// initDirectories creates the new directory structure if it doesn't exist
func (s *UUIDStorage) initDirectories() error {
	dirs := []string{
		s.getSchemasDir(),
		s.getSheetsDir(),
		s.getProjectsDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// Directory path helpers
func (s *UUIDStorage) getSchemasDir() string {
	return filepath.Join(s.config.BaseDir, schemasDir)
}

func (s *UUIDStorage) getSheetsDir() string {
	return filepath.Join(s.config.BaseDir, sheetsDir)
}

func (s *UUIDStorage) getProjectsDir() string {
	return filepath.Join(s.config.BaseDir, projectsDir)
}

func (s *UUIDStorage) getIndexPath(entityType string) string {
	var dir string
	switch entityType {
	case "schemas":
		dir = s.getSchemasDir()
	case "sheets":
		dir = s.getSheetsDir()
	case "projects":
		dir = s.getProjectsDir()
	default:
		return ""
	}
	return filepath.Join(dir, indexFile)
}

func (s *UUIDStorage) getEntityFilePath(entityType, uuid string) string {
	var dir string
	switch entityType {
	case "schemas":
		dir = s.getSchemasDir()
	case "sheets":
		dir = s.getSheetsDir()
	case "projects":
		dir = s.getProjectsDir()
	default:
		return ""
	}
	return filepath.Join(dir, uuid+fileExtension)
}

// Index operations

// LoadIndex loads an index for the given entity type
func (s *UUIDStorage) LoadIndex(entityType string) (*schema.Index, error) {
	// Check cache first
	if index, exists := s.indexCache[entityType]; exists {
		return index, nil
	}

	indexPath := s.getIndexPath(entityType)

	// Create empty index if file doesn't exist
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		index := schema.NewIndex()
		s.indexCache[entityType] = index
		return index, nil
	}

	// Read and parse index file
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}

	var index schema.Index
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse index file: %w", err)
	}

	// Ensure maps are initialized
	if index.NameToID == nil {
		index.NameToID = make(map[string]string)
	}
	if index.Summaries == nil {
		index.Summaries = make(map[string]schema.EntitySummary)
	}

	// Cache and return
	s.indexCache[entityType] = &index
	return &index, nil
}

// SaveIndex saves an index for the given entity type
func (s *UUIDStorage) SaveIndex(entityType string, index *schema.Index) error {
	indexPath := s.getIndexPath(entityType)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON with proper formatting
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// Write to file
	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	// Update cache
	s.indexCache[entityType] = index

	return nil
}

// Entity resolution

// ResolveUUID resolves a name or UUID to a UUID for the given entity type
func (s *UUIDStorage) ResolveUUID(entityType, nameOrUUID string) (string, error) {
	index, err := s.LoadIndex(entityType)
	if err != nil {
		return "", fmt.Errorf("failed to load index: %w", err)
	}

	uuid, exists := index.ResolveUUID(nameOrUUID)
	if !exists {
		return "", fmt.Errorf("%s not found: %s", entityType, nameOrUUID)
	}

	return uuid, nil
}

// Generic entity file operations

// saveEntityFile saves an entity to its JSON file
func (s *UUIDStorage) saveEntityFile(entityType, uuid string, entity interface{}) error {
	filePath := s.getEntityFilePath(entityType, uuid)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(entity, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entity: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write entity file: %w", err)
	}

	return nil
}

// loadEntityFile loads an entity from its JSON file
func (s *UUIDStorage) loadEntityFile(entityType, uuid string, entity interface{}) error {
	filePath := s.getEntityFilePath(entityType, uuid)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("entity file not found: %s", filePath)
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read entity file: %w", err)
	}

	// Unmarshal JSON
	if err := json.Unmarshal(data, entity); err != nil {
		return fmt.Errorf("failed to parse entity file: %w", err)
	}

	return nil
}

// deleteEntityFile deletes an entity's JSON file
func (s *UUIDStorage) deleteEntityFile(entityType, uuid string) error {
	filePath := s.getEntityFilePath(entityType, uuid)

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete entity file: %w", err)
	}

	return nil
}

// Schema operations

// LoadSchema loads a schema by name or UUID
func (s *UUIDStorage) LoadSchema(nameOrUUID string) (*schema.Schema, error) {
	// Resolve to UUID
	uuid, err := s.ResolveUUID("schemas", nameOrUUID)
	if err != nil {
		return nil, err
	}

	// Load from file
	var schemaData schema.Schema
	if err := s.loadEntityFile("schemas", uuid, &schemaData); err != nil {
		return nil, fmt.Errorf("failed to load schema: %w", err)
	}

	return &schemaData, nil
}

// SaveSchema saves a schema and updates the index
func (s *UUIDStorage) SaveSchema(schemaData *schema.Schema) error {
	// Save entity file
	if err := s.saveEntityFile("schemas", schemaData.ID, schemaData); err != nil {
		return fmt.Errorf("failed to save schema file: %w", err)
	}

	// Update index
	index, err := s.LoadIndex("schemas")
	if err != nil {
		return fmt.Errorf("failed to load schemas index: %w", err)
	}

	index.AddEntity(schemaData.Entity)

	if err := s.SaveIndex("schemas", index); err != nil {
		return fmt.Errorf("failed to save schemas index: %w", err)
	}

	return nil
}

// DeleteSchema deletes a schema by name or UUID
func (s *UUIDStorage) DeleteSchema(nameOrUUID string) error {
	return s.DeleteEntity("schemas", nameOrUUID)
}

// ListSchemas returns summaries of all schemas
func (s *UUIDStorage) ListSchemas() ([]*schema.EntitySummary, error) {
	index, err := s.LoadIndex("schemas")
	if err != nil {
		return nil, fmt.Errorf("failed to load schemas index: %w", err)
	}

	summaries := index.ListSummaries()
	result := make([]*schema.EntitySummary, len(summaries))
	for i := range summaries {
		result[i] = &summaries[i]
	}

	return result, nil
}

// Project operations

// LoadProject loads a project by name or UUID
func (s *UUIDStorage) LoadProject(nameOrUUID string) (*schema.Project, error) {
	// Resolve to UUID
	uuid, err := s.ResolveUUID("projects", nameOrUUID)
	if err != nil {
		return nil, err
	}

	// Load from file
	var projectData schema.Project
	if err := s.loadEntityFile("projects", uuid, &projectData); err != nil {
		return nil, fmt.Errorf("failed to load project: %w", err)
	}

	return &projectData, nil
}

// SaveProject saves a project and updates the index
func (s *UUIDStorage) SaveProject(projectData *schema.Project) error {
	// Save entity file
	if err := s.saveEntityFile("projects", projectData.ID, projectData); err != nil {
		return fmt.Errorf("failed to save project file: %w", err)
	}

	// Update index
	index, err := s.LoadIndex("projects")
	if err != nil {
		return fmt.Errorf("failed to load projects index: %w", err)
	}

	index.AddEntity(projectData.Entity)

	if err := s.SaveIndex("projects", index); err != nil {
		return fmt.Errorf("failed to save projects index: %w", err)
	}

	return nil
}

// ListProjects returns summaries of all projects
func (s *UUIDStorage) ListProjects() ([]*schema.EntitySummary, error) {
	index, err := s.LoadIndex("projects")
	if err != nil {
		return nil, fmt.Errorf("failed to load projects index: %w", err)
	}

	summaries := index.ListSummaries()
	result := make([]*schema.EntitySummary, len(summaries))
	for i := range summaries {
		result[i] = &summaries[i]
	}

	return result, nil
}

// Config sheet operations

// LoadConfigSheet loads a config sheet by name or UUID
func (s *UUIDStorage) LoadConfigSheet(nameOrUUID string) (*schema.ConfigSheet, error) {
	// Resolve to UUID
	uuid, err := s.ResolveUUID("sheets", nameOrUUID)
	if err != nil {
		return nil, err
	}

	// Load from file
	var sheetData schema.ConfigSheet
	if err := s.loadEntityFile("sheets", uuid, &sheetData); err != nil {
		return nil, fmt.Errorf("failed to load config sheet: %w", err)
	}

	return &sheetData, nil
}

// SaveConfigSheet saves a config sheet and updates the index
func (s *UUIDStorage) SaveConfigSheet(sheetData *schema.ConfigSheet) error {
	// Save entity file
	if err := s.saveEntityFile("sheets", sheetData.ID, sheetData); err != nil {
		return fmt.Errorf("failed to save config sheet file: %w", err)
	}

	// Update index
	index, err := s.LoadIndex("sheets")
	if err != nil {
		return fmt.Errorf("failed to load sheets index: %w", err)
	}

	index.AddEntity(sheetData.Entity)

	if err := s.SaveIndex("sheets", index); err != nil {
		return fmt.Errorf("failed to save sheets index: %w", err)
	}

	return nil
}

// ListConfigSheets returns summaries of config sheets with optional filtering
func (s *UUIDStorage) ListConfigSheets(filter *schema.ConfigSheetFilter) ([]*schema.ConfigSheetSummary, error) {
	index, err := s.LoadIndex("sheets")
	if err != nil {
		return nil, fmt.Errorf("failed to load sheets index: %w", err)
	}

	summaries := index.ListSummaries()
	result := make([]*schema.ConfigSheetSummary, 0)

	for _, summary := range summaries {
		// Load full config sheet to get project and environment info for filtering
		configSheet, err := s.LoadConfigSheet(summary.Name)
		if err != nil {
			continue // Skip if we can't load the sheet
		}

		// Apply filters if provided
		if filter != nil {
			if filter.StandaloneOnly && configSheet.Project != "" {
				continue // Skip project-associated sheets when filtering for standalone only
			}
			if filter.ProjectGUID != "" && configSheet.Project != filter.ProjectGUID {
				continue
			}
			if filter.Environment != "" && configSheet.Environment != filter.Environment {
				continue
			}
		}

		// Create ConfigSheetSummary
		sheetSummary := &schema.ConfigSheetSummary{
			EntitySummary: summary,
			ProjectGUID:   configSheet.Project,
			Environment:   configSheet.Environment,
		}

		result = append(result, sheetSummary)
	}

	return result, nil
}

// DeleteEntity removes an entity and updates the index
func (s *UUIDStorage) DeleteEntity(entityType, nameOrUUID string) error {
	// Resolve to UUID
	uuid, err := s.ResolveUUID(entityType, nameOrUUID)
	if err != nil {
		return err
	}

	// Delete entity file
	if err := s.deleteEntityFile(entityType, uuid); err != nil {
		return fmt.Errorf("failed to delete entity file: %w", err)
	}

	// Update index
	index, err := s.LoadIndex(entityType)
	if err != nil {
		return fmt.Errorf("failed to load index: %w", err)
	}

	index.RemoveEntity(uuid)

	if err := s.SaveIndex(entityType, index); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	return nil
}

// Utility methods

// EntityExists checks if an entity exists by name or UUID
func (s *UUIDStorage) EntityExists(entityType, nameOrUUID string) bool {
	_, err := s.ResolveUUID(entityType, nameOrUUID)
	return err == nil
}

// GetEntitySummary gets the summary for an entity
func (s *UUIDStorage) GetEntitySummary(entityType, nameOrUUID string) (*schema.EntitySummary, error) {
	index, err := s.LoadIndex(entityType)
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	summary, exists := index.GetSummary(nameOrUUID)
	if !exists {
		return nil, fmt.Errorf("entity not found: %s", nameOrUUID)
	}

	return &summary, nil
}

// ListAllEntityTypes returns all available entity types
func (s *UUIDStorage) ListAllEntityTypes() []string {
	return []string{"schemas", "projects", "sheets"}
}

// GetStorageStats returns statistics about the storage
func (s *UUIDStorage) GetStorageStats() (map[string]int, error) {
	stats := make(map[string]int)

	for _, entityType := range s.ListAllEntityTypes() {
		summaries, err := s.getEntityCount(entityType)
		if err != nil {
			return nil, fmt.Errorf("failed to get %s count: %w", entityType, err)
		}
		stats[entityType] = summaries
	}

	return stats, nil
}

// getEntityCount returns the number of entities of a given type
func (s *UUIDStorage) getEntityCount(entityType string) (int, error) {
	index, err := s.LoadIndex(entityType)
	if err != nil {
		return 0, err
	}

	return len(index.Summaries), nil
}

// Close cleans up resources (currently just clears caches)
func (s *UUIDStorage) Close() error {
	// Clear index cache
	s.indexCache = make(map[string]*schema.Index)
	return nil
}

// Validate checks the integrity of the storage
func (s *UUIDStorage) Validate() error {
	for _, entityType := range s.ListAllEntityTypes() {
		// Load index
		index, err := s.LoadIndex(entityType)
		if err != nil {
			return fmt.Errorf("failed to load %s index: %w", entityType, err)
		}

		// Check that all entities in index have corresponding files
		for uuid := range index.Summaries {
			filePath := s.getEntityFilePath(entityType, uuid)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				return fmt.Errorf("missing file for %s entity %s: %s", entityType, uuid, filePath)
			}
		}

		// Check for orphaned files (files without index entries)
		dir := s.getEntityDir(entityType)
		entries, err := os.ReadDir(dir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to read %s directory: %w", entityType, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || entry.Name() == indexFile {
				continue
			}

			// Extract UUID from filename
			name := entry.Name()
			if !strings.HasSuffix(name, fileExtension) {
				continue
			}

			uuid := strings.TrimSuffix(name, fileExtension)
			if _, exists := index.Summaries[uuid]; !exists {
				return fmt.Errorf("orphaned file in %s: %s (UUID: %s)", entityType, name, uuid)
			}
		}
	}

	return nil
}

// getEntityDir returns the directory for a given entity type
func (s *UUIDStorage) getEntityDir(entityType string) string {
	switch entityType {
	case "schemas":
		return s.getSchemasDir()
	case "sheets":
		return s.getSheetsDir()
	case "projects":
		return s.getProjectsDir()
	default:
		return ""
	}
}
