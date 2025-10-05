# ee-cli Integration Tests

This directory contains comprehensive integration tests for the ee-cli project using Python, pytest, and uv.

## Overview

The test suite validates the following functionality:
- **Schema Management**: Creating schemas via YAML, JSON, dotenv files, CLI flags, and interactive mode
- **Config Sheet Management**: Creating config sheets via various methods including interactive mode
- **Project Initialization**: Creating projects with schemas and environments
- **Config Sheet Merging**: Testing merge behavior with multiple stacked config sheets
- **Project Verification**: Validating project configurations

## Test Structure

```
tests/
├── conftest.py              # Pytest configuration and fixtures
├── fixtures/                # Test fixture files (YAML, JSON, .env)
├── test_schema.py          # Schema creation and management tests
├── test_sheet.py           # Config sheet creation and management tests
├── test_project.py         # Project initialization and verification tests
├── test_merge.py           # Config sheet merging tests
├── pyproject.toml          # Project configuration and dependencies
└── uv.lock                 # Dependency lock file
```

## Prerequisites

- [uv](https://docs.astral.sh/uv/) - Fast Python package installer
- Go 1.19+ (for building the ee binary)

## Running Tests

### Quick Start

Run all tests using make:

```bash
make test-integration
```

Or directly with uv:

```bash
cd tests && uv run pytest
```

### Run Specific Test Files

```bash
cd tests && uv run pytest test_schema.py
cd tests && uv run pytest test_merge.py
```

### Run Specific Test Classes or Functions

```bash
cd tests && uv run pytest test_schema.py::TestSchemaCreation
cd tests && uv run pytest test_sheet.py::TestSheetCreation::test_create_sheet_from_yaml_file
```

### Run Tests in Parallel

```bash
make test-parallel

# Or directly
cd tests && uv run pytest -n auto
```

### Run with Verbose Output

```bash
cd tests && uv run pytest -vv
```

### Run Tests Matching a Pattern

```bash
cd tests && uv run pytest -k "interactive"
cd tests && uv run pytest -k "merge"
```

## Setup

uv automatically manages the virtual environment and dependencies. No manual setup required!

```bash
# Sync dependencies (usually automatic)
cd tests && uv sync

# Add new test dependency
cd tests && uv add --dev package-name
```

## Test Fixtures

The `fixtures/` directory contains sample files used by tests:

- **schema-web-service.yaml**: YAML schema definition
- **schema-api.json**: JSON schema definition
- **schema-annotated.env**: Annotated .env file with schema
- **config-dev.yaml**: YAML config values
- **config-prod.json**: JSON config values
- **config-base.env**: .env config values

## Key Fixtures

The test suite provides several pytest fixtures:

- `ee_binary`: Builds and provides path to ee binary
- `temp_home`: Temporary isolated EE_HOME directory
- `temp_project_dir`: Temporary project directory
- `ee_runner`: Function to run ee commands with isolated storage
- `fixtures_dir`: Path to test fixtures
- `create_fixture_file`: Helper to create fixture files

## Test Coverage

### Schema Tests (`test_schema.py`)
- ✅ Create schema from YAML file
- ✅ Create schema from JSON file
- ✅ Create schema from annotated .env file
- ✅ Create schema with CLI variables
- ✅ Create schema interactively
- ✅ List and delete schemas
- ✅ Error handling and validation

### Config Sheet Tests (`test_sheet.py`)
- ✅ Create sheet from YAML, JSON, .env files
- ✅ Create sheet with CLI values
- ✅ Create sheet with file + CLI override
- ✅ Create sheet interactively (free-form and schema-guided)
- ✅ Sheet validation with schemas
- ✅ Set/unset values
- ✅ Export in multiple formats
- ✅ Error handling

### Project Tests (`test_project.py`)
- ✅ Initialize basic project
- ✅ Initialize with schema
- ✅ Initialize with remote URL
- ✅ Initialize with inline schema
- ✅ Project environment detection
- ✅ Project verification
- ✅ Apply project environments
- ✅ Apply standalone sheets
- ✅ Apply .env files directly

### Merge Tests (`test_merge.py`)
- ✅ Merge two sheets with precedence
- ✅ Merge three sheets
- ✅ Merge sheets from different formats
- ✅ Single vs array sheet references
- ✅ Merge with schema validation
- ✅ Override priority (later sheets win)
- ✅ Empty value overrides
- ✅ Error handling for missing sheets

## Writing New Tests

To add new tests:

1. Create a new test file or add to existing ones
2. Use the provided fixtures for setup
3. Follow the existing test patterns
4. Use descriptive test names
5. Include both success and error cases

Example:

```python
def test_my_feature(ee_runner, temp_project_dir):
    """Test description"""
    result = ee_runner([
        "command", "subcommand", "arg"
    ], cwd=temp_project_dir)

    assert result.returncode == 0
    assert "expected output" in result.stdout
```

## CI/CD Integration

The test suite is designed to be easily integrated into CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
- name: Run Integration Tests
  run: |
    chmod +x tests/run_tests.sh
    ./tests/run_tests.sh
```

## Troubleshooting

### Tests Fail to Build Binary

Ensure Go is installed and `go build` works in the project root:
```bash
go build -o build/ee ./cmd/ee
```

### Import Errors

Ensure pytest is installed in the virtual environment:
```bash
source tests/.venv/bin/activate
pip install -r tests/requirements.txt
```

### Isolation Issues

Each test runs with an isolated `EE_HOME` directory. If you see cross-test contamination, check that fixtures are properly scoped.

## Contributing

When adding new features to ee-cli:

1. Add corresponding integration tests
2. Include test fixtures if needed
3. Update this README if adding new test categories
4. Ensure all tests pass before submitting PR
