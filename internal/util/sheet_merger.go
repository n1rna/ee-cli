package util

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/parser"
)

// ConfigSheetMerger handles merging multiple config sheets according to environment definitions
type ConfigSheetMerger struct {
	manager *entities.Manager
}

// NewConfigSheetMerger creates a new config sheet merger
func NewConfigSheetMerger(manager *entities.Manager) *ConfigSheetMerger {
	return &ConfigSheetMerger{
		manager: manager,
	}
}

// MergeEnvironment merges all config sheets for the given environment definition
func (csm *ConfigSheetMerger) MergeEnvironment(
	env parser.EnvironmentDefinition,
) (map[string]string, error) {
	var sheetRefs []interface{}

	// Handle single sheet reference
	if env.Sheet != "" {
		sheetRefs = append(sheetRefs, env.Sheet)
	}

	// Handle multiple sheets
	if len(env.Sheets) > 0 {
		sheetRefs = append(sheetRefs, env.Sheets...)
	}

	if len(sheetRefs) == 0 {
		return map[string]string{}, nil
	}

	// Merge sheets in order (later sheets override earlier ones)
	result := make(map[string]string)
	for i, sheetRef := range sheetRefs {
		values, err := csm.resolveSheetReference(sheetRef)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve sheet reference %d: %w", i, err)
		}

		// Merge values (later values override earlier ones)
		for key, value := range values {
			result[key] = value
		}
	}

	return result, nil
}

// resolveSheetReference resolves a single sheet reference to its key-value pairs
func (csm *ConfigSheetMerger) resolveSheetReference(ref interface{}) (map[string]string, error) {
	switch v := ref.(type) {
	case string:
		return csm.resolveStringReference(v)
	case map[string]interface{}:
		return csm.resolveInlineObject(v)
	default:
		return nil, fmt.Errorf("unsupported sheet reference type: %T", ref)
	}
}

// resolveStringReference resolves string references (local, remote, or file paths)
func (csm *ConfigSheetMerger) resolveStringReference(ref string) (map[string]string, error) {
	switch {
	case strings.HasPrefix(ref, "local://"):
		// Local config sheet reference
		sheetName := strings.TrimPrefix(ref, "local://")
		return csm.resolveLocalConfigSheet(sheetName)

	case strings.HasPrefix(ref, "remote://"):
		// Remote config sheet reference
		return nil, fmt.Errorf("remote config sheet references not yet implemented: %s", ref)

	case strings.HasPrefix(ref, ".env"):
		// .env file reference (including .env.development, .env.production, etc.)
		return csm.resolveDotEnvFile(ref)

	default:
		// Assume it's a local config sheet name
		return csm.resolveLocalConfigSheet(ref)
	}
}

// resolveLocalConfigSheet resolves a local config sheet by name
func (csm *ConfigSheetMerger) resolveLocalConfigSheet(name string) (map[string]string, error) {
	cs, err := csm.manager.ConfigSheets.Get(name)
	if err != nil {
		return nil, fmt.Errorf("local config sheet '%s' not found: %w", name, err)
	}
	return cs.Values, nil
}

// resolveDotEnvFile resolves a .env file reference
func (csm *ConfigSheetMerger) resolveDotEnvFile(path string) (map[string]string, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf(".env file not found: %s", path)
	}

	return csm.parseDotEnvFile(path)
}

// resolveInlineObject resolves an inline object (direct key-value pairs)
func (csm *ConfigSheetMerger) resolveInlineObject(
	obj map[string]interface{},
) (map[string]string, error) {
	result := make(map[string]string)
	for key, value := range obj {
		// Convert value to string
		switch v := value.(type) {
		case string:
			result[key] = v
		case nil:
			result[key] = ""
		default:
			// Convert other types to JSON string representation
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to serialize value for key '%s': %w", key, err)
			}
			result[key] = string(jsonBytes)
		}
	}
	return result, nil
}

// parseDotEnvFile parses a .env file and returns its key-value pairs
func (csm *ConfigSheetMerger) parseDotEnvFile(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read .env file: %w", err)
	}

	result := make(map[string]string)
	lines := strings.Split(string(content), "\n")

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line %d in .env file %s: %s", lineNum+1, path, line)
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
			return nil, fmt.Errorf(
				"empty variable name on line %d in .env file %s",
				lineNum+1,
				path,
			)
		}

		result[key] = value
	}

	return result, nil
}
