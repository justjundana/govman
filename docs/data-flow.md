# Data Flow

How data flows through govman during various operations.

## Overview

govman follows a layered architecture:

```
User (CLI) → CLI Commands → Manager → Core Services → External APIs/Filesystem
```

## Install Flow

### User Command

```bash
govman install 1.25.1
```

### Flow Diagram

```
1. CLI (install.go)
   ↓ calls
2. Manager.Install(version)
   ↓ resolves version
3. Golang.GetDownloadURL(version)
   ↓ fetches from API
4. go.dev API
   ↓ returns metadata
5. Downloader.Download(url, installDir, version)
   ↓ downloads
6. HTTP GET official Go archive
   ↓ streams to
7. Cache (~/.govman/cache/)
   ↓ verifies
8. SHA-256 checksum
   ↓ extracts to
9. Install directory (~/.govman/versions/go1.25.1/)
   ↓ success
10. User notified
```

### Detailed Steps

1. **Command Parsing** (`internal/cli/install.go`):
   - Parse version argument
   - Validate input

2. **Version Resolution** (`internal/manager/manager.go`):
   - Resolve "latest" or partial versions
   - Query API for available versions

3. **Download URL Lookup** (`internal/golang/releases.go`):
   - Fetch release metadata from go.dev
   - Find appropriate file for OS/architecture
   - Return download URL

4. **Download** (`internal/downloader/downloader.go`):
   - Check cache for existing file
   - Download with progress tracking
   - Save to cache directory

5. **Verification** (`internal/downloader/downloader.go`):
   - Calculate SHA-256 of downloaded file
   - Compare with official checksum
   - Reject if mismatch

6. **Extraction** (`internal/downloader/downloader.go`):
   - Extract .tar.gz or .zip
   - Copy files to install directory
   - Set appropriate permissions

7. **Completion**:
   - Remove temporary files
   - Log success message

## Use Flow

### User Command

```bash
govman use 1.25.1 --default
```

### Flow Diagram

```
1. CLI (use.go)
   ↓ calls
2. Manager.Use(version, setDefault=true, setLocal=false)
   ↓ validates
3. Check if version installed
   ↓ if yes
4. Update configuration (DefaultVersion)
   ↓ saves
5. Config file (~/.govman/config.yaml)
   ↓ creates/updates
6. Symlink (~/.govman/bin/go → ~/.govman/versions/go1.25.1/bin/go)
   ↓ outputs
7. PATH export command (for shell wrapper)
   ↓ success
8. User notified
```

### Detailed Steps

1. **Command Parsing** (`internal/cli/use.go`):
   - Parse version and flags
   - Determine activation mode

2. **Validation** (`internal/manager/manager.go`):
   - Check if version is installed
   - Resolve version if needed

3. **Configuration Update** (if `--default`):
   - Update `config.DefaultVersion`
   - Save to `~/.govman/config.yaml`

4. **Local Version File** (if `--local`):
   - Write version to `.govman-goversion`
   - In current directory

5. **Symlink Creation** (if `--default`):
   - Create `~/.govman/bin/go` symlink
   - pointing to version's `bin/go`

6. **PATH Update** (always):
   - Generate PATH command
   - Output for shell wrapper to eval

## List Flow

### User Command

```bash
govman list --remote
```

### Flow Diagram

```
1. CLI (list.go)
   ↓ calls
2. Manager.ListRemote(includeUnstable=false)
   ↓ calls
3. Golang.GetAvailableVersions(includeUnstable)
   ↓ checks cache
4. In-memory cache (if valid)
   ↓ or fetches from
5. go.dev API
   ↓ parses JSON
6. Release data
   ↓ sorts versions
7. Sorted version list
   ↓ returns to
8. CLI formatter
   ↓ displays
9. User output
```

## Auto-Switch Flow

### Trigger

```bash
cd /path/to/project  # With .govman-goversion file
```

### Flow Diagram

```
1. Shell hook (chpwd/PROMPT_COMMAND/etc.)
   ↓ triggers
2. govman_auto_switch() function
   ↓ checks config
3. ~/.govman/config.yaml (auto_switch.enabled)
   ↓ if enabled, looks for
4. .govman-goversion file in current directory
   ↓ reads version
5. Required version (e.g., "1.25.1")
   ↓ checks current
6. go version output
   ↓ if different
7. govman use <required-version>
   ↓ follows Use Flow
8. Version switched automatically
```

## Download Cache Flow

### Cache Hit

```
Request version
   ↓ check cache
~/.govman/cache/go1.25.1.linux-amd64.tar.gz exists
   ↓ verify size matches expected
Cache hit → Skip download → Use cached file
```

### Cache Miss

```
Request version
   ↓ check cache
File not in cache or size mismatch
   ↓ download
HTTP GET from go.dev
   ↓ save to cache
~/.govman/cache/go1.25.1.linux-amd64.tar.gz
   ↓ use for installation
Extract to install directory
```

## Configuration Flow

### Loading

```
1. Application starts
   ↓ CLI init
2. Config.Load() called
   ↓ checks
3. ~/.govman/config.yaml exists?
   ↓ if no, create with defaults
4. Load YAML file
   ↓ parse
5. Viper unmarshals to Config struct
   ↓ expand paths
6. Resolve ~ in paths
   ↓ validate
7. Check path safety
   ↓ create directories
8. Ensure install/cache dirs exist
   ↓ return
9. Loaded Config available to application
```

### Saving

```
1. Config.Save() called (e.g., after 'use --default')
   ↓ set values
2. Viper.Set() for each field
   ↓ write
3. Viper.WriteConfigAs()
   ↓ saves to
4. ~/.govman/config.yaml
```

## Shell Integration Flow

### Initialization

```bash
govman init
```

### Flow Diagram

```
1. CLI (init.go)
   ↓ detects
2. Shell.Detect() → Current shell type
   ↓ gets setup code
3. Shell.SetupCommands(binPath) → Integration code
   ↓ reads existing
4. Shell config file (~/.bashrc, etc.)
   ↓ removes old
5. Remove existing GOVMAN sections
   ↓ appends new
6. Add new integration code
   ↓ writes
7. Updated shell config file
   ↓ notifies
8. User to reload shell
```

## Self-Update Flow

### User Command

```bash
govman selfupdate
```

### Flow Diagram

```
1. CLI (selfupdate.go)
   ↓ fetches
2. GitHub API (latest release)
   ↓ parses
3. Release metadata
   ↓ compares
4. Current version vs Latest version
   ↓ if newer available
5. Construct download URL for platform
   ↓ downloads
6. New govman binary (temp file)
   ↓ backup
7. Rename current binary to .bak
   ↓ replace
8. Move new binary to current location
   ↓ set permissions
9. chmod +x
   ↓ verify
10. Run new binary --version
   ↓ cleanup
11. Remove backup (if successful)
   ↓ success
12. User notified
```

## Error Handling Flow

### Download Failure Example

```
Download attempt
   ↓ fails (network error)
Retry logic (up to retry_count times)
   ↓ all retries failed
Log error details
   ↓ clean up
Remove partial download
   ↓ return error
Manager catches error
   ↓ formats
CLI formats user-friendly message
   ↓ display
User sees: "Failed to download: network timeout"
   ↓ suggest
Help message: "Check internet connection"
   ↓ exit
Exit code 4 (network error)
```

## Data Persistence

### Filesystem Layout

```
~/.govman/
├── config.yaml              # Configuration (persisted)
├── bin/
│   └── go → versions/go1.25.1/bin/go  # Symlink (persisted)
├── versions/
│   ├── go1.25.1/           # Installed version (persisted)
│   ├── go1.24.0/           # Installed version (persisted)
│   └── ...
└── cache/
    ├── go1.25.1.linux-amd64.tar.gz  # Download cache (temporary)
    └── ...
```

### Ephemeral Data

- API responses (cached in memory for 10 minutes)
- Progress bar state
- Current session PATH modifications

### Persistent Data

- Configuration file
- Installed Go versions
- Download cache (until cleaned)
- Shell configuration (in shell RC files)
- .govman-goversion project files

## Concurrency

govman is primarily single-threaded for simplicity and safety:

### No Concurrent Operations

- Only one `govman` command runs at a time per user
- Configuration updates are atomic (write to temp, then rename)
- No lock files needed (user-space isolation)

### Safe Concurrency

- Multiple users can run govman simultaneously (separate home directories)
- Download cache uses file system atomicity
- Symlink updates are atomic at OS level

## API Interactions

### go.dev API

**Endpoint**: `https://go.dev/dl/?mode=json&include=all`

**Request**:
```
GET https://go.dev/dl/?mode=json&include=all
```

**Response**: JSON array of releases

**Caching**: 10 minutes in-memory

### GitHub API (Self-Update)

**Endpoint**: `https://api.github.com/repos/justjundana/govman/releases/latest`

**Request**:
```
GET https://api.github.com/repos/justjundana/govman/releases/latest
```

**Response**: JSON release object

**Caching**: None (explicit user action)

## Data Transformation

### Version String Normalization

```
User input → "latest"
   ↓ resolve
API query → All stable versions
   ↓ sort
Semantic version comparison
   ↓ select
"1.25.1"
   ↓ use in
Installation/switching
```

### Path Handling

```
Config: install_dir: "~/.govman/versions"
   ↓ expand
Absolute: "/home/user/.govman/versions"
   ↓ validate
Check no ".." traversal
   ↓ use in
File operations
```
