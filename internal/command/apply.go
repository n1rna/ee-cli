// Package command implements the ee apply command with smart project detection
// This implements the enhanced apply functionality as specified in docs/entities.md
package command

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/schema"
	"github.com/n1rna/ee-cli/internal/storage"
)

// ApplyCommand handles the ee apply command
type ApplyCommand struct{}

// NewApplyCommand creates a new ee apply command
func NewApplyCommand(groupId string) *cobra.Command {
	ac := &ApplyCommand{}

	cmd := &cobra.Command{
		Use:   "apply [environment-name] [-- command [args...]]",
		Short: "Apply environment variables and optionally run a command",
		Long: `Apply environment variables to a new shell or run a specific command with the environment.

This command uses smart project detection to automatically find the current project
and apply environment variables from the specified environment's config sheet.

Examples:
  # Apply development environment and start new shell
  ee apply development

  # Apply production environment and run specific command
  ee apply production -- npm start

  # Apply environment from specific project
  ee apply development --project my-api

  # Apply standalone config sheet
  ee apply my-config --standalone

  # Show what would be applied without executing
  ee apply development --dry-run
`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    ac.Run,
		GroupID: groupId,
	}

	cmd.Flags().StringP("project", "p", "", "Project name (overrides auto-detection)")
	cmd.Flags().Bool("standalone", false, "Apply standalone config sheet")
	cmd.Flags().Bool("dry-run", false, "Show environment variables without applying them")

	return cmd
}

// Run executes the apply command
func (ac *ApplyCommand) Run(cmd *cobra.Command, args []string) error {
	uuidStorage := GetStorage(cmd.Context())
	if uuidStorage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// Get flags
	projectFlag, _ := cmd.Flags().GetString("project")
	standalone, _ := cmd.Flags().GetBool("standalone")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Get target name (environment or config sheet name)
	targetName := args[0]

	// Get command to run (everything after --)
	var commandArgs []string
	if len(args) > 1 {
		commandArgs = args[1:]
	}

	var configSheet *schema.ConfigSheet
	var projectName string
	var err error

	if standalone {
		// Load standalone config sheet
		configSheet, err = ac.loadStandaloneSheet(uuidStorage, targetName)
		if err != nil {
			return err
		}
		projectName = "standalone"
	} else {
		// Smart project detection and environment loading
		configSheet, projectName, err = ac.loadProjectEnvironment(uuidStorage, targetName, projectFlag)
		if err != nil {
			return err
		}
	}

	fmt.Printf("📋 Applying configuration: %s\n", configSheet.Name)
	if !standalone {
		fmt.Printf("Project: %s\n", projectName)
		fmt.Printf("Environment: %s\n", targetName)
	}
	fmt.Printf("Variables: %d\n", len(configSheet.Values))

	if dryRun {
		fmt.Println("\n🔍 Dry run - showing variables that would be applied:")
		ac.showVariables(configSheet.Values)
		return nil
	}

	// Apply environment and run command or start shell
	return ac.applyEnvironment(configSheet.Values, commandArgs)
}

// loadStandaloneSheet loads a standalone config sheet
func (ac *ApplyCommand) loadStandaloneSheet(
	uuidStorage *storage.UUIDStorage,
	sheetName string,
) (*schema.ConfigSheet, error) {
	if !uuidStorage.EntityExists("sheets", sheetName) {
		return nil, fmt.Errorf("config sheet '%s' not found", sheetName)
	}

	configSheet, err := uuidStorage.LoadConfigSheet(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to load config sheet '%s': %w", sheetName, err)
	}

	// Verify it's actually standalone
	if configSheet.Project != "" {
		return nil, fmt.Errorf(
			"config sheet '%s' is not standalone (belongs to project). Use without --standalone flag",
			sheetName,
		)
	}

	return configSheet, nil
}

// loadProjectEnvironment loads an environment config sheet with smart project detection
func (ac *ApplyCommand) loadProjectEnvironment(
	uuidStorage *storage.UUIDStorage,
	envName, projectFlag string,
) (*schema.ConfigSheet, string, error) {
	var project *schema.Project
	var projectName string
	var err error

	if projectFlag != "" {
		// Use specified project
		project, err = uuidStorage.LoadProject(projectFlag)
		if err != nil {
			return nil, "", fmt.Errorf("failed to load project '%s': %w", projectFlag, err)
		}
		projectName = project.Name
	} else {
		// Smart project detection
		project, projectName, err = ac.detectCurrentProject(uuidStorage)
		if err != nil {
			return nil, "", err
		}
	}

	// Check if environment exists in project
	envInfo, exists := project.Environments[envName]
	if !exists {
		available := make([]string, 0, len(project.Environments))
		for env := range project.Environments {
			available = append(available, env)
		}
		if len(available) > 0 {
			return nil, "", fmt.Errorf("environment '%s' not found in project '%s'. Available: %s",
				envName, projectName, strings.Join(available, ", "))
		} else {
			return nil, "", fmt.Errorf(
				"environment '%s' not found in project '%s'. No environments configured. "+
					"Run 'ee sheet create --env %s' to create one",
				envName, projectName, envName)
		}
	}

	// Load the config sheet using naming convention
	configSheetName := project.GetConfigSheetName(envInfo.Name)
	configSheet, err := uuidStorage.LoadConfigSheet(configSheetName)
	if err != nil {
		return nil, "", fmt.Errorf(
			"failed to load config sheet '%s' for environment '%s': %w",
			configSheetName,
			envName,
			err,
		)
	}

	return configSheet, projectName, nil
}

// detectCurrentProject detects the current project from .ee file or suggests projects
func (ac *ApplyCommand) detectCurrentProject(
	uuidStorage *storage.UUIDStorage,
) (*schema.Project, string, error) {
	// Try to load .ee file from current directory
	if EasyEnvFileExists("") {
		menvFile, err := LoadEasyEnvFile("")
		if err != nil {
			return nil, "", fmt.Errorf("failed to load .ee file: %w", err)
		}

		if menvFile.Project == "" {
			return nil, "", fmt.Errorf(".ee file found but no project ID specified")
		}

		project, err := uuidStorage.LoadProject(menvFile.Project)
		if err != nil {
			return nil, "", fmt.Errorf("failed to load project from .ee file: %w", err)
		}

		return project, project.Name, nil
	}

	// No .ee file found, suggest available projects
	projects, err := uuidStorage.ListProjects()
	if err != nil {
		return nil, "", fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		return nil, "", fmt.Errorf(
			"no projects found. Run 'ee init' to create a project or use --standalone flag",
		)
	}

	projectNames := make([]string, len(projects))
	for i, project := range projects {
		projectNames[i] = project.Name
	}

	return nil, "", fmt.Errorf(
		"no .ee file found in current directory. Available projects: %s\nRun 'ee init' or use --project flag",
		strings.Join(projectNames, ", "),
	)
}

// showVariables displays the environment variables that would be applied
func (ac *ApplyCommand) showVariables(values map[string]string) {
	for key, value := range values {
		// Mask sensitive values
		displayValue := value
		if ac.isSensitiveKey(key) {
			displayValue = "***masked***"
		}
		fmt.Printf("  %s=%s\n", key, displayValue)
	}
}

// applyEnvironment applies environment variables and runs command or starts shell
func (ac *ApplyCommand) applyEnvironment(values map[string]string, commandArgs []string) error {
	// Prepare environment
	env := os.Environ()

	// Add our variables
	for key, value := range values {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	if len(commandArgs) > 0 {
		// Run specific command
		fmt.Printf(
			"\n🚀 Running command with applied environment: %s\n",
			strings.Join(commandArgs, " "),
		)

		cmd := exec.Command(commandArgs[0], commandArgs[1:]...)
		cmd.Env = env
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd.Run()
	} else {
		// Start new shell
		shell := ac.getDefaultShell()
		fmt.Printf("\n🐚 Starting new shell with applied environment: %s\n", shell)
		fmt.Println("Type 'exit' to return to the original environment")

		cmd := exec.Command(shell)
		cmd.Env = env
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd.Run()
	}
}

// getDefaultShell returns the default shell for the current platform
func (ac *ApplyCommand) getDefaultShell() string {
	// Check SHELL environment variable first
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}

	// Platform-specific defaults
	switch runtime.GOOS {
	case "windows":
		if powershell, err := exec.LookPath("pwsh"); err == nil {
			return powershell
		}
		if powershell, err := exec.LookPath("powershell"); err == nil {
			return powershell
		}
		return "cmd"
	default:
		// Unix-like systems
		shells := []string{"zsh", "bash", "sh"}
		for _, shell := range shells {
			if path, err := exec.LookPath(shell); err == nil {
				return path
			}
		}
		return "/bin/sh"
	}
}

// isSensitiveKey checks if a key likely contains sensitive information
func (ac *ApplyCommand) isSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	sensitivePatterns := []string{
		"password", "secret", "key", "token", "credential",
		"api_key", "auth", "private", "cert", "ssl",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(key, pattern) {
			return true
		}
	}
	return false
}
