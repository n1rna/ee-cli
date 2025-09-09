// Package tui provides schema management functionality
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/n1rna/ee-cli/internal/schema"
)

// SchemasModel represents the schemas view state
type SchemasModel struct {
	summaries []*schema.EntitySummary
	cursor    int
}

// NewSchemasModel creates a new schemas model
func NewSchemasModel() *SchemasModel {
	return &SchemasModel{
		summaries: []*schema.EntitySummary{},
		cursor:    0,
	}
}

// SetSchemaSummaries updates the schema summaries list
func (m *SchemasModel) SetSchemaSummaries(summaries []*schema.EntitySummary) {
	m.summaries = summaries
	m.cursor = 0
}

// Init returns the initial command for the schemas view
func (m SchemasModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the schemas view
func (m SchemasModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.summaries)-1 {
				m.cursor++
			}

		case "enter", " ":
			if len(m.summaries) > 0 && m.cursor < len(m.summaries) {
				// Navigate to schema detail view by loading the full object
				selectedSchema := m.summaries[m.cursor]
				return m, func() tea.Msg {
					return LoadSchemaMsg{NameOrUUID: selectedSchema.Name}
				}
			}

		case "n":
			// Navigate to create schema view
			return m, func() tea.Msg { return NavigateMsg(CreateSchemaView) }
		}
	}

	return m, nil
}

// View renders the schemas list
func (m SchemasModel) View() string {
	if len(m.summaries) == 0 {
		return noItemsStyle.Render(
			"\nNo schemas found.\n\nPress 'n' to create a new schema\nPress 'esc' to go back",
		)
	}

	s := "\nSchemas:\n\n"

	for i, schema := range m.summaries {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		// Format schema info
		schemaInfo := fmt.Sprintf("%s %s", cursor, schema.Name)
		if schema.Description != "" {
			schemaInfo += fmt.Sprintf(" - %s", schema.Description)
		}

		// Style the current selection
		if m.cursor == i {
			s += selectedItemStyle.Render(schemaInfo)
		} else {
			s += normalItemStyle.Render(schemaInfo)
		}
		s += "\n"
	}

	s += "\n" + helpTextStyle.Render("enter: view details • n: new schema • esc: back")
	return s
}

// Styles for schemas view
var (
	noItemsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)

	helpTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)
