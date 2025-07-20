# menv - Environment Variable Manager with Schema Support

`menv` is a CLI tool that brings structure and validation to environment variable management. It enables teams to define, validate, and manage environment variables across different environments with schema-based validation and inheritance support.

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
curl -sSfL https://raw.githubusercontent.com/n1rna/menv/main/install.sh | sh
```

### Alternative Installation Methods

```bash
# Using Go
go install github.com/n1rna/menv/cmd/menv@latest

# Download specific version
curl -sSfL https://raw.githubusercontent.com/n1rna/menv/main/install.sh | sh -s -- --version v1.0.0

# From source
git clone https://github.com/n1rna/menv.git
cd menv
make install
```

### Manual Installation

1. Download the appropriate binary for your platform from the [releases page](https://github.com/n1rna/menv/releases)
2. Make it executable: `chmod +x menv`
3. Move it to your PATH: `sudo mv menv /usr/local/bin/`

### Verify Installation

```bash
menv version
```

## Quick Start

1. Create a new project with a schema:
```bash
menv new myproject
```

2. Create different environments:
```bash
menv new myproject --env development
menv new myproject --env production
```

3. Edit environment variables:
```bash
menv edit myproject --env development
```

4. Apply variables and run commands:
```bash
# Start a shell with the environment loaded
menv apply myproject --env development

# Run a command with the environment
menv apply myproject --env development -- npm start
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

- `menv new [project-name]` - Create a new project
- `menv new [project-name] --env [env-name]` - Add new environment
- `menv edit [project-name] --env [env-name]` - Edit environment variables
- `menv apply [project-name] --env [env-name] [-- command]` - Apply variables
- `menv export [project-name] --env [env-name] -f [format]` - Export configuration
- `menv set [project-name] --env [env-name] KEY=VALUE...` - Set variables
- `menv env [-e envfile] [-- command]` - Apply from .env file

## Configuration

`menv` stores all data in `~/.menv/` by default. You can override this with:
- `MENV_HOME` environment variable
- `--dir` flag in commands

Directory structure:
```
~/.menv/
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