// Package command provides shared editor functionality
package command

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// EditorInterface defines the methods needed for editing entities
type EditorInterface interface {
	createTempFile(prefix string, data []byte) (string, error)
	openEditor(editor, tmpFile string) error
}

// EditEntity provides a generic way to edit JSON entities with an external editor
func EditEntity(
	entityName string,
	entity interface{},
	editor EditorInterface,
	validator func([]byte) (interface{}, error),
	saver func(interface{}) error,
	changeReporter func(interface{}, interface{}),
) error {
	// Get editor command
	editorCmd := os.Getenv("EDITOR")
	if editorCmd == "" {
		editorCmd = "vim" // fallback
	}

	// Convert to JSON for editing
	jsonData, err := json.MarshalIndent(entity, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize %s: %w", entityName, err)
	}

	// Create temporary file
	tmpFile, err := editor.createTempFile(strings.ToLower(entityName), jsonData)
	if err != nil {
		return err
	}
	defer func() {
		if err := os.Remove(tmpFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove temporary file: %v\n", err)
		}
	}()

	fmt.Printf("📝 Editing %s using %s...\n", entityName, editorCmd)

	// Open editor
	if err := editor.openEditor(editorCmd, tmpFile); err != nil {
		return err
	}

	// Read back the edited content
	editedData, err := os.ReadFile(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to read edited file: %w", err)
	}

	// Validate and parse the edited JSON
	editedEntity, err := validator(editedData)
	if err != nil {
		return err
	}

	// Save the updated entity
	if err := saver(editedEntity); err != nil {
		return fmt.Errorf("failed to save %s: %w", entityName, err)
	}

	fmt.Printf("✅ %s updated successfully\n", entityName)

	// Show what changed
	if changeReporter != nil {
		changeReporter(entity, editedEntity)
	}

	return nil
}

// BaseEditorCommands provides common editor functionality for commands
type BaseEditorCommands struct{}

// createTempFile creates a temporary file for editing
func (b *BaseEditorCommands) createTempFile(prefix string, data []byte) (string, error) {
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
func (b *BaseEditorCommands) openEditor(editor, tmpFile string) error {
	// Split editor command in case it has arguments
	editorParts := strings.Fields(editor)
	if len(editorParts) == 0 {
		return fmt.Errorf("editor command is empty")
	}

	// Build command
	cmdArgs := make([]string, len(editorParts)-1+1)
	copy(cmdArgs, editorParts[1:])
	cmdArgs[len(editorParts)-1] = tmpFile
	cmd := exec.Command(editorParts[0], cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
