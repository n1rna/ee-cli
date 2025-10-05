"""
pytest configuration and fixtures for ee-cli integration tests
"""
import os
import subprocess
import tempfile
import shutil
from pathlib import Path
import pytest


@pytest.fixture(scope="session")
def ee_binary():
    """Build and return path to ee binary"""
    project_root = Path(__file__).parent.parent
    binary_path = project_root / "build" / "ee"

    # Build the binary
    print(f"\nBuilding ee binary at {binary_path}...")
    result = subprocess.run(
        ["go", "build", "-o", str(binary_path), "./cmd/ee"],
        cwd=project_root,
        capture_output=True,
        text=True
    )

    if result.returncode != 0:
        pytest.fail(f"Failed to build ee binary:\n{result.stderr}")

    if not binary_path.exists():
        pytest.fail(f"Binary not found at {binary_path}")

    print(f"Successfully built ee binary at {binary_path}")
    yield str(binary_path)


@pytest.fixture
def temp_home(tmp_path):
    """Create a temporary home directory for ee storage"""
    ee_home = tmp_path / ".ee"
    ee_home.mkdir()
    return str(ee_home)


@pytest.fixture
def temp_project_dir(tmp_path):
    """Create a temporary project directory"""
    project_dir = tmp_path / "test-project"
    project_dir.mkdir()
    return str(project_dir)


@pytest.fixture
def ee_runner(ee_binary, temp_home):
    """Return a function to run ee commands with isolated storage"""
    def run(args, input_text=None, cwd=None, check=True):
        """
        Run ee command with given arguments

        Args:
            args: List of command arguments (e.g., ['schema', 'create', 'test'])
            input_text: Optional stdin input for interactive commands
            cwd: Working directory (default: temp directory)
            check: Whether to raise exception on non-zero exit code

        Returns:
            subprocess.CompletedProcess object
        """
        env = os.environ.copy()
        env['EE_HOME'] = temp_home

        cmd = [ee_binary] + args

        result = subprocess.run(
            cmd,
            input=input_text,
            capture_output=True,
            text=True,
            env=env,
            cwd=cwd,
            check=False
        )

        if check and result.returncode != 0:
            raise AssertionError(
                f"Command failed: {' '.join(cmd)}\n"
                f"Exit code: {result.returncode}\n"
                f"Stdout: {result.stdout}\n"
                f"Stderr: {result.stderr}"
            )

        return result

    return run


@pytest.fixture
def fixtures_dir():
    """Return path to test fixtures directory"""
    return Path(__file__).parent / "fixtures"


@pytest.fixture
def create_fixture_file(tmp_path):
    """Helper to create fixture files in temp directory"""
    def _create(filename, content):
        file_path = tmp_path / filename
        file_path.write_text(content)
        return str(file_path)

    return _create


@pytest.fixture
def generic_schema(ee_runner):
    """Create a generic schema that accepts any string variables"""
    # Create a simple generic schema for this test's isolated environment
    schema_name = "generic-test-schema"

    # Try to create the schema (might fail if already exists in this temp_home)
    result = ee_runner(
        [
            "schema", "create", schema_name,
            "--description", "Generic schema for testing",
            "--variable", "VAR1:string:Variable 1:false",
            "--variable", "VAR2:string:Variable 2:false",
            "--variable", "VAR3:string:Variable 3:false",
            "--variable", "VAR4:string:Variable 4:false",
            "--variable", "STANDALONE_VAR:string:Standalone variable:false",
        ],
        check=False
    )

    # Return schema name regardless of whether creation succeeded or failed
    # (it's ok if it already exists)
    return schema_name
