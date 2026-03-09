// Package entities defines data structures for ee schemas and variables.
package entities

// Variable represents a single environment variable definition in the schema
type Variable struct {
	Name     string `json:"name"              yaml:"name"`              // Variable name (e.g., DATABASE_URL)
	Title    string `json:"title,omitempty"   yaml:"title,omitempty"`   // Human-readable title
	Type     string `json:"type"              yaml:"type"`              // Variable type: string, number, boolean, url
	Regex    string `json:"regex,omitempty"   yaml:"regex,omitempty"`   // Validation regex pattern
	Default  string `json:"default,omitempty" yaml:"default,omitempty"` // Default value
	Required bool   `json:"required"          yaml:"required"`          // Whether variable is required
}

// Schema represents a schema definition loaded from a file
type Schema struct {
	Name        string     `json:"name"                  yaml:"name"`
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`
	Variables   []Variable `json:"variables"             yaml:"variables"`
	Extends     []string   `json:"extends,omitempty"     yaml:"extends,omitempty"`
}
