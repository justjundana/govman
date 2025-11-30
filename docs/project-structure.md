# Project Structure

Overview of govman's codebase structure and organization.

## Directory Layout

```
govman/
├── cmd/
│   └── govman/              # Application entry point
│       └── main.go          # Main function
├── internal/                # Private application code
│   ├── cli/                 # CLI commands and subcommands
│   ├── config/              # Configuration management
│   ├── downloader/          # Download  and extraction logic
│   ├── golang/              # Go releases API integration
│   ├── logger/              # Logging functionality
│   ├── manager/             # Core version management
│   ├── progress/            # Progress bars and indicators
│   ├── shell/               # Shell integration
│   ├── symlink/             # Symlink creation and management
│   ├── util/                # Utility functions
│   └── version/             # Version information
├── scripts/                 # Installation and uninstall scripts
│   ├── install.sh          # Unix installation
│   ├── install.ps1         # PowerShell installation
│   ├── install.bat         # Windows batch installation
│   ├── uninstall.sh        # Unix uninstallation
│   ├── uninstall.ps1       # PowerShell uninstallation
│   └── uninstall.bat       # Windows batch uninstallation
├── Dockerfile              # Docker build configuration
├── Makefile                # Build automation
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
└── README.md               # Project documentation
```

## Package Structure

### cmd/govman

**Purpose**: Application entry point

**Files**:
- `main.go`: Initializes CLI and executes commands

**Responsibilities**:
- Parse command-line arguments
- Handle errors and exit codes

**Dependencies**: `internal/cli`

### internal/cli

**Purpose**: Command-line interface implementation

**Files**:
- `cli.go`: Root command and initialization
- `command.go`: Command registration
- `install.go`: Install and uninstall commands
- `use.go`: Version switching command
- `list.go`: List versions command
- `current.go`: Display current version
- `info.go`: Version information
- `clean.go`: Cache cleanup
- `init.go`: Shell integration setup
- `selfupdate.go`: Self-update functionality
- `refresh.go`: Manual version refresh

**Responsibilities**:
- Define CLI commands and flags
- User input validation
- Command execution flow
- User-facing error messages

**Dependencies**: `manager`, `logger`, `shell`, `config`

### internal/config

**Purpose**: Configuration file management

**Files**:
- `config.go`: Config structure and loading

**Responsibilities**:
- Load configuration from YAML
- Provide default values
- Path expansion and validation
- Save configuration changes

**Key Types**:
```go
type Config struct {
    InstallDir     string
    CacheDir       string
    DefaultVersion string
    Download       DownloadConfig
    Mirror         MirrorConfig
    AutoSwitch     AutoSwitchConfig
    Shell          ShellConfig
    GoReleases     GoReleasesConfig
    SelfUpdate     SelfUpdateConfig
}
```

**Dependencies**: `viper`

### internal/downloader

**Purpose**: Download and extract Go archives

**Files**:
- `downloader.go`: Download orchestration

**Responsibilities**:
- HTTP downloads with retries
- Progress reporting
- SHA-256 checksum verification
- Archive extraction (.tar.gz, .zip)
- Cache management

**Key Functions**:
- `Download()`: Main download orchestration
- `downloadFile()`: HTTP download with resume
- `verifyChecksum()`: SHA-256 verification
- `extractArchive()`: Archive extraction

**Dependencies**: `golang`, `progress`, `logger`, `config`

### internal/golang

**Purpose**: Go releases API integration

**Files**:
- `releases.go` Go releases data fetching and parsing

**Responsibilities**:
- Fetch available Go versions from go.dev API
- Parse release metadata
- Version comparison and sorting
- Cache release information
- Get download URLs for specific versions

**Key Types**:
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

**Key Functions**:
- `GetAvailableVersions()`: List all versions
- `GetDownloadURL()`: Get archive URL
- `CompareVersions()`: Version comparison

**Dependencies**: `net/http`, `encoding/json`

### internal/logger

**Purpose**: Logging and user output

**Files**:
- `logger.go`: Logging implementation

**Responsibilities**:
- Formatted console output
- Log levels (quiet, normal, verbose)
- Colored output support
- Progress indicators
- Error formatting with help messages

**Key Functions**:
- `Info()`, `Success()`, `Warning()`, `Error()`
- `Verbose()`, `Debug()`
- `Progress()`, `Download()`, `Extract()`, `Verify()`
- `ErrorWithHelp()`

**Dependencies**: `viper`

### internal/manager

**Purpose**: Core version management logic

**Files**:
- `manager.go`: Manager implementation

**Responsibilities**:
- Install Go versions
- Uninstall versions
- Switch between versions
- List installed/remote versions
- Version resolution (latest, partial versions)
- Symlink management
- Project-local version files

**Key Type**:
```go
type Manager struct {
    config     *config.Config
    downloader *downloader.Downloader
    shell      shell.Shell
}
```

**Key Functions**:
- `Install()`, `Uninstall()`: Version installation
- `Use()`: Version activation
- `Current()`, `CurrentGlobal()`: Get active version
- `ListInstalled()`, `ListRemote()`: Version listing
- `ResolveVersion()`: Version resolution
- `Clean()`: Cache cleanup

**Dependencies**: All other internal packages

### internal/progress

**Purpose**: Progress bars for downloads

**Files**:
- `progress.go`: Progress bar implementation

**Responsibilities**:
- Display progress bars
- Calculate download speed
- Estimate time remaining (ETA)
- Update display efficiently

**Key Type**:
```go
type ProgressBar struct {
    total       int64
    current     int64
    width       int
    description string
}
```

**Dependencies**: `util` (for formatting)

### internal/shell

**Purpose**: Shell integration and auto-switching

**Files**:
- `shell.go`: Shell detection and configuration

**Responsibilities**:
- Detect user's shell
- Generate shell integration code
- Support multiple shells (Bash, Zsh, Fish, PowerShell, Cmd)
- PATH command generation
- Configuration file modification

**Key Interfaces/Types**:
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

**Implementations**:
- `BashShell`
- `ZshShell`
- `FishShell`
- `PowerShell`
- `CmdShell`

**Key Functions**:
- `Detect()`: Auto-detect shell
- `InitializeShell()`: Setup integration
- `GetShellInstructions()`: Manual setup guide

**Dependencies**: `os`, `template`

### internal/symlink

**Purpose**: Symlink creation

**Files**:
- `symlink.go`: Symlink utilities

**Responsibilities**:
- Create symlinks pointing to Go binaries
- Remove existing symlinks
- Cross-platform symlink support

**Key Functions**:
- `Create()`: Create or update symlink

**Dependencies**: Standard library only

### internal/util

**Purpose**: Utility functions

**Files**:
- `format.go`: Formatting helpers

**Responsibilities**:
- Format byte sizes (KB, MB, GB)
- Format durations
- Common string operations

**Key Functions**:
- `FormatBytes()`: Human-readable file sizes
- `FormatDuration()`: Human-readable durations

**Dependencies**: Standard library only

### internal/version

**Purpose**: Version information embedding

**Files**:
- `version.go`: Build version info

**Responsibilities**:
- Store version number
- Build metadata (commit, date, builder)
- Version display formatting

**Key Variables**:
```go
var (
    Version = "dev"
    Commit  = "none"
    Date    = "unknown"
    BuildBy = "unknown"
)
```

**Set at build time** via `-ldflags`.

**Dependencies**: Standard library only

## Scripts Directory

### install.sh (Unix)

- Bash installation script
- Platform detection (Linux/macOS, amd64/arm64)
- Binary download from GitHub releases
- PATH configuration
- Shell integration setup

### install.ps1 (PowerShell)

- PowerShell installation script
- Windows platform detection
- Binary download
- User PATH update via registry
- PowerShell profile configuration

### install.bat (Windows Batch)

- Command Prompt installation script
- Simplified Windows installation
- Limited functionality compared to PowerShell version

### uninstall.sh (Unix)

- Bash uninstall script
- Two removal modes: minimal and complete
- Shell configuration cleanup
- PATH removal

### uninstall.ps1 (PowerShell)

- PowerShell uninstall script
- Registry PATH cleanup
- Profile configuration removal

### uninstall.bat (Windows Batch)

- Command Prompt uninstall script
- Windows uninstallation

## Build Files

### Makefile

Build automation for Unix-like systems:

```makefile
build          # Build for current platform
install        # Install to ~/.govman/bin
clean          # Clean built artifacts
test           # Run tests
test-coverage  # Run tests with coverage
lint           # Run linters
fmt            # Format code
vet            # Run go vet
release        # Build for all platforms
```

### Dockerfile

Container build configuration for testing/development.

## Dependency Management

### go.mod

Defines Go module and dependencies:
- `github.com/spf13/cobra`: CLI framework
- `github.com/spf13/viper`: Configuration management

### go.sum

Cryptographic checksums of dependencies for verification.

## Code Organization Principles

1. **Internal packages**: All application code is in `internal/` (not importable by other projects)
2. **Single responsibility**: Each package has a clear, focused purpose
3. **Dependency direction**: Flow from `cmd` → `cli` → `manager` → core packages
4. **Minimal dependencies**: Limited external dependencies
5. **Standard library first**: Prefer standard library over third-party packages
6. **Cross-platform**: Code works on Linux, macOS, Windows

## Testing Structure

Each package has corresponding test files:
- `package_test.go`: Tests for `package.go`
- Test files are co-located with implementation
- Use `package_test` to test public API
- Use `package` to test internals

## Configuration Files

Runtime configuration stored in:
- `~/.govman/config.yaml`: User configuration
- `~/.govman/versions/`: Installed Go versions
- `~/.govman/cache/`: Download cache
- `.govman-goversion`: Project version file
