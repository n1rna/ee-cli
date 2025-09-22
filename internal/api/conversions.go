// Package api provides conversion utilities between API and local schema models
package api

import (
	"time"

	"github.com/n1rna/ee-cli/internal/entities"
)

// ConvertSchemaFromAPI converts API Schema to local entities.Schema
func ConvertSchemaFromAPI(apiSchema *Schema) *entities.Schema {
	if apiSchema == nil {
		return nil
	}

	localSchema := &entities.Schema{
		Entity: entities.Entity{
			ID:          apiSchema.GUID, // Use GUID as ID for local storage
			Name:        apiSchema.Name,
			Description: apiSchema.Description,
			Remote:      "true", // Mark as coming from remote (string, not bool)
			Local:       false,
			CreatedAt:   apiSchema.CreatedAt.Time(),
			UpdatedAt:   apiSchema.UpdatedAt.Time(),
		},
		Variables: make([]entities.Variable, len(apiSchema.Variables)),
		Extends:   apiSchema.Extends,
	}

	// Convert variables
	for i, apiVar := range apiSchema.Variables {
		localSchema.Variables[i] = entities.Variable{
			Name:     apiVar.Name,
			Type:     apiVar.Type,
			Regex:    apiVar.Regex,
			Default:  apiVar.Default,
			Required: apiVar.Required,
		}
	}

	return localSchema
}

// ConvertSchemaToAPI converts local entities.Schema to API Schema
func ConvertSchemaToAPI(localSchema *entities.Schema) *Schema {
	if localSchema == nil {
		return nil
	}

	apiSchema := &Schema{
		GUID:        localSchema.ID, // Send local UUID as GUID
		Name:        localSchema.Name,
		Description: localSchema.Description,
		IsPublic:    false, // Default to private
		Variables:   make([]Variable, len(localSchema.Variables)),
		Extends:     localSchema.Extends,
	}

	// Convert variables
	for i, localVar := range localSchema.Variables {
		apiSchema.Variables[i] = Variable{
			Name:     localVar.Name,
			Type:     localVar.Type,
			Regex:    localVar.Regex,
			Default:  localVar.Default,
			Required: localVar.Required,
		}
	}

	return apiSchema
}

// ConvertProjectFromAPI converts API Project to local entities.Project
func ConvertProjectFromAPI(apiProject *Project) *entities.Project {
	if apiProject == nil {
		return nil
	}

	localProject := &entities.Project{
		Entity: entities.Entity{
			ID:          apiProject.GUID, // Use GUID as ID for local storage
			Name:        apiProject.Name,
			Description: apiProject.Description,
			Remote:      "true", // Mark as coming from remote (string, not bool)
			Local:       false,
			CreatedAt:   apiProject.CreatedAt.Time(),
			UpdatedAt:   apiProject.UpdatedAt.Time(),
		},
		Environments: make(map[string]entities.Environment), // Will be populated separately
	}

	// Schema will be resolved separately if needed
	if apiProject.DefaultSchemaGUID != nil {
		// Note: We'll need to look up the schema by GUID and set the name
		localProject.Schema = "" // Will be set by caller
	}

	return localProject
}

// ConvertProjectToAPI converts local entities.Project to API Project
func ConvertProjectToAPI(localProject *entities.Project) *Project {
	if localProject == nil {
		return nil
	}

	apiProject := &Project{
		GUID:        localProject.ID, // Send local UUID as GUID
		Name:        localProject.Name,
		Description: localProject.Description,
	}

	// Set DefaultSchemaGUID if the project has a schema
	if localProject.Schema != "" {
		apiProject.DefaultSchemaGUID = &localProject.Schema
	}

	return apiProject
}

// ConvertConfigSheetFromAPI converts API ConfigSheet to local entities.ConfigSheet
func ConvertConfigSheetFromAPI(apiConfigSheet *ConfigSheet) *entities.ConfigSheet {
	if apiConfigSheet == nil {
		return nil
	}

	localConfigSheet := &entities.ConfigSheet{
		Entity: entities.Entity{
			ID:          apiConfigSheet.GUID, // Use GUID as ID for local storage
			Name:        apiConfigSheet.Name,
			Description: apiConfigSheet.Description,
			Remote:      "true", // Mark as coming from remote (string, not bool)
			Local:       false,
			CreatedAt:   apiConfigSheet.CreatedAt.Time(),
			UpdatedAt:   apiConfigSheet.UpdatedAt.Time(),
		},
		Values:  apiConfigSheet.Variables, // API uses 'variables', local uses 'values'
		Extends: apiConfigSheet.Extends,
		// Schema will need to be set by caller based on schema reference
		Schema: entities.SchemaReference{}, // Empty for now
	}

	return localConfigSheet
}

// ConvertConfigSheetToAPI converts local entities.ConfigSheet to API ConfigSheet
func ConvertConfigSheetToAPI(
	localConfigSheet *entities.ConfigSheet,
	projectGUID, schemaGUID string,
) *ConfigSheet {
	if localConfigSheet == nil {
		return nil
	}

	apiConfigSheet := &ConfigSheet{
		GUID:        localConfigSheet.ID, // Send local UUID as GUID
		Name:        localConfigSheet.Name,
		Description: localConfigSheet.Description,
		ProjectGUID: projectGUID,
		SchemaGUID:  schemaGUID,
		Variables:   localConfigSheet.Values, // Local uses 'values', API uses 'variables'
		Extends:     localConfigSheet.Extends,
		IsActive:    true,
	}

	return apiConfigSheet
}

// ShouldPull determines if a local entity should be updated from remote
// Returns true if remote is newer or local doesn't exist
func ShouldPull(localTime, remoteTime time.Time, force bool) bool {
	if force {
		return true
	}

	// If local doesn't exist (zero time), always pull
	if localTime.IsZero() {
		return true
	}

	// Pull if remote is newer (with some tolerance for clock skew)
	return remoteTime.After(localTime.Add(-time.Second))
}

// ShouldPush determines if a local entity should be pushed to remote
// Returns true if local is newer or remote doesn't exist
func ShouldPush(localTime, remoteTime time.Time, force bool) bool {
	if force {
		return true
	}

	// If remote doesn't exist (zero time), always push
	if remoteTime.IsZero() {
		return true
	}

	// Push if local is newer (with some tolerance for clock skew)
	return localTime.After(remoteTime.Add(-time.Second))
}
