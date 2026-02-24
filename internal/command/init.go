// Package command implements the ee init command for project initialization
package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/config"
	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/output"
	"github.com/n1rna/ee-cli/internal/parser"
)

// InitCommand handles the ee init command
type InitCommand struct{}

// NewInitCommand creates a new ee init command
func NewInitCommand(groupId string) *cobra.Command {
	ic := &InitCommand{}

	cmd := &cobra.Command{
		Use:   "init [project-name]",
		Short: "Initialize a new ee project with .ee configuration file",
		Long: `Initialize a new ee project by creating a .ee configuration file in JSON format.

This command creates a .ee file in the current working directory with project
configuration including schema definitions, environments, and remote references.
Projects are now completely self-contained and portable.

Examples:
  # Initialize with current directory name
  ee init

  # Initialize with specific project name
  ee init my-api

  # Initialize with schema reference
  ee init my-api --schema web-service

  # Initialize with remote URL
  ee init my-api --remote https://api.ee.dev

  # Initialize with inline schema variables
  ee init my-api --var "PORT:number:Server port:false:3000" --var "NODE_ENV:string:Environment:true:development"
`,
		RunE:    ic.Run,
		GroupID: groupId,
	}

	cmd.Flags().
		StringP("schema", "s", "", "Schema reference to use (local://schema-name or remote://path)")
	cmd.Flags().StringP("remote", "r", "", "Remote URL for synchronization")
	cmd.Flags().
		StringSlice("var", []string{}, "Add schema variable (format:name:type:title:required:default)")
	cmd.Flags().BoolP("force", "f", false, "Overwrite existing .ee file")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress non-error output")

	return cmd
}

// Run executes the init command
func (c *InitCommand) Run(cmd *cobra.Command, args []string) error {
	// Set up printer
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.FormatTable, quiet)

	// Get flags
	schemaRef, _ := cmd.Flags().GetString("schema")
	remote, _ := cmd.Flags().GetString("remote")
	variables, _ := cmd.Flags().GetStringSlice("var")
	force, _ := cmd.Flags().GetBool("force")

	// Determine project name
	var projectName string
	if len(args) > 0 {
		projectName = args[0]
	} else {
		// Use current directory name
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projectName = filepath.Base(cwd)
		printer.Info(fmt.Sprintf("Using current directory name as project: %s", projectName))
	}

	// Check if .ee file already exists
	eeFile := config.ProjectConfigFileName
	if _, err := os.Stat(eeFile); err == nil && !force {
		return fmt.Errorf(
			"%s file already exists (use --force to overwrite)",
			config.ProjectConfigFileName,
		)
	}

	// Build schema configuration
	schema, err := c.buildSchemaConfig(schemaRef, variables)
	if err != nil {
		return fmt.Errorf("failed to build schema config: %w", err)
	}

	// Create project configuration
	projectConfig := &parser.ProjectConfig{
		Project: projectName,
		Remote:  remote,
		Schema:  schema,
		Environments: map[string]parser.EnvironmentDefinition{
			"development": {
				Sheets: []any{".env.development"},
			},
			"production": {
				Sheets: []any{".env.production"},
			},
		},
	}

	// Save .ee file
	err = parser.SaveProjectConfig(projectConfig, eeFile)
	if err != nil {
		return fmt.Errorf("failed to save %s file: %w", config.ProjectConfigFileName, err)
	}

	// Get entity manager from context for schema loading
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		printer.Warning("Entity manager not available - .env files will use basic schema")
	}

	// Create sample .env files
	err = c.createSampleEnvFiles(projectConfig, manager)
	if err != nil {
		printer.Warning(fmt.Sprintf("Failed to create sample .env files: %v", err))
	}

	printer.Success(fmt.Sprintf("✓ Initialized ee project: %s", projectName))
	printer.Info(fmt.Sprintf("✓ Created %s configuration file", config.ProjectConfigFileName))
	if len(projectConfig.Environments) > 0 {
		printer.Info("✓ Created sample .env files for environments")
	}

	// Show next steps
	printer.Info("\nNext steps:")
	printer.Info("  1. Edit .env files to add your environment variables")
	printer.Info("  2. Apply environment: ee apply development")
	printer.Info("  3. Run commands with environment: ee apply development -- npm start")

	return nil
}

// buildSchemaConfig builds the schema configuration from flags
func (c *InitCommand) buildSchemaConfig(
	schemaRef string,
	variables []string,
) (parser.ProjectConfigSchema, error) {
	schema := parser.ProjectConfigSchema{}

	// If schema reference is provided, use it
	if schemaRef != "" {
		schema.Ref = schemaRef
		return schema, nil
	}

	// If variables are provided, create inline schema
	if len(variables) > 0 {
		schema.Variables = make(map[string]entities.Variable)
		for _, varDef := range variables {
			variable, err := c.parseVariableDefinition(varDef)
			if err != nil {
				return schema, fmt.Errorf("invalid variable definition '%s': %w", varDef, err)
			}
			schema.Variables[variable.Name] = variable
		}
		return schema, nil
	}

	// Create default inline schema
	schema.Variables = map[string]entities.Variable{
		"NODE_ENV": {
			Name:     "NODE_ENV",
			Type:     "string",
			Title:    "Node environment",
			Required: false,
			Default:  "development",
		},
		"PORT": {
			Name:     "PORT",
			Type:     "number",
			Title:    "Server port",
			Required: false,
			Default:  "3000",
		},
		"DEBUG": {
			Name:     "DEBUG",
			Type:     "boolean",
			Title:    "Debug mode",
			Required: false,
			Default:  "false",
		},
	}

	return schema, nil
}

// parseVariableDefinition parses a variable definition string (name:type:title:required:default)
func (c *InitCommand) parseVariableDefinition(varDef string) (entities.Variable, error) {
	parts := strings.Split(varDef, ":")
	if len(parts) < 2 {
		return entities.Variable{}, fmt.Errorf("format should be name:type:title:required:default")
	}

	variable := entities.Variable{
		Name: parts[0],
		Type: parts[1],
	}

	if len(parts) > 2 {
		variable.Title = parts[2]
	}
	if len(parts) > 3 {
		variable.Required = parts[3] == "true"
	}
	if len(parts) > 4 {
		variable.Default = parts[4]
	}

	return variable, nil
}

// createSampleEnvFiles creates sample .env files for each environment
func (c *InitCommand) createSampleEnvFiles(
	projectConfig *parser.ProjectConfig,
	manager *entities.Manager,
) error {
	for envName, envDef := range projectConfig.Environments {
		// Find .env file references in the environment sheets
		for _, sheet := range envDef.Sheets {
			if sheetStr, ok := sheet.(string); ok &&
				(sheetStr == ".env" || sheetStr == ".env."+envName) {
				envFile := sheetStr
				if envFile == ".env" {
					envFile = ".env." + envName
				}

				// Create the .env file if it doesn't exist
				if _, err := os.Stat(envFile); os.IsNotExist(err) {
					err := c.createSampleEnvFile(envFile, projectConfig.Schema, manager)
					if err != nil {
						return fmt.Errorf("failed to create %s: %w", envFile, err)
					}
				}
			}
		}
	}
	return nil
}

// createSampleEnvFile creates a sample .env file with schema annotations
func (c *InitCommand) createSampleEnvFile(
	filename string,
	schema parser.ProjectConfigSchema,
	manager *entities.Manager,
) error {
	var variables map[string]entities.Variable

	// Handle schema reference vs inline schema
	if schema.Ref != "" {
		// Try to load the referenced schema
		loadedVars, err := c.loadSchemaVariables(schema.Ref, manager)
		if err != nil {
			return fmt.Errorf("failed to load schema '%s': %w", schema.Ref, err)
		}
		variables = loadedVars
	} else {
		variables = schema.Variables
	}

	// Prepare values map with default values
	values := make(map[string]string)
	varSlice := make([]entities.Variable, 0, len(variables))
	for _, variable := range variables {
		values[variable.Name] = variable.Default
		varSlice = append(varSlice, variable)
	}

	// Create schema entity for export
	schemaEntity := &entities.Schema{
		Variables: varSlice,
	}

	// Set schema reference in description if available
	if schema.Ref != "" {
		schemaEntity.Description = fmt.Sprintf("References schema: %s", schema.Ref)
	} else {
		schemaEntity.Description = "inline"
	}

	// Use the reusable dotenv parser to export the file
	dotenvParser := parser.NewAnnotatedDotEnvParser()
	return dotenvParser.ExportAnnotatedDotEnv(values, schemaEntity, filename)
}

// loadSchemaVariables loads variables from a schema reference using the schema manager
func (c *InitCommand) loadSchemaVariables(
	schemaRef string,
	manager *entities.Manager,
) (map[string]entities.Variable, error) {
	// Use the schema manager to load the schema by reference
	schema, err := manager.Schemas.GetByReference(schemaRef)
	if err != nil {
		return nil, err
	}

	// Convert schema variables to map
	variables := make(map[string]entities.Variable)
	for _, variable := range schema.Variables {
		variables[variable.Name] = variable
	}
	return variables, nil
}

// GetCurrentProject reads the project name from .ee file in current directory
func GetCurrentProject() (string, error) {
	projectConfig, err := parser.LoadProjectConfig()
	if err != nil {
		return "", err
	}
	return projectConfig.Project, nil
}

// GetCurrentRemote reads the remote URL from .ee file in current directory
func GetCurrentRemote() (string, error) {
	projectConfig, err := parser.LoadProjectConfig()
	if err != nil {
		return "", err
	}
	return projectConfig.Remote, nil
}
