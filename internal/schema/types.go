// Package schema defines data structures and validation for environment schemas.
// This is a complete rewrite for UUID-based entity architecture as specified in docs/entities.md
package schema

import (
	"fmt"
	"regexp"
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
func NewConfigSheet(name, description string, schema SchemaReference, values map[string]string) *ConfigSheet {
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
	projectUUID, envName string, values map[string]string) *ConfigSheet {
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

// Validator handles schema validation logic with the new architecture
type Validator struct {
	compiledRegexes map[string]*regexp.Regexp
	storage         Storage // Interface for loading entities
}

// Storage interface for loading entities with UUID support
// This will be implemented in the storage layer refactoring (Phase 2)
type Storage interface {
	// Entity resolution
	ResolveUUID(entityType, nameOrUUID string) (string, error)

	// Schema operations
	LoadSchema(nameOrUUID string) (*Schema, error)
	SaveSchema(schema *Schema) error
	ListSchemas() ([]*EntitySummary, error)

	// Project operations
	LoadProject(nameOrUUID string) (*Project, error)
	SaveProject(project *Project) error
	ListProjects() ([]*EntitySummary, error)

	// Config sheet operations
	LoadConfigSheet(nameOrUUID string) (*ConfigSheet, error)
	SaveConfigSheet(sheet *ConfigSheet) error
	ListConfigSheets(filters map[string]string) ([]*EntitySummary, error)

	// Index operations
	LoadIndex(entityType string) (*Index, error)
	SaveIndex(entityType string, index *Index) error
}

// NewValidator creates a new validator instance
func NewValidator(storage Storage) *Validator {
	return &Validator{
		compiledRegexes: make(map[string]*regexp.Regexp),
		storage:         storage,
	}
}

// validateVariable checks if a variable definition is valid
func (v *Validator) validateVariable(variable *Variable) error {
	if variable.Name == "" {
		return fmt.Errorf("variable name cannot be empty")
	}

	// Validate type
	switch variable.Type {
	case "string", "number", "boolean", "url":
		// Valid types
	default:
		return fmt.Errorf("unsupported type: %s", variable.Type)
	}

	// Compile and validate regex if provided
	if variable.Regex != "" {
		if _, exists := v.compiledRegexes[variable.Regex]; !exists {
			compiled, err := regexp.Compile(variable.Regex)
			if err != nil {
				return fmt.Errorf("invalid regex pattern: %w", err)
			}
			v.compiledRegexes[variable.Regex] = compiled
		}
	}

	// Validate default value if provided
	if variable.Default != "" {
		if err := v.ValidateValue(variable, variable.Default); err != nil {
			return fmt.Errorf("invalid default value: %w", err)
		}
	}

	return nil
}

// ValidateValue checks if a value matches the variable's constraints
func (v *Validator) ValidateValue(variable *Variable, value string) error {
	if value == "" && variable.Required {
		return fmt.Errorf("value is required")
	}

	switch variable.Type {
	case "number":
		// Add number validation logic
		// TODO: Implement number parsing and validation
	case "boolean":
		if value != "true" && value != "false" {
			return fmt.Errorf("boolean value must be 'true' or 'false'")
		}
	case "url":
		// Add URL validation logic
		// TODO: Implement URL validation
	}

	// Check regex pattern if defined
	if variable.Regex != "" {
		if regex, exists := v.compiledRegexes[variable.Regex]; exists {
			if !regex.MatchString(value) {
				return fmt.Errorf("value does not match regex pattern")
			}
		}
	}

	return nil
}

// resolveSchema resolves a schema with all its inherited variables
func (v *Validator) resolveSchema(schema *Schema, visited map[string]bool) (*Schema, error) {
	if visited[schema.ID] {
		return nil, fmt.Errorf("circular dependency detected in schema %s", schema.Name)
	}
	visited[schema.ID] = true

	resolved := &Schema{
		Entity:    schema.Entity,
		Extends:   schema.Extends,
		Variables: make([]Variable, 0),
	}

	// Resolve extended schemas first
	for _, extendNameOrUUID := range schema.Extends {
		extendSchema, err := v.storage.LoadSchema(extendNameOrUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to load extended schema %s: %w", extendNameOrUUID, err)
		}

		// Recursively resolve the extended schema
		resolvedExtend, err := v.resolveSchema(extendSchema, visited)
		if err != nil {
			return nil, err
		}

		// Add variables from extended schema
		resolved.Variables = append(resolved.Variables, resolvedExtend.Variables...)
	}

	// Add/override with current schema's variables
	variableMap := make(map[string]Variable)
	for _, v := range resolved.Variables {
		variableMap[v.Name] = v
	}
	for _, v := range schema.Variables {
		variableMap[v.Name] = v
	}

	// Convert back to slice
	resolved.Variables = make([]Variable, 0, len(variableMap))
	for _, v := range variableMap {
		resolved.Variables = append(resolved.Variables, v)
	}

	return resolved, nil
}

// resolveConfigSheet resolves a config sheet with all inherited values
func (v *Validator) resolveConfigSheet(
	sheet *ConfigSheet, schema *Schema, visited map[string]bool,
) (*ConfigSheet, error) {
	if visited[sheet.ID] {
		return nil, fmt.Errorf("circular dependency detected in config sheet %s", sheet.Name)
	}
	visited[sheet.ID] = true

	resolved := &ConfigSheet{
		Entity:      sheet.Entity,
		Schema:      sheet.Schema,
		Project:     sheet.Project,
		Environment: sheet.Environment,
		Extends:     sheet.Extends,
		Values:      make(map[string]string),
	}

	// Resolve extended config sheets first
	for _, extendNameOrUUID := range sheet.Extends {
		extendSheet, err := v.storage.LoadConfigSheet(extendNameOrUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to load extended config sheet %s: %w", extendNameOrUUID, err)
		}

		// Recursively resolve the extended config sheet
		resolvedExtend, err := v.resolveConfigSheet(extendSheet, schema, visited)
		if err != nil {
			return nil, err
		}

		// Add values from extended config sheet
		for k, v := range resolvedExtend.Values {
			resolved.Values[k] = v
		}
	}

	// Add/override with current sheet's values
	for k, v := range sheet.Values {
		resolved.Values[k] = v
	}

	return resolved, nil
}

// ValidateSchema checks if a schema definition is valid
func (v *Validator) ValidateSchema(schema *Schema) error {
	if schema.Name == "" {
		return fmt.Errorf("schema name cannot be empty")
	}

	// Resolve schema inheritance
	resolved, err := v.resolveSchema(schema, make(map[string]bool))
	if err != nil {
		return fmt.Errorf("failed to resolve schema inheritance: %w", err)
	}

	// Validate all variables
	for _, variable := range resolved.Variables {
		if err := v.validateVariable(&variable); err != nil {
			return fmt.Errorf("invalid variable %s: %w", variable.Name, err)
		}
	}

	return nil
}

// ValidateConfigSheet validates a config sheet against its schema
func (v *Validator) ValidateConfigSheet(sheet *ConfigSheet) error {
	if sheet.Name == "" {
		return fmt.Errorf("config sheet name cannot be empty")
	}

	// Load schema based on reference type
	var schema *Schema
	var err error

	if sheet.Schema.IsReference() {
		// Load referenced schema
		schema, err = v.storage.LoadSchema(sheet.Schema.Ref)
		if err != nil {
			return fmt.Errorf("failed to load referenced schema: %w", err)
		}
	} else if sheet.Schema.IsInline() {
		// Create temporary schema from inline definition
		variables := make([]Variable, 0, len(sheet.Schema.Variables))
		for _, variable := range sheet.Schema.Variables {
			variables = append(variables, variable)
		}
		schema = NewSchema("inline", "inline schema", variables, nil)
	} else {
		return fmt.Errorf("config sheet must have either schema reference or inline schema")
	}

	// Resolve schema inheritance
	resolvedSchema, err := v.resolveSchema(schema, make(map[string]bool))
	if err != nil {
		return fmt.Errorf("failed to resolve schema inheritance: %w", err)
	}

	// Resolve config sheet inheritance
	resolvedSheet, err := v.resolveConfigSheet(sheet, resolvedSchema, make(map[string]bool))
	if err != nil {
		return fmt.Errorf("failed to resolve config inheritance: %w", err)
	}

	// Validate all variables against the resolved schema
	for _, variable := range resolvedSchema.Variables {
		value, exists := resolvedSheet.Values[variable.Name]

		if !exists {
			if variable.Required {
				return fmt.Errorf("required variable %s is missing", variable.Name)
			}
			if variable.Default != "" {
				resolvedSheet.Values[variable.Name] = variable.Default
			}
			continue
		}

		if err := v.ValidateValue(&variable, value); err != nil {
			return fmt.Errorf("invalid value for %s: %w", variable.Name, err)
		}
	}

	// Update the original sheet with resolved values
	sheet.Values = resolvedSheet.Values

	return nil
}

// ConfigSheetFilter provides filtering options for config sheet queries
type ConfigSheetFilter struct {
	ProjectGUID    string // Filter by project UUID
	Environment    string // Filter by environment name
	StandaloneOnly bool   // Filter for standalone sheets only (no project association)
}

// ConfigSheetSummary provides summary information about config sheets for listing
type ConfigSheetSummary struct {
	EntitySummary        // Embedded summary
	ProjectGUID   string `json:"project_guid,omitempty"` // Associated project UUID
	Environment   string `json:"environment,omitempty"`  // Environment name if applicable
}
