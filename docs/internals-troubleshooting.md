# Internals

Deep dive into govman's internal implementation details.

## Core Packages

### internal/manager

The heart of govman, coordinating all version management operations.

#### Manager Structure

```go
type Manager struct {
    config     *config.Config
    downloader *downloader.Downloader
    shell      shell.Shell
}
```

#### Key Responsibilities

- **Version Installation**: Coordinates download, verification, and extraction
- **Version Switching**: Updates symlinks and configuration
- **Version Resolution**: Translates "latest", "1.25" to exact versions
- **State Management**: Tracks installed versions and active version

#### Critical Methods

**ResolveVersion**: Handles version string normalization
- `"latest"` → queries API for newest stable release
- `"1.25"` → finds latest 1.25.x patch version
- `"1.25.1"` → validates exact version exists

**Install**: Multi-step installation process
1. Resolve version string
2. Check if already installed
3. Get download metadata from API
4. Download via Downloader
5. Verify installation success

**Use**: Version activation with three modes
- Session-only: No persistence, PATH update only
- Default: Updates config.yaml, creates global symlink
- Local: Writes .govman-goversion file in current directory

### internal/downloader

Handles all download and extraction logic.

#### Download Strategy

**Intelligent Caching**:
```go
cachePath := filepath.Join(cacheDir, filename)
if fileExists(cachePath) && sizeMatches(cachePath, expectedSize) {
    return useCachedFile(cachePath)
}
```

**Resume Support**:
- Uses HTTP Range header for partial downloads
- Appends to existing partial file
- Continues from last byte received

**Parallel Downloads** (configurable):
- Multiple HTTP connections
- Chunks downloaded concurrently
- Progress aggregated across connections

#### Checksum Verification

```go
calculated := sha256.Sum256(fileBytes)
calculatedHex := hex.EncodeToString(calculated[:])
if calculatedHex != expectedChecksum {
    return ErrChecksumMismatch
}
```

#### Archive Extraction

**Tar.gz (Linux/macOS)**:
```go
gzipReader, _ := gzip.NewReader(file)
tarReader := tar.NewReader(gzipReader)

for {
    header, err := tarReader.Next()
    // Extract each file, preserving permissions
}
```

**Zip (Windows)**:
```go
zipReader, _ := zip.OpenReader(archivePath)
for _, file := range zipReader.File {
    // Extract each file
}
```

### internal/golang

Interfaces with Go's official releases API.

#### API Integration

**Endpoint**: `https://go.dev/dl/?mode=json&include=all`

**Response Caching**:
```go
type cachedReleases struct {
    releases []Release
    fetchedAt time.Time
    expiry    time.Duration
}

func (c *cachedReleases) isValid() bool {
    return time.Since(c.fetchedAt) < c.expiry
}
```

**Cache Expiry**: 10 minutes (configurable in config)

#### Version Comparison Algorithm

```go
func CompareVersions(v1, v2 string) int {
    // 1. Normalize versions (remove "go" prefix)
    // 2. Parse into major.minor.patch
    // 3. Compare numbers
    // 4. Compare prerelease tags if numbers equal
    // Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal
}
```

**Prerelease Ranking**:
1. Stable (no suffix) - highest
2. RC (release candidate)
3. Beta
4. α Alpha - lowest

### internal/config

Configuration management with validation.

#### Configuration Loading

```go
func Load() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("~/.govman")
    
    if err := viper.ReadInConfig(); err != nil {
        // Create default config
        return newDefaultConfig(), nil
    }
    
    var cfg Config
    viper.Unmarshal(&cfg)
    cfg.setDefaults()
    cfg.expandPaths()
    cfg.validate()
    cfg.createDirectories()
    
    return &cfg, nil
}
```

#### Path Expansion

```go
func expandPaths(cfg *Config) error {
    // Expand ~ to home directory
    home, _ := os.UserHomeDir()
    cfg.InstallDir = strings.Replace(cfg.InstallDir, "~", home, 1)
    cfg.CacheDir = strings.Replace(cfg.CacheDir, "~", home, 1)
}
```

#### Path Validation

```go
func validatePath(path string) error {
    // Prevent directory traversal
    if strings.Contains(path, "..") {
        return ErrInvalidPath
    }
    
    // Ensure absolute path
    if !filepath.IsAbs(path) {
        return ErrNotAbsolutePath
    }
    
    return nil
}
```

### internal/shell

Cross-platform shell integration.

#### Shell Detection

**Unix/Linux/macOS**:
```go
func Detect() Shell {
    shellEnv := os.Getenv("SHELL")
    
    if strings.Contains(shellEnv, "bash") {
        return &BashShell{}
    } else if strings.Contains(shellEnv, "zsh") {
        return &ZshShell{}
    } else if strings.Contains(shellEnv, "fish") {
        return &FishShell{}
    }
    
    return &BashShell{} // Default
}
```

**Windows**:
```go
func Detect() Shell {
    // Check if running in PowerShell
    if len(os.Getenv("PSModulePath")) > 0 {
        return &PowerShell{}
    }
    
    return &CmdShell{} // Default to cmd
}
```

#### Integration Code Generation

**Template-based**:
```go
const bashTemplate = `
# GOVMAN - Go Version Manager
export PATH="{{.BinPath}}:$PATH"
export GOTOOLCHAIN=local

govman() {
    # Wrapper function implementation
}
# END GOVMAN
`

func generateSetupCode(binPath string) string {
    tmpl := template.Must(template.New("bash").Parse(bashTemplate))
    var buf bytes.Buffer
    tmpl.Execute(&buf, map[string]string{"BinPath": binPath})
    return buf.String()
}
```

### internal/logger

Structured logging with levels and colors.

#### Log Levels

```go
type LogLevel int

const (
    LevelQuiet   LogLevel = 0  // Errors only
    LevelNormal  LogLevel = 1  // Info, success, warnings, errors
    LevelVerbose LogLevel = 2  // + Debug messages
)
```

#### ANSI Color Support

```go
const (
    ColorReset   = "\033[0m"
    ColorRed     = "\033[31m"
    ColorGreen   = "\033[32m"
    ColorYellow  = "\033[33m"
    ColorBlue    = "\033[34m"
    ColorMagenta = "\033[35m"
    ColorCyan    = "\033[36m"
)

func colorize(color, text string) string {
    if !supportsColor() {
        return text
    }
    return color + text + ColorReset
}
```

**Color Detection**:
- Check terminal type (`TERM` environment variable)
- Check if stdout is a terminal (not piped)
- Disable on Windows cmd.exe (unless Windows Terminal)

### internal/progress

Progress bar implementation.

#### Progress Bar Structure

```go
type ProgressBar struct {
    total       int64
    current     int64
    width       int
    description string
    startTime   time.Time
    lastUpdate  time.Time
}
```

#### Rendering

```go
func (p *ProgressBar) Render() string {
    percentage := float64(p.current) / float64(p.total) * 100
    filled := int(percentage / 100 * float64(p.width))
    
    bar := strings.Repeat("█", filled)
    bar += strings.Repeat("░", p.width-filled)
    
    speed := p.calculateSpeed()
    eta := p.calculateETA()
    
    return fmt.Sprintf("%s [%s] %.1f%% %s/s ETA: %s",
        p.description, bar, percentage, formatBytes(speed), formatDuration(eta))
}
```

**Update Throttling**:
- Only redraw every 100ms to avoid flickering
- Force update on completion

### internal/symlink

Symlink creation with cross-platform support.

#### Symlink Creation

```go
func Create(target, link string) error {
    // Remove existing symlink/file
    os.Remove(link)
    
    // Create parent directory
    os.MkdirAll(filepath.Dir(link), 0755)
    
    // Create symlink
    return os.Symlink(target, link)
}
```

**Windows Considerations**:
- Requires Developer Mode or admin rights
- Falls back to directory junction on older Windows
- PowerShell handles PATH correctly with symlinks

## Algorithms

### Version Resolution

```
Input: "1.25"
1. Fetch all available versions from API
2. Filter to 1.25.x versions
3. Sort by semantic version
4. Return highest (e.g., "1.25.1")
```

### Semantic Version Sorting

```go
func sortVersions(versions []string) {
    sort.Slice(versions, func(i, j int) bool {
        return CompareVersions(versions[i], versions[j]) > 0
    })
}
```

## Concurrency and Safety

### Atomic Operations

**Configuration Updates**:
```go
func (c *Config) Save() error {
    tempFile := filepath.Join(os.TempDir(), "govman-config-"+uuid.New()+".yaml")
    
    // Write to temp file
    viper.WriteConfigAs(tempFile)
    
    // Atomic rename
    os.Rename(tempFile, c.path)
}
```

**Symlink Updates**:
- OS-level atomic operation
- Old symlink removed, new one created in single syscall

### No Race Conditions

- Single-threaded command execution
- No shared mutable state
- Each invocation isolated

## Performance Optimizations

### API Response Caching

Avoids redundant API calls:
```go
var releaseCache struct {
    sync.RWMutex
    releases  []Release
    fetchedAt time.Time
}

func GetAvailableVersions() ([]Release, error) {
    releaseCache.RLock()
    if time.Since(releaseCache.fetchedAt) < 10*time.Minute {
        defer releaseCache.RUnlock()
        return releaseCache.releases, nil
    }
    releaseCache.RUnlock()
    
    // Fetch and update cache
}
```

### Download Cache

Persistent file cache:
- Downloads stored in `~/.govman/cache/`
- Reused across installations
- Verified by size before use

### Parallel Downloads

When enabled:
```go
const defaultMaxConnections = 4

func parallelDownload(url string, dest string) error {
    // Split file into chunks
    // Download chunks concurrently
    // Reassemble
}
```

## Error Handling Patterns

### Error Wrapping

```go
if err := download(url); err != nil {
    return fmt.Errorf("failed to download %s: %w", url, err)
}
```

### Retry Logic

```go
func withRetry(operation func() error, maxRetries int, delay time.Duration) error {
    for i := 0; i < maxRetries; i++ {
        if err := operation(); err == nil {
            return nil
        }
        time.Sleep(delay)
    }
    return ErrMaxRetriesExceeded
}
```

## Testing Strategies

### Unit Tests

Each package has comprehensive tests:
- Happy path scenarios
- Error conditions
- Edge cases

### Test Helpers

```go
func createTestConfig(t *testing.T) *Config {
    tmpDir := t.TempDir()
    return &Config{
        InstallDir: filepath.Join(tmpDir, "versions"),
        CacheDir:   filepath.Join(tmpDir, "cache"),
    }
}
```

### Mocking External Dependencies

```go
type mockDownloader struct {
    downloadFunc func(url, dest string) error
}

func (m *mockDownloader) Download(url, dest string) error {
    return m.downloadFunc(url, dest)
}
```
