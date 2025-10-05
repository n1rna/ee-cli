// Package command implements the ee apply command with smart project detection
package command

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/config"
	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/output"
	"github.com/n1rna/ee-cli/internal/parser"
	"github.com/n1rna/ee-cli/internal/util"
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

  # Apply standalone config sheet
  ee apply my-config --standalone

  # Show what would be applied without executing
  ee apply development --dry-run
`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    ac.Run,
		GroupID: groupId,
	}

	cmd.Flags().BoolP("standalone", "s", false,
		"Apply standalone config sheet instead of project environment")
	cmd.Flags().BoolP("dry-run", "d", false, "Show what would be applied without executing")
	cmd.Flags().StringP("format", "f", "env", "Output format for dry-run (env, dotenv, json)")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress informational output")

	return cmd
}

// Run executes the apply command
func (c *ApplyCommand) Run(cmd *cobra.Command, args []string) error {
	// Set up printer
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.Format(format), quiet)

	// Get command context from the global context
	context, err := RequireCommandContext(cmd.Context())
	if err != nil {
		return err
	}

	standalone, _ := cmd.Flags().GetBool("standalone")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

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

	// Detect if the argument is a file path or environment name
	if entities.IsFilePath(envOrSheetName) {
		// Apply .env file directly
		values, err = c.applyEnvFile(envOrSheetName)
		if err != nil {
			return err
		}
		if !quiet {
			printer.Info(fmt.Sprintf("Applying .env file: %s", envOrSheetName))
		}
	} else if standalone {
		// Apply standalone config sheet
		values, err = c.applyStandaloneSheet(context.Manager, envOrSheetName)
		if err != nil {
			return err
		}
		if !quiet {
			printer.Info(fmt.Sprintf("Applying standalone config sheet: %s", envOrSheetName))
		}
	} else {
		// Apply project environment using new .ee file system
		values, err = c.applyProjectEnvironment(context, envOrSheetName)
		if err != nil {
			return err
		}
		if !quiet {
			printer.Info(fmt.Sprintf("Applying environment '%s'", envOrSheetName))
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
func (c *ApplyCommand) applyStandaloneSheet(
	manager *entities.Manager,
	sheetName string,
) (map[string]string, error) {
	cs, err := manager.ConfigSheets.Get(sheetName)
	if err != nil {
		return nil, fmt.Errorf("config sheet '%s' not found: %w", sheetName, err)
	}

	return cs.Values, nil
}

// applyProjectEnvironment applies a project environment using the new .ee file system
func (c *ApplyCommand) applyProjectEnvironment(
	context *util.CommandContext,
	envName string,
) (map[string]string, error) {
	// Check if we're in a project context
	if !context.IsInProject {
		return nil, fmt.Errorf(
			"no %s file found - not in a project context",
			config.ProjectConfigFileName,
		)
	}

	// Validate that the environment exists
	if !context.HasEnvironment(envName) {
		return nil, fmt.Errorf("environment '%s' not found in project", envName)
	}

	// Get the environment definition
	envDef, err := context.GetEnvironment(envName)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment definition: %w", err)
	}

	// Create config sheet merger and merge the environment
	merger := util.NewConfigSheetMerger(context.Manager)
	values, err := merger.MergeEnvironment(envDef)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to merge config sheets for environment '%s': %w",
			envName,
			err,
		)
	}

	return values, nil
}

// applyEnvFile reads and parses a .env file using the parser.ParseFile function
func (c *ApplyCommand) applyEnvFile(filePath string) (map[string]string, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf(".env file not found: %s", filePath)
	}

	// Use the parser to parse the .env file
	p := parser.NewAnnotatedDotEnvParser()
	values, _, err := p.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse .env file: %w", err)
	}

	return values, nil
}

// runCommandWithEnvironment runs a command with the specified environment variables
func (c *ApplyCommand) runCommandWithEnvironment(
	values map[string]string,
	commandArgs []string,
	printer *output.Printer,
) error {
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
func (c *ApplyCommand) startShellWithEnvironment(
	values map[string]string,
	printer *output.Printer,
) error {
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

	printer.Info("Starting shell with environment variables applied")
	printer.Info(fmt.Sprintf("Shell: %s", shell))
	printer.Info(fmt.Sprintf("Applied %d environment variables", len(values)))

	// Start shell
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("shell failed: %w", err)
	}

	return nil
}
