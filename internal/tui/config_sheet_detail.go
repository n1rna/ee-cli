// Package tui provides config sheet detail view functionality
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/n1rna/ee-cli/internal/schema"
)

// ConfigSheetDetailModel represents the config sheet detail view state
type ConfigSheetDetailModel struct {
	configSheet *schema.ConfigSheet
	loading     bool
	error       string
	cursor      int
	maxItems    int
}

// NewConfigSheetDetailModel creates a new config sheet detail model
func NewConfigSheetDetailModel() *ConfigSheetDetailModel {
	return &ConfigSheetDetailModel{}
}

// SetConfigSheet updates the config sheet being displayed
func (m *ConfigSheetDetailModel) SetConfigSheet(configSheet *schema.ConfigSheet) {
	m.configSheet = configSheet
	m.cursor = 0
	m.loading = false
	m.error = ""
	// Calculate max items: values + actions
	if configSheet != nil {
		m.maxItems = len(configSheet.Values) + 1 // +1 for "Back" option
	}
}

// SetLoading sets the loading state
func (m *ConfigSheetDetailModel) SetLoading(loading bool) {
	m.loading = loading
}

// SetError sets an error message
func (m *ConfigSheetDetailModel) SetError(err string) {
	m.error = err
	m.loading = false
}

// Init returns the initial command for the config sheet detail view
func (m ConfigSheetDetailModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the config sheet detail view
func (m ConfigSheetDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < m.maxItems-1 {
				m.cursor++
			}

		case "enter", " ":
			// Handle selection based on cursor position
			if m.configSheet != nil {
				if m.cursor == m.maxItems-1 {
					// "Back" option selected
					return m, func() tea.Msg { return NavigateMsg(ConfigSheetsView) }
				}
				// Variable selected - could add variable editing here in the future
			}

		case "esc":
			// Go back to config sheets list
			return m, func() tea.Msg { return NavigateMsg(ConfigSheetsView) }
		}
	}

	return m, nil
}

// View renders the config sheet detail view
func (m ConfigSheetDetailModel) View() string {
	if m.loading {
		return "\nLoading config sheet details..."
	}

	if m.error != "" {
		return fmt.Sprintf("\nError: %s\n\nPress 'esc' to go back", m.error)
	}

	if m.configSheet == nil {
		return "\nNo config sheet selected.\n\nPress 'esc' to go back"
	}

	var b strings.Builder

	// Header
	b.WriteString(
		configSheetDetailTitleStyle.Render(fmt.Sprintf("Config Sheet: %s", m.configSheet.Name)),
	)
	b.WriteString("\n\n")

	// Basic information
	if m.configSheet.Description != "" {
		b.WriteString(fmt.Sprintf("Description: %s\n", m.configSheet.Description))
	}

	if m.configSheet.Project != "" {
		b.WriteString(fmt.Sprintf("Project: %s\n", m.configSheet.Project))
	}
	if m.configSheet.Environment != "" {
		b.WriteString(fmt.Sprintf("Environment: %s\n", m.configSheet.Environment))
	}

	if len(m.configSheet.Extends) > 0 {
		b.WriteString(fmt.Sprintf("Extends: %s\n", strings.Join(m.configSheet.Extends, ", ")))
	}

	b.WriteString(
		fmt.Sprintf("Created: %s\n", m.configSheet.CreatedAt.Format("2006-01-02 15:04:05")),
	)
	b.WriteString("\n")

	// Variables section
	if len(m.configSheet.Values) == 0 {
		b.WriteString(noVariablesStyle.Render("No variables defined"))
		b.WriteString("\n\n")
	} else {
		b.WriteString(configSheetVariablesSectionStyle.Render("Variables:"))
		b.WriteString("\n\n")

		// Create ordered list of variable keys for consistent navigation
		varKeys := make([]string, 0, len(m.configSheet.Values))
		for key := range m.configSheet.Values {
			varKeys = append(varKeys, key)
		}

		for i, key := range varKeys {
			cursor := "  "
			if m.cursor == i {
				cursor = "→ "
			}

			// Variable name and value
			varInfo := fmt.Sprintf("%s%s = %s", cursor, key, m.configSheet.Values[key])

			// Style based on selection
			if m.cursor == i {
				b.WriteString(selectedVariableStyle.Render(varInfo))
			} else {
				b.WriteString(normalVariableStyle.Render(varInfo))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Back option
	cursor := "  "
	if m.cursor == m.maxItems-1 {
		cursor = "→ "
	}

	backOption := fmt.Sprintf("%sBack to Config Sheets", cursor)
	if m.cursor == m.maxItems-1 {
		b.WriteString(selectedItemStyle.Render(backOption))
	} else {
		b.WriteString(normalItemStyle.Render(backOption))
	}

	b.WriteString("\n\n")
	b.WriteString(helpTextStyle.Render("↑/↓: navigate • enter: select • esc: back"))

	return b.String()
}

// Styles for config sheet detail view
var (
	configSheetDetailTitleStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("33")).
					MarginBottom(1)

	configSheetVariablesSectionStyle = lipgloss.NewStyle().
						Bold(true).
						Foreground(lipgloss.Color("205"))
)
