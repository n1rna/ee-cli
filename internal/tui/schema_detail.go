// Package tui provides schema detail view functionality
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/n1rna/ee-cli/internal/schema"
)

// SchemaDetailModel represents the schema detail view state
type SchemaDetailModel struct {
	schema   *schema.Schema
	loading  bool
	error    string
	cursor   int
	maxItems int
}

// NewSchemaDetailModel creates a new schema detail model
func NewSchemaDetailModel() *SchemaDetailModel {
	return &SchemaDetailModel{}
}

// SetSchema updates the schema being displayed
func (m *SchemaDetailModel) SetSchema(schema *schema.Schema) {
	m.schema = schema
	m.cursor = 0
	m.loading = false
	m.error = ""
	// Calculate max items: basic info + variables + actions
	if schema != nil {
		m.maxItems = len(schema.Variables) + 1 // +1 for "Back" option
	}
}

// SetLoading sets the loading state
func (m *SchemaDetailModel) SetLoading(loading bool) {
	m.loading = loading
}

// SetError sets an error message
func (m *SchemaDetailModel) SetError(err string) {
	m.error = err
	m.loading = false
}

// Init returns the initial command for the schema detail view
func (m SchemaDetailModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the schema detail view
func (m SchemaDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.schema != nil {
				if m.cursor == m.maxItems-1 {
					// "Back" option selected
					return m, func() tea.Msg { return NavigateMsg(SchemasView) }
				}
				// Variable selected - could add variable editing here in the future
			}

		case "esc":
			// Go back to schemas list
			return m, func() tea.Msg { return NavigateMsg(SchemasView) }
		}
	}

	return m, nil
}

// View renders the schema detail view
func (m SchemaDetailModel) View() string {
	if m.loading {
		return "\nLoading schema details..."
	}

	if m.error != "" {
		return fmt.Sprintf("\nError: %s\n\nPress 'esc' to go back", m.error)
	}

	if m.schema == nil {
		return "\nNo schema selected.\n\nPress 'esc' to go back"
	}

	var b strings.Builder

	// Header
	b.WriteString(schemaDetailTitleStyle.Render(fmt.Sprintf("Schema: %s", m.schema.Name)))
	b.WriteString("\n\n")

	// Basic information
	if m.schema.Description != "" {
		b.WriteString(fmt.Sprintf("Description: %s\n", m.schema.Description))
	}

	if len(m.schema.Extends) > 0 {
		b.WriteString(fmt.Sprintf("Extends: %s\n", strings.Join(m.schema.Extends, ", ")))
	}

	b.WriteString(fmt.Sprintf("Created: %s\n", m.schema.CreatedAt.Format("2006-01-02 15:04:05")))
	b.WriteString("\n")

	// Variables section
	if len(m.schema.Variables) == 0 {
		b.WriteString(noVariablesStyle.Render("No variables defined"))
		b.WriteString("\n\n")
	} else {
		b.WriteString(variablesSectionStyle.Render("Variables:"))
		b.WriteString("\n\n")

		for i, variable := range m.schema.Variables {
			cursor := "  "
			if m.cursor == i {
				cursor = "→ "
			}

			// Variable name and type
			varInfo := fmt.Sprintf("%s%s (%s)", cursor, variable.Name, variable.Type)

			if variable.Required {
				varInfo += " [Required]"
			}

			// Style based on selection
			if m.cursor == i {
				b.WriteString(selectedVariableStyle.Render(varInfo))
			} else {
				b.WriteString(normalVariableStyle.Render(varInfo))
			}
			b.WriteString("\n")

			// Variable details (indented)
			if variable.Default != "" {
				b.WriteString(variableDetailStyle.Render(fmt.Sprintf("    Default: %s", variable.Default)))
				b.WriteString("\n")
			}
			if variable.Regex != "" {
				b.WriteString(variableDetailStyle.Render(fmt.Sprintf("    Pattern: %s", variable.Regex)))
				b.WriteString("\n")
			}

			b.WriteString("\n")
		}
	}

	// Back option
	cursor := "  "
	if m.cursor == m.maxItems-1 {
		cursor = "→ "
	}

	backOption := fmt.Sprintf("%sBack to Schemas", cursor)
	if m.cursor == m.maxItems-1 {
		b.WriteString(selectedItemStyle.Render(backOption))
	} else {
		b.WriteString(normalItemStyle.Render(backOption))
	}

	b.WriteString("\n\n")
	b.WriteString(helpTextStyle.Render("↑/↓: navigate • enter: select • esc: back"))

	return b.String()
}

// Styles for schema detail view
var (
	schemaDetailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				MarginBottom(1)

	variablesSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("33"))

	selectedVariableStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("235")).
				Foreground(lipgloss.Color("255")).
				Padding(0, 1)

	normalVariableStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255"))

	variableDetailStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Italic(true)

	noVariablesStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Italic(true)
)
