# Commands Reference

Complete reference for all **govman** commands and their options.

## Command Overview

| Command | Description |
|---------|-------------|
| [`init`](#govman-init) | Initialize shell integration |
| [`install`](#govman-install) | Install Go versions |
| [`uninstall`](#govman-uninstall) | Remove Go versions |
| [`use`](#govman-use) | Switch Go versions |
| [`current`](#govman-current) | Show active version |
| [`list`](#govman-list) | List Go versions |
| [`info`](#govman-info) | Show version details |
| [`clean`](#govman-clean) | Clean download cache |
| [`refresh`](#govman-refresh) | Re-evaluate directory context |
| [`selfupdate`](#govman-selfupdate) | Update govman itself |

## Global Flags

Available for all commands:

```bash
--config string    Config file (default: ~/.govman/config.yaml)
--verbose          Verbose output
--quiet            Quiet output (errors only)
--help, -h         Help for any command
--version          Show version information
```

---

## govman init

Initialize shell integration for automatic version switching.

### Usage

```bash
govman init [flags]
```

### Flags

```bash
-f, --force              Force re-initialization (overwrite existing config)
    --shell string       Target specific shell (bash, zsh, fish, powershell)
```

### Description

Sets up your shell environment for govman:
- Adds govman to PATH
- Configures environment variables
- Sets up automatic version switching
- Adds directory change hooks

### Examples

```bash
# Auto-detect shell and initialize
govman init

# Force re-initialization
govman init --force

# Initialize specific shell
govman init --shell zsh
govman init --shell bash
govman init --shell fish
govman init --shell powershell
```

### What It Does

1. Detects your current shell (or uses --shell flag)
2. Adds configuration to shell RC file:
   - Bash: `~/.bashrc` or `~/.bash_profile`
   - Zsh: `~/.zshrc`
   - Fish: `~/.config/fish/config.fish`
   - PowerShell: `$PROFILE`
3. Sets up PATH management
4. Enables auto-switching based on `.govman-version` files

### Post-Initialization

Reload your shell:

```bash
# Bash
source ~/.bashrc

# Zsh
source ~/.zshrc

# Fish
source ~/.config/fish/config.fish

# PowerShell
. $PROFILE
```

---

## govman install

Install one or more Go versions.

### Usage

```bash
govman install <version>... [flags]
```

### Arguments

```bash
<version>    Go version to install (e.g., 1.21.5, latest)
```

### Description

Downloads and installs Go versions from official releases:
- Validates checksums (SHA-256)
- Supports resume for interrupted downloads
- Caches downloads for offline installation
- Installs to `~/.govman/versions/`

### Examples

```bash
# Install latest stable version
govman install latest

# Install specific version
govman install 1.21.5

# Install multiple versions
govman install 1.21.5 1.20.12 1.19.13

# Install pre-release version
govman install 1.22rc1
```

### Version Formats

- `latest` - Latest stable release
- `1.21.5` - Specific version
- `1.21` - Latest patch of 1.21.x
- `1.22rc1` - Pre-release version
- `1.22beta1` - Beta release

### What It Does

1. Resolves version (if using `latest` or partial version)
2. Checks if already installed (skips if present)
3. Downloads archive from `go.dev`
4. Verifies SHA-256 checksum
5. Extracts to `~/.govman/versions/go<version>/`
6. Cleans up temporary files

### Installation Output

```bash
$ govman install 1.21.5
Starting installation of 1 Go version(s)...
Progress: Preparing downloads and verifying version availability
[1/1] Installing Go 1.21.5...
Download: Downloading: go1.21.5.darwin-arm64.tar.gz
Downloading go1.21.5.darwin-arm64.tar.gz [████████████████] 100% (67.2 MB/67.2 MB) 15.2 MB/s
Verify: Verifying checksum...
Success: Checksum verified
Extract: Extracting archive...
Success: Successfully installed Go 1.21.5
──────────────────────────────────────────────────
Success: Successfully installed 1 version(s):
  • Go 1.21.5
All installations completed successfully!
Activate it with: govman use 1.21.5
```

---

## govman uninstall

Remove an installed Go version.

### Usage

```bash
govman uninstall <version> [flags]
```

### Arguments

```bash
<version>    Go version to uninstall (e.g., 1.21.5)
```

### Aliases

```bash
govman remove <version>
govman rm <version>
```

### Description

Completely removes an installed Go version:
- Deletes installation directory
- Frees up disk space
- Cannot uninstall currently active version

### Examples

```bash
# Uninstall specific version
govman uninstall 1.21.5

# Using aliases
govman remove 1.21.5
govman rm 1.21.5
```

### Safety Features

- ✅ Prevents removal of active version
- ✅ Shows disk space to be freed
- ✅ Confirms version exists before removal
- ✅ Preserves other installed versions

### What It Does

1. Checks if version is installed
2. Verifies it's not currently active
3. Removes `~/.govman/versions/go<version>/` directory
4. Reports freed disk space

---

## govman use

Switch to a specific Go version.

### Usage

```bash
govman use <version> [flags]
```

### Arguments

```bash
<version>    Go version to activate (e.g., 1.21.5, default)
```

### Flags

```bash
-d, --default    Set as system-wide default version
-l, --local      Set as project-local version (.govman-version file)
```

### Description

Activates a Go version with different scopes:
- **Session-only**: Temporary, current terminal only
- **System default**: Permanent across all terminals
- **Project-local**: Automatic for specific project

### Examples

```bash
# Switch for current session only
govman use 1.21.5

# Set as system default
govman use 1.21.5 --default

# Set for current project
govman use 1.21.5 --local

# Switch to configured default
govman use default
```

### Activation Modes

#### Session-Only (No Flags)

```bash
govman use 1.21.5
```

- ✅ Affects current terminal only
- ❌ Not preserved in new terminals
- ❌ Not saved to config
- **Use case**: Quick testing

#### System Default (--default)

```bash
govman use 1.21.5 --default
```

- ✅ Affects all new terminals
- ✅ Saved to `~/.govman/config.yaml`
- ✅ Creates/updates symlink
- **Use case**: Primary development version

#### Project-Local (--local)

```bash
govman use 1.21.5 --local
```

- ✅ Creates `.govman-version` file
- ✅ Auto-switches when entering directory
- ✅ Team-sharable (commit to git)
- **Use case**: Project-specific versions

### What It Does

**Session-only:**
1. Updates PATH for current terminal
2. Makes Go 1.21.5 available immediately

**System default:**
1. Updates `~/.govman/config.yaml`
2. Creates symlink: `~/.govman/bin/go` → `versions/go1.21.5/bin/go`
3. Updates PATH

**Project-local:**
1. Creates `.govman-version` with version number
2. Auto-switches when entering directory (if shell integration enabled)

---

## govman current

Display currently active Go version information.

### Usage

```bash
govman current [flags]
```

### Description

Shows detailed information about the active Go version:
- Version number
- Installation path
- Platform (OS/architecture)
- Installation date
- Disk usage
- Activation method

### Examples

```bash
# Show current version
govman current
```

### Output

```bash
Current Go Environment:
──────────────────────────────────────────────────
Version:         Go 1.21.5
Install Path:    /Users/username/.govman/versions/go1.21.5
Platform:        darwin/arm64
Installed:       2024-01-15 10:30:45 PST
Disk Usage:      147.3 MB
Activation:      system-default
──────────────────────────────────────────────────
Run 'go version' to verify your Go installation
```

### Activation Methods

- `session-only` - Temporary terminal activation
- `project-local` - Via `.govman-version` file
- `system-default` - Via default configuration

---

## govman list

List installed or available Go versions.

### Usage

```bash
govman list [flags]
```

### Flags

```bash
-r, --remote          List available versions from Go's official releases
    --stable-only     Show only stable versions (remote only)
    --beta            Include beta/rc versions (remote only)
    --pattern string  Filter versions using glob patterns (remote only)
```

### Aliases

```bash
govman ls
```

### Description

Lists Go versions either:
- **Local**: Installed on your system
- **Remote**: Available for installation

### Examples

```bash
# List installed versions
govman list

# List available versions
govman list --remote

# List only stable versions
govman list --remote --stable-only

# Include pre-releases
govman list --remote --beta

# Filter by pattern
govman list --remote --pattern "1.21*"
govman list --remote --pattern "1.2?"
```

### Output: Installed Versions

```bash
Installed Go Versions (3 total):
────────────────────────────────────────────────────────────
→ Active   1.21.5 [default]        147.3 MB   installed: 2024-01-15
  Installed 1.20.12                143.8 MB   installed: 2024-01-10
  Installed 1.19.13                139.2 MB   installed: 2024-01-05
────────────────────────────────────────────────────────────
Total disk usage: 430.3 MB across 3 versions
Currently active: Go 1.21.5
```

### Output: Remote Versions

```bash
Available Go stable versions (10 total, 2 already installed):
────────────────────────────────────────────────────────────
✓ Installed 1.21.5         installed
  Available 1.21.4         available
  Available 1.21.3         available
✓ Installed 1.20.12        installed
  Available 1.20.11        available
  Available 1.20.10        available
  Available 1.19.13        available
────────────────────────────────────────────────────────────
2 versions already installed (marked with ✓)
Install any version with: govman install <version>
```

---

## govman info

Show detailed information about a specific Go version.

### Usage

```bash
govman info <version> [flags]
```

### Arguments

```bash
<version>    Go version to inspect (e.g., 1.21.5)
```

### Description

Displays comprehensive details about an installed Go version:
- Version and status
- Platform information
- Installation path
- Installation date and age
- Disk usage
- Activation suggestions

### Examples

```bash
# Show info for specific version
govman info 1.21.5
```

### Output

```bash
Go Version Information:
════════════════════════════════════════════════════════════
Version:            Go 1.21.5 (Currently Active)
Platform:           darwin/arm64
Installation Path:  /Users/username/.govman/versions/go1.21.5
Installed On:       Monday, January 15, 2024 at 10:30:45 PST
Disk Usage:         147.3 MB
Age:                15 days old
════════════════════════════════════════════════════════════
This version is currently active in your environment
Run 'go version' to verify, or 'go env' to see full environment
```

### For Inactive Versions

```bash
Go Version Information:
════════════════════════════════════════════════════════════
Version:            Go 1.20.12 (Installed)
Platform:           darwin/arm64
Installation Path:  /Users/username/.govman/versions/go1.20.12
Installed On:       Monday, January 10, 2024 at 14:22:33 PST
Disk Usage:         143.8 MB
Age:                20 days old
════════════════════════════════════════════════════════════
Activate this version with: govman use 1.20.12
Set as default with: govman use 1.20.12 --default
Set for this project: govman use 1.20.12 --local
```

---

## govman clean

Clean download cache to free disk space.

### Usage

```bash
govman clean [flags]
```

### Description

Removes cached download files:
- Downloaded Go archives (.tar.gz, .zip)
- Temporary extraction directories
- Incomplete or corrupted downloads
- Cache metadata

**Safe operation:**
- ✅ Installed Go versions remain untouched
- ✅ Your projects and configurations preserved
- ✅ Only temporary cache files removed

### Examples

```bash
# Clean cache
govman clean
```

### What It Does

1. Scans `~/.govman/cache/` for removable files
2. Removes all cached archives
3. Removes temporary files
4. Reports freed disk space

### Output

```bash
Cleaning download cache and temporary files...
Progress: Scanning cache directories for removable files
Success: Cache cleanup completed successfully
Disk space has been optimized
Your installed Go versions remain untouched and ready to use
Future downloads will rebuild cache as needed
```

### When to Use

- After installing multiple versions
- When disk space is low
- Before backing up your system
- Periodically for maintenance

---

## govman refresh

Re-evaluate current directory for version switching.

### Usage

```bash
govman refresh [flags]
```

### Description

Manually triggers version switching based on current directory:
- Checks for `.govman-version` file
- Switches to specified version if found
- Falls back to default version if not found

Equivalent to the auto-switch that happens automatically with shell integration.

### Examples

```bash
# Refresh version based on current directory
govman refresh
```

### Behavior

**If `.govman-version` exists:**
```bash
$ govman refresh
Found local version file: .govman-version
Switching to Go 1.21.5
Success: Now using Go 1.21.5 for this session
```

**If no `.govman-version`:**
```bash
$ govman refresh
No local version file found
Switching to default Go version
Success: Now using Go 1.20.12 for this session
```

### Use Cases

- After creating/modifying `.govman-version`
- When auto-switch doesn't trigger
- Testing version switching behavior
- Debugging shell integration issues

---

## govman selfupdate

Update govman to the latest version.

### Usage

```bash
govman selfupdate [flags]
```

### Flags

```bash
    --check        Check for updates without installing
    --force        Force update even if already on latest
    --prerelease   Include pre-release versions
```

### Description

Automatically updates govman:
- Checks GitHub for latest release
- Downloads appropriate binary for your platform
- Creates backup of current version
- Replaces binary safely with rollback support

### Examples

```bash
# Check for updates
govman selfupdate --check

# Update to latest stable
govman selfupdate

# Force reinstall current version
govman selfupdate --force

# Include pre-release versions
govman selfupdate --prerelease
```

### Check for Updates

```bash
$ govman selfupdate --check
Checking for govman updates...
Version Information:
  Current: v1.0.0
  Latest:  v1.1.0
  Released: November 01, 2025

A new version is available: v1.0.0 → v1.1.0
Release Notes:
────────────────────────────────────────
### New Features
- Added support for Go 1.26
- Improved download performance
- Enhanced error messages
────────────────────────────────────────
Run 'govman selfupdate' to install this version
```

### Update Process

```bash
$ govman selfupdate
Checking for govman updates...
Version Information:
  Current: v1.0.0
  Latest:  v1.1.0
  Released: November 01, 2025

Download: Downloading v1.1.0...
Success: Update completed successfully!
```

### Safety Features

- ✅ Creates backup before updating
- ✅ Rolls back on failure
- ✅ Verifies download integrity
- ✅ Platform-specific binaries

---

## Exit Codes

govman uses standard exit codes:

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Invalid arguments |

### Examples

```bash
# Check exit code
govman install 1.21.5
echo $?  # 0 on success

# Use in scripts
if govman use 1.21.5; then
    echo "Switched successfully"
else
    echo "Failed to switch"
fi
```

## Output Control

### Quiet Mode

Suppress all output except errors:

```bash
govman --quiet install 1.21.5
```

### Verbose Mode

Show detailed debug information:

```bash
govman --verbose install 1.21.5
```

Output includes:
- Internal progress messages
- Timing information
- Detailed error traces
- Configuration values

## Shell Completion

Enable shell completion for command suggestions:

```bash
# Bash
govman completion bash > /etc/bash_completion.d/govman

# Zsh
govman completion zsh > /usr/local/share/zsh/site-functions/_govman

# Fish
govman completion fish > ~/.config/fish/completions/govman.fish

# PowerShell
govman completion powershell > govman.ps1
```

## See Also

- [Quick Start](quick-start.md) - Get started with govman
- [Configuration](configuration.md) - Configure govman
- [Shell Integration](shell-integration.md) - Set up auto-switching
- [Troubleshooting](troubleshooting.md) - Common issues

---

Happy Go development! 🚀
