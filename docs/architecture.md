# Architecture

High-level architecture and design principles of govman.

## System Overview

govman is a command-line tool for managing multiple Go versions with automatic version switching support.

### Design Goals

1. **Simple**: Easy to install and use
2. **Fast**: Quick downloads, efficient switching
3. **Reliable**: Verified downloads, atomic operations
4. **Cross-platform**: Works on Linux, macOS, Windows
5. **Shell-agnostic**: Supports Bash, Zsh, Fish, PowerShell

## Architecture Layers

```
┌─────────────────────────────────────────────────┐
│              CLI Layer (Cobra)                  │
│  User Interface, Command Parsing, Output        │
└───────────────────┬─────────────────────────────┘
                    │
┌───────────────────▼─────────────────────────────┐
│           Manager Layer (Orchestrator)          │
│  Business Logic, Workflow Coordination          │
└───┬───────┬──────┬────────┬─────────┬──────────┘
    │       │      │        │         │
┌───▼───┐ ┌▼────┐ ┌▼─────┐ ┌▼──────┐ ┌▼────────┐
│Config │ │Down-│ │Golang│ │Shell  │ │Symlink  │
│       │ │load │ │API   │ │       │ │         │
└───────┘ └─────┘ └──────┘ └───────┘ └─────────┘
                    │
            ┌───────▼───────┐
            │  External     │
            │  Resources    │
            │  (go.dev API) │
            └───────────────┘
```

### Layer Responsibilities

#### 1. CLI Layer (`internal/cli`)

**Purpose**: User interaction and command handling

**Responsibilities**:
- Parse command-line arguments
- Validate user input
- Display formatted output
- Handle errors gracefully
- Show progress indicators

**Key Components**:
- Root command initialization
- Subcommand registration
- Flag parsing
- Output formatting
- Help text generation

**Example**:
```go
// Command definition
var installCmd = &cobra.Command{
    Use:   "install <version>",
    Short: "Install a Go version",
    Args:  cobra.ExactArgs(1),
    Run:   runInstall,
}

func runInstall(cmd *cobra.Command, args []string) {
    version := args[0]
    
    // Call manager
    if err := manager.Install(version); err != nil {
        logger.Error("Installation failed: %v", err)
        os.Exit(1)
    }
    
    logger.Success("Installed Go %s", version)
}
```

#### 2. Manager Layer (`internal/manager`)

**Purpose**: Business logic orchestration

**Responsibilities**:
- Coordinate between services
- Implement workflows
- Handle transactions
- Manage state
- Error recovery

**Key Operations**:
- `Install()` - Download, verify, extract, install
- `Uninstall()` - Remove version and clean up
- `Use()` - Switch active version
- `List()` - Query installed/remote versions
- `Current()` - Get active version

**Example**:
```go
func (m *Manager) Install(version string) error {
    // 1. Validate
    if m.IsInstalled(version) {
        return ErrAlreadyInstalled
    }
    
    // 2. Get release info
    release, err := m.golang.GetRelease(version)
    if err != nil {
        return err
    }
    
    // 3. Download
    if err := m.downloader.Download(release); err != nil {
        return err
    }
    
    // 4. Extract
    if err := m.downloader.Extract(release, m.cfg.InstallDir); err != nil {
        m.downloader.Cleanup(release) // Rollback
        return err
    }
    
    // 5. Set as current (if first install)
    if m.shouldSetCurrent() {
        m.Use(version)
    }
    
    return nil
}
```

#### 3. Service Layer

##### Config Service (`internal/config`)

**Purpose**: Configuration management

**Features**:
- YAML configuration loading
- Default values
- Path expansion
- Validation
- Directory creation

**Configuration Structure**:
```yaml
install_dir: ~/.govman/versions
cache_dir: ~/.govman/cache
default_version: "1.21.5"

download:
  timeout: 300
  retry: 3
  verify_checksum: true

mirror:
  enabled: false
  url: https://golang.google.cn/dl

```yaml
auto_switch:
  enabled: true
  project_file: .govman-version
```
```

##### Downloader Service (`internal/downloader`)

**Purpose**: Download and extract Go distributions

**Features**:
- HTTP downloads with resume support
- SHA-256 verification
- Archive extraction (tar.gz, zip)
- Progress reporting
- Retry with backoff
- Path traversal protection

**Download Flow**:
```go
func (d *Downloader) Download(release Release) error {
    // Build URL
    url := d.buildURL(release)
    
    // Check cache
    cachePath := d.cachePath(release)
    if d.cacheValid(cachePath, release.SHA256) {
        return nil // Already downloaded
    }
    
    // Download with progress
    req := d.buildRequest(url)
    resp, _ := d.client.Do(req)
    defer resp.Body.Close()
    
    // Write to cache
    out, _ := os.Create(cachePath)
    defer out.Close()
    
    progress := progress.New(release.Size)
    io.Copy(io.MultiWriter(out, progress), resp.Body)
    
    // Verify checksum
    if err := d.verifyChecksum(cachePath, release.SHA256); err != nil {
        os.Remove(cachePath)
        return err
    }
    
    return nil
}
```

##### Golang API Service (`internal/golang`)

**Purpose**: Interact with Go releases API

**Features**:
- Fetch available versions
- Parse release metadata
- Version comparison
- Platform-specific file selection
- Cache management

**API Integration**:
```go
func (g *Golang) FetchReleases() ([]Release, error) {
    // Check cache
    if cached := g.cache.Get("releases"); cached != nil {
        return cached, nil
    }
    
    // Fetch from API
    resp, _ := http.Get("https://go.dev/dl/?mode=json")
    defer resp.Body.Close()
    
    var releases []Release
    json.NewDecoder(resp.Body).Decode(&releases)
    
    // Cache for 1 hour
    g.cache.Set("releases", releases, time.Hour)
    
    return releases, nil
}
```

##### Shell Service (`internal/shell`)

**Purpose**: Shell-specific integration

**Features**:
- Shell detection (Bash, Zsh, Fish, PowerShell)
- Configuration file management
- PATH manipulation
- Auto-switch hooks
- Initialization code generation

**Shell Interface**:
```go
type Shell interface {
    Name() string
    ConfigFile() string
    PathCommand(binPath string) string
    SetupCommands(binPath string) []string
    IsAvailable() bool
}
```

**Example Implementation**:
```go
type BashShell struct{}

func (b *BashShell) SetupCommands(binPath string) []string {
    return []string{
        `# govman initialization`,
        fmt.Sprintf(`export PATH="%s:$PATH"`, binPath),
        ``,
        `# govman auto-switch`,
        `govman_auto_switch() {`,
        `    if [[ -f .govman-version ]]; then`,
        `        local required_version=$(cat .govman-version 2>/dev/null)`,
        `        govman use "$required_version" >/dev/null 2>&1`,
        `    fi`,
        `}`,
        `__govman_check_dir_change() {`,
        `    if [[ "$PWD" != "$__govman_prev_pwd" ]]; then`,
        `        __govman_prev_pwd="$PWD"`,
        `        govman_auto_switch`,
        `    fi`,
        `}`,
        `PROMPT_COMMAND="__govman_check_dir_change; $PROMPT_COMMAND"`,
    }
}
```

##### Symlink Service (`internal/symlink`)

**Purpose**: Manage symbolic links

**Features**:
- Create symlinks
- Read symlink targets
- Cross-platform support
- Atomic updates

## Key Design Patterns

### 1. Layered Architecture

Each layer depends only on layers below it:

```
CLI → Manager → Services → External
```

Benefits:
- Clear separation of concerns
- Easy to test individual layers
- Can swap implementations

### 2. Dependency Injection

Services are injected into Manager:

```go
type Manager struct {
    config     *config.Config
    downloader *downloader.Downloader
    golang     *golang.Golang
    shell      *shell.Shell
    logger     *logger.Logger
}

func New(cfg *config.Config) *Manager {
    return &Manager{
        config:     cfg,
        downloader: downloader.New(cfg),
        golang:     golang.New(cfg),
        shell:      shell.New(cfg),
        logger:     logger.Get(),
    }
}
```

Benefits:
- Easy to mock for testing
- Configurable behavior
- Loose coupling

### 3. Strategy Pattern (Shell Integration)

Different shells implement common interface:

```go
type Shell interface {
    Name() string
    SetupCommands(binPath string) []string
}

// Select strategy at runtime
func DetectShell() Shell {
    shell := os.Getenv("SHELL")
    switch {
    case strings.Contains(shell, "bash"):
        return &BashShell{}
    case strings.Contains(shell, "zsh"):
        return &ZshShell{}
    // ...
    }
}
```

### 4. Singleton Pattern (Logger)

Global logger instance:

```go
var (
    globalLogger *Logger
    once         sync.Once
)

func Get() *Logger {
    once.Do(func() {
        globalLogger = New()
    })
    return globalLogger
}
```

### 5. Factory Pattern (Progress Bars)

Create progress bars based on context:

```go
func NewProgress(total int64, mode Mode) Progress {
    switch mode {
    case QuietMode:
        return &SilentProgress{}
    case VerboseMode:
        return &DetailedProgress{total: total}
    default:
        return &StandardProgress{total: total}
    }
}
```

### 6. Builder Pattern (HTTP Requests)

Build complex requests:

```go
func (d *Downloader) buildRequest(url string, resumeFrom int64) *http.Request {
    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("User-Agent", userAgent())
    
    if resumeFrom > 0 {
        req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeFrom))
    }
    
    if d.cfg.Download.Timeout > 0 {
        ctx, _ := context.WithTimeout(context.Background(), 
            time.Duration(d.cfg.Download.Timeout)*time.Second)
        req = req.WithContext(ctx)
    }
    
    return req
}
```

## Error Handling Strategy

### Error Wrapping

Always wrap errors with context:

```go
if err := download(); err != nil {
    return fmt.Errorf("failed to download: %w", err)
}
```

### Error Recovery

Manager handles rollback on failure:

```go
func (m *Manager) Install(version string) error {
    // Download
    if err := m.downloader.Download(release); err != nil {
        return err
    }
    
    // Extract (with rollback)
    if err := m.downloader.Extract(release, m.cfg.InstallDir); err != nil {
        m.downloader.Cleanup(release) // Clean up partial extraction
        return fmt.Errorf("extraction failed: %w", err)
    }
    
    return nil
}
```

### User-Friendly Errors

Logger provides helpful error messages:

```go
logger.ErrorWithHelp(
    "Failed to download Go 1.21.5",
    "Try:\n" +
    "  1. Check your internet connection\n" +
    "  2. Verify proxy settings in config\n" +
    "  3. Try a different mirror",
    map[string]interface{}{
        "error": err.Error(),
        "url":   downloadURL,
    },
)
```

## Concurrency Model

### Thread Safety

Manager uses mutex for critical sections:

```go
type Manager struct {
    mu sync.RWMutex
    // ...
}

func (m *Manager) Install(version string) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    // ... installation logic
}
```

### Concurrent Downloads

Future: Support parallel downloads:

```go
// Download multiple versions concurrently
func (m *Manager) InstallMultiple(versions []string) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(versions))
    
    for _, version := range versions {
        wg.Add(1)
        go func(v string) {
            defer wg.Done()
            if err := m.Install(v); err != nil {
                errChan <- err
            }
        }(version)
    }
    
    wg.Wait()
    close(errChan)
    
    // Check for errors
    for err := range errChan {
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

## State Management

### Version State

Tracked via filesystem:

```
~/.govman/
├── versions/
│   ├── go1.20.5/      ← Installed version
│   ├── go1.21.5/      ← Installed version
│   └── current →      ← Symlink to active version
└── cache/
    └── go1.21.5.linux-amd64.tar.gz
```

### Configuration State

Stored in YAML:

```yaml
# ~/.govman/config.yaml
install_dir: ~/.govman/versions
cache_dir: ~/.govman/cache
default_version: "1.21.5"
```

### Shell State

Maintained in shell config:

```bash
# ~/.bashrc or ~/.zshrc
export PATH="$HOME/.govman/versions/current/bin:$PATH"
PROMPT_COMMAND="govman refresh --silent; $PROMPT_COMMAND"
```

## Performance Considerations

### 1. Caching

- Release data cached for 1 hour
- Downloaded archives kept in cache
- Symlinks for fast switching

### 2. Incremental Operations

- Resume interrupted downloads
- Skip re-download if cached
- Atomic symlink updates

### 3. Efficient Extraction

- Stream extraction (no double disk usage)
- Skip unnecessary files
- Parallel extraction (future)

### 4. Minimal Overhead

- Auto-switch checks are fast (<1ms)
- Lazy initialization
- No background processes

## Security Considerations

### 1. Checksum Verification

Always verify SHA-256:

```go
if !d.verifyChecksum(file, expectedHash) {
    return ErrChecksumMismatch
}
```

### 2. Path Traversal Prevention

Validate all paths:

```go
func isSafe(extractPath, destination string) bool {
    rel, err := filepath.Rel(destination, extractPath)
    return err == nil && !strings.HasPrefix(rel, "..")
}
```

### 3. HTTPS Only

All downloads use HTTPS:

```go
const goDownloadURL = "https://go.dev/dl/"
```

### 4. Minimal Permissions

Files created with restrictive permissions:

```go
os.MkdirAll(dir, 0755)  // rwxr-xr-x
os.Create(file, 0644)    // rw-r--r--
```

## Extensibility

### Adding New Shells

1. Implement `Shell` interface
2. Add detection logic
3. Register in factory

```go
type PowerShellShell struct{}

func (p *PowerShellShell) Name() string {
    return "pwsh"
}

func (p *PowerShellShell) SetupCommands(binPath string) []string {
    return []string{
        `# govman initialization`,
        fmt.Sprintf(`$env:PATH = "%s;" + $env:PATH`, binPath),
    }
}
```

### Adding New Commands

1. Create command file in `internal/cli/`
2. Implement command logic
3. Register in `command.go`

```go
// internal/cli/mycommand.go
var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "Description",
    Run:   runMyCommand,
}

func runMyCommand(cmd *cobra.Command, args []string) {
    // Implementation
}
```

### Adding New Configuration Options

1. Update `Config` struct
2. Add default value
3. Document in `config.yaml.example`

```go
type Config struct {
    // ...
    MyNewOption string `mapstructure:"my_new_option"`
}

func (c *Config) setDefaults() {
    viper.SetDefault("my_new_option", "default_value")
}
```

## See Also

- [Project Structure](project-structure.md) - Code organization
- [Data Flow](data-flow.md) - How data moves
- [Architecture Diagrams](architecture-diagrams.md) - Visual representation

---

Understanding the architecture helps you contribute effectively! 🏗️
