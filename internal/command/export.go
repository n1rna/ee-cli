// Package command contains CLI command implementations.
package command

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/n1rna/menv/internal/util"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ExportCommand struct {
}

func NewExportCommand() *cobra.Command {
	ec := &ExportCommand{}

	cmd := &cobra.Command{
		Use:   "export [sheet-name]",
		Short: "Export environment variables in different formats",
		Args:  cobra.MaximumNArgs(1),
		RunE:  ec.Run,
	}

	cmd.Flags().StringP("project", "p", "", "Project name")
	cmd.Flags().StringP("env", "e", "", "Environment name")
	cmd.Flags().StringP("format", "f", "env", "Output format (env, json, or yaml)")
	cmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")
	if err := cmd.MarkFlagRequired("env"); err != nil {
		return nil
	}

	return cmd
}

func (c *ExportCommand) Run(cmd *cobra.Command, args []string) error {
	storage := GetStorage(cmd.Context())
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// Get sheet name from args or empty string if not provided
	sheetName := ""
	if len(args) > 0 {
		sheetName = args[0]
	}

	// Get project and env flags
	projectFlag, _ := cmd.Flags().GetString("project")
	envFlag, _ := cmd.Flags().GetString("env")

	// Parse sheet reference
	ref, err := util.ParseSheetReference(sheetName, projectFlag, envFlag)
	if err != nil {
		return err
	}

	// Validate sheet reference
	if err := util.ValidateSheetReference(ref, storage); err != nil {
		return err
	}

	// Load config sheet
	configSheet, err := storage.LoadConfigSheet(ref.Project, ref.Env)
	if err != nil {
		return fmt.Errorf("failed to load config sheet: %w", err)
	}

	format, _ := cmd.Flags().GetString("format")
	output, _ := cmd.Flags().GetString("output")

	// Generate output in specified format
	var content []byte
	switch strings.ToLower(format) {
	case "env":
		content = c.formatAsEnv(configSheet.Values)
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

func (c *ExportCommand) formatAsEnv(values map[string]string) []byte {
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

	return []byte(builder.String())
}

func (c *ExportCommand) formatAsJSON(values map[string]string) ([]byte, error) {
	return json.MarshalIndent(values, "", "  ")
}

func (c *ExportCommand) formatAsYAML(values map[string]string) ([]byte, error) {
	return yaml.Marshal(values)
}
