# Developer Getting Started

Guide for developers who want to contribute to govman or build it from source.

## Prerequisites

- **Go 1.25 or later**
- **Git**
- **Make** (Linux/macOS) or equivalent build tool
- Basic understanding of Go programming

## Quick Start

```bash
# Clone the repository
git clone https://github.com/justjundana/govman.git
cd govman

# Build
make build

# Run
./govman --help

# Install to ~/.govman/bin
make install
```

## Repository Structure

```
govman/
├── cmd/
│   └── govman/
│       └── main.go           # Entry point
├── internal/
│   ├── cli/                  # CLI commands
│   ├── config/               # Configuration management
│   ├── downloader/           # Download & extraction
│   ├── golang/               # Go releases API
│   ├── logger/               # Logging utilities
│   ├── manager/              # Core version management
│   ├── progress/             # Progress bars
│   ├── shell/                # Shell integration
│   ├── symlink/              # Symlink management
│   ├── util/                 # Helper functions
│   └── version/              # Version information
├── scripts/
│   ├── install.sh            # Unix installation script
│   ├── install.ps1           # PowerShell installation
│   ├── install.bat           # Windows batch installation
│   ├── uninstall.sh          # Unix uninstall script
│   ├── uninstall.ps1         # PowerShell uninstall
│   └── uninstall.bat         # Windows batch uninstall
├── Makefile                  # Build automation
├── go.mod                    # Go module definition
├── go.sum                    # Dependency checksums
└── README.md                 # Project readme
```

## Development Workflow

### 1. Setup Development Environment

```bash
# Fork and clone
git clone https://github.com/YOUR_USERNAME/govman.git
cd govman

# Add upstream remote
git remote add upstream https://github.com/justjundana/govman.git

# Install dependencies
go mod download
```

### 2. Build from Source

```bash
# Build binary
make build

# Build for specific platform
GOOS=linux GOARCH=amd64 make build
GOOS=windows GOARCH=amd64 make build
GOOS=darwin GOARCH=arm64 make build
```

### 3. Run Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./internal/config/...
```

### 4. Run Linters

```bash
# Run all linters
make lint

# Format code
make fmt

# Vet code
make vet
```

### 5. Local Installation

```bash
# Install to ~/.govman/bin
make install

# Verify
govman --version
```

## Makefile Targets

```bash
make build          # Build for current platform
make install        # Install to ~/.govman/bin
make clean          # Clean built artifacts
make test           # Run tests
make test-coverage  # Run tests with coverage
make lint           # Run linters
make fmt            # Format code
make vet            # Run go vet
make release        # Build for all platforms
```

## Making Changes

### Creating a Feature Branch

```bash
# Update main branch
git checkout main
git pull upstream main

# Create feature branch
git checkout -b feature/my-feature
```

### Coding Standards

- Follow Google Go Style Guide
- Use `gofmt` for formatting
- Add comments for exported functions
- Write unit tests for new code
- Update documentation

### Testing Your Changes

```bash
# Run tests
make test

# Test specific functionality
go test -run TestFeatureName ./internal/package/

# Manual testing
./govman install 1.25.1
./govman use 1.25.1
./govman list
```

### Committing

```bash
# Stage changes
git add .

# Commit with descriptive message
git commit -m "feat: add new feature X"

# Follow conventional commits:
# feat: new feature
# fix: bug fix
# docs: documentation
# test: testing
# refactor: code refactoring
# chore: maintenance
```

## Adding New Commands

1. Create new file in `internal/cli/`:
   ```go
   // internal/cli/mycommand.go
   package cli
   
   import "github.com/spf13/cobra"
   
   func newMyCommandCmd() *cobra.Command {
       cmd := &cobra.Command{
           Use:   "mycommand",
           Short: "Short description",
           Long:  `Long description`,
           RunE: func(cmd *cobra.Command, args []string) error {
               // Implementation
               return nil
           },
       }
       return cmd
   }
   ```

2. Register in `internal/cli/command.go`:
   ```go
   func addCommands() {
       rootCmd.AddCommand(
           // ...existing commands
           newMyCommandCmd(),
       )
   }
   ```

3. Add tests in `internal/cli/mycommand_test.go`

4. Update documentation

## Debugging

### Verbose Logging

```bash
./govman --verbose install 1.25.1
```

### Using Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug
dlv debug ./cmd/govman -- install 1.25.1

# Set breakpoint
(dlv) break main.main
(dlv) continue
```

### Print Debugging

```go
import _logger "github.com/justjundana/govman/internal/logger"

_logger.Debug("Debug message: %v", variable)
_logger.Verbose("Verbose message")
```

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run specific package
go test ./internal/config/

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Writing Tests

```go
// internal/package/feature_test.go
package package_test

import (
    "testing"
)

func TestFeature(t *testing.T) {
    t.Run("should do X", func(t *testing.T) {
        // Arrange
        // Act
        // Assert
    })
}
```

## Building Releases

### For All Platforms

```bash
make release
```

This creates binaries in `dist/`:
- `govman-linux-amd64`
- `govman-linux-arm64`
- `govman-darwin-amd64`
- `govman-darwin-arm64`
- `govman-windows-amd64.exe`
- `govman-windows-arm64.exe`

### Custom Build

```bash
GOOS=linux GOARCH=amd64 go build \
  -ldflags "-X github.com/justjundana/govman/internal/version.Version=1.0.0" \
  -o dist/govman-linux-amd64 \
  ./cmd/govman
```

## Dependency Management

### Adding Dependencies

```bash
# Add new dependency
go get github.com/package/name@version

# Tidy dependencies
go mod tidy

# Vendor dependencies (optional)
go mod vendor
```

### Updating Dependencies

```bash
# Update all dependencies
go get -u ./...
go mod tidy

# Update specific package
go get -u github.com/package/name
```

## Documentation

### Updating Documentation

```bash
# Documentation is in docs/
cd docs/

# Edit relevant .md files
# Build/preview documentation
```

### Code Documentation

```go
// ExportedFunction does something useful.
// It takes a parameter and returns a result.
// Example:
//   result := ExportedFunction("input")
func ExportedFunction(input string) string {
    // Implementation
}
```

## Pull Request Process

1. **Fork** the repository
2. **Create branch** from main
3. **Make changes** with tests
4. **Run tests** and linters
5. **Commit** following conventional commits
6. **Push** to your fork
7. **Open PR** with description
8. **Address review** feedback
9. **Squash and merge** when approved

### PR Checklist

- [ ] Code follows style guidelines
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] All tests passing
- [ ] Linters passing
- [ ] Commit messages follow convention
- [ ] PR description is clear

## Continuous Integration

GitHub Actions runs on every PR:
- Build for all platforms
- Run tests
- Run linters
- Check code coverage

View workflows in `.github/workflows/`.

## Release Process

1. Update version in `internal/version/version.go`
2. Update `CHANGELOG.md`
3. Create and push tag:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```
4. GitHub Actions builds and uploads binaries
5. Create GitHub Release with notes

## Getting Help

- **Documentation**: See `docs/` directory
- **Issues**: Open GitHub issue with details
- **Discussions**: Use GitHub Discussions for questions
- **Code Review**: Ask in PR comments
