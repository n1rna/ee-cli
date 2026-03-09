// Package command implements the ee auth command for checking origin authentication status.
package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/origin"
	"github.com/n1rna/ee-cli/internal/output"
)

// NewAuthCommand creates the ee auth command for checking authentication status.
func NewAuthCommand(groupId string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth [tool]",
		Short: "Check authentication status for origin tools",
		Long: `Check if the CLI tools required by your configured origins are authenticated.

Without arguments, checks all tools used by origins in your .ee config.
With a tool argument, checks only that specific tool.

Examples:
  # Check all configured origin tools
  ee auth

  # Check GitHub CLI auth
  ee auth gh

  # Check Cloudflare Wrangler auth
  ee auth wrangler
`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runAuth,
		GroupID: groupId,
	}

	cmd.Flags().BoolP("quiet", "q", false, "Suppress non-error output")

	return cmd
}

func runAuth(cmd *cobra.Command, args []string) error {
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.FormatTable, quiet)

	// If specific tool requested, check just that one
	if len(args) == 1 {
		return checkTool(printer, args[0])
	}

	// Try to get project context for configured origins
	ctx := GetCommandContext(cmd.Context())
	if ctx != nil && ctx.IsInProject && ctx.ProjectConfig != nil &&
		len(ctx.ProjectConfig.Origins) > 0 {
		// Check tools for configured origins
		tools := map[string]bool{}
		for _, cfg := range ctx.ProjectConfig.Origins {
			switch cfg.Type {
			case "github":
				tools["gh"] = true
			case "cloudflare":
				tools["wrangler"] = true
			}
		}

		hasError := false
		for tool := range tools {
			if err := checkTool(printer, tool); err != nil {
				hasError = true
			}
		}
		if hasError {
			return fmt.Errorf("some tools are not authenticated")
		}
		return nil
	}

	// No project context — check both common tools
	printer.Info("No .ee project found, checking common tools...")
	_ = checkTool(printer, "gh")
	_ = checkTool(printer, "wrangler")
	return nil
}

func checkTool(printer *output.Printer, tool string) error {
	if err := origin.CheckTool(tool); err != nil {
		printer.Error(fmt.Sprintf("%s: not installed", tool))
		return err
	}

	var authCmd []string
	switch tool {
	case "gh":
		authCmd = []string{"auth", "status"}
	case "wrangler":
		authCmd = []string{"whoami"}
	default:
		printer.Warning(fmt.Sprintf("%s: unknown tool, cannot check auth", tool))
		return nil
	}

	out, err := origin.RunCommand(tool, authCmd...)
	if err != nil {
		printer.Error(fmt.Sprintf("%s: not authenticated\n%s", tool, string(out)))
		return fmt.Errorf("%s is not authenticated", tool)
	}

	printer.Success(fmt.Sprintf("%s: authenticated", tool))
	return nil
}
