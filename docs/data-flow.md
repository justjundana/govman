# Data Flow

Understanding how data moves through govman during key operations.

## Overview

govman follows a layered architecture where data flows through:

```
CLI Layer → Manager Layer → Service Layer → External Resources
```

Each layer has specific responsibilities and transforms data as needed.

## Installation Flow

### Complete Installation Sequence

```
User Command: govman install 1.21.5
    ↓
[CLI Layer]
    Parse arguments → Validate version format
    ↓
[Manager Layer]
    Check if already installed → Get release info
    ↓
[Go Releases Service]
    Fetch from cache OR API → Parse JSON → Find matching file
    ↓
[Download Service]
    Calculate URL → Download file → Show progress
    ↓
[Verification Service]
    Calculate SHA-256 → Compare with expected
    ↓
[Extraction Service]
    Create version directory → Extract archive
    ↓
[Symlink Service]
    Update 'current' symlink (if requested)
    ↓
[Shell Integration]
    Update PATH (optional)
    ↓
Success Message → User
```

### Detailed Steps

#### 1. Command Parsing (CLI)

```go
// Input: ["install", "1.21.5", "--set-default"]
cmd.Flags().StringP("version", "v", "", "version to install")
cmd.Flags().BoolP("set-default", "d", false, "set as default")

// Output: ParsedCommand{
//   Action: "install"
//   Version: "1.21.5"
//   SetDefault: true
// }
```

#### 2. Version Resolution (Manager)

```go
// Input: "1.21.5"
version := normalizeVersion("1.21.5") // "go1.21.5"

// Check if installed
if _, err := os.Stat(filepath.Join(cfg.InstallDir, version)); err == nil {
    return ErrAlreadyInstalled
}

// Output: Validated version "go1.21.5"
```

#### 3. Release Information (Golang Service)

```go
// Fetch releases (cache or API)
releases, err := fetchReleases()

// Find matching release
release := findRelease(releases, "go1.21.5")
// Output: Release{
//   Version: "go1.21.5"
//   Files: [...]
// }

// Find platform-specific file
file := findFile(release.Files, runtime.GOOS, runtime.GOARCH)
// Output: File{
//   Filename: "go1.21.5.linux-amd64.tar.gz"
//   SHA256: "abc123..."
//   Size: 67108864
//   OS: "linux"
//   Arch: "amd64"
// }
```

#### 4. Download (Downloader Service)

```go
// Build download URL
url := fmt.Sprintf("%s/%s", cfg.Mirror.URL, file.Filename)
// "https://go.dev/dl/go1.21.5.linux-amd64.tar.gz"

// Create cache file
cachePath := filepath.Join(cfg.CacheDir, file.Filename)
outFile, _ := os.Create(cachePath)

// Download with progress
resp, _ := http.Get(url)
defer resp.Body.Close()

// Track progress
downloaded := int64(0)
buf := make([]byte, 32*1024)
for {
    n, err := resp.Body.Read(buf)
    if n > 0 {
        outFile.Write(buf[:n])
        downloaded += int64(n)
        progress.Update(downloaded, file.Size)
    }
    if err == io.EOF {
        break
    }
}

// Output: File at ~/.govman/cache/go1.21.5.linux-amd64.tar.gz
```

#### 5. Verification (Downloader Service)

```go
// Calculate checksum
hasher := sha256.New()
file, _ := os.Open(cachePath)
io.Copy(hasher, file)
calculated := hex.EncodeToString(hasher.Sum(nil))

// Compare
if calculated != file.SHA256 {
    return ErrChecksumMismatch
}

// Output: Verified checksum matches
```

#### 6. Extraction (Downloader Service)

```go
// Create version directory
versionDir := filepath.Join(cfg.InstallDir, "go1.21.5")
os.MkdirAll(versionDir, 0755)

// Extract archive
if strings.HasSuffix(cachePath, ".tar.gz") {
    extractTarGz(cachePath, versionDir)
} else {
    extractZip(cachePath, versionDir)
}

// Output: Extracted to ~/.govman/versions/go1.21.5/
```

#### 7. Symlink Update (Manager)

```go
// If --set-default or first installation
currentLink := filepath.Join(cfg.InstallDir, "current")
os.Remove(currentLink)
os.Symlink(versionDir, currentLink)

// Output: ~/.govman/versions/current → go1.21.5
```

#### 8. Shell Integration (Optional)

```go
// Update PATH in shell config
shell := detectShell()
shell.ExecutePathCommand(filepath.Join(currentLink, "bin"))

// Output: PATH includes ~/.govman/versions/current/bin
```

## Version Switching Flow

### Complete Use Sequence

```
User Command: govman use 1.20.5
    ↓
[CLI Layer]
    Parse version argument
    ↓
[Manager Layer]
    Verify version installed → Check symlink
    ↓
[Symlink Service]
    Remove old symlink → Create new symlink
    ↓
[Shell Integration]
    Update PATH (if auto-switch enabled)
    ↓
Success Message → User
```

### Detailed Steps

#### 1. Version Validation

```go
// Input: "1.20.5"
version := normalizeVersion("1.20.5") // "go1.20.5"

// Check installation
versionPath := filepath.Join(cfg.InstallDir, version)
if _, err := os.Stat(versionPath); os.IsNotExist(err) {
    return ErrVersionNotInstalled
}

// Output: Valid installed version "go1.20.5"
```

#### 2. Symlink Update

```go
// Current symlink path
currentLink := filepath.Join(cfg.InstallDir, "current")

// Read old target
oldTarget, _ := os.Readlink(currentLink)
logger.Info("Switching from %s to %s", filepath.Base(oldTarget), version)

// Remove and recreate
os.Remove(currentLink)
os.Symlink(versionPath, currentLink)

// Output: Symlink updated
```

#### 3. PATH Update

```go
// If in same terminal session
binPath := filepath.Join(currentLink, "bin")
os.Setenv("PATH", binPath + ":" + os.Getenv("PATH"))

// Output: Updated PATH for current session
```

## Auto-Switch Flow

### Directory-based Version Switching

```
User: cd /path/to/project
    ↓
[Shell Hook]
    chpwd hook triggered (Zsh) OR PROMPT_COMMAND (Bash)
    ↓
[govman_auto_switch function]
    Read .govman-version file
    Check current Go version
    ↓
[If version differs]
    Call: govman use "$required_version"
    ↓
[Manager Layer]
    Update symlink to new version
    ↓
[Shell]
    Update PATH
    ↓
New version active
```

### Detailed Steps

#### 1. Hook Execution

```bash
# Zsh: ~/.zshrc
govman_auto_switch() {
    if [[ -f .govman-version ]]; then
        local required_version=$(cat .govman-version 2>/dev/null | tr -d '\n\r')
        local current_version=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')
        if [[ "$current_version" != "$required_version" ]]; then
            govman use "$required_version" >/dev/null 2>&1
        fi
    fi
}
add-zsh-hook chpwd govman_auto_switch
```

```bash
# Bash: ~/.bashrc
govman_auto_switch() {
    if [[ -f .govman-version ]]; then
        local required_version=$(cat .govman-version 2>/dev/null | tr -d '\n\r')
        local current_version=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')
        if [[ "$current_version" != "$required_version" ]]; then
            govman use "$required_version" >/dev/null 2>&1
        fi
    fi
}
PROMPT_COMMAND="__govman_check_dir_change; $PROMPT_COMMAND"
```

#### 2. Version File Reading

```go
// Read .govman-version
cwd, _ := os.Getwd()
versionFile := filepath.Join(cwd, ".govman-version")

content, err := ioutil.ReadFile(versionFile)
if os.IsNotExist(err) {
    return // No version file, keep current
}

requestedVersion := strings.TrimSpace(string(content))
// Output: "1.21.5"
```

#### 3. Current Version Check

```go
// Get current version
current, _ := manager.Current()

// Compare
if current == requestedVersion {
    return // Already on correct version
}

// Switch needed
```

#### 4. Silent Switch

```bash
# Shell redirects output to suppress messages
govman use "$required_version" >/dev/null 2>&1

# Output: Suppressed in shell (no output)
```

## List Versions Flow

### Remote Versions

```
User Command: govman list --remote
    ↓
[CLI Layer]
    Parse --remote flag
    ↓
[Manager Layer]
    Call ListRemote()
    ↓
[Go Releases Service]
    Check cache validity (1 hour default)
    ↓
    If expired:
        Fetch https://go.dev/dl/?mode=json
        Parse JSON
        Cache result
    Else:
        Read from cache
    ↓
[CLI Layer]
    Format output → Display table
    ↓
User sees available versions
```

### Local Versions

```
User Command: govman list
    ↓
[CLI Layer]
    No --remote flag
    ↓
[Manager Layer]
    Call ListInstalled()
    ↓
[File System]
    Read ~/.govman/versions directory
    Filter directories (go*)
    Get metadata (size, date)
    ↓
[Symlink Service]
    Read 'current' symlink
    Mark active version
    ↓
[CLI Layer]
    Format output → Display table with marker
    ↓
User sees installed versions
```

## Configuration Flow

### Initial Setup

```
User Command: govman init
    ↓
[CLI Layer]
    Detect shell
    ↓
[Config Service]
    Check ~/.govman/config.yaml exists
    If not:
        Create with defaults
        Set install_dir, cache_dir
    ↓
[Shell Service]
    Detect shell (bash/zsh/fish/pwsh)
    Read shell config file
    ↓
    If govman not configured:
        Append initialization code
        Add PATH modification
        Add auto-switch hook
    ↓
Success Message → User must reload shell
```

### Configuration Loading

```
Application Start
    ↓
[Config Service]
    Search for config file:
        1. --config flag
        2. ~/.govman/config.yaml
        3. Use defaults
    ↓
    Load YAML
    ↓
    Parse and validate
    ↓
    Expand paths (~, $HOME)
    ↓
    Create directories if needed
    ↓
Config available to all services
```

## Error Flow

### Download Failure

```
Network Error during download
    ↓
[Downloader]
    Retry with exponential backoff
    Attempts: 3 (configurable)
    ↓
    Still failing?
    ↓
[Manager]
    Catch error
    Clean up partial download
    ↓
[CLI]
    Display error
    Show troubleshooting hints
    ↓
User sees helpful error message
```

### Verification Failure

```
Checksum mismatch
    ↓
[Downloader]
    Log expected vs actual hash
    ↓
[Manager]
    Delete corrupted file
    Suggest retry
    ↓
[CLI]
    Display verification error
    Suggest checking network/proxy
    ↓
User informed of corruption
```

## Data Caching

### Release Information Cache

```
Request for releases
    ↓
Check cache:
    Path: ~/.govman/cache/releases.json
    Age: < 1 hour (default)
    ↓
    If valid cache:
        Read from disk
        Parse JSON
        Return
    ↓
    If expired/missing:
        Fetch from go.dev
        Parse JSON
        Write to cache
        Return
```

### Download Cache

```
Request to install version
    ↓
Check cache:
    Path: ~/.govman/cache/go1.21.5.linux-amd64.tar.gz
    ↓
    If exists:
        Verify checksum
        If valid:
            Skip download
            Use cached file
        If invalid:
            Delete
            Re-download
    ↓
    If not exists:
        Download
        Verify
        Keep in cache
```

## Concurrent Operations

### Thread Safety

```go
// Manager uses mutex for thread safety
type Manager struct {
    mu     sync.RWMutex
    config *Config
}

func (m *Manager) Install(version string) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    // ... installation logic
}
```

### Progress Updates

```go
// Progress bars use channels
type Progress struct {
    updateChan chan ProgressUpdate
}

// Update from download goroutine
go func() {
    for {
        select {
        case update := <-p.updateChan:
            p.render(update)
        }
    }
}()
```

## Data Transformations

### Version Normalization

```
Input          → Normalized
"1.21.5"       → "go1.21.5"
"go1.21.5"     → "go1.21.5"
"1.21"         → "go1.21.0"
"latest"       → "go1.22.0" (fetched)
"stable"       → "go1.21.5" (fetched)
```

### Path Transformations

```
Input                → Expanded
"~/.govman"          → "/Users/user/.govman"
"$HOME/go/versions"  → "/Users/user/go/versions"
"./versions"         → "/current/dir/versions"
```

### Size Formatting

```
Bytes      → Human Readable
67108864   → "64.0 MB"
1073741824 → "1.0 GB"
1234567    → "1.2 MB"
```

## See Also

- [Architecture](architecture.md) - System design
- [Project Structure](project-structure.md) - Code organization
- [Architecture Diagrams](architecture-diagrams.md) - Visual representations

---

Understanding data flow helps debug issues and optimize performance! 🔄
