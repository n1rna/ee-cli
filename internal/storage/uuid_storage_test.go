package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/n1rna/ee-cli/internal/config"
	"github.com/n1rna/ee-cli/internal/schema"
)

func TestNewUUIDStorage(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "ee-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config
	cfg := &config.Config{
		BaseDir: tmpDir,
	}

	// Create storage
	storage, err := NewUUIDStorage(cfg)
	if err != nil {
		t.Fatalf("Failed to create UUID storage: %v", err)
	}
	defer storage.Close()

	// Verify directories were created
	expectedDirs := []string{
		filepath.Join(tmpDir, "schemas"),
		filepath.Join(tmpDir, "sheets"),
		filepath.Join(tmpDir, "projects"),
	}

	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Expected directory %s was not created", dir)
		}
	}
}

func TestIndexOperations(t *testing.T) {
	// Create temporary directory and storage
	tmpDir, err := os.MkdirTemp("", "ee-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{BaseDir: tmpDir}
	storage, err := NewUUIDStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	// Test loading non-existent index (should create empty)
	index, err := storage.LoadIndex("schemas")
	if err != nil {
		t.Fatalf("Failed to load non-existent index: %v", err)
	}

	if index == nil {
		t.Fatal("Index should not be nil")
	}

	if len(index.NameToID) != 0 {
		t.Errorf("Empty index should have 0 name mappings, got %d", len(index.NameToID))
	}

	if len(index.Summaries) != 0 {
		t.Errorf("Empty index should have 0 summaries, got %d", len(index.Summaries))
	}

	// Add an entity to the index
	entity := schema.NewEntity("test-schema", "Test schema")
	index.AddEntity(entity)

	// Save index
	if err := storage.SaveIndex("schemas", index); err != nil {
		t.Fatalf("Failed to save index: %v", err)
	}

	// Reload index and verify
	reloadedIndex, err := storage.LoadIndex("schemas")
	if err != nil {
		t.Fatalf("Failed to reload index: %v", err)
	}

	if len(reloadedIndex.NameToID) != 1 {
		t.Errorf("Reloaded index should have 1 name mapping, got %d", len(reloadedIndex.NameToID))
	}

	if len(reloadedIndex.Summaries) != 1 {
		t.Errorf("Reloaded index should have 1 summary, got %d", len(reloadedIndex.Summaries))
	}

	// Test UUID resolution
	resolvedUUID, err := storage.ResolveUUID("schemas", "test-schema")
	if err != nil {
		t.Fatalf("Failed to resolve UUID: %v", err)
	}

	if resolvedUUID != entity.ID {
		t.Errorf("Expected UUID %s, got %s", entity.ID, resolvedUUID)
	}
}

func TestSchemaOperations(t *testing.T) {
	// Create temporary directory and storage
	tmpDir, err := os.MkdirTemp("", "ee-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{BaseDir: tmpDir}
	storage, err := NewUUIDStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	// Create a schema
	variables := []schema.Variable{
		{
			Name:     "DATABASE_URL",
			Title:    "Database URL",
			Type:     "url",
			Required: true,
		},
		{
			Name:    "DEBUG",
			Type:    "boolean",
			Default: "false",
		},
	}

	testSchema := schema.NewSchema("api-schema", "API Schema", variables, nil)

	// Save schema
	if err := storage.SaveSchema(testSchema); err != nil {
		t.Fatalf("Failed to save schema: %v", err)
	}

	// Load schema by name
	loadedSchema, err := storage.LoadSchema("api-schema")
	if err != nil {
		t.Fatalf("Failed to load schema by name: %v", err)
	}

	if loadedSchema.ID != testSchema.ID {
		t.Errorf("Expected schema ID %s, got %s", testSchema.ID, loadedSchema.ID)
	}

	if loadedSchema.Name != "api-schema" {
		t.Errorf("Expected schema name 'api-schema', got %s", loadedSchema.Name)
	}

	if len(loadedSchema.Variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(loadedSchema.Variables))
	}

	// Load schema by UUID
	loadedByUUID, err := storage.LoadSchema(testSchema.ID)
	if err != nil {
		t.Fatalf("Failed to load schema by UUID: %v", err)
	}

	if loadedByUUID.Name != "api-schema" {
		t.Errorf("Expected schema name 'api-schema', got %s", loadedByUUID.Name)
	}

	// List schemas
	summaries, err := storage.ListSchemas()
	if err != nil {
		t.Fatalf("Failed to list schemas: %v", err)
	}

	if len(summaries) != 1 {
		t.Errorf("Expected 1 schema, got %d", len(summaries))
	}

	if summaries[0].Name != "api-schema" {
		t.Errorf("Expected schema name 'api-schema', got %s", summaries[0].Name)
	}
}

func TestProjectOperations(t *testing.T) {
	// Create temporary directory and storage
	tmpDir, err := os.MkdirTemp("", "ee-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{BaseDir: tmpDir}
	storage, err := NewUUIDStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	// Create a project
	schemaID := "550e8400-e29b-41d4-a716-446655440000"
	testProject := schema.NewProject("my-api", "My API project", schemaID)

	// Add environments
	testProject.AddEnvironment("development")
	testProject.AddEnvironment("production")

	// Save project
	if err := storage.SaveProject(testProject); err != nil {
		t.Fatalf("Failed to save project: %v", err)
	}

	// Load project by name
	loadedProject, err := storage.LoadProject("my-api")
	if err != nil {
		t.Fatalf("Failed to load project by name: %v", err)
	}

	if loadedProject.ID != testProject.ID {
		t.Errorf("Expected project ID %s, got %s", testProject.ID, loadedProject.ID)
	}

	if len(loadedProject.Environments) != 2 {
		t.Errorf("Expected 2 environments, got %d", len(loadedProject.Environments))
	}

	// Verify environments
	_, exists := loadedProject.Environments["development"]
	if !exists {
		t.Error("Development environment should exist")
	}

	// List projects
	summaries, err := storage.ListProjects()
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}

	if len(summaries) != 1 {
		t.Errorf("Expected 1 project, got %d", len(summaries))
	}
}

func TestConfigSheetOperations(t *testing.T) {
	// Create temporary directory and storage
	tmpDir, err := os.MkdirTemp("", "ee-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{BaseDir: tmpDir}
	storage, err := NewUUIDStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	// Create config sheets
	schemaRef := schema.SchemaReference{
		Ref: "#/schemas/api-schema",
	}

	values := map[string]string{
		"DATABASE_URL": "postgresql://localhost:5432/test",
		"DEBUG":        "true",
	}

	// Standalone config sheet
	standaloneSheet := schema.NewConfigSheet("standalone", "Standalone config", schemaRef, values)

	// Project environment config sheet
	projectSheet := schema.NewConfigSheetForProject("api-dev", "API Dev config", schemaRef, "project-uuid", "development", values)

	// Save both sheets
	if err := storage.SaveConfigSheet(standaloneSheet); err != nil {
		t.Fatalf("Failed to save standalone sheet: %v", err)
	}

	if err := storage.SaveConfigSheet(projectSheet); err != nil {
		t.Fatalf("Failed to save project sheet: %v", err)
	}

	// Load sheets
	loadedStandalone, err := storage.LoadConfigSheet("standalone")
	if err != nil {
		t.Fatalf("Failed to load standalone sheet: %v", err)
	}

	if !loadedStandalone.IsStandalone() {
		t.Error("Sheet should be detected as standalone")
	}

	loadedProject, err := storage.LoadConfigSheet("api-dev")
	if err != nil {
		t.Fatalf("Failed to load project sheet: %v", err)
	}

	if !loadedProject.IsProjectEnvironment() {
		t.Error("Sheet should be detected as project environment")
	}

	if loadedProject.Project != "project-uuid" {
		t.Errorf("Expected project UUID 'project-uuid', got %s", loadedProject.Project)
	}

	// List config sheets
	summaries, err := storage.ListConfigSheets(nil)
	if err != nil {
		t.Fatalf("Failed to list config sheets: %v", err)
	}

	if len(summaries) != 2 {
		t.Errorf("Expected 2 config sheets, got %d", len(summaries))
	}

	// Test filtering for standalone sheets
	filter := &schema.ConfigSheetFilter{StandaloneOnly: true}
	filtered, err := storage.ListConfigSheets(filter)
	if err != nil {
		t.Fatalf("Failed to list filtered config sheets: %v", err)
	}

	if len(filtered) != 1 {
		t.Errorf("Expected 1 filtered config sheet, got %d", len(filtered))
	}

	if filtered[0].Name != "standalone" {
		t.Errorf("Expected filtered sheet name 'standalone', got %s", filtered[0].Name)
	}
}

func TestDeleteOperations(t *testing.T) {
	// Create temporary directory and storage
	tmpDir, err := os.MkdirTemp("", "ee-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{BaseDir: tmpDir}
	storage, err := NewUUIDStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	// Create and save a schema
	testSchema := schema.NewSchema("test-schema", "Test schema", []schema.Variable{}, nil)
	if err := storage.SaveSchema(testSchema); err != nil {
		t.Fatal(err)
	}

	// Verify it exists
	if !storage.EntityExists("schemas", "test-schema") {
		t.Error("Schema should exist before deletion")
	}

	// Delete it
	if err := storage.DeleteEntity("schemas", "test-schema"); err != nil {
		t.Fatalf("Failed to delete schema: %v", err)
	}

	// Verify it no longer exists
	if storage.EntityExists("schemas", "test-schema") {
		t.Error("Schema should not exist after deletion")
	}

	// Verify file was deleted
	filePath := storage.getEntityFilePath("schemas", testSchema.ID)
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("Schema file should be deleted")
	}

	// Verify it's not in the index
	index, err := storage.LoadIndex("schemas")
	if err != nil {
		t.Fatal(err)
	}

	if _, exists := index.ResolveUUID("test-schema"); exists {
		t.Error("Schema should not be in index after deletion")
	}
}

func TestStorageValidation(t *testing.T) {
	// Create temporary directory and storage
	tmpDir, err := os.MkdirTemp("", "ee-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{BaseDir: tmpDir}
	storage, err := NewUUIDStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	// Create and save some entities
	testSchema := schema.NewSchema("test-schema", "Test schema", []schema.Variable{}, nil)
	if err := storage.SaveSchema(testSchema); err != nil {
		t.Fatal(err)
	}

	testProject := schema.NewProject("test-project", "Test project", testSchema.ID)
	if err := storage.SaveProject(testProject); err != nil {
		t.Fatal(err)
	}

	// Validation should pass
	if err := storage.Validate(); err != nil {
		t.Errorf("Validation should pass for consistent storage: %v", err)
	}

	// Create an orphaned file (file without index entry)
	orphanUUID := "orphaned-uuid"
	orphanPath := storage.getEntityFilePath("schemas", orphanUUID)
	if err := os.WriteFile(orphanPath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Validation should fail due to orphaned file
	if err := storage.Validate(); err == nil {
		t.Error("Validation should fail due to orphaned file")
	}
}

func TestStorageStats(t *testing.T) {
	// Create temporary directory and storage
	tmpDir, err := os.MkdirTemp("", "ee-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{BaseDir: tmpDir}
	storage, err := NewUUIDStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	// Initially should have 0 of everything
	stats, err := storage.GetStorageStats()
	if err != nil {
		t.Fatalf("Failed to get storage stats: %v", err)
	}

	expectedEmpty := map[string]int{
		"schemas":  0,
		"projects": 0,
		"sheets":   0,
	}

	for entityType, expectedCount := range expectedEmpty {
		if stats[entityType] != expectedCount {
			t.Errorf("Expected %d %s, got %d", expectedCount, entityType, stats[entityType])
		}
	}

	// Add some entities
	testSchema := schema.NewSchema("test-schema", "Test schema", []schema.Variable{}, nil)
	if err := storage.SaveSchema(testSchema); err != nil {
		t.Fatal(err)
	}

	testProject := schema.NewProject("test-project", "Test project", testSchema.ID)
	if err := storage.SaveProject(testProject); err != nil {
		t.Fatal(err)
	}

	testSheet := schema.NewConfigSheet("test-sheet", "Test sheet", schema.SchemaReference{Ref: testSchema.ID}, nil)
	if err := storage.SaveConfigSheet(testSheet); err != nil {
		t.Fatal(err)
	}

	// Check updated stats
	stats, err = storage.GetStorageStats()
	if err != nil {
		t.Fatalf("Failed to get updated storage stats: %v", err)
	}

	expectedCounts := map[string]int{
		"schemas":  1,
		"projects": 1,
		"sheets":   1,
	}

	for entityType, expectedCount := range expectedCounts {
		if stats[entityType] != expectedCount {
			t.Errorf("Expected %d %s, got %d", expectedCount, entityType, stats[entityType])
		}
	}
}

func TestEntitySummary(t *testing.T) {
	// Create temporary directory and storage
	tmpDir, err := os.MkdirTemp("", "ee-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{BaseDir: tmpDir}
	storage, err := NewUUIDStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	// Create and save a schema
	testSchema := schema.NewSchema("test-schema", "Test schema description", []schema.Variable{}, nil)
	if err := storage.SaveSchema(testSchema); err != nil {
		t.Fatal(err)
	}

	// Get entity summary
	summary, err := storage.GetEntitySummary("schemas", "test-schema")
	if err != nil {
		t.Fatalf("Failed to get entity summary: %v", err)
	}

	if summary.Name != "test-schema" {
		t.Errorf("Expected name 'test-schema', got %s", summary.Name)
	}

	if summary.Description != "Test schema description" {
		t.Errorf("Expected description 'Test schema description', got %s", summary.Description)
	}

	if !summary.Local {
		t.Error("Entity should be marked as local")
	}

	if summary.Remote != "" {
		t.Errorf("Entity should not have remote URL, got %s", summary.Remote)
	}

	// Check timestamps are recent
	now := time.Now()
	if now.Sub(summary.CreatedAt) > time.Minute {
		t.Error("CreatedAt should be recent")
	}

	if now.Sub(summary.UpdatedAt) > time.Minute {
		t.Error("UpdatedAt should be recent")
	}
}

func TestJSONSerialization(t *testing.T) {
	// Create temporary directory and storage
	tmpDir, err := os.MkdirTemp("", "ee-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{BaseDir: tmpDir}
	storage, err := NewUUIDStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	// Create a complex schema with all features
	variables := []schema.Variable{
		{
			Name:     "DATABASE_URL",
			Title:    "Database Connection URL",
			Type:     "url",
			Required: true,
		},
		{
			Name:    "DEBUG",
			Type:    "boolean",
			Default: "false",
		},
	}

	testSchema := schema.NewSchema("complex-schema", "Complex test schema", variables, []string{"base-schema"})

	// Save and reload
	if err := storage.SaveSchema(testSchema); err != nil {
		t.Fatal(err)
	}

	loadedSchema, err := storage.LoadSchema("complex-schema")
	if err != nil {
		t.Fatal(err)
	}

	// Verify all fields were preserved
	if len(loadedSchema.Variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(loadedSchema.Variables))
	}

	if len(loadedSchema.Extends) != 1 {
		t.Errorf("Expected 1 extended schema, got %d", len(loadedSchema.Extends))
	}

	if loadedSchema.Variables[0].Title != "Database Connection URL" {
		t.Errorf("Variable title not preserved: %s", loadedSchema.Variables[0].Title)
	}

	// Verify JSON structure by manually reading file
	filePath := storage.getEntityFilePath("schemas", testSchema.ID)
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Check that all expected fields are present
	expectedFields := []string{"id", "name", "description", "local", "created_at", "updated_at", "variables", "extends"}
	for _, field := range expectedFields {
		if _, exists := jsonData[field]; !exists {
			t.Errorf("Expected field %s not found in JSON", field)
		}
	}
}
