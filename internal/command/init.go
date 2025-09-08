// Package command implements the ee init command for project initialization
// This implements the new project-based workflow as specified in docs/entities.md
package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/n1rna/ee-cli/internal/schema"
	"github.com/n1rna/ee-cli/internal/storage"
	"github.com/spf13/cobra"
)

// InitCommand handles the ee init command
type InitCommand struct {
	projectName string
	schemaName  string
	remote      string
	force       bool
}

// NewInitCommand creates a new ee init command
func NewInitCommand(groupId string) *cobra.Command {
	ic := &InitCommand{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a local ee project configuration",
		Long: `Initialize a local ee project configuration by creating a .ee file.

This command creates a .ee file in the current working directory and registers
the project in your local ee storage. If the project doesn't exist, it will
be created with a default schema.

Examples:
  # Initialize with project name from current directory
  ee init

  # Initialize with specific project name
  ee init --project my-api

  # Initialize with specific project and schema
  ee init --project my-api --schema api-service

  # Initialize with remote URL
  ee init --project my-api --remote https://api.ee.dev
`,
		RunE:    ic.Run,
		GroupID: groupId,
	}

	cmd.Flags().StringVar(&ic.projectName, "project", "",
		"Project name (defaults to current directory name)")
	cmd.Flags().StringVar(&ic.schemaName, "schema", "",
		"Schema name to use for new projects (defaults to 'default')")
	cmd.Flags().StringVar(&ic.remote, "remote", "",
		"Remote API base URL")
	cmd.Flags().BoolVar(&ic.force, "force", false,
		"Overwrite existing .ee file if it exists")

	return cmd
}

// Run executes the init command
func (ic *InitCommand) Run(cmd *cobra.Command, args []string) error {
	storage := GetStorage(cmd.Context())
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// Check if .ee file already exists
	if EasyEnvFileExists("") && !ic.force {
		return fmt.Errorf(".ee file already exists in current directory. Use --force to overwrite")
	}

	// Determine project name
	projectName := ic.projectName
	if projectName == "" {
		// Use current directory name as project name
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projectName = filepath.Base(cwd)

		// Sanitize project name
		projectName = sanitizeProjectName(projectName)
		fmt.Printf("Using directory name as project name: %s\n", projectName)
	}

	// Check if project already exists
	var project *schema.Project
	var err error

	if storage.EntityExists("projects", projectName) {
		// Load existing project
		project, err = storage.LoadProject(projectName)
		if err != nil {
			return fmt.Errorf("failed to load existing project: %w", err)
		}
		fmt.Printf("Using existing project: %s (%s)\n", project.Name, project.ID)

		// If schema is specified but different from existing, show warning
		if ic.schemaName != "" {
			existingSchemaName := ""
			if summary, err := storage.GetEntitySummary("schemas", project.Schema); err == nil {
				existingSchemaName = summary.Name
			}
			if existingSchemaName != ic.schemaName {
				fmt.Printf("Warning: Existing project uses schema '%s', ignoring --schema flag\n", existingSchemaName)
			}
		}
	} else {
		// Create new project
		project, err = ic.createNewProject(storage, projectName)
		if err != nil {
			return fmt.Errorf("failed to create new project: %w", err)
		}
		fmt.Printf("Created new project: %s (%s)\n", project.Name, project.ID)
	}

	// Create .ee file
	if ic.force {
		if err := UpdateEasyEnvFile(project.ID, ic.remote, ""); err != nil {
			return fmt.Errorf("failed to create .ee file: %w", err)
		}
	} else {
		if err := CreateEasyEnvFile(project.ID, ic.remote, ""); err != nil {
			return fmt.Errorf("failed to create .ee file: %w", err)
		}
	}

	fmt.Printf("✅ Initialized ee project in current directory\n")
	fmt.Printf("Project: %s\n", project.Name)
	fmt.Printf("Project ID: %s\n", project.ID)
	if ic.remote != "" {
		fmt.Printf("Remote: %s\n", ic.remote)
	}

	// Show next steps
	fmt.Printf("\nNext steps:\n")
	if len(project.Environments) == 0 {
		fmt.Printf("  1. Create environment config sheets:\n")
		fmt.Printf("     ee sheet create --env development\n")
		fmt.Printf("     ee sheet create --env production\n")
	} else {
		fmt.Printf("  1. Apply environment variables:\n")
		for envName := range project.Environments {
			fmt.Printf("     ee apply %s\n", envName)
		}
	}
	if ic.remote != "" {
		fmt.Printf("  2. Sync with remote:\n")
		fmt.Printf("     ee sync\n")
	}

	return nil
}

// createNewProject creates a new project with the specified name
func (ic *InitCommand) createNewProject(uuidStorage *storage.UUIDStorage, projectName string) (*schema.Project, error) {
	// Determine schema to use
	var schemaID string
	schemaName := ic.schemaName
	if schemaName == "" {
		schemaName = "default"
	}

	// Check if schema exists, create if it doesn't
	if uuidStorage.EntityExists("schemas", schemaName) {
		// Load existing schema
		existingSchema, err := uuidStorage.LoadSchema(schemaName)
		if err != nil {
			return nil, fmt.Errorf("failed to load schema '%s': %w", schemaName, err)
		}
		schemaID = existingSchema.ID
		fmt.Printf("Using existing schema: %s (%s)\n", existingSchema.Name, existingSchema.ID)
	} else {
		// Create default schema
		defaultSchema := ic.createDefaultSchema(schemaName)
		if err := uuidStorage.SaveSchema(defaultSchema); err != nil {
			return nil, fmt.Errorf("failed to save default schema: %w", err)
		}
		schemaID = defaultSchema.ID
		fmt.Printf("Created default schema: %s (%s)\n", defaultSchema.Name, defaultSchema.ID)
	}

	// Create the project
	description := fmt.Sprintf("Project initialized from %s", projectName)
	project := schema.NewProject(projectName, description, schemaID)

	// Save the project
	if err := uuidStorage.SaveProject(project); err != nil {
		return nil, fmt.Errorf("failed to save project: %w", err)
	}

	return project, nil
}

// createDefaultSchema creates a default schema with common environment variables
func (ic *InitCommand) createDefaultSchema(name string) *schema.Schema {
	variables := []schema.Variable{
		{
			Name:     "NODE_ENV",
			Title:    "Node.js Environment",
			Type:     "string",
			Default:  "development",
			Required: false,
		},
		{
			Name:     "PORT",
			Title:    "Server Port",
			Type:     "number",
			Default:  "3000",
			Required: false,
		},
		{
			Name:     "DATABASE_URL",
			Title:    "Database Connection URL",
			Type:     "url",
			Required: false,
		},
		{
			Name:     "LOG_LEVEL",
			Title:    "Logging Level",
			Type:     "string",
			Default:  "info",
			Required: false,
		},
	}

	description := "Default schema created by ee init"
	return schema.NewSchema(name, description, variables, nil)
}

// sanitizeProjectName cleans up a project name to make it valid
func sanitizeProjectName(name string) string {
	// Replace invalid characters with hyphens
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)

	// Remove leading/trailing hyphens
	result = strings.Trim(result, "-")

	// Ensure it's not empty
	if result == "" {
		result = "my-project"
	}

	// Convert to lowercase
	return strings.ToLower(result)
}
