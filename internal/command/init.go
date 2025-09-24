// Package command implements the ee init command for project initialization
package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/output"
)

// InitCommand handles the ee init command
type InitCommand struct{}

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

	cmd.Flags().String("project", "", "Project name (defaults to current directory name)")
	cmd.Flags().String("schema", "", "Schema to use for the project")
	cmd.Flags().String("remote", "", "Remote URL for synchronization")
	cmd.Flags().Bool("force", false, "Overwrite existing .ee file")
	cmd.Flags().Bool("quiet", false, "Suppress non-error output")

	return cmd
}

// Run executes the init command
func (c *InitCommand) Run(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.FormatTable, quiet)

	projectName, _ := cmd.Flags().GetString("project")
	schemaName, _ := cmd.Flags().GetString("schema")
	remote, _ := cmd.Flags().GetString("remote")
	force, _ := cmd.Flags().GetBool("force")

	// If no project name specified, use current directory name
	if projectName == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projectName = filepath.Base(cwd)
		printer.Info(fmt.Sprintf("Using current directory name as project: %s", projectName))
	}

	// Check if .ee file already exists
	eeFile := ".ee"
	if _, err := os.Stat(eeFile); err == nil && !force {
		return fmt.Errorf(".ee file already exists (use --force to overwrite)")
	}

	// Resolve schema
	var schemaID string
	if schemaName != "" {
		s, err := manager.Schemas.Get(schemaName)
		if err != nil {
			return fmt.Errorf("schema '%s' not found: %w", schemaName, err)
		}
		schemaID = s.ID
		printer.Info(fmt.Sprintf("Using schema: %s", schemaName))
	} else {
		// Create a default schema if none specified
		defaultSchema, err := c.createDefaultSchema(manager)
		if err != nil {
			return fmt.Errorf("failed to create default schema: %w", err)
		}
		schemaID = defaultSchema.ID
		printer.Info(fmt.Sprintf("Created default schema: %s", defaultSchema.Name))
	}

	// Create or get project
	project, err := c.getOrCreateProject(manager, projectName, schemaID)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	// Create .ee file
	err = c.createEEFile(eeFile, project.Name, remote)
	if err != nil {
		return fmt.Errorf("failed to create .ee file: %w", err)
	}

	printer.Success(fmt.Sprintf("Initialized ee project: %s", projectName))
	printer.Info("Created .ee file in current directory")
	printer.Info(fmt.Sprintf("Project ID: %s", project.ID))

	// Show next steps
	printer.Info("Next steps:")
	printer.Info("  1. Add environments: ee project env add development")
	printer.Info("  2. Create config sheets: ee sheet create my-app-dev --project " +
		projectName + " --environment development")
	printer.Info("  3. Apply environment: ee apply development")

	return nil
}

// createDefaultSchema creates a default schema for the project
func (c *InitCommand) createDefaultSchema(
	manager *entities.Manager,
) (*entities.Schema, error) {
	schemaName := "default"

	// Check if default schema already exists
	if s, err := manager.Schemas.GetByName(schemaName); err == nil {
		return s, nil
	}

	// Create default schema with common variables
	variables := []entities.Variable{
		{
			Name:     "NODE_ENV",
			Type:     "string",
			Title:    "Node environment",
			Required: false,
			Default:  "development",
		},
		{
			Name:     "PORT",
			Type:     "number",
			Title:    "Server port",
			Required: false,
			Default:  "3000",
		},
		{
			Name:     "DEBUG",
			Type:     "boolean",
			Title:    "Debug mode",
			Required: false,
			Default:  "false",
		},
	}

	return manager.Schemas.Create(
		schemaName,
		"Default schema created by ee init",
		variables,
		nil,
	)
}

// getOrCreateProject gets an existing project or creates a new one
func (c *InitCommand) getOrCreateProject(
	manager *entities.Manager,
	projectName, schemaID string,
) (*entities.Project, error) {
	// Try to get existing project
	if p, err := manager.Projects.GetByName(projectName); err == nil {
		// Project exists, update schema if needed
		if p.Schema != schemaID {
			return manager.Projects.Update(projectName, func(proj *entities.Project) error {
				proj.Schema = schemaID
				return nil
			})
		}
		return p, nil
	}

	// Create new project
	return manager.Projects.Create(
		projectName,
		fmt.Sprintf("Project initialized for %s", projectName),
		schemaID,
	)
}

// createEEFile creates the .ee configuration file
func (c *InitCommand) createEEFile(filename, projectName, remote string) error {
	content := fmt.Sprintf("project: %s\n", projectName)
	if remote != "" {
		content += fmt.Sprintf("remote: %s\n", remote)
	}

	return os.WriteFile(filename, []byte(content), 0o644)
}

// GetCurrentProject reads the project name from .ee file in current directory
func GetCurrentProject() (string, error) {
	eeFile := ".ee"

	// Check if .ee file exists
	if _, err := os.Stat(eeFile); os.IsNotExist(err) {
		return "", fmt.Errorf(".ee file not found in current directory")
	}

	// Read .ee file
	content, err := os.ReadFile(eeFile)
	if err != nil {
		return "", fmt.Errorf("failed to read .ee file: %w", err)
	}

	// Parse .ee file (simple key: value format)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			if key == "project" {
				return value, nil
			}
		}
	}

	return "", fmt.Errorf("no project specified in .ee file")
}

// GetCurrentRemote reads the remote URL from .ee file in current directory
func GetCurrentRemote() (string, error) {
	eeFile := ".ee"

	// Check if .ee file exists
	if _, err := os.Stat(eeFile); os.IsNotExist(err) {
		return "", fmt.Errorf(".ee file not found in current directory")
	}

	// Read .ee file
	content, err := os.ReadFile(eeFile)
	if err != nil {
		return "", fmt.Errorf("failed to read .ee file: %w", err)
	}

	// Parse .ee file (simple key: value format)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			if key == "remote" {
				return value, nil
			}
		}
	}

	return "", nil // Remote is optional
}
