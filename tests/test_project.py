"""
Integration tests for project initialization and management
"""
import json
import os
from pathlib import Path
import pytest


class TestProjectInit:
    """Test project initialization"""

    def test_init_basic_project(self, ee_runner, temp_project_dir):
        """Test initializing a basic project"""
        result = ee_runner(
            ["init", "test-project"],
            cwd=temp_project_dir
        )

        assert result.returncode == 0
        assert "Initialized ee project" in result.stdout
        assert "test-project" in result.stdout

        # Verify .ee file was created
        ee_file = Path(temp_project_dir) / ".ee"
        assert ee_file.exists()

        # Verify .ee file content
        with open(ee_file) as f:
            project_config = json.load(f)

        assert project_config["project"] == "test-project"
        assert "environments" in project_config
        assert "development" in project_config["environments"]

    def test_init_project_with_schema(self, ee_runner, temp_project_dir, fixtures_dir):
        """Test initializing a project with a schema"""
        # First create a schema
        schema_file = fixtures_dir / "schema-web-service.yaml"
        ee_runner(["schema", "create", "web-schema", "--import", str(schema_file)])

        # Initialize project with schema
        result = ee_runner(
            ["init", "web-project", "--schema", "web-schema"],
            cwd=temp_project_dir
        )

        assert result.returncode == 0

        # Verify project config
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            project_config = json.load(f)

        assert project_config["project"] == "web-project"
        assert "schema" in project_config

    def test_init_project_with_inline_schema(self, ee_runner, temp_project_dir):
        """Test initializing a project with inline schema variables"""
        result = ee_runner(
            [
                "init", "inline-project",
                "--var", "DATABASE_URL:string:Database URL:true",
                "--var", "PORT:number:Server Port:false:8080"
            ],
            cwd=temp_project_dir
        )

        assert result.returncode == 0

        # Verify inline schema
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            project_config = json.load(f)

        assert "schema" in project_config
        assert "variables" in project_config["schema"]
        assert len(project_config["schema"]["variables"]) == 2

    def test_init_creates_sample_env_files(self, ee_runner, temp_project_dir):
        """Test that init creates sample .env files for environments"""
        result = ee_runner(
            ["init", "env-project"],
            cwd=temp_project_dir
        )

        assert result.returncode == 0

        # Check for .env files
        project_path = Path(temp_project_dir)
        assert (project_path / ".env.development").exists() or \
               (project_path / "development.env").exists()


class TestProjectEnvironments:
    """Test project environment management"""

    def test_project_environment_detection(self, ee_runner, temp_project_dir, fixtures_dir):
        """Test that project environments are detected from .ee file"""
        # Initialize project
        ee_runner(["init", "env-test"], cwd=temp_project_dir)

        # Modify .ee file to add custom environments
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["staging"] = {"env": ".env.staging"}
        config["environments"]["production"] = {"env": ".env.production"}

        with open(ee_file, 'w') as f:
            json.dump(config, f, indent=2)

        # Verify command recognizes project context
        result = ee_runner(["verify"], cwd=temp_project_dir, check=False)
        # Command should recognize we're in a project (even if verification fails)
        assert result.returncode != 0
        assert "issue" in result.stdout.lower() or "staging" in result.stdout


class TestProjectVerify:
    """Test project verification"""

    def test_verify_valid_project(self, ee_runner, temp_project_dir, fixtures_dir, generic_schema):
        """Test verifying a valid project configuration"""
        # Create schema
        schema_file = fixtures_dir / "schema-web-service.yaml"
        ee_runner(["schema", "create", "verify-schema", "--import", str(schema_file)])

        # Initialize project
        ee_runner(["init", "verify-project", "--schema", "verify-schema"], cwd=temp_project_dir)

        # Create .env files for environments with required variables
        config_dev = fixtures_dir / "config-dev.yaml"
        with open(config_dev) as f:
            import yaml
            dev_vars = yaml.safe_load(f)

        dev_env = Path(temp_project_dir) / ".env.development"
        with open(dev_env, 'w') as f:
            for k, v in dev_vars.items():
                f.write(f"{k}={v}\n")

        # Verify project
        result = ee_runner(["verify"], cwd=temp_project_dir, check=False)

        # Project should be verifiable
        assert result.returncode == 0 or "project" in result.stdout.lower()

    def test_verify_project_with_missing_env_files(self, ee_runner, temp_project_dir):
        """Test verifying a project with missing .env files"""
        # Initialize project
        ee_runner(["init", "missing-env-project"], cwd=temp_project_dir)

        # Update .ee to reference non-existent .env file
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["test"] = {"env": ".env.nonexistent"}

        with open(ee_file, 'w') as f:
            json.dump(config, f, indent=2)

        # Verify should report issues
        result = ee_runner(["verify"], cwd=temp_project_dir, check=False)

        # Should fail or report errors
        assert result.returncode != 0 or "not found" in result.stderr.lower() or \
               "missing" in result.stdout.lower()


class TestProjectApply:
    """Test applying project environments"""

    def test_apply_project_environment(self, ee_runner, temp_project_dir, fixtures_dir):
        """Test applying a project environment"""
        # Initialize project
        ee_runner(["init", "apply-project"], cwd=temp_project_dir)

        # Create .env file for development
        dev_env = Path(temp_project_dir) / ".env.development"
        dev_env.write_text("DATABASE_URL=postgres://localhost/dev\nPORT=3000\n")

        # Update .ee to use our .env file
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["development"]["env"] = ".env.development"

        with open(ee_file, 'w') as f:
            json.dump(config, f, indent=2)

        # Apply environment with dry-run to see what would be applied
        result = ee_runner(
            ["apply", "development", "--dry-run", "--format", "json"],
            cwd=temp_project_dir
        )

        assert result.returncode == 0

        # Parse output
        env_vars = json.loads(result.stdout)
        assert "DATABASE_URL" in env_vars
        assert env_vars["PORT"] == "3000"

    def test_apply_env_file_directly(self, ee_runner, temp_project_dir, fixtures_dir):
        """Test applying a .env file directly"""
        env_file = fixtures_dir / "config-base.env"

        # Apply .env file directly
        result = ee_runner(
            ["apply", str(env_file), "--dry-run", "--format", "json"],
            cwd=temp_project_dir
        )

        assert result.returncode == 0

        env_vars = json.loads(result.stdout)
        assert "DATABASE_URL" in env_vars
        assert env_vars["PORT"] == "8000"


class TestProjectWithoutContext:
    """Test commands that require project context"""

    def test_verify_without_project_fails(self, ee_runner, temp_project_dir):
        """Test that verify fails when not in a project directory"""
        result = ee_runner(
            ["verify"],
            cwd=temp_project_dir,
            check=False
        )

        assert result.returncode != 0
        assert ".ee" in result.stderr or "project" in result.stderr.lower()

    def test_apply_environment_without_project_fails(self, ee_runner, temp_project_dir):
        """Test that applying environment fails without project context"""
        result = ee_runner(
            ["apply", "development", "--dry-run"],
            cwd=temp_project_dir,
            check=False
        )

        assert result.returncode != 0
        assert ".ee" in result.stderr or "project" in result.stderr.lower()
