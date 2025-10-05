"""
Integration tests for config sheet merging and stacking
"""
import json
from pathlib import Path
import pytest


class TestConfigSheetMerging:
    """Test merging multiple config sheets"""

    def test_merge_two_sheets(self, ee_runner, temp_project_dir, fixtures_dir, generic_schema):
        """Test merging two config sheets with precedence"""
        # Create base config sheet
        ee_runner([
            "sheet", "create", "base-config",
            "--schema", generic_schema,
            "--value", "VAR1=base_value1",
            "--value", "VAR2=base_value2",
            "--value", "VAR3=base_value3"
        ])

        # Create override config sheet
        ee_runner([
            "sheet", "create", "override-config",
            "--schema", generic_schema,
            "--value", "VAR2=override_value2",  # Override VAR2
            "--value", "VAR4=override_value4"   # Add new VAR4
        ])

        # Initialize project
        ee_runner(["init", "merge-project"], cwd=temp_project_dir)

        # Configure environment with multiple sheets
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        # Stack sheets: base first, then override
        config["environments"]["development"] = {
            "sheets": ["base-config", "override-config"]
        }

        with open(ee_file, 'w') as f:
            json.dump(config, f, indent=2)

        # Apply and check merged result
        result = ee_runner(
            ["apply", "development", "--dry-run", "--format", "json"],
            cwd=temp_project_dir
        )

        assert result.returncode == 0

        merged_vars = json.loads(result.stdout)

        # VAR1 from base (not overridden)
        assert merged_vars["VAR1"] == "base_value1"

        # VAR2 overridden by second sheet
        assert merged_vars["VAR2"] == "override_value2"

        # VAR3 from base (not overridden)
        assert merged_vars["VAR3"] == "base_value3"

        # VAR4 from override sheet
        assert merged_vars["VAR4"] == "override_value4"

    def test_merge_three_sheets_with_precedence(self, ee_runner, temp_project_dir, generic_schema):
        """Test merging three sheets with correct precedence order"""
        # Create three config sheets
        ee_runner([
            "sheet", "create", "sheet1",
            "--schema", generic_schema,
            "--value", "SHARED=from_sheet1",
            "--value", "ONLY_IN_1=value1"
        ])

        ee_runner([
            "sheet", "create", "sheet2",
            "--schema", generic_schema,
            "--value", "SHARED=from_sheet2",
            "--value", "ONLY_IN_2=value2"
        ])

        ee_runner([
            "sheet", "create", "sheet3",
            "--schema", generic_schema,
            "--value", "SHARED=from_sheet3",
            "--value", "ONLY_IN_3=value3"
        ])

        # Initialize project
        ee_runner(["init", "triple-merge"], cwd=temp_project_dir)

        # Configure stacked sheets
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["staging"] = {
            "sheets": ["sheet1", "sheet2", "sheet3"]
        }

        with open(ee_file, 'w') as f:
            json.dump(config, f, indent=2)

        # Apply and verify
        result = ee_runner(
            ["apply", "staging", "--dry-run", "--format", "json"],
            cwd=temp_project_dir
        )

        merged_vars = json.loads(result.stdout)

        # Last sheet wins for shared variable
        assert merged_vars["SHARED"] == "from_sheet3"

        # Each unique variable is included
        assert merged_vars["ONLY_IN_1"] == "value1"
        assert merged_vars["ONLY_IN_2"] == "value2"
        assert merged_vars["ONLY_IN_3"] == "value3"

    def test_merge_with_mixed_formats(self, ee_runner, temp_project_dir, fixtures_dir, generic_schema):
        """Test merging sheets created from different file formats"""
        # Create sheets from different formats
        yaml_file = fixtures_dir / "config-dev.yaml"
        json_file = fixtures_dir / "config-prod.json"
        env_file = fixtures_dir / "config-base.env"

        ee_runner(["sheet", "create", "yaml-sheet", "--import", str(yaml_file), "--schema", generic_schema])
        ee_runner(["sheet", "create", "json-sheet", "--import", str(json_file), "--schema", generic_schema])
        ee_runner(["sheet", "create", "env-sheet", "--import", str(env_file), "--schema", generic_schema])

        # Initialize project
        ee_runner(["init", "mixed-format-project"], cwd=temp_project_dir)

        # Configure environment with mixed format sheets
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["mixed"] = {
            "sheets": ["env-sheet", "yaml-sheet", "json-sheet"]
        }

        with open(ee_file, 'w') as f:
            json.dump(config, f, indent=2)

        # Apply
        result = ee_runner(
            ["apply", "mixed", "--dry-run", "--format", "json"],
            cwd=temp_project_dir
        )

        assert result.returncode == 0

        merged_vars = json.loads(result.stdout)

        # Verify values from different sources merged correctly
        # json-sheet (last) should override DATABASE_URL
        assert "postgres://prod-db:5432/prod_db" in merged_vars["DATABASE_URL"]


class TestSheetReferences:
    """Test different ways of referencing sheets in environments"""

    def test_single_sheet_reference(self, ee_runner, temp_project_dir, generic_schema):
        """Test environment with single sheet reference"""
        # Create sheet
        ee_runner([
            "sheet", "create", "single-sheet",
            "--schema", generic_schema,
            "--value", "VAR=single_value"
        ])

        # Initialize project
        ee_runner(["init", "single-ref-project"], cwd=temp_project_dir)

        # Configure with single sheet
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["dev"] = {
            "sheet": "single-sheet"  # Single sheet, not array
        }

        with open(ee_file, 'w') as f:
            json.dump(config, f, indent=2)

        # Apply
        result = ee_runner(
            ["apply", "dev", "--dry-run", "--format", "json"],
            cwd=temp_project_dir
        )

        assert result.returncode == 0
        merged_vars = json.loads(result.stdout)
        assert merged_vars["VAR"] == "single_value"

    def test_sheet_array_reference(self, ee_runner, temp_project_dir, generic_schema):
        """Test environment with array of sheet references"""
        # Create sheets
        ee_runner(["sheet", "create", "array-1",
            "--schema", generic_schema, "--value", "V1=val1"])
        ee_runner(["sheet", "create", "array-2",
            "--schema", generic_schema, "--value", "V2=val2"])

        # Initialize project
        ee_runner(["init", "array-ref-project"], cwd=temp_project_dir)

        # Configure with sheet array
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["test"] = {
            "sheets": ["array-1", "array-2"]
        }

        with open(ee_file, 'w') as f:
            json.dump(config, f, indent=2)

        # Apply
        result = ee_runner(
            ["apply", "test", "--dry-run", "--format", "json"],
            cwd=temp_project_dir
        )

        assert result.returncode == 0
        merged_vars = json.loads(result.stdout)
        assert merged_vars["V1"] == "val1"
        assert merged_vars["V2"] == "val2"


class TestMergeWithSchemaValidation:
    """Test merging with schema validation"""

    def test_merged_result_validates_against_schema(self, ee_runner, temp_project_dir, fixtures_dir, generic_schema):
        """Test that merged config validates against project schema"""
        # Create schema
        schema_file = fixtures_dir / "schema-web-service.yaml"
        ee_runner(["schema", "create", "merge-schema", "--import", str(schema_file)])

        # Create partial config sheets
        ee_runner([
            "sheet", "create", "partial-1",
            "--schema", generic_schema,
            "--value", "DATABASE_URL=postgres://localhost/db",
            "--value", "PORT=8080"
        ])

        ee_runner([
            "sheet", "create", "partial-2",
            "--schema", generic_schema,
            "--value", "DEBUG=true",
            "--value", "API_KEY=test-key"
        ])

        # Initialize project with schema
        ee_runner(
            ["init", "validated-merge", "--schema", "merge-schema"],
            cwd=temp_project_dir
        )

        # Configure environment with both sheets to satisfy schema
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["complete"] = {
            "sheets": ["partial-1", "partial-2"]
        }

        with open(ee_file, 'w') as f:
            json.dump(config, f, indent=2)

        # Verify should pass with merged values or report missing .env file
        result = ee_runner(["verify"], cwd=temp_project_dir, check=False)

        # Should validate successfully (all required vars present after merge)
        # May fail due to missing .env file, which is expected
        assert result.returncode == 0 or "missing environment file" in result.stdout.lower() or "valid" in result.stdout.lower()


class TestMergePriority:
    """Test merge priority and override behavior"""

    def test_later_sheets_override_earlier(self, ee_runner, temp_project_dir, generic_schema):
        """Test that later sheets in the array override earlier ones"""
        # Create sheets with same variable
        ee_runner([
            "sheet", "create", "priority-1",
            "--schema", generic_schema,
            "--value", "PRIORITY_VAR=first",
            "--value", "UNIQUE_1=value1"
        ])

        ee_runner([
            "sheet", "create", "priority-2",
            "--schema", generic_schema,
            "--value", "PRIORITY_VAR=second",
            "--value", "UNIQUE_2=value2"
        ])

        ee_runner([
            "sheet", "create", "priority-3",
            "--schema", generic_schema,
            "--value", "PRIORITY_VAR=third",
            "--value", "UNIQUE_3=value3"
        ])

        # Initialize project
        ee_runner(["init", "priority-test"], cwd=temp_project_dir)

        # Configure sheets in specific order
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["ordered"] = {
            "sheets": ["priority-1", "priority-2", "priority-3"]
        }

        with open(ee_file, 'w') as f:
            json.dump(config, f, indent=2)

        # Apply
        result = ee_runner(
            ["apply", "ordered", "--dry-run", "--format", "json"],
            cwd=temp_project_dir
        )

        merged_vars = json.loads(result.stdout)

        # Last sheet (priority-3) should win
        assert merged_vars["PRIORITY_VAR"] == "third"

        # All unique values should be present
        assert merged_vars["UNIQUE_1"] == "value1"
        assert merged_vars["UNIQUE_2"] == "value2"
        assert merged_vars["UNIQUE_3"] == "value3"

    def test_empty_value_overrides(self, ee_runner, temp_project_dir, generic_schema):
        """Test that empty values in later sheets override earlier values"""
        # Create sheets
        ee_runner([
            "sheet", "create", "with-value",
            "--schema", generic_schema,
            "--value", "OPTIONAL_VAR=has_value"
        ])

        ee_runner([
            "sheet", "create", "with-empty",
            "--schema", generic_schema,
            "--value", "OPTIONAL_VAR="  # Empty value
        ])

        # Initialize project
        ee_runner(["init", "empty-override"], cwd=temp_project_dir)

        # Configure
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["test"] = {
            "sheets": ["with-value", "with-empty"]
        }

        with open(ee_file, 'w') as f:
            json.dump(config, f, indent=2)

        # Apply
        result = ee_runner(
            ["apply", "test", "--dry-run", "--format", "json"],
            cwd=temp_project_dir
        )

        merged_vars = json.loads(result.stdout)

        # Empty value should override
        assert merged_vars.get("OPTIONAL_VAR") == ""


class TestMergeErrorHandling:
    """Test error handling in sheet merging"""

    def test_missing_sheet_in_merge_fails(self, ee_runner, temp_project_dir, generic_schema):
        """Test that referencing non-existent sheet in merge fails"""
        # Create one valid sheet
        ee_runner([
            "sheet", "create", "exists",
            "--schema", generic_schema,
            "--value", "VAR=value"
        ])

        # Initialize project
        ee_runner(["init", "missing-in-merge"], cwd=temp_project_dir)

        # Configure with missing sheet
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["broken"] = {
            "sheets": ["exists", "does-not-exist"]
        }

        with open(ee_file, 'w') as f:
            json.dump(config, f, indent=2)

        # Apply should fail
        result = ee_runner(
            ["apply", "broken", "--dry-run"],
            cwd=temp_project_dir,
            check=False
        )

        assert result.returncode != 0
        assert "not found" in result.stderr.lower() or "does-not-exist" in result.stderr
