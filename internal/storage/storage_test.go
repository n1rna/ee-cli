// internal/storage/storage_test.go
package storage

import (
	"os"
	"testing"

	"github.com/n1rna/menv/internal/config"
	"github.com/n1rna/menv/internal/schema"
)

func setupTestStorage(t *testing.T) (*Storage, func()) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "menv-test-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}

	// Create configuration
	cfg := &config.Config{
		BaseDir: tmpDir,
	}

	storage, err := NewStorage(cfg)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create storage: %v", err)
	}

	// Return a cleanup function
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return storage, cleanup
}

func TestSaveAndLoadSchema(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Test schema with inheritance
	baseSchema := &schema.Schema{
		Name: "base-schema",
		Variables: []schema.Variable{
			{
				Name:     "BASE_VAR",
				Type:     "string",
				Default:  "base-value",
				Required: true,
			},
		},
	}

	testSchema := &schema.Schema{
		Name:    "test-schema",
		Extends: []string{"base-schema"},
		Variables: []schema.Variable{
			{
				Name:     "TEST_VAR",
				Type:     "string",
				Default:  "test",
				Required: true,
			},
		},
	}

	// Save base schema
	if err := storage.SaveSchema(baseSchema); err != nil {
		t.Fatalf("failed to save base schema: %v", err)
	}

	// Save test schema
	if err := storage.SaveSchema(testSchema); err != nil {
		t.Fatalf("failed to save test schema: %v", err)
	}

	// Test cache behavior
	// First load should cache
	loadedSchema1, err := storage.LoadSchema("test-schema")
	if err != nil {
		t.Fatalf("failed to load schema first time: %v", err)
	}

	// Second load should use cache
	loadedSchema2, err := storage.LoadSchema("test-schema")
	if err != nil {
		t.Fatalf("failed to load schema second time: %v", err)
	}

	// Verify it's the same instance (cached)
	if loadedSchema1 != loadedSchema2 {
		t.Error("cache not working, got different instances")
	}

	// Test schema updates and cache invalidation
	testSchema.Variables = append(testSchema.Variables, schema.Variable{
		Name: "NEW_VAR",
		Type: "string",
	})

	if err := storage.SaveSchema(testSchema); err != nil {
		t.Fatalf("failed to save updated schema: %v", err)
	}

	// Load again, should get new version
	loadedSchema3, err := storage.LoadSchema("test-schema")
	if err != nil {
		t.Fatalf("failed to load updated schema: %v", err)
	}

	if len(loadedSchema3.Variables) != len(testSchema.Variables) {
		t.Errorf("cache not invalidated properly, expected %d variables, got %d",
			len(testSchema.Variables), len(loadedSchema3.Variables))
	}
}

func TestSaveAndLoadConfigSheet(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create base config
	baseConfig := &schema.ConfigSheet{
		ProjectName: "base-project",
		EnvName:     "development",
		Schema:      "test-schema",
		Values: map[string]string{
			"BASE_VAR": "base-value",
		},
	}

	// Create test config with inheritance
	testConfig := &schema.ConfigSheet{
		ProjectName: "test-project",
		EnvName:     "development",
		Schema:      "test-schema",
		Extends:     []string{"base-project:development"},
		Values: map[string]string{
			"TEST_VAR": "test-value",
		},
	}

	// Save configs
	if err := storage.SaveConfigSheet(baseConfig); err != nil {
		t.Fatalf("failed to save base config: %v", err)
	}

	if err := storage.SaveConfigSheet(testConfig); err != nil {
		t.Fatalf("failed to save test config: %v", err)
	}

	// Test cache behavior
	// First load should cache
	loaded1, err := storage.LoadConfigSheet("test-project", "development")
	if err != nil {
		t.Fatalf("failed to load config first time: %v", err)
	}

	// Second load should use cache
	loaded2, err := storage.LoadConfigSheet("test-project", "development")
	if err != nil {
		t.Fatalf("failed to load config second time: %v", err)
	}

	// Verify it's the same instance (cached)
	if loaded1 != loaded2 {
		t.Error("cache not working, got different instances")
	}

	// Test config updates and cache invalidation
	testConfig.Values["NEW_VAR"] = "new-value"
	if err := storage.SaveConfigSheet(testConfig); err != nil {
		t.Fatalf("failed to save updated config: %v", err)
	}

	// Load again, should get new version
	loaded3, err := storage.LoadConfigSheet("test-project", "development")
	if err != nil {
		t.Fatalf("failed to load updated config: %v", err)
	}

	if _, exists := loaded3.Values["NEW_VAR"]; !exists {
		t.Error("cache not invalidated properly, new value not found")
	}
}

func TestListProjects(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create test projects with inheritance
	baseProject := &schema.ConfigSheet{
		ProjectName: "base-project",
		EnvName:     "development",
		Schema:      "base-schema",
		Values:      map[string]string{"BASE_VAR": "base-value"},
	}

	testProjects := []struct {
		name    string
		extends []string
		values  map[string]string
	}{
		{
			name:    "project1",
			extends: []string{"base-project:development"},
			values:  map[string]string{"PROJ1_VAR": "value1"},
		},
		{
			name:    "project2",
			extends: []string{"base-project:development"},
			values:  map[string]string{"PROJ2_VAR": "value2"},
		},
		{
			name:    "project3",
			extends: []string{"project1:development"},
			values:  map[string]string{"PROJ3_VAR": "value3"},
		},
	}

	// Save base project
	if err := storage.SaveConfigSheet(baseProject); err != nil {
		t.Fatalf("failed to save base project: %v", err)
	}

	// Save test projects
	for _, proj := range testProjects {
		config := &schema.ConfigSheet{
			ProjectName: proj.name,
			EnvName:     "development",
			Schema:      "test-schema",
			Extends:     proj.extends,
			Values:      proj.values,
		}
		if err := storage.SaveConfigSheet(config); err != nil {
			t.Fatalf("failed to save config sheet for %s: %v", proj.name, err)
		}
	}

	// List projects
	projects, err := storage.ListProjects()
	if err != nil {
		t.Fatalf("failed to list projects: %v", err)
	}

	// Verify projects
	expectedCount := len(testProjects) + 1 // +1 for base project
	if len(projects) != expectedCount {
		t.Errorf("expected %d projects, got %d", expectedCount, len(projects))
	}

	// Check for all projects
	allProjects := append([]string{"base-project"},
		"project1", "project2", "project3")
	for _, expectedProject := range allProjects {
		found := false
		for _, p := range projects {
			if p == expectedProject {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("project %s not found in list", expectedProject)
		}
	}
}

func TestDeleteAndCacheInvalidation(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create and save a test schema
	testSchema := &schema.Schema{
		Name: "test-schema",
		Variables: []schema.Variable{
			{Name: "TEST_VAR", Type: "string"},
		},
	}
	if err := storage.SaveSchema(testSchema); err != nil {
		t.Fatalf("failed to save schema: %v", err)
	}

	// Load it to cache
	_, err := storage.LoadSchema("test-schema")
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	// Delete the schema
	if err := storage.DeleteSchema("test-schema"); err != nil {
		t.Fatalf("failed to delete schema: %v", err)
	}

	// Try to load it again, should fail
	_, err = storage.LoadSchema("test-schema")
	if err == nil {
		t.Error("expected error loading deleted schema, got nil")
	}

	// Similar test for config sheets
	config := &schema.ConfigSheet{
		ProjectName: "test-project",
		EnvName:     "development",
		Schema:      "other-schema",
		Values:      map[string]string{"TEST_VAR": "value"},
	}
	if err := storage.SaveConfigSheet(config); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load to cache
	_, err = storage.LoadConfigSheet("test-project", "development")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Delete environment
	if err := storage.DeleteEnvironment("test-project", "development"); err != nil {
		t.Fatalf("failed to delete environment: %v", err)
	}

	// Try to load it again, should fail
	_, err = storage.LoadConfigSheet("test-project", "development")
	if err == nil {
		t.Error("expected error loading deleted config, got nil")
	}
}
