// internal/command/new.go
package command

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/n1rna/menv/internal/schema"
	"github.com/spf13/cobra"
)

type NewCommand struct {
	reader *bufio.Reader
}

func NewNewCommand() *cobra.Command {
	nc := &NewCommand{
		reader: bufio.NewReader(os.Stdin),
	}

	cmd := &cobra.Command{
		Use:   "new [project-name]",
		Short: "Create a new project or environment",
		Args:  cobra.MaximumNArgs(1),
		RunE:  nc.Run,
	}

	cmd.Flags().String("env", "", "Create a new environment for an existing project")
	cmd.Flags().String("schema", "", "Use specified schema (for new projects)")

	return cmd
}

func (c *NewCommand) Run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("project name is required")
	}

	// Get storage from context
	storage := GetStorage(cmd.Context())
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	projectName := args[0]
	envFlag, _ := cmd.Flags().GetString("env")
	schemaFlag, _ := cmd.Flags().GetString("schema")

	if envFlag != "" {
		return c.createNewEnvironment(cmd.Context(), projectName, envFlag)
	}

	return c.createNewProject(cmd.Context(), projectName, schemaFlag)
}

func (c *NewCommand) createNewProject(ctx context.Context, projectName string, schemaName string) error {
	storage := GetStorage(ctx)
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// If schema name not provided, create new schema
	if schemaName == "" {
		var err error
		schemaName, err = c.createNewSchema(ctx, projectName+"-schema")
		if err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}
	}

	// Verify schema exists
	_, err := storage.LoadSchema(schemaName)
	if err != nil {
		return fmt.Errorf("schema %s not found: %w", schemaName, err)
	}

	// Create default environment
	fmt.Print("Enter name for initial environment (default 'development'): ")
	envName, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read environment name: %w", err)
	}

	envName = strings.TrimSpace(envName)
	if envName == "" {
		envName = "development"
	}

	configSheet := &schema.ConfigSheet{
		ProjectName: projectName,
		EnvName:     envName,
		Schema:      schemaName,
		Values:      make(map[string]string),
	}

	if err := storage.SaveConfigSheet(configSheet); err != nil {
		return fmt.Errorf("failed to save config sheet: %w", err)
	}

	fmt.Printf("Successfully created project '%s' with environment '%s'\n", projectName, envName)
	return nil
}

func (c *NewCommand) createNewEnvironment(ctx context.Context, projectName, envName string) error {
	storage := GetStorage(ctx)
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// Check if project exists
	projects, err := storage.ListProjects()
	projectExists := false
	for _, p := range projects {
		if p == projectName {
			projectExists = true
			break
		}
	}

	if !projectExists {
		return fmt.Errorf("project %s does not exist", projectName)
	}

	// Get existing environments to check for duplicates
	envs, err := storage.ListEnvironments(projectName)
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	for _, env := range envs {
		if env == envName {
			return fmt.Errorf("environment %s already exists for project %s", envName, projectName)
		}
	}

	// Load an existing environment to get the schema
	existingEnv := envs[0]
	existing, err := storage.LoadConfigSheet(projectName, existingEnv)
	if err != nil {
		return fmt.Errorf("failed to load existing config: %w", err)
	}

	// Create new environment with same schema
	configSheet := &schema.ConfigSheet{
		ProjectName: projectName,
		EnvName:     envName,
		Schema:      existing.Schema,
		Values:      make(map[string]string),
	}

	if err := storage.SaveConfigSheet(configSheet); err != nil {
		return fmt.Errorf("failed to save config sheet: %w", err)
	}

	fmt.Printf("Successfully created environment '%s' for project '%s'\n", envName, projectName)
	return nil
}

func (c *NewCommand) createNewSchema(ctx context.Context, name string) (string, error) {
	storage := GetStorage(ctx)
	if storage == nil {
		return "", fmt.Errorf("storage not initialized")
	}

	fmt.Println("Creating new schema...")

	_schema := &schema.Schema{
		Name:      name,
		Variables: []schema.Variable{},
	}

	for {
		var variable schema.Variable

		fmt.Print("Enter variable name (or empty to finish): ")
		name, err := c.reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read variable name: %w", err)
		}

		name = strings.TrimSpace(name)
		if name == "" {
			break
		}

		variable.Name = name

		fmt.Print("Enter variable type (string/number/boolean/url): ")
		varType, err := c.reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read variable type: %w", err)
		}

		variable.Type = strings.TrimSpace(varType)

		fmt.Print("Enter regex pattern (optional): ")
		regex, err := c.reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read regex pattern: %w", err)
		}

		regex = strings.TrimSpace(regex)
		if regex != "" {
			variable.Regex = regex
		}

		fmt.Print("Enter default value (optional): ")
		defaultVal, err := c.reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read default value: %w", err)
		}

		defaultVal = strings.TrimSpace(defaultVal)
		if defaultVal != "" {
			variable.Default = defaultVal
		}

		fmt.Print("Is this variable required? (y/N): ")
		required, err := c.reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read required flag: %w", err)
		}

		required = strings.TrimSpace(strings.ToLower(required))
		variable.Required = required == "y" || required == "yes"

		_schema.Variables = append(_schema.Variables, variable)
	}

	// Validate and save schema
	validator := schema.NewValidator()
	if err := validator.ValidateSchema(_schema); err != nil {
		return "", fmt.Errorf("invalid schema: %w", err)
	}

	if err := storage.SaveSchema(_schema); err != nil {
		return "", fmt.Errorf("failed to save schema: %w", err)
	}

	fmt.Printf("Successfully created schema '%s'\n", name)
	return name, nil
}
