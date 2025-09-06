// Package tui provides project detail view functionality
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/n1rna/ee-cli/internal/schema"
)

// ProjectDetailModel represents the project detail view state
type ProjectDetailModel struct {
	project      *schema.Project
	environments []schema.Environment
	loading      bool
	error        string
	cursor       int
	maxItems     int
}

// NewProjectDetailModel creates a new project detail model
func NewProjectDetailModel() *ProjectDetailModel {
	return &ProjectDetailModel{}
}

// SetProject updates the project being displayed
func (m *ProjectDetailModel) SetProject(project *schema.Project) {
	m.project = project
	m.cursor = 0
	m.loading = false
	m.error = ""
	// Calculate max items: environments + actions
	if project != nil {
		m.maxItems = len(m.environments) + 1 // +1 for "Back" option
	}
}

// SetEnvironments updates the environments list
func (m *ProjectDetailModel) SetEnvironments(environments []schema.Environment) {
	m.environments = environments
	if m.project != nil {
		m.maxItems = len(m.environments) + 1 // +1 for "Back" option
	}
}

// SetLoading sets the loading state
func (m *ProjectDetailModel) SetLoading(loading bool) {
	m.loading = loading
}

// SetError sets an error message
func (m *ProjectDetailModel) SetError(err string) {
	m.error = err
	m.loading = false
}

// Init returns the initial command for the project detail view
func (m ProjectDetailModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the project detail view
func (m ProjectDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.project != nil {
				if m.cursor == m.maxItems-1 {
					// "Back" option selected
					return m, func() tea.Msg { return NavigateMsg(ProjectsView) }
				}
				// Environment selected - could add environment detail view here in the future
			}

		case "esc":
			// Go back to projects list
			return m, func() tea.Msg { return NavigateMsg(ProjectsView) }
		}
	}

	return m, nil
}

// View renders the project detail view
func (m ProjectDetailModel) View() string {
	if m.loading {
		return "\nLoading project details..."
	}

	if m.error != "" {
		return fmt.Sprintf("\nError: %s\n\nPress 'esc' to go back", m.error)
	}

	if m.project == nil {
		return "\nNo project selected.\n\nPress 'esc' to go back"
	}

	var b strings.Builder

	// Header
	b.WriteString(projectDetailTitleStyle.Render(fmt.Sprintf("Project: %s", m.project.Name)))
	b.WriteString("\n\n")

	// Basic information
	if m.project.Description != "" {
		b.WriteString(fmt.Sprintf("Description: %s\n", m.project.Description))
	}

	if m.project.Schema != "" {
		b.WriteString(fmt.Sprintf("Schema: %s\n", m.project.Schema))
	}

	b.WriteString(fmt.Sprintf("Created: %s\n", m.project.CreatedAt.Format("2006-01-02 15:04:05")))
	b.WriteString("\n")

	// Environments section
	if len(m.environments) == 0 {
		b.WriteString(noEnvironmentsStyle.Render("No environments found"))
		b.WriteString("\n\n")
	} else {
		b.WriteString(environmentsSectionStyle.Render("Environments:"))
		b.WriteString("\n\n")

		for i, environment := range m.environments {
			cursor := "  "
			if m.cursor == i {
				cursor = "→ "
			}

			// Environment name and config sheet
			configSheetName := m.project.GetConfigSheetName(environment.Name)
			envInfo := fmt.Sprintf("%s%s (Sheet: %s)", cursor, environment.Name, configSheetName)

			// Style based on selection
			if m.cursor == i {
				b.WriteString(selectedEnvironmentStyle.Render(envInfo))
			} else {
				b.WriteString(normalEnvironmentStyle.Render(envInfo))
			}
			b.WriteString("\n")

			b.WriteString("\n")
		}
	}

	// Back option
	cursor := "  "
	if m.cursor == m.maxItems-1 {
		cursor = "→ "
	}

	backOption := fmt.Sprintf("%sBack to Projects", cursor)
	if m.cursor == m.maxItems-1 {
		b.WriteString(selectedItemStyle.Render(backOption))
	} else {
		b.WriteString(normalItemStyle.Render(backOption))
	}

	b.WriteString("\n\n")
	b.WriteString(helpTextStyle.Render("↑/↓: navigate • enter: select • esc: back"))

	return b.String()
}

// Styles for project detail view
var (
	projectDetailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("33")).
				MarginBottom(1)

	environmentsSectionStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("205"))

	selectedEnvironmentStyle = lipgloss.NewStyle().
					Background(lipgloss.Color("235")).
					Foreground(lipgloss.Color("255")).
					Padding(0, 1)

	normalEnvironmentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255"))

	environmentDetailStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Italic(true)

	noEnvironmentsStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Italic(true)
)
