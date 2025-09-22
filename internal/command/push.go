// Package command implements the ee push command for pushing local entities to remote
package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/api"
	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/output"
)

// PushCommand handles the ee push command
type PushCommand struct{}

// NewPushCommand creates a new ee push command
func NewPushCommand(groupId string) *cobra.Command {
	pc := &PushCommand{}

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push local entities to remote",
		Long: `Push local schemas, projects, and config sheets to the remote server.

This command synchronizes your local entities with the remote server, uploading
any changes or new entities that exist locally but not remotely.

Examples:
  # Push all entities
  ee push

  # Push only schemas
  ee push --schemas

  # Push only projects
  ee push --projects

  # Push only config sheets
  ee push --sheets

  # Dry run to see what would be pushed
  ee push --dry-run`,
		RunE:    pc.Run,
		GroupID: groupId,
	}

	cmd.Flags().Bool("schemas", false, "Push only schemas")
	cmd.Flags().Bool("projects", false, "Push only projects")
	cmd.Flags().Bool("sheets", false, "Push only config sheets")
	cmd.Flags().Bool("dry-run", false, "Show what would be pushed without actually pushing")
	cmd.Flags().Bool("quiet", false, "Suppress non-error output")

	return cmd
}

// Run executes the push command
func (c *PushCommand) Run(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.FormatTable, quiet)

	// Get remote URL
	remote, err := GetCurrentRemote()
	if err != nil {
		return fmt.Errorf("failed to get remote URL: %w", err)
	}
	if remote == "" {
		return fmt.Errorf("no remote URL configured. Use 'ee init --remote <url>' to set one")
	}

	// Create API client
	client, err := api.ClientFromRemoteURL(remote)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Get flags
	schemasOnly, _ := cmd.Flags().GetBool("schemas")
	projectsOnly, _ := cmd.Flags().GetBool("projects")
	sheetsOnly, _ := cmd.Flags().GetBool("sheets")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// If no specific flags, push everything
	pushAll := !schemasOnly && !projectsOnly && !sheetsOnly

	if dryRun {
		printer.Info("Dry run mode - showing what would be pushed:")
	}

	// Push schemas
	if pushAll || schemasOnly {
		if err := c.pushSchemas(manager, client, printer, dryRun); err != nil {
			return fmt.Errorf("failed to push schemas: %w", err)
		}
	}

	// Push projects
	if pushAll || projectsOnly {
		if err := c.pushProjects(manager, client, printer, dryRun); err != nil {
			return fmt.Errorf("failed to push projects: %w", err)
		}
	}

	// Push config sheets
	if pushAll || sheetsOnly {
		if err := c.pushConfigSheets(manager, client, printer, dryRun); err != nil {
			return fmt.Errorf("failed to push config sheets: %w", err)
		}
	}

	if !dryRun {
		printer.Success("Successfully pushed all entities to remote")
	} else {
		printer.Info("Dry run completed")
	}

	return nil
}

// pushSchemas pushes all local schemas to remote
func (c *PushCommand) pushSchemas(manager *entities.Manager, client *api.Client, printer *output.Printer, dryRun bool) error {
	summaries, err := manager.Schemas.List()
	if err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}

	if len(summaries) == 0 {
		printer.Info("No schemas to push")
		return nil
	}

	printer.Info(fmt.Sprintf("Pushing %d schemas...", len(summaries)))

	for _, summary := range summaries {
		schema, err := manager.Schemas.GetByID(summary.Name) // summary.Name is UUID in index
		if err != nil {
			printer.Warning(fmt.Sprintf("Failed to load schema %s: %v", summary.Name, err))
			continue
		}

		if dryRun {
			printer.Info(fmt.Sprintf("  Would push schema: %s (%s)", schema.Name, schema.ID))
		} else {
			// Convert to API type
			apiSchema := api.SchemaToAPI(schema)
			if _, err := client.PushSchema(apiSchema); err != nil {
				printer.Warning(fmt.Sprintf("Failed to push schema %s: %v", schema.Name, err))
				continue
			}
			printer.Info(fmt.Sprintf("  Pushed schema: %s", schema.Name))
		}
	}

	return nil
}

// pushProjects pushes all local projects to remote
func (c *PushCommand) pushProjects(manager *entities.Manager, client *api.Client, printer *output.Printer, dryRun bool) error {
	summaries, err := manager.Projects.List()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(summaries) == 0 {
		printer.Info("No projects to push")
		return nil
	}

	printer.Info(fmt.Sprintf("Pushing %d projects...", len(summaries)))

	for _, summary := range summaries {
		project, err := manager.Projects.GetByID(summary.Name) // summary.Name is UUID in index
		if err != nil {
			printer.Warning(fmt.Sprintf("Failed to load project %s: %v", summary.Name, err))
			continue
		}

		if dryRun {
			printer.Info(fmt.Sprintf("  Would push project: %s (%s)", project.Name, project.ID))
		} else {
			// Convert to API type
			apiProject := api.ProjectToAPI(project)
			if _, err := client.PushProject(apiProject); err != nil {
				printer.Warning(fmt.Sprintf("Failed to push project %s: %v", project.Name, err))
				continue
			}
			printer.Info(fmt.Sprintf("  Pushed project: %s", project.Name))
		}
	}

	return nil
}

// pushConfigSheets pushes all local config sheets to remote
func (c *PushCommand) pushConfigSheets(manager *entities.Manager, client *api.Client, printer *output.Printer, dryRun bool) error {
	summaries, err := manager.ConfigSheets.List()
	if err != nil {
		return fmt.Errorf("failed to list config sheets: %w", err)
	}

	if len(summaries) == 0 {
		printer.Info("No config sheets to push")
		return nil
	}

	printer.Info(fmt.Sprintf("Pushing %d config sheets...", len(summaries)))

	for _, summary := range summaries {
		sheet, err := manager.ConfigSheets.GetByID(summary.Name) // summary.Name is UUID in index
		if err != nil {
			printer.Warning(fmt.Sprintf("Failed to load config sheet %s: %v", summary.Name, err))
			continue
		}

		if dryRun {
			printer.Info(fmt.Sprintf("  Would push config sheet: %s (%s)", sheet.Name, sheet.ID))
		} else {
			// Convert to API type
			apiSheet := api.ConfigSheetToAPI(sheet)
			if _, err := client.PushConfigSheet(apiSheet); err != nil {
				printer.Warning(fmt.Sprintf("Failed to push config sheet %s: %v", sheet.Name, err))
				continue
			}
			printer.Info(fmt.Sprintf("  Pushed config sheet: %s", sheet.Name))
		}
	}

	return nil
}