// Package manager provides validation logic for ee manager.
package manager

import (
	"fmt"
	"regexp"

	"github.com/n1rna/ee-cli/internal/entities"
)

// Validator handles schema validation logic with direct access to entity managers
type Validator struct {
	compiledRegexes map[string]*regexp.Regexp
	manager         *Manager // Direct access to entity managers
}

// NewValidator creates a new validator instance
func NewValidator(manager *Manager) *Validator {
	return &Validator{
		compiledRegexes: make(map[string]*regexp.Regexp),
		manager:         manager,
	}
}

// validateVariable checks if a variable definition is valid
func (v *Validator) validateVariable(variable *entities.Variable) error {
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
func (v *Validator) ValidateValue(variable *entities.Variable, value string) error {
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
func (v *Validator) resolveSchema(
	schema *entities.Schema,
	visited map[string]bool,
) (*entities.Schema, error) {
	if visited[schema.ID] {
		return nil, fmt.Errorf("circular dependency detected in schema %s", schema.Name)
	}
	visited[schema.ID] = true

	resolved := &entities.Schema{
		Entity:    schema.Entity,
		Extends:   schema.Extends,
		Variables: make([]entities.Variable, 0),
	}

	// Resolve extended schemas first
	for _, extendNameOrUUID := range schema.Extends {
		extendSchema, err := v.manager.Schemas.Get(extendNameOrUUID)
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
	variableMap := make(map[string]entities.Variable)
	for _, v := range resolved.Variables {
		variableMap[v.Name] = v
	}
	for _, v := range schema.Variables {
		variableMap[v.Name] = v
	}

	// Convert back to slice
	resolved.Variables = make([]entities.Variable, 0, len(variableMap))
	for _, v := range variableMap {
		resolved.Variables = append(resolved.Variables, v)
	}

	return resolved, nil
}

// resolveConfigSheet resolves a config sheet with all inherited values
func (v *Validator) resolveConfigSheet(
	sheet *entities.ConfigSheet, visited map[string]bool,
) (*entities.ConfigSheet, error) {
	if visited[sheet.ID] {
		return nil, fmt.Errorf("circular dependency detected in config sheet %s", sheet.Name)
	}
	visited[sheet.ID] = true

	resolved := &entities.ConfigSheet{
		Entity:      sheet.Entity,
		Schema:      sheet.Schema,
		Project:     sheet.Project,
		Environment: sheet.Environment,
		Extends:     sheet.Extends,
		Values:      make(map[string]string),
	}

	// Resolve extended config sheets first
	for _, extendNameOrUUID := range sheet.Extends {
		extendSheet, err := v.manager.ConfigSheets.Get(extendNameOrUUID)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to load extended config sheet %s: %w",
				extendNameOrUUID,
				err,
			)
		}

		// Recursively resolve the extended config sheet
		resolvedExtend, err := v.resolveConfigSheet(extendSheet, visited)
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
func (v *Validator) ValidateSchema(schema *entities.Schema) error {
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
func (v *Validator) ValidateConfigSheet(sheet *entities.ConfigSheet) error {
	if sheet.Name == "" {
		return fmt.Errorf("config sheet name cannot be empty")
	}

	// Load schema based on reference type
	var schema *entities.Schema
	var err error

	if sheet.Schema.IsReference() {
		// Load referenced schema
		schema, err = v.manager.Schemas.Get(sheet.Schema.Ref)
		if err != nil {
			return fmt.Errorf("failed to load referenced schema: %w", err)
		}
	} else if sheet.Schema.IsInline() {
		// Create temporary schema from inline definition
		variables := make([]entities.Variable, 0, len(sheet.Schema.Variables))
		for _, variable := range sheet.Schema.Variables {
			variables = append(variables, variable)
		}
		schema = entities.NewSchema("inline", "inline schema", variables, nil)
	} else {
		return fmt.Errorf("config sheet must have either schema reference or inline schema")
	}

	// Resolve schema inheritance
	resolvedSchema, err := v.resolveSchema(schema, make(map[string]bool))
	if err != nil {
		return fmt.Errorf("failed to resolve schema inheritance: %w", err)
	}

	// Resolve config sheet inheritance
	resolvedSheet, err := v.resolveConfigSheet(sheet, make(map[string]bool))
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
