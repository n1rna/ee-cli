// Package command implements the ee skill command for installing the ee usage
// guide into the convention expected by a given AI coding agent.
package command

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/output"
)

// eeUsageSkillBody is the canonical, agent-oriented ee usage guide. It is the
// single source of truth shipped with the binary and installed by `ee skill`.
//
//go:embed assets/ee-usage.md
var eeUsageSkillBody string

const (
	// skillName is the slug used for the installed skill.
	skillName = "ee-usage"
	// skillDescription is a one-line summary used in agent frontmatter.
	skillDescription = "Manage environment variables with the ee CLI: .ee project files, " +
		"schemas, .env environments, hydration and pushing secrets to origins. " +
		"Use when working with environment variables, .env files, secrets, or a .ee file."
)

// agentTarget describes where and how to install the skill for a coding agent.
type agentTarget struct {
	// Name is the identifier passed on the command line (e.g. "claude").
	Name string
	// DisplayName is the human-readable agent name.
	DisplayName string
	// Path is the file path (relative to the project root) to write.
	Path string
	// Wrap turns the shared skill body into the final file contents, adding any
	// agent-specific frontmatter or heading.
	Wrap func(body string) string
}

// noFrontmatter returns the body unchanged, for agents that read plain markdown.
func noFrontmatter(body string) string {
	return body
}

// claudeFrontmatter wraps the body in a Claude skill (SKILL.md) frontmatter.
func claudeFrontmatter(body string) string {
	return fmt.Sprintf(
		"---\nname: %s\ndescription: %s\n---\n\n%s",
		skillName,
		skillDescription,
		body,
	)
}

// cursorFrontmatter wraps the body in a Cursor rule (.mdc) frontmatter.
func cursorFrontmatter(body string) string {
	return fmt.Sprintf(
		"---\ndescription: %s\nglobs:\nalwaysApply: false\n---\n\n%s",
		skillDescription,
		body,
	)
}

// skillTargets returns the supported coding agents keyed by their command-line
// name. Ordering-sensitive callers should use sortedAgentNames.
func skillTargets() map[string]agentTarget {
	return map[string]agentTarget{
		"claude": {
			Name:        "claude",
			DisplayName: "Claude Code",
			Path:        filepath.Join(".claude", "skills", skillName, "SKILL.md"),
			Wrap:        claudeFrontmatter,
		},
		"cursor": {
			Name:        "cursor",
			DisplayName: "Cursor",
			Path:        filepath.Join(".cursor", "rules", skillName+".mdc"),
			Wrap:        cursorFrontmatter,
		},
		"copilot": {
			Name:        "copilot",
			DisplayName: "GitHub Copilot",
			Path:        filepath.Join(".github", "copilot-instructions.md"),
			Wrap:        noFrontmatter,
		},
		"codex": {
			Name:        "codex",
			DisplayName: "OpenAI Codex",
			Path:        "AGENTS.md",
			Wrap:        noFrontmatter,
		},
		"opencode": {
			Name:        "opencode",
			DisplayName: "opencode",
			Path:        "AGENTS.md",
			Wrap:        noFrontmatter,
		},
	}
}

// sortedAgentNames returns the supported agent names in a stable order.
func sortedAgentNames() []string {
	targets := skillTargets()
	names := make([]string, 0, len(targets))
	for name := range targets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SkillCommand handles the ee skill command.
type SkillCommand struct{}

// NewSkillCommand creates a new ee skill command.
func NewSkillCommand(groupId string) *cobra.Command {
	sc := &SkillCommand{}

	cmd := &cobra.Command{
		Use:   "skill [agent]",
		Short: "Install the ee usage guide as a skill for your AI coding agent",
		Long: `Install the ee usage guide into the location your AI coding agent expects,
so it knows how to work with ee in this project.

Supported agents:
  claude     -> .claude/skills/ee-usage/SKILL.md
  cursor     -> .cursor/rules/ee-usage.mdc
  copilot    -> .github/copilot-instructions.md
  codex      -> AGENTS.md
  opencode   -> AGENTS.md
  all        -> install for every supported agent

Examples:
  # Install the skill for Claude Code
  ee skill claude

  # Install for Cursor, overwriting an existing rule
  ee skill cursor --force

  # Install for every supported agent
  ee skill all

  # List supported agents
  ee skill --list

  # Print the guide to stdout instead of writing a file
  ee skill claude --print
`,
		RunE:    sc.Run,
		GroupID: groupId,
	}

	cmd.Flags().BoolP("force", "f", false, "Overwrite an existing skill file")
	cmd.Flags().BoolP("list", "l", false, "List supported coding agents and exit")
	cmd.Flags().
		BoolP("print", "p", false, "Print the skill contents to stdout instead of writing a file")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress non-error output")

	return cmd
}

// Run executes the skill command.
func (c *SkillCommand) Run(cmd *cobra.Command, args []string) error {
	quiet, _ := cmd.Flags().GetBool("quiet")
	force, _ := cmd.Flags().GetBool("force")
	list, _ := cmd.Flags().GetBool("list")
	printOnly, _ := cmd.Flags().GetBool("print")

	printer := output.NewPrinter(output.FormatTable, quiet)

	if list {
		c.printAgentList(printer)
		return nil
	}

	if len(args) == 0 {
		c.printAgentList(printer)
		return fmt.Errorf("no agent specified (choose one of: %s, all)",
			strings.Join(sortedAgentNames(), ", "))
	}

	agent := strings.ToLower(strings.TrimSpace(args[0]))

	// Resolve the set of targets to install.
	var targets []agentTarget
	if agent == "all" {
		for _, name := range sortedAgentNames() {
			t := skillTargets()[name]
			targets = append(targets, t)
		}
	} else {
		t, ok := skillTargets()[agent]
		if !ok {
			return fmt.Errorf("unknown agent %q (supported: %s, all)",
				agent, strings.Join(sortedAgentNames(), ", "))
		}
		targets = []agentTarget{t}
	}

	// Print mode: write to stdout and stop.
	if printOnly {
		// When multiple agents are selected the body is identical, so print once
		// using the first target's wrapper.
		printer.Printf("%s", targets[0].Wrap(eeUsageSkillBody))
		return nil
	}

	written := make(map[string]bool)
	for _, target := range targets {
		// Deduplicate targets that share a path (e.g. codex and opencode both
		// use AGENTS.md) so "all" doesn't fail on the second write.
		if written[target.Path] {
			continue
		}

		if err := c.installTarget(target, force, printer); err != nil {
			return err
		}
		written[target.Path] = true
	}

	if !quiet {
		printer.Info("\nThe ee usage guide is now available to your coding agent.")
		printer.Info("Re-run with --force after upgrading ee to refresh it.")
	}

	return nil
}

// installTarget writes the skill file for a single agent target.
func (c *SkillCommand) installTarget(
	target agentTarget,
	force bool,
	printer *output.Printer,
) error {
	if _, err := os.Stat(target.Path); err == nil && !force {
		return fmt.Errorf("%s already exists (use --force to overwrite)", target.Path)
	}

	if dir := filepath.Dir(target.Path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	content := target.Wrap(eeUsageSkillBody)
	if err := os.WriteFile(target.Path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", target.Path, err)
	}

	printer.Success(
		fmt.Sprintf("Installed %s skill -> %s", target.DisplayName, target.Path),
	)
	return nil
}

// printAgentList prints the supported agents and their install paths.
func (c *SkillCommand) printAgentList(printer *output.Printer) {
	printer.Info("Supported coding agents:")
	targets := skillTargets()
	for _, name := range sortedAgentNames() {
		t := targets[name]
		printer.Printf("  %-10s %-16s -> %s\n", t.Name, "("+t.DisplayName+")", t.Path)
	}
	printer.Printf("  %-10s %-16s -> %s\n", "all", "(every agent)", "all of the above")
}
