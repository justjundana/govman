# Architecture Overview

High-level architecture and design decisions for govman.

## Design Philosophy

govman is built with these core principles:

1. **Simplicity**: Easy to use, easy to understand
2. **Safety**: No root required, safe defaults
3. **Speed**: Fast downloads, instant switching
4. **Reliability**: Checksum verification, atomic operations
5. **Cross-platform**: Works on Linux, macOS, Windows

## Architecture Layers

```
┌─────────────────────────────────────┐
│         User Interface              │  (CLI, Shell Integration)
├─────────────────────────────────────┤
│      Application Logic              │  (Commands, Workflows)
├─────────────────────────────────────┤
│       Core Services                 │  (Manager, Downloader, Config)
├─────────────────────────────────────┤
│        Utilities                    │  (Logger, Progress, Format)
├─────────────────────────────────────┤
│    External Dependencies            │  (Cobra, Viper, stdlib)
└─────────────────────────────────────┘
```

### Layer Descriptions

**User Interface**:
- CLI commands (`internal/cli/`)
- Shell integration code generation (`internal/shell/`)
- User-facing messages and help text

**Application Logic**:
- Command orchestration
- Input validation
- Workflow coordination

**Core Services**:
- Version management (`internal/manager/`)
- Download and extraction (`internal/downloader/`)
- Configuration (`internal/config/`)
- Go releases API (`internal/golang/`)

**Utilities**:
- Logging (`internal/logger/`)
- Progress reporting (`internal/progress/`)
- String formatting (`internal/util/`)

**External Dependencies**:
- Cobra (CLI framework)
- Viper (configuration)
- Go standard library

## Component Diagram

```
┌────────────────────────────────────────────────────────┐
│                       CLI Layer                        │
│  ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐     │
│  │Install│ │ Use   │ │ List  │ │ Info  │ │ Init  │ ... │
│  └───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘     │
└──────┼───-─────┼────-────┼-────────┼────-────┼─────────┘
       │         │         │         │         │
       └──────-─-┴────-────┴─────-───┴───-─────┘
                           │
                  ┌────────▼────────┐
                  │                 │
                  │    Manager      │
                  │                 │
                  └────┬───┬───┬────┘
                       │   │   │
           ┌───────────┘   │   └──────────┐
           │               │              │
      ┌────▼─────┐  ┌-─────▼─────┐  ┌───-─▼─-──┐
      │Downloader│  │   Config   │  │  Golang  │
      └────┬─────┘  └─-──────────┘  └────-┬────┘
           │                              │
      ┌────▼─────┐                   ┌───-▼-───┐
      │ Progress │                   │ go.dev  │
      └──────────┘                   │   API   │
                                     └─────────┘
```

## Key Design Patterns

### 1. Facade Pattern

`Manager` acts as a facade for core services:

```go
type Manager struct {
    config     *config.Config
    downloader *downloader.Downloader
    shell      shell.Shell
}
```

CLI commands interact with `Manager`, which coordinates lower-level services.

### 2. Strategy Pattern

Different shell implementations via `Shell` interface:

```go
type Shell interface {
    Name() string
    ConfigFile() string
    PathCommand(path string) string
    SetupCommands(binPath string) []string
    // ...
}

// Implementations:
type BashShell struct{}
type ZshShell struct{}
type FishShell struct{}
type PowerShell struct{}
```

### 3. Singleton Pattern

Global logger instance:

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

### 4. Template Method

Download workflow in Downloader:

```go
func (d *Downloader) Download(url, installDir, version string) error {
    // Template method defines steps:
    1. Get file info
    2. Download file
    3. Verify checksum
    4. Extract archive
}
```

## Data Flow Architecture

```
User Input
    ↓
CLI Parsing (Cobra)
    ↓
Command Validation
    ↓
Manager Orchestration
    ↓
Service Execution (Parallel where safe)
    ↓
Result Aggregation
    ↓
User Output (Formatted by Logger)
```

## State Management

### Application State

- **Method**: Configuration file (`~/.govman/config.yaml`)
- **Format**: YAML
- **Persistence**: Disk-based
- **Updates**: Atomic write (temp file + rename)

### Runtime State

- **Current version**: Resolved from symlink or environment
- **Session state**: In-memory (not persisted)
- **Progress**: Ephemeral UI state

### No Global Mutable State

- Configuration passed explicitly
- No global variables (except logger singleton)
- Each command execution is isolated

## Error Handling Strategy

### Layered Error Handling

```
Low-level error (e.g., HTTP 404)
    ↓ wrapped with context
Mid-level error (e.g., "failed to download")
    ↓ formatted for user
High-level error (e.g., "Go 1.25.1 not available for your platform")
    ↓ displayed with help
User sees actionable message
```

### Error Types

1. **Validation errors**: User input issues
2. **Network errors**: Download/API failures
3. **Filesystem errors**: Permission/space issues
4. **Logic errors**: Invalid state

All errors include:
- Clear message
- Suggested action (via `ErrorWithHelp`)
- Exit code

## Security Architecture

### Principle of Least Privilege

- **No root required**: All operations in user space
- **Limited file access**: Only ~/.govman/ and shell configs
- **No network server**: Client-only architecture

### Defense in Depth

1. **Input validation**: All user inputs validated
2. **Path validation**: Prevent directory traversal
3. **Checksum verification**: SHA-256 for all downloads
4. **HTTPS only**: Encrypted connections
5. **Safe defaults**: Secure configuration out of the box

### Trust Model

**Trusted**:
- go.dev (official Go releases)
- github.com (govman releases)
- User's local system

**Not trusted**:
-User input (validated before use)
- Custom mirror URLs (optional, user-configured)

## Concurrency Model

### Single-threaded Command Execution

- One command at a time per user
- No locking needed (user-space isolation)
- Simple, predictable behavior

### Safe Parallel Downloading

- HTTP connections can be parallel (configurable)
- Uses standard library's goroutines
- Progress reporting thread-safe

### Atomic Operations

- File writes: temp file + rename
- Symlink updates: atomic at OS level
- Configuration updates: single write operation

## Extensibility Points

### Adding New Commands

1. Create file in `internal/cli/`
2. Implement `cobra.Command`
3. Register in `addCommands()`

### Adding New Shells

1. Implement `Shell` interface
2. Add to `Detect()` logic
3. Add to `getShellByName()`

### Changing Configuration

1. Update `Config` struct in `internal/config/`
2. Set default in `setDefaults()`
3. Configuration automatically persists

## Performance Considerations

### Optimization Strategies

1. **Caching**: API responses cached for 10 minutes
2. **Parallel downloads**: Multiple connections (configurable)
3. **Resume support**: Incomplete downloads resume
4. **Minimal I/O**: Only read/write when necessary

### Trade-offs

- **Simplicity over speed**: Single-threaded for safety
- **Safety over size**: Self-contained binary with dependencies
- **UX over efficiency**: Progress bars worth the overhead

## Platform Abstractions

### Cross-platform Code

```go
// Path handling
filepath.Join()  // Works on all platforms

// Symlinks
os.Symlink()  // Supported on all modern OSes

// Shell detection
runtime.GOOS  // Conditional logic per platform
```

### Platform-specific Code

```go
// Shell integration
if runtime.GOOS == "windows" {
    // PowerShell or cmd.exe
} else {
    // Bash/Zsh/Fish
}

// Binary naming
if runtime.GOOS == "windows" {
    name += ".exe"
}
```

## Testing Architecture

### Test Organization

- Unit tests: `*_test.go` alongside implementation
- Test package: `package_test` for public API
- Test helpers: Shared fixtures and mocks

### Test Coverage Goals

- Core logic: 80%+ coverage
- Critical paths: 100% coverage (install, use, download)
- Edge cases: Comprehensive error handling tests

## Deployment Architecture

### Distribution

```
GitHub Releases
    ↓ provides
Pre-built binaries for all platforms
    ↓ installed via
Installation scripts (install.sh, install.ps1, install.bat)
    ↓ placed in
~/.govman/bin/govman
    ↓ added to
User's PATH
```

### Update Mechanism

```
govman selfupdate
    ↓ queries
GitHub API (latest release)
    ↓ downloads
New binary
    ↓ replaces
Old binary (with backup)
    ↓ verifies
New version works
    ↓ removes
Backup
```

## Scalability

### Personal Use (Design Goal)

- Manages 5-10 Go versions efficiently
- Handles daily version switching
- Fast enough for interactive use

### Not Designed For

- Enterprise-wide deployment (no central management)
- Hundreds of installations (filesystem limits)
- Concurrent multi-user on same account

## Future Architecture Considerations

**Potential Enhancements**:
- Plugin system for extensibility
- Remote version cache sharing
- Integration with IDEs
- API mode for programmatic access

**Constraints**:
- Must remain simple
- No breaking changes to core UX
- Maintain cross-platform support
- Keep binary size reasonable
