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
		Short:   "Apply environment variables from .env files or project environments",
		Long: `Apply environment variables to a new shell or run a specific command with the environment.

This command supports multiple sources:
- Project environments (using smart project detection from .ee file)
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

  # Show what would be applied without executing
  ee apply development --dry-run
`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    ac.Run,
		GroupID: groupId,
	}

	cmd.Flags().BoolP("dry-run", "d", false,
		"Show what would be applied without executing")
	cmd.Flags().StringP("format", "f", "env",
		"Output format for dry-run (env, dotenv, json)")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress informational output")

	return cmd
}

// Run executes the apply command
func (c *ApplyCommand) Run(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.Format(format), quiet)

	context, err := RequireCommandContext(cmd.Context())
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")

	envOrFile := args[0]
	var commandArgs []string

	// Get all original arguments from os.Args
	for i, arg := range os.Args {
		if arg == "--" && i < len(os.Args)-1 {
			commandArgs = os.Args[i+1:]
			break
		}
	}

	var values map[string]string

	// Detect if the argument is a file path or environment name
	if isFilePath(envOrFile) {
		values, err = c.applyEnvFile(envOrFile)
		if err != nil {
			return err
		}
		if !quiet && format != "json" {
			printer.Info(fmt.Sprintf(
				"Applying .env file: %s", envOrFile,
			))
		}
	} else {
		values, err = c.applyProjectEnvironment(context, envOrFile)
		if err != nil {
			return err
		}
		if !quiet && format != "json" {
			printer.Info(fmt.Sprintf(
				"Applying environment '%s'", envOrFile,
			))
		}
	}

	if dryRun {
		if format != "json" && !quiet {
			printer.Info("Environment variables that would be applied:")
		}
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
		return c.runCommandWithEnvironment(values, commandArgs, printer)
	}
	return c.startShellWithEnvironment(values, printer)
}

// applyProjectEnvironment applies a project environment using the .ee file
func (c *ApplyCommand) applyProjectEnvironment(
	context *util.CommandContext,
	envName string,
) (map[string]string, error) {
	if !context.IsInProject {
		return nil, fmt.Errorf(
			"no %s file found - not in a project context",
			config.ProjectConfigFileName,
		)
	}

	if !context.HasEnvironment(envName) {
		return nil, fmt.Errorf(
			"environment '%s' not found in project", envName,
		)
	}

	envDef, err := context.GetEnvironment(envName)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get environment definition: %w", err,
		)
	}

	resolver := util.NewEnvResolver()
	values, err := resolver.MergeEnvironment(
		util.EnvironmentSources{
			Env:     envDef.Env,
			Sources: envDef.Sources,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to resolve environment '%s': %w", envName, err,
		)
	}

	return values, nil
}

// applyEnvFile reads and parses a .env file
func (c *ApplyCommand) applyEnvFile(
	filePath string,
) (map[string]string, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf(".env file not found: %s", filePath)
	}

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

	cmd := exec.Command(cmdName, args...)
	cmd.Env = os.Environ()
	for key, value := range values {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	printer.Info(fmt.Sprintf(
		"Running command: %s", strings.Join(commandArgs, " "),
	))

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
	shell := os.Getenv("SHELL")
	if shell == "" {
		if runtime.GOOS == "windows" {
			shell = "cmd"
		} else {
			shell = "/bin/bash"
		}
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd")
	} else {
		cmd = exec.Command(shell)
	}

	cmd.Env = os.Environ()
	for key, value := range values {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	printer.Info("Starting shell with environment variables applied")
	printer.Info(fmt.Sprintf("Shell: %s", shell))
	printer.Info(fmt.Sprintf(
		"Applied %d environment variables", len(values),
	))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("shell failed: %w", err)
	}
	return nil
}

// isFilePath detects if the argument is a file path rather than an environment name
func isFilePath(arg string) bool {
	if strings.HasPrefix(arg, ".") || strings.HasPrefix(arg, "/") ||
		strings.HasPrefix(arg, "~") {
		return true
	}
	if _, err := os.Stat(arg); err == nil {
		return true
	}
	return false
}
