package util

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// EnvResolver handles resolving environment definitions to key-value pairs
// from .env files and inline objects.
type EnvResolver struct{}

// NewEnvResolver creates a new environment resolver
func NewEnvResolver() *EnvResolver {
	return &EnvResolver{}
}

// MergeEnvironment merges all sources for the given environment definition
func (r *EnvResolver) MergeEnvironment(
	env EnvironmentSources,
) (map[string]string, error) {
	var refs []interface{}

	// Handle single env file reference
	if env.Env != "" {
		refs = append(refs, env.Env)
	}

	// Handle sheets (list of .env file paths)
	for _, sheet := range env.Sheets {
		refs = append(refs, sheet)
	}

	// Handle multiple sources
	if len(env.Sources) > 0 {
		refs = append(refs, env.Sources...)
	}

	if len(refs) == 0 {
		return map[string]string{}, nil
	}

	// Merge sources in order (later sources override earlier ones)
	result := make(map[string]string)
	for i, ref := range refs {
		values, err := r.resolveReference(ref)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to resolve source reference %d: %w",
				i, err,
			)
		}

		for key, value := range values {
			result[key] = value
		}
	}

	return result, nil
}

// resolveReference resolves a single source reference to its key-value pairs
func (r *EnvResolver) resolveReference(
	ref interface{},
) (map[string]string, error) {
	switch v := ref.(type) {
	case string:
		return r.resolveDotEnvFile(v)
	case map[string]interface{}:
		return r.resolveInlineObject(v)
	default:
		return nil, fmt.Errorf("unsupported source reference type: %T", ref)
	}
}

// resolveDotEnvFile resolves a .env file reference
func (r *EnvResolver) resolveDotEnvFile(
	path string,
) (map[string]string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf(".env file not found: %s", path)
	}

	return r.parseDotEnvFile(path)
}

// resolveInlineObject resolves an inline object (direct key-value pairs)
func (r *EnvResolver) resolveInlineObject(
	obj map[string]interface{},
) (map[string]string, error) {
	result := make(map[string]string)
	for key, value := range obj {
		switch v := value.(type) {
		case string:
			result[key] = v
		case nil:
			result[key] = ""
		default:
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to serialize value for key '%s': %w",
					key, err,
				)
			}
			result[key] = string(jsonBytes)
		}
	}
	return result, nil
}

// parseDotEnvFile parses a .env file and returns its key-value pairs
func (r *EnvResolver) parseDotEnvFile(
	path string,
) (map[string]string, error) {
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
			return nil, fmt.Errorf(
				"invalid line %d in .env file %s: %s",
				lineNum+1, path, line,
			)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes if present
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") &&
				strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") &&
					strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}

		if key == "" {
			return nil, fmt.Errorf(
				"empty variable name on line %d in .env file %s",
				lineNum+1, path,
			)
		}

		result[key] = value
	}

	return result, nil
}

// EnvironmentSources defines the sources for resolving environment values.
// This is a simplified view used by the resolver.
type EnvironmentSources struct {
	Env     string        // Single .env file reference
	Sources []interface{} // Multiple sources (.env files, inline objects)
	Sheets  []string      // List of .env file paths
}
