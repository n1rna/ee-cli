# Contributing to menv

Thank you for your interest in contributing to menv! This document provides guidelines for contributing to the project.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. Please be respectful and professional in all interactions.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/yourusername/menv.git
   cd menv
   ```
3. Install dependencies:
   ```bash
   make deps
   ```
4. Make sure tests pass:
   ```bash
   make test
   ```

## Development Workflow

### Setting Up Your Environment

1. Ensure you have Go 1.23+ installed
2. Install golangci-lint for linting:
   ```bash
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```
3. Run the verification suite:
   ```bash
   make verify
   ```

### Making Changes

1. Create a new branch for your feature or fix:
   ```bash
   git checkout -b feature/your-feature-name
   ```
2. Make your changes
3. Add tests for any new functionality
4. Ensure all tests pass:
   ```bash
   make test
   ```
5. Format your code:
   ```bash
   make fmt
   ```
6. Run linting:
   ```bash
   make lint
   ```

### Commit Messages

Follow conventional commit format:
- `feat: add new feature`
- `fix: resolve issue with X`
- `docs: update documentation`
- `test: add tests for Y`
- `refactor: improve code structure`

### Pull Request Process

1. Update documentation if needed
2. Add tests for new functionality
3. Ensure all CI checks pass
4. Fill out the pull request template completely
5. Request review from maintainers

## Testing Guidelines

### Writing Tests

- Write tests for all new functionality
- Use table-driven tests where appropriate
- Include both positive and negative test cases
- Test error conditions and edge cases

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run specific test
go test -v ./internal/schema -run TestValidateSchema
```

### Test Structure

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected ExpectedType
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    validInput,
            expected: expectedOutput,
            wantErr:  false,
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Write clear, descriptive variable and function names
- Add comments for exported functions and complex logic
- Keep functions focused and small

## Documentation

- Update README.md for user-facing changes
- Update CLAUDE.md for architectural changes
- Add inline comments for complex code
- Include examples in documentation

## Issue Guidelines

### Bug Reports

- Use the bug report template
- Include reproduction steps
- Provide environment information
- Include relevant configuration files

### Feature Requests

- Use the feature request template
- Explain the use case clearly
- Consider alternative approaches
- Provide examples of desired usage

## Architecture Guidelines

### Adding New Commands

1. Create command file in `internal/command/`
2. Follow existing command patterns
3. Add to main.go command registration
4. Include comprehensive tests
5. Update documentation

### Schema Changes

- Maintain backward compatibility when possible
- Update validation logic
- Add migration path if needed
- Document breaking changes

### Storage Changes

- Consider data migration requirements
- Maintain file format compatibility
- Test with existing data structures

## Release Process

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create pull request for release preparation
4. After merge, create and push git tag
5. GitHub Actions will handle the release

## Getting Help

- Join discussions in GitHub Issues
- Check existing documentation
- Look at similar implementations in the codebase
- Ask questions in pull requests

## Recognition

Contributors will be acknowledged in:
- Release notes
- README.md contributors section
- Git commit history

Thank you for contributing to menv!