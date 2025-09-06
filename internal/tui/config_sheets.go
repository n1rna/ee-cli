// Package tui provides config sheets management functionality
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/n1rna/ee-cli/internal/schema"
)

// ConfigSheetsModel represents the config sheets view state
type ConfigSheetsModel struct {
	summaries []*schema.ConfigSheetSummary
	cursor    int
	selected  bool
}

// NewConfigSheetsModel creates a new config sheets model
func NewConfigSheetsModel() *ConfigSheetsModel {
	return &ConfigSheetsModel{
		summaries: []*schema.ConfigSheetSummary{},
		cursor:    0,
	}
}

// SetConfigSheetSummaries updates the config sheet summaries list
func (m *ConfigSheetsModel) SetConfigSheetSummaries(summaries []*schema.ConfigSheetSummary) {
	m.summaries = summaries
	m.cursor = 0
}

// Init returns the initial command for the config sheets view
func (m ConfigSheetsModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the config sheets view
func (m ConfigSheetsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				// Navigate to config sheet detail view by loading the full object
				selectedSheet := m.summaries[m.cursor]
				return m, func() tea.Msg {
					return LoadConfigSheetMsg{NameOrUUID: selectedSheet.Name}
				}
			}

		case "n":
			// Navigate to create config sheet view
			return m, func() tea.Msg { return NavigateMsg(CreateConfigSheetView) }
		}
	}

	return m, nil
}

// View renders the config sheets list
func (m ConfigSheetsModel) View() string {
	if len(m.summaries) == 0 {
		return noItemsStyle.Render("\nNo config sheets found.\n\nPress 'n' to create a new config sheet\nPress 'esc' to go back")
	}

	s := "\nConfig Sheets:\n\n"

	for i, sheet := range m.summaries {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		// Format config sheet info
		sheetInfo := fmt.Sprintf("%s %s", cursor, sheet.Name)
		if sheet.Description != "" {
			sheetInfo += fmt.Sprintf(" - %s", sheet.Description)
		}
		if sheet.ProjectGUID != "" {
			sheetInfo += fmt.Sprintf(" [Project: %s", sheet.ProjectGUID)
			if sheet.Environment != "" {
				sheetInfo += fmt.Sprintf("/%s", sheet.Environment)
			}
			sheetInfo += "]"
		} else {
			sheetInfo += " [Standalone]"
		}

		// Style the current selection
		if m.cursor == i {
			s += selectedItemStyle.Render(sheetInfo)
		} else {
			s += normalItemStyle.Render(sheetInfo)
		}
		s += "\n"
	}

	s += "\n" + helpTextStyle.Render("enter: view details • n: new config sheet • esc: back")
	return s
}

// ViewConfigSheetMsg represents a request to view a config sheet's details
