// Package command contains CLI command implementations.
package command

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/output"
	"github.com/n1rna/ee-cli/internal/parser"
)

type SchemaCommand struct{}

func NewSchemaCommand(groupId string) *cobra.Command {
	sc := &SchemaCommand{}

	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Manage environment variable schemas",
		Long: `Create and manage schemas for environment variables.

Schemas define the structure and validation rules for environment variables.
Each variable can have a type, regex pattern, default value, and required flag.`,
		GroupID: groupId,
	}

	// Add subcommands
	cmd.AddCommand(
		sc.newCreateCommand(),
		sc.newShowCommand(),
		sc.newEditCommand(),
		sc.newDeleteCommand(),
		sc.newListCommand(),
	)

	return cmd
}

func (c *SchemaCommand) newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [schema-name]",
		Short: "Create a new schema",
		Long: `Create a new schema interactively, via CLI flags, or from a file.

Examples:
  # Interactive mode
  ee schema create api-schema

  # CLI mode with variables
  ee schema create web-service \
    --description 'Schema for web services' \
    --variable 'DATABASE_URL:string:Database connection:true' \
    --variable 'PORT:number:Server port:false:8080' \
    --variable 'DEBUG:boolean:Debug mode:false:false'

  # Import from file
  ee schema create api-schema --import schema.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: c.runCreate,
	}

	cmd.Flags().String("import", "", "Import schema definition from a YAML, JSON, or dotenv file")
	cmd.Flags().String("description", "", "Schema description")
	cmd.Flags().
		StringSlice("variable", []string{}, "Add variable in format 'name:type:title:required[:default]'")
	cmd.Flags().String("format", "table", "Output format (table, json)")
	cmd.Flags().Bool("quiet", false, "Suppress non-error output")

	return cmd
}

func (c *SchemaCommand) newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [schema-name]",
		Short: "Show details of a schema",
		Args:  cobra.ExactArgs(1),
		RunE:  c.runShow,
	}

	cmd.Flags().String("format", "table", "Output format (table, json)")

	return cmd
}

func (c *SchemaCommand) newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Show list of schemas",
		Args:  cobra.ExactArgs(0),
		RunE:  c.runList,
	}

	cmd.Flags().String("format", "table", "Output format (table, json)")

	return cmd
}

func (c *SchemaCommand) newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [schema-name]",
		Short: "Delete a schema",
		Args:  cobra.ExactArgs(1),
		RunE:  c.runDelete,
	}

	cmd.Flags().Bool("quiet", false, "Suppress non-error output")

	return cmd
}

func (c *SchemaCommand) newEditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit [schema-name]",
		Short: "Edit a schema using your preferred editor",
		Long: `Edit a schema using your preferred editor.

The editor is determined by the $EDITOR environment variable, falling back to 'vim' if not set.
The schema is presented as JSON for editing, and changes are validated and applied upon saving.

Examples:
  # Edit a schema
  ee schema edit my-schema`,
		Args: cobra.ExactArgs(1),
		RunE: c.runEdit,
	}

	return cmd
}

func (c *SchemaCommand) runCreate(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.Format(format), quiet)

	schemaName := args[0]
	schemaParser := parser.NewSchemaParser()

	var schemaData *parser.SchemaData
	var err error

	// Check if we should import from file
	if importFile, _ := cmd.Flags().GetString("import"); importFile != "" {
		schemaData, err = schemaParser.ParseFile(importFile)
		if err != nil {
			return fmt.Errorf("failed to parse schema from file: %w", err)
		}
	} else if variables, _ := cmd.Flags().GetStringSlice("variable"); len(variables) > 0 {
		// Create via CLI flags
		description, _ := cmd.Flags().GetString("description")
		schemaData, err = schemaParser.ParseCLISpecs(description, variables)
		if err != nil {
			return err
		}
		for _, v := range schemaData.Variables {
			printer.Info(fmt.Sprintf("Added variable: %s (%s)", v.Name, v.Type))
		}
	} else {
		// Interactive mode
		schemaData, err = schemaParser.ParseInteractive()
		if err != nil {
			return err
		}
	}

	// Create schema using the manager
	s, err := manager.Schemas.Create(
		schemaName,
		schemaData.Description,
		schemaData.Variables,
		schemaData.Extends,
	)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	printer.Success(
		fmt.Sprintf(
			"Successfully created schema '%s' with %d variables",
			schemaName,
			len(s.Variables),
		),
	)
	return printer.PrintSchema(s)
}

func (c *SchemaCommand) runShow(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	format, _ := cmd.Flags().GetString("format")
	printer := output.NewPrinter(output.Format(format), false)

	schemaName := args[0]
	s, err := manager.Schemas.Get(schemaName)
	if err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	return printer.PrintSchema(s)
}

func (c *SchemaCommand) runList(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	format, _ := cmd.Flags().GetString("format")
	printer := output.NewPrinter(output.Format(format), false)

	summaries, err := manager.Schemas.List()
	if err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}

	return printer.PrintSchemaList(summaries)
}

func (c *SchemaCommand) runDelete(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.FormatTable, quiet)

	schemaName := args[0]

	if err := manager.Schemas.Delete(schemaName); err != nil {
		return fmt.Errorf("failed to delete schema: %w", err)
	}

	printer.Success(fmt.Sprintf("Successfully deleted schema '%s'", schemaName))
	return nil
}

func (c *SchemaCommand) runEdit(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	schemaName := args[0]

	// Load the schema
	s, err := manager.Schemas.Get(schemaName)
	if err != nil {
		return fmt.Errorf("failed to load schema '%s': %w", schemaName, err)
	}

	validator := func(data []byte) (interface{}, error) {
		var editedSchema entities.Schema
		if err := json.Unmarshal(data, &editedSchema); err != nil {
			return nil, fmt.Errorf("invalid JSON in edited file: %w", err)
		}

		// Preserve the original ID and timestamps if they weren't changed
		if editedSchema.ID == "" {
			editedSchema.ID = s.ID
		}
		if editedSchema.CreatedAt.IsZero() {
			editedSchema.CreatedAt = s.CreatedAt
		}

		// Validate the edited schema
		if editedSchema.Name == "" {
			return nil, fmt.Errorf("schema name cannot be empty")
		}

		return &editedSchema, nil
	}

	saver := func(entity interface{}) error {
		editedSchema := entity.(*entities.Schema)
		return manager.Schemas.Save(editedSchema)
	}

	changeReporter := func(original, edited interface{}) {
		origSchema := original.(*entities.Schema)
		editedSchema := edited.(*entities.Schema)
		printer := output.NewPrinter(output.FormatTable, false)

		if origSchema.Name != editedSchema.Name {
			printer.PrintChange("Name", origSchema.Name, editedSchema.Name)
		}
		if origSchema.Description != editedSchema.Description {
			printer.PrintUpdate("Description updated")
		}
		if len(origSchema.Variables) != len(editedSchema.Variables) {
			printer.PrintChange("Variables",
				fmt.Sprintf("%d", len(origSchema.Variables)),
				fmt.Sprintf("%d", len(editedSchema.Variables)))
		}
	}

	return EditEntity(
		fmt.Sprintf("schema '%s'", schemaName),
		s,
		&BaseEditorCommands{},
		validator,
		saver,
		changeReporter,
	)
}
