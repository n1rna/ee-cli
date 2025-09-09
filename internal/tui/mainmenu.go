// Package tui provides main menu functionality
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MainMenuModel represents the main menu state
type MainMenuModel struct {
	choices  []string
	cursor   int
	selected bool
}

// NewMainMenuModel creates a new main menu model
func NewMainMenuModel() *MainMenuModel {
	return &MainMenuModel{
		choices: []string{
			"Manage Schemas",
			"Manage Projects",
			"Manage Config Sheets",
			"Exit",
		},
		cursor: 0,
	}
}

// Init returns the initial command for the main menu
func (m MainMenuModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the main menu
func (m MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter", " ":
			m.selected = true
			switch m.cursor {
			case 0: // Manage Schemas
				return m, func() tea.Msg { return NavigateMsg(SchemasView) }
			case 1: // Manage Projects
				return m, func() tea.Msg { return NavigateMsg(ProjectsView) }
			case 2: // Manage Config Sheets
				return m, func() tea.Msg { return NavigateMsg(ConfigSheetsView) }
			case 3: // Exit
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

// View renders the main menu
func (m MainMenuModel) View() string {
	s := "\nChoose an option:\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		// Style the current selection
		line := cursor + " " + choice
		if m.cursor == i {
			s += selectedItemStyle.Render(line)
		} else {
			s += normalItemStyle.Render(line)
		}
		s += "\n"
	}

	s += "\n"
	return s
}

// Styles for main menu
var (
	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)
