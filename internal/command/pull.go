// Package command implements the ee pull command for pulling remote entities to local
package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/api"
	"github.com/n1rna/ee-cli/internal/manager"
	"github.com/n1rna/ee-cli/internal/output"
)

// PullCommand handles the ee pull command
type PullCommand struct{}

// NewPullCommand creates a new ee pull command
func NewPullCommand(groupId string) *cobra.Command {
	pc := &PullCommand{}

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull remote entities to local",
		Long: `Pull schemas and config sheets from the remote server to local storage.

This command synchronizes your local entities with the remote server, downloading
any changes or new entities that exist remotely but not locally.

Examples:
  # Pull all entities
  ee pull

  # Pull only schemas
  ee pull --schemas

  # Pull only config sheets
  ee pull --sheets

  # Dry run to see what would be pulled
  ee pull --dry-run`,
		RunE:    pc.Run,
		GroupID: groupId,
	}

	cmd.Flags().Bool("schemas", false, "Pull only schemas")
	cmd.Flags().Bool("sheets", false, "Pull only config sheets")
	cmd.Flags().Bool("dry-run", false, "Show what would be pulled without actually pulling")
	cmd.Flags().Bool("quiet", false, "Suppress non-error output")

	return cmd
}

// Run executes the pull command
func (c *PullCommand) Run(cmd *cobra.Command, args []string) error {
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
	sheetsOnly, _ := cmd.Flags().GetBool("sheets")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// If no specific flags, pull everything
	pullAll := !schemasOnly && !sheetsOnly

	if dryRun {
		printer.Info("Dry run mode - showing what would be pulled:")
	}

	// Pull schemas first (since sheets depend on them)
	if pullAll || schemasOnly {
		if err := c.pullSchemas(manager, client, printer, dryRun); err != nil {
			return fmt.Errorf("failed to pull schemas: %w", err)
		}
	}

	// Pull config sheets
	if pullAll || sheetsOnly {
		if err := c.pullConfigSheets(manager, client, printer, dryRun); err != nil {
			return fmt.Errorf("failed to pull config sheets: %w", err)
		}
	}

	if !dryRun {
		printer.Success("Successfully pulled all entities from remote")
	} else {
		printer.Info("Dry run completed")
	}

	return nil
}

// pullSchemas pulls all remote schemas to local
func (c *PullCommand) pullSchemas(
	manager *manager.Manager,
	client *api.Client,
	printer *output.Printer,
	dryRun bool,
) error {
	remoteSchemas, err := client.ListSchemas()
	if err != nil {
		return fmt.Errorf("failed to list remote schemas: %w", err)
	}

	if len(remoteSchemas) == 0 {
		printer.Info("No schemas to pull")
		return nil
	}

	printer.Info(fmt.Sprintf("Pulling %d schemas...", len(remoteSchemas)))

	pulled := 0
	skipped := 0

	for _, remoteSchema := range remoteSchemas {
		// Convert API schema to local type
		localSchemaData := api.SchemaFromAPI(&remoteSchema)

		// Check if schema already exists locally
		if existingSchema, err := manager.Schemas.GetByID(remoteSchema.GUID); err == nil {
			// Schema exists locally - check if remote is newer
			if !remoteSchema.UpdatedAt.Time().After(existingSchema.UpdatedAt) {
				skipped++
				continue
			}

			if dryRun {
				printer.Info(
					fmt.Sprintf(
						"  Would update schema: %s (%s)",
						remoteSchema.Name,
						remoteSchema.GUID,
					),
				)
			} else {
				// Update local schema with remote data
				if err := manager.Schemas.Save(localSchemaData); err != nil {
					printer.Warning(fmt.Sprintf("Failed to update schema %s: %v", remoteSchema.Name, err))
					continue
				}
				printer.Info(fmt.Sprintf("  Updated schema: %s", remoteSchema.Name))
				pulled++
			}
		} else {
			// Schema doesn't exist locally - create it
			if dryRun {
				printer.Info(fmt.Sprintf("  Would create schema: %s (%s)", remoteSchema.Name, remoteSchema.GUID))
			} else {
				if err := manager.Schemas.Save(localSchemaData); err != nil {
					printer.Warning(fmt.Sprintf("Failed to create schema %s: %v", remoteSchema.Name, err))
					continue
				}
				printer.Info(fmt.Sprintf("  Created schema: %s", remoteSchema.Name))
				pulled++
			}
		}
	}

	if !dryRun && pulled > 0 {
		printer.Info(fmt.Sprintf("Pulled %d schemas (%d skipped)", pulled, skipped))
	}

	return nil
}

// pullConfigSheets pulls all remote config sheets to local
func (c *PullCommand) pullConfigSheets(
	manager *manager.Manager,
	client *api.Client,
	printer *output.Printer,
	dryRun bool,
) error {
	remoteSheets, err := client.ListAllConfigSheets()
	if err != nil {
		return fmt.Errorf("failed to list remote config sheets: %w", err)
	}

	if len(remoteSheets) == 0 {
		printer.Info("No config sheets to pull")
		return nil
	}

	printer.Info(fmt.Sprintf("Pulling %d config sheets...", len(remoteSheets)))

	pulled := 0
	skipped := 0

	for _, remoteSheet := range remoteSheets {
		// Convert API config sheet to local type
		localSheetData := api.ConfigSheetFromAPI(&remoteSheet)

		// Check if config sheet already exists locally
		if existingSheet, err := manager.ConfigSheets.GetByID(remoteSheet.GUID); err == nil {
			// Config sheet exists locally - check if remote is newer
			if !remoteSheet.UpdatedAt.Time().After(existingSheet.UpdatedAt) {
				skipped++
				continue
			}

			if dryRun {
				printer.Info(
					fmt.Sprintf(
						"  Would update config sheet: %s (%s)",
						remoteSheet.Name,
						remoteSheet.GUID,
					),
				)
			} else {
				// Update local config sheet with remote data
				if err := manager.ConfigSheets.Save(localSheetData); err != nil {
					printer.Warning(fmt.Sprintf("Failed to update config sheet %s: %v", remoteSheet.Name, err))
					continue
				}
				printer.Info(fmt.Sprintf("  Updated config sheet: %s", remoteSheet.Name))
				pulled++
			}
		} else {
			// Config sheet doesn't exist locally - create it
			if dryRun {
				printer.Info(fmt.Sprintf("  Would create config sheet: %s (%s)", remoteSheet.Name, remoteSheet.GUID))
			} else {
				if err := manager.ConfigSheets.Save(localSheetData); err != nil {
					printer.Warning(fmt.Sprintf("Failed to create config sheet %s: %v", remoteSheet.Name, err))
					continue
				}
				printer.Info(fmt.Sprintf("  Created config sheet: %s", remoteSheet.Name))
				pulled++
			}
		}
	}

	if !dryRun && pulled > 0 {
		printer.Info(fmt.Sprintf("Pulled %d config sheets (%d skipped)", pulled, skipped))
	}

	return nil
}
