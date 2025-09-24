// Package command implements the ee project command for managing projects
package command

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/output"
)

// ProjectCommand handles the ee project command
type ProjectCommand struct {
	reader *bufio.Reader
}

// resolveProjectName resolves the project name from args or .ee file
// If args has at least one element, uses args[0] as project name
// Otherwise, tries to get project name from .ee file in current directory
func (c *ProjectCommand) resolveProjectName(args []string) (string, error) {
	if len(args) > 0 && args[0] != "" {
		// Project name provided as argument
		return args[0], nil
	}

	// Try to get project name from .ee file
	projectName, err := GetCurrentProject()
	if err != nil {
		return "", fmt.Errorf(
			"no project specified and no .ee file found in current directory: %w",
			err,
		)
	}

	if projectName == "" {
		return "", fmt.Errorf("no project specified and .ee file does not contain a project")
	}

	return projectName, nil
}

// resolveProjectNameForEnvCommand resolves project name for env commands that expect:
// - 1 arg: environment name (use .ee file for project)
// - 2 args: project name + environment name
func (c *ProjectCommand) resolveProjectNameForEnvCommand(args []string) (string, []string, error) {
	switch len(args) {
	case 0:
		return "", args, fmt.Errorf("environment name is required")
	case 1:
		// Only environment name provided, get project from .ee file
		projectName, err := GetCurrentProject()
		if err != nil {
			return "", args, fmt.Errorf("no .ee file found, please specify project name: %w", err)
		}
		if projectName == "" {
			return "", args, fmt.Errorf(
				".ee file does not contain a project, please specify project name",
			)
		}
		return projectName, args, nil
	case 2:
		// Both project and environment provided
		return args[0], args[1:], nil
	default:
		return "", args, fmt.Errorf("too many arguments")
	}
}

// NewProjectCommand creates the project command with all subcommands
func NewProjectCommand(groupId string) *cobra.Command {
	pc := &ProjectCommand{
		reader: bufio.NewReader(os.Stdin),
	}

	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
		Long: `Create and manage projects for organizing environment configurations.

Projects group related environments together and define which schema to use
for validation. Each project can have multiple environments (development,
staging, production, etc.).`,
		GroupID: groupId,
	}

	// Add subcommands
	cmd.AddCommand(
		pc.newCreateCommand(),
		pc.newShowCommand(),
		pc.newEditCommand(),
		pc.newDeleteCommand(),
		pc.newListCommand(),
		pc.newSchemaCommand(),
		pc.newEnvCommand(),
	)

	return cmd
}

func (c *ProjectCommand) newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [project-name]",
		Short: "Create a new project",
		Long: `Create a new project with a specified schema.

Examples:
  # Interactive mode
  ee project create my-app

  # With schema specified
  ee project create my-app --schema web-service

  # With description
  ee project create my-app --schema web-service --description "My web application"`,
		Args: cobra.ExactArgs(1),
		RunE: c.runCreate,
	}

	cmd.Flags().String("schema", "", "Schema to use for this project")
	cmd.Flags().String("description", "", "Project description")
	cmd.Flags().String("format", "table", "Output format (table, json)")
	cmd.Flags().Bool("quiet", false, "Suppress non-error output")

	return cmd
}

func (c *ProjectCommand) newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [project-name]",
		Short: "Show details of a project",
		Long: `Show detailed information about a project including its environments.

If no project name is provided, uses the project from .ee file in current directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: c.runShow,
	}

	cmd.Flags().String("format", "table", "Output format (table, json)")

	return cmd
}

func (c *ProjectCommand) newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		Args:  cobra.ExactArgs(0),
		RunE:  c.runList,
	}

	cmd.Flags().String("format", "table", "Output format (table, json)")

	return cmd
}

func (c *ProjectCommand) newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [project-name]",
		Short: "Delete a project",
		Long: `Delete a project and all its associated configuration sheets.

WARNING: This will permanently delete all environments and their configurations
for this project. This action cannot be undone.

If no project name is provided, uses the project from .ee file in current directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: c.runDelete,
	}

	cmd.Flags().Bool("quiet", false, "Suppress non-error output")

	return cmd
}

func (c *ProjectCommand) newEditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit [project-name]",
		Short: "Edit a project using your preferred editor",
		Long: `Edit a project using your preferred editor.

The editor is determined by the $EDITOR environment variable, falling back to 'vim' if not set.
The project is presented as JSON for editing, and changes are validated and applied upon saving.

If no project name is provided, uses the project from .ee file in current directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: c.runEdit,
	}

	return cmd
}

func (c *ProjectCommand) newSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema [project-name] [schema-name]",
		Short: "Set or show the schema for a project",
		Long: `Set or show the schema for a project.

With one argument (project name), shows the current schema.
With two arguments, sets the schema for the project.

If no project name is provided, uses the project from .ee file in current directory.`,
		Args: cobra.RangeArgs(0, 2),
		RunE: c.runSetSchema,
	}

	cmd.Flags().String("format", "table", "Output format (table, json)")
	cmd.Flags().Bool("quiet", false, "Suppress non-error output")

	return cmd
}

func (c *ProjectCommand) newEnvCommand() *cobra.Command {
	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Manage project environments",
		Long:  `Add, remove, and list environments for a project.`,
	}

	// Add subcommands
	envCmd.AddCommand(
		c.newEnvAddCommand(),
		c.newEnvRemoveCommand(),
		c.newEnvListCommand(),
	)

	return envCmd
}

func (c *ProjectCommand) newEnvAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add [project-name] <environment-name>",
		Short: "Add an environment to a project",
		Long: `Add a new environment to a project.

Examples:
  # Add environment to current project (from .ee file)
  ee project env add development

  # Add environment to specific project
  ee project env add my-app staging`,
		Args: cobra.RangeArgs(1, 2),
		RunE: c.runEnvAdd,
	}
}

func (c *ProjectCommand) newEnvRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [project-name] <environment-name>",
		Short: "Remove an environment from a project",
		Long: `Remove an environment from a project.

WARNING: This will also delete any configuration sheets associated with this environment.

Examples:
  # Remove environment from current project (from .ee file)
  ee project env remove development

  # Remove environment from specific project
  ee project env remove my-app staging`,
		Args: cobra.RangeArgs(1, 2),
		RunE: c.runEnvRemove,
	}
}

func (c *ProjectCommand) newEnvListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list [project-name]",
		Short: "List environments for a project",
		Long: `List all environments for a project.

If no project name is provided, uses the project from .ee file in current directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: c.runEnvList,
	}
}

// Implementation methods

func (c *ProjectCommand) runCreate(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.Format(format), quiet)

	projectName := args[0]
	description, _ := cmd.Flags().GetString("description")
	schemaName, _ := cmd.Flags().GetString("schema")

	var schemaID string
	if schemaName != "" {
		// Resolve schema name to ID
		s, err := manager.Schemas.Get(schemaName)
		if err != nil {
			return fmt.Errorf("schema '%s' not found: %w", schemaName, err)
		}
		schemaID = s.ID
	}

	// Create project
	p, err := manager.Projects.Create(projectName, description, schemaID)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	printer.Success(fmt.Sprintf("Successfully created project '%s'", projectName))
	return printer.PrintProject(p)
}

func (c *ProjectCommand) runShow(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	format, _ := cmd.Flags().GetString("format")
	printer := output.NewPrinter(output.Format(format), false)

	// Resolve project name
	projectName, err := c.resolveProjectName(args)
	if err != nil {
		return err
	}

	p, err := manager.Projects.Get(projectName)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	return printer.PrintProject(p)
}

func (c *ProjectCommand) runList(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	format, _ := cmd.Flags().GetString("format")
	printer := output.NewPrinter(output.Format(format), false)

	summaries, err := manager.Projects.List()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	return printer.PrintProjectList(summaries)
}

func (c *ProjectCommand) runDelete(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.FormatTable, quiet)

	// Resolve project name
	projectName, err := c.resolveProjectName(args)
	if err != nil {
		return err
	}

	// TODO: Add warning about deleting associated config sheets

	if err := manager.Projects.Delete(projectName); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	printer.Success(fmt.Sprintf("Successfully deleted project '%s'", projectName))
	return nil
}

func (c *ProjectCommand) runEdit(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Resolve project name
	projectName, err := c.resolveProjectName(args)
	if err != nil {
		return err
	}

	// Load the project
	p, err := manager.Projects.Get(projectName)
	if err != nil {
		return fmt.Errorf("failed to load project '%s': %w", projectName, err)
	}

	validator := func(data []byte) (interface{}, error) {
		var editedProject entities.Project
		if err := json.Unmarshal(data, &editedProject); err != nil {
			return nil, fmt.Errorf("invalid JSON in edited file: %w", err)
		}

		// Preserve the original ID and timestamps if they weren't changed
		if editedProject.ID == "" {
			editedProject.ID = p.ID
		}
		if editedProject.CreatedAt.IsZero() {
			editedProject.CreatedAt = p.CreatedAt
		}

		// Validate the edited project
		if editedProject.Name == "" {
			return nil, fmt.Errorf("project name cannot be empty")
		}

		return &editedProject, nil
	}

	saver := func(entity interface{}) error {
		editedProject := entity.(*entities.Project)
		return manager.Projects.Save(editedProject)
	}

	changeReporter := func(original, edited interface{}) {
		origProject := original.(*entities.Project)
		editedProject := edited.(*entities.Project)

		if origProject.Name != editedProject.Name {
			fmt.Printf("  Name: %s → %s\n", origProject.Name, editedProject.Name)
		}
		if origProject.Description != editedProject.Description {
			fmt.Printf("  Description updated\n")
		}
		if origProject.Schema != editedProject.Schema {
			fmt.Printf("  Schema: %s → %s\n", origProject.Schema, editedProject.Schema)
		}
		if len(origProject.Environments) != len(editedProject.Environments) {
			fmt.Printf(
				"  Environments: %d → %d\n",
				len(origProject.Environments),
				len(editedProject.Environments),
			)
		}
	}

	return EditEntity(
		fmt.Sprintf("project '%s'", projectName),
		p,
		&BaseEditorCommands{},
		validator,
		saver,
		changeReporter,
	)
}

func (c *ProjectCommand) runSetSchema(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.Format(format), quiet)

	var projectName string
	var err error

	switch len(args) {
	case 0:
		// Get project from .ee file
		projectName, err = GetCurrentProject()
		if err != nil {
			return fmt.Errorf("no .ee file found, please specify project name: %w", err)
		}
		if projectName == "" {
			return fmt.Errorf(".ee file does not contain a project, please specify project name")
		}

		// Show current schema
		p, err := manager.Projects.Get(projectName)
		if err != nil {
			return fmt.Errorf("failed to load project: %w", err)
		}

		if p.Schema == "" {
			printer.Info(fmt.Sprintf("Project '%s' has no schema set", projectName))
		} else {
			// Try to resolve schema ID to name
			s, err := manager.Schemas.GetByID(p.Schema)
			if err != nil {
				printer.Info(fmt.Sprintf("Project '%s' uses schema ID: %s", projectName, p.Schema))
			} else {
				printer.Info(fmt.Sprintf("Project '%s' uses schema: %s (%s)", projectName, s.Name, s.ID))
			}
		}
		return nil

	case 1:
		// Could be project name (show schema) or schema name (set for current project)
		// Try to determine based on .ee file existence
		currentProject, err := GetCurrentProject()
		if err == nil && currentProject != "" {
			// .ee file exists, treat arg as schema name
			projectName = currentProject
			schemaName := args[0]

			s, err := manager.Schemas.Get(schemaName)
			if err != nil {
				return fmt.Errorf("schema '%s' not found: %w", schemaName, err)
			}

			_, err = manager.Projects.Update(projectName, func(p *entities.Project) error {
				p.Schema = s.ID
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to update project schema: %w", err)
			}

			printer.Success(
				fmt.Sprintf("Set schema for project '%s' to '%s'", projectName, schemaName),
			)
			return nil
		} else {
			// No .ee file, treat arg as project name and show schema
			projectName = args[0]
			p, err := manager.Projects.Get(projectName)
			if err != nil {
				return fmt.Errorf("failed to load project: %w", err)
			}

			if p.Schema == "" {
				printer.Info(fmt.Sprintf("Project '%s' has no schema set", projectName))
			} else {
				s, err := manager.Schemas.GetByID(p.Schema)
				if err != nil {
					printer.Info(fmt.Sprintf("Project '%s' uses schema ID: %s", projectName, p.Schema))
				} else {
					printer.Info(fmt.Sprintf("Project '%s' uses schema: %s (%s)", projectName, s.Name, s.ID))
				}
			}
			return nil
		}

	case 2:
		// project name + schema name
		projectName = args[0]
		schemaName := args[1]

		s, err := manager.Schemas.Get(schemaName)
		if err != nil {
			return fmt.Errorf("schema '%s' not found: %w", schemaName, err)
		}

		_, err = manager.Projects.Update(projectName, func(p *entities.Project) error {
			p.Schema = s.ID
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to update project schema: %w", err)
		}

		printer.Success(fmt.Sprintf("Set schema for project '%s' to '%s'", projectName, schemaName))
		return nil

	default:
		return fmt.Errorf("too many arguments")
	}
}

func (c *ProjectCommand) runEnvAdd(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	printer := output.NewPrinter(output.FormatTable, false)

	// Resolve project name and environment name
	projectName, remainingArgs, err := c.resolveProjectNameForEnvCommand(args)
	if err != nil {
		return err
	}

	envName := remainingArgs[0]

	_, err = manager.Projects.AddEnvironment(projectName, envName)
	if err != nil {
		return fmt.Errorf("failed to add environment: %w", err)
	}

	printer.Success(fmt.Sprintf("Added environment '%s' to project '%s'", envName, projectName))
	return nil
}

func (c *ProjectCommand) runEnvRemove(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	printer := output.NewPrinter(output.FormatTable, false)

	// Resolve project name and environment name
	projectName, remainingArgs, err := c.resolveProjectNameForEnvCommand(args)
	if err != nil {
		return err
	}

	envName := remainingArgs[0]

	_, err = manager.Projects.RemoveEnvironment(projectName, envName)
	if err != nil {
		return fmt.Errorf("failed to remove environment: %w", err)
	}

	printer.Success(fmt.Sprintf("Removed environment '%s' from project '%s'", envName, projectName))
	return nil
}

func (c *ProjectCommand) runEnvList(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	printer := output.NewPrinter(output.FormatTable, false)

	// Resolve project name
	projectName, err := c.resolveProjectName(args)
	if err != nil {
		return err
	}

	p, err := manager.Projects.Get(projectName)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	fmt.Printf("Environments for project '%s':\n", projectName)
	if len(p.Environments) == 0 {
		printer.Info("No environments defined")
		return nil
	}

	for envName := range p.Environments {
		fmt.Printf("  - %s\n", envName)
	}

	return nil
}
