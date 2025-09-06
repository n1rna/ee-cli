// Package tui provides project creation functionality
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/n1rna/ee-cli/internal/schema"
)

// CreateProjectModel represents the create project form state
type CreateProjectModel struct {
	nameInput            textinput.Model
	descriptionInput     textinput.Model
	defaultSchemaIDInput textinput.Model

	// Available schemas for selection
	availableSchemas []schema.Schema

	// Navigation
	focused  int
	finished bool
	err      error
}

const (
	projectNameInput int = iota
	projectDescriptionInput
	projectDefaultSchemaInput
)

// NewCreateProjectModel creates a new project creation form
func NewCreateProjectModel() *CreateProjectModel {
	// Name input
	nameInput := textinput.New()
	nameInput.Placeholder = "Enter project name (e.g., my-web-app)"
	nameInput.Focus()
	nameInput.CharLimit = 50
	nameInput.Width = 50
	nameInput.Prompt = "Name: "

	// Description input
	descriptionInput := textinput.New()
	descriptionInput.Placeholder = "Enter description (optional)"
	descriptionInput.CharLimit = 200
	descriptionInput.Width = 50
	descriptionInput.Prompt = "Description: "

	// Default Schema ID input
	defaultSchemaInput := textinput.New()
	defaultSchemaInput.Placeholder = "Enter default schema ID (optional)"
	defaultSchemaInput.CharLimit = 10
	defaultSchemaInput.Width = 50
	defaultSchemaInput.Prompt = "Default Schema ID: "

	return &CreateProjectModel{
		nameInput:            nameInput,
		descriptionInput:     descriptionInput,
		defaultSchemaIDInput: defaultSchemaInput,
		availableSchemas:     []schema.Schema{},
	}
}

// SetAvailableSchemas sets the list of available schemas
func (m *CreateProjectModel) SetAvailableSchemas(schemas []schema.Schema) {
	m.availableSchemas = schemas
}

// Init returns the initial command
func (m CreateProjectModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the create project form
func (m CreateProjectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			// Cancel and go back
			return m, func() tea.Msg { return NavigateMsg(ProjectsView) }

		case "tab", "shift+tab", "up", "down":
			// Switch between inputs
			if msg.String() == "up" || msg.String() == "shift+tab" {
				m.focused--
			} else {
				m.focused++
			}

			if m.focused > 2 { // 3 inputs total (0, 1, 2)
				m.focused = 0
			} else if m.focused < 0 {
				m.focused = 2
			}

			// Update focus
			m.updateFocus()
			return m, textinput.Blink

		case "enter":
			// Submit form if name is provided
			if strings.TrimSpace(m.nameInput.Value()) != "" {
				// Create the project
				project := m.buildProject()
				return m, func() tea.Msg {
					return CreateProjectMsg{Project: project}
				}
			}

		default:
			// Handle text input updates
			switch m.focused {
			case projectNameInput:
				m.nameInput, cmd = m.nameInput.Update(msg)
			case projectDescriptionInput:
				m.descriptionInput, cmd = m.descriptionInput.Update(msg)
			case projectDefaultSchemaInput:
				m.defaultSchemaIDInput, cmd = m.defaultSchemaIDInput.Update(msg)
			}
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// updateFocus updates the focus state of inputs
func (m *CreateProjectModel) updateFocus() {
	// Reset all focus
	m.nameInput.Blur()
	m.descriptionInput.Blur()
	m.defaultSchemaIDInput.Blur()

	// Set appropriate focus
	switch m.focused {
	case projectNameInput:
		m.nameInput.Focus()
	case projectDescriptionInput:
		m.descriptionInput.Focus()
	case projectDefaultSchemaInput:
		m.defaultSchemaIDInput.Focus()
	}
}

// buildProject creates the final project from the form state
func (m *CreateProjectModel) buildProject() *schema.Project {
	name := strings.TrimSpace(m.nameInput.Value())
	description := strings.TrimSpace(m.descriptionInput.Value())
	// For now, we'll use an empty schema ID - TODO: implement schema selection
	return schema.NewProject(name, description, "")
}

// View renders the create project form
func (m CreateProjectModel) View() string {
	var b strings.Builder

	b.WriteString(createProjectTitleStyle.Render("Create New Project"))
	b.WriteString("\n\n")

	// Form inputs
	b.WriteString(m.nameInput.View())
	b.WriteString("\n")
	b.WriteString(m.descriptionInput.View())
	b.WriteString("\n")
	b.WriteString(m.defaultSchemaIDInput.View())
	b.WriteString("\n\n")

	// Available schemas help
	if len(m.availableSchemas) > 0 {
		b.WriteString(availableSchemasStyle.Render("Available Schemas:"))
		b.WriteString("\n")
		for _, schema := range m.availableSchemas {
			b.WriteString(schemaHintStyle.Render(fmt.Sprintf("  %s - %s", schema.ID, schema.Name)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Submit button
	button := blurredButtonStyle.Render("[ Create Project ]")
	if m.focused == 3 { // After all text inputs
		button = focusedButtonStyle.Render("[ Create Project ]")
	}

	b.WriteString(button)
	b.WriteString("\n\n")
	b.WriteString(helpTextStyle.Render("tab: next field • enter: create • esc: cancel"))

	return b.String()
}

// CreateProjectMsg represents a request to create a project

// Styles for the project creation form
var (
	createProjectTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("33")).
				MarginBottom(1)

	availableSchemasStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205"))

	schemaHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)

	focusedButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Background(lipgloss.Color("235")).
				Padding(0, 3).
				Bold(true)

	blurredButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Background(lipgloss.Color("235")).
				Padding(0, 3)
)
