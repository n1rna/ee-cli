// internal/command/env.go
package command

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// EnvCommand handles direct .env file application
type EnvCommand struct{}

func NewEnvCommand() *cobra.Command {
	ec := &EnvCommand{}

	cmd := &cobra.Command{
		Use:   "env [-e envfile] [-- command [args...]]",
		Short: "Apply environment variables from a .env file",
		Long: `Apply environment variables from a .env file and optionally run a command.
If no .env file is specified, it looks for .env in the current directory.

Examples:
  # Apply .env from current directory
  menv env

  # Apply specific .env file
  menv env -e ./config/.env

  # Apply .env and run a command
  menv env -e ./config/.env -- npm start`,
		RunE: ec.Run,
	}

	cmd.Flags().StringP("env-file", "e", ".env", "Path to .env file")
	return cmd
}

func (c *EnvCommand) Run(cmd *cobra.Command, args []string) error {
	envFile, _ := cmd.Flags().GetString("env-file")

	// Resolve relative path to absolute
	absEnvFile, err := filepath.Abs(envFile)
	if err != nil {
		return fmt.Errorf("failed to resolve env file path: %w", err)
	}

	// Load environment variables from file
	env, err := c.loadEnvFile(absEnvFile)
	if err != nil {
		return err
	}

	// Add current environment variables
	env = append(os.Environ(), env...)

	// Get command args
	var cmdArgs []string
	for i, arg := range os.Args {
		if arg == "--" && i < len(os.Args)-1 {
			cmdArgs = os.Args[i+1:]
			break
		}
	}

	if len(cmdArgs) > 0 {
		return c.runCommand(cmdArgs, env)
	}

	return c.startShell(env)
}

func (c *EnvCommand) loadEnvFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open env file: %w", err)
	}
	defer file.Close()

	var env []string
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
			return nil, fmt.Errorf("invalid format at line %d: %s", lineNum, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes if present
		value = strings.Trim(value, `"'`)

		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading env file: %w", err)
	}

	return env, nil
}

func (c *EnvCommand) runCommand(cmdArgs []string, env []string) error {
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			os.Exit(status.ExitStatus())
		}
	}
	return err
}

func (c *EnvCommand) startShell(env []string) error {
	var shell, shellArg string

	if runtime.GOOS == "windows" {
		shell = os.Getenv("COMSPEC")
		if shell == "" {
			shell = "cmd.exe"
		}
	} else {
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
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

	if runtime.GOOS != "windows" {
		return syscall.Exec(shell, []string{shell, shellArg}, env)
	}

	return cmd.Run()
}
