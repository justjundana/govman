# Contributing to GOVMAN

Thank you for your interest in contributing to GOVMAN! This guide will help you get started with contributing to this Go version manager project.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Contributing Process](#contributing-process)
- [Code Style Guidelines](#code-style-guidelines)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)

## Getting Started

### Prerequisites

- **Go 1.25 or later**
- **Git** - [Install Git](https://git-scm.com/downloads)
- **Make** (optional) - For using Makefile commands
- **A code editor** - VS Code, GoLand, Vim, etc.

### First Contribution

1. **Star the repository** to show your support
2. **Fork the repository** to your GitHub account
3. **Clone your fork** locally
4. **Create a feature branch** for your changes
5. **Make your changes** following our guidelines
6. **Test your changes** thoroughly
7. **Submit a pull request**

## Development Setup

### Fork and Clone

```bash
# Fork the repository on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/govman.git
cd govman

# Add the original repository as upstream
git remote add upstream https://github.com/justjundana/govman.git
```

### Install Dependencies

```bash
# Download Go module dependencies
go mod download

# Verify everything works
go build -o govman ./cmd/govman
./govman --help
```

### Development Environment

```bash
# Run tests to ensure everything is working
go test ./...

# Run with race detection
go test -race ./...
```

### Optional: Development Tools

```bash
# Install useful Go tools for development
go install golang.org/x/tools/cmd/goimports@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Contributing Process

### Choose What to Work On

- **Good First Issues**: Look for issues labeled `good first issue`
- **Bug Fixes**: Issues labeled `bug` are always welcome
- **Features**: Check the roadmap or propose new features in discussions
- **Documentation**: Improvements to docs, examples, and guides
- **Tests**: Increase test coverage or add edge case tests

### Before You Start

1. **Check existing issues** to avoid duplication
2. **Create an issue** for new features to discuss the approach
3. **Comment on the issue** to let others know you're working on it
4. **Ask questions** if you need clarification

### Development Workflow

```bash
# 1. Stay up to date with upstream
git checkout main
git pull upstream main

# 2. Create a feature branch
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-number-description

# 3. Make your changes
# Edit files, add tests, update docs...

# 4. Test your changes
go test ./...
go test -race ./...
./govman --help  # Manual testing

# 5. Commit your changes
git add .
git commit -m "feat: add new feature description"

# 6. Push to your fork
git push origin feature/your-feature-name

# 7. Create a Pull Request on GitHub
```

## Code Style Guidelines

### Go Code Standards

1. **Follow Go conventions**:
   ```bash
   # Format code with gofmt (or goimports)
   gofmt -w .
   goimports -w .

   # Run go vet
   go vet ./...

   # Run staticcheck
   staticcheck ./...
   ```

2. **Naming conventions**:
   - Use `CamelCase` for exported functions/types
   - Use `camelCase` for unexported functions/variables
   - Use descriptive names: `installManager` not `im`

3. **Package structure**:
   - Keep packages focused and cohesive
   - Use the `internal/` directory for private packages
   - Avoid circular dependencies

4. **Error handling**:
   ```go
   // Good: Wrap errors with context
   if err != nil {
       return fmt.Errorf("failed to install Go %s: %w", version, err)
   }

   // Good: Use specific error types when appropriate
   if errors.Is(err, os.ErrNotExist) {
       // handle file not found
   }
   ```

5. **Documentation**:
   ```go
   // Good: Document all exported functions
   // InstallVersion downloads and installs the specified Go version.
   // It returns an error if the version is invalid or installation fails.
   func InstallVersion(version string) error {
       // ...
   }
   ```

### Code Organization

1. **File naming**:
   - Use snake_case: `version_manager.go`
   - Group related functionality: `install.go`, `uninstall.go`
   - Test files: `manager_test.go`

2. **Import grouping**:
   ```go
   import (
       // Standard library
       "fmt"
       "os"

       // Third-party packages
       cobra "github.com/spf13/cobra"

       // Internal packages
       _config "github.com/justjundana/govman/internal/config"
   )
   ```

3. **Function organization**:
   - Group related functions together
   - Put exported functions before unexported ones
   - Keep functions focused and single-purpose

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run specific package tests
go test ./internal/manager

# Run tests with verbose output
go test -v ./...

# Run benchmarks
go test -bench=. ./...
```

### Writing Tests

1. **Test file naming**: `filename_test.go`

2. **Test function naming**: `TestFunctionName`

3. **Table-driven tests**:
   ```go
   func TestInstallVersion(t *testing.T) {
       testCases := []struct {
           name          string
           version       string
           expectedError bool
       }{
           {
               name:          "Valid version",
               version:       "1.21.5",
               expectedError: false,
           },
           {
               name:          "Invalid version",
               version:       "invalid",
               expectedError: true,
           },
       }

       for _, tc := range testCases {
           t.Run(tc.name, func(t *testing.T) {
               err := InstallVersion(tc.version)
               if tc.expectedError && err == nil {
                   t.Error("Expected error but got none")
               }
               if !tc.expectedError && err != nil {
                   t.Errorf("Unexpected error: %v", err)
               }
           })
       }
   }
   ```

4. **Test helpers**:
   ```go
   func createTempDir(t *testing.T) string {
       dir, err := os.MkdirTemp("", "govman-test")
       if err != nil {
           t.Fatalf("Failed to create temp dir: %v", err)
       }
       t.Cleanup(func() {
           os.RemoveAll(dir)
       })
       return dir
   }
   ```

### Test Coverage Goals

- **New features**: Must have test coverage
- **Bug fixes**: Must include regression tests
- **Critical paths**: Aim for 90%+ coverage
- **Edge cases**: Test error conditions and boundary cases

## Documentation

### Code Documentation

1. **Function comments**: Document all exported functions
2. **Package comments**: Add package-level documentation
3. **Complex logic**: Add inline comments for clarity

### User Documentation

1. **README.md**: Keep usage examples up to date
2. **Help text**: Update command help when adding features
3. **Error messages**: Make them clear and actionable

### Examples

```bash
# Add examples to demonstrate new features
govman install latest --verbose
govman use 1.21.5 --local
```

## Issue Guidelines

### Bug Reports

When reporting bugs, please include:

1. **GOVMAN version**: `govman --version`
2. **Operating system**: `uname -a` or Windows version
3. **Go version**: `go version`
4. **Steps to reproduce**: Clear, minimal reproduction steps
5. **Expected behavior**: What should happen
6. **Actual behavior**: What actually happens
7. **Logs**: Include relevant error messages or logs

### Feature Requests

When requesting features, please include:

1. **Problem description**: What problem does this solve?
2. **Proposed solution**: How should it work?
3. **Use cases**: When would this be useful?
4. **Alternatives**: What workarounds exist?

## Pull Request Process

### Before Submitting

1. **Update your branch**:
   ```bash
   git checkout main
   git pull upstream main
   git checkout your-branch
   git rebase main
   ```

2. **Run the full test suite**:
   ```bash
   go test ./...
   go test -race ./...
   go vet ./...
   ```

3. **Update documentation** if needed

4. **Check your commits**:
   - Use conventional commit messages
   - Squash related commits if needed
   - Each commit should be logical and complete

### PR Description Template

```markdown
## Description
Brief description of changes made.

## Type of Change
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Testing
- [ ] Tests added/updated
- [ ] All tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review of changes completed
- [ ] Documentation updated (if needed)
- [ ] No new linting warnings/errors

## Related Issues
Closes #123
```

### Review Process

1. **Automated checks**: All CI checks must pass
2. **Code review**: At least one maintainer review required
3. **Testing**: New code must have appropriate tests
4. **Documentation**: User-facing changes need doc updates

## Release Process

### Versioning

We follow [Semantic Versioning](https://semver.org/):
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Checklist

1. Update version in `internal/version/version.go`
2. Update `CHANGELOG.md`
3. Create release tag: `git tag -a v1.2.3 -m "Release v1.2.3"`
4. Push tag: `git push origin v1.2.3`
5. GitHub Actions will create the release automatically

Thank you for contributing to GOVMAN! Your help makes this project better for everyone.