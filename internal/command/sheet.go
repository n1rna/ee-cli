// Package command implements the ee sheet command for managing config sheets
package command

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/output"
	"github.com/n1rna/ee-cli/internal/parser"
)

// SheetCommand handles the ee sheet command
type SheetCommand struct{}

// NewSheetCommand creates a new ee sheet command
func NewSheetCommand(groupId string) *cobra.Command {
	sc := &SheetCommand{}

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
  # Create with CLI values
  ee sheet create my-config --schema web-service --value DATABASE_URL=postgres://localhost --value PORT=5432

  # Import from file
  ee sheet create my-config --import config.yaml

  # Create interactively (free-form)
  ee sheet create my-config --interactive

  # Create interactively with schema guidance
  ee sheet create my-config --schema web-service --interactive`,
		Args: cobra.ExactArgs(1),
		RunE: c.runCreate,
	}

	cmd.Flags().String("schema", "", "Schema to use for validation")
	cmd.Flags().String("description", "", "Sheet description")
	cmd.Flags().
		StringToString("value", map[string]string{}, "Set variable values (format: --value KEY=VALUE)")
	cmd.Flags().String("import", "", "Import values from a file (YAML, JSON, or dotenv)")
	cmd.Flags().Bool("interactive", false, "Create config sheet interactively")
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
	cliValues, _ := cmd.Flags().GetStringToString("value")
	importFile, _ := cmd.Flags().GetString("import")
	interactive, _ := cmd.Flags().GetBool("interactive")

	sheetParser := parser.NewSheetParser()
	var sheetData *parser.SheetData
	var err error

	// Parse values based on input method
	if interactive {
		// Interactive mode - if schema is provided, get variable names from it
		var schemaVarNames []string
		if schemaName != "" {
			s, err := manager.Schemas.Get(schemaName)
			if err != nil {
				return fmt.Errorf("schema '%s' not found: %w", schemaName, err)
			}
			for _, v := range s.Variables {
				schemaVarNames = append(schemaVarNames, v.Name)
			}
		}
		sheetData, err = sheetParser.ParseInteractive(schemaVarNames)
		if err != nil {
			return err
		}
	} else if importFile != "" {
		// File import mode
		sheetData, err = sheetParser.ParseFile(importFile)
		if err != nil {
			return fmt.Errorf("failed to import values from file: %w", err)
		}

		// Merge with CLI values (CLI values take precedence)
		if len(cliValues) > 0 {
			cliData, _ := sheetParser.ParseCLIValues(cliValues)
			sheetData = sheetParser.MergeValues(sheetData, cliData)
		}
	} else if len(cliValues) > 0 {
		// CLI values mode
		sheetData, err = sheetParser.ParseCLIValues(cliValues)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("must provide values via --value, --import, or --interactive flag")
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

	// Create standalone config sheet
	cs := entities.NewConfigSheet(sheetName, description, schemaRef, sheetData.Values)

	// Validate using manager's validator
	validator := manager.GetValidator()
	if err := validator.ValidateConfigSheet(cs); err != nil {
		return fmt.Errorf("config sheet validation failed: %w", err)
	}

	// Save the validated config sheet
	if err := manager.ConfigSheets.Save(cs); err != nil {
		return fmt.Errorf("failed to save config sheet: %w", err)
	}

	printer.Success(fmt.Sprintf("Successfully created config sheet '%s'", sheetName))
	return printer.PrintConfigSheet(cs)
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
	// Create printer with appropriate format for values export
	var printerFormat output.Format
	if format == "json" {
		printerFormat = output.FormatJSON
	} else {
		printerFormat = output.FormatTable
	}
	printer := output.NewPrinter(printerFormat, false)

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
		printer := output.NewPrinter(output.FormatTable, false)

		if origSheet.Name != editedSheet.Name {
			printer.PrintChange("Name", origSheet.Name, editedSheet.Name)
		}
		if origSheet.Description != editedSheet.Description {
			printer.PrintUpdate("Description updated")
		}
		if len(origSheet.Values) != len(editedSheet.Values) {
			printer.PrintChange("Values",
				fmt.Sprintf("%d", len(origSheet.Values)),
				fmt.Sprintf("%d", len(editedSheet.Values)))
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
