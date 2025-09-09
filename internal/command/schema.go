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

	"github.com/n1rna/ee-cli/internal/schema"
	"github.com/n1rna/ee-cli/internal/storage"
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
	return cmd
}

func (c *SchemaCommand) newShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show [schema-name]",
		Short: "Show details of a schema",
		Args:  cobra.ExactArgs(1),
		RunE:  c.runShow,
	}
}

func (c *SchemaCommand) newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show list of a schemas",
		Args:  cobra.ExactArgs(0),
		RunE:  c.runList,
	}
}

func (c *SchemaCommand) newDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [schema-name]",
		Short: "Delete a schema",
		Args:  cobra.ExactArgs(1),
		RunE:  c.runDelete,
	}
}

func (c *SchemaCommand) runCreate(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	schemaName := args[0]

	// Check if schema already exists
	schemas, err := uuidStorage.ListSchemas()
	if err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}

	for _, s := range schemas {
		if s.Name == schemaName {
			return fmt.Errorf("schema %s already exists", schemaName)
		}
	}

	// Check if we should import from file
	if importFile, _ := cmd.Flags().GetString("import"); importFile != "" {
		return c.importSchema(uuidStorage, schemaName, importFile)
	}

	// Check if we should create via CLI flags
	if variables, _ := cmd.Flags().GetStringSlice("variable"); len(variables) > 0 {
		description, _ := cmd.Flags().GetString("description")
		return c.createSchemaFromCLI(uuidStorage, schemaName, description, variables)
	}

	return c.createSchemaInteractively(uuidStorage, schemaName)
}

func (c *SchemaCommand) createSchemaInteractively(
	uuidStorage *storage.UUIDStorage,
	name string,
) error {
	fmt.Println("Creating new schema...")
	fmt.Println("For each variable, you'll need to specify:")
	fmt.Println("- Name (e.g., DATABASE_URL)")
	fmt.Println("- Type (string/number/boolean/url)")
	fmt.Println("- Regex pattern (optional)")
	fmt.Println("- Default value (optional)")
	fmt.Println("- Required flag (y/n)")
	fmt.Println()

	schemaObj := schema.NewSchema(name, "Schema created interactively", []schema.Variable{}, nil)

	for {
		var variable schema.Variable

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
		for _, v := range schemaObj.Variables {
			if v.Name == name {
				fmt.Printf("Variable %s already exists in schema\n", name)
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
			fmt.Printf("Invalid type %s, defaulting to string\n", varType)
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

		schemaObj.Variables = append(schemaObj.Variables, variable)
		fmt.Println()
	}

	if len(schemaObj.Variables) == 0 {
		return fmt.Errorf("schema must contain at least one variable")
	}

	// TODO: Implement validation for UUID storage if needed

	// Save schema
	if err := uuidStorage.SaveSchema(schemaObj); err != nil {
		return fmt.Errorf("failed to save schema: %w", err)
	}

	fmt.Printf(
		"Successfully created schema '%s' with %d variables\n",
		name,
		len(schemaObj.Variables),
	)
	return nil
}

// createSchemaFromCLI creates a schema from CLI flags
func (c *SchemaCommand) createSchemaFromCLI(
	uuidStorage *storage.UUIDStorage,
	name, description string,
	variableSpecs []string,
) error {
	fmt.Printf("Creating schema '%s' from CLI specifications...\n", name)

	variables := []schema.Variable{}

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
		fmt.Printf("  ✓ Added variable: %s (%s)\n", variable.Name, variable.Type)
	}

	if len(variables) == 0 {
		return fmt.Errorf("schema must contain at least one variable")
	}

	// Create schema object
	schemaObj := schema.NewSchema(name, description, variables, nil)

	// Save schema
	if err := uuidStorage.SaveSchema(schemaObj); err != nil {
		return fmt.Errorf("failed to save schema: %w", err)
	}

	fmt.Printf("✅ Successfully created schema '%s' with %d variables\n", name, len(variables))
	return nil
}

// parseVariableSpec parses a variable specification in the format: name:type:title:required[:default]
func (c *SchemaCommand) parseVariableSpec(spec string) (schema.Variable, error) {
	// Split into at most 5 parts to handle cases where default values contain colons
	parts := strings.SplitN(spec, ":", 5)
	if len(parts) < 4 {
		return schema.Variable{}, fmt.Errorf(
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
		return schema.Variable{}, fmt.Errorf("variable name cannot be empty")
	}

	// Validate type
	validTypes := map[string]bool{"string": true, "number": true, "boolean": true, "url": true}
	if !validTypes[varType] {
		return schema.Variable{}, fmt.Errorf(
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
		return schema.Variable{}, fmt.Errorf(
			"invalid required value '%s', must be true/false",
			requiredStr,
		)
	}

	// Parse default value (optional)
	var defaultValue string
	if len(parts) == 5 {
		defaultValue = strings.TrimSpace(parts[4])
	}

	return schema.Variable{
		Name:     name,
		Type:     varType,
		Title:    title,
		Required: required,
		Default:  defaultValue,
	}, nil
}

func (c *SchemaCommand) importSchema(
	uuidStorage *storage.UUIDStorage,
	name string,
	filename string,
) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	var schemaObj schema.Schema
	if err := yaml.Unmarshal(data, &schemaObj); err != nil {
		return fmt.Errorf("failed to parse schema file: %w", err)
	}

	// Convert to new schema format
	newSchema := schema.NewSchema(
		name,
		schemaObj.Description,
		schemaObj.Variables,
		schemaObj.Extends,
	)
	// TODO: Implement validation for UUID storage if needed

	// Save schema
	if err := uuidStorage.SaveSchema(newSchema); err != nil {
		return fmt.Errorf("failed to save schema: %w", err)
	}

	fmt.Printf(
		"Successfully imported schema '%s' with %d variables\n",
		name,
		len(newSchema.Variables),
	)
	return nil
}

func (c *SchemaCommand) runShow(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	schemaName := args[0]
	schemaObj, err := uuidStorage.LoadSchema(schemaName)
	if err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	fmt.Printf("Schema: %s\n", schemaObj.Name)
	fmt.Println("Variables:")
	fmt.Println("─────────")

	for _, v := range schemaObj.Variables {
		fmt.Printf("• %s\n", v.Name)
		fmt.Printf("  Type: %s\n", v.Type)
		if v.Regex != "" {
			fmt.Printf("  Pattern: %s\n", v.Regex)
		}
		if v.Default != "" {
			fmt.Printf("  Default: %s\n", v.Default)
		}
		fmt.Printf("  Required: %v\n", v.Required)
		fmt.Println()
	}

	return nil
}

func (c *SchemaCommand) runList(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	schemas, err := uuidStorage.ListSchemas()
	if err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}

	fmt.Println("Schemas:")
	fmt.Println("────────")

	for _, s := range schemas {
		fmt.Println(s.Name)
	}

	return nil
}

func (c *SchemaCommand) runDelete(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	schemaName := args[0]

	// Check if schema is in use by any projects
	projects, err := uuidStorage.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	for _, project := range projects {
		// Load project to check its environments
		projectObj, err := uuidStorage.LoadProject(project.Name)
		if err != nil {
			continue
		}

		for envName, envInfo := range projectObj.Environments {
			configSheetName := projectObj.GetConfigSheetName(envInfo.Name)
			configSheet, err := uuidStorage.LoadConfigSheet(configSheetName)
			if err != nil {
				continue
			}

			// Check if this schema is referenced
			if configSheet.Schema.Ref != "" {
				if schemaID := strings.TrimPrefix(configSheet.Schema.Ref, "#/schemas/"); schemaID != configSheet.Schema.Ref {
					if summary, err := uuidStorage.GetEntitySummary("schemas", schemaID); err == nil &&
						summary.Name == schemaName {
						return fmt.Errorf(
							"cannot delete schema: in use by project %s (environment: %s)",
							project.Name,
							envName,
						)
					}
				}
			}
		}
	}

	if err := uuidStorage.DeleteSchema(schemaName); err != nil {
		return fmt.Errorf("failed to delete schema: %w", err)
	}

	fmt.Printf("Successfully deleted schema '%s'\n", schemaName)
	return nil
}

func (c *SchemaCommand) newEditCommand() *cobra.Command {
	return &cobra.Command{
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
}

func (c *SchemaCommand) runEdit(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	schemaName := args[0]

	// Load the schema
	schemaObj, err := uuidStorage.LoadSchema(schemaName)
	if err != nil {
		return fmt.Errorf("failed to load schema '%s': %w", schemaName, err)
	}

	validator := func(data []byte) (interface{}, error) {
		var editedSchema schema.Schema
		if err := json.Unmarshal(data, &editedSchema); err != nil {
			return nil, fmt.Errorf("invalid JSON in edited file: %w", err)
		}

		// Preserve the original ID and timestamps if they weren't changed
		if editedSchema.ID == "" {
			editedSchema.ID = schemaObj.ID
		}
		if editedSchema.CreatedAt.IsZero() {
			editedSchema.CreatedAt = schemaObj.CreatedAt
		}

		// Validate the edited schema
		if editedSchema.Name == "" {
			return nil, fmt.Errorf("schema name cannot be empty")
		}

		return &editedSchema, nil
	}

	saver := func(entity interface{}) error {
		editedSchema := entity.(*schema.Schema)
		return uuidStorage.SaveSchema(editedSchema)
	}

	changeReporter := func(original, edited interface{}) {
		origSchema := original.(*schema.Schema)
		editedSchema := edited.(*schema.Schema)

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
		schemaObj,
		&BaseEditorCommands{},
		validator,
		saver,
		changeReporter,
	)
}
