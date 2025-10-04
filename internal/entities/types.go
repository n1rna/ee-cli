// Package entities defines data structures and business logic for ee entities.
// This is a complete rewrite for UUID-based entity architecture as specified in docs/entities.md
package entities

import (
	"github.com/n1rna/ee-cli/internal/storage"
)

// Variable represents a single environment variable definition in the schema
// Enhanced with title field for better UX
type Variable struct {
	Name     string `json:"name"`              // Variable name (e.g., DATABASE_URL)
	Title    string `json:"title,omitempty"`   // Human-readable title (e.g., "Database Connection URL")
	Type     string `json:"type"`              // Variable type: string, number, boolean, url
	Regex    string `json:"regex,omitempty"`   // Validation regex pattern
	Default  string `json:"default,omitempty"` // Default value
	Required bool   `json:"required"`          // Whether variable is required
}

// Schema represents the complete schema definition for environment variables
// Now inherits from Entity base with UUID support
type Schema struct {
	storage.Entity            // Embedded Entity base
	Variables      []Variable `json:"variables"` // Variable definitions
	Extends        []string   `json:"extends"`   // UUIDs or names of schemas to inherit from
}

// NewSchema creates a new schema with generated UUID
func NewSchema(name, description string, variables []Variable, extends []string) *Schema {
	if extends == nil {
		extends = []string{}
	}
	return &Schema{
		Entity:    storage.NewEntity(name, description),
		Variables: variables,
		Extends:   extends,
	}
}

// SchemaReference represents either a reference to an existing schema or inline schema definition
// Supports both referenced and dynamic schemas as specified in docs/entities.md
type SchemaReference struct {
	Ref       string              `json:"ref,omitempty"`       // Reference like "#/schemas/{uuid}" or schema name
	Variables map[string]Variable `json:"variables,omitempty"` // Inline schema variables for dynamic schemas
}

// IsInline returns true if this is an inline schema definition
func (sr *SchemaReference) IsInline() bool {
	return len(sr.Variables) > 0 && sr.Ref == ""
}

// IsReference returns true if this references an existing schema
func (sr *SchemaReference) IsReference() bool {
	return sr.Ref != ""
}

// ConfigSheet represents a standalone configuration sheet
// Can exist independently or be associated with a project environment
type ConfigSheet struct {
	storage.Entity                   // Embedded Entity base
	Schema         SchemaReference   `json:"schema"`            // Schema definition or reference
	Values         map[string]string `json:"values"`            // Variable values
	Extends        []string          `json:"extends,omitempty"` // UUIDs or names of config sheets to inherit from
}

// NewConfigSheet creates a new config sheet with generated UUID
func NewConfigSheet(
	name, description string,
	schema SchemaReference,
	values map[string]string,
) *ConfigSheet {
	if values == nil {
		values = make(map[string]string)
	}
	return &ConfigSheet{
		Entity:  storage.NewEntity(name, description),
		Schema:  schema,
		Values:  values,
		Extends: []string{},
	}
}
