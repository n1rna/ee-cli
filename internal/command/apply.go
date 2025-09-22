// Package command implements the ee apply command with smart project detection
package command

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/output"
)

// ApplyCommand handles the ee apply command
type ApplyCommand struct{}

// NewApplyCommand creates a new ee apply command
func NewApplyCommand(groupId string) *cobra.Command {
	ac := &ApplyCommand{}

	cmd := &cobra.Command{
		Use:     "apply [environment-name|file-path] [-- command [args...]]",
		Aliases: []string{"a"},
		Short:   "Apply environment variables from config sheets or .env files",
		Long: `Apply environment variables to a new shell or run a specific command with the environment.

This command supports multiple sources:
- Project environments (using smart project detection from .ee file)
- Standalone config sheets
- .env files (detected automatically by file path)

Examples:
  # Apply development environment and start new shell
  ee apply development

  # Apply production environment and run specific command
  ee apply production -- npm start

  # Apply .env file from current directory
  ee apply .env

  # Apply .env file with absolute path
  ee apply /path/to/my-app/.env -- npm start

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

	cmd.Flags().String("project", "", "Project name (auto-detected from .ee file if not specified)")
	cmd.Flags().BoolP("standalone", "s", false, "Apply standalone config sheet instead of project environment")
	cmd.Flags().BoolP("dry-run", "d", false, "Show what would be applied without executing")
	cmd.Flags().StringP("format", "f", "env", "Output format for dry-run (env, dotenv, json)")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress informational output")

	return cmd
}

// Run executes the apply command
func (c *ApplyCommand) Run(cmd *cobra.Command, args []string) error {
	// Get manager from context
	manager := GetEntityManager(cmd.Context())
	if manager == nil {
		return fmt.Errorf("entity manager not initialized")
	}

	// Set up printer
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.Format(format), quiet)

	standalone, _ := cmd.Flags().GetBool("standalone")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	projectName, _ := cmd.Flags().GetString("project")

	envOrSheetName := args[0]
	var commandArgs []string

	// Find the separator "--" and split command args
	for i, arg := range args {
		if arg == "--" {
			if i+1 < len(args) {
				commandArgs = args[i+1:]
			}
			break
		}
	}

	var values map[string]string
	var err error

	// Detect if the argument is a file path or environment name
	if c.isFilePath(envOrSheetName) {
		// Apply .env file
		values, err = c.applyEnvFile(envOrSheetName)
		if err != nil {
			return err
		}
		if !quiet {
			absPath, _ := filepath.Abs(envOrSheetName)
			printer.Info(fmt.Sprintf("Applying .env file: %s", absPath))
		}
	} else if standalone {
		// Apply standalone config sheet
		values, err = c.applyStandaloneSheet(manager, envOrSheetName)
		if err != nil {
			return err
		}
		if !quiet {
			printer.Info(fmt.Sprintf("Applying standalone config sheet: %s", envOrSheetName))
		}
	} else {
		// Apply project environment
		values, err = c.applyProjectEnvironment(manager, projectName, envOrSheetName)
		if err != nil {
			return err
		}
		if !quiet {
			actualProject := projectName
			if actualProject == "" {
				actualProject, _ = GetCurrentProject()
			}
			printer.Info(fmt.Sprintf("Applying environment '%s' from project '%s'", envOrSheetName, actualProject))
		}
	}

	if dryRun {
		// Show what would be applied
		printer.Info("Environment variables that would be applied:")
		switch format {
		case "env":
			return printer.PrintEnvironmentExport(values)
		case "dotenv":
			return printer.PrintDotEnv(values)
		case "json":
			return printer.PrintValues(values)
		default:
			return fmt.Errorf("unsupported format: %s", format)
		}
	}

	// Apply environment variables
	if len(commandArgs) > 0 {
		// Run specific command with environment
		return c.runCommandWithEnvironment(values, commandArgs, printer)
	} else {
		// Start new shell with environment
		return c.startShellWithEnvironment(values, printer)
	}
}

// applyStandaloneSheet applies a standalone config sheet
func (c *ApplyCommand) applyStandaloneSheet(manager *entities.Manager, sheetName string) (map[string]string, error) {
	cs, err := manager.ConfigSheets.Get(sheetName)
	if err != nil {
		return nil, fmt.Errorf("config sheet '%s' not found: %w", sheetName, err)
	}

	if !cs.IsStandalone() {
		return nil, fmt.Errorf("config sheet '%s' is not standalone (it's associated with project '%s')", sheetName, cs.Project)
	}

	return cs.Values, nil
}

// applyProjectEnvironment applies a project environment
func (c *ApplyCommand) applyProjectEnvironment(manager *entities.Manager, projectName, envName string) (map[string]string, error) {
	// If no project name specified, try to get from .ee file
	if projectName == "" {
		var err error
		projectName, err = GetCurrentProject()
		if err != nil {
			return nil, fmt.Errorf("no project specified and no .ee file found: %w", err)
		}
		if projectName == "" {
			return nil, fmt.Errorf("no project specified and .ee file is empty")
		}
	}

	// Load project
	p, err := manager.Projects.Get(projectName)
	if err != nil {
		return nil, fmt.Errorf("project '%s' not found: %w", projectName, err)
	}

	// Check if environment exists in project
	if _, exists := p.Environments[envName]; !exists {
		return nil, fmt.Errorf("environment '%s' not found in project '%s'", envName, projectName)
	}

	// Find config sheet for this environment
	configSheetName := p.GetConfigSheetName(envName)
	cs, err := manager.ConfigSheets.Get(configSheetName)
	if err != nil {
		return nil, fmt.Errorf("config sheet '%s' not found for environment '%s': %w", configSheetName, envName, err)
	}

	return cs.Values, nil
}

// runCommandWithEnvironment runs a command with the specified environment variables
func (c *ApplyCommand) runCommandWithEnvironment(values map[string]string, commandArgs []string, printer *output.Printer) error {
	if len(commandArgs) == 0 {
		return fmt.Errorf("no command specified")
	}

	cmdName := commandArgs[0]
	args := commandArgs[1:]

	// Create command
	cmd := exec.Command(cmdName, args...)

	// Set up environment
	cmd.Env = os.Environ()
	for key, value := range values {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Set up I/O
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	printer.Info(fmt.Sprintf("Running command: %s", strings.Join(commandArgs, " ")))

	// Run command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

// startShellWithEnvironment starts a new shell with the specified environment variables
func (c *ApplyCommand) startShellWithEnvironment(values map[string]string, printer *output.Printer) error {
	// Determine shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		if runtime.GOOS == "windows" {
			shell = "cmd"
		} else {
			shell = "/bin/bash"
		}
	}

	// Create shell command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd")
	} else {
		cmd = exec.Command(shell)
	}

	// Set up environment
	cmd.Env = os.Environ()
	for key, value := range values {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Set up I/O
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	printer.Info(fmt.Sprintf("Starting shell with environment variables applied"))
	printer.Info(fmt.Sprintf("Shell: %s", shell))
	printer.Info(fmt.Sprintf("Applied %d environment variables", len(values)))

	// Start shell
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("shell failed: %w", err)
	}

	return nil
}

// isFilePath detects if the argument is a file path rather than an environment name
// Returns true if the argument starts with '.', '/', '~', or contains a file extension
func (c *ApplyCommand) isFilePath(arg string) bool {
	// Check if it's a relative path starting with '.' or current directory
	if strings.HasPrefix(arg, ".") {
		return true
	}

	// Check if it's an absolute path starting with '/' or '~'
	if strings.HasPrefix(arg, "/") || strings.HasPrefix(arg, "~") {
		return true
	}

	// Check if it contains a file extension
	if filepath.Ext(arg) != "" {
		return true
	}

	// Check if the file actually exists
	if _, err := os.Stat(arg); err == nil {
		return true
	}

	return false
}

// applyEnvFile reads and parses a .env file
func (c *ApplyCommand) applyEnvFile(filePath string) (map[string]string, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf(".env file not found: %s", filePath)
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open .env file: %w", err)
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line %d in .env file: %s", lineNum, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes if present
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}

		// Validate key name
		if key == "" {
			return nil, fmt.Errorf("empty variable name on line %d", lineNum)
		}

		values[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .env file: %w", err)
	}

	return values, nil
}
