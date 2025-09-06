// Package tui provides enhanced schema creation functionality with variable management
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/n1rna/ee-cli/internal/schema"
)

// CreateSchemaEnhancedModel represents the enhanced create schema form state
type CreateSchemaEnhancedModel struct {
	// Basic schema fields
	nameInput        textinput.Model
	descriptionInput textinput.Model

	// Variable management
	variables       []schema.Variable
	editingVariable int // -1 if not editing, index if editing
	varForm         *VariableFormModel

	// Navigation
	mode           SchemaFormMode
	cursor         int
	maxBasicFields int

	// State
	finished bool
	err      error
}

// SchemaFormMode represents the current mode of the form
type SchemaFormMode int

const (
	BasicInfoMode SchemaFormMode = iota
	VariablesListMode
	VariableEditMode
)

// VariableFormModel represents the variable editing form
type VariableFormModel struct {
	nameInput    textinput.Model
	typeInput    textinput.Model
	defaultInput textinput.Model
	regexInput   textinput.Model
	requiredBool bool
	focused      int
}

// NewCreateSchemaEnhancedModel creates a new enhanced schema creation form
func NewCreateSchemaEnhancedModel() *CreateSchemaEnhancedModel {
	// Basic info inputs
	nameInput := textinput.New()
	nameInput.Placeholder = "Enter schema name (e.g., web-service)"
	nameInput.Focus()
	nameInput.CharLimit = 50
	nameInput.Width = 50
	nameInput.Prompt = "Name: "

	descriptionInput := textinput.New()
	descriptionInput.Placeholder = "Enter description (optional)"
	descriptionInput.CharLimit = 200
	descriptionInput.Width = 50
	descriptionInput.Prompt = "Description: "

	return &CreateSchemaEnhancedModel{
		nameInput:        nameInput,
		descriptionInput: descriptionInput,
		variables:        []schema.Variable{},
		editingVariable:  -1,
		mode:             BasicInfoMode,
		maxBasicFields:   2, // name and description
	}
}

// NewVariableFormModel creates a new variable form
func NewVariableFormModel() *VariableFormModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Variable name (e.g., DATABASE_URL)"
	nameInput.Focus()
	nameInput.CharLimit = 50
	nameInput.Width = 40
	nameInput.Prompt = "Name: "

	typeInput := textinput.New()
	typeInput.Placeholder = "Type (string, number, boolean, url)"
	typeInput.CharLimit = 20
	typeInput.Width = 40
	typeInput.Prompt = "Type: "
	typeInput.SetValue("string") // default

	defaultInput := textinput.New()
	defaultInput.Placeholder = "Default value (optional)"
	defaultInput.CharLimit = 100
	defaultInput.Width = 40
	defaultInput.Prompt = "Default: "

	regexInput := textinput.New()
	regexInput.Placeholder = "Regex pattern (optional)"
	regexInput.CharLimit = 100
	regexInput.Width = 40
	regexInput.Prompt = "Pattern: "

	return &VariableFormModel{
		nameInput:    nameInput,
		typeInput:    typeInput,
		defaultInput: defaultInput,
		regexInput:   regexInput,
		requiredBool: false,
	}
}

// Init returns the initial command
func (m CreateSchemaEnhancedModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the enhanced create schema form
func (m CreateSchemaEnhancedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			// Cancel and go back
			return m, func() tea.Msg { return NavigateMsg(SchemasView) }

		case "tab", "shift+tab":
			m.handleTabNavigation(msg.String() == "shift+tab")
			return m, nil

		case "up", "down":
			m.handleUpDownNavigation(msg.String() == "up")
			return m, nil

		case "enter":
			newModel, cmd := m.handleEnterKey()
			return newModel, cmd

		case "a":
			// Add new variable (only in variables list mode)
			if m.mode == VariablesListMode {
				m.mode = VariableEditMode
				m.editingVariable = -1 // new variable
				m.varForm = NewVariableFormModel()
				return m, textinput.Blink
			}

		case "d":
			// Delete variable (only in variables list mode)
			if m.mode == VariablesListMode && len(m.variables) > 0 {
				varIndex := m.cursor
				if varIndex < len(m.variables) {
					m.variables = append(m.variables[:varIndex], m.variables[varIndex+1:]...)
					if m.cursor >= len(m.variables) && len(m.variables) > 0 {
						m.cursor = len(m.variables) - 1
					}
				}
			}

		case "r":
			// Toggle required for variable (only in variable edit mode)
			if m.mode == VariableEditMode && m.varForm != nil {
				m.varForm.requiredBool = !m.varForm.requiredBool
			}

		default:
			// Handle text input updates
			switch m.mode {
			case BasicInfoMode:
				if m.cursor == 0 {
					m.nameInput, cmd = m.nameInput.Update(msg)
				} else if m.cursor == 1 {
					m.descriptionInput, cmd = m.descriptionInput.Update(msg)
				}
				cmds = append(cmds, cmd)

			case VariableEditMode:
				if m.varForm != nil {
					switch m.varForm.focused {
					case 0:
						m.varForm.nameInput, cmd = m.varForm.nameInput.Update(msg)
					case 1:
						m.varForm.typeInput, cmd = m.varForm.typeInput.Update(msg)
					case 2:
						m.varForm.defaultInput, cmd = m.varForm.defaultInput.Update(msg)
					case 3:
						m.varForm.regexInput, cmd = m.varForm.regexInput.Update(msg)
					}
					cmds = append(cmds, cmd)
				}
			}
		}
	}

	m.updateFocus()
	return m, tea.Batch(cmds...)
}

// handleTabNavigation handles tab/shift+tab navigation
func (m *CreateSchemaEnhancedModel) handleTabNavigation(reverse bool) {
	switch m.mode {
	case BasicInfoMode:
		if reverse {
			if m.cursor > 0 {
				m.cursor--
			}
		} else {
			if m.cursor < m.maxBasicFields-1 {
				m.cursor++
			} else {
				// Move to variables mode
				m.mode = VariablesListMode
				m.cursor = 0
			}
		}

	case VariablesListMode:
		if reverse {
			m.mode = BasicInfoMode
			m.cursor = m.maxBasicFields - 1
		}
		// Stay in variables mode for forward navigation

	case VariableEditMode:
		if m.varForm != nil {
			maxVarFields := 4 // name, type, default, regex
			if reverse {
				if m.varForm.focused > 0 {
					m.varForm.focused--
				}
			} else {
				if m.varForm.focused < maxVarFields-1 {
					m.varForm.focused++
				}
			}
		}
	}
}

// handleUpDownNavigation handles up/down arrow navigation
func (m *CreateSchemaEnhancedModel) handleUpDownNavigation(up bool) {
	switch m.mode {
	case VariablesListMode:
		if up {
			if m.cursor > 0 {
				m.cursor--
			}
		} else {
			maxCursor := len(m.variables) // +0 because we can navigate past to add new
			if m.cursor < maxCursor {
				m.cursor++
			}
		}
	}
}

// handleEnterKey handles enter key press
func (m CreateSchemaEnhancedModel) handleEnterKey() (CreateSchemaEnhancedModel, tea.Cmd) {
	switch m.mode {
	case BasicInfoMode:
		// Move to variables mode
		m.mode = VariablesListMode
		m.cursor = 0
		return m, nil

	case VariablesListMode:
		if m.cursor < len(m.variables) {
			// Edit existing variable
			m.mode = VariableEditMode
			m.editingVariable = m.cursor
			m.varForm = m.loadVariableToForm(m.variables[m.cursor])
			return m, textinput.Blink
		} else {
			// Submit schema if we have basic info
			if strings.TrimSpace(m.nameInput.Value()) != "" {
				schema := m.buildSchema()
				return m, func() tea.Msg {
					return CreateSchemaMsg{Schema: schema}
				}
			}
		}

	case VariableEditMode:
		// Save variable
		if m.varForm != nil && strings.TrimSpace(m.varForm.nameInput.Value()) != "" {
			variable := m.buildVariableFromForm()

			if m.editingVariable >= 0 {
				// Update existing
				m.variables[m.editingVariable] = variable
			} else {
				// Add new
				m.variables = append(m.variables, variable)
			}

			// Return to variables list
			m.mode = VariablesListMode
			m.cursor = len(m.variables) - 1
			m.editingVariable = -1
			m.varForm = nil
		}
		return m, nil
	}

	return m, nil
}

// loadVariableToForm loads a variable into the edit form
func (m *CreateSchemaEnhancedModel) loadVariableToForm(variable schema.Variable) *VariableFormModel {
	form := NewVariableFormModel()
	form.nameInput.SetValue(variable.Name)
	form.typeInput.SetValue(variable.Type)
	form.defaultInput.SetValue(variable.Default)
	form.regexInput.SetValue(variable.Regex)
	form.requiredBool = variable.Required
	return form
}

// buildVariableFromForm creates a variable from the current form state
func (m *CreateSchemaEnhancedModel) buildVariableFromForm() schema.Variable {
	if m.varForm == nil {
		return schema.Variable{}
	}

	return schema.Variable{
		Name:     strings.TrimSpace(m.varForm.nameInput.Value()),
		Type:     strings.TrimSpace(m.varForm.typeInput.Value()),
		Default:  strings.TrimSpace(m.varForm.defaultInput.Value()),
		Regex:    strings.TrimSpace(m.varForm.regexInput.Value()),
		Required: m.varForm.requiredBool,
	}
}

// buildSchema creates the final schema from the form state
func (m *CreateSchemaEnhancedModel) buildSchema() *schema.Schema {
	name := strings.TrimSpace(m.nameInput.Value())
	description := strings.TrimSpace(m.descriptionInput.Value())
	return schema.NewSchema(name, description, m.variables, []string{})
}

// updateFocus updates the focus state of inputs
func (m *CreateSchemaEnhancedModel) updateFocus() {
	// Reset all focus
	m.nameInput.Blur()
	m.descriptionInput.Blur()

	if m.varForm != nil {
		m.varForm.nameInput.Blur()
		m.varForm.typeInput.Blur()
		m.varForm.defaultInput.Blur()
		m.varForm.regexInput.Blur()
	}

	// Set appropriate focus
	switch m.mode {
	case BasicInfoMode:
		if m.cursor == 0 {
			m.nameInput.Focus()
		} else if m.cursor == 1 {
			m.descriptionInput.Focus()
		}

	case VariableEditMode:
		if m.varForm != nil {
			switch m.varForm.focused {
			case 0:
				m.varForm.nameInput.Focus()
			case 1:
				m.varForm.typeInput.Focus()
			case 2:
				m.varForm.defaultInput.Focus()
			case 3:
				m.varForm.regexInput.Focus()
			}
		}
	}
}

// View renders the enhanced create schema form
func (m CreateSchemaEnhancedModel) View() string {
	var b strings.Builder

	b.WriteString(enhancedFormTitleStyle.Render("Create New Schema"))
	b.WriteString("\n\n")

	switch m.mode {
	case BasicInfoMode:
		b.WriteString(m.renderBasicInfoMode())
	case VariablesListMode:
		b.WriteString(m.renderVariablesListMode())
	case VariableEditMode:
		b.WriteString(m.renderVariableEditMode())
	}

	return b.String()
}

// renderBasicInfoMode renders the basic info form
func (m CreateSchemaEnhancedModel) renderBasicInfoMode() string {
	var b strings.Builder

	b.WriteString("Basic Information:\n\n")
	b.WriteString(m.nameInput.View())
	b.WriteString("\n")
	b.WriteString(m.descriptionInput.View())
	b.WriteString("\n\n")

	b.WriteString(helpTextStyle.Render("tab: next • enter: continue to variables • esc: cancel"))
	return b.String()
}

// renderVariablesListMode renders the variables list
func (m CreateSchemaEnhancedModel) renderVariablesListMode() string {
	var b strings.Builder

	b.WriteString("Variables:\n\n")

	if len(m.variables) == 0 {
		b.WriteString(noVariablesStyle.Render("No variables defined yet"))
		b.WriteString("\n\n")
	} else {
		for i, variable := range m.variables {
			cursor := "  "
			if m.cursor == i {
				cursor = "→ "
			}

			varText := fmt.Sprintf("%s%s (%s)", cursor, variable.Name, variable.Type)
			if variable.Required {
				varText += " [Required]"
			}

			if m.cursor == i {
				b.WriteString(selectedVariableStyle.Render(varText))
			} else {
				b.WriteString(normalVariableStyle.Render(varText))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Submit option
	cursor := "  "
	if m.cursor == len(m.variables) {
		cursor = "→ "
	}

	submitText := fmt.Sprintf("%sCreate Schema", cursor)
	if m.cursor == len(m.variables) {
		b.WriteString(selectedItemStyle.Render(submitText))
	} else {
		b.WriteString(normalItemStyle.Render(submitText))
	}

	b.WriteString("\n\n")
	b.WriteString(helpTextStyle.Render("↑/↓: navigate • enter: edit/create • a: add variable • d: delete • esc: cancel"))

	return b.String()
}

// renderVariableEditMode renders the variable edit form
func (m CreateSchemaEnhancedModel) renderVariableEditMode() string {
	if m.varForm == nil {
		return "Error: Variable form not initialized"
	}

	var b strings.Builder

	if m.editingVariable >= 0 {
		b.WriteString("Edit Variable:\n\n")
	} else {
		b.WriteString("Add New Variable:\n\n")
	}

	b.WriteString(m.varForm.nameInput.View())
	b.WriteString("\n")
	b.WriteString(m.varForm.typeInput.View())
	b.WriteString("\n")
	b.WriteString(m.varForm.defaultInput.View())
	b.WriteString("\n")
	b.WriteString(m.varForm.regexInput.View())
	b.WriteString("\n\n")

	// Required toggle
	requiredText := fmt.Sprintf("Required: %t (press 'r' to toggle)", m.varForm.requiredBool)
	b.WriteString(variableDetailStyle.Render(requiredText))
	b.WriteString("\n\n")

	b.WriteString(helpTextStyle.Render("tab: next field • r: toggle required • enter: save • esc: cancel"))

	return b.String()
}

// CreateSchemaMsg represents a request to create a schema

// Styles for enhanced form
var (
	enhancedFormTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)
)
