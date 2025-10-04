// Package command contains CLI command implementations.
package command

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/output"
)

type SchemaCommand struct {
	reader *bufio.Reader
}

func NewSchemaCommand(groupId string) *cobra.Command {
	sc := &SchemaCommand{
		reader: bufio.NewReader(os.Stdin),
	}

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

	cmd.Flags().String("import", "", "Import schema definition from a YAML file")
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

	// Check if we should import from file
	if importFile, _ := cmd.Flags().GetString("import"); importFile != "" {
		return c.importSchema(manager, printer, schemaName, importFile)
	}

	// Check if we should create via CLI flags
	if variables, _ := cmd.Flags().GetStringSlice("variable"); len(variables) > 0 {
		description, _ := cmd.Flags().GetString("description")
		return c.createSchemaFromCLI(manager, printer, schemaName, description, variables)
	}

	return c.createSchemaInteractively(manager, printer, schemaName)
}

func (c *SchemaCommand) createSchemaInteractively(
	manager *entities.Manager,
	printer *output.Printer,
	name string,
) error {
	printer.Info("Creating new schema...")
	printer.Info("For each variable, you'll need to specify:")
	printer.Info("- Name (e.g., DATABASE_URL)")
	printer.Info("- Type (string/number/boolean/url)")
	printer.Info("- Regex pattern (optional)")
	printer.Info("- Default value (optional)")
	printer.Info("- Required flag (y/n)")

	var variables []entities.Variable

	for {
		var variable entities.Variable

		fmt.Print("Enter variable name (or empty to finish): ")
		name, err := c.reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read variable name: %w", err)
		}

		name = strings.TrimSpace(name)
		if name == "" {
			break
		}

		// Check for duplicate variable names
		for _, v := range variables {
			if v.Name == name {
				printer.Warning(fmt.Sprintf("Variable %s already exists in schema", name))
				continue
			}
		}

		variable.Name = name

		fmt.Print("Enter variable type (string/number/boolean/url): ")
		varType, err := c.reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read variable type: %w", err)
		}

		varType = strings.TrimSpace(strings.ToLower(varType))
		switch varType {
		case "string", "number", "boolean", "url":
			variable.Type = varType
		default:
			printer.Warning(fmt.Sprintf("Invalid type %s, defaulting to string", varType))
			variable.Type = "string"
		}

		fmt.Print("Enter regex pattern (optional): ")
		regex, err := c.reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read regex pattern: %w", err)
		}

		regex = strings.TrimSpace(regex)
		if regex != "" {
			variable.Regex = regex
		}

		fmt.Print("Enter default value (optional): ")
		defaultVal, err := c.reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read default value: %w", err)
		}

		defaultVal = strings.TrimSpace(defaultVal)
		if defaultVal != "" {
			variable.Default = defaultVal
		}

		fmt.Print("Is this variable required? (y/N): ")
		required, err := c.reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read required flag: %w", err)
		}

		required = strings.TrimSpace(strings.ToLower(required))
		variable.Required = required == "y" || required == "yes"

		variables = append(variables, variable)
	}

	if len(variables) == 0 {
		return fmt.Errorf("schema must contain at least one variable")
	}

	// Create schema using the manager
	s, err := manager.Schemas.Create(name, "Schema created interactively", variables, nil)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	printer.Success(
		fmt.Sprintf("Successfully created schema '%s' with %d variables", name, len(variables)),
	)
	return printer.PrintSchema(s)
}

// createSchemaFromCLI creates a schema from CLI flags
func (c *SchemaCommand) createSchemaFromCLI(
	manager *entities.Manager,
	printer *output.Printer,
	name, description string,
	variableSpecs []string,
) error {
	printer.Info(fmt.Sprintf("Creating schema '%s' from CLI specifications...", name))

	variables := []entities.Variable{}

	// Parse each variable specification
	for _, varSpec := range variableSpecs {
		variable, err := c.parseVariableSpec(varSpec)
		if err != nil {
			return fmt.Errorf("invalid variable specification '%s': %w", varSpec, err)
		}

		// Check for duplicate variable names
		for _, existingVar := range variables {
			if existingVar.Name == variable.Name {
				return fmt.Errorf("duplicate variable name '%s'", variable.Name)
			}
		}

		variables = append(variables, variable)
		printer.Info(fmt.Sprintf("Added variable: %s (%s)", variable.Name, variable.Type))
	}

	if len(variables) == 0 {
		return fmt.Errorf("schema must contain at least one variable")
	}

	// Create schema using the manager
	s, err := manager.Schemas.Create(name, description, variables, nil)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	printer.Success(
		fmt.Sprintf("Successfully created schema '%s' with %d variables", name, len(variables)),
	)
	return printer.PrintSchema(s)
}

// parseVariableSpec parses a variable specification in the format: name:type:title:required[:default]
func (c *SchemaCommand) parseVariableSpec(spec string) (entities.Variable, error) {
	// Split into at most 5 parts to handle cases where default values contain colons
	parts := strings.SplitN(spec, ":", 5)
	if len(parts) < 4 {
		return entities.Variable{}, fmt.Errorf(
			"format should be 'name:type:title:required[:default]', got %d parts",
			len(parts),
		)
	}

	name := strings.TrimSpace(parts[0])
	varType := strings.TrimSpace(strings.ToLower(parts[1]))
	title := strings.TrimSpace(parts[2])
	requiredStr := strings.TrimSpace(strings.ToLower(parts[3]))

	// Validate name
	if name == "" {
		return entities.Variable{}, fmt.Errorf("variable name cannot be empty")
	}

	// Validate type
	validTypes := map[string]bool{"string": true, "number": true, "boolean": true, "url": true}
	if !validTypes[varType] {
		return entities.Variable{}, fmt.Errorf(
			"invalid type '%s', must be one of: string, number, boolean, url",
			varType,
		)
	}

	// Parse required flag
	var required bool
	switch requiredStr {
	case "true", "t", "1", "yes", "y":
		required = true
	case "false", "f", "0", "no", "n":
		required = false
	default:
		return entities.Variable{}, fmt.Errorf(
			"invalid required value '%s', must be true/false",
			requiredStr,
		)
	}

	// Parse default value (optional)
	var defaultValue string
	if len(parts) == 5 {
		defaultValue = strings.TrimSpace(parts[4])
	}

	return entities.Variable{
		Name:     name,
		Type:     varType,
		Title:    title,
		Required: required,
		Default:  defaultValue,
	}, nil
}

func (c *SchemaCommand) importSchema(
	manager *entities.Manager,
	printer *output.Printer,
	name string,
	filename string,
) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	var schemaObj entities.Schema
	if err := yaml.Unmarshal(data, &schemaObj); err != nil {
		return fmt.Errorf("failed to parse schema file: %w", err)
	}

	// Create schema using the manager
	s, err := manager.Schemas.Create(
		name,
		schemaObj.Description,
		schemaObj.Variables,
		schemaObj.Extends,
	)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	printer.Success(
		fmt.Sprintf("Successfully imported schema '%s' with %d variables", name, len(s.Variables)),
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

	// TODO: Add dependency checking - see if schema is in use by projects/config sheets

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

		if origSchema.Name != editedSchema.Name {
			fmt.Printf("  Name: %s → %s\n", origSchema.Name, editedSchema.Name)
		}
		if origSchema.Description != editedSchema.Description {
			fmt.Printf("  Description updated\n")
		}
		if len(origSchema.Variables) != len(editedSchema.Variables) {
			fmt.Printf(
				"  Variables: %d → %d\n",
				len(origSchema.Variables),
				len(editedSchema.Variables),
			)
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
