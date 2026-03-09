// Package entities provides schema loading from files.
package entities

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadSchemaFromFile loads a schema from a YAML or JSON file.
// The path can be absolute or relative to the current working directory.
func LoadSchemaFromFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file %s: %w", path, err)
	}

	var schema Schema

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &schema)
	case ".json":
		err = json.Unmarshal(data, &schema)
	default:
		// Try YAML first, then JSON
		if err = yaml.Unmarshal(data, &schema); err != nil {
			err = json.Unmarshal(data, &schema)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema file %s: %w", path, err)
	}

	if schema.Name == "" {
		schema.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	return &schema, nil
}

// ResolveSchemaRef resolves a schema reference string to a file path and loads it.
// Supported formats:
//   - "file://path/to/schema.yaml" — explicit file reference
//   - "./schema.yaml", "../schema.yaml" — relative paths
//   - "/absolute/path/schema.yaml" — absolute paths
//   - "schema.yaml" — plain filename (must have .yaml/.yml/.json extension)
func ResolveSchemaRef(ref string) (*Schema, error) {
	switch {
	case strings.HasPrefix(ref, "file://"):
		return LoadSchemaFromFile(strings.TrimPrefix(ref, "file://"))

	case strings.HasPrefix(ref, "./"),
		strings.HasPrefix(ref, "../"),
		filepath.IsAbs(ref):
		return LoadSchemaFromFile(ref)

	default:
		ext := strings.ToLower(filepath.Ext(ref))
		if ext == ".yaml" || ext == ".yml" || ext == ".json" {
			return LoadSchemaFromFile(ref)
		}
		return nil, fmt.Errorf(
			"unsupported schema reference: %s (use a file path with .yaml/.yml/.json extension)",
			ref,
		)
	}
}
