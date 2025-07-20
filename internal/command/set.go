// Package command contains CLI command implementations.
package command

import (
	"fmt"
	"strings"

	"github.com/n1rna/menv/internal/schema"
	"github.com/n1rna/menv/internal/util"
	"github.com/spf13/cobra"
)

type SetCommand struct {
}

func NewSetCommand() *cobra.Command {
	sc := &SetCommand{}

	cmd := &cobra.Command{
		Use:   "set [sheet-name] [key=value]...",
		Short: "Set environment variable values",
		Long: `Set one or more environment variable values.
Example: menv set myproject --env dev DB_HOST=localhost DB_PORT=5432`,
		Args: cobra.MinimumNArgs(2),
		RunE: sc.Run,
	}

	cmd.Flags().StringP("project", "p", "", "Project name")
	cmd.Flags().StringP("env", "e", "", "Environment name")

	return cmd
}

func (c *SetCommand) Run(cmd *cobra.Command, args []string) error {
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

	// Parse key-value pairs
	updates := make(map[string]string)
	for _, arg := range args[1:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid key-value pair: %s (use KEY=VALUE format)", arg)
		}
		updates[parts[0]] = parts[1]
	}

	// Load schema
	schemaObj, err := storage.LoadSchema(configSheet.Schema)
	if err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	// Create validator
	validator := schema.NewValidator(storage)

	// Validate each update
	for key, value := range updates {
		// Find variable in schema
		var variable *schema.Variable
		for i := range schemaObj.Variables {
			if schemaObj.Variables[i].Name == key {
				variable = &schemaObj.Variables[i]
				break
			}
		}

		if variable == nil {
			return fmt.Errorf("variable %s not found in schema", key)
		}

		// Validate new value
		if err := validator.ValidateValue(variable, value); err != nil {
			return fmt.Errorf("invalid value for %s: %w", key, err)
		}

		// Update value
		configSheet.Values[key] = value
	}

	// Validate entire config sheet
	if err := validator.ValidateConfigSheet(configSheet, schemaObj); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Save updated config sheet
	if err := storage.SaveConfigSheet(configSheet); err != nil {
		return fmt.Errorf("failed to save config sheet: %w", err)
	}

	fmt.Printf("Successfully updated %d value(s)\n", len(updates))
	return nil
}
