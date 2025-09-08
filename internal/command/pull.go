// Package command implements the ee pull command for remote synchronization
package command

import (
	"fmt"

	"github.com/n1rna/ee-cli/internal/api"
	"github.com/n1rna/ee-cli/internal/schema"
	"github.com/n1rna/ee-cli/internal/storage"
	"github.com/spf13/cobra"
)

// PullCommand handles the ee pull command
type PullCommand struct {
	dryRun bool
	force  bool
}

// NewPullCommand creates a new ee pull command
func NewPullCommand(groupId string) *cobra.Command {
	pc := &PullCommand{}

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull changes from remote to local project",
		Long: `Pull changes from remote API to local project.

This command operates within a project context and requires a .ee file with a remote URL
configured in the current directory. It fetches the latest project data, schemas, and
config sheets from the remote and applies them locally.

Examples:
  # Pull changes from remote
  ee pull

  # Dry run to see what would be pulled
  ee pull --dry-run

  # Force pull even if there are local changes
  ee pull --force
`,
		RunE:    pc.Run,
		GroupID: groupId,
	}

	cmd.Flags().BoolVar(&pc.dryRun, "dry-run", false,
		"Show what would be pulled without making changes")
	cmd.Flags().BoolVar(&pc.force, "force", false,
		"Force pull even if there are local changes that would be overwritten")

	return cmd
}

// Run executes the pull command
func (pc *PullCommand) Run(cmd *cobra.Command, args []string) error {
	storage := GetStorage(cmd.Context())
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// Check if we're in a project directory with .ee file
	if !EasyEnvFileExists("") {
		return fmt.Errorf(".ee file not found in current directory. Run 'ee init' first or cd to a project directory")
	}

	// Load .ee file to get remote URL and project ID
	menvFile, err := LoadEasyEnvFile("")
	if err != nil {
		return fmt.Errorf("failed to load .ee file: %w", err)
	}

	if menvFile.Remote == "" {
		return fmt.Errorf("no remote URL configured in .ee file. Use 'ee remote <url>' to configure")
	}

	if menvFile.Project == "" {
		return fmt.Errorf("no project ID found in .ee file")
	}

	fmt.Printf("📥 Pulling from remote: %s\n", menvFile.Remote)
	fmt.Printf("Project ID: %s\n", menvFile.Project)

	if pc.dryRun {
		fmt.Println("🔍 Dry run mode - no changes will be made")
	}
	fmt.Println()

	// Try to load project locally first
	var project *schema.Project

	project, err = storage.LoadProject(menvFile.Project)
	if err != nil {
		// Project doesn't exist locally, we'll fetch it from remote
		fmt.Println("📥 Project not found locally, fetching from remote...")
	}

	// Execute pull (will handle both cases: existing local project or fetch from remote)
	return pc.pullFromRemote(storage, project, menvFile.Project, menvFile.Remote)
}

// pullFromRemote pulls changes from the remote API
func (pc *PullCommand) pullFromRemote(storage *storage.UUIDStorage, project *schema.Project, projectID, remoteURL string) error {
	fmt.Println("📥 Pulling changes from remote...")

	// Create API client
	client, err := api.ClientFromRemoteURL(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Test API connection
	if err := client.Health(); err != nil {
		return fmt.Errorf("API health check failed: %w", err)
	}

	// Handle case where project doesn't exist locally
	var projectGUID string
	if project != nil {
		projectGUID = project.ID
	} else {
		projectGUID = projectID
	}

	changes := &pullChanges{}

	// 1. Pull project metadata (will create locally if missing)
	project, err = pc.pullProject(client, storage, project, projectGUID, changes)
	if err != nil {
		return fmt.Errorf("failed to pull project: %w", err)
	}

	// 2. Pull project schema if it has one
	if project.Schema != "" {
		if err := pc.pullProjectSchema(client, storage, project, changes); err != nil {
			return fmt.Errorf("failed to pull project schema: %w", err)
		}
	}

	// 3. Pull config sheets for all environments
	if err := pc.pullConfigSheets(client, storage, project, projectGUID, changes); err != nil {
		return fmt.Errorf("failed to pull config sheets: %w", err)
	}

	// Display what would be changed in dry-run mode
	if pc.dryRun {
		return pc.displayDryRunResults(changes)
	}

	// Apply changes
	return pc.applyChanges(storage, changes)
}

// pullChanges tracks what changes would be made during pull
type pullChanges struct {
	projectUpdated      bool
	schemaUpdated       bool
	configSheetsNew     []string
	configSheetsUpdated []string
}

// No longer needed - we use GUIDs directly

// pullProject pulls and updates project metadata, returns the project (created or updated)
func (pc *PullCommand) pullProject(client *api.Client, storage *storage.UUIDStorage, project *schema.Project, projectGUID string, changes *pullChanges) (*schema.Project, error) {
	remoteProject, err := client.GetProject(projectGUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project from API: %w", err)
	}

	if project == nil {
		// Project doesn't exist locally, create it
		fmt.Printf("📦 Creating project locally: %s\n", remoteProject.Name)
		changes.projectUpdated = true

		if !pc.dryRun {
			// Convert from API and save locally
			project = api.ConvertProjectFromAPI(remoteProject)
			// Use the same ID as the remote GUID for consistency
			project.ID = remoteProject.GUID

			// Handle project's default schema if it has one
			if remoteProject.DefaultSchemaGUID != nil && *remoteProject.DefaultSchemaGUID != "" {
				_, err := pc.ensureSchemaExists(client, storage, *remoteProject.DefaultSchemaGUID)
				if err != nil {
					return nil, fmt.Errorf("failed to ensure project schema exists: %w", err)
				}
				// Set the schema reference using the GUID directly (local storage uses GUIDs)
				project.Schema = *remoteProject.DefaultSchemaGUID
			}

			if err := storage.SaveProject(project); err != nil {
				return nil, fmt.Errorf("failed to save new project: %w", err)
			}

			fmt.Printf("✅ Created project '%s' locally\n", project.Entity.Name)
		} else {
			// In dry-run mode, just return a mock project for further processing
			project = api.ConvertProjectFromAPI(remoteProject)
			project.ID = remoteProject.GUID

			// Handle project's default schema in dry-run mode too
			if remoteProject.DefaultSchemaGUID != nil && *remoteProject.DefaultSchemaGUID != "" {
				_, err := pc.ensureSchemaExists(client, storage, *remoteProject.DefaultSchemaGUID)
				if err != nil {
					return nil, fmt.Errorf("failed to ensure project schema exists: %w", err)
				}
				// Set the schema reference using the GUID directly (local storage uses GUIDs)
				project.Schema = *remoteProject.DefaultSchemaGUID
			}
		}
	} else {
		// Project exists locally, check if we should update
		if api.ShouldPull(project.Entity.UpdatedAt, remoteProject.UpdatedAt.Time(), pc.force) {
			changes.projectUpdated = true

			if !pc.dryRun {
				// Convert and update project
				updatedProject := api.ConvertProjectFromAPI(remoteProject)
				updatedProject.ID = project.ID                     // Preserve local ID
				updatedProject.Environments = project.Environments // Preserve environment mappings

				// Handle project's default schema if it has one
				if remoteProject.DefaultSchemaGUID != nil && *remoteProject.DefaultSchemaGUID != "" {
					_, err := pc.ensureSchemaExists(client, storage, *remoteProject.DefaultSchemaGUID)
					if err != nil {
						return nil, fmt.Errorf("failed to ensure project schema exists: %w", err)
					}
					// Set the schema reference using the GUID directly (local storage uses GUIDs)
					updatedProject.Schema = *remoteProject.DefaultSchemaGUID
				}

				if err := storage.SaveProject(updatedProject); err != nil {
					return nil, fmt.Errorf("failed to save updated project: %w", err)
				}

				project = updatedProject // Update our reference
				fmt.Printf("✅ Updated project '%s'\n", project.Entity.Name)
			}
		} else {
			fmt.Printf("✨ Project '%s' is already up to date\n", project.Entity.Name)
		}
	}

	return project, nil
}

// pullProjectSchema pulls the project's default schema
func (pc *PullCommand) pullProjectSchema(client *api.Client, storage *storage.UUIDStorage, project *schema.Project, changes *pullChanges) error {
	// Get all schemas and find the one matching our project schema
	schemas, err := client.ListSchemas()
	if err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}

	for _, remoteSchema := range schemas {
		if remoteSchema.Name == project.Schema {
			// Check if we have this schema locally
			localSchema, err := storage.LoadSchema(project.Schema)

			shouldUpdate := false
			if err != nil {
				// Schema doesn't exist locally
				shouldUpdate = true
			} else {
				// Check if remote is newer
				shouldUpdate = api.ShouldPull(localSchema.UpdatedAt, remoteSchema.UpdatedAt.Time(), pc.force)
			}

			if shouldUpdate {
				changes.schemaUpdated = true

				if !pc.dryRun {
					updatedSchema := api.ConvertSchemaFromAPI(&remoteSchema)
					if err := storage.SaveSchema(updatedSchema); err != nil {
						return fmt.Errorf("failed to save schema '%s': %w", remoteSchema.Name, err)
					}

					fmt.Printf("✅ Updated schema '%s'\n", remoteSchema.Name)
				}
			}
			break
		}
	}

	return nil
}

// pullConfigSheets pulls all config sheets for the project's environments
func (pc *PullCommand) pullConfigSheets(client *api.Client, storage *storage.UUIDStorage, project *schema.Project, projectGUID string, changes *pullChanges) error {
	// Get config sheets for this project from API
	configSheets, err := client.ListConfigSheetsByProject(projectGUID, true)
	if err != nil {
		return fmt.Errorf("failed to list config sheets: %w", err)
	}

	for _, remoteConfigSheet := range configSheets {
		// Check if we have this config sheet locally
		localConfigSheet, err := storage.LoadConfigSheet(remoteConfigSheet.Name)

		shouldUpdate := false
		isNew := false

		if err != nil {
			// Config sheet doesn't exist locally
			shouldUpdate = true
			isNew = true
		} else {
			// Check if remote is newer
			shouldUpdate = api.ShouldPull(localConfigSheet.UpdatedAt, remoteConfigSheet.UpdatedAt.Time(), pc.force)
		}

		if shouldUpdate {
			if isNew {
				changes.configSheetsNew = append(changes.configSheetsNew, remoteConfigSheet.Name)
			} else {
				changes.configSheetsUpdated = append(changes.configSheetsUpdated, remoteConfigSheet.Name)
			}

			if !pc.dryRun {
				updatedConfigSheet := api.ConvertConfigSheetFromAPI(&remoteConfigSheet)

				// Handle schema reference if config sheet has a schema GUID
				if remoteConfigSheet.SchemaGUID != "" {
					schemaName, err := pc.ensureSchemaExists(client, storage, remoteConfigSheet.SchemaGUID)
					if err != nil {
						return fmt.Errorf("failed to ensure schema exists for config sheet '%s': %w", remoteConfigSheet.Name, err)
					}
					// Set the schema reference to use the schema name
					updatedConfigSheet.Schema.Ref = schemaName
				}

				if err := storage.SaveConfigSheet(updatedConfigSheet); err != nil {
					return fmt.Errorf("failed to save config sheet '%s': %w", remoteConfigSheet.Name, err)
				}

				if isNew {
					fmt.Printf("✅ Created config sheet '%s'\n", remoteConfigSheet.Name)
				} else {
					fmt.Printf("✅ Updated config sheet '%s'\n", remoteConfigSheet.Name)
				}
			}
		}
	}

	return nil
}

// displayDryRunResults shows what would be changed
func (pc *PullCommand) displayDryRunResults(changes *pullChanges) error {
	fmt.Println("🔍 Dry run - would make the following changes:")

	hasChanges := false

	if changes.projectUpdated {
		fmt.Println("  📝 Update project metadata")
		hasChanges = true
	}

	if changes.schemaUpdated {
		fmt.Println("  📋 Update project schema")
		hasChanges = true
	}

	for _, name := range changes.configSheetsNew {
		fmt.Printf("  ➕ Create config sheet '%s'\n", name)
		hasChanges = true
	}

	for _, name := range changes.configSheetsUpdated {
		fmt.Printf("  📝 Update config sheet '%s'\n", name)
		hasChanges = true
	}

	if !hasChanges {
		fmt.Println("  ✨ No changes needed - everything is up to date")
	}

	return nil
}

// applyChanges applies all the accumulated changes
func (pc *PullCommand) applyChanges(storage *storage.UUIDStorage, changes *pullChanges) error {
	totalChanges := 0
	if changes.projectUpdated {
		totalChanges++
	}
	if changes.schemaUpdated {
		totalChanges++
	}
	totalChanges += len(changes.configSheetsNew) + len(changes.configSheetsUpdated)

	if totalChanges > 0 {
		fmt.Printf("\n🎉 Successfully pulled %d changes from remote\n", totalChanges)
	} else {
		fmt.Println("\n✨ No changes needed - everything is up to date")
	}

	return nil
}

// ensureSchemaExists checks if a schema exists locally and pulls it if not, returns schema name
func (pc *PullCommand) ensureSchemaExists(client *api.Client, storage *storage.UUIDStorage, schemaGUID string) (string, error) {
	// First, get the schema from the API to find its name
	remoteSchema, err := client.GetSchema(schemaGUID)
	if err != nil {
		return "", fmt.Errorf("failed to get schema from API: %w", err)
	}

	// Check if we already have this schema locally
	localSchema, err := storage.LoadSchema(remoteSchema.Name)
	if err != nil {
		// Schema doesn't exist locally, create it
		fmt.Printf("📋 Creating schema '%s' locally\n", remoteSchema.Name)

		convertedSchema := api.ConvertSchemaFromAPI(remoteSchema)
		if err := storage.SaveSchema(convertedSchema); err != nil {
			return "", fmt.Errorf("failed to save schema '%s': %w", remoteSchema.Name, err)
		}

		fmt.Printf("✅ Created schema '%s'\n", remoteSchema.Name)
	} else {
		// Check if we should update it
		if api.ShouldPull(localSchema.UpdatedAt, remoteSchema.UpdatedAt.Time(), pc.force) {
			fmt.Printf("📋 Updating schema '%s'\n", remoteSchema.Name)

			convertedSchema := api.ConvertSchemaFromAPI(remoteSchema)
			if err := storage.SaveSchema(convertedSchema); err != nil {
				return "", fmt.Errorf("failed to save schema '%s': %w", remoteSchema.Name, err)
			}

			fmt.Printf("✅ Updated schema '%s'\n", remoteSchema.Name)
		}
	}

	return remoteSchema.Name, nil
}
