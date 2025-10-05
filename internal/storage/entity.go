package storage

import (
	"time"

	"github.com/google/uuid"
)

// Entity represents the base structure for all ee entities (Schema, Project, ConfigSheet)
// with UUID-based identification and remote/local tracking capabilities
type Entity struct {
	ID          string    `json:"id"`                    // UUID for distributed identification
	Name        string    `json:"name"`                  // Human-readable name
	Description string    `json:"description,omitempty"` // Optional description
	Remote      string    `json:"remote,omitempty"`      // Remote URL if synced with API
	Local       bool      `json:"local"`                 // Whether entity exists locally
	CreatedAt   time.Time `json:"created_at"`            // Creation timestamp
	UpdatedAt   time.Time `json:"updated_at"`            // Last update timestamp
}

// NewEntity creates a new entity with generated UUID and current timestamps
func NewEntity(name, description string) Entity {
	now := time.Now()
	return Entity{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Local:       true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// EntitySummary represents a lightweight summary of an entity for index.json files
type EntitySummary struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Remote      string    `json:"remote,omitempty"`
	Local       bool      `json:"local"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
