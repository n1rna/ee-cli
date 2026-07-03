"""
Integration tests for the `ee skill` command that installs the ee usage guide
into the location expected by each supported AI coding agent.
"""
from pathlib import Path
import pytest


AGENT_PATHS = {
    "claude": ".claude/skills/ee-usage/SKILL.md",
    "cursor": ".cursor/rules/ee-usage.mdc",
    "copilot": ".github/copilot-instructions.md",
    "codex": "AGENTS.md",
    "opencode": "AGENTS.md",
}


class TestSkillInstall:
    """Test installing the skill for individual agents."""

    @pytest.mark.parametrize("agent,rel_path", sorted(AGENT_PATHS.items()))
    def test_install_single_agent(self, ee_runner, temp_project_dir, agent, rel_path):
        result = ee_runner(["skill", agent], cwd=temp_project_dir)

        assert result.returncode == 0
        assert "Installed" in result.stdout

        target = Path(temp_project_dir) / rel_path
        assert target.exists(), f"expected {rel_path} to be created"

        content = target.read_text()
        # Every install contains the shared body.
        assert "Working with the `ee` environment variable manager" in content
        assert "Adding `ee` to a new project" in content
        assert "Working with an existing `ee` project" in content

    def test_claude_has_skill_frontmatter(self, ee_runner, temp_project_dir):
        ee_runner(["skill", "claude"], cwd=temp_project_dir)
        content = (Path(temp_project_dir) / AGENT_PATHS["claude"]).read_text()
        assert content.startswith("---\n")
        assert "name: ee-usage" in content
        assert "description:" in content

    def test_cursor_has_rule_frontmatter(self, ee_runner, temp_project_dir):
        ee_runner(["skill", "cursor"], cwd=temp_project_dir)
        content = (Path(temp_project_dir) / AGENT_PATHS["cursor"]).read_text()
        assert content.startswith("---\n")
        assert "alwaysApply: false" in content
        assert "globs:" in content

    def test_copilot_is_plain_markdown(self, ee_runner, temp_project_dir):
        ee_runner(["skill", "copilot"], cwd=temp_project_dir)
        content = (Path(temp_project_dir) / AGENT_PATHS["copilot"]).read_text()
        # No YAML frontmatter for plain-markdown agents.
        assert not content.startswith("---\n")
        assert content.lstrip().startswith("# Working with the `ee`")


class TestSkillInstallAll:
    """Test installing the skill for every agent at once."""

    def test_install_all(self, ee_runner, temp_project_dir):
        result = ee_runner(["skill", "all"], cwd=temp_project_dir)
        assert result.returncode == 0

        for rel_path in set(AGENT_PATHS.values()):
            assert (Path(temp_project_dir) / rel_path).exists(), rel_path


class TestSkillFlags:
    """Test skill command flags and error handling."""

    def test_list_agents(self, ee_runner, temp_project_dir):
        result = ee_runner(["skill", "--list"], cwd=temp_project_dir)
        assert result.returncode == 0
        for agent in AGENT_PATHS:
            assert agent in result.stdout
        # Listing must not create files.
        assert not (Path(temp_project_dir) / "AGENTS.md").exists()

    def test_print_does_not_write(self, ee_runner, temp_project_dir):
        result = ee_runner(["skill", "claude", "--print"], cwd=temp_project_dir)
        assert result.returncode == 0
        assert "name: ee-usage" in result.stdout
        assert not (Path(temp_project_dir) / AGENT_PATHS["claude"]).exists()

    def test_unknown_agent_fails(self, ee_runner, temp_project_dir):
        result = ee_runner(["skill", "notanagent"], cwd=temp_project_dir, check=False)
        assert result.returncode != 0
        assert "unknown agent" in result.stderr.lower()

    def test_no_agent_fails(self, ee_runner, temp_project_dir):
        result = ee_runner(["skill"], cwd=temp_project_dir, check=False)
        assert result.returncode != 0
        assert "no agent specified" in result.stderr.lower()

    def test_existing_file_requires_force(self, ee_runner, temp_project_dir):
        ee_runner(["skill", "claude"], cwd=temp_project_dir)

        # Second run without --force should fail.
        result = ee_runner(["skill", "claude"], cwd=temp_project_dir, check=False)
        assert result.returncode != 0
        assert "already exists" in result.stderr.lower()

        # With --force it should succeed.
        result = ee_runner(["skill", "claude", "--force"], cwd=temp_project_dir)
        assert result.returncode == 0
