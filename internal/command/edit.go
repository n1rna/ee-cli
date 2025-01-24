// internal/command/edit.go
package command

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/n1rna/menv/internal/schema"
	"github.com/spf13/cobra"
)

type EditCommand struct {
}

func NewEditCommand() *cobra.Command {
	ec := &EditCommand{}

	cmd := &cobra.Command{
		Use:   "edit [project-name]",
		Short: "Edit environment variables for a project",
		Args:  cobra.ExactArgs(1),
		RunE:  ec.Run,
	}

	cmd.Flags().String("env", "", "Environment to edit (required)")
	cmd.MarkFlagRequired("env")

	return cmd
}

func (c *EditCommand) Run(cmd *cobra.Command, args []string) error {
	projectName := args[0]
	envName, _ := cmd.Flags().GetString("env")

	return c.editEnvironment(cmd.Context(), projectName, envName)
}

func (c *EditCommand) editEnvironment(ctx context.Context, projectName, envName string) error {
	storage := GetStorage(ctx)
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}
	// Load config sheet
	configSheet, err := storage.LoadConfigSheet(projectName, envName)
	if err != nil {
		return fmt.Errorf("failed to load config sheet: %w", err)
	}

	// Load schema
	schemaObj, err := storage.LoadSchema(configSheet.Schema)
	if err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	validator := schema.NewValidator()

	for {
		// Create temporary file
		tmpFile, err := c.createTempEnvFile(configSheet, schemaObj)
		if err != nil {
			return fmt.Errorf("failed to create temporary file: %w", err)
		}

		// Open editor
		if err := c.openEditor(tmpFile); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("failed to open editor: %w", err)
		}

		// Parse edited file
		newValues, err := c.parseEnvFile(tmpFile)
		if err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("failed to parse edited file: %w", err)
		}

		// Update config sheet with new values
		configSheet.Values = newValues

		// Validate the updated config
		err = validator.ValidateConfigSheet(configSheet, schemaObj)
		if err != nil {
			// Add error message to the file
			if err := c.appendErrorToFile(tmpFile, err.Error()); err != nil {
				os.Remove(tmpFile)
				return fmt.Errorf("failed to append error message: %w", err)
			}
			continue // Reopen editor
		}

		// Save the valid config
		if err := storage.SaveConfigSheet(configSheet); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("failed to save config sheet: %w", err)
		}

		// Clean up
		os.Remove(tmpFile)
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
	defer tmpFile.Close()

	// Write header comment
	fmt.Fprintf(tmpFile, "# Environment variables for %s (%s)\n", configSheet.ProjectName, configSheet.EnvName)
	fmt.Fprintf(tmpFile, "# Schema: %s\n\n", configSheet.Schema)

	// Write variables in .env format
	for _, v := range schemaObj.Variables {
		value := configSheet.Values[v.Name]
		if value == "" {
			value = v.Default
		}

		// Add comment for variable type and constraints
		if v.Required {
			fmt.Fprintf(tmpFile, "# Required - Type: %s\n", v.Type)
		} else {
			fmt.Fprintf(tmpFile, "# Optional - Type: %s\n", v.Type)
		}
		if v.Regex != "" {
			fmt.Fprintf(tmpFile, "# Pattern: %s\n", v.Regex)
		}

		fmt.Fprintf(tmpFile, "%s=%s\n\n", v.Name, value)
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
	defer file.Close()

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
