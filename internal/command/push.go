// Package command implements the ee push command for remote synchronization
package command

import (
	"fmt"

	"github.com/n1rna/ee-cli/internal/api"
	"github.com/n1rna/ee-cli/internal/schema"
	"github.com/n1rna/ee-cli/internal/storage"
	"github.com/spf13/cobra"
)

// PushCommand handles the ee push command
type PushCommand struct {
	dryRun     bool
	force      bool
	remote     string
	pushType   string // "project", "schema", or "" (default - current project)
	targetName string // name of specific project/schema to push
}

// NewPushCommand creates a new ee push command
func NewPushCommand() *cobra.Command {
	pc := &PushCommand{}

	cmd := &cobra.Command{
		Use:   "push [type] [name]",
		Short: "Push local changes to remote",
		Long: `Push local changes to remote API.

By default, pushes the current project (requires .ee file).
Can also push specific projects or schemas to any remote.

Examples:
  # Push current project (uses .ee file for remote)
  ee push

  # Push specific project to configured remote
  ee push project my-project

  # Push schema to specific remote
  ee push schema my-schema --remote company@ee.dev

  # Dry run to see what would be pushed
  ee push --dry-run

  # Force push even if remote has newer changes
  ee push --force
`,
		Args: cobra.MaximumNArgs(2),
		RunE: pc.Run,
	}

	cmd.Flags().BoolVar(&pc.dryRun, "dry-run", false,
		"Show what would be pushed without making changes")
	cmd.Flags().BoolVar(&pc.force, "force", false,
		"Force push even if remote has newer changes")
	cmd.Flags().StringVar(&pc.remote, "remote", "",
		"Remote URL to push to (overrides .ee file)")

	return cmd
}

// Run executes the push command
func (pc *PushCommand) Run(cmd *cobra.Command, args []string) error {
	storage := GetStorage(cmd.Context())
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// Parse arguments
	if len(args) >= 1 {
		pc.pushType = args[0]
		if len(args) >= 2 {
			pc.targetName = args[1]
		}
	}

	// Validate push type
	if pc.pushType != "" && pc.pushType != "project" && pc.pushType != "schema" {
		return fmt.Errorf("invalid push type '%s'. Must be 'project' or 'schema'", pc.pushType)
	}

	var remoteURL, projectID string

	// Determine remote URL and target
	if pc.pushType == "" {
		// Default: push current project using .ee file
		if !EasyEnvFileExists("") {
			return fmt.Errorf(".ee file not found. Either create one with 'ee init' or specify explicit push: 'ee push project <name> --remote <url>'")
		}

		menvFile, err := LoadEasyEnvFile("")
		if err != nil {
			return fmt.Errorf("failed to load .ee file: %w", err)
		}

		if menvFile.Remote == "" {
			return fmt.Errorf("no remote configured in .ee file. Use 'ee remote <url>' to configure or use --remote flag")
		}

		remoteURL = menvFile.Remote
		projectID = menvFile.Project
		pc.pushType = "project"

		// Load project name for display
		if project, err := storage.LoadProject(projectID); err == nil {
			pc.targetName = project.Name
		}
	} else {
		// Explicit project/schema push
		if pc.targetName == "" {
			return fmt.Errorf("must specify %s name when using explicit push", pc.pushType)
		}

		if pc.remote != "" {
			remoteURL = pc.remote
		} else if EasyEnvFileExists("") {
			// Try to get remote from .ee file as fallback
			if menvFile, err := LoadEasyEnvFile(""); err == nil && menvFile.Remote != "" {
				remoteURL = menvFile.Remote
			}
		}

		if remoteURL == "" {
			return fmt.Errorf("no remote specified. Use --remote flag or configure in .ee file")
		}
	}

	fmt.Printf("📤 Pushing %s '%s' to remote: %s\n", pc.pushType, pc.targetName, remoteURL)

	if pc.dryRun {
		fmt.Println("🔍 Dry run mode - no changes will be made")
	}
	fmt.Println()

	// Execute push based on type
	switch pc.pushType {
	case "project":
		return pc.pushProject(storage, pc.targetName, remoteURL)
	case "schema":
		return pc.pushSchema(storage, pc.targetName, remoteURL)
	default:
		return fmt.Errorf("unknown push type: %s", pc.pushType)
	}
}

// pushProject pushes a project and its dependencies to remote
func (pc *PushCommand) pushProject(storage *storage.UUIDStorage, projectName, remoteURL string) error {
	// Load the project
	project, err := storage.LoadProject(projectName)
	if err != nil {
		return fmt.Errorf("failed to load project '%s': %w", projectName, err)
	}

	fmt.Printf("📤 Pushing project: %s (ID: %s)\n", project.Entity.Name, project.Entity.ID)

	// Create API client
	client, err := api.ClientFromRemoteURL(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Test API connection
	if err := client.Health(); err != nil {
		return fmt.Errorf("API health check failed: %w", err)
	}

	changes := &pushChanges{}

	// 1. Push all schemas used by project and environments (including dependencies)
	if err := pc.pushAllProjectSchemas(client, storage, project, changes); err != nil {
		return fmt.Errorf("failed to push project schemas and dependencies: %w", err)
	}

	// 2. Push project metadata
	if err := pc.pushProjectMetadata(client, storage, project, changes); err != nil {
		return fmt.Errorf("failed to push project metadata: %w", err)
	}

	// 3. Push config sheets for all environments
	if err := pc.pushProjectConfigSheets(client, storage, project, changes); err != nil {
		return fmt.Errorf("failed to push config sheets: %w", err)
	}

	// Display what would be changed in dry-run mode
	if pc.dryRun {
		return pc.displayPushDryRunResults(changes)
	}

	// Apply changes (update local Remote flags)
	return pc.applyPushChanges(storage, changes, remoteURL)
}

// pushChanges tracks what changes would be made during push
type pushChanges struct {
	schemasCreated      []string
	schemasUpdated      []string
	projectCreated      bool
	projectUpdated      bool
	configSheetsCreated []string
	configSheetsUpdated []string
}

// pushProjectSchema pushes the project's schema if it's local
func (pc *PushCommand) pushProjectSchema(client *api.Client, storage *storage.UUIDStorage, project *schema.Project, changes *pushChanges) error {
	localSchema, err := storage.LoadSchema(project.Schema)
	if err != nil {
		// Schema might not exist locally, which is okay for references
		return nil
	}

	// Check if this is a local schema that needs to be pushed
	if localSchema.Entity.Remote == "" || localSchema.Entity.Remote == "false" {
		// Check if schema already exists on remote
		remoteSchemas, err := client.ListSchemas()
		if err != nil {
			return fmt.Errorf("failed to list remote schemas: %w", err)
		}

		var existingSchema *api.Schema
		for _, remoteSchema := range remoteSchemas {
			if remoteSchema.Name == localSchema.Entity.Name {
				existingSchema = &remoteSchema
				break
			}
		}

		if existingSchema != nil {
			// Check if we should update
			if api.ShouldPush(localSchema.Entity.UpdatedAt, existingSchema.UpdatedAt.Time(), pc.force) {
				changes.schemasUpdated = append(changes.schemasUpdated, localSchema.Entity.Name)

				if !pc.dryRun {
					// Update existing schema
					apiSchema := api.ConvertSchemaToAPI(localSchema)
					_, err := client.UpdateSchema(existingSchema.GUID, apiSchema)
					if err != nil {
						return fmt.Errorf("failed to update schema '%s': %w", localSchema.Entity.Name, err)
					}

					fmt.Printf("✅ Updated schema '%s'\n", localSchema.Entity.Name)
				}
			}
		} else {
			// Create new schema
			changes.schemasCreated = append(changes.schemasCreated, localSchema.Entity.Name)

			if !pc.dryRun {
				apiSchema := api.ConvertSchemaToAPI(localSchema)
				_, err := client.CreateSchema(apiSchema)
				if err != nil {
					return fmt.Errorf("failed to create schema '%s': %w", localSchema.Entity.Name, err)
				}

				fmt.Printf("✅ Created schema '%s'\n", localSchema.Entity.Name)
			}
		}
	}

	return nil
}

// pushAllProjectSchemas pushes all schemas used by the project and its environments (including dependencies)
func (pc *PushCommand) pushAllProjectSchemas(client *api.Client, storage *storage.UUIDStorage, project *schema.Project, changes *pushChanges) error {
	// Get remote schemas for dependency checking
	remoteSchemas, err := client.ListSchemas()
	if err != nil {
		return fmt.Errorf("failed to list remote schemas: %w", err)
	}

	// Create a map of existing remote schemas for quick lookup
	remoteSchemaMap := make(map[string]*api.Schema)
	for i, remoteSchema := range remoteSchemas {
		remoteSchemaMap[remoteSchema.Name] = &remoteSchemas[i]
	}

	// Track which schemas we've processed to avoid cycles
	processed := make(map[string]bool)

	// Collect all schema names that need to be pushed
	schemasToPush := make(map[string]bool)

	// 1. Add project schema if it exists
	if project.Schema != "" {
		schemasToPush[project.Schema] = true
	}

	// 2. Scan all config sheets for schema references
	for _, env := range project.Environments {
		// Derive config sheet name using naming convention
		configSheetName := project.GetConfigSheetName(env.Name)

		configSheet, err := storage.LoadConfigSheet(configSheetName)
		if err != nil {
			fmt.Printf("⚠️  Skipping environment '%s': config sheet '%s' not found locally\n", env.Name, configSheetName)
			continue
		}

		// Extract schema name from config sheet reference
		if configSheet.Schema.Ref != "" {
			if len(configSheet.Schema.Ref) > 10 && configSheet.Schema.Ref[:10] == "#/schemas/" {
				schemaID := configSheet.Schema.Ref[10:]
				// Try to load local schema to get its name
				if localSchema, err := storage.LoadSchema(schemaID); err == nil {
					schemasToPush[localSchema.Entity.Name] = true
				}
			}
		}
	}

	// 3. Recursively push all discovered schemas and their dependencies
	for schemaName := range schemasToPush {
		if err := pc.pushSchemaRecursively(client, storage, schemaName, remoteSchemaMap, processed, changes); err != nil {
			return fmt.Errorf("failed to push schema '%s': %w", schemaName, err)
		}
	}

	return nil
}

// pushSchemaRecursively pushes a schema and all its dependencies recursively
func (pc *PushCommand) pushSchemaRecursively(client *api.Client, storage *storage.UUIDStorage, schemaName string, remoteSchemaMap map[string]*api.Schema, processed map[string]bool, changes *pushChanges) error {
	// Skip if already processed (avoid cycles)
	if processed[schemaName] {
		return nil
	}

	// Load the local schema
	localSchema, err := storage.LoadSchema(schemaName)
	if err != nil {
		// Schema might not exist locally, which is okay for remote references
		return nil
	}

	// Mark as processed
	processed[schemaName] = true

	// First, recursively push all dependencies (extended schemas)
	for _, extendedSchemaName := range localSchema.Extends {
		if err := pc.pushSchemaRecursively(client, storage, extendedSchemaName, remoteSchemaMap, processed, changes); err != nil {
			return fmt.Errorf("failed to push dependency schema '%s': %w", extendedSchemaName, err)
		}
	}

	// Now push this schema if it's local and needs to be pushed
	if localSchema.Entity.Remote == "" || localSchema.Entity.Remote == "false" {
		existingSchema := remoteSchemaMap[localSchema.Entity.Name]

		if existingSchema != nil {
			// Check if we should update
			if api.ShouldPush(localSchema.Entity.UpdatedAt, existingSchema.UpdatedAt.Time(), pc.force) {
				changes.schemasUpdated = append(changes.schemasUpdated, localSchema.Entity.Name)

				if !pc.dryRun {
					// Update existing schema
					apiSchema := api.ConvertSchemaToAPI(localSchema)
					_, err := client.UpdateSchema(existingSchema.GUID, apiSchema)
					if err != nil {
						return fmt.Errorf("failed to update schema '%s': %w", localSchema.Entity.Name, err)
					}

					fmt.Printf("✅ Updated schema '%s'\n", localSchema.Entity.Name)

					// Update remote map for subsequent operations
					if updatedSchema, err := client.GetSchema(existingSchema.GUID); err == nil {
						remoteSchemaMap[localSchema.Entity.Name] = updatedSchema
					}
				}
			}
		} else {
			// Create new schema
			changes.schemasCreated = append(changes.schemasCreated, localSchema.Entity.Name)

			if !pc.dryRun {
				apiSchema := api.ConvertSchemaToAPI(localSchema)
				createdSchema, err := client.CreateSchema(apiSchema)
				if err != nil {
					return fmt.Errorf("failed to create schema '%s': %w", localSchema.Entity.Name, err)
				}

				fmt.Printf("✅ Created schema '%s'\n", localSchema.Entity.Name)

				// Update remote map for subsequent operations
				remoteSchemaMap[localSchema.Entity.Name] = createdSchema
			}
		}
	}

	return nil
}

// pushProjectMetadata pushes project metadata
func (pc *PushCommand) pushProjectMetadata(client *api.Client, storage *storage.UUIDStorage, project *schema.Project, changes *pushChanges) error {
	// Check if project exists on remote
	remoteProjects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list remote projects: %w", err)
	}

	var existingProject *api.Project
	for _, remoteProject := range remoteProjects {
		if remoteProject.Name == project.Entity.Name {
			existingProject = &remoteProject
			break
		}
	}

	if existingProject != nil {
		// Check if we should update
		if api.ShouldPush(project.Entity.UpdatedAt, existingProject.UpdatedAt.Time(), pc.force) {
			changes.projectUpdated = true

			if !pc.dryRun {
				// Update existing project
				apiProject := api.ConvertProjectToAPI(project)
				_, err := client.UpdateProject(existingProject.GUID, apiProject)
				if err != nil {
					return fmt.Errorf("failed to update project '%s': %w", project.Entity.Name, err)
				}

				fmt.Printf("✅ Updated project '%s'\n", project.Entity.Name)
			}
		}
	} else {
		// Create new project
		changes.projectCreated = true

		if !pc.dryRun {
			apiProject := api.ConvertProjectToAPI(project)
			_, err := client.CreateProject(apiProject)
			if err != nil {
				return fmt.Errorf("failed to create project '%s': %w", project.Entity.Name, err)
			}

			fmt.Printf("✅ Created project '%s'\n", project.Entity.Name)
		}
	}

	return nil
}

// pushProjectConfigSheets pushes all config sheets for project environments
func (pc *PushCommand) pushProjectConfigSheets(client *api.Client, storage *storage.UUIDStorage, project *schema.Project, changes *pushChanges) error {
	// Use project's local UUID as GUID for API operations
	projectGUID := project.Entity.ID

	// Get remote config sheets for comparison
	remoteConfigSheets, err := client.ListConfigSheetsByProject(projectGUID, true)
	if err != nil {
		return fmt.Errorf("failed to list remote config sheets: %w", err)
	}

	// Push config sheet for each environment
	for _, env := range project.Environments {
		// Derive config sheet name using naming convention
		configSheetName := project.GetConfigSheetName(env.Name)

		localConfigSheet, err := storage.LoadConfigSheet(configSheetName)
		if err != nil {
			fmt.Printf("⚠️  Skipping environment '%s': config sheet '%s' not found locally\n", env.Name, configSheetName)
			continue
		}

		// Check if config sheet exists on remote
		var existingConfigSheet *api.ConfigSheet
		for _, remoteSheet := range remoteConfigSheets {
			if remoteSheet.Name == localConfigSheet.Entity.Name {
				existingConfigSheet = &remoteSheet
				break
			}
		}

		if existingConfigSheet != nil {
			// Check if we should update
			if api.ShouldPush(localConfigSheet.Entity.UpdatedAt, existingConfigSheet.UpdatedAt.Time(), pc.force) {
				changes.configSheetsUpdated = append(changes.configSheetsUpdated, localConfigSheet.Entity.Name)

				if !pc.dryRun {
					// Update existing config sheet
					updates := map[string]interface{}{
						"variables": localConfigSheet.Values,
						"extends":   localConfigSheet.Extends,
					}
					_, err := client.UpdateConfigSheet(existingConfigSheet.GUID, updates)
					if err != nil {
						return fmt.Errorf("failed to update config sheet '%s': %w", localConfigSheet.Entity.Name, err)
					}

					fmt.Printf("✅ Updated config sheet '%s'\n", localConfigSheet.Entity.Name)
				}
			}
		} else {
			// Create new config sheet
			changes.configSheetsCreated = append(changes.configSheetsCreated, localConfigSheet.Entity.Name)

			if !pc.dryRun {
				// Resolve schema GUID from config sheet's schema reference
				schemaGUID, err := pc.resolveSchemaGUID(client, localConfigSheet, storage)
				if err != nil {
					return fmt.Errorf("failed to resolve schema GUID for config sheet '%s': %w", localConfigSheet.Entity.Name, err)
				}

				// Convert to API format and create
				apiConfigSheet := api.ConvertConfigSheetToAPI(localConfigSheet, projectGUID, schemaGUID)
				_, err = client.CreateConfigSheet(apiConfigSheet)
				if err != nil {
					return fmt.Errorf("failed to create config sheet '%s': %w", localConfigSheet.Entity.Name, err)
				}

				fmt.Printf("✅ Created config sheet '%s'\n", localConfigSheet.Entity.Name)
			}
		}
	}

	return nil
}

// resolveSchemaGUID resolves the schema GUID from a config sheet's schema reference
func (pc *PushCommand) resolveSchemaGUID(client *api.Client, configSheet *schema.ConfigSheet, storage *storage.UUIDStorage) (string, error) {
	// Check if config sheet has inline schema (no reference)
	if configSheet.Schema.Ref == "" {
		// For project config sheets, use the project's schema instead of allowing inline
		if configSheet.IsProjectEnvironment() {
			project, err := storage.LoadProject(configSheet.Project)
			if err != nil {
				return "", fmt.Errorf("failed to load project for config sheet '%s': %w", configSheet.Entity.Name, err)
			}

			if project.Schema == "" {
				return "", fmt.Errorf("project '%s' has no schema defined, cannot create config sheet without schema", project.Entity.Name)
			}

			// Use the project's schema
			projectSchemaID := project.Schema
			localSchema, err := storage.LoadSchema(projectSchemaID)
			if err != nil {
				return "", fmt.Errorf("project schema '%s' not found locally: %w", projectSchemaID, err)
			}

			// Find the corresponding schema on remote by name
			remoteSchemas, err := client.ListSchemas()
			if err != nil {
				return "", fmt.Errorf("failed to list remote schemas: %w", err)
			}

			// Find schema by name
			for _, schema := range remoteSchemas {
				if schema.Name == localSchema.Entity.Name {
					return schema.GUID, nil
				}
			}

			return "", fmt.Errorf("project schema '%s' not found on remote", localSchema.Entity.Name)
		}

		// For standalone sheets, allow empty string (inline schema)
		return "", nil
	}

	// Extract schema ID from reference (format: "#/schemas/{schema-id}")
	schemaRef := configSheet.Schema.Ref
	// Debug: fmt.Printf("Debug: schema reference = '%s', length = %d\n", schemaRef, len(schemaRef))
	if len(schemaRef) > 10 && schemaRef[:10] == "#/schemas/" {
		schemaID := schemaRef[10:] // Remove "#/schemas/" prefix

		// Load the local schema by ID (works with both UUID and name)
		localSchema, err := storage.LoadSchema(schemaID)
		if err != nil {
			return "", fmt.Errorf("local schema '%s' not found: %w", schemaID, err)
		}

		// Find the corresponding schema on remote by name
		remoteSchemas, err := client.ListSchemas()
		if err != nil {
			return "", fmt.Errorf("failed to list remote schemas: %w", err)
		}

		// Find schema by name
		var remoteSchema *api.Schema
		for _, schema := range remoteSchemas {
			if schema.Name == localSchema.Entity.Name {
				remoteSchema = &schema
				break
			}
		}

		if remoteSchema == nil {
			return "", fmt.Errorf("schema '%s' not found on remote (it may need to be pushed first)", localSchema.Entity.Name)
		}

		return remoteSchema.GUID, nil
	}

	return "", fmt.Errorf("invalid schema reference format: %s", schemaRef)
}

// displayPushDryRunResults shows what would be changed
func (pc *PushCommand) displayPushDryRunResults(changes *pushChanges) error {
	fmt.Println("🔍 Dry run - would make the following changes:")

	hasChanges := false

	for _, name := range changes.schemasCreated {
		fmt.Printf("  ➕ Create schema '%s' on remote\n", name)
		hasChanges = true
	}

	for _, name := range changes.schemasUpdated {
		fmt.Printf("  📝 Update schema '%s' on remote\n", name)
		hasChanges = true
	}

	if changes.projectCreated {
		fmt.Println("  ➕ Create project on remote")
		hasChanges = true
	}

	if changes.projectUpdated {
		fmt.Println("  📝 Update project metadata on remote")
		hasChanges = true
	}

	for _, name := range changes.configSheetsCreated {
		fmt.Printf("  ➕ Create config sheet '%s' on remote\n", name)
		hasChanges = true
	}

	for _, name := range changes.configSheetsUpdated {
		fmt.Printf("  📝 Update config sheet '%s' on remote\n", name)
		hasChanges = true
	}

	if !hasChanges {
		fmt.Println("  ✨ No changes needed - remote is up to date")
	}

	return nil
}

// applyPushChanges applies push changes and updates local Remote flags
func (pc *PushCommand) applyPushChanges(storage *storage.UUIDStorage, changes *pushChanges, remoteURL string) error {
	totalChanges := 0
	totalChanges += len(changes.schemasCreated) + len(changes.schemasUpdated)
	if changes.projectCreated || changes.projectUpdated {
		totalChanges++
	}
	totalChanges += len(changes.configSheetsCreated) + len(changes.configSheetsUpdated)

	if totalChanges > 0 {
		fmt.Printf("\n🎉 Successfully pushed %d changes to remote\n", totalChanges)

		// TODO: Update local entities' Remote field to track they're now on remote
		fmt.Printf("📍 Remote URL: %s\n", remoteURL)
	} else {
		fmt.Println("\n✨ No changes needed - remote is up to date")
	}

	return nil
}

// pushSchema pushes a schema to remote
func (pc *PushCommand) pushSchema(storage *storage.UUIDStorage, schemaName, remoteURL string) error {
	// Load the schema
	schemaObj, err := storage.LoadSchema(schemaName)
	if err != nil {
		return fmt.Errorf("failed to load schema '%s': %w", schemaName, err)
	}

	fmt.Printf("📤 Pushing schema: %s (ID: %s)\n", schemaObj.Entity.Name, schemaObj.Entity.ID)

	// Create API client
	client, err := api.ClientFromRemoteURL(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Test API connection
	if err := client.Health(); err != nil {
		return fmt.Errorf("API health check failed: %w", err)
	}

	// Check if schema extends other schemas and validate dependencies
	if len(schemaObj.Extends) > 0 {
		if err := pc.validateSchemaDependencies(client, schemaObj); err != nil {
			return fmt.Errorf("schema dependency validation failed: %w", err)
		}
	}

	// Check if schema exists on remote
	remoteSchemas, err := client.ListSchemas()
	if err != nil {
		return fmt.Errorf("failed to list remote schemas: %w", err)
	}

	var existingSchema *api.Schema
	for _, remoteSchema := range remoteSchemas {
		if remoteSchema.Name == schemaObj.Entity.Name {
			existingSchema = &remoteSchema
			break
		}
	}

	if pc.dryRun {
		fmt.Println("Would push:")
		fmt.Println("  - Schema definition and variables")
		fmt.Println("  - Schema metadata and inheritance chain")

		// Check if schema extends other schemas
		if len(schemaObj.Extends) > 0 {
			fmt.Println("  - Note: Extended schemas must exist on remote")
			for _, extendedSchema := range schemaObj.Extends {
				fmt.Printf("    - Depends on schema: %s\n", extendedSchema)
			}
		}

		if existingSchema != nil {
			fmt.Println("  - Would update existing schema on remote")
		} else {
			fmt.Println("  - Would create new schema on remote")
		}

		return nil
	}

	if existingSchema != nil {
		// Check if we should update
		if api.ShouldPush(schemaObj.Entity.UpdatedAt, existingSchema.UpdatedAt.Time(), pc.force) {
			// Update existing schema
			apiSchema := api.ConvertSchemaToAPI(schemaObj)
			_, err := client.UpdateSchema(existingSchema.GUID, apiSchema)
			if err != nil {
				return fmt.Errorf("failed to update schema '%s': %w", schemaObj.Entity.Name, err)
			}

			fmt.Printf("✅ Updated schema '%s' on remote\n", schemaObj.Entity.Name)

			// TODO: Update local entity's Remote field
			fmt.Printf("📍 Remote URL: %s\n", remoteURL)
		} else {
			fmt.Printf("✨ Schema '%s' is already up to date on remote\n", schemaObj.Entity.Name)
		}
	} else {
		// Create new schema
		apiSchema := api.ConvertSchemaToAPI(schemaObj)
		_, err := client.CreateSchema(apiSchema)
		if err != nil {
			return fmt.Errorf("failed to create schema '%s': %w", schemaObj.Entity.Name, err)
		}

		fmt.Printf("✅ Created schema '%s' on remote\n", schemaObj.Entity.Name)

		// TODO: Update local entity's Remote field
		fmt.Printf("📍 Remote URL: %s\n", remoteURL)
	}

	return nil
}

// validateSchemaDependencies checks that all extended schemas exist on remote
func (pc *PushCommand) validateSchemaDependencies(client *api.Client, schemaObj *schema.Schema) error {
	if len(schemaObj.Extends) == 0 {
		return nil
	}

	remoteSchemas, err := client.ListSchemas()
	if err != nil {
		return fmt.Errorf("failed to list remote schemas: %w", err)
	}

	// Create a set of remote schema names for quick lookup
	remoteSchemaNames := make(map[string]bool)
	for _, remoteSchema := range remoteSchemas {
		remoteSchemaNames[remoteSchema.Name] = true
	}

	// Check each dependency
	var missingDeps []string
	for _, extendedSchema := range schemaObj.Extends {
		if !remoteSchemaNames[extendedSchema] {
			missingDeps = append(missingDeps, extendedSchema)
		}
	}

	if len(missingDeps) > 0 {
		return fmt.Errorf("schema '%s' depends on schemas that don't exist on remote: %v", schemaObj.Entity.Name, missingDeps)
	}

	return nil
}
