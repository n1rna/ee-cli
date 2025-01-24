// cmd/menv/main.go
package main

import (
	"fmt"
	"os"

	"github.com/n1rna/menv/internal/command"
	"github.com/n1rna/menv/internal/config"
	"github.com/n1rna/menv/internal/storage"
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
		Use:   "menv",
		Short: "menv - Environment variable manager with schema support",
		Long: `menv is a CLI tool for managing environment variables in a structured way.
It supports schema validation, multiple environments, and inheritance.`,
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Create configuration
			cfg := config.DefaultConfig()

			// Override base directory if specified
			if cfgBaseDir != "" {
				cfg.BaseDir = cfgBaseDir
			}

			// Validate configuration
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			// Initialize storage with configuration
			store, err := storage.NewStorage(cfg)
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}

			// Store in command context
			cmd.SetContext(command.WithStorage(cmd.Context(), store))
			return nil
		},
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVar(&cfgBaseDir, "dir", "", "Base directory for menv storage (default: $MENV_HOME or ~/.menv)")
	rootCmd.PersistentFlags().BoolVar(&globalFlags.debug, "debug", false, "Enable debug output")

	// Add commands
	rootCmd.AddCommand(
		command.NewNewCommand(),
		command.NewEditCommand(),
		command.NewExportCommand(),
		command.NewSetCommand(),
		command.NewApplyCommand(),
		command.NewEnvCommand(),
		command.NewListCommand(),
		command.NewSchemaCommand(),
	)

	// Enable version flag
	rootCmd.SetVersionTemplate("menv version {{.Version}}\n")

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
