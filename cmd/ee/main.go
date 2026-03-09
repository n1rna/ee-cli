// ee is a CLI tool for managing environment variables with schema-based validation.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/command"
	"github.com/n1rna/ee-cli/internal/config"
	"github.com/n1rna/ee-cli/internal/util"
)

var (
	version     = "0.8.0"
	cfgFile     string
	globalFlags = struct {
		debug bool
	}{}
)

func main() {
	// Create root command using the dedicated root command module
	rootCmd := command.NewRootCommand()
	rootCmd.Version = version

	// Only show usage for invalid commands/arguments, not for runtime errors
	rootCmd.SilenceUsage = true

	// Set up persistent pre-run for command context initialization
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Load configuration from environment
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Override config file if specified via flag
		if cfgFile != "" {
			cfg.ConfigFile = cfgFile
		}

		// Initialize command context (includes project detection)
		commandContext, err := util.NewCommandContext(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize command context: %w", err)
		}

		ctx := command.WithCommandContext(cmd.Context(), commandContext)
		cmd.SetContext(ctx)
		return nil
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "",
		"Path to project config file (default: .ee in current directory)")
	rootCmd.PersistentFlags().BoolVar(&globalFlags.debug, "debug", false, "Enable debug output")

	// Add command groups
	rootCmd.AddGroup(&cobra.Group{
		ID:    "global",
		Title: "Global Commands:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "authenticated",
		Title: "Remote Operations:",
	})

	// Add commands organized by groups
	rootCmd.AddCommand(
		// Global Commands - basic project operations
		command.NewInitCommand("global"),    // Project initialization
		command.NewApplyCommand("global"),   // Apply environment variables
		command.NewHydrateCommand("global"), // Generate env file from schema + shell env
		command.NewVerifyCommand("global"),  // Verify project configuration
		command.NewAuthCommand("global"),    // Authentication

		// Remote Operations - push secrets to origins
		command.NewPushCommand("authenticated"),
	)

	// Enable version flag
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
