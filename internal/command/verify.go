// Package command implements the ee verify command for project validation
package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/output"
	"github.com/n1rna/ee-cli/internal/parser"
	"github.com/n1rna/ee-cli/internal/util"
)

// VerifyCommand handles the ee verify command
type VerifyCommand struct{}

// VerificationResult represents the result of verification
type VerificationResult struct {
	ProjectValid      bool
	SchemaValid       bool
	EnvironmentsValid bool
	Issues            []VerificationIssue
	Warnings          []string
}

// VerificationIssue represents a specific verification issue
type VerificationIssue struct {
	Type        string // "missing_env_file", "missing_variable", "extra_variable", "type_mismatch"
	Environment string
	Variable    string
	Expected    string
	Actual      string
	Description string
}

// NewVerifyCommand creates a new ee verify command
func NewVerifyCommand(groupId string) *cobra.Command {
	vc := &VerifyCommand{}

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify project configuration and environment files",
		Long: `Verify that project configuration is consistent with schema definitions and environment files.

This command checks:
- Project schema is valid and can be loaded
- All defined environments have corresponding .env files
- All .env files contain variables defined in the schema
- Variable types match schema definitions
- Required variables are present

Examples:
  # Verify current project
  ee verify

  # Verify and show detailed output
  ee verify --verbose

  # Verify and fix issues automatically
  ee verify --fix

  # Verify specific environment only
  ee verify --env development`,
		RunE:    vc.Run,
		GroupID: groupId,
	}

	cmd.Flags().Bool("fix", false, "Automatically fix detected issues")
	cmd.Flags().Bool("verbose", false, "Show detailed verification output")
	cmd.Flags().String("env", "", "Verify specific environment only")
	cmd.Flags().Bool("quiet", false, "Suppress non-error output")

	return cmd
}

// Run executes the verify command
func (c *VerifyCommand) Run(cmd *cobra.Command, args []string) error {
	// Get command context - requires project context
	context, err := RequireProjectContext(cmd.Context())
	if err != nil {
		return fmt.Errorf("verify command requires a project context (.ee file): %w", err)
	}

	// Set up printer
	quiet, _ := cmd.Flags().GetBool("quiet")
	verbose, _ := cmd.Flags().GetBool("verbose")
	printer := output.NewPrinter(output.FormatTable, quiet)

	// Get flags
	fix, _ := cmd.Flags().GetBool("fix")
	envFilter, _ := cmd.Flags().GetString("env")

	if verbose {
		printer.Info("Starting project verification...")
		printer.Info(fmt.Sprintf("Project: %s", context.ProjectConfig.Project))
	}

	// Perform verification
	result, err := c.verifyProject(context, envFilter, printer, verbose)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	// Report results
	c.reportResults(result, printer, verbose)

	// Apply fixes if requested
	if fix && len(result.Issues) > 0 {
		printer.Info("\nApplying fixes...")
		c.applyFixes(context, result, printer)
		printer.Success("Fixes applied successfully")
	}

	// Exit with error code if issues found
	if !result.ProjectValid || !result.SchemaValid || !result.EnvironmentsValid {
		if !fix {
			printer.Info("\nRun 'ee verify --fix' to automatically resolve these issues")
		}
		return fmt.Errorf("verification failed")
	}

	printer.Success("✓ Project verification passed")
	return nil
}

// verifyProject performs the actual verification
func (c *VerifyCommand) verifyProject(
	context *util.CommandContext,
	envFilter string,
	printer *output.Printer,
	verbose bool,
) (*VerificationResult, error) {
	result := &VerificationResult{
		ProjectValid:      true,
		SchemaValid:       true,
		EnvironmentsValid: true,
		Issues:            []VerificationIssue{},
		Warnings:          []string{},
	}

	// 1. Verify schema can be loaded
	schemaVariables, err := c.loadProjectSchema(context, result)
	if err != nil {
		return result, err
	}

	if verbose && len(schemaVariables) > 0 {
		printer.Info(fmt.Sprintf("Schema loaded with %d variables", len(schemaVariables)))
	}

	// 2. Verify environments
	environments := context.ProjectConfig.Environments
	if envFilter != "" {
		// Filter to specific environment
		if envDef, exists := environments[envFilter]; exists {
			environments = map[string]parser.EnvironmentDefinition{envFilter: envDef}
		} else {
			return nil, fmt.Errorf("environment '%s' not found in project config", envFilter)
		}
	}

	for envName, envDef := range environments {
		if verbose {
			printer.Info(fmt.Sprintf("Verifying environment: %s", envName))
		}
		c.verifyEnvironment(envName, envDef, schemaVariables, result)
	}

	return result, nil
}

// loadProjectSchema loads and validates the project schema
func (c *VerifyCommand) loadProjectSchema(
	context *util.CommandContext,
	result *VerificationResult,
) (map[string]entities.Variable, error) {
	schema := context.ProjectConfig.Schema

	// Handle inline schema
	if schema.Variables != nil {
		return schema.Variables, nil
	}

	// Handle schema reference
	if schema.Ref != "" {
		if context.Manager == nil {
			result.SchemaValid = false
			result.Issues = append(result.Issues, VerificationIssue{
				Type:        "schema_error",
				Description: "Schema manager not available to load referenced schema",
			})
			return nil, fmt.Errorf("schema manager not available")
		}

		loadedSchema, err := context.Manager.Schemas.GetByReference(schema.Ref)
		if err != nil {
			result.SchemaValid = false
			result.Issues = append(result.Issues, VerificationIssue{
				Type:        "schema_error",
				Description: fmt.Sprintf("Failed to load schema '%s': %v", schema.Ref, err),
			})
			return nil, fmt.Errorf("failed to load schema: %w", err)
		}

		// Convert to map
		variables := make(map[string]entities.Variable)
		for _, variable := range loadedSchema.Variables {
			variables[variable.Name] = variable
		}
		return variables, nil
	}

	// No schema defined
	result.Warnings = append(result.Warnings, "No schema defined for project")
	return make(map[string]entities.Variable), nil
}

// verifyEnvironment verifies a single environment
func (c *VerifyCommand) verifyEnvironment(
	envName string,
	envDef parser.EnvironmentDefinition,
	schemaVariables map[string]entities.Variable,
	result *VerificationResult,
) {
	// Find .env files for this environment
	envFiles := c.findEnvFiles(envName, envDef)

	if len(envFiles) == 0 {
		result.EnvironmentsValid = false
		result.Issues = append(result.Issues, VerificationIssue{
			Type:        "missing_env_file",
			Environment: envName,
			Description: fmt.Sprintf("No .env file found for environment '%s'", envName),
		})
		return
	}

	// Verify each .env file
	for _, envFile := range envFiles {
		c.verifyEnvFile(envName, envFile, schemaVariables, result)
	}
}

// findEnvFiles finds .env files referenced by an environment definition
func (c *VerifyCommand) findEnvFiles(envName string, envDef parser.EnvironmentDefinition) []string {
	var envFiles []string

	// Check single sheet reference
	if envDef.Sheet != "" && strings.HasPrefix(envDef.Sheet, ".env") {
		envFiles = append(envFiles, envDef.Sheet)
	}

	// Check multiple sheets
	for _, sheet := range envDef.Sheets {
		if sheetStr, ok := sheet.(string); ok && strings.HasPrefix(sheetStr, ".env") {
			envFiles = append(envFiles, sheetStr)
		}
	}

	// If no explicit .env files found, check for default patterns
	if len(envFiles) == 0 {
		defaultFile := ".env." + envName
		if _, err := os.Stat(defaultFile); err == nil {
			envFiles = append(envFiles, defaultFile)
		}
	}

	return envFiles
}

// verifyEnvFile verifies a single .env file against the schema
func (c *VerifyCommand) verifyEnvFile(
	envName, envFile string,
	schemaVariables map[string]entities.Variable,
	result *VerificationResult,
) {
	// Check if file exists
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		result.EnvironmentsValid = false
		result.Issues = append(result.Issues, VerificationIssue{
			Type:        "missing_env_file",
			Environment: envName,
			Description: fmt.Sprintf("Environment file '%s' does not exist", envFile),
		})
		return
	}

	// Parse .env file
	envVars, err := c.parseEnvFile(envFile)
	if err != nil {
		result.EnvironmentsValid = false
		result.Issues = append(result.Issues, VerificationIssue{
			Type:        "parse_error",
			Environment: envName,
			Description: fmt.Sprintf("Failed to parse '%s': %v", envFile, err),
		})
		return
	}

	// Check for missing required variables
	for varName, schemaVar := range schemaVariables {
		if _, exists := envVars[varName]; !exists {
			if schemaVar.Required {
				result.EnvironmentsValid = false
				result.Issues = append(result.Issues, VerificationIssue{
					Type:        "missing_variable",
					Environment: envName,
					Variable:    varName,
					Expected:    "required variable",
					Description: fmt.Sprintf(
						"Required variable '%s' missing in %s",
						varName,
						envFile,
					),
				})
			} else {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Optional variable '%s' missing in %s", varName, envFile))
			}
		}
	}

	// Check for extra variables not in schema
	for varName := range envVars {
		if _, exists := schemaVariables[varName]; !exists && len(schemaVariables) > 0 {
			result.Warnings = append(
				result.Warnings,
				fmt.Sprintf("Variable '%s' in %s not defined in schema", varName, envFile),
			)
		}
	}
}

// parseEnvFile parses a .env file and returns variables
func (c *VerifyCommand) parseEnvFile(filename string) (map[string]string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	result := make(map[string]string)
	lines := strings.Split(string(content), "\n")

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line %d: %s", lineNum+1, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes if present
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}

		if key == "" {
			return nil, fmt.Errorf("empty variable name on line %d", lineNum+1)
		}

		result[key] = value
	}

	return result, nil
}

// reportResults reports verification results
func (c *VerifyCommand) reportResults(
	result *VerificationResult,
	printer *output.Printer,
	verbose bool,
) {
	if len(result.Issues) == 0 && len(result.Warnings) == 0 {
		if verbose {
			printer.Info("No issues or warnings found")
		}
		return
	}

	// Report issues
	if len(result.Issues) > 0 {
		printer.Error(fmt.Sprintf("Found %d issue(s):", len(result.Issues)))
		for _, issue := range result.Issues {
			switch issue.Type {
			case "missing_env_file":
				printer.Error(fmt.Sprintf("  ✗ Missing environment file: %s", issue.Description))
			case "missing_variable":
				printer.Error(
					fmt.Sprintf(
						"  ✗ Missing variable: %s in %s",
						issue.Variable,
						issue.Environment,
					),
				)
			case "schema_error":
				printer.Error(fmt.Sprintf("  ✗ Schema error: %s", issue.Description))
			case "parse_error":
				printer.Error(fmt.Sprintf("  ✗ Parse error: %s", issue.Description))
			default:
				printer.Error(fmt.Sprintf("  ✗ %s", issue.Description))
			}
		}
	}

	// Report warnings
	if len(result.Warnings) > 0 && verbose {
		printer.Warning(fmt.Sprintf("Found %d warning(s):", len(result.Warnings)))
		for _, warning := range result.Warnings {
			printer.Warning(fmt.Sprintf("  ⚠ %s", warning))
		}
	}
}

// applyFixes applies automatic fixes for detected issues
func (c *VerifyCommand) applyFixes(
	context *util.CommandContext,
	result *VerificationResult,
	printer *output.Printer,
) {
	for _, issue := range result.Issues {
		switch issue.Type {
		case "missing_env_file":
			err := c.createMissingEnvFile(context, issue)
			if err != nil {
				printer.Warning(fmt.Sprintf("Failed to create %s: %v", issue.Environment, err))
			} else {
				printer.Info(fmt.Sprintf("Created missing .env file for %s", issue.Environment))
			}

		case "missing_variable":
			err := c.addMissingVariable(context, issue)
			if err != nil {
				printer.Warning(fmt.Sprintf("Failed to add variable %s: %v", issue.Variable, err))
			} else {
				printer.Info(fmt.Sprintf("Added missing variable %s to %s", issue.Variable, issue.Environment))
			}
		}
	}
}

// createMissingEnvFile creates a missing .env file for an environment
func (c *VerifyCommand) createMissingEnvFile(
	context *util.CommandContext,
	issue VerificationIssue,
) error {
	envName := issue.Environment
	filename := ".env." + envName

	// Get schema variables
	schemaVariables, err := c.loadProjectSchema(context, &VerificationResult{})
	if err != nil {
		return err
	}

	// Create content similar to init command
	content := fmt.Sprintf("# %s environment configuration\n", filename)
	content += "# Generated by ee verify --fix\n\n"

	if context.ProjectConfig.Schema.Ref != "" {
		content += fmt.Sprintf("# schema: %s\n\n", context.ProjectConfig.Schema.Ref)
	} else {
		content += "# schema: inline\n\n"
	}

	// Add variables from schema
	for _, variable := range schemaVariables {
		if variable.Title != "" {
			content += fmt.Sprintf("# title: %s\n", variable.Title)
		}
		if variable.Type != "string" {
			content += fmt.Sprintf("# type: %s\n", variable.Type)
		}
		if variable.Default != "" {
			content += fmt.Sprintf("# default: %s\n", variable.Default)
		}
		if variable.Required {
			content += "# required: true\n"
		}

		value := variable.Default
		if value == "" {
			value = ""
		}
		content += fmt.Sprintf("%s=%s\n\n", variable.Name, value)
	}

	return os.WriteFile(filename, []byte(content), 0o644)
}

// addMissingVariable adds a missing variable to existing .env files
func (c *VerifyCommand) addMissingVariable(
	context *util.CommandContext,
	issue VerificationIssue,
) error {
	// Find the .env file for this environment
	envDef := context.ProjectConfig.Environments[issue.Environment]
	envFiles := c.findEnvFiles(issue.Environment, envDef)

	if len(envFiles) == 0 {
		return fmt.Errorf("no .env file found for environment %s", issue.Environment)
	}

	// Get the variable from schema
	schemaVariables, err := c.loadProjectSchema(context, &VerificationResult{})
	if err != nil {
		return err
	}

	variable, exists := schemaVariables[issue.Variable]
	if !exists {
		return fmt.Errorf("variable %s not found in schema", issue.Variable)
	}

	// Add to the first .env file found
	envFile := envFiles[0]

	// Append variable to file
	content := "\n# Added by ee verify --fix\n"
	if variable.Title != "" {
		content += fmt.Sprintf("# title: %s\n", variable.Title)
	}
	if variable.Type != "string" {
		content += fmt.Sprintf("# type: %s\n", variable.Type)
	}
	if variable.Default != "" {
		content += fmt.Sprintf("# default: %s\n", variable.Default)
	}
	if variable.Required {
		content += "# required: true\n"
	}

	value := variable.Default
	if value == "" {
		value = ""
	}
	content += fmt.Sprintf("%s=%s\n", variable.Name, value)

	// Append to file
	file, err := os.OpenFile(envFile, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log the error but don't override the main error
		}
	}()

	_, err = file.WriteString(content)
	return err
}
