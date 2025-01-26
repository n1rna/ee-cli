// internal/schema/types.go
package schema

import (
	"fmt"
	"regexp"
	"strings"
)

// Variable represents a single environment variable definition in the schema
type Variable struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Regex    string `yaml:"regex,omitempty"`
	Default  string `yaml:"default,omitempty"`
	Required bool   `yaml:"required,omitempty"`
}

// Schema represents the complete schema definition for a project
type Schema struct {
	Name      string     `yaml:"name"`
	Variables []Variable `yaml:"variables"`
	Extends   []string   `yaml:"extends,omitempty"` // Names of schemas to inherit from
}

// ConfigSheet represents a specific environment configuration for a project
type ConfigSheet struct {
	ProjectName string            `yaml:"project_name"`
	EnvName     string            `yaml:"env_name"`
	Schema      string            `yaml:"schema"`
	Values      map[string]string `yaml:"values"`
	Extends     []string          `yaml:"extends,omitempty"` // Projects to inherit from
}

// Validator handles schema validation logic
type Validator struct {
	compiledRegexes map[string]*regexp.Regexp
	storage         SchemaStorage // Interface for loading schemas
}

// SchemaStorage interface for loading schemas
type SchemaStorage interface {
	LoadSchema(name string) (*Schema, error)
	LoadConfigSheet(projectName, envName string) (*ConfigSheet, error)
}

// NewValidator creates a new validator instance
func NewValidator(storage SchemaStorage) *Validator {
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
	if visited[schema.Name] {
		return nil, fmt.Errorf("circular dependency detected in schema %s", schema.Name)
	}
	visited[schema.Name] = true

	resolved := &Schema{
		Name:      schema.Name,
		Extends:   schema.Extends,
		Variables: make([]Variable, 0),
	}

	// Resolve extended schemas first
	for _, extendName := range schema.Extends {
		extendSchema, err := v.storage.LoadSchema(extendName)
		if err != nil {
			return nil, fmt.Errorf("failed to load extended schema %s: %w", extendName, err)
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
func (v *Validator) resolveConfigSheet(config *ConfigSheet, schema *Schema, visited map[string]bool) (*ConfigSheet, error) {
	if visited[config.ProjectName+":"+config.EnvName] {
		return nil, fmt.Errorf("circular dependency detected in config %s:%s", config.ProjectName, config.EnvName)
	}
	visited[config.ProjectName+":"+config.EnvName] = true

	resolved := &ConfigSheet{
		ProjectName: config.ProjectName,
		EnvName:     config.EnvName,
		Schema:      config.Schema,
		Extends:     config.Extends,
		Values:      make(map[string]string),
	}

	// Resolve extended projects first
	for _, extendName := range config.Extends {
		parts := strings.Split(extendName, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid extend format %s, should be project:env", extendName)
		}
		projectName, envName := parts[0], parts[1]

		extendConfig, err := v.storage.LoadConfigSheet(projectName, envName)
		if err != nil {
			return nil, fmt.Errorf("failed to load extended config %s:%s: %w", projectName, envName, err)
		}

		// Recursively resolve the extended config
		resolvedExtend, err := v.resolveConfigSheet(extendConfig, schema, visited)
		if err != nil {
			return nil, err
		}

		// Add values from extended config
		for k, v := range resolvedExtend.Values {
			resolved.Values[k] = v
		}
	}

	// Add/override with current config's values
	for k, v := range config.Values {
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
func (v *Validator) ValidateConfigSheet(sheet *ConfigSheet, schema *Schema) error {
	if sheet.ProjectName == "" {
		return fmt.Errorf("project name cannot be empty")
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
