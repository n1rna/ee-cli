// Package command implements the ee root command for displaying environment variables
package command

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/output"
	"github.com/n1rna/ee-cli/internal/util"
)

// RootCommand handles the root ee command functionality
type RootCommand struct{}

// NewRootCommand creates a new root command with environment variable display functionality
func NewRootCommand() *cobra.Command {
	rc := &RootCommand{}

	cmd := &cobra.Command{
		Use:   "ee",
		Short: "ee - Environment variable manager with schema support",
		Long: `ee is a CLI tool for managing environment variables in a structured way.
It supports schema validation, multiple environments, and inheritance.

When run without subcommands, displays all environment variables in the current shell.`,
		RunE:         rc.Run,
		SilenceUsage: true, // Don't show usage on RunE errors
	}

	// Add flags for filtering and formatting
	cmd.Flags().StringP("filter", "I", "",
		"Filter environment variables using wildcard patterns separated by comma, pipe, or slash "+
			"(e.g., 'PATH*,USER*', '*_URL|*_KEY', '!CLAUDE*/NODE*')")
	cmd.Flags().StringP("format", "f", "env", "Output format (env, json, dotenv)")
	cmd.Flags().BoolP("mask", "m", false, "Mask sensitive environment variable values")

	return cmd
}

// Run executes the root command to display environment variables
func (c *RootCommand) Run(cmd *cobra.Command, args []string) error {
	// Get flags
	filter, _ := cmd.Flags().GetString("filter")
	format, _ := cmd.Flags().GetString("format")
	mask, _ := cmd.Flags().GetBool("mask")

	// Get all environment variables
	envVars := os.Environ()

	// Parse into key-value map
	envMap := make(map[string]string)
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			// Apply filter if specified
			if filter != "" {
				patterns := c.parsePatterns(filter)
				matched, err := c.matchesAnyPattern(key, patterns)
				if err != nil {
					return fmt.Errorf("invalid filter pattern '%s': %w", filter, err)
				}
				if !matched {
					continue
				}
			}

			// Apply masking if requested
			if mask {
				value = util.MaskSensitiveValue(key, value)
			}

			envMap[key] = value
		}
	}

	// Create printer and output the environment variables
	printer := output.NewPrinter(output.Format(format), false)

	switch format {
	case "env":
		return c.printEnvFormat(envMap)
	case "json":
		return printer.PrintValues(envMap)
	case "dotenv":
		return printer.PrintDotEnv(envMap)
	default:
		return fmt.Errorf("unsupported format: %s (supported: env, json, dotenv)", format)
	}
}

// parsePatterns splits the filter string into individual patterns
// Supports comma (,), pipe (|), and forward slash (/) as separators
func (c *RootCommand) parsePatterns(filter string) []string {
	// Use regex to split on comma, pipe, or forward slash
	re := regexp.MustCompile(`[,|/]`)
	patterns := re.Split(filter, -1)

	// Trim whitespace from each pattern
	for i, pattern := range patterns {
		patterns[i] = strings.TrimSpace(pattern)
	}

	return patterns
}

// matchesAnyPattern checks if a key matches any of the provided patterns
// Returns true if the key should be included based on the patterns
func (c *RootCommand) matchesAnyPattern(key string, patterns []string) (bool, error) {
	positiveMatches := false
	negativeMatches := false
	hasPositivePatterns := false

	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}

		// Check if pattern is negated
		negate := false
		actualPattern := pattern
		if strings.HasPrefix(pattern, "!") {
			negate = true
			actualPattern = pattern[1:] // Remove the exclamation mark
		} else {
			hasPositivePatterns = true
		}

		// Skip empty patterns after removing negation
		if actualPattern == "" {
			continue
		}

		matched, err := filepath.Match(actualPattern, key)
		if err != nil {
			return false, err
		}

		if negate {
			if matched {
				negativeMatches = true
			}
		} else {
			if matched {
				positiveMatches = true
			}
		}
	}

	// Logic for inclusion:
	// 1. If there are negative patterns that match, exclude the key
	// 2. If there are positive patterns, include only if at least one matches
	// 3. If there are only negative patterns, include if none match

	if negativeMatches {
		return false, nil // Exclude if any negative pattern matches
	}

	if hasPositivePatterns {
		return positiveMatches, nil // Include only if positive pattern matches
	}

	// If only negative patterns and none matched, include
	return true, nil
}

// printEnvFormat prints environment variables in standard env format (KEY=VALUE)
func (c *RootCommand) printEnvFormat(envMap map[string]string) error {
	// Sort keys for consistent output
	keys := make([]string, 0, len(envMap))
	for key := range envMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Print in KEY=VALUE format
	printer := output.NewPrinter(output.FormatTable, false)
	for _, key := range keys {
		printer.Printf("%s=%s\n", key, envMap[key])
	}

	return nil
}
