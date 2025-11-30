# Commands Reference

Complete reference for all govman commands and options.

## Global Flags

These flags work with all commands:

```bash
--config string   # Config file path (default: ~/.govman/config.yaml)
--verbose         # Enable verbose output
--quiet           # Suppress all output except errors
--help, -h        # Show help
--version         # Show govman version
```

## Commands

### govman

Display help and version information.

```bash
govman            # Show banner and usage
govman --version  # Show version
govman --help     # Show help
```

###govman init

Initialize shell integration for automatic version switching.

```bash
govman init [flags]
```

**Flags:**
- `--force, -f`: Force re-initialization (overwrite existing configuration)
- `--shell string`: Target specific shell (bash, zsh, fish, powershell)

**Examples:**
```bash
govman init
govman init --force
govman init --shell zsh
```

**What it does:**
- Detects or uses specified shell
- Adds integration code to shell config file
- Sets up PATH and environment variables
- Enables automatic version switching

### govman install

Install one or more Go versions.

```bash
govman install [version...] [flags]
```

**Arguments:**
- `version`: Go version to install (`latest`, `1.25.1`, `1.25`, etc.)
- Can install multiple versions: `govman install 1.25.1 1.24.0`

**Examples:**
```bash
govman install latest              # Latest stable
govman install 1.25.1              # Specific version
govman install 1.25                # Latest 1.25.x patch
govman install 1.25.1 1.24.0       # Multiple versions
govman install 1.25rc1             # Pre-release
```

**Features:**
- Lightning-fast parallel downloads with resume capability
- Automatic integrity verification and checksum validation
- Smart caching to avoid re-downloading
- Batch installation with progress tracking

### govman uninstall

Remove one or more installed Go versions.

```bash
govman uninstall [version...] [flags]
```

**Aliases:** `remove`, `rm`

**Arguments:**
- `version`: Go version(s) to uninstall
- Can uninstall multiple versions: `govman uninstall 1.24.1 1.24.2 1.24.3`

**Examples:**
```bash
govman uninstall 1.24.0              # Single version
govman uninstall 1.24.1 1.24.2       # Multiple versions
govman remove 1.23.0
govman rm 1.22.0
govman rm 1.21.1 1.22.0 1.23.0       # Batch removal
```

**Features:**
- Batch uninstallation with progress tracking for each version
- Displays total disk space freed across all versions
- Continues processing remaining versions if one fails
- Comprehensive summary output showing successes and failures

**Safety features:**
- Prevents removal of currently active version
- Confirms version exists before removal
- Automatic recalculation of disk space

### govman use

Switch to a specific Go version.

```bash
govman use <version> [flags]
```

**Arguments:**
- `version`: Go version to activate (`1.25.1`, `latest`, `default`)

**Flags:**
- `--default, -d`: Set as system-wide default (persistent)
- `--local, -l`: Set as project-local version (creates `.govman-goversion`)

**Examples:**
```bash
govman use 1.25.1                 # Session-only
govman use 1.25.1 --default       # System default
govman use 1.25.1 --local         # Project-specific
govman use latest                 # Use latest installed
govman use default                # Use system default
```

**Activation modes:**
- **Session-only**: Temporary, current terminal only
- **System default**: Permanent across all new sessions
- **Project-local**: Tied to specific directory

### govman current

Display current Go version information.

```bash
govman current [flags]
```

**Examples:**
```bash
govman current
```

**Output includes:**
- Version number and release status
- Installation path and size
- Platform architecture details
- Installation date and source
- Activation method

**Example output:**
```
Current Go Environment:
──────────────────────────────────────────────────
Version:         Go 1.25.1
Install Path:    /home/user/.govman/versions/go1.25.1
Platform:        linux/amd64
Installed:       2025-01-15 14:30:45 MST
Disk Usage:      404 MB
Activation:      system-default
──────────────────────────────────────────────────
Run 'go version' to verify your Go installation
```

### govman list

List installed or available Go versions.

```bash
govman list [flags]
```

**Aliases:** `ls`

**Flags:**
- `--remote, -r`: List available versions from official releases
- `--stable-only`: Show only stable versions (remote only)
- `--beta`: Include beta/rc versions (remote only)
- `--pattern string`: Filter versions using glob patterns (remote only)

**Examples:**
```bash
govman list                        # Installed versions
govman list --remote               # Available stable versions
govman list --remote --beta        # Include pre-releases
govman list --remote --pattern "1.25*"  # Filter by pattern
```

**Installed versions output:**
```
Installed Go Versions (3 total):
────────────────────────────────────────────────────────────
→ Active %-25s     89 MB   installed: 2025-01-15
  Installed 1.24.0 [default]        103 MB  installed: 2024-12-01
  Installed 1.23.5                   98 MB  installed: 2024-11-10
────────────────────────────────────────────────────────────
Total disk usage: 290 MB across 3 versions
Currently active: Go 1.25.1
```

### govman info

Display detailed information about a specific Go version.

```bash
govman info <version> [flags]
```

**Arguments:**
- `version`: Go version to query

**Examples:**
```bash
govman info 1.25.1
```

**Output includes:**
- Version number and status (installed/active)
- Platform architecture
- Complete installation path
- Installation date and age
- Disk usage
- Binary locations
- Suggested actions

### govman clean

Clean download cache and optimize disk usage.

```bash
govman clean [flags]
```

**Examples:**
```bash
govman clean
```

**What gets cleaned:**
- Downloaded Go archive files (.tar.gz, .zip)
- Temporary extraction directories
- Incomplete or corrupted downloads
- Obsolete cache metadata

**What's preserved:**
- Installed Go versions
- Configuration files
- Project `.govman-goversion` files

### govman selfupdate

Update govman to the latest version.

```bash
govman selfupdate [flags]
```

**Flags:**
- `--check`: Check for updates without installing
- `--force`: Force update even if already on latest
- `--prerelease`: Include pre-release versions

**Examples:**
```bash
govman selfupdate                  # Update to latest
govman selfupdate --check          # Check only
govman selfupdate --prerelease     # Include beta/rc
govman selfupdate --force          # Force reinstall
```

**Features:**
- Automatic platform detection
- Safe backup and rollback on failure
- Integrity verification
- Release notes display

### govman refresh

Manually trigger version switching based on current directory.

```bash
govman refresh [flags]
```

**Examples:**
```bash
govman refresh
```

**Purpose:**
- Re-evaluate current directory for `.govman-goversion`
- Switch to appropriate version (local or default)
- Useful after adding/removing `.govman-goversion` files

**Behavior:**
- If `.govman-goversion` exists: switch to that version
- If no `.govman-goversion`: switch to default version
- Equivalent to auto-switch that happens on `cd`

## Version Resolution

govman supports flexible version specifications:

| Input      | Resolves To                    | Example       |
|------------|--------------------------------|---------------|
| `latest`   | Latest stable release          | 1.25.1        |
| `1.25`     | Latest 1.25.x patch            | 1.25.1        |
| `1.25.1`   | Exact version                  | 1.25.1        |
| `1.25rc1`  | Specific pre-release           | 1.25rc1       |
| `default`  | Configured default version     | (from config) |

## Exit Codes

| Code | Meaning                              |
|------|--------------------------------------|
| 0    | Success                              |
| 1    | General error                        |
| 2    | Invalid arguments or usage           |
| 3    | Version not found or not installed   |
| 4    | Network or download error            |
| 5    | Checksum verification failed         |
| 6    | Permission denied                    |

## Environment Variables

govman respects these environment variables:

```bash
HTTP_PROXY        # HTTP proxy server
HTTPS_PROXY       # HTTPS proxy server
NO_PROXY          # Proxy bypass list
GOVMAN_CONFIG     # Custom config file path
```

## Configuration File Commands

```bash
# View current config
cat ~/.govman/config.yaml

# Edit config
nano ~/.govman/config.yaml

# Reset to defaults
rm ~/.govman/config.yaml
govman --version  # Recreates with defaults
```

## Shell Integration Commands

These are shell functions/aliases created by `govman init`:

```bash
govman_auto_switch           # Manually trigger auto-switch
type govman                  # Show wrapper function
type govman_auto_switch      # Show auto-switch function
```

## Common Command Combinations

### Install and Activate

```bash
govman install latest && govman use latest --default
```

### List and Install Specific Version

```bash
govman list --remote --pattern "1.25*"
govman install 1.25.1
govman use 1.25.1 --default
```

### View Then Remove Old Version

```bash
govman list
govman info 1.23.0
govman uninstall 1.23.0
```

### Update All Components

```bash
govman selfupdate                    # Update govman
govman list --remote                 # Check for new Go versions
govman install 1.26.0                # Install new version
govman use 1.26.0 --default          # Activate it
```

### Check Current Setup

```bash
govman current                       # Current version info
govman list                          # All installed versions
go version                           # Verify active version
go env                               # Full Go environment
```

## Tips & Tricks

### Quick Installation Workflow

```bash
# One-liner: install and activate latest
govman install latest && govman use latest --default && go version
```

### Project Setup

```bash
# Set version for project
cd /path/to/project
govman use 1.25.1 --local
git add .govman-goversion
git  commit -m "Set Go version"
```

### Batch Management

```bash
# Install multiple versions
govman install 1.25.1 1.24.0 1.23.5

# Uninstall multiple versions
govman uninstall 1.23.5 1.22.0 1.21.1

# List with status
govman list | grep -E "(Active|default)"
```

### Cleanup Routine

```bash
# Free up disk space
govman list                              # See what's installed
govman uninstall 1.23.0 1.22.0 1.21.0    # Remove multiple old versions
govman clean                             # Clean download cache
```
