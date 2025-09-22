// Package entities defines data structures and business logic for ee entities.
// This is a complete rewrite for UUID-based entity architecture as specified in docs/entities.md
package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Entity represents the base structure for all ee entities (Schema, Project, ConfigSheet)
// with UUID-based identification and remote/local tracking capabilities
type Entity struct {
	ID          string    `json:"id"`                    // UUID for distributed identification
	Name        string    `json:"name"`                  // Human-readable name
	Description string    `json:"description,omitempty"` // Optional description
	Remote      string    `json:"remote,omitempty"`      // Remote URL if synced with API
	Local       bool      `json:"local"`                 // Whether entity exists locally
	CreatedAt   time.Time `json:"created_at"`            // Creation timestamp
	UpdatedAt   time.Time `json:"updated_at"`            // Last update timestamp
}

// NewEntity creates a new entity with generated UUID and current timestamps
func NewEntity(name, description string) Entity {
	now := time.Now()
	return Entity{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Local:       true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

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
	Entity               // Embedded Entity base
	Variables []Variable `json:"variables"` // Variable definitions
	Extends   []string   `json:"extends"`   // UUIDs or names of schemas to inherit from
}

// NewSchema creates a new schema with generated UUID
func NewSchema(name, description string, variables []Variable, extends []string) *Schema {
	if extends == nil {
		extends = []string{}
	}
	return &Schema{
		Entity:    NewEntity(name, description),
		Variables: variables,
		Extends:   extends,
	}
}

// Environment represents an environment within a project
// Only stores the environment name - config sheet is derived via naming convention
type Environment struct {
	Name string `json:"name"` // Environment name (e.g., "development", "production")
}

// Project represents a project containing multiple environments
// Now inherits from Entity base and embeds environments
type Project struct {
	Entity                              // Embedded Entity base
	Schema       string                 `json:"schema"`       // UUID reference to schema
	Environments map[string]Environment `json:"environments"` // Map of environment name -> Environment
}

// NewProject creates a new project with generated UUID
func NewProject(name, description, schemaID string) *Project {
	return &Project{
		Entity:       NewEntity(name, description),
		Schema:       schemaID,
		Environments: make(map[string]Environment),
	}
}

// AddEnvironment adds an environment to the project
func (p *Project) AddEnvironment(envName string) {
	p.Environments[envName] = Environment{
		Name: envName,
	}
	p.UpdatedAt = time.Now()
}

// GetConfigSheetName returns the config sheet name for a given environment
// Uses naming convention: "{project-name}-{environment-name}"
func (p *Project) GetConfigSheetName(envName string) string {
	return fmt.Sprintf("%s-%s", p.Name, envName)
}

// RemoveEnvironment removes an environment from the project
func (p *Project) RemoveEnvironment(envName string) {
	delete(p.Environments, envName)
	p.UpdatedAt = time.Now()
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
	Entity                        // Embedded Entity base
	Schema      SchemaReference   `json:"schema"`                // Schema definition or reference
	Project     string            `json:"project,omitempty"`     // Optional project UUID
	Environment string            `json:"environment,omitempty"` // Optional environment name
	Values      map[string]string `json:"values"`                // Variable values
	Extends     []string          `json:"extends,omitempty"`     // UUIDs or names of config sheets to inherit from
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
		Entity:  NewEntity(name, description),
		Schema:  schema,
		Values:  values,
		Extends: []string{},
	}
}

// NewConfigSheetForProject creates a config sheet associated with a project environment
func NewConfigSheetForProject(name, description string, schema SchemaReference,
	projectUUID, envName string, values map[string]string,
) *ConfigSheet {
	sheet := NewConfigSheet(name, description, schema, values)
	sheet.Project = projectUUID
	sheet.Environment = envName
	return sheet
}

// IsStandalone returns true if this config sheet is not associated with a project
func (cs *ConfigSheet) IsStandalone() bool {
	return cs.Project == ""
}

// IsProjectEnvironment returns true if this config sheet is associated with a project environment
func (cs *ConfigSheet) IsProjectEnvironment() bool {
	return cs.Project != "" && cs.Environment != ""
}

// EntitySummary represents a lightweight summary of an entity for index.json files
type EntitySummary struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Remote      string    `json:"remote,omitempty"`
	Local       bool      `json:"local"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Index represents the structure of index.json files for entity management
// Provides fast name-to-UUID resolution and entity summaries
type Index struct {
	NameToID  map[string]string        `json:"name_to_id"` // Map name -> UUID
	Summaries map[string]EntitySummary `json:"summaries"`  // Map UUID -> EntitySummary
}

// NewIndex creates a new empty index
func NewIndex() *Index {
	return &Index{
		NameToID:  make(map[string]string),
		Summaries: make(map[string]EntitySummary),
	}
}

// AddEntity adds an entity to the index
func (idx *Index) AddEntity(entity Entity) {
	idx.NameToID[entity.Name] = entity.ID
	idx.Summaries[entity.ID] = EntitySummary{
		Name:        entity.Name,
		Description: entity.Description,
		Remote:      entity.Remote,
		Local:       entity.Local,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}
}

// RemoveEntity removes an entity from the index
func (idx *Index) RemoveEntity(nameOrUUID string) {
	// Try to resolve UUID first
	uuid := nameOrUUID
	if resolvedUUID, exists := idx.NameToID[nameOrUUID]; exists {
		uuid = resolvedUUID
		delete(idx.NameToID, nameOrUUID)
	}

	// Remove from summaries
	delete(idx.Summaries, uuid)

	// Remove any other name mappings to this UUID
	for name, id := range idx.NameToID {
		if id == uuid {
			delete(idx.NameToID, name)
		}
	}
}

// ResolveUUID resolves a name or UUID to a UUID
func (idx *Index) ResolveUUID(nameOrUUID string) (string, bool) {
	// If it's already a UUID and exists in summaries, return it
	if _, exists := idx.Summaries[nameOrUUID]; exists {
		return nameOrUUID, true
	}

	// Try to resolve as name
	if uuid, exists := idx.NameToID[nameOrUUID]; exists {
		return uuid, true
	}

	return "", false
}

// GetSummary gets the summary for an entity by name or UUID
func (idx *Index) GetSummary(nameOrUUID string) (EntitySummary, bool) {
	uuid, exists := idx.ResolveUUID(nameOrUUID)
	if !exists {
		return EntitySummary{}, false
	}

	summary, exists := idx.Summaries[uuid]
	return summary, exists
}

// ListSummaries returns all entity summaries
func (idx *Index) ListSummaries() []EntitySummary {
	summaries := make([]EntitySummary, 0, len(idx.Summaries))
	for _, summary := range idx.Summaries {
		summaries = append(summaries, summary)
	}
	return summaries
}