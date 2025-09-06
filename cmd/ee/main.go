// ee is a CLI tool for managing environment variables with schema-based validation.
package main

import (
	"fmt"
	"os"

	"github.com/n1rna/ee-cli/internal/command"
	"github.com/n1rna/ee-cli/internal/config"
	"github.com/n1rna/ee-cli/internal/storage"
	"github.com/spf13/cobra"
)

var (
	version     = "dev"
	cfgBaseDir  string
	globalFlags = struct {
		debug bool
	}{}
)

func main() {
	// Create root command
	rootCmd := &cobra.Command{
		Use:   "ee",
		Short: "ee - Environment variable manager with schema support",
		Long: `ee is a CLI tool for managing environment variables in a structured way.
It supports schema validation, multiple environments, and inheritance.`,
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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

			// Initialize UUID-based storage with configuration
			store, err := storage.NewUUIDStorage(cfg)
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}

			// Store in command context
			cmd.SetContext(command.WithStorage(cmd.Context(), store))
			return nil
		},
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

	// Create commands and set group IDs
	initCmd := command.NewInitCommand()
	initCmd.GroupID = "global"

	applyCmd := command.NewApplyCommand()
	applyCmd.GroupID = "global"

	uiCmd := command.NewUICommand()
	uiCmd.GroupID = "global"

	schemaCmd := command.NewSchemaCommand()
	schemaCmd.GroupID = "entities"

	sheetCmd := command.NewSheetCommand()
	sheetCmd.GroupID = "entities"

	projectCmd := command.NewProjectCommand()
	projectCmd.GroupID = "entities"

	pushCmd := command.NewPushCommand()
	pushCmd.GroupID = "authenticated"

	pullCmd := command.NewPullCommand()
	pullCmd.GroupID = "authenticated"

	remoteCmd := command.NewRemoteCommand()
	remoteCmd.GroupID = "authenticated"

	// Add commands organized by groups
	rootCmd.AddCommand(
		// Global Commands - basic project operations
		initCmd,  // Project initialization
		applyCmd, // Apply environment variables
		uiCmd,    // Terminal user interface

		// Entity Management - local entity operations
		schemaCmd,  // Schema management
		sheetCmd,   // Config sheet management
		projectCmd, // Project management

		// Remote Operations - require authentication
		pushCmd,   // Push to remote
		pullCmd,   // Pull from remote
		remoteCmd, // Remote configuration
	)

	// Enable version flag
	rootCmd.SetVersionTemplate("ee version {{.Version}}\n")

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
