// ee is a CLI tool for managing environment variables with schema-based validation.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/command"
	"github.com/n1rna/ee-cli/internal/config"
	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/util"
)

var (
	version     = "0.4.0"
	cfgBaseDir  string
	globalFlags = struct {
		debug bool
	}{}
)

func main() {
	// Create root command using the dedicated root command module
	rootCmd := command.NewRootCommand()
	rootCmd.Version = version

	// Set up persistent pre-run for entity manager and command context initialization
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Load configuration from environment
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Override base directory if specified via flag
		if cfgBaseDir != "" {
			cfg.BaseDir = cfgBaseDir
			// Re-validate after override
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}
		}

		// Initialize entity manager
		entityManager, err := entities.NewManager(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize entity manager: %w", err)
		}

		// Initialize command context (includes project detection and entity manager)
		commandContext, err := util.NewCommandContext(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize command context: %w", err)
		}

		// Store both entity manager and command context in command context
		ctx := command.WithEntityManager(cmd.Context(), entityManager)
		ctx = command.WithCommandContext(ctx, commandContext)
		cmd.SetContext(ctx)
		return nil
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVar(&cfgBaseDir, "dir", "",
		"Base directory for ee storage (default: $EE_HOME or ~/.ee)")
	rootCmd.PersistentFlags().BoolVar(&globalFlags.debug, "debug", false, "Enable debug output")

	// Add command groups
	rootCmd.AddGroup(&cobra.Group{
		ID:    "global",
		Title: "Global Commands:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "entities",
		Title: "Entity Management:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "authenticated",
		Title: "Remote Operations:",
	})

	// Add commands organized by groups
	rootCmd.AddCommand(
		// Global Commands - basic project operations
		command.NewInitCommand("global"),   // Project initialization
		command.NewApplyCommand("global"),  // Apply environment variables
		command.NewVerifyCommand("global"), // Verify project configuration

		// Entity Management - local entity operations
		command.NewSchemaCommand("entities"), // Schema management
		command.NewSheetCommand("entities"),  // Config sheet management

		// Remote Operations - require authentication
		command.NewPushCommand("authenticated"), // Push to remote
		command.NewPullCommand("authenticated"), // Pull from remote
		// command.NewUICommand("authenticated"),     // Terminal user interface - TODO: refactor
		// command.NewRemoteCommand("authenticated"), // Remote configuration - TODO: refactor
	)

	// Enable version flag
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
