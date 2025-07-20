# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`menv` is a CLI tool for managing environment variables with schema-based validation. It's written in Go using the Cobra CLI framework and provides structured environment variable management across different environments with inheritance support.

## Build and Development Commands

### Basic Commands
- `make build` - Build the binary to `build/menv`
- `make test` - Run all tests
- `make clean` - Clean build artifacts
- `make install` - Build and install to `$GOPATH/bin`
- `make dev` - Run the application in development mode

### Code Quality
- `make fmt` - Format Go code using `go fmt`
- `make lint` - Run golangci-lint (requires installation)
- `make vet` - Run `go vet` for static analysis
- `make verify` - Run all verification steps (fmt, vet, lint, test)

### Testing
- `make test` - Run all tests with verbose output
- `make coverage` - Generate HTML coverage report in `coverage/`

### Cross-platform Builds
- `make build-linux` - Build for Linux AMD64
- `make build-windows` - Build for Windows AMD64  
- `make build-darwin` - Build for macOS AMD64
- `make build-all` - Build for all platforms

## Architecture

### Core Components

1. **CLI Layer** (`cmd/menv/main.go`)
   - Entry point using Cobra command framework
   - Global flags: `--dir` (storage location), `--debug`
   - Commands are defined in `internal/command/`

2. **Configuration** (`internal/config/`)
   - Manages global settings and storage locations
   - Default storage: `~/.menv/` or `$MENV_HOME`
   - Creates `schemas/` and `projects/` subdirectories

3. **Schema System** (`internal/schema/`)
   - **Schema**: Defines variable types, validation rules, inheritance
   - **ConfigSheet**: Environment-specific variable values
   - **Validator**: Handles schema validation and inheritance resolution
   - Supports schema inheritance via `extends` field
   - Variable types: string, number, boolean, url
   - Regex validation support

4. **Storage Layer** (`internal/storage/`)
   - File-based storage using YAML format
   - Manages schemas and project configurations
   - Directory structure:
     ```
     ~/.menv/
     ├── schemas/           # Schema definitions
     └── projects/         # Project configurations
         └── projectname/
             ├── development.yaml
             └── production.yaml
     ```

5. **Commands** (`internal/command/`)
   - `apply` - Load environment variables and run commands
   - `create` - Create new projects and environments
   - `edit` - Edit environment configurations
   - `export` - Export to different formats (.env, JSON, YAML)
   - `set` - Set individual variables
   - `env` - Apply from .env files
   - `list` - List projects and environments
   - `schema` - Manage schemas

### Key Features

1. **Schema Validation**: Variables are validated against type definitions with regex patterns
2. **Inheritance**: Both schemas and config sheets support inheritance via `extends`
3. **Environment Management**: Multiple environments per project (dev, staging, prod)
4. **Export Formats**: Support for .env, JSON, and YAML export formats
5. **Shell Integration**: Can start new shells or run commands with applied environment

### Dependencies
- `github.com/spf13/cobra` - CLI framework
- `gopkg.in/yaml.v3` - YAML parsing
- Go 1.23.3+ required

## Development Notes

- The project uses Go modules (`go.mod`)
- Tests should be run before commits (`make test`)
- Code formatting is enforced (`make fmt`)
- Linting requires `golangci-lint` installation
- All storage operations go through the `storage.Storage` interface
- Schema resolution handles circular dependency detection
- Environment variables are validated on load and can have default values