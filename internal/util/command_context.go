// Package util provides unified command context for ee commands.
package util

import (
	"fmt"
	"os"
	"strings"

	"github.com/n1rna/ee-cli/internal/config"
	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/parser"
)

// CommandContext provides unified context for all commands, including project detection
type CommandContext struct {
	Config           *config.Config
	Manager          *entities.Manager
	ProjectConfig    *parser.ProjectConfig
	IsInProject      bool
	ProjectLoadError error // Stores any error from loading project config
}

// NewCommandContext creates a new command context with automatic project detection
func NewCommandContext(cfg *config.Config) (*CommandContext, error) {
	// Initialize manager for entity operations
	manager, err := entities.NewManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize manager: %w", err)
	}

	// Try to load project configuration
	projectConfig, projectLoadErr := parser.LoadProjectConfig()
	isInProject := projectLoadErr == nil

	context := &CommandContext{
		Config:           cfg,
		Manager:          manager,
		ProjectConfig:    projectConfig,
		IsInProject:      isInProject,
		ProjectLoadError: projectLoadErr,
	}

	return context, nil
}

// RequireProjectContext ensures that the command is running in a project context
func (ctx *CommandContext) RequireProjectContext() error {
	if !ctx.IsInProject {
		// If there was an error loading the project, provide detailed feedback
		if ctx.ProjectLoadError != nil {
			// Check if .ee file exists but is malformed
			if _, statErr := os.Stat(config.ProjectConfigFileName); statErr == nil {
				// File exists but couldn't be loaded - provide specific error
				if strings.Contains(ctx.ProjectLoadError.Error(), "failed to parse") {
					return fmt.Errorf("found %s file but it contains invalid JSON: %w", config.ProjectConfigFileName, ctx.ProjectLoadError)
				}
				if strings.Contains(ctx.ProjectLoadError.Error(), "missing required") {
					return fmt.Errorf("found %s file but it's incomplete: %w", config.ProjectConfigFileName, ctx.ProjectLoadError)
				}
				if strings.Contains(ctx.ProjectLoadError.Error(), "failed to read") {
					return fmt.Errorf("found %s file but cannot read it: %w", config.ProjectConfigFileName, ctx.ProjectLoadError)
				}
				// Generic error for file that exists but failed to load
				return fmt.Errorf("found %s file but failed to load it: %w", config.ProjectConfigFileName, ctx.ProjectLoadError)
			}
		}
		// File doesn't exist - standard message
		return fmt.Errorf(
			"this command requires a %s file (not in a project directory)",
			config.ProjectConfigFileName,
		)
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
