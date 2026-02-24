// Package command implements the ee hydrate command for generating env files from schema + shell env.
package command

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/n1rna/ee-cli/internal/config"
	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/output"
	"github.com/n1rna/ee-cli/internal/util"
)

// HydrateCommand handles the ee hydrate command
type HydrateCommand struct{}

// NewHydrateCommand creates a new ee hydrate command
func NewHydrateCommand(groupId string) *cobra.Command {
	hc := &HydrateCommand{}

	cmd := &cobra.Command{
		Use:   "hydrate <environment>",
		Short: "Generate an env file by hydrating schema variables from the current shell",
		Long: `Generate an environment file by resolving schema variables against the current shell environment.

For each variable defined in the project schema:
  1. If the variable exists in the current shell environment, use that value
  2. Otherwise, fall back to the default value from the schema
  3. If neither exists, the variable is left empty (a warning is shown for required variables)

The output format can be dotenv (default), json, or yaml.

Examples:
  # Generate .env for the dev environment (prints to stdout)
  ee hydrate dev

  # Write to a file
  ee hydrate dev -o .env

  # Output as JSON
  ee hydrate dev -f json

  # Output as YAML to a file
  ee hydrate dev -f yaml -o config.yaml`,
		Args:    cobra.ExactArgs(1),
		RunE:    hc.Run,
		GroupID: groupId,
	}

	cmd.Flags().StringP("output", "o", "", "Write output to file instead of stdout")
	cmd.Flags().StringP("format", "f", "dotenv", "Output format: dotenv, json, yaml")

	return cmd
}

// Run executes the hydrate command
func (c *HydrateCommand) Run(cmd *cobra.Command, args []string) error {
	// Require project context
	context, err := RequireProjectContext(cmd.Context())
	if err != nil {
		return fmt.Errorf(
			"hydrate command requires a project context (%s file): %w",
			config.ProjectConfigFileName,
			err,
		)
	}

	envName := args[0]
	format, _ := cmd.Flags().GetString("format")
	outputFile, _ := cmd.Flags().GetString("output")

	// Validate the environment exists
	if !context.HasEnvironment(envName) {
		available := context.GetEnvironmentNames()
		return fmt.Errorf("environment '%s' not found in project (available: %s)",
			envName, strings.Join(available, ", "))
	}

	// Load schema variables
	schemaVariables, err := c.loadSchema(context)
	if err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	if len(schemaVariables) == 0 {
		return fmt.Errorf("no variables defined in project schema")
	}

	// Hydrate values from shell environment
	printer := output.NewPrinter(output.FormatTable, false)
	values := c.hydrateValues(schemaVariables, printer)

	// Render output
	rendered, err := c.render(values, format)
	if err != nil {
		return err
	}

	// Write to file or stdout
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(rendered), 0o644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		printer.Success(fmt.Sprintf("Wrote %d variables to %s", len(values), outputFile))
	} else {
		fmt.Print(rendered)
	}

	return nil
}

// loadSchema loads the project schema variables (inline or referenced)
func (c *HydrateCommand) loadSchema(context *util.CommandContext) (map[string]entities.Variable, error) {
	schema := context.ProjectConfig.Schema

	// Inline schema
	if schema.Variables != nil {
		return schema.Variables, nil
	}

	// Referenced schema
	if schema.Ref != "" {
		if context.Manager == nil {
			return nil, fmt.Errorf("entity manager not available to load referenced schema")
		}

		loaded, err := context.Manager.Schemas.GetByReference(schema.Ref)
		if err != nil {
			return nil, fmt.Errorf("failed to load schema '%s': %w", schema.Ref, err)
		}

		variables := make(map[string]entities.Variable, len(loaded.Variables))
		for _, v := range loaded.Variables {
			variables[v.Name] = v
		}
		return variables, nil
	}

	return nil, fmt.Errorf("no schema defined in project config")
}

// hydrateValues resolves each schema variable from the shell environment or its default
func (c *HydrateCommand) hydrateValues(
	schemaVariables map[string]entities.Variable,
	printer *output.Printer,
) map[string]string {
	values := make(map[string]string, len(schemaVariables))

	for name, variable := range schemaVariables {
		if envVal, ok := os.LookupEnv(name); ok {
			values[name] = envVal
		} else if variable.Default != "" {
			values[name] = variable.Default
		} else {
			values[name] = ""
			if variable.Required {
				printer.Warning(fmt.Sprintf("Required variable %s has no value in shell and no default", name))
			}
		}
	}

	return values
}

// render produces the output string in the requested format
func (c *HydrateCommand) render(values map[string]string, format string) (string, error) {
	switch format {
	case "dotenv", "env":
		return c.renderDotenv(values), nil
	case "json":
		return c.renderJSON(values)
	case "yaml", "yml":
		return c.renderYAML(values)
	default:
		return "", fmt.Errorf("unsupported format '%s' (supported: dotenv, json, yaml)", format)
	}
}

func (c *HydrateCommand) renderDotenv(values map[string]string) string {
	keys := sortedKeys(values)
	var sb strings.Builder
	for _, key := range keys {
		value := values[key]
		value = strings.ReplaceAll(value, "\\", "\\\\")
		value = strings.ReplaceAll(value, "\"", "\\\"")
		value = strings.ReplaceAll(value, "\n", "\\n")
		sb.WriteString(fmt.Sprintf("%s=\"%s\"\n", key, value))
	}
	return sb.String()
}

func (c *HydrateCommand) renderJSON(values map[string]string) (string, error) {
	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data) + "\n", nil
}

func (c *HydrateCommand) renderYAML(values map[string]string) (string, error) {
	data, err := yaml.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return string(data), nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
