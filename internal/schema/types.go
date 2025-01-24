// internal/schema/types.go
package schema

import (
    "fmt"
    "regexp"
)

// Variable represents a single environment variable definition in the schema
type Variable struct {
    Name     string  `yaml:"name"`
    Type     string  `yaml:"type"`
    Regex    string  `yaml:"regex,omitempty"`
    Default  string  `yaml:"default,omitempty"`
    Required bool    `yaml:"required,omitempty"`
}

// Schema represents the complete schema definition for a project
type Schema struct {
    Name      string     `yaml:"name"`
    Variables []Variable `yaml:"variables"`
}

// ConfigSheet represents a specific environment configuration for a project
type ConfigSheet struct {
    ProjectName string            `yaml:"project_name"`
    EnvName     string            `yaml:"env_name"`
    Schema      string            `yaml:"schema"`
    Values      map[string]string `yaml:"values"`
}

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

// ValidateConfigSheet validates a config sheet against its schema
func (v *Validator) ValidateConfigSheet(sheet *ConfigSheet, schema *Schema) error {
    if sheet.ProjectName == "" {
        return fmt.Errorf("project name cannot be empty")
    }

    if sheet.EnvName == "" {
        return fmt.Errorf("environment name cannot be empty")
    }

    // Validate all variables in the schema
    for _, variable := range schema.Variables {
        value, exists := sheet.Values[variable.Name]
        
        // Check if required variable is missing
        if !exists {
            if variable.Required {
                return fmt.Errorf("required variable %s is missing", variable.Name)
            }
            // If not required and not provided, use default value
            if variable.Default != "" {
                sheet.Values[variable.Name] = variable.Default
            }
            continue
        }

        // Validate the value
        if err := v.ValidateValue(&variable, value); err != nil {
            return fmt.Errorf("invalid value for %s: %w", variable.Name, err)
        }
    }

    return nil
}