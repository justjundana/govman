# Release Notes

This document contains the release history and changes for govman.

## Version Format

govman follows Semantic Versioning (SemVer):
- MAJOR.MINOR.PATCH (e.g., 1.2.3)
- MAJOR: Breaking changes
- MINOR: New features (backwards compatible)
- PATCH: Bug fixes (backwards compatible)

## Latest Release

### v1.2.0

**Release Date:** February 01, 2026

**Highlights:**
- 🎯 **New:** Wildcard pattern support for batch install/uninstall operations
- 📦 Install or uninstall multiple versions matching a pattern (e.g., `1.14.*`)
- ✅ Confirmation prompt for batch operations with `-y` flag to skip

**Features:**
- Wildcard pattern support for `install` and `uninstall` commands
  - `govman install '1.14.*'` - Install all stable versions matching the pattern
  - `govman uninstall '1.14.*'` - Uninstall all installed versions matching the pattern
  - Shows list of matching versions and asks for confirmation before proceeding
  - Use `-y` or `--yes` flag to skip confirmation: `govman install '1.14.*' -y`
  - **Important:** Quote the pattern (`'1.14.*'`) to prevent shell glob expansion!
- `--unstable` flag for `install` command
  - Include beta/rc versions when using wildcard patterns
  - Example: `govman install '1.22.*' --unstable` includes `1.22rc1`, `1.22beta1`, etc.

---

### v1.1.1

**Release Date:** January 01, 2026

**Highlights:**
- 🐛 **Critical:** Fixed version alias resolution bug affecting `use`, `info`, and `uninstall` commands
- 🐛 **Critical:** Added HTTP status validation for GitHub API in selfupdate
- 🔒 Added version format validation for enhanced security
- 🏎️ Performance and code quality improvements from comprehensive code audit

**Bug Fixes:**
- **Critical:** Fixed version alias resolution bug in `use`, `info`, and `uninstall` commands
  - `govman use latest` would fail after `govman install latest` with "version latest not installed"
  - All commands now resolve aliases and partial versions to actual installed versions before processing
- **Critical:** Added HTTP status code validation in `getLatestRelease` (selfupdate.go)
  - Previously, 404/500 responses caused confusing JSON parsing errors
  - Now provides clear error: "GitHub API returned status 404: Not Found"
- Simplified redundant prerelease logic in selfupdate
- Fixed download cache - archives no longer deleted after extraction (use `govman clean` to manage cache)
- Added atomic config writes to prevent corruption on crash
- Added percentage/width clamping in progress bar for edge cases
- Clean up backup files after successful selfupdate (removes `.bak.*` accumulation)
- Improved symlink version extraction using regex (more robust across platforms)
- Fixed tar extraction not handling symlinks (validates targets for security)
- Implemented atomic symlink creation to eliminate TOCTOU race condition
- Added version format validation in `refresh` command for `.govman-goversion` files
- Improved selfupdate temp file cleanup (checks existence before removal after rename)

**Internal Changes:**
- Removed redundant `min` function (Go 1.21+ built-in)
- Simplified config loading by removing unnecessary mutex
- Added shared HTTP client for selfupdate operations (connection reuse)
- Eliminated TOCTOU race condition in releases cache with double-checked locking
- Improved error logging throughout CLI commands
- Extracted `configMarkers` slices to package-level constant for maintainability
- Removed unused `shell` parameter from `initializeCmdShell` function
- Added package documentation for `progress` package
- Extracted update throttle interval to named constant `updateThreshold`

---

### v1.1.0

**Release Date:** December 01, 2025

**Highlights:**
- 🎉 Version aliases support for `use` command  
- 🗑️ Multi-version uninstall support
- 🎯 Flexible version matching for `.govman-goversion` files
- 🔒 Critical security fixes for shell integration
- 🛡️ Enhanced stability and robustness improvements
- 🧹 Internal code cleanup and refactoring

**Features:**
- `use` command now supports version aliases (e.g., `latest`, `1.25`)
- `uninstall` command now supports multiple versions in a single command
  - Batch uninstallation with progress tracking for each version (`[1/4]`, `[2/4]`, etc.)
  - Displays total disk space freed across all uninstalled versions
  - Continues processing remaining versions if one fails
  - Example: `govman uninstall 1.24.1 1.24.2 1.24.3`
  - Comprehensive summary output showing successes and failures
  - Matches the behavior of the `install` command for consistency
- Improved pattern matching and display for `list --remote` command
- Flexible version matching for `.govman-goversion` files
  - `.govman-goversion` can now contain partial versions (e.g., `1.25`) that match any installed version with the same major.minor
  - Automatically selects the highest available patch version when multiple matches exist
  - Example: `1.25` in `.govman-goversion` will match `1.25.1`, `1.25.4`, or `1.25.9` (picks highest)
  - Backward compatible with exact version specifications
  - Improves developer experience by removing strict version requirements

**Internal Changes:**
- Renamed `.govman-version` to `.govman-goversion` for clarity and improved specificity in tracking Go versions.
- Removed `IsValidVersion` from `golang` package
- Removed `Step` logging functionality from `logger` package
- Removed `GetDefaultVersionFromSymlink` from `manager` package
- Removed `MultiProgress` from `progress` package as it was unused
- Removed `DetectAll` from `shell` package
- Removed custom `ReadLink` from `symlink` package
- Refactored tests to be more robust and less dependent on removed code
- Only internal/unused code was removed; no breaking changes

**Bug Fixes:**
- **Critical:** Fixed CmdShell using wrong escaping function (`escapeBashPath` → `escapeCmdPath`)
  - Windows Command Prompt users were experiencing path failures with special characters
  - Now correctly escapes paths using CMD-specific escape rules
- Improved YAML parsing reliability in shell integration
  - Replaced fragile `grep -A 10` approach with robust `awk`-based parsing
  - No longer depends on hardcoded line limits
  - Added default values and proper fallback logic for auto-switch configuration
- Enhanced Go version extraction and validation across all shells
  - Now properly handles pre-release versions (e.g., `go1.21rc1`, `go1.22beta1`)
  - Added format validation to prevent malformed version strings from causing issues
  - More precise regex patterns: `\d+\.\d+(?:\.\d+)?` instead of `[\d\.]+`
- Fixed duplicate hook registration issues
  - Bash: Prevents multiple `__govman_check_dir_change` entries in `PROMPT_COMMAND` when sourcing `.bashrc` multiple times
  - Zsh: Prevents duplicate `chpwd` hooks using Zsh's array check
  - Fish: Clears existing `__govman_cd_hook` function before redefining
  - PowerShell: Prevents nested prompt function hijacking with `$Global:GovmanPromptInjected` flag

**Security Improvements:**
- **Critical:** Eliminated command injection vulnerabilities in shell integration
  - Bash/Zsh: Added strict regex validation for `eval` statements
    - Changed from unsafe `echo "$output" | grep` to safe `printf '%s\n' "$output" | grep`
    - Validates export commands match pattern: `^export PATH="[^"]*"$` before eval
  - PowerShell: Added strict validation for `Invoke-Expression`
    - Validates PATH commands match expected format: `^\$env:PATH\s*=\s*"[^"]+"\s*\+\s*\$env:PATH$`
    - Double validation (filter + regex match) before executing
  - Fish: Improved pattern matching for `fish_add_path` commands
- Enhanced input validation across all shell `SetupCommands` functions
- All export/PATH commands are now validated before execution
- Improved regex patterns to prevent injection through malformed PATH values
- Better escaping for all special characters in path handling

---

### v1.0.0

**Release Date:** November 01, 2025

**Highlights:**
- 🎉 Initial stable release
- Cross-platform support (Linux, macOS, Windows)
- Intelligent shell integration
- Automatic version switching
- Fast parallel downloads
- Project-specific version management

**Features:**

**Core Features:**
- Complete Go version management system (install, uninstall, switch versions)
- Cross-platform support: Linux (amd64, arm64), macOS (amd64, arm64/Apple Silicon), Windows (amd64, arm64)
- Project-specific versions via `.govman-goversion` files
- System-wide default version (persistent across shell sessions)
- Session-only activation (temporary version switching)

**Commands:**
- `govman install <version>` - Install Go versions with intelligent caching
- `govman uninstall <version>` - Remove installed Go versions
- `govman use <version>` - Switch Go versions (session, default, or local)
- `govman list` - Display installed versions
- `govman list --remote` - Show available versions from go.dev
- `govman current` - Show currently active Go version
- `govman info <version>` - Display detailed version information
- `govman clean` - Remove download cache
- `govman init` - Set up shell integration
- `govman selfupdate` - Update govman to latest version
- `govman refresh` - Manually trigger version refresh

**Shell Integration:**
- Multi-shell support (Bash, Zsh, Fish, PowerShell, Command Prompt)
- Automatic version switching on directory change
- Smart PATH management without shell restart
- Hook integration (PROMPT_COMMAND, chpwd, PWD events, Set-Location)
- Wrapper functions for seamless `govman use` execution

**Download & Installation:**
- Parallel downloads with configurable max connections
- Resume capability for interrupted downloads
- Intelligent caching to avoid re-downloading
- SHA-256 checksum verification
- Real-time progress bars with speed and ETA
- Mirror support for restricted regions
- Automatic retry logic

**Configuration:**
- YAML configuration file (`~/.govman/config.yaml`)
- Customizable install and cache directories
- Download options (parallel, timeout, retry settings)
- Auto-switch control
- Mirror configuration
- API caching (10-minute expiry)

**Architecture:**
- CLI framework (Cobra)
- Configuration management (Viper)
- Modular packages (cli, manager, downloader, golang, shell, logger, config)
- Smart version resolution ("latest", partial versions like 1.25 → 1.25.1)
- Cross-platform symlink management
- Multi-format archive extraction (tar.gz, zip)

**Security:**
- No root/admin privileges required
- Directory traversal protection
- HTTPS-only connections
- SHA-256 checksum verification for all downloads
- Secure defaults
- All downloads verified with SHA-256 checksums
- HTTPS-only connections to go.dev and github.com

**Supported Platforms:**
- Linux (amd64, arm64)
- macOS (amd64, arm64/Apple Silicon)
- Windows (amd64, arm64)

**Supported Shells:**
- Bash
- Zsh
- Fish
- PowerShell
- Command Prompt (limited support)

## Development Versions

### dev

Development builds track the main branch.

**Features in Development:**
- Continuous improvements and bug fixes
- Performance optimizations
- Enhanced error messages

## Release Process

1. Version bump in `internal/version/version.go`
2. Update `CHANGELOG.md` and `docs/release-notes.md`
3. Create git tag: `git tag -a v1.0.0 -m "Release v1.0.0"`
4. Push tag: `git push origin v1.0.0`
5. GitHub Actions builds and publishes binaries
6. Release notes published on GitHub

## Upgrade Path

### From Development Version to Stable

```bash
# Development versions cannot auto-update
# Reinstall from release:
curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
```

### Between Stable Versions

```bash
govman selfupdate
```

## Breaking Changes

No breaking changes in v1.0.0 (initial release).

## Deprecation Policy

- Features will be marked deprecated for at least one minor version before removal
- Deprecated features will show warnings with migration guidance
- Breaking changes only in major version updates (2.0.0, 3.0.0, etc.)

## Security Releases

Security issues are addressed as soon as possible:
- Critical: Patch release within 24 hours
- High: Patch release within 7 days
- Medium/Low: Included in next scheduled release

## Platform-Specific Notes

### Windows
- PowerShell 5.1+ supported
- Windows 10+ recommended
- Command Prompt has limited features

### macOS
- Apple Silicon (M1/M2/M3) fully supported
- Older Go versions (< 1.16) use Rosetta 2 on ARM

### Linux
- Works on all major distributions
- No specific kernel required
- arm64 fully supported

## Known Issues

Track known issues on [GitHub Issues](https://github.com/justjundana/govman/issues).

## Reporting Issues

Report bugs with:
- govman version (`govman --version`)
- Operating system and version
- Shell type and version
- Steps to reproduce
- Expected vs actual behavior
