# Developer Onboarding

Complete guide for new developers joining the govman project.

## Welcome! 🎉

Thank you for your interest in contributing to govman! This guide will help you get started quickly and effectively.

## What is govman?

govman is a Go Version Manager that allows developers to:
- Install and manage multiple Go versions
- Quickly switch between Go versions
- Automatically switch versions based on project requirements
- Work across Linux, macOS, and Windows

## Prerequisites

Before you begin, ensure you have:

### Required
- **Go 1.20+**: [Download](https://go.dev/dl/)
- **Git**: [Download](https://git-scm.com/downloads)
- **Make**: Usually pre-installed on Linux/macOS; Windows users can use [Make for Windows](http://gnuwin32.sourceforge.net/packages/make.htm)

### Recommended
- **VS Code** with Go extension
- **delve** debugger: `go install github.com/go-delve/delve/cmd/dlv@latest`
- **Terminal** with shell support (Bash, Zsh, Fish, or PowerShell)

### Check Your Setup

```bash
# Verify Go
go version  # Should be 1.20 or higher

# Verify Git
git --version

# Verify Make
make --version
```

## Quick Start (5 Minutes)

### 1. Clone the Repository

```bash
# Clone
git clone https://github.com/justjundana/govman.git
cd govman

# Fork first if you plan to contribute
# Then clone your fork:
# git clone https://github.com/YOUR_USERNAME/govman.git
```

### 2. Build the Project

```bash
# Download dependencies
go mod download

# Build binary
make build

# Verify build
./build/govman --version
```

### 3. Run Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test ./internal/manager
```

### 4. Try It Out

```bash
# Install locally (optional)
make install

# Or run directly
./build/govman list

# Try installing a Go version
./build/govman install 1.21.5
```

## Project Structure

```
govman/
├── cmd/
│   └── govman/          # Main application entry point
│       └── main.go      # CLI initialization
│
├── internal/            # Private application code
│   ├── cli/             # CLI commands (10 files)
│   ├── config/          # Configuration management
│   ├── downloader/      # Download & extraction
│   ├── golang/          # Go releases API
│   ├── logger/          # Logging system
│   ├── manager/         # Core business logic
│   ├── progress/        # Progress bars
│   ├── shell/           # Shell integration
│   ├── symlink/         # Symlink management
│   ├── util/            # Utilities
│   └── version/         # Version information
│
├── docs/                # Documentation (you're here!)
├── scripts/             # Installation scripts
├── build/               # Build output (ignored by git)
├── Makefile             # Build automation
├── go.mod               # Go module definition
└── go.sum               # Dependency checksums
```

### Key Files to Know

| File | Purpose |
|------|---------|
| `cmd/govman/main.go` | Application entry point |
| `internal/cli/cli.go` | Root command & CLI setup |
| `internal/manager/manager.go` | Core business logic |
| `internal/config/config.go` | Configuration management |
| `Makefile` | Build commands |

## Development Workflow

### 1. Create a Branch

```bash
# Update main
git checkout main
git pull origin main

# Create feature branch
git checkout -b feature/my-new-feature

# Or for bug fixes
git checkout -b fix/issue-123
```

### 2. Make Changes

```bash
# Edit files
vim internal/cli/install.go

# Build frequently
make build

# Test your changes
go test ./internal/cli

# Run the binary
./build/govman install 1.21.5
```

### 3. Write Tests

```go
// internal/cli/install_test.go
func TestInstall(t *testing.T) {
    tests := []struct {
        name    string
        version string
        wantErr bool
    }{
        {"valid version", "1.21.5", false},
        {"invalid version", "invalid", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := runInstall(tt.version)
            if (err != nil) != tt.wantErr {
                t.Errorf("runInstall() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 4. Format and Lint

```bash
# Format code
go fmt ./...

# Run linter (install first: go install golang.org/x/lint/golint@latest)
golint ./...

# Run vet
go vet ./...
```

### 5. Commit Your Changes

```bash
# Stage changes
git add .

# Commit with descriptive message
git commit -m "feat: add support for arm64 architecture"

# Or for bug fixes
git commit -m "fix: resolve symlink issue on Windows"
```

**Commit Message Format**:
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Test additions/changes
- `refactor:` Code refactoring
- `chore:` Maintenance tasks

### 6. Push and Create PR

```bash
# Push to your fork
git push origin feature/my-new-feature

# Create Pull Request on GitHub
# Go to https://github.com/justjundana/govman/pulls
# Click "New Pull Request"
```

## Common Development Tasks

### Adding a New CLI Command

1. **Create command file**:
```bash
touch internal/cli/mycommand.go
```

2. **Implement command**:
```go
// internal/cli/mycommand.go
package cli

import (
    "github.com/spf13/cobra"
)

var myCmd = &cobra.Command{
    Use:   "mycommand <arg>",
    Short: "Description of my command",
    Long:  "Longer description...",
    Args:  cobra.ExactArgs(1),
    Run:   runMyCommand,
}

func runMyCommand(cmd *cobra.Command, args []string) {
    arg := args[0]
    
    // Use manager
    mgr := getManager()
    if err := mgr.MyOperation(arg); err != nil {
        _logger.Error("Operation failed: %v", err)
        os.Exit(1)
    }
    
    _logger.Success("Operation completed!")
}

func init() {
    // Add flags if needed
    myCmd.Flags().StringP("option", "o", "", "option description")
}
```

3. **Register command**:
```go
// internal/cli/command.go
func init() {
    rootCmd.AddCommand(
        // ... existing commands
        myCmd,  // Add your command
    )
}
```

4. **Add tests**:
```go
// internal/cli/mycommand_test.go
func TestMyCommand(t *testing.T) {
    // Test implementation
}
```

### Adding a Configuration Option

1. **Update Config struct**:
```go
// internal/config/config.go
type Config struct {
    // Existing fields...
    MyNewOption string `mapstructure:"my_new_option"`
}
```

2. **Set default value**:
```go
func (c *Config) setDefaults() {
    // Existing defaults...
    viper.SetDefault("my_new_option", "default_value")
}
```

3. **Update example config**:
```yaml
# config.yaml.example
my_new_option: "default_value"  # Description
```

4. **Document in docs**:
```markdown
<!-- docs/configuration.md -->
### my_new_option
Description of the option.
```

### Adding Shell Support

1. **Implement Shell interface**:
```go
// internal/shell/shell.go
type MyShell struct{}

func (s *MyShell) Name() string {
    return "myshell"
}

func (s *MyShell) ConfigFile() string {
    return filepath.Join(os.Getenv("HOME"), ".myshellrc")
}

func (s *MyShell) PathCommand(binPath string) string {
    return fmt.Sprintf(`export PATH="%s:$PATH"`, binPath)
}

func (s *MyShell) SetupCommands(binPath string) []string {
    return []string{
        "# govman initialization",
        fmt.Sprintf(`export PATH="%s:$PATH"`, binPath),
        // Auto-switch hook
        `govman_auto_switch() {`,
        `    if [[ -f .govman-version ]]; then`,
        `        local required_version=$(cat .govman-version 2>/dev/null)`,
        `        govman use "$required_version" >/dev/null 2>&1`,
        `    fi`,
        `}`,
        `PROMPT_COMMAND="govman_auto_switch; $PROMPT_COMMAND"`,
    }
}
```

2. **Register shell**:
```go
// internal/shell/shell.go
func Detect() Shell {
    shell := os.Getenv("SHELL")
    switch {
    // Existing cases...
    case strings.Contains(shell, "myshell"):
        return &MyShell{}
    }
}
```

## Testing Guidelines

### Unit Tests

Test individual functions:

```go
func TestNormalizeVersion(t *testing.T) {
    tests := []struct {
        input string
        want  string
    }{
        {"1.21.5", "go1.21.5"},
        {"go1.21.5", "go1.21.5"},
        {"1.21", "go1.21.0"},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            got := normalizeVersion(tt.input)
            if got != tt.want {
                t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
            }
        })
    }
}
```

### Integration Tests

Test component interactions:

```go
//go:build integration

func TestRealInstall(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Real test with actual downloads
    mgr := setupTestManager(t)
    err := mgr.Install("1.21.5")
    if err != nil {
        t.Fatalf("Install failed: %v", err)
    }
}
```

**Run**:
```bash
# Unit tests only
go test -short ./...

# Integration tests
go test -tags=integration ./...
```

## Debugging

### VS Code Debug Configuration

```json
// .vscode/launch.json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug govman",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/govman",
            "args": ["install", "1.21.5"],
            "env": {
                "GOVMAN_CONFIG": "${workspaceFolder}/test-config.yaml"
            }
        }
    ]
}
```

### Command Line Debugging

```bash
# Using delve
dlv debug ./cmd/govman -- install 1.21.5

# Set breakpoint
(dlv) break internal/manager/manager.go:45
(dlv) continue

# Inspect
(dlv) print version
(dlv) locals
```

### Print Debugging

```go
func (m *Manager) Install(version string) error {
    fmt.Printf("DEBUG: Install called with version=%q\n", version)
    fmt.Printf("DEBUG: config=%+v\n", m.config)
    
    // ... rest of function
}
```

## Code Style

### Follow Effective Go

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### Key Conventions

1. **Naming**:
```go
// Good
func GetVersion() string
var maxRetries int

// Bad
func get_version() string
var MAX_RETRIES int
```

2. **Error handling**:
```go
// Good
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Bad
if err != nil {
    panic(err)
}
```

3. **Comments**:
```go
// Good: Comment explains why
// Use larger buffer for better performance with large files
const bufferSize = 64 * 1024

// Bad: Comment states the obvious
// Set buffer size
const bufferSize = 64 * 1024
```

## Getting Help

### Documentation

- [Quick Start](quick-start.md) - User guide
- [Architecture](architecture.md) - System design
- [Troubleshooting](internals-troubleshooting.md) - Debugging guide

### Community

- **GitHub Issues**: [govman/issues](https://github.com/justjundana/govman/issues)
- **Discussions**: [govman/discussions](https://github.com/justjundana/govman/discussions)
- **Email**: Contact maintainers

### Before Asking

1. Check existing documentation
2. Search closed issues
3. Try debugging yourself
4. Prepare a minimal reproduction

### When Asking

Include:
1. What you're trying to do
2. What you've tried
3. Relevant code snippets
4. Error messages
5. Environment details (OS, Go version)

## Contribution Checklist

Before submitting a PR:

- [ ] Code builds successfully: `make build`
- [ ] Tests pass: `make test`
- [ ] Code is formatted: `go fmt ./...`
- [ ] Code is vetted: `go vet ./...`
- [ ] New features have tests
- [ ] Documentation is updated
- [ ] Commit messages are clear
- [ ] Branch is up to date with main

## Learning Path

### Week 1: Familiarization
- [ ] Clone and build project
- [ ] Run all tests
- [ ] Read architecture documentation
- [ ] Try using govman yourself
- [ ] Explore codebase structure

### Week 2: Small Changes
- [ ] Fix a typo in documentation
- [ ] Add a test case
- [ ] Improve an error message
- [ ] Submit your first PR

### Week 3: Feature Development
- [ ] Pick a "good first issue"
- [ ] Implement the feature
- [ ] Write comprehensive tests
- [ ] Update documentation

### Week 4+: Core Contributor
- [ ] Review others' PRs
- [ ] Help with issues
- [ ] Propose new features
- [ ] Improve architecture

## Best Practices

### 1. Write Tests First

```go
// Write the test
func TestNewFeature(t *testing.T) {
    got := newFeature()
    want := "expected"
    if got != want {
        t.Errorf("got %v, want %v", got, want)
    }
}

// Then implement
func newFeature() string {
    return "expected"
}
```

### 2. Keep Changes Small

- One PR per feature/fix
- Easy to review
- Faster to merge

### 3. Document As You Go

- Update docs with code
- Add code comments
- Write clear commit messages

### 4. Communicate Early

- Open issue before big changes
- Ask questions in discussions
- Request feedback early

## Next Steps

Now that you're set up:

1. **Explore**: Browse the codebase, read the architecture docs
2. **Build**: Make a small change and see it work
3. **Contribute**: Pick an issue and submit a PR
4. **Learn**: Review others' code, ask questions

Welcome to the team! 🚀

## See Also

- [Getting Started](getting-started.md) - Quick development setup
- [Architecture](architecture.md) - System design
- [Project Structure](project-structure.md) - Code organization
- [Contributing Guide](../CONTRIBUTING.md) - Contribution guidelines

---

Happy coding! We're excited to have you here! 💻✨
