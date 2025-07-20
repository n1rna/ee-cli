// Package command contains CLI command implementations.
package command

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/n1rna/menv/internal/logger"
	"github.com/n1rna/menv/internal/util"
	"github.com/spf13/cobra"
)

type ApplyCommand struct {
}

func NewApplyCommand() *cobra.Command {
	ac := &ApplyCommand{}

	cmd := &cobra.Command{
		Use:   "apply [sheet-name] [-- command [args...]]",
		Short: "Apply environment variables and optionally run a command",
		Long: `Apply environment variables to a new shell or run a specific command with the environment.

Examples:
  # Start a new shell with the environment variables
  menv apply myproject --env dev

  # Run a specific command with the environment variables
  menv apply myproject --env dev -- echo $BASE_URL
  menv apply myproject --env dev -- npm start
`,
		Args: cobra.MinimumNArgs(1),
		RunE: ac.Run,
	}

	cmd.Flags().StringP("project", "p", "", "Project name")
	cmd.Flags().StringP("env", "e", "", "Environment name")

	return cmd
}

func (c *ApplyCommand) Run(cmd *cobra.Command, args []string) error {
	storage := GetStorage(cmd.Context())
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// Get sheet name from args or empty string if not provided
	sheetName := ""
	if len(args) > 0 {
		sheetName = args[0]
	}

	// Get project and env flags
	projectFlag, _ := cmd.Flags().GetString("project")
	envFlag, _ := cmd.Flags().GetString("env")

	// Parse sheet reference
	ref, err := util.ParseSheetReference(sheetName, projectFlag, envFlag)
	if err != nil {
		return err
	}

	// Validate sheet reference
	if err := util.ValidateSheetReference(ref, storage); err != nil {
		return err
	}

	// Load config sheet
	configSheet, err := storage.LoadConfigSheet(ref.Project, ref.Env)
	if err != nil {
		return fmt.Errorf("failed to load config sheet: %w", err)
	}

	// Get all original arguments from os.Args
	var cmdArgs []string
	for i, arg := range os.Args {
		if arg == "--" && i < len(os.Args)-1 {
			cmdArgs = os.Args[i+1:]
			break
		}
	}

	// Prepare environment
	env := os.Environ() // Start with current environment
	for key, value := range configSheet.Values {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	logger.Infof("Running command: %s", strings.Join(os.Args, " "))
	if len(cmdArgs) > 0 {
		// Run specific command
		return c.runCommand(cmdArgs, env)
	}

	// No command specified, start a new shell
	return c.startShell(env)
}

func (c *ApplyCommand) runCommand(cmdArgs []string, env []string) error {
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		// Preserve the exit code of the command
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			os.Exit(status.ExitStatus())
		}
	}
	return err
}

func (c *ApplyCommand) startShell(env []string) error {
	var shell, shellArg string

	if runtime.GOOS == "windows" {
		// On Windows, use Command Prompt by default
		shell = os.Getenv("COMSPEC")
		if shell == "" {
			shell = "cmd.exe"
		}
	} else {
		// On Unix-like systems, respect SHELL or fall back to /bin/sh
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}

		// Add interactive flag for better user experience
		shellArg = "-i"
	}

	cmd := exec.Command(shell)
	if shellArg != "" {
		cmd = exec.Command(shell, shellArg)
	}

	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Use syscall.Exec on Unix-like systems for proper shell behavior
	if runtime.GOOS != "windows" {
		return syscall.Exec(shell, []string{shell, shellArg}, env)
	}

	return cmd.Run()
}
