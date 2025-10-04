// Package api provides conversion functions between local and API types
package api

import (
	"time"

	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/storage"
)

// SchemaToAPI converts a local Schema to API Schema
func SchemaToAPI(localSchema *entities.Schema) *Schema {
	apiVariables := make([]Variable, len(localSchema.Variables))
	for i, v := range localSchema.Variables {
		apiVariables[i] = Variable{
			Name:     v.Name,
			Type:     v.Type,
			Regex:    v.Regex,
			Default:  v.Default,
			Required: v.Required,
		}
	}

	return &Schema{
		GUID:        localSchema.ID,
		Name:        localSchema.Name,
		Description: localSchema.Description,
		Variables:   apiVariables,
		Extends:     localSchema.Extends,
		CreatedAt:   APITime(localSchema.CreatedAt),
		UpdatedAt:   APITime(localSchema.UpdatedAt),
	}
}

// SchemaFromAPI converts an API Schema to local Schema
func SchemaFromAPI(apiSchema *Schema) *entities.Schema {
	localVariables := make([]entities.Variable, len(apiSchema.Variables))
	for i, v := range apiSchema.Variables {
		localVariables[i] = entities.Variable{
			Name:     v.Name,
			Type:     v.Type,
			Regex:    v.Regex,
			Default:  v.Default,
			Required: v.Required,
		}
	}

	return &entities.Schema{
		Entity: storage.Entity{
			ID:          apiSchema.GUID,
			Name:        apiSchema.Name,
			Description: apiSchema.Description,
			CreatedAt:   time.Time(apiSchema.CreatedAt),
			UpdatedAt:   time.Time(apiSchema.UpdatedAt),
		},
		Variables: localVariables,
		Extends:   apiSchema.Extends,
	}
}

// ConfigSheetToAPI converts a local ConfigSheet to API ConfigSheet
func ConfigSheetToAPI(localSheet *entities.ConfigSheet) *ConfigSheet {
	// Extract schema GUID from reference
	schemaGUID := ""
	if localSheet.Schema.IsReference() {
		// Remove "#/schemas/" prefix if present
		if len(localSheet.Schema.Ref) > 10 && localSheet.Schema.Ref[:10] == "#/schemas/" {
			schemaGUID = localSheet.Schema.Ref[10:]
		} else {
			schemaGUID = localSheet.Schema.Ref
		}
	}

	return &ConfigSheet{
		GUID:        localSheet.ID,
		Name:        localSheet.Name,
		Description: localSheet.Description,
		SchemaGUID:  schemaGUID,
		Variables:   localSheet.Values,
		IsActive:    true, // Default to active
		CreatedAt:   APITime(localSheet.CreatedAt),
		UpdatedAt:   APITime(localSheet.UpdatedAt),
	}
}

// ConfigSheetFromAPI converts an API ConfigSheet to local ConfigSheet
func ConfigSheetFromAPI(apiSheet *ConfigSheet) *entities.ConfigSheet {
	// Create schema reference
	schemaRef := entities.SchemaReference{}
	if apiSheet.SchemaGUID != "" {
		schemaRef.Ref = "#/schemas/" + apiSheet.SchemaGUID
	}

	return &entities.ConfigSheet{
		Entity: storage.Entity{
			ID:          apiSheet.GUID,
			Name:        apiSheet.Name,
			Description: apiSheet.Description,
			CreatedAt:   time.Time(apiSheet.CreatedAt),
			UpdatedAt:   time.Time(apiSheet.UpdatedAt),
		},
		Schema: schemaRef,
		Values: apiSheet.Variables,
	}
}
