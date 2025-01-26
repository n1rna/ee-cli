// internal/command/list.go
package command

import (
	"fmt"
	"strings"

	"github.com/n1rna/menv/internal/storage"
	"github.com/spf13/cobra"
)

type ListCommand struct {
}

func NewListCommand() *cobra.Command {
	lc := &ListCommand{}

	cmd := &cobra.Command{
		Use:   "list [type]",
		Short: "List available schemas, projects, and environments",
		Long: `List available schemas, projects, and their environments.
		
Examples:
  # List everything
  menv list

  # List only schemas
  menv list schemas

  # List only projects
  menv list projects

  # List environments for a specific project
  menv list envs myproject`,
		RunE: lc.Run,
	}

	return cmd
}

func (c *ListCommand) Run(cmd *cobra.Command, args []string) error {
	storage := GetStorage(cmd.Context())
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// Determine what to list based on args
	if len(args) == 0 {
		// List everything if no specific type is provided
		return c.listAll(storage)
	}

	switch strings.ToLower(args[0]) {
	case "schemas":
		return c.listSchemas(storage)
	case "projects":
		return c.listProjects(storage)
	case "envs":
		if len(args) < 2 {
			return fmt.Errorf("project name required for listing environments")
		}
		return c.listEnvironments(storage, args[1])
	default:
		return fmt.Errorf("unknown list type: %s (use 'schemas', 'projects', or 'envs')", args[0])
	}
}

func (c *ListCommand) listAll(storage *storage.Storage) error {
	// List schemas
	fmt.Println("Available Schemas:")
	fmt.Println("─────────────────")
	if err := c.listSchemas(storage); err != nil {
		return err
	}
	fmt.Println()

	// List projects and their environments
	fmt.Println("Projects and Environments:")
	fmt.Println("────────────────────────")
	projects, err := storage.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects found")
		return nil
	}

	for _, project := range projects {
		if err := c.listEnvironments(storage, project); err != nil {
			return err
		}
		fmt.Println()
	}

	return nil
}

func (c *ListCommand) listSchemas(storage *storage.Storage) error {
	schemas, err := storage.ListSchemas()
	if err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}

	if len(schemas) == 0 {
		fmt.Println("No schemas found")
		return nil
	}

	for _, schema := range schemas {
		// Load schema to get variable count
		schemaObj, err := storage.LoadSchema(schema)
		if err != nil {
			fmt.Printf("• %s (error loading schema)\n", schema)
			continue
		}
		fmt.Printf("• %s (%d variables)\n", schema, len(schemaObj.Variables))
	}

	return nil
}

func (c *ListCommand) listProjects(storage *storage.Storage) error {
	projects, err := storage.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects found")
		return nil
	}

	for _, project := range projects {
		fmt.Printf("• %s\n", project)
	}

	return nil
}

func (c *ListCommand) listEnvironments(storage *storage.Storage, projectName string) error {
	envs, err := storage.ListEnvironments(projectName)
	if err != nil {
		return fmt.Errorf("failed to list environments for project %s: %w", projectName, err)
	}

	if len(envs) == 0 {
		fmt.Printf("Project %s: No environments\n", projectName)
		return nil
	}

	fmt.Printf("Project %s:\n", projectName)
	for _, env := range envs {
		// Load config sheet to get schema information
		configSheet, err := storage.LoadConfigSheet(projectName, env)
		if err != nil {
			fmt.Printf("  • %s (error loading configuration)\n", env)
			continue
		}
		fmt.Printf("  • %s (schema: %s)\n", env, configSheet.Schema)
	}

	return nil
}
