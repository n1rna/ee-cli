// Package entities provides validation logic for ee schemas.
package entities

import (
	"fmt"
	"regexp"
)

// Validator handles schema validation logic
type Validator struct {
	compiledRegexes map[string]*regexp.Regexp
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{
		compiledRegexes: make(map[string]*regexp.Regexp),
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
		// TODO: Implement number parsing and validation
	case "boolean":
		if value != "true" && value != "false" {
			return fmt.Errorf("boolean value must be 'true' or 'false'")
		}
	case "url":
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

// ValidateSchema checks if a schema definition is valid
func (v *Validator) ValidateSchema(schema *Schema) error {
	if schema.Name == "" {
		return fmt.Errorf("schema name cannot be empty")
	}

	for _, variable := range schema.Variables {
		if err := v.validateVariable(&variable); err != nil {
			return fmt.Errorf("invalid variable %s: %w", variable.Name, err)
		}
	}

	return nil
}
