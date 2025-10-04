package storage

import "time"

// EntitySummary represents a lightweight summary of an entity for index.json files
type EntitySummary struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Remote      string    `json:"remote,omitempty"`
	Local       bool      `json:"local"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Index represents the structure of index.json files for entity management
// Provides fast name-to-UUID resolution and entity summaries
type Index struct {
	NameToID  map[string]string        `json:"name_to_id"` // Map name -> UUID
	Summaries map[string]EntitySummary `json:"summaries"`  // Map UUID -> EntitySummary
}

// NewIndex creates a new empty index
func NewIndex() *Index {
	return &Index{
		NameToID:  make(map[string]string),
		Summaries: make(map[string]EntitySummary),
	}
}

// AddEntity adds an entity to the index
func (idx *Index) AddEntity(entity Entity) {
	idx.NameToID[entity.Name] = entity.ID
	idx.Summaries[entity.ID] = EntitySummary{
		Name:        entity.Name,
		Description: entity.Description,
		Remote:      entity.Remote,
		Local:       entity.Local,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}
}

// RemoveEntity removes an entity from the index
func (idx *Index) RemoveEntity(nameOrUUID string) {
	// Try to resolve UUID first
	uuid := nameOrUUID
	if resolvedUUID, exists := idx.NameToID[nameOrUUID]; exists {
		uuid = resolvedUUID
		delete(idx.NameToID, nameOrUUID)
	}

	// Remove from summaries
	delete(idx.Summaries, uuid)

	// Remove any other name mappings to this UUID
	for name, id := range idx.NameToID {
		if id == uuid {
			delete(idx.NameToID, name)
		}
	}
}

// ResolveUUID resolves a name or UUID to a UUID
func (idx *Index) ResolveUUID(nameOrUUID string) (string, bool) {
	// If it's already a UUID and exists in summaries, return it
	if _, exists := idx.Summaries[nameOrUUID]; exists {
		return nameOrUUID, true
	}

	// Try to resolve as name
	if uuid, exists := idx.NameToID[nameOrUUID]; exists {
		return uuid, true
	}

	return "", false
}

// GetSummary gets the summary for an entity by name or UUID
func (idx *Index) GetSummary(nameOrUUID string) (EntitySummary, bool) {
	uuid, exists := idx.ResolveUUID(nameOrUUID)
	if !exists {
		return EntitySummary{}, false
	}

	summary, exists := idx.Summaries[uuid]
	return summary, exists
}

// ListSummaries returns all entity summaries
func (idx *Index) ListSummaries() []EntitySummary {
	summaries := make([]EntitySummary, 0, len(idx.Summaries))
	for _, summary := range idx.Summaries {
		summaries = append(summaries, summary)
	}
	return summaries
}
