// Package parser provides support for annotated dotenv files with schema definitions.
package parser

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/storage"
)

// AnnotatedDotEnvParser parses .env files with schema annotations in comments
type AnnotatedDotEnvParser struct{}

// NewAnnotatedDotEnvParser creates a new annotated dotenv parser
func NewAnnotatedDotEnvParser() *AnnotatedDotEnvParser {
	return &AnnotatedDotEnvParser{}
}

// ParseFile parses an annotated .env file and returns both the values and extracted schema
func (p *AnnotatedDotEnvParser) ParseFile(path string) (map[string]string, entities.Schema, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, entities.Schema{}, fmt.Errorf("failed to open .env file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log the error but don't override the main error
		}
	}()

	values := make(map[string]string)
	variables := make(map[string]entities.Variable)

	scanner := bufio.NewScanner(file)
	lineNum := 0

	currentVarAnnotations := make(map[string]string)
	var schemaRef string

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Handle schema reference comment
		if strings.HasPrefix(line, "# schema:") {
			schemaRef = strings.TrimSpace(strings.TrimPrefix(line, "# schema:"))
			continue
		}

		// Handle variable annotation comments
		if strings.HasPrefix(line, "#") {
			p.parseAnnotationComment(line, currentVarAnnotations)
			continue
		}

		// Handle KEY=VALUE lines
		if strings.Contains(line, "=") {
			key, value, err := p.parseKeyValue(line, lineNum)
			if err != nil {
				return nil, entities.Schema{}, err
			}

			values[key] = value

			// Create variable definition from annotations
			variable := p.createVariableFromAnnotations(key, currentVarAnnotations)
			variables[key] = variable

			// Clear annotations for next variable
			currentVarAnnotations = make(map[string]string)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, entities.Schema{}, fmt.Errorf("error reading .env file: %w", err)
	}

	// Create schema from extracted variables
	// Convert map to slice for Schema.Variables field
	varSlice := make([]entities.Variable, 0, len(variables))
	for _, variable := range variables {
		varSlice = append(varSlice, variable)
	}

	schema := entities.Schema{
		Entity:    storage.NewEntity("", "Schema extracted from .env file"),
		Variables: varSlice,
	}

	// Set schema reference if found
	if schemaRef != "" {
		// This would be used to reference an external schema
		// For now, we'll just include it as a comment in the description
		schema.Description = fmt.Sprintf("References schema: %s", schemaRef)
	}

	return values, schema, nil
}

// parseAnnotationComment parses a comment line for variable annotations
func (p *AnnotatedDotEnvParser) parseAnnotationComment(
	line string,
	annotations map[string]string,
) {
	// Remove the leading #
	content := strings.TrimSpace(strings.TrimPrefix(line, "#"))

	// Skip empty comments or schema comments
	if content == "" || strings.HasPrefix(content, "schema:") {
		return
	}

	// Parse key: value format
	parts := strings.SplitN(content, ":", 2)
	if len(parts) != 2 {
		// Ignore non-annotation comments
		return
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	annotations[key] = value
}

// parseKeyValue parses a KEY=VALUE line
func (p *AnnotatedDotEnvParser) parseKeyValue(line string, lineNum int) (string, string, error) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("line %d: invalid format, expected KEY=VALUE", lineNum)
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	// Remove surrounding quotes if present
	if len(value) >= 2 {
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}
	}

	if key == "" {
		return "", "", fmt.Errorf("line %d: empty variable name", lineNum)
	}

	return key, value, nil
}

// createVariableFromAnnotations creates a Variable definition from comment annotations
func (p *AnnotatedDotEnvParser) createVariableFromAnnotations(
	name string,
	annotations map[string]string,
) entities.Variable {
	variable := entities.Variable{
		Name: name,
		Type: "string", // default type
	}

	// Apply annotations
	if typ, exists := annotations["type"]; exists {
		variable.Type = typ
	}

	if title, exists := annotations["title"]; exists {
		variable.Title = title
	}

	if defaultVal, exists := annotations["default"]; exists {
		variable.Default = defaultVal
	}

	if regex, exists := annotations["regex"]; exists {
		variable.Regex = regex
	}

	if required, exists := annotations["required"]; exists {
		variable.Required = strings.ToLower(required) == "true"
	}

	return variable
}

// ExportAnnotatedDotEnv exports values and schema to an annotated .env format
func (p *AnnotatedDotEnvParser) ExportAnnotatedDotEnv(
	values map[string]string,
	schema *entities.Schema,
	path string,
) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create .env file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log the error but don't override the main error
		}
	}()

	// Write schema reference if available
	if schema != nil && strings.Contains(schema.Description, "References schema:") {
		schemaRef := strings.TrimPrefix(schema.Description, "References schema: ")
		if _, err := fmt.Fprintf(file, "# schema: %s\n\n", schemaRef); err != nil {
			return fmt.Errorf("failed to write schema reference: %w", err)
		}
	} else if schema != nil && schema.Description != "" {
		if _, err := fmt.Fprintf(file, "# schema: inline\n\n"); err != nil {
			return fmt.Errorf("failed to write schema header: %w", err)
		}
	}

	// Write variables with annotations
	for key, value := range values {
		// Write annotations if schema is available
		if schema != nil {
			// Find variable in schema by name
			for _, variable := range schema.Variables {
				if variable.Name == key {
					if err := p.writeVariableAnnotations(file, variable); err != nil {
						return err
					}
					break
				}
			}
		}

		// Write the key=value line
		if _, err := fmt.Fprintf(file, "%s=%s\n\n", key, p.escapeValue(value)); err != nil {
			return fmt.Errorf("failed to write variable %s: %w", key, err)
		}
	}

	return nil
}

// writeVariableAnnotations writes the annotation comments for a variable
func (p *AnnotatedDotEnvParser) writeVariableAnnotations(
	file *os.File,
	variable entities.Variable,
) error {
	if variable.Title != "" {
		if _, err := fmt.Fprintf(file, "# title: %s\n", variable.Title); err != nil {
			return fmt.Errorf("failed to write title annotation: %w", err)
		}
	}

	if variable.Type != "" && variable.Type != "string" {
		if _, err := fmt.Fprintf(file, "# type: %s\n", variable.Type); err != nil {
			return fmt.Errorf("failed to write type annotation: %w", err)
		}
	}

	if variable.Default != "" {
		if _, err := fmt.Fprintf(file, "# default: %s\n", variable.Default); err != nil {
			return fmt.Errorf("failed to write default annotation: %w", err)
		}
	}

	if variable.Regex != "" {
		if _, err := fmt.Fprintf(file, "# regex: %s\n", variable.Regex); err != nil {
			return fmt.Errorf("failed to write regex annotation: %w", err)
		}
	}

	if variable.Required {
		if _, err := fmt.Fprintf(file, "# required: true\n"); err != nil {
			return fmt.Errorf("failed to write required annotation: %w", err)
		}
	}
	return nil
}

// escapeValue escapes a value for .env format if needed
func (p *AnnotatedDotEnvParser) escapeValue(value string) string {
	// If value contains spaces or special characters, quote it
	if strings.ContainsAny(value, " \t\n\r\"'\\") {
		return fmt.Sprintf("\"%s\"", strings.ReplaceAll(value, "\"", "\\\""))
	}
	return value
}
