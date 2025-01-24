// internal/storage/storage_test.go
package storage

import (
	"os"
	"path/filepath"
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

	// Override the default storage location for testing
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, nil
	}

	originalBaseDir := cfg.BaseDir
	cfg.BaseDir = filepath.Base(tmpDir)

	storage, err := NewStorage(cfg)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create storage: %v", err)
	}

	// Return a cleanup function
	cleanup := func() {
		cfg.BaseDir = originalBaseDir
		os.RemoveAll(tmpDir)
	}

	return storage, cleanup
}

func TestSaveAndLoadSchema(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Test schema
	testSchema := &schema.Schema{
		Name: "test-schema",
		Variables: []schema.Variable{
			{
				Name:     "TEST_VAR",
				Type:     "string",
				Default:  "test",
				Required: true,
			},
		},
	}

	// Save schema
	if err := storage.SaveSchema(testSchema); err != nil {
		t.Fatalf("failed to save schema: %v", err)
	}

	// Load schema
	loadedSchema, err := storage.LoadSchema("test-schema")
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	// Compare schemas
	if loadedSchema.Name != testSchema.Name {
		t.Errorf("expected schema name %s, got %s", testSchema.Name, loadedSchema.Name)
	}

	if len(loadedSchema.Variables) != len(testSchema.Variables) {
		t.Errorf("expected %d variables, got %d", len(testSchema.Variables), len(loadedSchema.Variables))
	}
}

func TestSaveAndLoadConfigSheet(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Test config sheet
	testConfig := &schema.ConfigSheet{
		ProjectName: "test-project",
		EnvName:     "development",
		Schema:      "test-schema",
		Values: map[string]string{
			"TEST_VAR": "test-value",
		},
	}

	// Save config sheet
	if err := storage.SaveConfigSheet(testConfig); err != nil {
		t.Fatalf("failed to save config sheet: %v", err)
	}

	// Load config sheet
	loadedConfig, err := storage.LoadConfigSheet("test-project", "development")
	if err != nil {
		t.Fatalf("failed to load config sheet: %v", err)
	}

	// Compare config sheets
	if loadedConfig.ProjectName != testConfig.ProjectName {
		t.Errorf("expected project name %s, got %s", testConfig.ProjectName, loadedConfig.ProjectName)
	}

	if loadedConfig.EnvName != testConfig.EnvName {
		t.Errorf("expected env name %s, got %s", testConfig.EnvName, loadedConfig.EnvName)
	}

	if v, ok := loadedConfig.Values["TEST_VAR"]; !ok || v != "test-value" {
		t.Errorf("expected TEST_VAR value 'test-value', got %s", v)
	}
}

func TestListProjects(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create test projects
	testProjects := []string{"project1", "project2", "project3"}
	for _, project := range testProjects {
		config := &schema.ConfigSheet{
			ProjectName: project,
			EnvName:     "development",
			Schema:      "test-schema",
			Values:      map[string]string{},
		}
		if err := storage.SaveConfigSheet(config); err != nil {
			t.Fatalf("failed to save config sheet: %v", err)
		}
	}

	// List projects
	projects, err := storage.ListProjects()
	if err != nil {
		t.Fatalf("failed to list projects: %v", err)
	}

	// Verify projects
	if len(projects) != len(testProjects) {
		t.Errorf("expected %d projects, got %d", len(testProjects), len(projects))
	}

	for _, project := range testProjects {
		found := false
		for _, p := range projects {
			if p == project {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("project %s not found in list", project)
		}
	}
}
