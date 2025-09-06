// Migration utilities for converting old YAML-based storage to new UUID-based JSON storage
// This handles the transition from the old architecture to the new refactored architecture
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/n1rna/ee-cli/internal/config"
	"github.com/n1rna/ee-cli/internal/schema"
	"gopkg.in/yaml.v3"
)

// OldSchema represents the old YAML-based schema structure
type OldSchema struct {
	Name      string        `yaml:"name"`
	Variables []OldVariable `yaml:"variables"`
	Extends   []string      `yaml:"extends,omitempty"`
}

// OldVariable represents the old variable structure
type OldVariable struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Regex    string `yaml:"regex,omitempty"`
	Default  string `yaml:"default,omitempty"`
	Required bool   `yaml:"required,omitempty"`
}

// OldConfigSheet represents the old YAML-based config sheet structure
type OldConfigSheet struct {
	ProjectName string            `yaml:"project_name"`
	EnvName     string            `yaml:"env_name"`
	Schema      string            `yaml:"schema"`
	Values      map[string]string `yaml:"values"`
	Extends     []string          `yaml:"extends,omitempty"`
}

// MigrationReport contains information about the migration process
type MigrationReport struct {
	StartTime       time.Time        `json:"start_time"`
	EndTime         time.Time        `json:"end_time"`
	Duration        time.Duration    `json:"duration"`
	SchemasFound    int              `json:"schemas_found"`
	SchemasMigrated int              `json:"schemas_migrated"`
	SheetsFound     int              `json:"sheets_found"`
	SheetsMigrated  int              `json:"sheets_migrated"`
	ProjectsCreated int              `json:"projects_created"`
	Errors          []MigrationError `json:"errors"`
	Warnings        []string         `json:"warnings"`
}

// MigrationError represents an error that occurred during migration
type MigrationError struct {
	File  string `json:"file"`
	Type  string `json:"type"` // "schema", "config_sheet", "project"
	Error string `json:"error"`
}

// Migrator handles the migration from old YAML storage to new UUID storage
type Migrator struct {
	oldBaseDir string
	newStorage *UUIDStorage
	report     *MigrationReport
	dryRun     bool
}

// NewMigrator creates a new migrator instance
func NewMigrator(oldBaseDir string, newStorage *UUIDStorage, dryRun bool) *Migrator {
	return &Migrator{
		oldBaseDir: oldBaseDir,
		newStorage: newStorage,
		dryRun:     dryRun,
		report: &MigrationReport{
			StartTime: time.Now(),
			Errors:    []MigrationError{},
			Warnings:  []string{},
		},
	}
}

// Migrate performs the complete migration from old to new storage format
func (m *Migrator) Migrate() (*MigrationReport, error) {
	fmt.Println("Starting migration from old YAML storage to new UUID-based JSON storage...")

	if m.dryRun {
		fmt.Println("DRY RUN MODE - No changes will be made")
	}

	// Step 1: Migrate schemas
	if err := m.migrateSchemas(); err != nil {
		return m.report, fmt.Errorf("schema migration failed: %w", err)
	}

	// Step 2: Migrate config sheets and create projects
	if err := m.migrateConfigSheets(); err != nil {
		return m.report, fmt.Errorf("config sheet migration failed: %w", err)
	}

	// Step 3: Finalize report
	m.report.EndTime = time.Now()
	m.report.Duration = m.report.EndTime.Sub(m.report.StartTime)

	fmt.Printf("Migration completed in %v\n", m.report.Duration)
	fmt.Printf("Schemas: %d found, %d migrated\n", m.report.SchemasFound, m.report.SchemasMigrated)
	fmt.Printf("Config sheets: %d found, %d migrated\n", m.report.SheetsFound, m.report.SheetsMigrated)
	fmt.Printf("Projects created: %d\n", m.report.ProjectsCreated)
	fmt.Printf("Errors: %d\n", len(m.report.Errors))
	fmt.Printf("Warnings: %d\n", len(m.report.Warnings))

	return m.report, nil
}

// migrateSchemas migrates all schemas from the old format
func (m *Migrator) migrateSchemas() error {
	schemasDir := filepath.Join(m.oldBaseDir, "schemas")

	// Check if schemas directory exists
	if _, err := os.Stat(schemasDir); os.IsNotExist(err) {
		m.report.Warnings = append(m.report.Warnings, "No schemas directory found in old storage")
		return nil
	}

	// Walk through schema files
	return filepath.Walk(schemasDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-YAML files
		if info.IsDir() || (!strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml")) {
			return nil
		}

		m.report.SchemasFound++

		// Load old schema
		oldSchema, err := m.loadOldSchema(path)
		if err != nil {
			m.report.Errors = append(m.report.Errors, MigrationError{
				File:  path,
				Type:  "schema",
				Error: err.Error(),
			})
			return nil // Continue with other files
		}

		// Convert to new schema
		newSchema := m.convertSchema(oldSchema)

		// Save new schema (unless dry run)
		if !m.dryRun {
			if err := m.newStorage.SaveSchema(newSchema); err != nil {
				m.report.Errors = append(m.report.Errors, MigrationError{
					File:  path,
					Type:  "schema",
					Error: fmt.Sprintf("failed to save converted schema: %v", err),
				})
				return nil
			}
		}

		m.report.SchemasMigrated++
		fmt.Printf("Migrated schema: %s -> %s (%s)\n", oldSchema.Name, newSchema.Name, newSchema.ID)

		return nil
	})
}

// migrateConfigSheets migrates config sheets and creates projects as needed
func (m *Migrator) migrateConfigSheets() error {
	projectsDir := filepath.Join(m.oldBaseDir, "projects")

	// Check if projects directory exists
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		m.report.Warnings = append(m.report.Warnings, "No projects directory found in old storage")
		return nil
	}

	// Track created projects to avoid duplicates
	createdProjects := make(map[string]*schema.Project)

	// Walk through project directories and config files
	return filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-YAML files
		if info.IsDir() || (!strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml")) {
			return nil
		}

		m.report.SheetsFound++

		// Load old config sheet
		oldSheet, err := m.loadOldConfigSheet(path)
		if err != nil {
			m.report.Errors = append(m.report.Errors, MigrationError{
				File:  path,
				Type:  "config_sheet",
				Error: err.Error(),
			})
			return nil
		}

		// Get or create project
		project, err := m.getOrCreateProject(oldSheet, createdProjects)
		if err != nil {
			m.report.Errors = append(m.report.Errors, MigrationError{
				File:  path,
				Type:  "project",
				Error: err.Error(),
			})
			return nil
		}

		// Convert config sheet
		newSheet, err := m.convertConfigSheet(oldSheet, project.ID)
		if err != nil {
			m.report.Errors = append(m.report.Errors, MigrationError{
				File:  path,
				Type:  "config_sheet",
				Error: err.Error(),
			})
			return nil
		}

		// Save config sheet (unless dry run)
		if !m.dryRun {
			if err := m.newStorage.SaveConfigSheet(newSheet); err != nil {
				m.report.Errors = append(m.report.Errors, MigrationError{
					File:  path,
					Type:  "config_sheet",
					Error: fmt.Sprintf("failed to save converted config sheet: %v", err),
				})
				return nil
			}
		}

		// Add environment to project
		project.AddEnvironment(oldSheet.EnvName)

		// Save updated project (unless dry run)
		if !m.dryRun {
			if err := m.newStorage.SaveProject(project); err != nil {
				m.report.Errors = append(m.report.Errors, MigrationError{
					File:  path,
					Type:  "project",
					Error: fmt.Sprintf("failed to update project: %v", err),
				})
				return nil
			}
		}

		m.report.SheetsMigrated++
		fmt.Printf("Migrated config sheet: %s/%s -> %s (%s)\n",
			oldSheet.ProjectName, oldSheet.EnvName, newSheet.Name, newSheet.ID)

		return nil
	})
}

// loadOldSchema loads a schema from the old YAML format
func (m *Migrator) loadOldSchema(filePath string) (*OldSchema, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var oldSchema OldSchema
	if err := yaml.Unmarshal(data, &oldSchema); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &oldSchema, nil
}

// loadOldConfigSheet loads a config sheet from the old YAML format
func (m *Migrator) loadOldConfigSheet(filePath string) (*OldConfigSheet, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var oldSheet OldConfigSheet
	if err := yaml.Unmarshal(data, &oldSheet); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &oldSheet, nil
}

// convertSchema converts an old schema to the new format
func (m *Migrator) convertSchema(oldSchema *OldSchema) *schema.Schema {
	// Convert variables
	variables := make([]schema.Variable, len(oldSchema.Variables))
	for i, oldVar := range oldSchema.Variables {
		variables[i] = schema.Variable{
			Name:     oldVar.Name,
			Title:    "", // Old format didn't have titles
			Type:     oldVar.Type,
			Regex:    oldVar.Regex,
			Default:  oldVar.Default,
			Required: oldVar.Required,
		}
	}

	// Create new schema with UUID
	return schema.NewSchema(oldSchema.Name, "", variables, oldSchema.Extends)
}

// convertConfigSheet converts an old config sheet to the new format
func (m *Migrator) convertConfigSheet(oldSheet *OldConfigSheet, projectUUID string) (*schema.ConfigSheet, error) {
	// Try to resolve schema reference
	var schemaRef schema.SchemaReference
	if oldSheet.Schema != "" {
		// Try to find the schema by name in the new storage
		if !m.dryRun {
			if uuid, err := m.newStorage.ResolveUUID("schemas", oldSheet.Schema); err == nil {
				schemaRef.Ref = "#/schemas/" + uuid
			} else {
				// Schema not found - create a warning
				m.report.Warnings = append(m.report.Warnings,
					fmt.Sprintf("Schema '%s' not found for config sheet %s/%s",
						oldSheet.Schema, oldSheet.ProjectName, oldSheet.EnvName))
				// Create a placeholder reference
				schemaRef.Ref = "#/schemas/" + oldSheet.Schema
			}
		} else {
			// In dry run mode, just create a placeholder
			schemaRef.Ref = "#/schemas/" + oldSheet.Schema
		}
	}

	// Generate name for the config sheet
	name := fmt.Sprintf("%s-%s", oldSheet.ProjectName, oldSheet.EnvName)
	description := fmt.Sprintf("Migrated from %s/%s", oldSheet.ProjectName, oldSheet.EnvName)

	// Create new config sheet
	return schema.NewConfigSheetForProject(name, description, schemaRef,
		projectUUID, oldSheet.EnvName, oldSheet.Values), nil
}

// getOrCreateProject gets an existing project or creates a new one
func (m *Migrator) getOrCreateProject(oldSheet *OldConfigSheet, createdProjects map[string]*schema.Project) (*schema.Project, error) {
	// Check if we've already created this project in this migration
	if project, exists := createdProjects[oldSheet.ProjectName]; exists {
		return project, nil
	}

	// Check if project already exists in new storage
	if !m.dryRun {
		if existingProject, err := m.newStorage.LoadProject(oldSheet.ProjectName); err == nil {
			createdProjects[oldSheet.ProjectName] = existingProject
			return existingProject, nil
		}
	}

	// Create new project
	// Try to resolve schema for the project
	var schemaID string
	if oldSheet.Schema != "" {
		if !m.dryRun {
			if uuid, err := m.newStorage.ResolveUUID("schemas", oldSheet.Schema); err == nil {
				schemaID = uuid
			} else {
				// Schema not found, create a placeholder
				schemaID = "unknown-schema-" + oldSheet.Schema
			}
		} else {
			schemaID = "unknown-schema-" + oldSheet.Schema
		}
	}

	description := fmt.Sprintf("Migrated project from old storage")
	project := schema.NewProject(oldSheet.ProjectName, description, schemaID)

	// Save project (unless dry run)
	if !m.dryRun {
		if err := m.newStorage.SaveProject(project); err != nil {
			return nil, fmt.Errorf("failed to save new project: %w", err)
		}
	}

	createdProjects[oldSheet.ProjectName] = project
	m.report.ProjectsCreated++

	fmt.Printf("Created project: %s (%s)\n", project.Name, project.ID)

	return project, nil
}

// SaveMigrationReport saves the migration report to a file
func (m *Migrator) SaveMigrationReport(filePath string) error {
	data, err := json.MarshalIndent(m.report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal migration report: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write migration report: %w", err)
	}

	return nil
}

// ValidateMigration performs validation checks on the migrated data
func (m *Migrator) ValidateMigration() error {
	fmt.Println("Validating migrated data...")

	// Run storage validation
	if err := m.newStorage.Validate(); err != nil {
		return fmt.Errorf("storage validation failed: %w", err)
	}

	// Get statistics
	stats, err := m.newStorage.GetStorageStats()
	if err != nil {
		return fmt.Errorf("failed to get storage stats: %w", err)
	}

	fmt.Printf("Validation successful. Storage contains:\n")
	for entityType, count := range stats {
		fmt.Printf("  %s: %d\n", entityType, count)
	}

	return nil
}

// BackupOldStorage creates a backup of the old storage before migration
func BackupOldStorage(oldBaseDir, backupDir string) error {
	fmt.Printf("Creating backup of old storage from %s to %s...\n", oldBaseDir, backupDir)

	// Create backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy old storage to backup
	return copyDir(oldBaseDir, backupDir)
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = dstFile.ReadFrom(srcFile)
		return err
	})
}

// MigrateCommand provides a high-level interface for migration
func MigrateCommand(oldBaseDir, newBaseDir string, dryRun bool) error {
	// Create backup first (unless dry run)
	if !dryRun {
		backupDir := oldBaseDir + ".backup." + time.Now().Format("20060102-150405")
		if err := BackupOldStorage(oldBaseDir, backupDir); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("Backup created at: %s\n", backupDir)
	}

	// Create new storage
	cfg := &config.Config{BaseDir: newBaseDir}
	newStorage, err := NewUUIDStorage(cfg)
	if err != nil {
		return fmt.Errorf("failed to create new storage: %w", err)
	}
	defer newStorage.Close()

	// Create migrator and run migration
	migrator := NewMigrator(oldBaseDir, newStorage, dryRun)
	report, err := migrator.Migrate()
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Save migration report
	reportPath := filepath.Join(newBaseDir, "migration-report.json")
	if !dryRun {
		if err := migrator.SaveMigrationReport(reportPath); err != nil {
			fmt.Printf("Warning: failed to save migration report: %v\n", err)
		} else {
			fmt.Printf("Migration report saved to: %s\n", reportPath)
		}
	}

	// Validate migration (unless dry run)
	if !dryRun {
		if err := migrator.ValidateMigration(); err != nil {
			return fmt.Errorf("migration validation failed: %w", err)
		}
	}

	// Print summary
	if len(report.Errors) > 0 {
		fmt.Println("\nErrors encountered during migration:")
		for _, err := range report.Errors {
			fmt.Printf("  %s (%s): %s\n", err.File, err.Type, err.Error)
		}
	}

	if len(report.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("  %s\n", warning)
		}
	}

	fmt.Println("\nMigration completed successfully!")
	return nil
}
