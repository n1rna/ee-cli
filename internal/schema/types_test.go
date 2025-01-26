// internal/schema/types_test.go
package schema

import (
	"fmt"
	"testing"
)

// Mock storage implementation for testing
type mockStorage struct {
	schemas      map[string]*Schema
	configSheets map[string]*ConfigSheet
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		schemas:      make(map[string]*Schema),
		configSheets: make(map[string]*ConfigSheet),
	}
}

func (m *mockStorage) LoadSchema(name string) (*Schema, error) {
	if schema, ok := m.schemas[name]; ok {
		return schema, nil
	}
	return nil, fmt.Errorf("schema not found: %s", name)
}

func (m *mockStorage) LoadConfigSheet(projectName, envName string) (*ConfigSheet, error) {
	key := fmt.Sprintf("%s:%s", projectName, envName)
	if config, ok := m.configSheets[key]; ok {
		return config, nil
	}
	return nil, fmt.Errorf("config sheet not found: %s", key)
}

func TestValidateSchema(t *testing.T) {
	storage := newMockStorage()
	validator := NewValidator(storage)

	// Add base schema for inheritance tests
	storage.schemas["base-schema"] = &Schema{
		Name: "base-schema",
		Variables: []Variable{
			{
				Name:     "INHERITED_VAR",
				Type:     "string",
				Required: true,
			},
		},
	}

	tests := []struct {
		name    string
		schema  Schema
		wantErr bool
	}{
		{
			name: "valid schema",
			schema: Schema{
				Name: "test-schema",
				Variables: []Variable{
					{
						Name:     "BASE_URL",
						Type:     "string",
						Regex:    "^https?://.*$",
						Default:  "http://localhost:8000",
						Required: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid schema with inheritance",
			schema: Schema{
				Name:    "inherited-schema",
				Extends: []string{"base-schema"},
				Variables: []Variable{
					{
						Name:     "NEW_VAR",
						Type:     "string",
						Required: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid regex",
			schema: Schema{
				Name: "invalid-regex",
				Variables: []Variable{
					{
						Name:  "TEST_VAR",
						Type:  "string",
						Regex: "[Invalid Regex",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			schema: Schema{
				Name: "invalid-type",
				Variables: []Variable{
					{
						Name: "TEST_VAR",
						Type: "invalid_type",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "circular inheritance",
			schema: Schema{
				Name:    "circular-schema",
				Extends: []string{"circular-schema"},
				Variables: []Variable{
					{
						Name: "TEST_VAR",
						Type: "string",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "non-existent parent schema",
			schema: Schema{
				Name:    "orphan-schema",
				Extends: []string{"non-existent-schema"},
				Variables: []Variable{
					{
						Name: "TEST_VAR",
						Type: "string",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSchema(&tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfigSheet(t *testing.T) {
	storage := newMockStorage()
	validator := NewValidator(storage)

	// Add base schema and config for inheritance tests
	baseSchema := &Schema{
		Name: "base-schema",
		Variables: []Variable{
			{
				Name:     "SHARED_VAR",
				Type:     "string",
				Required: true,
			},
		},
	}
	storage.schemas["base-schema"] = baseSchema

	testSchema := &Schema{
		Name:    "test-schema",
		Extends: []string{"base-schema"},
		Variables: []Variable{
			{
				Name:     "BASE_URL",
				Type:     "string",
				Regex:    "^https?://.*$",
				Default:  "http://localhost:8000",
				Required: true,
			},
			{
				Name:     "DEBUG",
				Type:     "boolean",
				Required: true,
			},
		},
	}
	storage.schemas["test-schema"] = testSchema

	// Add base config
	storage.configSheets["base:development"] = &ConfigSheet{
		ProjectName: "base",
		EnvName:     "development",
		Schema:      "base-schema",
		Values: map[string]string{
			"SHARED_VAR": "shared-value",
		},
	}

	tests := []struct {
		name        string
		configSheet ConfigSheet
		wantErr     bool
	}{
		{
			name: "valid config",
			configSheet: ConfigSheet{
				ProjectName: "test-project",
				EnvName:     "development",
				Schema:      "test-schema",
				Values: map[string]string{
					"BASE_URL":   "http://localhost:8000",
					"DEBUG":      "true",
					"SHARED_VAR": "inherited-value",
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with inheritance",
			configSheet: ConfigSheet{
				ProjectName: "test-project",
				EnvName:     "development",
				Schema:      "test-schema",
				Extends:     []string{"base:development"},
				Values: map[string]string{
					"BASE_URL": "http://localhost:8000",
					"DEBUG":    "true",
				},
			},
			wantErr: false,
		},
		{
			name: "missing required value",
			configSheet: ConfigSheet{
				ProjectName: "test-project",
				EnvName:     "development",
				Schema:      "test-schema",
				Values: map[string]string{
					"BASE_URL": "http://localhost:8000",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid URL format",
			configSheet: ConfigSheet{
				ProjectName: "test-project",
				EnvName:     "development",
				Schema:      "test-schema",
				Values: map[string]string{
					"BASE_URL": "invalid-url",
					"DEBUG":    "true",
				},
			},
			wantErr: true,
		},
		{
			name: "circular inheritance",
			configSheet: ConfigSheet{
				ProjectName: "circular",
				EnvName:     "development",
				Schema:      "test-schema",
				Extends:     []string{"circular:development"},
				Values:      map[string]string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfigSheet(&tt.configSheet, testSchema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfigSheet() error = %v, wantErr %v", err, tt.wantErr)
			}

			// For inheritance tests, verify that inherited values are present
			if !tt.wantErr && len(tt.configSheet.Extends) > 0 {
				if value, exists := tt.configSheet.Values["SHARED_VAR"]; !exists {
					t.Error("Inherited value SHARED_VAR not present")
				} else if value != "shared-value" {
					t.Errorf("Expected inherited value 'shared-value', got '%s'", value)
				}
			}
		})
	}
}

func TestSchemaInheritanceResolution(t *testing.T) {
	storage := newMockStorage()
	validator := NewValidator(storage)

	// Set up a chain of schemas
	storage.schemas["base"] = &Schema{
		Name: "base",
		Variables: []Variable{
			{Name: "BASE_VAR", Type: "string", Default: "base-value"},
		},
	}

	storage.schemas["middle"] = &Schema{
		Name:    "middle",
		Extends: []string{"base"},
		Variables: []Variable{
			{Name: "MIDDLE_VAR", Type: "string", Required: true},
			{Name: "BASE_VAR", Type: "string", Default: "override-value"}, // Override
		},
	}

	storage.schemas["leaf"] = &Schema{
		Name:    "leaf",
		Extends: []string{"middle"},
		Variables: []Variable{
			{Name: "LEAF_VAR", Type: "string"},
		},
	}

	// Test resolution
	leafSchema := storage.schemas["leaf"]
	err := validator.ValidateSchema(leafSchema)
	if err != nil {
		t.Errorf("Failed to validate inherited schema chain: %v", err)
	}

	// Verify resolution order and overrides
	resolved, err := validator.resolveSchema(leafSchema, make(map[string]bool))
	if err != nil {
		t.Fatalf("Failed to resolve schema: %v", err)
	}

	// Check if all variables are present
	varMap := make(map[string]Variable)
	for _, v := range resolved.Variables {
		varMap[v.Name] = v
	}

	if _, exists := varMap["BASE_VAR"]; !exists {
		t.Error("BASE_VAR not found in resolved schema")
	}
	if _, exists := varMap["MIDDLE_VAR"]; !exists {
		t.Error("MIDDLE_VAR not found in resolved schema")
	}
	if _, exists := varMap["LEAF_VAR"]; !exists {
		t.Error("LEAF_VAR not found in resolved schema")
	}

	// Check override value
	if varMap["BASE_VAR"].Default != "override-value" {
		t.Error("BASE_VAR override not applied correctly")
	}
}
