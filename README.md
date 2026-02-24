# ee - Environment Variable Manager with Schema Support

`ee` is a CLI tool that brings structure and validation to environment variable management. It enables teams to define, validate, and manage environment variables across different environments with schema-based validation and inheritance support.

## Features

- Schema-based validation with type checking
- Support for multiple environments (dev, staging, prod)
- Variable inheritance between environments
- Regex pattern validation
- Default values
- Required field enforcement
- Multiple export formats (.env, JSON, YAML)
- Cross-platform support (Linux, macOS, Windows)
- Editor integration for configuration
- Comprehensive logging system

## Installation

### Quick Install (Recommended)

Install the latest version with a single command:

```bash
curl -sSfL https://raw.githubusercontent.com/n1rna/ee-cli/main/install.sh | sh
```

### Alternative Installation Methods

```bash
# Using Go
go install github.com/n1rna/ee-cli/cmd/ee@latest

# Download specific version
curl -sSfL https://raw.githubusercontent.com/n1rna/ee-cli/main/install.sh | sh -s -- --version v1.0.0

# From source
git clone https://github.com/n1rna/ee.git
cd ee
make install
```

### Manual Installation

1. Download the appropriate binary for your platform from the [releases page](https://github.com/n1rna/ee-cli/releases)
2. Make it executable: `chmod +x ee`
3. Move it to your PATH: `sudo mv ee /usr/local/bin/`

### Verify Installation

```bash
ee version
```

## Quick Start

1. Create a new project with a schema:
```bash
ee new myproject
```

2. Create different environments:
```bash
ee new myproject --env development
ee new myproject --env production
```

3. Edit environment variables:
```bash
ee edit myproject --env development
```

4. Apply variables and run commands:
```bash
# Start a shell with the environment loaded
ee apply myproject --env development

# Run a command with the environment
ee apply myproject --env development -- npm start
```

## Schema Example

```yaml
name: myproject-schema
variables:
  - name: BASE_URL
    type: string
    regex: "^https?://.*$"
    default: "http://localhost:8000"
    required: true

  - name: DEBUG
    type: boolean
    default: "false"
    required: true

  - name: API_KEY
    type: string
    required: true
```

## Commands

- `ee new [project-name]` - Create a new project
- `ee new [project-name] --env [env-name]` - Add new environment
- `ee edit [project-name] --env [env-name]` - Edit environment variables
- `ee apply [project-name] --env [env-name] [-- command]` - Apply variables
- `ee export [project-name] --env [env-name] -f [format]` - Export configuration
- `ee set [project-name] --env [env-name] KEY=VALUE...` - Set variables
- `ee env [-e envfile] [-- command]` - Apply from .env file

## Configuration

`ee` stores all data in `~/.ee/` by default. You can override this with:
- `ee_HOME` environment variable
- `--dir` flag in commands

Directory structure:
```
~/.ee/
├── schemas/           # Schema definitions
└── projects/         # Project configurations
    └── myproject/
        ├── development.yaml
        └── production.yaml
```

## Building from Source

```bash
# Install dependencies
make deps

# Build
make build

# Run tests
make test

# Install locally
make install
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

MIT License - see the [LICENSE](LICENSE) file for details