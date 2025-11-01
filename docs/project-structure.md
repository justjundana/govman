# Project Structure

Understanding the govman codebase organization and architecture.

## Directory Overview

```
govman/
├── cmd/
│   └── govman/              # Main application entry point
│       └── main.go          # CLI initialization
│
├── internal/                # Private application code
│   ├── cli/                 # CLI commands and interface
│   │   ├── cli.go           # Root command and banner
│   │   ├── command.go       # Command registration
│   │   ├── clean.go         # Clean cache command
│   │   ├── current.go       # Show current version
│   │   ├── info.go          # Version information
│   │   ├── init.go          # Shell integration setup
│   │   ├── install.go       # Install/uninstall commands
│   │   ├── list.go          # List versions
│   │   ├── refresh.go       # Refresh context
│   │   ├── selfupdate.go    # Self-update functionality
│   │   └── use.go           # Version switching
│   │
│   ├── config/              # Configuration management
│   │   ├── config.go        # Config struct and loading
│   │   └── config_test.go   # Config tests
│   │
│   ├── downloader/          # Download and extraction
│   │   ├── downloader.go    # Download orchestration
│   │   └── downloader_test.go
│   │
│   ├── golang/              # Go version management
│   │   ├── releases.go      # Version API and comparison
│   │   └── releases_test.go
│   │
│   ├── logger/              # Logging system
│   │   ├── logger.go        # Logger implementation
│   │   └── logger_test.go
│   │
│   ├── manager/             # Core version management
│   │   ├── manager.go       # Manager orchestration
│   │   └── manager_test.go
│   │
│   ├── progress/            # Progress bars
│   │   ├── progress.go      # Progress bar implementation
│   │   └── progress_test.go
│   │
│   ├── shell/               # Shell integration
│   │   ├── shell.go         # Shell interface and implementations
│   │   └── shell_test.go
│   │
│   ├── symlink/             # Symlink management
│   │   ├── symlink.go       # Symlink creation/reading
│   │   └── symlink_test.go
│   │
│   ├── util/                # Utility functions
│   │   ├── format.go        # Formatting helpers
│   │   └── format_test.go
│   │
│   └── version/             # Version information
│       ├── version.go       # Build version metadata
│       └── version_test.go
│
├── scripts/                 # Installation scripts
│   ├── install.sh           # Unix/Linux/macOS installer
│   ├── install.ps1          # PowerShell installer
│   ├── install.bat          # Windows CMD installer
│   ├── uninstall.sh         # Unix uninstaller
│   ├── uninstall.ps1        # PowerShell uninstaller
│   └── uninstall.bat        # Windows CMD uninstaller
│
├── docs/                    # Documentation
│   ├── quick-start.md       # Quick start guide
│   ├── installation.md      # Installation instructions
│   ├── configuration.md     # Configuration reference
│   ├── shell-integration.md # Shell setup guide
│   ├── commands.md          # Commands reference
│   ├── troubleshooting.md   # Troubleshooting guide
│   ├── release-notes.md     # Release notes
│   ├── getting-started.md   # Developer guide
│   ├── project-structure.md # This file
│   ├── dependencies.md      # Dependencies documentation
│   ├── data-flow.md         # Data flow diagrams
│   ├── architecture.md      # Architecture overview
│   └── architecture-diagrams.md
│
├── go.mod                   # Go module definition
├── go.sum                   # Dependency checksums
├── Makefile                 # Build automation
├── README.md                # Project README
├── LICENSE.md               # MIT License
└── CHANGELOG.md             # Version history
```

## Core Components

### 1. Entry Point (`cmd/govman`)

**Purpose**: Application initialization  
**Files**: `main.go`

Minimal entry point that:
- Calls `cli.Execute()`
- Handles top-level errors
- Sets exit codes

### 2. CLI Layer (`internal/cli`)

**Purpose**: User interface and command handling  
**Key Files**:
- `cli.go` - Root command, banner, config initialization
- `command.go` - Command registration and global flags
- `*_cmd.go` - Individual command implementations

**Responsibilities**:
- Parse command-line arguments
- Display user-friendly output
- Call Manager for business logic
- Handle errors gracefully

### 3. Manager (`internal/manager`)

**Purpose**: Core business logic orchestration  
**Key File**: `manager.go`

**Main Functions**:
- `Install()` - Download and install Go versions
- `Uninstall()` - Remove installed versions
- `Use()` - Switch between versions
- `Current()` - Get active version
- `ListInstalled()` - List local versions
- `ListRemote()` - Fetch available versions

**Coordinates**:
- Configuration
- Downloader
- Shell integration
- Symlink management

### 4. Configuration (`internal/config`)

**Purpose**: Configuration management  
**Key File**: `config.go`

**Features**:
- YAML-based configuration
- Default values
- Path expansion and validation
- Security checks (path traversal prevention)
- Config file creation and updates

### 5. Downloader (`internal/downloader`)

**Purpose**: Download and extract Go distributions  
**Key File**: `downloader.go`

**Capabilities**:
- HTTP downloads with retry logic
- Resume interrupted downloads
- SHA-256 checksum verification
- Archive extraction (.tar.gz, .zip)
- Path traversal protection
- Progress reporting

### 6. Go Releases (`internal/golang`)

**Purpose**: Go version information and comparison  
**Key File**: `releases.go`

**Functions**:
- Fetch available versions from go.dev
- Parse and cache version data
- Compare semantic versions
- Generate download URLs
- Extract version metadata

### 7. Shell Integration (`internal/shell`)

**Purpose**: Shell-specific integration  
**Key File**: `shell.go`

**Supported Shells**:
- Bash
- Zsh
- Fish
- PowerShell
- Command Prompt (limited)

**Features**:
- Shell detection
- Configuration file management
- PATH manipulation
- Auto-switch hooks
- Wrapper function generation

### 8. Logger (`internal/logger`)

**Purpose**: Structured logging  
**Key File**: `logger.go`

**Log Levels**:
- Quiet (errors only)
- Normal (standard output)
- Verbose (debug information)

**Output Types**:
- Info, Success, Warning, Error
- Progress indicators
- Download/Extract/Verify status
- Timing instrumentation

### 9. Progress (`internal/progress`)

**Purpose**: Progress bars for long operations  
**Key File**: `progress.go`

**Features**:
- Real-time progress updates
- Download speed calculation
- ETA estimation
- Byte formatting
- Multi-progress support

### 10. Symlink (`internal/symlink`)

**Purpose**: Cross-platform symlink management  
**Key File**: `symlink.go`

**Functions**:
- Create symlinks
- Read symlink targets
- Handle Windows/Unix differences

### 11. Utilities (`internal/util`)

**Purpose**: Helper functions  
**Key File**: `format.go`

**Helpers**:
- Byte size formatting (KB, MB, GB)
- Duration formatting (3m12s, 2h05m)

### 12. Version (`internal/version`)

**Purpose**: Build information  
**Key File**: `version.go`

**Metadata**:
- Version number
- Git commit hash
- Build date
- Builder information
- Go version used

## Code Organization Principles

### 1. Separation of Concerns

Each package has a single, well-defined responsibility:
- CLI handles user interaction
- Manager handles business logic
- Downloader handles downloads
- Config handles configuration

### 2. Dependency Direction

```
cmd/govman
    ↓
internal/cli
    ↓
internal/manager
    ↓
internal/{config, downloader, golang, shell, ...}
```

Higher-level packages depend on lower-level packages, never the reverse.

### 3. Internal Packages

All implementation code is in `internal/` to prevent external imports.

### 4. Test Co-location

Tests are placed alongside the code they test:
- `manager.go` → `manager_test.go`

### 5. Minimal main.go

The entry point is minimal - just error handling and CLI invocation.

## Design Patterns

### 1. Singleton Pattern (Logger)

```go
var globalLogger *Logger
var once sync.Once

func Get() *Logger {
    once.Do(func() {
        globalLogger = New()
    })
    return globalLogger
}
```

### 2. Strategy Pattern (Shell Integration)

```go
type Shell interface {
    Name() string
    ConfigFile() string
    PathCommand(path string) string
    SetupCommands(binPath string) []string
}

type BashShell struct{}
type ZshShell struct{}
type FishShell struct{}
```

### 3. Builder Pattern (Configuration)

```go
cfg := &Config{}
cfg.setDefaults()
cfg.expandPaths()
cfg.createDirectories()
return cfg
```

### 4. Facade Pattern (Manager)

Manager provides a simplified interface to complex subsystems:

```go
manager := New(config)
manager.Install(version)  // Coordinates: download, extract, verify
```

## File Naming Conventions

- **Commands**: `<command>.go` (e.g., `install.go`, `list.go`)
- **Tests**: `<file>_test.go`
- **Integration tests**: `<file>_integration_test.go`
- **Interfaces**: Defined in the main package file
- **Implementations**: Separate files or inline in main file

## Import Organization

```go
import (
    // Standard library
    "fmt"
    "os"
    "path/filepath"
    
    // External dependencies
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    
    // Internal packages (with aliases)
    _config "github.com/justjundana/govman/internal/config"
    _manager "github.com/justjundana/govman/internal/manager"
)
```

## Key Interfaces

### Shell Interface

```go
type Shell interface {
    Name() string
    DisplayName() string
    ConfigFile() string
    PathCommand(path string) string
    SetupCommands(binPath string) []string
    IsAvailable() bool
    ExecutePathCommand(path string) error
}
```

### Logger Interface

Singleton pattern, package-level functions:

```go
logger.Info("message")
logger.Error("error message")
logger.Success("success message")
```

## Data Structures

### Config

```go
type Config struct {
    InstallDir     string
    CacheDir       string
    DefaultVersion string
    Download       DownloadConfig
    Mirror         MirrorConfig
    AutoSwitch     AutoSwitchConfig
    // ...
}
```

### VersionInfo

```go
type VersionInfo struct {
    Version     string
    Path        string
    OS          string
    Arch        string
    InstallDate time.Time
    Size        int64
}
```

### Release

```go
type Release struct {
    Version string
    Stable  bool
    Files   []File
}

type File struct {
    Filename string
    OS       string
    Arch     string
    Sha256   string
    Size     int64
}
```

## Testing Structure

### Unit Tests

- Located alongside source files
- Test individual functions
- Use table-driven tests

### Integration Tests

- Tagged with `//go:build integration`
- Test component interactions
- Require `go test -tags=integration`

### Test Helpers

Common patterns:
- `setupTest()` - Initialize test environment
- `cleanupTest()` - Clean up after tests
- Table-driven tests for multiple cases

## Build Process

### Development Build

```bash
go build -o govman ./cmd/govman
```

### Production Build

```bash
go build -ldflags="-s -w -X 'github.com/justjundana/govman/internal/version.Version=v1.0.0'" \
    -o govman ./cmd/govman
```

### Multi-Platform Build

```bash
GOOS=linux GOARCH=amd64 go build -o govman-linux-amd64 ./cmd/govman
GOOS=darwin GOARCH=arm64 go build -o govman-darwin-arm64 ./cmd/govman
GOOS=windows GOARCH=amd64 go build -o govman-windows-amd64.exe ./cmd/govman
```

## Error Handling

### Pattern

```go
if err := doSomething(); err != nil {
    _logger.ErrorWithHelp(
        "Failed to do something",
        "Try this workaround...",
        details
    )
    return fmt.Errorf("context: %w", err)
}
```

### Error Wrapping

Always wrap errors with context using `%w`:

```go
return fmt.Errorf("failed to download: %w", err)
```

## See Also

- [Architecture](architecture.md) - System design
- [Data Flow](data-flow.md) - How data moves
- [Dependencies](dependencies.md) - External packages
- [Developer Guide](getting-started.md) - Development setup

---

Understanding the structure helps you navigate and contribute effectively! 🗺️
