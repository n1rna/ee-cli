"""
Integration tests for config sheet creation and management
"""
import json
import pytest


class TestSheetCreation:
    """Test config sheet creation using different methods"""

    def test_create_sheet_from_yaml_file(self, ee_runner, fixtures_dir, generic_schema):
        """Test creating a config sheet from a YAML file"""
        config_file = fixtures_dir / "config-dev.yaml"

        # Create config sheet from YAML file
        result = ee_runner([
            "sheet", "create", "dev-config",
            "--import", str(config_file),
            "--schema", generic_schema,
            "--description", "Development configuration"
        ])

        assert result.returncode == 0
        assert "Successfully" in result.stdout
        assert "dev-config" in result.stdout

        # Verify sheet was created
        result = ee_runner(["sheet", "show", "dev-config", "--format", "json"])
        assert result.returncode == 0

        sheet_data = json.loads(result.stdout)
        assert sheet_data["name"] == "dev-config"
        assert sheet_data["description"] == "Development configuration"
        assert "DATABASE_URL" in sheet_data["values"]
        assert sheet_data["values"]["PORT"] == "3000"
        assert sheet_data["values"]["DEBUG"] == "true"

    def test_create_sheet_from_json_file(self, ee_runner, fixtures_dir, generic_schema):
        """Test creating a config sheet from a JSON file"""
        config_file = fixtures_dir / "config-prod.json"

        # Create config sheet from JSON file
        result = ee_runner([
            "sheet", "create", "prod-config",
            "--import", str(config_file),
            "--schema", generic_schema
        ])

        assert result.returncode == 0

        # Verify sheet
        result = ee_runner(["sheet", "show", "prod-config", "--format", "json"])
        sheet_data = json.loads(result.stdout)

        assert sheet_data["name"] == "prod-config"
        assert sheet_data["values"]["DATABASE_URL"] == "postgres://prod-db:5432/prod_db"
        assert sheet_data["values"]["DEBUG"] == "false"

    def test_create_sheet_from_env_file(self, ee_runner, fixtures_dir, generic_schema):
        """Test creating a config sheet from a .env file"""
        config_file = fixtures_dir / "config-base.env"

        # Create config sheet from .env file
        result = ee_runner([
            "sheet", "create", "base-config",
            "--import", str(config_file),
            "--schema", generic_schema
        ])

        assert result.returncode == 0

        # Verify sheet
        result = ee_runner(["sheet", "show", "base-config", "--format", "json"])
        sheet_data = json.loads(result.stdout)

        assert sheet_data["values"]["DATABASE_URL"] == "postgres://localhost:5432/base_db"
        assert sheet_data["values"]["PORT"] == "8000"
        assert sheet_data["values"]["DEBUG"] == "false"

    def test_create_sheet_with_cli_values(self, ee_runner, generic_schema):
        """Test creating a config sheet using CLI values"""
        result = ee_runner([
            "sheet", "create", "cli-config",
            "--schema", generic_schema,
            "--description", "CLI created config",
            "--value", "APP_NAME=MyApp",
            "--value", "VERSION=1.0.0",
            "--value", "ENVIRONMENT=production"
        ])

        assert result.returncode == 0
        assert "Successfully" in result.stdout

        # Verify sheet
        result = ee_runner(["sheet", "show", "cli-config", "--format", "json"])
        sheet_data = json.loads(result.stdout)

        assert sheet_data["name"] == "cli-config"
        assert sheet_data["values"]["APP_NAME"] == "MyApp"
        assert sheet_data["values"]["VERSION"] == "1.0.0"
        assert sheet_data["values"]["ENVIRONMENT"] == "production"

    def test_create_sheet_with_file_and_cli_override(self, ee_runner, fixtures_dir, generic_schema):
        """Test creating a config sheet from file with CLI value overrides"""
        config_file = fixtures_dir / "config-dev.yaml"

        # Import file and override some values via CLI
        result = ee_runner([
            "sheet", "create", "override-config",
            "--import", str(config_file),
            "--schema", generic_schema,
            "--value", "PORT=9000",  # Override PORT from file
            "--value", "NEW_VAR=new_value"  # Add new variable
        ])

        assert result.returncode == 0

        # Verify CLI values took precedence
        result = ee_runner(["sheet", "show", "override-config", "--format", "json"])
        sheet_data = json.loads(result.stdout)

        assert sheet_data["values"]["PORT"] == "9000"  # Overridden
        assert sheet_data["values"]["DEBUG"] == "true"  # From file
        assert sheet_data["values"]["NEW_VAR"] == "new_value"  # Added

    @pytest.mark.xfail(reason="Interactive mode EOF handling needs improvement")
    def test_create_sheet_interactively_freeform(self, ee_runner):
        """Test creating a config sheet interactively without schema"""
        # Simulate interactive input (free-form mode)
        interactive_input = "\n".join([
            "API_URL",                    # Variable name
            "https://api.example.com",    # Value
            "TIMEOUT",                    # Variable name
            "30",                         # Value
            "RETRY_COUNT",                # Variable name
            "3",                          # Value
            "",                           # Empty name to finish
        ])

        result = ee_runner(
            ["sheet", "create", "interactive-config", "--interactive"],
            input_text=interactive_input
        )

        assert result.returncode == 0
        assert "Successfully" in result.stdout

        # Verify created sheet
        result = ee_runner(["sheet", "show", "interactive-config", "--format", "json"])
        sheet_data = json.loads(result.stdout)

        assert len(sheet_data["values"]) == 3
        assert sheet_data["values"]["API_URL"] == "https://api.example.com"
        assert sheet_data["values"]["TIMEOUT"] == "30"
        assert sheet_data["values"]["RETRY_COUNT"] == "3"

    @pytest.mark.xfail(reason="Interactive mode EOF handling needs improvement")
    def test_create_sheet_interactively_with_schema(self, ee_runner, fixtures_dir):
        """Test creating a config sheet interactively with schema guidance"""
        # First create a schema
        schema_file = fixtures_dir / "schema-web-service.yaml"
        ee_runner(["schema", "create", "guided-schema", "--import", str(schema_file)])

        # Create sheet interactively with schema
        # Schema has: DATABASE_URL (required), PORT, DEBUG, API_KEY (required)
        interactive_input = "\n".join([
            "postgres://localhost:5432/testdb",  # DATABASE_URL
            "5000",                              # PORT
            "true",                              # DEBUG
            "test-api-key",                      # API_KEY
        ])

        result = ee_runner(
            [
                "sheet", "create", "guided-config",
                "--schema", "guided-schema",
                "--interactive"
            ],
            input_text=interactive_input
        )

        assert result.returncode == 0

        # Verify sheet
        result = ee_runner(["sheet", "show", "guided-config", "--format", "json"])
        sheet_data = json.loads(result.stdout)

        assert sheet_data["values"]["DATABASE_URL"] == "postgres://localhost:5432/testdb"
        assert sheet_data["values"]["PORT"] == "5000"
        assert sheet_data["values"]["API_KEY"] == "test-api-key"


class TestSheetWithSchema:
    """Test config sheet creation and validation with schemas"""

    def test_create_sheet_with_schema_validation(self, ee_runner, fixtures_dir):
        """Test that sheet validates against schema"""
        # Create schema
        schema_file = fixtures_dir / "schema-web-service.yaml"
        ee_runner(["schema", "create", "validation-schema", "--import", str(schema_file)])

        # Create sheet with valid values
        result = ee_runner([
            "sheet", "create", "validated-config",
            "--schema", "validation-schema",
            "--value", "DATABASE_URL=postgres://localhost:5432/db",
            "--value", "API_KEY=valid-key",
            "--value", "PORT=8080"
        ])

        assert result.returncode == 0

        # Verify schema reference
        result = ee_runner(["sheet", "show", "validated-config", "--format", "json"])
        sheet_data = json.loads(result.stdout)
        assert "schema" in sheet_data and "ref" in sheet_data["schema"]


class TestSheetManagement:
    """Test config sheet management operations"""

    def test_list_sheets(self, ee_runner, fixtures_dir, generic_schema):
        """Test listing all config sheets"""
        # Create multiple sheets
        config1 = fixtures_dir / "config-dev.yaml"
        config2 = fixtures_dir / "config-prod.json"

        ee_runner(["sheet", "create", "sheet1", "--import", str(config1), "--schema", generic_schema])
        ee_runner(["sheet", "create", "sheet2", "--import", str(config2), "--schema", generic_schema])

        # List sheets
        result = ee_runner(["sheet", "list", "--format", "json"])
        assert result.returncode == 0

        sheets = json.loads(result.stdout)
        assert len(sheets) >= 2

        sheet_names = [s["name"] for s in sheets]
        assert "sheet1" in sheet_names
        assert "sheet2" in sheet_names

    def test_delete_sheet(self, ee_runner, generic_schema):
        """Test deleting a config sheet"""
        # Create sheet
        ee_runner([
            "sheet", "create", "temp-sheet",
            "--schema", generic_schema,
            "--value", "KEY=value"
        ])

        # Verify it exists
        result = ee_runner(["sheet", "show", "temp-sheet"])
        assert result.returncode == 0

        # Delete sheet
        result = ee_runner(["sheet", "delete", "temp-sheet", "--quiet"])
        assert result.returncode == 0

        # Verify it's deleted
        result = ee_runner(["sheet", "show", "temp-sheet"], check=False)
        assert result.returncode != 0

    def test_export_sheet_formats(self, ee_runner, generic_schema):
        """Test exporting sheet in different formats"""
        # Create sheet
        ee_runner([
            "sheet", "create", "export-test",
            "--schema", generic_schema,
            "--value", "VAR1=value1",
            "--value", "VAR2=value2"
        ])

        # Export as dotenv
        result = ee_runner(["sheet", "export", "export-test", "--format", "dotenv"])
        assert result.returncode == 0
        assert "VAR1" in result.stdout and "value1" in result.stdout
        assert "VAR2" in result.stdout and "value2" in result.stdout

        # Export as JSON
        result = ee_runner(["sheet", "export", "export-test", "--format", "json"])
        assert result.returncode == 0
        export_data = json.loads(result.stdout)
        assert export_data["VAR1"] == "value1"

        # Export as env (bash export format)
        result = ee_runner(["sheet", "export", "export-test", "--format", "env"])
        assert result.returncode == 0
        assert "export VAR1=" in result.stdout or "VAR1=" in result.stdout


class TestSheetValueManagement:
    """Test setting and unsetting values in sheets"""

    def test_set_value_in_sheet(self, ee_runner, generic_schema):
        """Test setting a value in an existing sheet"""
        # Create sheet
        ee_runner([
            "sheet", "create", "update-test",
            "--schema", generic_schema,
            "--value", "ORIGINAL=value"
        ])

        # Set new value
        result = ee_runner([
            "sheet", "set", "update-test",
            "NEW_VAR", "new_value"
        ])
        assert result.returncode == 0

        # Verify value was set
        result = ee_runner(["sheet", "show", "update-test", "--format", "json"])
        sheet_data = json.loads(result.stdout)
        assert sheet_data["values"]["NEW_VAR"] == "new_value"
        assert sheet_data["values"]["ORIGINAL"] == "value"

    def test_unset_value_in_sheet(self, ee_runner, generic_schema):
        """Test unsetting a value from a sheet"""
        # Create sheet
        ee_runner([
            "sheet", "create", "unset-test",
            "--schema", generic_schema,
            "--value", "VAR1=value1",
            "--value", "VAR2=value2"
        ])

        # Unset value
        result = ee_runner([
            "sheet", "unset", "unset-test", "VAR1"
        ])
        assert result.returncode == 0

        # Verify value was removed
        result = ee_runner(["sheet", "show", "unset-test", "--format", "json"])
        sheet_data = json.loads(result.stdout)
        assert "VAR1" not in sheet_data["values"]
        assert sheet_data["values"]["VAR2"] == "value2"


class TestSheetErrorHandling:
    """Test error handling in sheet operations"""

    def test_create_sheet_without_values_fails(self, ee_runner):
        """Test that creating sheet without values fails"""
        result = ee_runner(
            ["sheet", "create", "empty-sheet"],
            check=False
        )
        assert result.returncode != 0
        assert "must provide values" in result.stderr.lower()

    def test_create_sheet_with_nonexistent_schema_fails(self, ee_runner):
        """Test that referencing non-existent schema fails"""
        result = ee_runner(
            [
                "sheet", "create", "bad-schema-sheet",
                "--schema", "nonexistent-schema",
                "--value", "KEY=value"
            ],
            check=False
        )
        assert result.returncode != 0
        assert "not found" in result.stderr.lower()
