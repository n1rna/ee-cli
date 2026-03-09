"""
Integration tests for environment source merging and stacking
"""
import json
import os
from pathlib import Path
import pytest


class TestEnvFileMerging:
    """Test merging multiple .env file sources"""

    def test_merge_two_env_files(self, ee_runner, temp_project_dir):
        """Test merging two .env files with precedence"""
        # Create base .env file
        base_env = Path(temp_project_dir) / ".env.base"
        base_env.write_text(
            "VAR1=base_value1\nVAR2=base_value2\nVAR3=base_value3\n"
        )

        # Create override .env file
        override_env = Path(temp_project_dir) / ".env.override"
        override_env.write_text(
            "VAR2=override_value2\nVAR4=override_value4\n"
        )

        # Initialize project
        ee_runner(["init", "merge-project"], cwd=temp_project_dir)

        # Configure environment with multiple sources
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["development"] = {
            "sources": [".env.base", ".env.override"]
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

        # VAR2 overridden by second source
        assert merged_vars["VAR2"] == "override_value2"

        # VAR3 from base (not overridden)
        assert merged_vars["VAR3"] == "base_value3"

        # VAR4 from override
        assert merged_vars["VAR4"] == "override_value4"

    def test_merge_three_sources_with_precedence(self, ee_runner, temp_project_dir):
        """Test merging three .env files with correct precedence order"""
        for i in range(1, 4):
            env_file = Path(temp_project_dir) / f".env.layer{i}"
            env_file.write_text(
                f"SHARED=from_layer{i}\nONLY_IN_{i}=value{i}\n"
            )

        # Initialize project
        ee_runner(["init", "triple-merge"], cwd=temp_project_dir)

        # Configure stacked sources
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["staging"] = {
            "sources": [".env.layer1", ".env.layer2", ".env.layer3"]
        }

        with open(ee_file, 'w') as f:
            json.dump(config, f, indent=2)

        # Apply and verify
        result = ee_runner(
            ["apply", "staging", "--dry-run", "--format", "json"],
            cwd=temp_project_dir
        )

        merged_vars = json.loads(result.stdout)

        # Last source wins for shared variable
        assert merged_vars["SHARED"] == "from_layer3"

        # Each unique variable is included
        assert merged_vars["ONLY_IN_1"] == "value1"
        assert merged_vars["ONLY_IN_2"] == "value2"
        assert merged_vars["ONLY_IN_3"] == "value3"


class TestEnvFileReferences:
    """Test different ways of referencing .env files in environments"""

    def test_single_env_reference(self, ee_runner, temp_project_dir):
        """Test environment with single env file reference"""
        env_file = Path(temp_project_dir) / ".env.dev"
        env_file.write_text("VAR=single_value\n")

        # Initialize project
        ee_runner(["init", "single-ref-project"], cwd=temp_project_dir)

        # Configure with single env file
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["dev"] = {
            "env": ".env.dev"
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

    def test_sources_array_reference(self, ee_runner, temp_project_dir):
        """Test environment with array of source references"""
        (Path(temp_project_dir) / ".env.s1").write_text("V1=val1\n")
        (Path(temp_project_dir) / ".env.s2").write_text("V2=val2\n")

        # Initialize project
        ee_runner(["init", "array-ref-project"], cwd=temp_project_dir)

        # Configure with sources array
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["test"] = {
            "sources": [".env.s1", ".env.s2"]
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


class TestMergePriority:
    """Test merge priority and override behavior"""

    def test_later_sources_override_earlier(self, ee_runner, temp_project_dir):
        """Test that later sources in the array override earlier ones"""
        for i in range(1, 4):
            env_file = Path(temp_project_dir) / f".env.priority{i}"
            content = f"PRIORITY_VAR={'first' if i == 1 else 'second' if i == 2 else 'third'}\n"
            content += f"UNIQUE_{i}=value{i}\n"
            env_file.write_text(content)

        # Initialize project
        ee_runner(["init", "priority-test"], cwd=temp_project_dir)

        # Configure sources in specific order
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["ordered"] = {
            "sources": [".env.priority1", ".env.priority2", ".env.priority3"]
        }

        with open(ee_file, 'w') as f:
            json.dump(config, f, indent=2)

        # Apply
        result = ee_runner(
            ["apply", "ordered", "--dry-run", "--format", "json"],
            cwd=temp_project_dir
        )

        merged_vars = json.loads(result.stdout)

        # Last source should win
        assert merged_vars["PRIORITY_VAR"] == "third"

        # All unique values should be present
        assert merged_vars["UNIQUE_1"] == "value1"
        assert merged_vars["UNIQUE_2"] == "value2"
        assert merged_vars["UNIQUE_3"] == "value3"


class TestMergeErrorHandling:
    """Test error handling in source merging"""

    def test_missing_env_file_in_merge_fails(self, ee_runner, temp_project_dir):
        """Test that referencing non-existent .env file in merge fails"""
        # Create one valid .env file
        (Path(temp_project_dir) / ".env.exists").write_text("VAR=value\n")

        # Initialize project
        ee_runner(["init", "missing-in-merge"], cwd=temp_project_dir)

        # Configure with missing source
        ee_file = Path(temp_project_dir) / ".ee"
        with open(ee_file) as f:
            config = json.load(f)

        config["environments"]["broken"] = {
            "sources": [".env.exists", ".env.does-not-exist"]
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
        assert "not found" in result.stderr.lower() or \
               "does-not-exist" in result.stderr
