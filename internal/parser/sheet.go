package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/n1rna/ee-cli/internal/output"
)

// SheetParser handles parsing config sheet values from various sources
type SheetParser struct {
	reader *bufio.Reader
}

// NewSheetParser creates a new config sheet parser
func NewSheetParser() *SheetParser {
	return &SheetParser{
		reader: bufio.NewReader(os.Stdin),
	}
}

// SheetData represents the parsed config sheet values
type SheetData struct {
	Values map[string]string
}

// ParseFile parses config sheet values from a YAML, JSON, or dotenv file
func (p *SheetParser) ParseFile(path string) (*SheetData, error) {
	// Detect file format based on extension
	ext := strings.ToLower(filepath.Ext(path))

	// If it's a .env file, use the dotenv parser
	if ext == ".env" || strings.Contains(strings.ToLower(path), ".env") {
		dotenvParser := NewAnnotatedDotEnvParser()
		values, _, err := dotenvParser.ParseFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse .env file: %w", err)
		}
		return &SheetData{Values: values}, nil
	}

	// For other files, read and try YAML/JSON
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	values := make(map[string]string)

	// Try YAML first, then JSON, then dotenv as fallback
	if err := yaml.Unmarshal(data, &values); err != nil {
		if err := json.Unmarshal(data, &values); err != nil {
			// Try parsing as dotenv file as fallback
			dotenvParser := NewAnnotatedDotEnvParser()
			parsedValues, _, parseErr := dotenvParser.ParseFile(path)
			if parseErr != nil {
				return nil, fmt.Errorf("file is neither valid YAML, JSON, nor dotenv format")
			}
			return &SheetData{Values: parsedValues}, nil
		}
	}

	return &SheetData{Values: values}, nil
}

// ParseCLIValues parses config sheet values from CLI key-value pairs
func (p *SheetParser) ParseCLIValues(cliValues map[string]string) (*SheetData, error) {
	if len(cliValues) == 0 {
		return &SheetData{Values: make(map[string]string)}, nil
	}

	// CLI values are already in the correct format (map[string]string)
	return &SheetData{Values: cliValues}, nil
}

// MergeValues merges multiple SheetData, with later values taking precedence
func (p *SheetParser) MergeValues(sheets ...*SheetData) *SheetData {
	merged := make(map[string]string)

	for _, sheet := range sheets {
		if sheet != nil {
			for k, v := range sheet.Values {
				merged[k] = v
			}
		}
	}

	return &SheetData{Values: merged}
}

// ParseInteractive interactively prompts the user for config sheet values
// If a schema is provided, it will prompt for each variable defined in the schema
func (p *SheetParser) ParseInteractive(schemaVariableNames []string) (*SheetData, error) {
	values := make(map[string]string)
	printer := output.NewPrinter(output.FormatTable, false)

	if len(schemaVariableNames) > 0 {
		// Schema-guided interactive creation
		printer.Println("Creating config sheet with schema-defined variables...")
		printer.Println("Enter values for each variable (press Enter to skip optional variables):")
		printer.Println("")

		for _, varName := range schemaVariableNames {
			printer.Printf("Enter value for %s: ", varName)
			value, err := p.reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("failed to read value for %s: %w", varName, err)
			}

			value = strings.TrimSpace(value)
			if value != "" {
				values[varName] = value
			}
		}
	} else {
		// Free-form interactive creation
		printer.Println("Creating config sheet interactively...")
		printer.Println("Enter key-value pairs (press Enter with empty key to finish):")
		printer.Println("")

		for {
			printer.Printf("Enter variable name (or empty to finish): ")
			key, err := p.reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("failed to read variable name: %w", err)
			}

			key = strings.TrimSpace(key)
			if key == "" {
				break
			}

			// Check for duplicate keys
			if _, exists := values[key]; exists {
				printer.Warning(fmt.Sprintf("Variable %s already exists, overwriting...", key))
			}

			printer.Printf("Enter value for %s: ", key)
			value, err := p.reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("failed to read value for %s: %w", key, err)
			}

			value = strings.TrimSpace(value)
			values[key] = value
		}
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("config sheet must contain at least one value")
	}

	return &SheetData{Values: values}, nil
}
