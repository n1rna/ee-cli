// Package command implements the ee project command for managing projects
package command

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/n1rna/ee-cli/internal/schema"
	"github.com/n1rna/ee-cli/internal/storage"
	"github.com/spf13/cobra"
)

// ProjectCommand handles the ee project command
type ProjectCommand struct {
	reader *bufio.Reader
}

// NewProjectCommand creates a new ee project command
func NewProjectCommand() *cobra.Command {
	pc := &ProjectCommand{
		reader: bufio.NewReader(os.Stdin),
	}

	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
		Long: `Create and manage projects for environment variable management.

Projects contain multiple environments and can be synchronized with remote APIs.`,
	}

	// Add subcommands
	cmd.AddCommand(
		pc.newListCommand(),
		pc.newShowCommand(),
		pc.newEditCommand(),
		pc.newDeleteCommand(),
		pc.newEnvCommand(),
	)

	return cmd
}

func (c *ProjectCommand) newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all projects with their environments",
		Long: `List all projects with their environments.

Examples:
  # List all projects
  ee project list`,
		Args: cobra.NoArgs,
		RunE: c.runList,
	}
}

func (c *ProjectCommand) runList(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	fmt.Println("Projects and Environments:")
	fmt.Println("────────────────────────")

	projects, err := uuidStorage.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects found")
		return nil
	}

	for _, projectSummary := range projects {
		// Load full project to get environment information
		project, err := uuidStorage.LoadProject(projectSummary.Name)
		if err != nil {
			fmt.Printf("Project %s: (error loading details)\n", projectSummary.Name)
			continue
		}

		if len(project.Environments) == 0 {
			fmt.Printf("Project %s: No environments\n", projectSummary.Name)
		} else {
			fmt.Printf("Project %s:\n", projectSummary.Name)
			for envName, envInfo := range project.Environments {
				// Load config sheet to get schema information using naming convention
				configSheetName := project.GetConfigSheetName(envInfo.Name)
				configSheet, err := uuidStorage.LoadConfigSheet(configSheetName)
				if err != nil {
					fmt.Printf("  • %s (error loading configuration '%s')\n", envName, configSheetName)
					continue
				}

				// Get schema name from sheet's schema reference
				schemaName := "unknown"
				if configSheet.Schema.Ref != "" {
					// Extract schema ID from reference
					if schemaID := strings.TrimPrefix(configSheet.Schema.Ref, "#/schemas/"); schemaID != configSheet.Schema.Ref {
						if summary, err := uuidStorage.GetEntitySummary("schemas", schemaID); err == nil {
							schemaName = summary.Name
						}
					}
				} else {
					schemaName = "inline"
				}
				fmt.Printf("  • %s (schema: %s)\n", envName, schemaName)
			}
		}
		fmt.Println()
	}

	return nil
}

func (c *ProjectCommand) newShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show [project-name]",
		Short: "Show detailed information about a project",
		Long: `Show detailed information about a project including its environments and configuration.

Examples:
  # Show project details
  ee project show my-project`,
		Args: cobra.ExactArgs(1),
		RunE: c.runShow,
	}
}

func (c *ProjectCommand) runShow(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	projectName := args[0]

	// Load the project
	project, err := uuidStorage.LoadProject(projectName)
	if err != nil {
		return fmt.Errorf("failed to load project '%s': %w", projectName, err)
	}

	// Display project information
	fmt.Printf("Project: %s\n", project.Name)
	fmt.Printf("ID: %s\n", project.ID)
	if project.Description != "" {
		fmt.Printf("Description: %s\n", project.Description)
	}
	fmt.Printf("Schema: %s\n", project.Schema)
	fmt.Printf("Created: %s\n", project.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s\n", project.UpdatedAt.Format("2006-01-02 15:04:05"))

	if len(project.Environments) > 0 {
		fmt.Printf("\nEnvironments (%d):\n", len(project.Environments))
		for envName, envInfo := range project.Environments {
			configSheetName := project.GetConfigSheetName(envInfo.Name)
			fmt.Printf("  • %s → %s\n", envName, configSheetName)
		}
	} else {
		fmt.Println("\nNo environments configured")
	}

	return nil
}

func (c *ProjectCommand) newEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "edit [project-name]",
		Short: "Edit a project using your preferred editor",
		Long: `Edit a project using your preferred editor.

The editor is determined by the $EDITOR environment variable, falling back to 'vim' if not set.
The project is presented as JSON for editing, and changes are validated and applied upon saving.

Examples:
  # Edit a project
  ee project edit my-project`,
		Args: cobra.ExactArgs(1),
		RunE: c.runEdit,
	}
}

func (c *ProjectCommand) runEdit(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	projectName := args[0]

	// Load the project
	project, err := uuidStorage.LoadProject(projectName)
	if err != nil {
		return fmt.Errorf("failed to load project '%s': %w", projectName, err)
	}

	// Get editor command
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim" // fallback
	}

	// Convert to JSON for editing
	jsonData, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize project: %w", err)
	}

	// Create temporary file
	tmpFile, err := c.createTempFile("project", jsonData)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	fmt.Printf("📝 Editing project '%s' using %s...\n", projectName, editor)

	// Open editor
	if err := c.openEditor(editor, tmpFile); err != nil {
		return err
	}

	// Read back the edited content
	editedData, err := ioutil.ReadFile(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to read edited file: %w", err)
	}

	// Parse the edited JSON
	var editedProject schema.Project
	if err := json.Unmarshal(editedData, &editedProject); err != nil {
		return fmt.Errorf("invalid JSON in edited file: %w", err)
	}

	// Preserve the original ID and timestamps if they weren't changed
	if editedProject.ID == "" {
		editedProject.ID = project.ID
	}
	if editedProject.CreatedAt.IsZero() {
		editedProject.CreatedAt = project.CreatedAt
	}

	// Validate the edited project
	if editedProject.Name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	// Save the updated project
	if err := uuidStorage.SaveProject(&editedProject); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	fmt.Printf("✅ Project '%s' updated successfully\n", editedProject.Name)

	// Show what changed
	if project.Name != editedProject.Name {
		fmt.Printf("  Name: %s → %s\n", project.Name, editedProject.Name)
	}
	if project.Description != editedProject.Description {
		fmt.Printf("  Description updated\n")
	}
	if len(project.Environments) != len(editedProject.Environments) {
		fmt.Printf("  Environments: %d → %d\n", len(project.Environments), len(editedProject.Environments))
	}

	return nil
}

func (c *ProjectCommand) newDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [project-name]",
		Short: "Delete a project",
		Long: `Delete a project and all its associated environments and config sheets.

This operation cannot be undone.

Examples:
  # Delete a project
  ee project delete my-project`,
		Args: cobra.ExactArgs(1),
		RunE: c.runDelete,
	}
}

func (c *ProjectCommand) runDelete(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	projectName := args[0]

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete project '%s'? This will also delete all its environments and config sheets. (y/N): ", projectName)

	response, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Deletion cancelled")
		return nil
	}

	// TODO: Implement project deletion in storage
	// For now, just return an error indicating it's not implemented
	return fmt.Errorf("project deletion not yet implemented")
}

// createTempFile creates a temporary file for editing
func (c *ProjectCommand) createTempFile(prefix string, data []byte) (string, error) {
	tmpDir := os.TempDir()

	// Create temp file
	file, err := ioutil.TempFile(tmpDir, fmt.Sprintf("ee-%s-*.json", prefix))
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer file.Close()

	// Write data to temp file
	if _, err := file.Write(data); err != nil {
		return "", fmt.Errorf("failed to write to temporary file: %w", err)
	}

	return file.Name(), nil
}

// openEditor opens the specified editor with the given file
func (c *ProjectCommand) openEditor(editor, filename string) error {
	// Split editor command (in case it has arguments)
	editorParts := strings.Fields(editor)
	if len(editorParts) == 0 {
		return fmt.Errorf("editor command is empty")
	}

	// Prepare command
	editorCmd := editorParts[0]
	editorArgs := append(editorParts[1:], filename)

	// Execute editor
	cmd := exec.Command(editorCmd, editorArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Opening %s...\n", filename)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor command failed: %w", err)
	}

	return nil
}

func (c *ProjectCommand) newEnvCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environments for projects",
		Long: `Manage environments for projects.

Examples:
  # Add environment to project
  ee project env add my-project development

  # Remove environment from project
  ee project env remove my-project development

  # List environments in project
  ee project env list my-project`,
	}

	// Add env subcommands
	cmd.AddCommand(
		c.newEnvAddCommand(),
		c.newEnvRemoveCommand(),
		c.newEnvListCommand(),
		c.newEnvFixCommand(),
	)

	return cmd
}

func (c *ProjectCommand) newEnvAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add [project-name] [environment-name]",
		Short: "Add an environment to a project",
		Long: `Add an environment to a project.

This will create the environment entry in the project and auto-create
a corresponding config sheet using the naming convention.

Examples:
  # Add development environment to my-api project
  ee project env add my-api development`,
		Args: cobra.ExactArgs(2),
		RunE: c.runEnvAdd,
	}
}

func (c *ProjectCommand) runEnvAdd(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	projectName := args[0]
	envName := args[1]

	// Load the project
	project, err := uuidStorage.LoadProject(projectName)
	if err != nil {
		return fmt.Errorf("failed to load project '%s': %w", projectName, err)
	}

	// Check if environment already exists
	if _, exists := project.Environments[envName]; exists {
		return fmt.Errorf("environment '%s' already exists in project '%s'", envName, projectName)
	}

	// Add environment to project
	project.AddEnvironment(envName)

	// Auto-create config sheet for this environment
	configSheetName := project.GetConfigSheetName(envName)
	description := fmt.Sprintf("Config sheet for %s", configSheetName)

	// Use the project's schema - all environments inherit the project schema
	var schemaRef schema.SchemaReference
	if project.Schema != "" {
		schemaRef = schema.SchemaReference{
			Ref: "#/schemas/" + project.Schema,
		}
	}

	configSheet := schema.NewConfigSheetForProject(configSheetName, description, schemaRef, project.ID, envName, make(map[string]string))

	// Save config sheet
	if err := uuidStorage.SaveConfigSheet(configSheet); err != nil {
		return fmt.Errorf("failed to create config sheet: %w", err)
	}

	// Save updated project
	if err := uuidStorage.SaveProject(project); err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	fmt.Printf("✅ Added environment '%s' to project '%s'\n", envName, projectName)
	fmt.Printf("📋 Created config sheet: %s\n", configSheetName)

	return nil
}

func (c *ProjectCommand) newEnvRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [project-name] [environment-name]",
		Short: "Remove an environment from a project",
		Long: `Remove an environment from a project.

This will remove the environment from the project and optionally
delete the associated config sheet.

Examples:
  # Remove development environment from my-api project
  ee project env remove my-api development`,
		Args: cobra.ExactArgs(2),
		RunE: c.runEnvRemove,
	}
}

func (c *ProjectCommand) runEnvRemove(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	projectName := args[0]
	envName := args[1]

	// Load the project
	project, err := uuidStorage.LoadProject(projectName)
	if err != nil {
		return fmt.Errorf("failed to load project '%s': %w", projectName, err)
	}

	// Check if environment exists
	if _, exists := project.Environments[envName]; !exists {
		return fmt.Errorf("environment '%s' does not exist in project '%s'", envName, projectName)
	}

	configSheetName := project.GetConfigSheetName(envName)

	// Confirm deletion
	fmt.Printf("Are you sure you want to remove environment '%s' from project '%s'? This will also delete the config sheet '%s'. (y/N): ", envName, projectName, configSheetName)

	response, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Removal cancelled")
		return nil
	}

	// Remove environment from project
	delete(project.Environments, envName)

	// Remove config sheet
	if uuidStorage.EntityExists("sheets", configSheetName) {
		if err := uuidStorage.DeleteEntity("sheets", configSheetName); err != nil {
			fmt.Printf("Warning: failed to delete config sheet '%s': %v\n", configSheetName, err)
		} else {
			fmt.Printf("🗑️ Deleted config sheet: %s\n", configSheetName)
		}
	}

	// Save updated project
	if err := uuidStorage.SaveProject(project); err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	fmt.Printf("✅ Removed environment '%s' from project '%s'\n", envName, projectName)

	return nil
}

func (c *ProjectCommand) newEnvListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list [project-name]",
		Short: "List environments in a project",
		Long: `List environments in a project with their config sheets.

Examples:
  # List environments in my-api project
  ee project env list my-api`,
		Args: cobra.ExactArgs(1),
		RunE: c.runEnvList,
	}
}

func (c *ProjectCommand) runEnvList(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	projectName := args[0]

	// Load the project
	project, err := uuidStorage.LoadProject(projectName)
	if err != nil {
		return fmt.Errorf("failed to load project '%s': %w", projectName, err)
	}

	fmt.Printf("Environments in project '%s':\n", projectName)
	fmt.Println("─────────────────────────────")

	if len(project.Environments) == 0 {
		fmt.Println("No environments configured")
		return nil
	}

	for envName, envInfo := range project.Environments {
		configSheetName := project.GetConfigSheetName(envInfo.Name)
		status := "✅"
		schemaStatus := ""

		// Check if config sheet exists and has correct schema
		if !uuidStorage.EntityExists("sheets", configSheetName) {
			status = "❌"
		} else {
			// Load config sheet to check schema
			if configSheet, err := uuidStorage.LoadConfigSheet(configSheetName); err == nil {
				if configSheet.Schema.Ref == "" {
					schemaStatus = " (⚠️ using inline schema)"
				} else {
					projectSchemaRef := "#/schemas/" + project.Schema
					if configSheet.Schema.Ref != projectSchemaRef {
						schemaStatus = " (⚠️ schema mismatch)"
					}
				}
			}
		}

		fmt.Printf("  %s %s → %s%s\n", status, envName, configSheetName, schemaStatus)
	}

	return nil
}

// fixProjectConfigSheetSchemas fixes config sheets to use the project's schema
func (c *ProjectCommand) fixProjectConfigSheetSchemas(uuidStorage *storage.UUIDStorage, project *schema.Project) error {
	if project.Schema == "" {
		return fmt.Errorf("project '%s' has no schema defined", project.Entity.Name)
	}

	projectSchemaRef := schema.SchemaReference{
		Ref: "#/schemas/" + project.Schema,
	}

	fmt.Printf("Fixing config sheets to use project schema: %s\n", project.Schema)

	for _, envInfo := range project.Environments {
		configSheetName := project.GetConfigSheetName(envInfo.Name)

		if !uuidStorage.EntityExists("sheets", configSheetName) {
			fmt.Printf("  ⏭️  Skipping %s (not found)\n", configSheetName)
			continue
		}

		configSheet, err := uuidStorage.LoadConfigSheet(configSheetName)
		if err != nil {
			fmt.Printf("  ❌ Failed to load %s: %v\n", configSheetName, err)
			continue
		}

		// Check if it needs fixing
		if configSheet.Schema.Ref == projectSchemaRef.Ref {
			fmt.Printf("  ✅ %s (already correct)\n", configSheetName)
			continue
		}

		// Update the schema reference
		configSheet.Schema = projectSchemaRef

		if err := uuidStorage.SaveConfigSheet(configSheet); err != nil {
			fmt.Printf("  ❌ Failed to save %s: %v\n", configSheetName, err)
			continue
		}

		fmt.Printf("  🔧 Fixed %s\n", configSheetName)
	}

	return nil
}

func (c *ProjectCommand) newEnvFixCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "fix [project-name]",
		Short: "Fix config sheets to use the project's schema",
		Long: `Fix config sheets to use the project's schema instead of inline schemas.

This command updates all environment config sheets in a project to use
the project's schema, removing any inline schema references.

Examples:
  # Fix config sheets in my-api project
  ee project env fix my-api`,
		Args: cobra.ExactArgs(1),
		RunE: c.runEnvFix,
	}
}

func (c *ProjectCommand) runEnvFix(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	projectName := args[0]

	// Load the project
	project, err := uuidStorage.LoadProject(projectName)
	if err != nil {
		return fmt.Errorf("failed to load project '%s': %w", projectName, err)
	}

	fmt.Printf("🔧 Fixing config sheets for project '%s'\n", projectName)

	return c.fixProjectConfigSheetSchemas(uuidStorage, project)
}
