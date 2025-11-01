# Developer Getting Started

Welcome to govman development! This guide will help you set up your development environment and make your first contribution.

## Prerequisites

### Required

- **Go 1.25 or later** - [Install Go](https://go.dev/doc/install)
- **Git** - Version control
- **Make** - Build automation (optional but recommended)

### Recommended

- **Docker** - For testing installation scripts
- **VS Code** - With Go extension
- Code editor with Go support

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/justjundana/govman.git
cd govman
```

### 2. Install Dependencies

```bash
go mod download
go mod verify
```

### 3. Build the Project

```bash
# Using Make
make build

# Or directly with Go
go build -o govman ./cmd/govman
```

### 4. Run Tests

```bash
# Run all tests
go test ./...

# With verbose output
go test -v ./...

# With coverage
go test -cover ./...
```

### 5. Run govman Locally

```bash
./govman --version
./govman --help
```

## Development Workflow

### Building

```bash
# Development build
make build

# Build for all platforms
make build-all

# Clean build artifacts
make clean
```

### Testing

```bash
# Unit tests
make test

# Integration tests
make test-integration

# Coverage report
make test-coverage

# Run specific test
go test -v ./internal/manager -run TestInstall
```

### Code Quality

```bash
# Format code
make fmt
# Or: go fmt ./...

# Lint code
make lint
# Or: golangci-lint run

# Vet code
go vet ./...
```

## Project Commands

### Makefile Targets

```bash
make help          # Show all available targets
make build         # Build the binary
make test          # Run tests
make fmt           # Format code
make lint          # Run linters
make clean         # Clean build artifacts
make install       # Install to ~/.govman/bin
make uninstall     # Remove from ~/.govman/bin
```

## Development Environment

### VS Code Setup

Create `.vscode/settings.json`:

```json
{
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "go.formatTool": "gofmt",
  "editor.formatOnSave": true,
  "go.useLanguageServer": true,
  "go.testFlags": ["-v"],
  "go.coverOnSave": true
}
```

### Debugging

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug govman",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/govman",
      "args": ["list", "--verbose"]
    }
  ]
}
```

## Making Changes

### 1. Create a Branch

```bash
git checkout -b feature/my-new-feature
# Or
git checkout -b fix/bug-description
```

### 2. Make Your Changes

Follow the [Project Structure](project-structure.md) to understand the codebase.

### 3. Write Tests

Every new feature should include tests:

```go
func TestMyNewFeature(t *testing.T) {
    // Arrange
    expected := "value"
    
    // Act
    result := MyNewFeature()
    
    // Assert
    if result != expected {
        t.Errorf("Expected %s, got %s", expected, result)
    }
}
```

### 4. Run Tests

```bash
go test ./...
```

### 5. Commit Your Changes

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```bash
git add .
git commit -m "feat: add new awesome feature"
# Or
git commit -m "fix: resolve issue with version switching"
```

### 6. Push and Create PR

```bash
git push origin feature/my-new-feature
```

Then create a Pull Request on GitHub.

## Code Style Guidelines

### Go Code

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting
- Keep functions small and focused
- Write descriptive variable names
- Add comments for exported functions

### Example

```go
// Manager handles Go version management operations.
// It coordinates between configuration, downloads, and shell integration.
type Manager struct {
    config     *config.Config
    downloader *downloader.Downloader
    shell      shell.Shell
}

// New creates a Manager with the provided configuration.
// It initializes a downloader and detects the user's shell.
func New(cfg *config.Config) *Manager {
    return &Manager{
        config:     cfg,
        downloader: downloader.New(cfg),
        shell:      shell.Detect(),
    }
}
```

### Comments

- Add package documentation
- Document exported types and functions
- Explain complex logic
- Use TODO for future improvements

```go
// TODO: Add support for custom mirror URLs
// TODO(username): Optimize caching strategy
```

## Testing Guidelines

### Unit Tests

- Test each function in isolation
- Use table-driven tests for multiple cases
- Mock external dependencies

```go
func TestCompareVersions(t *testing.T) {
    tests := []struct {
        name     string
        v1       string
        v2       string
        expected int
    }{
        {"equal versions", "1.21.5", "1.21.5", 0},
        {"v1 greater", "1.21.5", "1.20.12", 1},
        {"v2 greater", "1.20.12", "1.21.5", -1},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := CompareVersions(tt.v1, tt.v2)
            if result != tt.expected {
                t.Errorf("Expected %d, got %d", tt.expected, result)
            }
        })
    }
}
```

### Integration Tests

Place in `*_integration_test.go` files:

```go
//go:build integration
// +build integration

func TestInstallIntegration(t *testing.T) {
    // Test actual installation
}
```

Run with:
```bash
go test -tags=integration ./...
```

## Debugging Tips

### Print Debugging

```go
import "log"

log.Printf("Debug: value = %v", value)
```

### Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug govman
dlv debug ./cmd/govman -- list
```

### Verbose Mode

Use the `--verbose` flag to see detailed output:

```bash
./govman --verbose install 1.21.5
```

## Common Development Tasks

### Adding a New Command

1. Create command file in `internal/cli/`:

```go
// internal/cli/mynewcmd.go
package cli

func newMyNewCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "mynew",
        Short: "Description",
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
        // ... existing commands
        newMyNewCmd(),
    )
}
```

### Adding Configuration Option

1. Add to `internal/config/config.go`:

```go
type Config struct {
    // ... existing fields
    MyNewOption string `mapstructure:"my_new_option"`
}
```

2. Add to `setDefaults()`:

```go
func (c *Config) setDefaults() {
    // ... existing defaults
    c.MyNewOption = "default_value"
}
```

### Adding a Shell

1. Create shell implementation in `internal/shell/`:

```go
type MyShell struct{}

func (s *MyShell) Name() string { return "myshell" }
func (s *MyShell) DisplayName() string { return "My Shell" }
func (s *MyShell) ConfigFile() string { return "~/.myshellrc" }
// ... implement other methods
```

2. Add to `Detect()` in `internal/shell/shell.go`

## Release Process

### Creating a Release

1. Update version in `internal/version/version.go`
2. Update `CHANGELOG.md`
3. Create git tag:

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

4. GitHub Actions will build and publish

## Resources

- [Project Structure](project-structure.md) - Codebase organization
- [Architecture](architecture.md) - System design
- [Dependencies](dependencies.md) - External packages
- [Data Flow](data-flow.md) - How data moves through the system

## Getting Help

- 💬 [GitHub Discussions](https://github.com/justjundana/govman/discussions)
- 🐛 [Issue Tracker](https://github.com/justjundana/govman/issues)
- 📖 Read the code - It's well-documented!

## Next Steps

1. Read [Project Structure](project-structure.md)
2. Explore [Architecture](architecture.md)
3. Check [Open Issues](https://github.com/justjundana/govman/issues)
4. Make your first contribution!

---

Happy coding! 🚀
