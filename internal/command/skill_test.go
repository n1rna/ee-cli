package command

import (
	"strings"
	"testing"
)

func TestSkillTargetsCoverExpectedAgents(t *testing.T) {
	targets := skillTargets()
	expected := map[string]string{
		"claude":   ".claude/skills/ee-usage/SKILL.md",
		"cursor":   ".cursor/rules/ee-usage.mdc",
		"copilot":  ".github/copilot-instructions.md",
		"codex":    "AGENTS.md",
		"opencode": "AGENTS.md",
	}

	for name, wantPath := range expected {
		target, ok := targets[name]
		if !ok {
			t.Fatalf("expected agent %q to be supported", name)
		}
		// filepath.Join uses the OS separator; normalise for comparison.
		gotPath := strings.ReplaceAll(target.Path, "\\", "/")
		if gotPath != wantPath {
			t.Errorf("agent %q: expected path %q, got %q", name, wantPath, gotPath)
		}
		if target.Wrap == nil {
			t.Errorf("agent %q: Wrap function must not be nil", name)
		}
		if target.DisplayName == "" {
			t.Errorf("agent %q: DisplayName must not be empty", name)
		}
	}
}

func TestSkillBodyIsEmbedded(t *testing.T) {
	if !strings.Contains(eeUsageSkillBody, "ee") {
		t.Fatal("embedded skill body appears empty or missing")
	}
	// Sanity check that the special sections requested are present.
	for _, marker := range []string{
		"Adding `ee` to a new project",
		"Working with an existing `ee` project",
		"Command reference",
	} {
		if !strings.Contains(eeUsageSkillBody, marker) {
			t.Errorf("embedded skill body missing section %q", marker)
		}
	}
}

func TestClaudeFrontmatter(t *testing.T) {
	out := claudeFrontmatter("BODY")
	if !strings.HasPrefix(out, "---\n") {
		t.Fatal("claude output must start with frontmatter delimiter")
	}
	if !strings.Contains(out, "name: "+skillName) {
		t.Error("claude frontmatter must contain the skill name")
	}
	if !strings.Contains(out, "description: ") {
		t.Error("claude frontmatter must contain a description")
	}
	if !strings.HasSuffix(out, "BODY") {
		t.Error("claude output must end with the body")
	}
}

func TestCursorFrontmatter(t *testing.T) {
	out := cursorFrontmatter("BODY")
	for _, want := range []string{"description: ", "globs:", "alwaysApply: false"} {
		if !strings.Contains(out, want) {
			t.Errorf("cursor frontmatter missing %q", want)
		}
	}
	if !strings.Contains(out, "\nBODY") {
		t.Error("cursor output must contain the body")
	}
}

func TestNoFrontmatterIsPassthrough(t *testing.T) {
	if got := noFrontmatter("BODY"); got != "BODY" {
		t.Errorf("noFrontmatter should return body unchanged, got %q", got)
	}
}

func TestSortedAgentNamesStable(t *testing.T) {
	got := sortedAgentNames()
	want := []string{"claude", "codex", "copilot", "cursor", "opencode"}
	if len(got) != len(want) {
		t.Fatalf("expected %d agents, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sortedAgentNames[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
