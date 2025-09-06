// Package tui provides a terminal user interface for ee
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/n1rna/ee-cli/internal/schema"
	"github.com/n1rna/ee-cli/internal/storage"
)

// ViewState represents the current view in the TUI
type ViewState int

const (
	MainMenuView ViewState = iota
	SchemasView
	ProjectsView
	SchemaDetailView
	ProjectDetailView
	CreateSchemaView
	CreateProjectView
	ConfigSheetsView
	ConfigSheetDetailView
	CreateConfigSheetView
)

// Model represents the main TUI application state
type Model struct {
	// Navigation
	currentView ViewState
	width       int
	height      int

	// Storage
	storage *storage.UUIDStorage

	// Data
	schemaSummaries      []*schema.EntitySummary
	projectSummaries     []*schema.EntitySummary
	configSheetSummaries []*schema.ConfigSheetSummary

	// State
	loading      bool
	error        string
	selectedItem int

	// Views
	mainMenu              *MainMenuModel
	schemasView           *SchemasModel
	projectsView          *ProjectsModel
	createSchemaView      *CreateSchemaEnhancedModel
	schemaDetailView      *SchemaDetailModel
	projectDetailView     *ProjectDetailModel
	createProjectView     *CreateProjectModel
	configSheetsView      *ConfigSheetsModel
	configSheetDetailView *ConfigSheetDetailModel
}

// NewModel creates a new TUI model
func NewModel(storage *storage.UUIDStorage) *Model {
	return &Model{
		currentView:           MainMenuView,
		storage:               storage,
		selectedItem:          0,
		mainMenu:              NewMainMenuModel(),
		schemasView:           NewSchemasModel(),
		projectsView:          NewProjectsModel(),
		createSchemaView:      NewCreateSchemaEnhancedModel(),
		schemaDetailView:      NewSchemaDetailModel(),
		projectDetailView:     NewProjectDetailModel(),
		createProjectView:     NewCreateProjectModel(),
		configSheetsView:      NewConfigSheetsModel(),
		configSheetDetailView: NewConfigSheetDetailModel(),
	}
}

// Init returns initial commands for the application
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.currentView == MainMenuView {
				return m, tea.Quit
			}
			// Go back to main menu from other views
			m.currentView = MainMenuView
			m.error = ""
			return m, nil

		case "esc":
			// Go back to main menu
			m.currentView = MainMenuView
			m.error = ""
			return m, nil
		}

	case SchemasLoadedMsg:
		m.schemaSummaries = []*schema.EntitySummary(msg)
		m.loading = false
		m.schemasView.SetSchemaSummaries(m.schemaSummaries)

	case ProjectsLoadedMsg:
		m.projectSummaries = []*schema.EntitySummary(msg)
		m.loading = false
		m.projectsView.SetProjectSummaries(m.projectSummaries)

	case ErrorMsg:
		m.loading = false
		m.error = string(msg)

	case NavigateMsg:
		m.currentView = ViewState(msg)
		m.error = ""
		switch m.currentView {
		case SchemasView:
			if len(m.schemaSummaries) == 0 {
				m.loading = true
				cmds = append(cmds, m.loadSchemas())
			}
		case ProjectsView:
			if len(m.projectSummaries) == 0 {
				m.loading = true
				cmds = append(cmds, m.loadProjects())
			}
		case ConfigSheetsView:
			if len(m.configSheetSummaries) == 0 {
				m.loading = true
				cmds = append(cmds, m.loadConfigSheets())
			}
		case CreateSchemaView:
			m.createSchemaView = NewCreateSchemaEnhancedModel() // Reset form
		case CreateProjectView:
			m.createProjectView = NewCreateProjectModel() // Reset form
			// Load available schemas for the project form - we'll need to load full schemas
			// TODO: Implement loading full schemas for the form
		case CreateConfigSheetView:
			// TODO: Reset config sheet form when implemented
		}

	case CreateSchemaMsg:
		m.loading = true
		cmds = append(cmds, m.createSchema(msg.Schema))

	case SchemaCreatedMsg:
		m.loading = false
		m.schemaSummaries = msg.Summaries
		m.schemasView.SetSchemaSummaries(m.schemaSummaries)
		m.currentView = SchemasView
		m.error = "" // Clear any previous errors

	case ViewSchemaMsg:
		m.schemaDetailView.SetSchema(&msg.Schema)
		m.currentView = SchemaDetailView
		m.error = ""

	case ViewProjectMsg:
		m.projectDetailView.SetProject(&msg.Project)
		m.currentView = ProjectDetailView
		m.error = ""

	case CreateProjectMsg:
		m.loading = true
		cmds = append(cmds, m.createProject(msg.Project))

	case ProjectCreatedMsg:
		m.loading = false
		m.projectSummaries = msg.Summaries
		m.projectsView.SetProjectSummaries(m.projectSummaries)
		m.currentView = ProjectsView
		m.error = ""

	case ConfigSheetsLoadedMsg:
		m.configSheetSummaries = []*schema.ConfigSheetSummary(msg)
		m.loading = false
		m.configSheetsView.SetConfigSheetSummaries(m.configSheetSummaries)

	case ViewConfigSheetMsg:
		m.configSheetDetailView.SetConfigSheet(&msg.ConfigSheet)
		m.currentView = ConfigSheetDetailView
		m.error = ""

	case LoadConfigSheetMsg:
		m.loading = true
		cmds = append(cmds, m.loadConfigSheet(msg.NameOrUUID))

	case LoadSchemaMsg:
		m.loading = true
		cmds = append(cmds, m.loadSchema(msg.NameOrUUID))

	case LoadProjectMsg:
		m.loading = true
		cmds = append(cmds, m.loadProject(msg.NameOrUUID))
	}

	// Update current view
	switch m.currentView {
	case MainMenuView:
		var mainMenuModel tea.Model
		mainMenuModel, cmd = m.mainMenu.Update(msg)
		if mm, ok := mainMenuModel.(MainMenuModel); ok {
			m.mainMenu = &mm
		}
		cmds = append(cmds, cmd)

	case SchemasView:
		var schemasModel tea.Model
		schemasModel, cmd = m.schemasView.Update(msg)
		if sm, ok := schemasModel.(SchemasModel); ok {
			m.schemasView = &sm
		}
		cmds = append(cmds, cmd)

	case ProjectsView:
		var projectsModel tea.Model
		projectsModel, cmd = m.projectsView.Update(msg)
		if pm, ok := projectsModel.(ProjectsModel); ok {
			m.projectsView = &pm
		}
		cmds = append(cmds, cmd)

	case CreateSchemaView:
		var createSchemaModel tea.Model
		createSchemaModel, cmd = m.createSchemaView.Update(msg)
		if csm, ok := createSchemaModel.(CreateSchemaEnhancedModel); ok {
			m.createSchemaView = &csm
		}
		cmds = append(cmds, cmd)

	case SchemaDetailView:
		var schemaDetailModel tea.Model
		schemaDetailModel, cmd = m.schemaDetailView.Update(msg)
		if sdm, ok := schemaDetailModel.(SchemaDetailModel); ok {
			m.schemaDetailView = &sdm
		}
		cmds = append(cmds, cmd)

	case ProjectDetailView:
		var projectDetailModel tea.Model
		projectDetailModel, cmd = m.projectDetailView.Update(msg)
		if pdm, ok := projectDetailModel.(ProjectDetailModel); ok {
			m.projectDetailView = &pdm
		}
		cmds = append(cmds, cmd)

	case CreateProjectView:
		var createProjectModel tea.Model
		createProjectModel, cmd = m.createProjectView.Update(msg)
		if cpm, ok := createProjectModel.(CreateProjectModel); ok {
			m.createProjectView = &cpm
		}
		cmds = append(cmds, cmd)

	case ConfigSheetsView:
		var configSheetsModel tea.Model
		configSheetsModel, cmd = m.configSheetsView.Update(msg)
		if csm, ok := configSheetsModel.(ConfigSheetsModel); ok {
			m.configSheetsView = &csm
		}
		cmds = append(cmds, cmd)

	case ConfigSheetDetailView:
		var configSheetDetailModel tea.Model
		configSheetDetailModel, cmd = m.configSheetDetailView.Update(msg)
		if csdm, ok := configSheetDetailModel.(ConfigSheetDetailModel); ok {
			m.configSheetDetailView = &csdm
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the current view
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content string

	// Header
	header := m.headerView()

	// Content based on current view
	switch m.currentView {
	case MainMenuView:
		content = m.mainMenu.View()
	case SchemasView:
		if m.loading {
			content = "Loading schemas..."
		} else {
			content = m.schemasView.View()
		}
	case ProjectsView:
		if m.loading {
			content = "Loading projects..."
		} else {
			content = m.projectsView.View()
		}
	case CreateSchemaView:
		if m.loading {
			content = "Creating schema..."
		} else {
			content = m.createSchemaView.View()
		}
	case SchemaDetailView:
		content = m.schemaDetailView.View()
	case ProjectDetailView:
		content = m.projectDetailView.View()
	case CreateProjectView:
		if m.loading {
			content = "Creating project..."
		} else {
			content = m.createProjectView.View()
		}
	case ConfigSheetsView:
		if m.loading {
			content = "Loading config sheets..."
		} else {
			content = m.configSheetsView.View()
		}
	case ConfigSheetDetailView:
		content = m.configSheetDetailView.View()
	case CreateConfigSheetView:
		if m.loading {
			content = "Creating config sheet..."
		} else {
			content = "Config sheet creation not implemented yet"
		}
	default:
		content = "View not implemented"
	}

	// Error display
	if m.error != "" {
		content += "\n" + errorStyle.Render("Error: "+m.error)
	}

	// Footer
	footer := m.footerView()

	// Combine all parts
	return header + "\n" + content + "\n" + footer
}

// headerView renders the application header
func (m Model) headerView() string {
	title := titleStyle.Render("ee TUI")

	var subtitle string
	switch m.currentView {
	case MainMenuView:
		subtitle = "Main Menu"
	case SchemasView:
		subtitle = "Schemas"
	case ProjectsView:
		subtitle = "Projects"
	case CreateSchemaView:
		subtitle = "Create Schema"
	case SchemaDetailView:
		subtitle = "Schema Details"
	case ProjectDetailView:
		subtitle = "Project Details"
	case CreateProjectView:
		subtitle = "Create Project"
	case ConfigSheetsView:
		subtitle = "Config Sheets"
	case ConfigSheetDetailView:
		subtitle = "Config Sheet Details"
	case CreateConfigSheetView:
		subtitle = "Create Config Sheet"
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, subtitleStyle.Render(subtitle))
}

// footerView renders the application footer with help
func (m Model) footerView() string {
	help := ""
	switch m.currentView {
	case MainMenuView:
		help = "↑/↓: navigate • enter: select • q: quit"
	case SchemasView, ProjectsView:
		help = "↑/↓: navigate • enter: select • esc: back • q: quit"
	case CreateSchemaView:
		help = "tab: next field • enter: submit • esc: cancel"
	}

	return helpStyle.Render(help)
}

// loadSchemas creates a command to load schema summaries from storage
func (m Model) loadSchemas() tea.Cmd {
	return func() tea.Msg {
		summaries, err := m.storage.ListSchemas()
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Failed to load schemas: %v", err))
		}
		return SchemasLoadedMsg(summaries)
	}
}

// loadProjects creates a command to load project summaries from storage
func (m Model) loadProjects() tea.Cmd {
	return func() tea.Msg {
		summaries, err := m.storage.ListProjects()
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Failed to load projects: %v", err))
		}
		return ProjectsLoadedMsg(summaries)
	}
}

// loadConfigSheets creates a command to load config sheet summaries from storage
func (m Model) loadConfigSheets() tea.Cmd {
	return func() tea.Msg {
		filter := schema.ConfigSheetFilter{} // Empty filter to get all
		summaries, err := m.storage.ListConfigSheets(&filter)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Failed to load config sheets: %v", err))
		}
		return ConfigSheetsLoadedMsg(summaries)
	}
}

// loadConfigSheet creates a command to load a single config sheet for detail view
func (m Model) loadConfigSheet(nameOrUUID string) tea.Cmd {
	return func() tea.Msg {
		configSheet, err := m.storage.LoadConfigSheet(nameOrUUID)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Failed to load config sheet: %v", err))
		}
		return ViewConfigSheetMsg{ConfigSheet: *configSheet}
	}
}

// loadSchema creates a command to load a single schema for detail view
func (m Model) loadSchema(nameOrUUID string) tea.Cmd {
	return func() tea.Msg {
		schema, err := m.storage.LoadSchema(nameOrUUID)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Failed to load schema: %v", err))
		}
		return ViewSchemaMsg{Schema: *schema}
	}
}

// loadProject creates a command to load a single project for detail view
func (m Model) loadProject(nameOrUUID string) tea.Cmd {
	return func() tea.Msg {
		project, err := m.storage.LoadProject(nameOrUUID)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Failed to load project: %v", err))
		}
		return ViewProjectMsg{Project: *project}
	}
}

// createSchema creates a command to create a schema via storage
func (m Model) createSchema(schema *schema.Schema) tea.Cmd {
	return func() tea.Msg {
		err := m.storage.SaveSchema(schema)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Failed to create schema: %v", err))
		}
		// After successful creation, refresh the schemas list and go back
		summaries, err := m.storage.ListSchemas()
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Schema created but failed to refresh list: %v", err))
		}
		return SchemaCreatedMsg{
			Schema:    *schema,
			Summaries: summaries,
		}
	}
}

// createProject creates a command to create a project via storage
func (m Model) createProject(project *schema.Project) tea.Cmd {
	return func() tea.Msg {
		err := m.storage.SaveProject(project)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Failed to create project: %v", err))
		}
		// After successful creation, refresh the projects list and go back
		summaries, err := m.storage.ListProjects()
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Project created but failed to refresh list: %v", err))
		}
		return ProjectCreatedMsg{
			Project:   *project,
			Summaries: summaries,
		}
	}
}

// Custom messages
type SchemasLoadedMsg []*schema.EntitySummary
type ProjectsLoadedMsg []*schema.EntitySummary
type ConfigSheetsLoadedMsg []*schema.ConfigSheetSummary
type ErrorMsg string
type NavigateMsg ViewState

// SchemaCreatedMsg represents a successful schema creation
type SchemaCreatedMsg struct {
	Schema    schema.Schema
	Summaries []*schema.EntitySummary
}

// ViewSchemaMsg represents a request to view a schema's details
type ViewSchemaMsg struct {
	Schema schema.Schema
}

// ViewProjectMsg represents a request to view a project's details
type ViewProjectMsg struct {
	Project schema.Project
}

// ProjectCreatedMsg represents a successful project creation
type ProjectCreatedMsg struct {
	Project   schema.Project
	Summaries []*schema.EntitySummary
}

// CreateSchemaMsg represents a request to create a schema
type CreateSchemaMsg struct {
	Schema *schema.Schema
}

// CreateProjectMsg represents a request to create a project
type CreateProjectMsg struct {
	Project *schema.Project
}

// ViewConfigSheetMsg represents a request to view a config sheet
type ViewConfigSheetMsg struct {
	ConfigSheet schema.ConfigSheet
}

// LoadConfigSheetMsg represents a request to load and view a config sheet by name/ID
type LoadConfigSheetMsg struct {
	NameOrUUID string
}

// LoadSchemaMsg represents a request to load and view a schema by name/ID
type LoadSchemaMsg struct {
	NameOrUUID string
}

// LoadProjectMsg represents a request to load and view a project by name/ID
type LoadProjectMsg struct {
	NameOrUUID string
}

// Styles
var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)
