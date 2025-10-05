"""
Integration tests for schema creation and management
"""
import json
import pytest


class TestSchemaCreation:
    """Test schema creation using different methods"""

    def test_create_schema_from_yaml_file(self, ee_runner, fixtures_dir):
        """Test creating a schema from a YAML file"""
        schema_file = fixtures_dir / "schema-web-service.yaml"

        # Create schema from YAML file
        result = ee_runner([
            "schema", "create", "web-service",
            "--import", str(schema_file)
        ])

        assert result.returncode == 0
        assert "Successfully" in result.stdout
        assert "web-service" in result.stdout
        assert "4 variables" in result.stdout

        # Verify schema was created
        result = ee_runner(["schema", "show", "web-service", "--format", "json"])
        assert result.returncode == 0

        schema_data = json.loads(result.stdout)
        assert schema_data["name"] == "web-service"
        assert len(schema_data["variables"]) == 4
        assert schema_data["description"] == "Schema for web service applications"

        # Verify variable details
        var_names = [v["name"] for v in schema_data["variables"]]
        assert "DATABASE_URL" in var_names
        assert "PORT" in var_names
        assert "DEBUG" in var_names
        assert "API_KEY" in var_names

    def test_create_schema_from_json_file(self, ee_runner, fixtures_dir):
        """Test creating a schema from a JSON file"""
        schema_file = fixtures_dir / "schema-api.json"

        # Create schema from JSON file
        result = ee_runner([
            "schema", "create", "api-service",
            "--import", str(schema_file)
        ])

        assert result.returncode == 0
        assert "Successfully" in result.stdout
        assert "api-service" in result.stdout

        # Verify schema
        result = ee_runner(["schema", "show", "api-service", "--format", "json"])
        schema_data = json.loads(result.stdout)

        assert schema_data["name"] == "api-service"
        assert schema_data["description"] == "Schema for API services"
        assert len(schema_data["variables"]) == 3

    def test_create_schema_from_annotated_env_file(self, ee_runner, fixtures_dir):
        """Test creating a schema from an annotated .env file"""
        schema_file = fixtures_dir / "schema-annotated.env"

        # Create schema from annotated .env file
        result = ee_runner([
            "schema", "create", "db-service",
            "--import", str(schema_file)
        ])

        assert result.returncode == 0
        assert "Successfully" in result.stdout

        # Verify schema extracted from annotations
        result = ee_runner(["schema", "show", "db-service", "--format", "json"])
        schema_data = json.loads(result.stdout)

        assert schema_data["name"] == "db-service"
        assert len(schema_data["variables"]) == 3

        # Check that annotations were parsed correctly
        db_url_var = next(v for v in schema_data["variables"] if v["name"] == "DATABASE_URL")
        assert db_url_var["type"] == "string"
        assert db_url_var["required"] == True
        assert db_url_var["title"] == "Database URL"

    def test_create_schema_with_cli_variables(self, ee_runner):
        """Test creating a schema using CLI variable specifications"""
        result = ee_runner([
            "schema", "create", "cli-schema",
            "--description", "Schema created via CLI",
            "--variable", "APP_NAME:string:Application Name:true",
            "--variable", "MAX_CONNECTIONS:number:Max Connections:false:100",
            "--variable", "ENABLE_CACHE:boolean:Enable Caching:false:false"
        ])

        assert result.returncode == 0
        assert "Successfully" in result.stdout
        assert "3 variables" in result.stdout

        # Verify schema
        result = ee_runner(["schema", "show", "cli-schema", "--format", "json"])
        schema_data = json.loads(result.stdout)

        assert schema_data["name"] == "cli-schema"
        assert schema_data["description"] == "Schema created via CLI"
        assert len(schema_data["variables"]) == 3

        # Verify variable properties
        app_name = next(v for v in schema_data["variables"] if v["name"] == "APP_NAME")
        assert app_name["type"] == "string"
        assert app_name["required"] == True

        max_conn = next(v for v in schema_data["variables"] if v["name"] == "MAX_CONNECTIONS")
        assert max_conn["type"] == "number"
        assert max_conn["default"] == "100"

    @pytest.mark.xfail(reason="Interactive mode EOF handling needs improvement")
    def test_create_schema_interactively(self, ee_runner):
        """Test creating a schema interactively"""
        # Simulate interactive input
        interactive_input = "\n".join([
            "SERVICE_URL",        # Variable name
            "string",             # Type
            "",                   # Regex (skip)
            "http://localhost",   # Default value
            "y",                  # Required
            "TIMEOUT",            # Variable name
            "number",             # Type
            "",                   # Regex (skip)
            "30",                 # Default value
            "n",                  # Not required
            "",                   # Empty name to finish
        ])

        result = ee_runner(
            ["schema", "create", "interactive-schema"],
            input_text=interactive_input
        )

        assert result.returncode == 0
        assert "Successfully" in result.stdout

        # Verify created schema
        result = ee_runner(["schema", "show", "interactive-schema", "--format", "json"])
        schema_data = json.loads(result.stdout)

        assert len(schema_data["variables"]) == 2
        assert schema_data["variables"][0]["name"] == "SERVICE_URL"
        assert schema_data["variables"][0]["required"] == True
        assert schema_data["variables"][1]["name"] == "TIMEOUT"
        assert schema_data["variables"][1]["required"] == False


class TestSchemaManagement:
    """Test schema listing and deletion"""

    def test_list_schemas(self, ee_runner, fixtures_dir):
        """Test listing all schemas"""
        # Create multiple schemas
        schema_file1 = fixtures_dir / "schema-web-service.yaml"
        schema_file2 = fixtures_dir / "schema-api.json"

        ee_runner(["schema", "create", "schema1", "--import", str(schema_file1)])
        ee_runner(["schema", "create", "schema2", "--import", str(schema_file2)])

        # List schemas
        result = ee_runner(["schema", "list", "--format", "json"])
        assert result.returncode == 0

        schemas = json.loads(result.stdout)
        assert len(schemas) >= 2

        schema_names = [s["name"] for s in schemas]
        assert "schema1" in schema_names
        assert "schema2" in schema_names

    def test_delete_schema(self, ee_runner, fixtures_dir):
        """Test deleting a schema"""
        schema_file = fixtures_dir / "schema-web-service.yaml"

        # Create schema
        ee_runner(["schema", "create", "temp-schema", "--import", str(schema_file)])

        # Verify it exists
        result = ee_runner(["schema", "show", "temp-schema"])
        assert result.returncode == 0

        # Delete schema
        result = ee_runner(["schema", "delete", "temp-schema", "--quiet"])
        assert result.returncode == 0

        # Verify it's deleted
        result = ee_runner(["schema", "show", "temp-schema"], check=False)
        assert result.returncode != 0


class TestSchemaValidation:
    """Test schema validation and error handling"""

    def test_duplicate_schema_name_fails(self, ee_runner):
        """Test that creating a schema with duplicate name fails"""
        # Create first schema
        result = ee_runner([
            "schema", "create", "duplicate-test",
            "--variable", "VAR1:string:Variable 1:true"
        ])
        assert result.returncode == 0

        # Try to create schema with same name
        result = ee_runner(
            [
                "schema", "create", "duplicate-test",
                "--variable", "VAR2:string:Variable 2:true"
            ],
            check=False
        )
        assert result.returncode != 0
        assert "already exists" in result.stderr.lower()

    def test_invalid_variable_spec_fails(self, ee_runner):
        """Test that invalid variable specification fails"""
        result = ee_runner(
            [
                "schema", "create", "invalid-schema",
                "--variable", "INVALID_SPEC"  # Missing required fields
            ],
            check=False
        )
        assert result.returncode != 0

    @pytest.mark.xfail(reason="Dotenv parser accepts any file, validation needs improvement")
    def test_invalid_file_format_fails(self, ee_runner, create_fixture_file):
        """Test that invalid file format fails gracefully"""
        invalid_file = create_fixture_file("invalid.txt", "This is not valid YAML or JSON")

        result = ee_runner(
            ["schema", "create", "invalid-file-schema", "--import", invalid_file],
            check=False
        )
        assert result.returncode != 0
        assert "neither valid" in result.stderr.lower() or "failed to parse" in result.stderr.lower()
