# ee - Environment Variable Manager with Schema Support

[![Go](https://1tt.dev/badge/Go-1.23+-00ADD8.svg?logo=go&logoColor=white)](https://go.dev/)
[![Version](https://1tt.dev/badge/version-0.10.0-blue.svg)](https://github.com/n1rna/ee-cli/releases)
[![License](https://1tt.dev/badge/license-MIT-green.svg)](https://github.com/n1rna/ee-cli/blob/main/LICENSE)
[![AI Skill](https://1tt.dev/badge/AI%20skill-ee%20skill-brightgreen.svg)](#ai-coding-agent-integration)
[![Platform](https://1tt.dev/badge/platform-linux%20%7C%20macOS%20%7C%20windows-lightgrey.svg)](https://github.com/n1rna/ee-cli/releases)

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

- `ee init [project-name]` - Initialize a new ee project (creates `.ee` + sample `.env` files)
- `ee apply <environment|file> [-- command]` - Apply an environment (or `.env` file) and run a command
- `ee verify [--fix]` - Validate the project against its schema and environment files
- `ee hydrate <environment>` - Generate an env file from the shell environment + schema defaults
- `ee push [origin] <environment>` - Push secrets to a remote origin (GitHub, Cloudflare)
- `ee auth [tool]` - Check authentication status for origin CLI tools (`gh`, `wrangler`)

> Cloudflare pushes use `wrangler`. If it isn't on your `PATH`, `ee` automatically
> runs it via `bunx wrangler` or `npx wrangler`, so a project-local install (or no
> install at all, with `bun`/`npx` fetching it on demand) works without a global setup.
- `ee skill <agent>` - Install the ee usage guide for your AI coding agent (see below)
- `ee` - Inspect/filter the current shell's environment variables

## AI Coding Agent Integration

`ee` can teach your AI coding agent how to work with environment variables in a
project by installing a usage guide (an "ee-usage" skill) into the location each
agent expects. The guide explains how to add `ee` to a new project, how to work
with a project that already has `ee` set up, the `.ee`/schema/`.env` formats, and
the full command reference.

```bash
# Install the skill for your agent of choice
ee skill claude       # -> .claude/skills/ee-usage/SKILL.md
ee skill cursor       # -> .cursor/rules/ee-usage.mdc
ee skill copilot      # -> .github/copilot-instructions.md
ee skill codex        # -> AGENTS.md
ee skill opencode     # -> AGENTS.md

# Install for every supported agent at once
ee skill all

# List supported agents, or preview the guide without writing a file
ee skill --list
ee skill claude --print
```

Use `--force` to overwrite an existing file (for example, to refresh the guide
after upgrading `ee`). Commit the generated file so your whole team's agents pick
it up.

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