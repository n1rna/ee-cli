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
            args: List of command arguments (e.g., ['init', 'my-project'])
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
