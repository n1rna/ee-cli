// Package command implements the ee push command for pushing secrets to remote origins.
package command

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/origin"
	"github.com/n1rna/ee-cli/internal/output"
	"github.com/n1rna/ee-cli/internal/util"
)

// NewPushCommand creates the ee push command for pushing secrets to origins.
func NewPushCommand(groupId string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push [origin] <environment>",
		Short: "Push environment secrets to a remote origin",
		Long: `Push secrets from a local environment to a remote origin (GitHub, Cloudflare).

The environment is resolved from the .ee project file and its .env files.
The origin specifies where to push (configured in the .ee file under "origins").

Push modes:
  bundled     - Push all secrets as a single KEY=VALUE secret (default for GitHub)
  individual  - Push each secret as a separate key-value pair (default for Cloudflare)

Examples:
  # Push to the only configured origin
  ee push production

  # Push to a specific origin
  ee push github production

  # Preview what would be pushed
  ee push production --dry-run

  # Override push mode
  ee push github production --mode individual
`,
		Args:    cobra.RangeArgs(1, 2),
		RunE:    runPush,
		GroupID: groupId,
	}

	cmd.Flags().Bool("dry-run", false, "Show what would be pushed without executing")
	cmd.Flags().Bool("quiet", false, "Suppress non-error output")
	cmd.Flags().String("mode", "", "Override push mode (bundled or individual)")

	return cmd
}

func runPush(cmd *cobra.Command, args []string) error {
	quiet, _ := cmd.Flags().GetBool("quiet")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	modeOverride, _ := cmd.Flags().GetString("mode")

	printer := output.NewPrinter(output.FormatTable, quiet)

	// Require project context
	ctx, err := RequireProjectContext(cmd.Context())
	if err != nil {
		return err
	}

	pc := ctx.ProjectConfig

	// Resolve origin and environment from args
	originName, envName, err := resolveArgs(args, pc.Origins)
	if err != nil {
		return err
	}

	// Validate environment exists
	if !pc.HasEnvironment(envName) {
		available := strings.Join(pc.GetEnvironmentNames(), ", ")
		return fmt.Errorf("environment %q not found in project (available: %s)", envName, available)
	}

	// Get origin config
	originCfg, ok := pc.Origins[originName]
	if !ok {
		available := make([]string, 0, len(pc.Origins))
		for k := range pc.Origins {
			available = append(available, k)
		}
		sort.Strings(available)
		return fmt.Errorf(
			"origin %q not found in project (available: %s)",
			originName, strings.Join(available, ", "),
		)
	}

	// Determine push mode
	mode := originCfg.Mode
	if modeOverride != "" {
		mode = origin.PushMode(modeOverride)
	}
	if mode == "" {
		mode = origin.DefaultMode(originCfg.Type)
	}

	// Create origin driver and check prerequisites early (before environment
	// resolution loads .env files that could pollute the subprocess environment)
	driver, err := origin.New(originName, originCfg)
	if err != nil {
		return err
	}

	printer.Info(fmt.Sprintf("Checking %s prerequisites...", originCfg.Type))
	if err := driver.CheckPrerequisites(); err != nil {
		return err
	}

	// Resolve environment values
	envDef, err := pc.GetEnvironment(envName)
	if err != nil {
		return err
	}

	resolver := util.NewEnvResolver()
	values, err := resolver.MergeEnvironment(util.EnvironmentSources{
		Env:     envDef.Env,
		Sources: envDef.Sources,
		Sheets:  envDef.Sheets,
	})
	if err != nil {
		return fmt.Errorf("failed to resolve environment %q: %w", envName, err)
	}

	if len(values) == 0 {
		printer.Warning(fmt.Sprintf("No values found for environment %q", envName))
		return nil
	}

	// Show what we're about to do
	if dryRun {
		printer.Info(fmt.Sprintf("Dry run: would push %d secrets to %s (%s, mode: %s)",
			len(values), originName, originCfg.Type, mode))
		printSecretsSummary(printer, values, mode, originCfg)
		return nil
	}

	printer.Info(fmt.Sprintf("Pushing %d secrets to %s (%s, mode: %s)...",
		len(values), originName, originCfg.Type, mode))

	// Push
	result, err := driver.Push(envName, values, mode, false)
	if err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	// Report results
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			printer.Error(fmt.Sprintf("  %v", e))
		}
		printer.Warning(fmt.Sprintf("Pushed %d secrets with %d errors",
			result.SecretsCount, len(result.Errors)))
	} else {
		printer.Success(fmt.Sprintf("Pushed %d secrets to %s", result.SecretsCount, originName))
	}

	return nil
}

// resolveArgs resolves origin name and environment name from positional arguments.
func resolveArgs(
	args []string, origins map[string]origin.Config,
) (originName, envName string, err error) {
	if len(origins) == 0 {
		return "", "", fmt.Errorf(
			"no origins configured in .ee file. " +
				"Add an 'origins' section to your project config",
		)
	}

	if len(args) == 2 {
		return args[0], args[1], nil
	}

	// Single arg — must be environment name, auto-resolve origin
	envName = args[0]

	if len(origins) == 1 {
		for k := range origins {
			return k, envName, nil
		}
	}

	available := make([]string, 0, len(origins))
	for k := range origins {
		available = append(available, k)
	}
	sort.Strings(available)
	return "", "", fmt.Errorf(
		"multiple origins configured — specify which one: "+
			"ee push <origin> %s\navailable origins: %s",
		envName,
		strings.Join(available, ", "),
	)
}

// printSecretsSummary shows a preview of what would be pushed.
func printSecretsSummary(
	printer *output.Printer,
	values map[string]string,
	mode origin.PushMode,
	cfg origin.Config,
) {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if mode == origin.ModeBundled {
		secretName := cfg.SecretName
		if secretName == "" {
			secretName = "ENV_<ENV>"
		}
		printer.Info(fmt.Sprintf("  Secret: %s (bundled, %d variables)", secretName, len(keys)))
		for _, k := range keys {
			printer.Printf("    %s=***\n", k)
		}
	} else {
		for _, k := range keys {
			printer.Printf("  %s=***\n", k)
		}
	}
}
