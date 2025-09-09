// Package command provides UI command functionality
package command

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/tui"
)

// NewUICommand creates the UI command
func NewUICommand(groupId string) *cobra.Command {
	return &cobra.Command{
		Use:     "ui",
		Short:   "Launch interactive terminal interface",
		Long:    "Launch the ee terminal user interface for managing schemas and projects through the API.",
		RunE:    runUI,
		GroupID: groupId,
	}
}

func runUI(cmd *cobra.Command, args []string) error {
	// Get storage from context
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not available")
	}

	// Create TUI model with storage
	model := tui.NewModel(uuidStorage)

	// Create and run the Bubble Tea program
	program := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
