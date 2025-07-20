// Package command contains CLI command implementations.
package command

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/n1rna/menv/internal/schema"
	"github.com/n1rna/menv/internal/storage"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type SchemaCommand struct {
	reader *bufio.Reader
}

func NewSchemaCommand() *cobra.Command {
	sc := &SchemaCommand{
		reader: bufio.NewReader(os.Stdin),
	}

	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Manage environment variable schemas",
		Long: `Create and manage schemas for environment variables.

Schemas define the structure and validation rules for environment variables.
Each variable can have a type, regex pattern, default value, and required flag.`,
	}

	// Add subcommands
	cmd.AddCommand(
		sc.newCreateCommand(),
		sc.newShowCommand(),
		sc.newDeleteCommand(),
		sc.newListCommand(),
	)

	return cmd
}

func (c *SchemaCommand) newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [schema-name]",
		Short: "Create a new schema",
		Long: `Create a new schema interactively.
		
Example:
  menv schema create api-schema`,
		Args: cobra.ExactArgs(1),
		RunE: c.runCreate,
	}

	cmd.Flags().String("import", "", "Import schema definition from a YAML file")
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
	storage := GetStorage(cmd.Context())
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	schemaName := args[0]

	// Check if schema already exists
	schemas, err := storage.ListSchemas()
	if err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}

	for _, s := range schemas {
		if s == schemaName {
			return fmt.Errorf("schema %s already exists", schemaName)
		}
	}

	// Check if we should import from file
	if importFile, _ := cmd.Flags().GetString("import"); importFile != "" {
		return c.importSchema(storage, schemaName, importFile)
	}

	return c.createSchemaInteractively(storage, schemaName)
}

func (c *SchemaCommand) createSchemaInteractively(storage *storage.Storage, name string) error {
	fmt.Println("Creating new schema...")
	fmt.Println("For each variable, you'll need to specify:")
	fmt.Println("- Name (e.g., DATABASE_URL)")
	fmt.Println("- Type (string/number/boolean/url)")
	fmt.Println("- Regex pattern (optional)")
	fmt.Println("- Default value (optional)")
	fmt.Println("- Required flag (y/n)")
	fmt.Println()

	schemaObj := &schema.Schema{
		Name:      name,
		Variables: []schema.Variable{},
	}

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

	// Validate schema
	validator := schema.NewValidator(storage)
	if err := validator.ValidateSchema(schemaObj); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	// Save schema
	if err := storage.SaveSchema(schemaObj); err != nil {
		return fmt.Errorf("failed to save schema: %w", err)
	}

	fmt.Printf("Successfully created schema '%s' with %d variables\n", name, len(schemaObj.Variables))
	return nil
}

func (c *SchemaCommand) importSchema(storage *storage.Storage, name string, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	var schemaObj schema.Schema
	if err := yaml.Unmarshal(data, &schemaObj); err != nil {
		return fmt.Errorf("failed to parse schema file: %w", err)
	}

	// Override name with provided name
	schemaObj.Name = name

	// Validate schema
	validator := schema.NewValidator(storage)
	if err := validator.ValidateSchema(&schemaObj); err != nil {
		return fmt.Errorf("invalid schema in import file: %w", err)
	}

	// Save schema
	if err := storage.SaveSchema(&schemaObj); err != nil {
		return fmt.Errorf("failed to save schema: %w", err)
	}

	fmt.Printf("Successfully imported schema '%s' with %d variables\n", name, len(schemaObj.Variables))
	return nil
}

func (c *SchemaCommand) runShow(cmd *cobra.Command, args []string) error {
	storage := GetStorage(cmd.Context())
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	schemaName := args[0]
	schemaObj, err := storage.LoadSchema(schemaName)
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
	storage := GetStorage(cmd.Context())
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	schemas, err := storage.ListSchemas()
	if err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}

	fmt.Println("Schemas:")
	fmt.Println("────────")

	for _, s := range schemas {
		fmt.Println(s)
	}

	return nil
}

func (c *SchemaCommand) runDelete(cmd *cobra.Command, args []string) error {
	storage := GetStorage(cmd.Context())
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	schemaName := args[0]

	// Check if schema is in use by any projects
	projects, err := storage.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	for _, project := range projects {
		envs, err := storage.ListEnvironments(project)
		if err != nil {
			return fmt.Errorf("failed to list environments for project %s: %w", project, err)
		}

		for _, env := range envs {
			configSheet, err := storage.LoadConfigSheet(project, env)
			if err != nil {
				continue
			}

			if configSheet.Schema == schemaName {
				return fmt.Errorf("cannot delete schema: in use by project %s (environment: %s)", project, env)
			}
		}
	}

	if err := storage.DeleteSchema(schemaName); err != nil {
		return fmt.Errorf("failed to delete schema: %w", err)
	}

	fmt.Printf("Successfully deleted schema '%s'\n", schemaName)
	return nil
}
