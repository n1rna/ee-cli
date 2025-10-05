package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/output"
)

// SchemaParser handles parsing schemas from various sources
type SchemaParser struct {
	reader *bufio.Reader
}

// NewSchemaParser creates a new schema parser
func NewSchemaParser() *SchemaParser {
	return &SchemaParser{
		reader: bufio.NewReader(os.Stdin),
	}
}

// SchemaData represents the parsed schema data before entity creation
type SchemaData struct {
	Description string
	Variables   []entities.Variable
	Extends     []string
}

// ParseFile parses a schema from a YAML, JSON, or dotenv file
func (p *SchemaParser) ParseFile(path string) (*SchemaData, error) {
	// Detect file format based on extension
	ext := strings.ToLower(filepath.Ext(path))

	// If it's a .env file, use the dotenv parser to extract schema
	if ext == ".env" || strings.Contains(strings.ToLower(path), ".env") {
		dotenvParser := NewAnnotatedDotEnvParser()
		_, schema, err := dotenvParser.ParseFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse .env file: %w", err)
		}
		return &SchemaData{
			Description: schema.Description,
			Variables:   schema.Variables,
			Extends:     nil,
		}, nil
	}

	// For other files, read and try YAML/JSON
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Use a temporary struct for unmarshaling that matches the file structure
	var fileData struct {
		Description string              `json:"description" yaml:"description"`
		Variables   []entities.Variable `json:"variables" yaml:"variables"`
		Extends     []string            `json:"extends" yaml:"extends"`
	}

	// Try YAML first, then JSON, then dotenv as fallback
	if err := yaml.Unmarshal(data, &fileData); err != nil {
		if err := json.Unmarshal(data, &fileData); err != nil {
			// Try parsing as dotenv file as fallback
			dotenvParser := NewAnnotatedDotEnvParser()
			_, schema, parseErr := dotenvParser.ParseFile(path)
			if parseErr != nil {
				return nil, fmt.Errorf(
					"file is neither valid YAML, JSON, nor dotenv format: %w",
					parseErr,
				)
			}
			return &SchemaData{
				Description: schema.Description,
				Variables:   schema.Variables,
				Extends:     nil,
			}, nil
		}
		// JSON parsing succeeded
		return &SchemaData{
			Description: fileData.Description,
			Variables:   fileData.Variables,
			Extends:     fileData.Extends,
		}, nil
	}

	// YAML parsing succeeded
	return &SchemaData{
		Description: fileData.Description,
		Variables:   fileData.Variables,
		Extends:     fileData.Extends,
	}, nil
}

// ParseCLISpecs parses schema variables from CLI specifications
// Format: name:type:title:required[:default]
func (p *SchemaParser) ParseCLISpecs(
	description string,
	variableSpecs []string,
) (*SchemaData, error) {
	if len(variableSpecs) == 0 {
		return nil, fmt.Errorf("no variable specifications provided")
	}

	variables := []entities.Variable{}

	for _, varSpec := range variableSpecs {
		variable, err := p.parseVariableSpec(varSpec)
		if err != nil {
			return nil, fmt.Errorf("invalid variable specification '%s': %w", varSpec, err)
		}

		// Check for duplicate variable names
		for _, existingVar := range variables {
			if existingVar.Name == variable.Name {
				return nil, fmt.Errorf("duplicate variable name '%s'", variable.Name)
			}
		}

		variables = append(variables, variable)
	}

	return &SchemaData{
		Description: description,
		Variables:   variables,
		Extends:     nil,
	}, nil
}

// parseVariableSpec parses a variable specification in the format: name:type:title:required[:default]
func (p *SchemaParser) parseVariableSpec(spec string) (entities.Variable, error) {
	// Split into at most 5 parts to handle cases where default values contain colons
	parts := strings.SplitN(spec, ":", 5)
	if len(parts) < 4 {
		return entities.Variable{}, fmt.Errorf(
			"format should be 'name:type:title:required[:default]', got %d parts",
			len(parts),
		)
	}

	name := strings.TrimSpace(parts[0])
	varType := strings.TrimSpace(strings.ToLower(parts[1]))
	title := strings.TrimSpace(parts[2])
	requiredStr := strings.TrimSpace(strings.ToLower(parts[3]))

	// Validate name
	if name == "" {
		return entities.Variable{}, fmt.Errorf("variable name cannot be empty")
	}

	// Validate type
	validTypes := map[string]bool{"string": true, "number": true, "boolean": true, "url": true}
	if !validTypes[varType] {
		return entities.Variable{}, fmt.Errorf(
			"invalid type '%s', must be one of: string, number, boolean, url",
			varType,
		)
	}

	// Parse required flag
	var required bool
	switch requiredStr {
	case "true", "t", "1", "yes", "y":
		required = true
	case "false", "f", "0", "no", "n":
		required = false
	default:
		return entities.Variable{}, fmt.Errorf(
			"invalid required value '%s', must be true/false",
			requiredStr,
		)
	}

	// Parse default value (optional)
	var defaultValue string
	if len(parts) == 5 {
		defaultValue = strings.TrimSpace(parts[4])
	}

	return entities.Variable{
		Name:     name,
		Type:     varType,
		Title:    title,
		Required: required,
		Default:  defaultValue,
	}, nil
}

// ParseInteractive interactively prompts the user for schema variables
func (p *SchemaParser) ParseInteractive() (*SchemaData, error) {
	printer := output.NewPrinter(output.FormatTable, false)

	printer.Println("Creating new schema...")
	printer.Println("For each variable, you'll need to specify:")
	printer.Println("- Name (e.g., DATABASE_URL)")
	printer.Println("- Type (string/number/boolean/url)")
	printer.Println("- Regex pattern (optional)")
	printer.Println("- Default value (optional)")
	printer.Println("- Required flag (y/n)")
	printer.Println("")

	var variables []entities.Variable

	for {
		var variable entities.Variable

		printer.Printf("Enter variable name (or empty to finish): ")
		name, err := p.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read variable name: %w", err)
		}

		name = strings.TrimSpace(name)
		if name == "" {
			break
		}

		// Check for duplicate variable names
		for _, v := range variables {
			if v.Name == name {
				printer.Warning(fmt.Sprintf("Variable %s already exists in schema", name))
				continue
			}
		}

		variable.Name = name

		printer.Printf("Enter variable type (string/number/boolean/url): ")
		varType, err := p.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read variable type: %w", err)
		}

		varType = strings.TrimSpace(strings.ToLower(varType))
		switch varType {
		case "string", "number", "boolean", "url":
			variable.Type = varType
		default:
			printer.Warning(fmt.Sprintf("Invalid type %s, defaulting to string", varType))
			variable.Type = "string"
		}

		printer.Printf("Enter regex pattern (optional): ")
		regex, err := p.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read regex pattern: %w", err)
		}

		regex = strings.TrimSpace(regex)
		if regex != "" {
			variable.Regex = regex
		}

		printer.Printf("Enter default value (optional): ")
		defaultVal, err := p.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read default value: %w", err)
		}

		defaultVal = strings.TrimSpace(defaultVal)
		if defaultVal != "" {
			variable.Default = defaultVal
		}

		printer.Printf("Is this variable required? (y/N): ")
		required, err := p.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read required flag: %w", err)
		}

		required = strings.TrimSpace(strings.ToLower(required))
		variable.Required = required == "y" || required == "yes"

		variables = append(variables, variable)
	}

	if len(variables) == 0 {
		return nil, fmt.Errorf("schema must contain at least one variable")
	}

	return &SchemaData{
		Description: "Schema created interactively",
		Variables:   variables,
		Extends:     nil,
	}, nil
}
