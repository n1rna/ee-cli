// Package api contains models for API communication
package api

import (
	"encoding/json"
	"time"
)

// APITime handles timestamp parsing from the API
type APITime time.Time

// UnmarshalJSON implements json.Unmarshaler for APITime
func (t *APITime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// Try different timestamp formats that the API might return
	formats := []string{
		"2006-01-02T15:04:05.000000",
		"2006-01-02T15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range formats {
		if parsed, err := time.Parse(format, s); err == nil {
			*t = APITime(parsed)
			return nil
		}
	}

	return nil // Return nil to avoid breaking on unknown formats
}

// Time converts APITime to time.Time
func (t APITime) Time() time.Time {
	return time.Time(t)
}

// Variable represents a schema variable for API communication
type Variable struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Regex    string `json:"regex,omitempty"`
	Default  string `json:"default,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// Schema represents a schema for API communication
type Schema struct {
	GUID        string     `json:"guid"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	IsPublic    bool       `json:"is_public,omitempty"`
	Variables   []Variable `json:"variables,omitempty"`
	Extends     []string   `json:"extends,omitempty"`
	CreatedAt   APITime    `json:"created_at,omitempty"`
	UpdatedAt   APITime    `json:"updated_at,omitempty"`
}

// Project represents a project for API communication
type Project struct {
	GUID              string  `json:"guid"`
	Name              string  `json:"name"`
	Description       string  `json:"description,omitempty"`
	DefaultSchemaGUID *string `json:"default_schema_guid,omitempty"`
	CreatedAt         APITime `json:"created_at,omitempty"`
	UpdatedAt         APITime `json:"updated_at,omitempty"`
}

// ConfigSheet represents a config sheet for API communication
type ConfigSheet struct {
	GUID        string            `json:"guid"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	ProjectGUID string            `json:"project_guid"`
	SchemaGUID  string            `json:"schema_guid"`
	Variables   map[string]string `json:"variables,omitempty"`
	Extends     []string          `json:"extends,omitempty"`
	IsActive    bool              `json:"is_active,omitempty"`
	CreatedAt   APITime           `json:"created_at,omitempty"`
	UpdatedAt   APITime           `json:"updated_at,omitempty"`
}
