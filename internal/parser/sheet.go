package parser

import (
	"fmt"
	"strings"
)

// SheetReference holds the parsed project and environment names
type SheetReference struct {
	Project string
	Env     string
}

// ParseSheetReference parses a sheet reference from either:
// 1. A colon-separated string (project:env)
// 2. Explicit project and env values from flags
func ParseSheetReference(
	sheetName string,
	projectFlag string,
	envFlag string,
) (*SheetReference, error) {
	// Case 1: Using flags
	if projectFlag != "" || envFlag != "" {
		if sheetName != "" {
			return nil, fmt.Errorf("cannot specify both sheet name and project/env flags")
		}
		if projectFlag == "" {
			return nil, fmt.Errorf("project name is required")
		}
		// if envFlag == "" {
		// 	return nil, fmt.Errorf("environment name is required")
		// }
		return &SheetReference{
			Project: projectFlag,
			Env:     envFlag,
		}, nil
	}

	// Case 2: Using colon-separated string
	if sheetName == "" {
		return nil, fmt.Errorf("sheet name or project/env flags are required")
	}

	parts := strings.Split(sheetName, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid sheet name format, expected 'project:env'")
	}

	project := strings.TrimSpace(parts[0])
	env := strings.TrimSpace(parts[1])

	if project == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}

	return &SheetReference{
		Project: project,
		Env:     env,
	}, nil
}

// ValidateSheetReference ensures the referenced project and environment exist
func ValidateSheetReference(ref *SheetReference, storage Storage) error {
	// Check if project exists
	projects, err := storage.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	projectExists := false
	for _, p := range projects {
		if p == ref.Project {
			projectExists = true
			break
		}
	}

	if !projectExists {
		return fmt.Errorf("project %s does not exist", ref.Project)
	}

	// Check if environment exists
	envs, err := storage.ListEnvironments(ref.Project)
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	envExists := false
	for _, e := range envs {
		if e == ref.Env {
			envExists = true
			break
		}
	}

	if !envExists {
		return fmt.Errorf("environment %s does not exist in project %s", ref.Env, ref.Project)
	}

	return nil
}

// Storage interface defines the minimum storage operations needed
type Storage interface {
	ListProjects() ([]string, error)
	ListEnvironments(projectName string) ([]string, error)
}
