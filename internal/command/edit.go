// Package command contains CLI command implementations.
package command

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/n1rna/menv/internal/schema"
	"github.com/n1rna/menv/internal/util"
	"github.com/spf13/cobra"
)

type EditCommand struct {
}

func NewEditCommand() *cobra.Command {
	ec := &EditCommand{}

	cmd := &cobra.Command{
		Use:   "edit [sheet-name]",
		Short: "Edit environment variables for a project",
		Args:  cobra.MaximumNArgs(1),
		RunE:  ec.Run,
	}

	cmd.Flags().StringP("project", "p", "", "Project name")
	cmd.Flags().StringP("env", "e", "", "Environment name")

	return cmd
}

func (c *EditCommand) Run(cmd *cobra.Command, args []string) error {
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

	return c.editEnvironment(cmd.Context(), configSheet)
}

func (c *EditCommand) editEnvironment(ctx context.Context, configSheet *schema.ConfigSheet) error {
	storage := GetStorage(ctx)
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// Load schema
	schemaObj, err := storage.LoadSchema(configSheet.Schema)
	if err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	validator := schema.NewValidator(storage)

	for {
		// Create temporary file
		tmpFile, err := c.createTempEnvFile(configSheet, schemaObj)
		if err != nil {
			return fmt.Errorf("failed to create temporary file: %w", err)
		}

		// Open editor
		if err := c.openEditor(tmpFile); err != nil {
			if removeErr := os.Remove(tmpFile); removeErr != nil {
				fmt.Printf("warning: failed to remove temp file: %v\n", removeErr)
			}
			return fmt.Errorf("failed to open editor: %w", err)
		}

		// Parse edited file
		newValues, err := c.parseEnvFile(tmpFile)
		if err != nil {
			if removeErr := os.Remove(tmpFile); removeErr != nil {
				fmt.Printf("warning: failed to remove temp file: %v\n", removeErr)
			}
			return fmt.Errorf("failed to parse edited file: %w", err)
		}

		// Update config sheet with new values
		configSheet.Values = newValues

		// Validate the updated config
		err = validator.ValidateConfigSheet(configSheet, schemaObj)
		if err != nil {
			// Add error message to the file
			if err := c.appendErrorToFile(tmpFile, err.Error()); err != nil {
				if removeErr := os.Remove(tmpFile); removeErr != nil {
					fmt.Printf("warning: failed to remove temp file: %v\n", removeErr)
				}
				return fmt.Errorf("failed to append error message: %w", err)
			}
			continue // Reopen editor
		}

		// Save the valid config
		if err := storage.SaveConfigSheet(configSheet); err != nil {
			if removeErr := os.Remove(tmpFile); removeErr != nil {
				fmt.Printf("warning: failed to remove temp file: %v\n", removeErr)
			}
			return fmt.Errorf("failed to save config sheet: %w", err)
		}

		// Clean up
		if err := os.Remove(tmpFile); err != nil {
			fmt.Printf("warning: failed to remove temp file: %v\n", err)
		}
		break
	}

	return nil
}

func (c *EditCommand) createTempEnvFile(configSheet *schema.ConfigSheet, schemaObj *schema.Schema) (string, error) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "menv-*.env")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			fmt.Printf("warning: failed to close temp file: %v\n", err)
		}
	}()

	// Write header comment
	if _, err := fmt.Fprintf(tmpFile, "# Environment variables for %s (%s)\n",
		configSheet.ProjectName, configSheet.EnvName); err != nil {
		return "", fmt.Errorf("failed to write header: %w", err)
	}
	if _, err := fmt.Fprintf(tmpFile, "# Schema: %s\n\n", configSheet.Schema); err != nil {
		return "", fmt.Errorf("failed to write schema: %w", err)
	}

	// Write variables in .env format
	for _, v := range schemaObj.Variables {
		value := configSheet.Values[v.Name]
		if value == "" {
			value = v.Default
		}

		// Add comment for variable type and constraints
		if v.Required {
			if _, err := fmt.Fprintf(tmpFile, "# Required - Type: %s\n", v.Type); err != nil {
				return "", fmt.Errorf("failed to write required comment: %w", err)
			}
		} else {
			if _, err := fmt.Fprintf(tmpFile, "# Optional - Type: %s\n", v.Type); err != nil {
				return "", fmt.Errorf("failed to write optional comment: %w", err)
			}
		}
		if v.Regex != "" {
			if _, err := fmt.Fprintf(tmpFile, "# Pattern: %s\n", v.Regex); err != nil {
				return "", fmt.Errorf("failed to write pattern comment: %w", err)
			}
		}

		if _, err := fmt.Fprintf(tmpFile, "%s=%s\n\n", v.Name, value); err != nil {
			return "", fmt.Errorf("failed to write variable: %w", err)
		}
	}

	return tmpFile.Name(), nil
}

func (c *EditCommand) openEditor(filename string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim" // Default to vim if EDITOR is not set
	}

	cmd := exec.Command(editor, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (c *EditCommand) parseEnvFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("warning: failed to close file: %v\n", err)
		}
	}()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes if present
		value = strings.Trim(value, `"'`)

		values[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return values, nil
}

func (c *EditCommand) appendErrorToFile(filename string, errorMsg string) error {
	// Read existing content
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create new content with error message at top
	newContent := fmt.Sprintf("# ERROR: %s\n# Please fix the error and save the file\n\n%s",
		errorMsg, string(content))

	// Write back to file
	if err := os.WriteFile(filename, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
