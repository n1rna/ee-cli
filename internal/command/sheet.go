// Package command implements the ee sheet command for managing config sheets
package command

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/output"
)

// SheetCommand handles the ee sheet command
type SheetCommand struct {
	reader *bufio.Reader
}

// NewSheetCommand creates a new ee sheet command
func NewSheetCommand(groupId string) *cobra.Command {
	sc := &SheetCommand{
		reader: bufio.NewReader(os.Stdin),
	}

	cmd := &cobra.Command{
		Use:   "sheet",
		Short: "Manage configuration sheets",
		Long: `Create and manage configuration sheets for environment variables.

Config sheets contain actual values for variables defined in schemas.
They can be standalone or associated with projects and environments.`,
		GroupID: groupId,
	}

	// Add subcommands
	cmd.AddCommand(
		sc.newCreateCommand(),
		sc.newShowCommand(),
		sc.newListCommand(),
		sc.newExportCommand(),
		sc.newEditCommand(),
		sc.newDeleteCommand(),
		sc.newSetCommand(),
		sc.newUnsetCommand(),
	)

	return cmd
}

func (c *SheetCommand) newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [sheet-name]",
		Short: "Create a new config sheet",
		Long: `Create a new configuration sheet with values for environment variables.

Examples:
  # Create standalone config sheet
  ee sheet create my-config --schema web-service

  # Create config sheet for project environment
  ee sheet create my-app-dev --project my-app --environment development --schema web-service

  # Create interactively
  ee sheet create my-config`,
		Args: cobra.ExactArgs(1),
		RunE: c.runCreate,
	}

	cmd.Flags().String("schema", "", "Schema to use for validation")
	cmd.Flags().String("project", "", "Project to associate this sheet with")
	cmd.Flags().String("environment", "", "Environment within the project")
	cmd.Flags().String("description", "", "Sheet description")
	cmd.Flags().
		StringToString("value", map[string]string{}, "Set variable values (format: --value KEY=VALUE)")
	cmd.Flags().String("import", "", "Import values from a file (YAML or JSON)")
	cmd.Flags().String("format", "table", "Output format (table, json)")
	cmd.Flags().Bool("quiet", false, "Suppress non-error output")

	return cmd
}

func (c *SheetCommand) newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [sheet-name]",
		Short: "Show details of a config sheet",
		Args:  cobra.ExactArgs(1),
		RunE:  c.runShow,
	}

	cmd.Flags().String("format", "table", "Output format (table, json)")

	return cmd
}

func (c *SheetCommand) newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configuration sheets",
		Args:  cobra.ExactArgs(0),
		RunE:  c.runList,
	}

	cmd.Flags().String("project", "", "Filter by project")
	cmd.Flags().String("environment", "", "Filter by environment")
	cmd.Flags().Bool("standalone", false, "Show only standalone sheets")
	cmd.Flags().String("format", "table", "Output format (table, json)")

	return cmd
}

func (c *SheetCommand) newExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [sheet-name]",
		Short: "Export config sheet values",
		Long: `Export configuration sheet values in various formats.

Examples:
  # Export as environment variables
  ee sheet export my-config --format env

  # Export as .env file
  ee sheet export my-config --format dotenv

  # Export as JSON
  ee sheet export my-config --format json`,
		Args: cobra.ExactArgs(1),
		RunE: c.runExport,
	}

	cmd.Flags().String("format", "env", "Export format (env, dotenv, json)")

	return cmd
}

func (c *SheetCommand) newEditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit [sheet-name]",
		Short: "Edit a config sheet using your preferred editor",
		Long: `Edit a config sheet using your preferred editor.

The editor is determined by the $EDITOR environment variable, falling back to 'vim' if not set.
The sheet is presented as JSON for editing, and changes are validated and applied upon saving.`,
		Args: cobra.ExactArgs(1),
		RunE: c.runEdit,
	}

	return cmd
}

func (c *SheetCommand) newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [sheet-name]",
		Short: "Delete a config sheet",
		Args:  cobra.ExactArgs(1),
		RunE:  c.runDelete,
	}

	cmd.Flags().Bool("quiet", false, "Suppress non-error output")

	return cmd
}

func (c *SheetCommand) newSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set [sheet-name] <variable-name> <value>",
		Short: "Set a variable value in a config sheet",
		Args:  cobra.ExactArgs(3),
		RunE:  c.runSet,
	}

	cmd.Flags().Bool("quiet", false, "Suppress non-error output")

	return cmd
}

func (c *SheetCommand) newUnsetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset [sheet-name] <variable-name>",
		Short: "Remove a variable value from a config sheet",
		Args:  cobra.ExactArgs(2),
		RunE:  c.runUnset,
	}

	cmd.Flags().Bool("quiet", false, "Suppress non-error output")

	return cmd
}

// Implementation methods

func (c *SheetCommand) runCreate(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.Format(format), quiet)

	sheetName := args[0]
	description, _ := cmd.Flags().GetString("description")
	schemaName, _ := cmd.Flags().GetString("schema")
	projectName, _ := cmd.Flags().GetString("project")
	envName, _ := cmd.Flags().GetString("environment")
	values, _ := cmd.Flags().GetStringToString("value")
	importFile, _ := cmd.Flags().GetString("import")

	// Import values from file if specified
	if importFile != "" {
		importedValues, err := c.importValuesFromFile(importFile)
		if err != nil {
			return fmt.Errorf("failed to import values: %w", err)
		}
		// Merge with CLI values (CLI values take precedence)
		for k, v := range values {
			importedValues[k] = v
		}
		values = importedValues
	}

	// Create schema reference
	var schemaRef entities.SchemaReference
	if schemaName != "" {
		s, err := manager.Schemas.Get(schemaName)
		if err != nil {
			return fmt.Errorf("schema '%s' not found: %w", schemaName, err)
		}
		schemaRef.Ref = "#/schemas/" + s.ID
	}

	var cs *entities.ConfigSheet

	if projectName != "" {
		// Create project-associated config sheet
		if envName == "" {
			return fmt.Errorf("environment name is required when project is specified")
		}

		p, err := manager.Projects.Get(projectName)
		if err != nil {
			return fmt.Errorf("project '%s' not found: %w", projectName, err)
		}

		// Create config sheet without validation first
		cs = entities.NewConfigSheetForProject(
			sheetName,
			description,
			schemaRef,
			p.ID,
			envName,
			values,
		)

		// Validate using manager's validator
		validator := manager.GetValidator()
		if err := validator.ValidateConfigSheet(cs); err != nil {
			return fmt.Errorf("config sheet validation failed: %w", err)
		}

		// Save the validated config sheet
		if err := manager.ConfigSheets.Save(cs); err != nil {
			return fmt.Errorf("failed to save config sheet: %w", err)
		}

		// Add environment to project if it doesn't exist
		if _, exists := p.Environments[envName]; !exists {
			_, err = manager.Projects.AddEnvironment(projectName, envName)
			if err != nil {
				printer.Warning(fmt.Sprintf("Failed to add environment to project: %v", err))
			}
		}
	} else {
		// Create standalone config sheet
		if envName != "" {
			return fmt.Errorf("environment can only be specified with a project")
		}

		// Create config sheet without validation first
		cs = entities.NewConfigSheet(sheetName, description, schemaRef, values)

		// Validate using manager's validator
		validator := manager.GetValidator()
		if err := validator.ValidateConfigSheet(cs); err != nil {
			return fmt.Errorf("config sheet validation failed: %w", err)
		}

		// Save the validated config sheet
		if err := manager.ConfigSheets.Save(cs); err != nil {
			return fmt.Errorf("failed to save config sheet: %w", err)
		}
	}

	printer.Success(fmt.Sprintf("Successfully created config sheet '%s'", sheetName))
	return printer.PrintConfigSheet(cs)
}

func (c *SheetCommand) importValuesFromFile(filename string) (map[string]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	values := make(map[string]string)

	// Try YAML first, then JSON
	if err := yaml.Unmarshal(data, &values); err != nil {
		if err := json.Unmarshal(data, &values); err != nil {
			return nil, fmt.Errorf("file is neither valid YAML nor JSON")
		}
	}

	return values, nil
}

func (c *SheetCommand) runShow(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	format, _ := cmd.Flags().GetString("format")
	printer := output.NewPrinter(output.Format(format), false)

	sheetName := args[0]
	cs, err := manager.ConfigSheets.Get(sheetName)
	if err != nil {
		return fmt.Errorf("failed to load config sheet: %w", err)
	}

	return printer.PrintConfigSheet(cs)
}

func (c *SheetCommand) runList(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	format, _ := cmd.Flags().GetString("format")
	printer := output.NewPrinter(output.Format(format), false)

	// Build filters
	filters := make(map[string]string)
	if projectName, _ := cmd.Flags().GetString("project"); projectName != "" {
		// Resolve project name to UUID
		p, err := manager.Projects.Get(projectName)
		if err != nil {
			return fmt.Errorf("project '%s' not found: %w", projectName, err)
		}
		filters["project"] = p.ID
	}

	if envName, _ := cmd.Flags().GetString("environment"); envName != "" {
		filters["environment"] = envName
	}

	if standalone, _ := cmd.Flags().GetBool("standalone"); standalone {
		filters["standalone"] = "true"
	}

	summaries, err := manager.ConfigSheets.ListWithFilters(filters)
	if err != nil {
		return fmt.Errorf("failed to list config sheets: %w", err)
	}

	return printer.PrintConfigSheetList(summaries)
}

func (c *SheetCommand) runExport(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	format, _ := cmd.Flags().GetString("format")
	printer := output.NewPrinter(output.FormatTable, false)

	sheetName := args[0]
	cs, err := manager.ConfigSheets.Get(sheetName)
	if err != nil {
		return fmt.Errorf("failed to load config sheet: %w", err)
	}

	switch format {
	case "env":
		return printer.PrintEnvironmentExport(cs.Values)
	case "dotenv":
		return printer.PrintDotEnv(cs.Values)
	case "json":
		return printer.PrintValues(cs.Values)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

func (c *SheetCommand) runEdit(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	sheetName := args[0]

	// Load the config sheet
	cs, err := manager.ConfigSheets.Get(sheetName)
	if err != nil {
		return fmt.Errorf("failed to load config sheet '%s': %w", sheetName, err)
	}

	validator := func(data []byte) (interface{}, error) {
		var editedSheet entities.ConfigSheet
		if err := json.Unmarshal(data, &editedSheet); err != nil {
			return nil, fmt.Errorf("invalid JSON in edited file: %w", err)
		}

		// Preserve the original ID and timestamps if they weren't changed
		if editedSheet.ID == "" {
			editedSheet.ID = cs.ID
		}
		if editedSheet.CreatedAt.IsZero() {
			editedSheet.CreatedAt = cs.CreatedAt
		}

		// Validate the edited sheet
		if editedSheet.Name == "" {
			return nil, fmt.Errorf("config sheet name cannot be empty")
		}

		return &editedSheet, nil
	}

	saver := func(entity interface{}) error {
		editedSheet := entity.(*entities.ConfigSheet)
		return manager.ConfigSheets.Save(editedSheet)
	}

	changeReporter := func(original, edited interface{}) {
		origSheet := original.(*entities.ConfigSheet)
		editedSheet := edited.(*entities.ConfigSheet)

		if origSheet.Name != editedSheet.Name {
			fmt.Printf("  Name: %s → %s\n", origSheet.Name, editedSheet.Name)
		}
		if origSheet.Description != editedSheet.Description {
			fmt.Printf("  Description updated\n")
		}
		if len(origSheet.Values) != len(editedSheet.Values) {
			fmt.Printf(
				"  Values: %d → %d\n",
				len(origSheet.Values),
				len(editedSheet.Values),
			)
		}
	}

	return EditEntity(
		fmt.Sprintf("config sheet '%s'", sheetName),
		cs,
		&BaseEditorCommands{},
		validator,
		saver,
		changeReporter,
	)
}

func (c *SheetCommand) runDelete(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.FormatTable, quiet)

	sheetName := args[0]

	if err := manager.ConfigSheets.Delete(sheetName); err != nil {
		return fmt.Errorf("failed to delete config sheet: %w", err)
	}

	printer.Success(fmt.Sprintf("Successfully deleted config sheet '%s'", sheetName))
	return nil
}

func (c *SheetCommand) runSet(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.FormatTable, quiet)

	sheetName := args[0]
	varName := args[1]
	value := args[2]

	_, err := manager.ConfigSheets.SetValue(sheetName, varName, value)
	if err != nil {
		return fmt.Errorf("failed to set value: %w", err)
	}

	printer.Success(fmt.Sprintf("Set %s=%s in config sheet '%s'", varName, value, sheetName))
	return nil
}

func (c *SheetCommand) runUnset(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.FormatTable, quiet)

	sheetName := args[0]
	varName := args[1]

	_, err := manager.ConfigSheets.UnsetValue(sheetName, varName)
	if err != nil {
		return fmt.Errorf("failed to unset value: %w", err)
	}

	printer.Success(fmt.Sprintf("Unset %s in config sheet '%s'", varName, sheetName))
	return nil
}
