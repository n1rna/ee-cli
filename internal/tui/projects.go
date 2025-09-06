// Package tui provides project management functionality
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/n1rna/ee-cli/internal/schema"
)

// ProjectsModel represents the projects view state
type ProjectsModel struct {
	summaries []*schema.EntitySummary
	cursor    int
	selected  bool
}

// NewProjectsModel creates a new projects model
func NewProjectsModel() *ProjectsModel {
	return &ProjectsModel{
		summaries: []*schema.EntitySummary{},
		cursor:    0,
	}
}

// SetProjectSummaries updates the project summaries list
func (m *ProjectsModel) SetProjectSummaries(summaries []*schema.EntitySummary) {
	m.summaries = summaries
	m.cursor = 0
}

// Init returns the initial command for the projects view
func (m ProjectsModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the projects view
func (m ProjectsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				// Navigate to project detail view by loading the full object
				selectedProject := m.summaries[m.cursor]
				return m, func() tea.Msg {
					return LoadProjectMsg{NameOrUUID: selectedProject.Name}
				}
			}

		case "n":
			// Navigate to create project view
			return m, func() tea.Msg { return NavigateMsg(CreateProjectView) }
		}
	}

	return m, nil
}

// View renders the projects list
func (m ProjectsModel) View() string {
	if len(m.summaries) == 0 {
		return noItemsStyle.Render("\nNo projects found.\n\nPress 'n' to create a new project\nPress 'esc' to go back")
	}

	s := "\nProjects:\n\n"

	for i, project := range m.summaries {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		// Format project info
		projectInfo := fmt.Sprintf("%s %s", cursor, project.Name)
		if project.Description != "" {
			projectInfo += fmt.Sprintf(" - %s", project.Description)
		}

		// Style the current selection
		if m.cursor == i {
			s += selectedItemStyle.Render(projectInfo)
		} else {
			s += normalItemStyle.Render(projectInfo)
		}
		s += "\n"
	}

	s += "\n" + helpTextStyle.Render("enter: view details • n: new project • esc: back")
	return s
}
