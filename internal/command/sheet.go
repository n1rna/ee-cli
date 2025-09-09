// Package command implements the ee sheet command for managing config sheets
// This implements the new config sheet management as specified in docs/entities.md
package command

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/n1rna/ee-cli/internal/schema"
	"github.com/n1rna/ee-cli/internal/storage"
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
	)

	return cmd
}

// newCreateCommand creates the sheet create subcommand
func (sc *SheetCommand) newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [sheet-name]",
		Short: "Create a new configuration sheet",
		Long: `Create a new configuration sheet interactively or by importing from files.

Examples:
  # Create sheet interactively
  ee sheet create my-config

  # Create sheet for project environment
  ee sheet create --env development

  # Import from .env file
  ee sheet create my-config --import-env config.env

  # Import from JSON file
  ee sheet create my-config --import-json config.json

  # Create sheet with specific schema
  ee sheet create my-config --schema api-schema
`,
		Args: cobra.MaximumNArgs(1),
		RunE: sc.runCreate,
	}

	cmd.Flags().String("env", "", "Environment name (creates sheet for current project)")
	cmd.Flags().String("schema", "", "Schema name to use for validation")
	cmd.Flags().String("import-env", "", "Import values from .env file")
	cmd.Flags().String("import-json", "", "Import values from JSON file")
	cmd.Flags().Bool("standalone", false, "Create standalone sheet (not associated with project)")
	cmd.Flags().Bool("interactive", false, "Create sheet interactively even when importing")

	// Make import flags mutually exclusive
	cmd.MarkFlagsMutuallyExclusive("import-env", "import-json")

	return cmd
}

// newShowCommand creates the sheet show subcommand
func (sc *SheetCommand) newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [sheet-name]",
		Short: "Show configuration sheet details",
		Long: `Show details of a configuration sheet.

You can reference sheets either by name or by project/environment combination.

Examples:
  # Show by sheet name
  ee sheet show my-config

  # Show by environment (uses current project from .ee file)
  ee sheet show --env development

  # Show by project and environment
  ee sheet show --env production --project my-api
`,
		Args: cobra.MaximumNArgs(1),
		RunE: sc.runShow,
	}

	cmd.Flags().StringP("project", "p", "", "Project name (overrides auto-detection from .ee)")
	cmd.Flags().String("env", "", "Environment name")

	return cmd
}

// newListCommand creates the sheet list subcommand
func (sc *SheetCommand) newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configuration sheets",
		RunE:  sc.runList,
	}

	cmd.Flags().String("project", "", "Filter by project name")
	cmd.Flags().Bool("standalone", false, "Show only standalone sheets")

	return cmd
}

// newExportCommand creates the sheet export subcommand
func (sc *SheetCommand) newExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [sheet-name]",
		Short: "Export configuration sheet in different formats",
		Long: `Export a configuration sheet in various formats for different use cases.

You can reference sheets either by name or by project/environment combination.

Supported formats:
  - dotenv: .env file format (KEY=value)
  - bash: Bash export commands script
  - json: JSON object format
  - yaml: YAML format

Examples:
  # Export by sheet name
  ee sheet export my-config

  # Export by environment (uses current project from .ee file)
  ee sheet export --env development

  # Export by project and environment
  ee sheet export --env production --project my-api

  # Export as bash script
  ee sheet export --env development --format bash

  # Export to JSON file
  ee sheet export my-config --format json --output config.json
`,
		Args: cobra.MaximumNArgs(1),
		RunE: sc.runExport,
	}

	cmd.Flags().StringP("format", "f", "dotenv", "Export format (dotenv, bash, json, yaml)")
	cmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	cmd.Flags().Bool("no-comments", false, "Exclude comments in output")
	cmd.Flags().StringP("project", "p", "", "Project name (overrides auto-detection from .ee)")
	cmd.Flags().String("env", "", "Environment name")

	return cmd
}

// newDeleteCommand creates the sheet delete subcommand
func (sc *SheetCommand) newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [sheet-name]",
		Short: "Delete a configuration sheet",
		Long: `Delete a configuration sheet.

You can reference sheets either by name or by project/environment combination.

Examples:
  # Delete by sheet name
  ee sheet delete my-config

  # Delete by environment (uses current project from .ee file)
  ee sheet delete --env development

  # Delete by project and environment
  ee sheet delete --env staging --project my-api
`,
		Args: cobra.MaximumNArgs(1),
		RunE: sc.runDelete,
	}

	cmd.Flags().StringP("project", "p", "", "Project name (overrides auto-detection from .ee)")
	cmd.Flags().String("env", "", "Environment name")

	return cmd
}

// runCreate handles the sheet create command
func (sc *SheetCommand) runCreate(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// Get flags
	envFlag, _ := cmd.Flags().GetString("env")
	schemaFlag, _ := cmd.Flags().GetString("schema")
	importEnvFlag, _ := cmd.Flags().GetString("import-env")
	importJSONFlag, _ := cmd.Flags().GetString("import-json")
	standalone, _ := cmd.Flags().GetBool("standalone")
	interactive, _ := cmd.Flags().GetBool("interactive")

	var sheetName string
	var projectAssociated bool
	var projectID string

	// Determine sheet name and project association
	if envFlag != "" {
		// Creating sheet for project environment
		if !EasyEnvFileExists("") {
			return fmt.Errorf(".ee file not found. Run 'ee init' first or use --standalone flag")
		}

		menvFile, err := LoadEasyEnvFile("")
		if err != nil {
			return fmt.Errorf("failed to load .ee file: %w", err)
		}

		if menvFile.Project == "" {
			return fmt.Errorf("no project ID found in .ee file")
		}

		project, err := uuidStorage.LoadProject(menvFile.Project)
		if err != nil {
			return fmt.Errorf("failed to load project: %w", err)
		}

		// Check if environment already exists
		if _, exists := project.Environments[envFlag]; exists {
			return fmt.Errorf(
				"environment '%s' already exists for project '%s'",
				envFlag,
				project.Name,
			)
		}

		sheetName = fmt.Sprintf("%s-%s", project.Name, envFlag)
		projectAssociated = true
		projectID = project.ID

		fmt.Printf(
			"Creating config sheet for project '%s' environment '%s'\n",
			project.Name,
			envFlag,
		)
	} else if len(args) > 0 {
		// Using provided sheet name
		sheetName = args[0]
		projectAssociated = !standalone

		if projectAssociated && EasyEnvFileExists("") {
			menvFile, err := LoadEasyEnvFile("")
			if err == nil && menvFile.Project != "" {
				projectID = menvFile.Project
			}
		}
	} else {
		return fmt.Errorf("sheet name or --env flag required")
	}

	// Check if sheet already exists
	if uuidStorage.EntityExists("sheets", sheetName) {
		return fmt.Errorf("config sheet '%s' already exists", sheetName)
	}

	// Determine schema to use
	var schemaRef schema.SchemaReference
	if schemaFlag != "" {
		// Use specified schema
		if !uuidStorage.EntityExists("schemas", schemaFlag) {
			return fmt.Errorf("schema '%s' not found", schemaFlag)
		}
		schemaObj, err := uuidStorage.LoadSchema(schemaFlag)
		if err != nil {
			return fmt.Errorf("failed to load schema '%s': %w", schemaFlag, err)
		}
		schemaRef = schema.SchemaReference{Ref: "#/schemas/" + schemaObj.ID}
	} else if projectAssociated && projectID != "" {
		// Use project's schema
		project, err := uuidStorage.LoadProject(projectID)
		if err != nil {
			return fmt.Errorf("failed to load project: %w", err)
		}
		schemaRef = schema.SchemaReference{Ref: "#/schemas/" + project.Schema}
	} else {
		// Create with inline schema (will be populated during creation)
		schemaRef = schema.SchemaReference{Variables: make(map[string]schema.Variable)}
	}

	// Initialize values map
	var values map[string]string

	// Import values from file if specified
	if importEnvFlag != "" {
		var err error
		values, err = sc.importFromEnvFile(importEnvFlag)
		if err != nil {
			return fmt.Errorf("failed to import from .env file: %w", err)
		}
		fmt.Printf("Imported %d values from %s\n", len(values), importEnvFlag)
	} else if importJSONFlag != "" {
		var err error
		values, err = sc.importFromJSONFile(importJSONFlag)
		if err != nil {
			return fmt.Errorf("failed to import from JSON file: %w", err)
		}
		fmt.Printf("Imported %d values from %s\n", len(values), importJSONFlag)
	} else {
		values = make(map[string]string)
	}

	// Interactive mode for additional/missing values
	if interactive || (len(values) == 0 && (importEnvFlag == "" && importJSONFlag == "")) {
		var err error
		values, err = sc.collectValuesInteractively(schemaRef, values, uuidStorage)
		if err != nil {
			return fmt.Errorf("interactive collection failed: %w", err)
		}
	}

	// Create config sheet
	var configSheet *schema.ConfigSheet
	if projectAssociated {
		configSheet = schema.NewConfigSheetForProject(
			sheetName,
			fmt.Sprintf("Config sheet for %s", sheetName),
			schemaRef,
			projectID,
			envFlag, // environment name (empty for non-env sheets)
			values,
		)
	} else {
		configSheet = schema.NewConfigSheet(
			sheetName,
			fmt.Sprintf("Standalone config sheet: %s", sheetName),
			schemaRef,
			values,
		)
	}

	// Save config sheet
	if err := uuidStorage.SaveConfigSheet(configSheet); err != nil {
		return fmt.Errorf("failed to save config sheet: %w", err)
	}

	fmt.Printf("✅ Created config sheet: %s (%s)\n", configSheet.Name, configSheet.ID)

	// Update project if this is an environment sheet
	if projectAssociated && envFlag != "" && projectID != "" {
		project, err := uuidStorage.LoadProject(projectID)
		if err != nil {
			return fmt.Errorf("failed to load project: %w", err)
		}

		project.AddEnvironment(envFlag)
		if err := uuidStorage.SaveProject(project); err != nil {
			return fmt.Errorf("failed to update project: %w", err)
		}

		fmt.Printf("✅ Added environment '%s' to project '%s'\n", envFlag, project.Name)
	}

	return nil
}

// importFromEnvFile imports key-value pairs from a .env file
func (sc *SheetCommand) importFromEnvFile(filename string) (map[string]string, error) {
	if !filepath.IsAbs(filename) {
		// Make relative paths relative to current directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		filename = filepath.Join(cwd, filename)
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file: %v\n", err)
		}
	}()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			fmt.Printf("Warning: skipping invalid line %d: %s\n", lineNum, line)
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}

		values[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return values, nil
}

// importFromJSONFile imports key-value pairs from a JSON file
func (sc *SheetCommand) importFromJSONFile(filename string) (map[string]string, error) {
	if !filepath.IsAbs(filename) {
		// Make relative paths relative to current directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		filename = filepath.Join(cwd, filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Try to parse as map[string]string first
	var stringMap map[string]string
	if err := json.Unmarshal(data, &stringMap); err == nil {
		return stringMap, nil
	}

	// If that fails, try map[string]interface{} and convert
	var interfaceMap map[string]interface{}
	if err := json.Unmarshal(data, &interfaceMap); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	values := make(map[string]string)
	for key, value := range interfaceMap {
		// Convert all values to strings
		values[key] = fmt.Sprintf("%v", value)
	}

	return values, nil
}

// collectValuesInteractively collects values from user input
func (sc *SheetCommand) collectValuesInteractively(
	schemaRef schema.SchemaReference,
	existing map[string]string,
	uuidStorage *storage.UUIDStorage,
) (map[string]string, error) {
	values := make(map[string]string)

	// Copy existing values
	for k, v := range existing {
		values[k] = v
	}

	fmt.Println("\n📝 Interactive configuration sheet creation")
	fmt.Println("Enter values for environment variables. Press Enter with empty value to skip.")
	fmt.Println()

	// If we have a schema reference, collect values for schema variables
	if schemaRef.Ref != "" {
		// Load referenced schema
		schemaID := strings.TrimPrefix(schemaRef.Ref, "#/schemas/")
		schemaObj, err := uuidStorage.LoadSchema(schemaID)
		if err != nil {
			return nil, fmt.Errorf("failed to load schema: %w", err)
		}

		for _, variable := range schemaObj.Variables {
			if existingValue, exists := values[variable.Name]; exists {
				fmt.Printf("%s (imported: %s): ", variable.Name, existingValue)
			} else {
				prompt := variable.Name
				if variable.Default != "" {
					prompt += fmt.Sprintf(" [default: %s]", variable.Default)
				}
				if variable.Required {
					prompt += " (required)"
				}
				fmt.Printf("%s: ", prompt)
			}

			input, err := sc.reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("failed to read input: %w", err)
			}

			input = strings.TrimSpace(input)
			if input != "" {
				values[variable.Name] = input
			} else if variable.Default != "" && values[variable.Name] == "" {
				values[variable.Name] = variable.Default
			}
		}
	}

	// Allow adding custom variables
	fmt.Println("\nAdd custom variables (enter empty name to finish):")
	for {
		fmt.Print("Variable name: ")
		name, err := sc.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read variable name: %w", err)
		}

		name = strings.TrimSpace(name)
		if name == "" {
			break
		}

		fmt.Printf("Value for %s: ", name)
		value, err := sc.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read variable value: %w", err)
		}

		value = strings.TrimSpace(value)
		values[name] = value
	}

	return values, nil
}

// runShow handles the sheet show command
func (sc *SheetCommand) runShow(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// Get arguments and flags
	var sheetName string
	if len(args) > 0 {
		sheetName = args[0]
	}

	projectFlag, _ := cmd.Flags().GetString("project")
	envFlag, _ := cmd.Flags().GetString("env")

	// Resolve config sheet reference
	configSheet, err := sc.resolveSheetReference(uuidStorage, sheetName, projectFlag, envFlag)
	if err != nil {
		return err
	}

	fmt.Printf("Config Sheet: %s\n", configSheet.Name)
	fmt.Printf("ID: %s\n", configSheet.ID)
	fmt.Printf("Description: %s\n", configSheet.Description)

	if configSheet.Project != "" {
		// Load project to get name
		if summary, err := uuidStorage.GetEntitySummary("projects", configSheet.Project); err == nil {
			fmt.Printf("Project: %s\n", summary.Name)
		}
	}

	if configSheet.Environment != "" {
		fmt.Printf("Environment: %s\n", configSheet.Environment)
	}

	// Show schema info
	if configSheet.Schema.Ref != "" {
		schemaID := strings.TrimPrefix(configSheet.Schema.Ref, "#/schemas/")
		if summary, err := uuidStorage.GetEntitySummary("schemas", schemaID); err == nil {
			fmt.Printf("Schema: %s\n", summary.Name)
		}
	} else {
		fmt.Println("Schema: inline")
	}

	fmt.Printf("Variables (%d):\n", len(configSheet.Values))
	fmt.Println("─────────────")

	for key, value := range configSheet.Values {
		// Mask sensitive values
		displayValue := value
		if sc.isSensitiveKey(key) {
			displayValue = "***masked***"
		}
		fmt.Printf("  %s = %s\n", key, displayValue)
	}

	return nil
}

// runList handles the sheet list command
func (sc *SheetCommand) runList(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	projectFilter, _ := cmd.Flags().GetString("project")
	standaloneOnly, _ := cmd.Flags().GetBool("standalone")

	// Build filter
	var filter *schema.ConfigSheetFilter
	if projectFilter != "" || standaloneOnly {
		filter = &schema.ConfigSheetFilter{
			StandaloneOnly: standaloneOnly,
		}
		if projectFilter != "" {
			// Resolve project name to UUID
			project, err := uuidStorage.LoadProject(projectFilter)
			if err != nil {
				return fmt.Errorf("failed to load project '%s': %w", projectFilter, err)
			}
			filter.ProjectGUID = project.ID
		}
	}

	sheets, err := uuidStorage.ListConfigSheets(filter)
	if err != nil {
		return fmt.Errorf("failed to list config sheets: %w", err)
	}

	fmt.Println("Configuration Sheets:")
	fmt.Println("────────────────────")

	if len(sheets) == 0 {
		fmt.Println("No configuration sheets found")
		return nil
	}

	for _, sheet := range sheets {
		status := "standalone"
		if sheet.ProjectGUID != "" {
			if summary, err := uuidStorage.GetEntitySummary("projects", sheet.ProjectGUID); err == nil {
				status = fmt.Sprintf("project: %s", summary.Name)
			}
		}
		fmt.Printf("• %s (%s)\n", sheet.Name, status)
	}

	return nil
}

// resolveSheetReference resolves a sheet reference from either name or project/environment
func (sc *SheetCommand) resolveSheetReference(
	uuidStorage *storage.UUIDStorage,
	sheetName, projectFlag, envFlag string,
) (*schema.ConfigSheet, error) {
	// If environment flag is provided, resolve by project/environment
	if envFlag != "" {
		var project *schema.Project
		var err error

		if projectFlag != "" {
			// Use specified project
			project, err = uuidStorage.LoadProject(projectFlag)
			if err != nil {
				return nil, fmt.Errorf("failed to load project '%s': %w", projectFlag, err)
			}
		} else {
			// Try to detect project from .ee file
			if !EasyEnvFileExists("") {
				return nil, fmt.Errorf(
					"no .ee file found and no --project specified. " +
						"Either specify --project or run from project directory")
			}

			menvFile, err := LoadEasyEnvFile("")
			if err != nil {
				return nil, fmt.Errorf("failed to load .ee file: %w", err)
			}

			if menvFile.Project == "" {
				return nil, fmt.Errorf(".ee file found but no project ID specified")
			}

			project, err = uuidStorage.LoadProject(menvFile.Project)
			if err != nil {
				return nil, fmt.Errorf("failed to load project from .ee file: %w", err)
			}
		}

		// Check if environment exists in project
		envInfo, exists := project.Environments[envFlag]
		if !exists {
			available := make([]string, 0, len(project.Environments))
			for env := range project.Environments {
				available = append(available, env)
			}
			if len(available) > 0 {
				return nil, fmt.Errorf("environment '%s' not found in project '%s'. Available: %s",
					envFlag, project.Name, strings.Join(available, ", "))
			} else {
				return nil, fmt.Errorf("environment '%s' not found in project '%s'. No environments configured",
					envFlag, project.Name)
			}
		}

		// Load the config sheet using naming convention
		configSheetName := project.GetConfigSheetName(envInfo.Name)
		configSheet, err := uuidStorage.LoadConfigSheet(configSheetName)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to load config sheet '%s' for environment '%s': %w",
				configSheetName,
				envFlag,
				err,
			)
		}

		return configSheet, nil
	}

	// If sheet name is provided, load directly
	if sheetName != "" {
		configSheet, err := uuidStorage.LoadConfigSheet(sheetName)
		if err != nil {
			return nil, fmt.Errorf("failed to load config sheet '%s': %w", sheetName, err)
		}
		return configSheet, nil
	}

	return nil, fmt.Errorf("either sheet name or --env flag must be provided")
}

// runExport handles the sheet export command
func (sc *SheetCommand) runExport(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// Get arguments and flags
	var sheetName string
	if len(args) > 0 {
		sheetName = args[0]
	}

	format, _ := cmd.Flags().GetString("format")
	output, _ := cmd.Flags().GetString("output")
	noComments, _ := cmd.Flags().GetBool("no-comments")
	projectFlag, _ := cmd.Flags().GetString("project")
	envFlag, _ := cmd.Flags().GetString("env")

	// Validate format
	validFormats := map[string]bool{
		"dotenv": true,
		"bash":   true,
		"json":   true,
		"yaml":   true,
	}
	if !validFormats[format] {
		return fmt.Errorf(
			"invalid format '%s'. Supported formats: dotenv, bash, json, yaml",
			format,
		)
	}

	// Resolve config sheet reference
	configSheet, err := sc.resolveSheetReference(uuidStorage, sheetName, projectFlag, envFlag)
	if err != nil {
		return err
	}

	// Generate export content
	content, err := sc.generateExport(configSheet, format, !noComments)
	if err != nil {
		return fmt.Errorf("failed to generate export: %w", err)
	}

	// Output to file or stdout
	if output != "" {
		if err := os.WriteFile(output, []byte(content), 0o644); err != nil {
			return fmt.Errorf("failed to write to file '%s': %w", output, err)
		}
		fmt.Printf("✅ Exported %s to %s (%s format)\n", sheetName, output, format)
	} else {
		fmt.Print(content)
	}

	return nil
}

// generateExport generates the export content in the specified format
func (sc *SheetCommand) generateExport(
	configSheet *schema.ConfigSheet,
	format string,
	includeComments bool,
) (string, error) {
	switch format {
	case "dotenv":
		return sc.generateDotEnv(configSheet, includeComments), nil
	case "bash":
		return sc.generateBashScript(configSheet, includeComments), nil
	case "json":
		return sc.generateJSON(configSheet, includeComments)
	case "yaml":
		return sc.generateYAML(configSheet, includeComments)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// generateDotEnv generates .env format output
func (sc *SheetCommand) generateDotEnv(
	configSheet *schema.ConfigSheet,
	includeComments bool,
) string {
	var result strings.Builder

	if includeComments {
		result.WriteString("# Configuration exported from ee\n")
		result.WriteString(fmt.Sprintf("# Sheet: %s\n", configSheet.Name))
		if configSheet.Description != "" {
			result.WriteString(fmt.Sprintf("# Description: %s\n", configSheet.Description))
		}
		result.WriteString(
			fmt.Sprintf("# Generated: %s\n\n", configSheet.UpdatedAt.Format("2006-01-02 15:04:05")),
		)
	}

	for key, value := range configSheet.Values {
		// Escape value if it contains spaces or special characters
		if strings.ContainsAny(value, " \t\n\"'$") {
			value = fmt.Sprintf("\"%s\"", strings.ReplaceAll(value, "\"", "\\\""))
		}
		result.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}

	return result.String()
}

// generateBashScript generates bash export script
func (sc *SheetCommand) generateBashScript(
	configSheet *schema.ConfigSheet,
	includeComments bool,
) string {
	var result strings.Builder

	result.WriteString("#!/bin/bash\n")
	if includeComments {
		result.WriteString("# Configuration export script generated by ee\n")
		result.WriteString(fmt.Sprintf("# Sheet: %s\n", configSheet.Name))
		if configSheet.Description != "" {
			result.WriteString(fmt.Sprintf("# Description: %s\n", configSheet.Description))
		}
		result.WriteString(
			fmt.Sprintf("# Generated: %s\n\n", configSheet.UpdatedAt.Format("2006-01-02 15:04:05")),
		)
		result.WriteString("# Run with: source <(ee sheet export sheet-name --format bash)\n")
		result.WriteString(
			"# Or save to file: ee sheet export sheet-name --format bash --output export.sh\n\n",
		)
	}

	for key, value := range configSheet.Values {
		// Properly escape value for shell
		escapedValue := strings.ReplaceAll(value, "'", "'\"'\"'")
		result.WriteString(fmt.Sprintf("export %s='%s'\n", key, escapedValue))
	}

	if includeComments {
		result.WriteString("\necho \"Environment variables exported successfully\"\n")
	}

	return result.String()
}

// generateJSON generates JSON format output
func (sc *SheetCommand) generateJSON(
	configSheet *schema.ConfigSheet,
	includeComments bool,
) (string, error) {
	if includeComments {
		// JSON with metadata
		export := map[string]interface{}{
			"_meta": map[string]interface{}{
				"sheet":       configSheet.Name,
				"description": configSheet.Description,
				"generated":   configSheet.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
				"format":      "ee-export-v1",
			},
			"variables": configSheet.Values,
		}
		data, err := json.MarshalIndent(export, "", "  ")
		return string(data), err
	} else {
		// Simple JSON object
		data, err := json.MarshalIndent(configSheet.Values, "", "  ")
		return string(data), err
	}
}

// generateYAML generates YAML format output
func (sc *SheetCommand) generateYAML(
	configSheet *schema.ConfigSheet,
	includeComments bool,
) (string, error) {
	if includeComments {
		// YAML with metadata
		export := map[string]interface{}{
			"_meta": map[string]interface{}{
				"sheet":       configSheet.Name,
				"description": configSheet.Description,
				"generated":   configSheet.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
				"format":      "ee-export-v1",
			},
			"variables": configSheet.Values,
		}
		data, err := yaml.Marshal(export)
		return string(data), err
	} else {
		// Simple YAML
		data, err := yaml.Marshal(configSheet.Values)
		return string(data), err
	}
}

// runDelete handles the sheet delete command
func (sc *SheetCommand) runDelete(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// Get arguments and flags
	var sheetName string
	if len(args) > 0 {
		sheetName = args[0]
	}

	projectFlag, _ := cmd.Flags().GetString("project")
	envFlag, _ := cmd.Flags().GetString("env")

	// Resolve config sheet reference
	configSheet, err := sc.resolveSheetReference(uuidStorage, sheetName, projectFlag, envFlag)
	if err != nil {
		return err
	}

	// If this sheet is associated with a project environment, remove it from the project
	if configSheet.Project != "" && configSheet.Environment != "" {
		project, err := uuidStorage.LoadProject(configSheet.Project)
		if err == nil {
			project.RemoveEnvironment(configSheet.Environment)
			if err := uuidStorage.SaveProject(project); err != nil {
				fmt.Fprintf(
					os.Stderr,
					"Warning: failed to update project after removing environment: %v\n",
					err,
				)
			}
		}
	}

	// Delete the sheet
	if err := uuidStorage.DeleteEntity("sheets", configSheet.Name); err != nil {
		return fmt.Errorf("failed to delete config sheet: %w", err)
	}

	fmt.Printf("✅ Deleted config sheet: %s\n", configSheet.Name)
	return nil
}

// newEditCommand creates the sheet edit subcommand
func (sc *SheetCommand) newEditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit [sheet-name]",
		Short: "Edit a configuration sheet using your preferred editor",
		Long: `Edit a configuration sheet using your preferred editor ($EDITOR).

You can edit by sheet name directly, or by specifying a project and environment.
The sheet will be opened as JSON in your preferred editor for modification.

Examples:
  # Edit sheet by name
  ee sheet edit my-config

  # Edit environment sheet using project from .ee file
  ee sheet edit --env development

  # Edit environment sheet for specific project
  ee sheet edit --env development --project my-project
`,
		Args: cobra.MaximumNArgs(1),
		RunE: sc.runEdit,
	}

	cmd.Flags().String("project", "", "Project name (overrides auto-detection)")
	cmd.Flags().String("env", "", "Environment name")

	return cmd
}

// runEdit handles the sheet edit command
func (sc *SheetCommand) runEdit(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	var sheetName string

	// Get flags
	projectFlag, _ := cmd.Flags().GetString("project")
	envFlag, _ := cmd.Flags().GetString("env")

	// Resolve sheet by name or project/environment
	if envFlag != "" {
		// Edit by project/environment
		projectName := projectFlag

		// If no project specified, try to get from .ee file
		if projectName == "" {
			if !EasyEnvFileExists("") {
				return fmt.Errorf(
					"no project specified and no .ee file found. Use --project flag or run from project directory",
				)
			}

			menvFile, err := LoadEasyEnvFile("")
			if err != nil {
				return fmt.Errorf("failed to load .ee file: %w", err)
			}

			if menvFile.Project == "" {
				return fmt.Errorf("no project ID in .ee file")
			}

			// Load project to get name
			project, err := uuidStorage.LoadProject(menvFile.Project)
			if err != nil {
				return fmt.Errorf("failed to load project: %w", err)
			}
			projectName = project.Name
		}

		// Find the config sheet for this project/environment
		return fmt.Errorf(
			"editing by project/environment not yet implemented for project '%s'. Use sheet name directly",
			projectName,
		)
	} else if len(args) > 0 {
		// Edit by sheet name
		sheetName = args[0]
	} else {
		return fmt.Errorf("either sheet name or --env flag is required")
	}

	// Load the config sheet
	configSheet, err := uuidStorage.LoadConfigSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to load config sheet '%s': %w", sheetName, err)
	}

	// Get editor command
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim" // fallback
	}

	// Convert to JSON for editing
	jsonData, err := json.MarshalIndent(configSheet, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config sheet: %w", err)
	}

	// Create temporary file
	tmpFile, err := sc.createTempFile("sheet", jsonData)
	if err != nil {
		return err
	}
	defer func() {
		if err := os.Remove(tmpFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove temporary file: %v\n", err)
		}
	}()

	fmt.Printf("📝 Editing config sheet '%s' using %s...\n", sheetName, editor)

	// Open editor
	if err := sc.openEditor(editor, tmpFile); err != nil {
		return err
	}

	// Read back the edited content
	editedData, err := os.ReadFile(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to read edited file: %w", err)
	}

	// Parse the edited JSON
	var editedSheet schema.ConfigSheet
	if err := json.Unmarshal(editedData, &editedSheet); err != nil {
		return fmt.Errorf("invalid JSON in edited file: %w", err)
	}

	// Preserve the original ID and timestamps if they weren't changed
	if editedSheet.ID == "" {
		editedSheet.ID = configSheet.ID
	}
	if editedSheet.CreatedAt.IsZero() {
		editedSheet.CreatedAt = configSheet.CreatedAt
	}

	// Validate the edited sheet
	if editedSheet.Name == "" {
		return fmt.Errorf("config sheet name cannot be empty")
	}

	// Save the updated sheet
	if err := uuidStorage.SaveConfigSheet(&editedSheet); err != nil {
		return fmt.Errorf("failed to save config sheet: %w", err)
	}

	fmt.Printf("✅ Config sheet '%s' updated successfully\n", editedSheet.Name)

	// Show what changed
	if configSheet.Name != editedSheet.Name {
		fmt.Printf("  Name: %s → %s\n", configSheet.Name, editedSheet.Name)
	}
	if len(configSheet.Values) != len(editedSheet.Values) {
		fmt.Printf("  Values: %d → %d\n", len(configSheet.Values), len(editedSheet.Values))
	}

	return nil
}

// isSensitiveKey checks if a key likely contains sensitive information
func (sc *SheetCommand) isSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	sensitivePatterns := []string{
		"password", "secret", "key", "token", "credential",
		"api_key", "auth", "private", "cert", "ssl",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(key, pattern) {
			return true
		}
	}
	return false
}

// createTempFile creates a temporary file for editing
func (sc *SheetCommand) createTempFile(prefix string, data []byte) (string, error) {
	tmpDir := os.TempDir()

	// Create temp file
	file, err := os.CreateTemp(tmpDir, fmt.Sprintf("ee-%s-*.json", prefix))
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close temporary file: %v\n", err)
		}
	}()

	// Write data to temp file
	if _, err := file.Write(data); err != nil {
		return "", fmt.Errorf("failed to write to temporary file: %w", err)
	}

	return file.Name(), nil
}

// openEditor opens the specified editor with the given file
func (sc *SheetCommand) openEditor(editor, filename string) error {
	// Split editor command (in case it has arguments)
	editorParts := strings.Fields(editor)
	if len(editorParts) == 0 {
		return fmt.Errorf("editor command is empty")
	}

	// Prepare command
	editorCmd := editorParts[0]
	cmdArgs := make([]string, len(editorParts)-1+1)
	copy(cmdArgs, editorParts[1:])
	cmdArgs[len(editorParts)-1] = filename

	// Execute editor
	cmd := exec.Command(editorCmd, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Opening %s...\n", filename)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor command failed: %w", err)
	}

	return nil
}
