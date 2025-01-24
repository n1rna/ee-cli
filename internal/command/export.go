// internal/command/export.go
package command

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ExportCommand struct {
}

func NewExportCommand() *cobra.Command {
	ec := &ExportCommand{}

	cmd := &cobra.Command{
		Use:   "export [project-name]",
		Short: "Export environment variables in different formats",
		Args:  cobra.ExactArgs(1),
		RunE:  ec.Run,
	}

	cmd.Flags().String("env", "", "Environment to export (required)")
	cmd.Flags().StringP("format", "f", "env", "Output format (env, json, or yaml)")
	cmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")
	cmd.MarkFlagRequired("env")

	return cmd
}

func (c *ExportCommand) Run(cmd *cobra.Command, args []string) error {
	storage := GetStorage(cmd.Context())
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	projectName := args[0]
	envName, _ := cmd.Flags().GetString("env")
	format, _ := cmd.Flags().GetString("format")
	output, _ := cmd.Flags().GetString("output")

	// Load config sheet
	configSheet, err := storage.LoadConfigSheet(projectName, envName)
	if err != nil {
		return fmt.Errorf("failed to load config sheet: %w", err)
	}

	// Generate output in specified format
	var content []byte
	switch strings.ToLower(format) {
	case "env":
		content, err = c.formatAsEnv(configSheet.Values)
	case "json":
		content, err = c.formatAsJSON(configSheet.Values)
	case "yaml":
		content, err = c.formatAsYAML(configSheet.Values)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	// Write to output
	if output == "" {
		// Write to stdout
		fmt.Print(string(content))
		return nil
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(output, content, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

func (c *ExportCommand) formatAsEnv(values map[string]string) ([]byte, error) {
	var builder strings.Builder

	// Sort keys for consistent output
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := values[key]
		// Check if value needs quoting
		if strings.Contains(value, " ") || strings.Contains(value, "#") {
			value = fmt.Sprintf(`"%s"`, value)
		}
		builder.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}

	return []byte(builder.String()), nil
}

func (c *ExportCommand) formatAsJSON(values map[string]string) ([]byte, error) {
	return json.MarshalIndent(values, "", "  ")
}

func (c *ExportCommand) formatAsYAML(values map[string]string) ([]byte, error) {
	return yaml.Marshal(values)
}
