// Package util provides unified command context for ee commands.
package util

import (
	"fmt"

	"github.com/n1rna/ee-cli/internal/config"
	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/parser"
)

// CommandContext provides unified context for all commands, including project detection
type CommandContext struct {
	Config        *config.Config
	Manager       *entities.Manager
	ProjectConfig *parser.ProjectConfig
	IsInProject   bool
}

// NewCommandContext creates a new command context with automatic project detection
func NewCommandContext(cfg *config.Config) (*CommandContext, error) {
	// Initialize manager for entity operations
	manager, err := entities.NewManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize manager: %w", err)
	}

	// Try to load project configuration
	projectConfig, err := parser.LoadProjectConfig()
	isInProject := err == nil

	context := &CommandContext{
		Config:        cfg,
		Manager:       manager,
		ProjectConfig: projectConfig,
		IsInProject:   isInProject,
	}

	return context, nil
}

// RequireProjectContext ensures that the command is running in a project context
func (ctx *CommandContext) RequireProjectContext() error {
	if !ctx.IsInProject {
		return fmt.Errorf("this command requires a .ee file (not in a project directory)")
	}
	return nil
}

// GetProjectName returns the project name if in project context
func (ctx *CommandContext) GetProjectName() string {
	if ctx.IsInProject && ctx.ProjectConfig != nil {
		return ctx.ProjectConfig.Project
	}
	return ""
}

// GetEnvironmentNames returns available environment names if in project context
func (ctx *CommandContext) GetEnvironmentNames() []string {
	if ctx.IsInProject && ctx.ProjectConfig != nil {
		return ctx.ProjectConfig.GetEnvironmentNames()
	}
	return []string{}
}

// HasEnvironment checks if the specified environment exists in the project
func (ctx *CommandContext) HasEnvironment(name string) bool {
	if ctx.IsInProject && ctx.ProjectConfig != nil {
		return ctx.ProjectConfig.HasEnvironment(name)
	}
	return false
}

// GetEnvironment returns the environment definition for the specified name
func (ctx *CommandContext) GetEnvironment(name string) (parser.EnvironmentDefinition, error) {
	if !ctx.IsInProject {
		return parser.EnvironmentDefinition{}, fmt.Errorf("not in a project context")
	}

	if ctx.ProjectConfig == nil {
		return parser.EnvironmentDefinition{}, fmt.Errorf("project config not loaded")
	}

	return ctx.ProjectConfig.GetEnvironment(name)
}
