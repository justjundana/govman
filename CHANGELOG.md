# Changelog

All notable changes to GOVMAN (Go Version Manager) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [1.2.0] - 2026-02-01

### 🎯 Minor Release - Wildcard Pattern Support & Batch Operations

This release adds wildcard pattern support for batch install/uninstall operations, allowing users to manage multiple versions matching a pattern with a single command. Includes confirmation prompts and optional unstable version support.

### Added
- Wildcard pattern support for `install` and `uninstall` commands
  - `govman install '1.14.*'` - Install all stable 1.14.x versions
  - `govman uninstall '1.14.*'` - Uninstall all installed 1.14.x versions
  - Confirmation prompt before batch operations (skip with `-y` flag)
  - **Note:** Quote the pattern to prevent shell glob expansion!
- `--unstable` flag for `install` command to include beta/rc versions in pattern expansion
- `-y` / `--yes` flag for both `install` and `uninstall` to skip confirmation prompts
- New utility functions: `IsWildcardPattern()`, `MatchVersionPattern()` for pattern matching

## [1.1.1] - 2026-01-01

### Changed
- Removed redundant `min` function from CLI list command (Go 1.21+ has built-in `min`)
- Simplified configuration loading by removing unnecessary `cfgMutex`
- Replaced per-request HTTP client with shared client in selfupdate for connection reuse
- Improved download caching - archives now preserved for reuse
- Extracted `configMarkers` to package-level constant in shell integration
- Removed unused `shell` parameter from `initializeCmdShell` function
- Added package documentation for `progress` package
- Extracted magic number `100ms` to named constant `updateThreshold`

### Fixed
- **Critical:** Fixed version alias resolution bug in `use`, `info`, and `uninstall` commands
  - `govman use latest` would fail after `govman install latest`
  - All commands now resolve aliases and partial versions before processing
- **Critical:** Added HTTP status code validation in selfupdate before parsing GitHub API response
- Simplified redundant prerelease logic in `getLatestRelease` function
- Eliminated TOCTOU race condition in releases cache using double-checked locking
- Improved error logging - errors now logged at verbose level instead of silently ignored
- Implemented atomic config file writes using temp file + rename pattern
- Added guards for progress bar calculations to prevent edge case issues
- Fixed download cache - archives now preserved instead of deleted after extraction
- Clean up backup files after successful selfupdate
- Improved symlink version extraction using regex pattern matching
- Fixed tar extraction not handling symlinks (with security validation)
- Implemented atomic symlink creation using temp symlink + rename pattern
- Added version format validation to `refresh` command with helpful error messages
- Improved selfupdate temp file cleanup (avoids removing successfully renamed files)

## [1.1.0] - 2025-12-01

### 🎉 Minor Release - Version Aliasing, Multi-Version Management & Flexible Matching

This release adds support for version aliases in the `use` command, multi-version batch uninstallation, flexible version matching for `.govman-goversion` files, bug fixes for auto-switching functionality, and includes internal code cleanup and refactoring.

### Added
- `use` command now supports version aliases (e.g., `latest`, `1.25`)
- `uninstall` command now supports multiple versions in a single command
  - Batch uninstallation with progress tracking for each version
  - Displays total disk space freed across all uninstalled versions
  - Continues processing remaining versions if one fails
  - Example: `govman uninstall 1.24.1 1.24.2 1.24.3`
  - Matches the behavior of the `install` command for consistency
- Flexible version matching for `.govman-goversion` files
  - `.govman-goversion` can now contain partial versions (e.g., `1.25`) that match any installed version with the same major.minor
  - Automatically selects the highest available patch version when multiple matches exist
  - Example: `1.25` in `.govman-goversion` will match `1.25.1`, `1.25.4`, or `1.25.9` (picks highest)
  - Backward compatible with exact version specifications

### Changed
- Renamed `.govman-version` to `.govman-goversion` for clarity and improved specificity in tracking Go versions.
- Removed `IsValidVersion` from `golang` package
- Removed `Step` logging functionality from `logger` package
- Removed `GetDefaultVersionFromSymlink` from `manager` package
- Removed `MultiProgress` from `progress` package as it was unused
- Removed `DetectAll` from `shell` package
- Removed custom `ReadLink` from `symlink` package
- Refactored tests to be more robust and less dependent on removed code

### Deprecated
- N/A

### Removed
- Only internal/unused code was removed; no breaking changes

### Fixed
- **Critical:** Fixed CmdShell using wrong escaping function (`escapeBashPath` → `escapeCmdPath`)
  - This was causing path failures on Windows Command Prompt with special characters
- Improved YAML parsing reliability in shell integration
  - Replaced fragile `grep -A 10` approach with robust `awk`-based parsing
  - Added default values and fallback logic for auto-switch configuration
  - No longer depends on hardcoded line limits, handles edge cases better
- Enhanced Go version extraction and validation
  - Now properly handles pre-release versions (e.g., `1.21rc1`)
  - Added format validation to prevent malformed version strings
  - More precise regex patterns for version matching
- Fixed duplicate hook registration issues
  - Prevents multiple PROMPT_COMMAND entries in Bash when sourcing config multiple times
  - Prevents duplicate chpwd hooks in Zsh
  - Prevents duplicate PWD event hooks in Fish
  - Prevents nested prompt function hijacking in PowerShell
- Improved pattern matching and display for `list --remote` command

### Security
- **Critical:** Eliminated command injection vulnerabilities in shell integration
  - Added strict validation for `eval` statements in Bash/Zsh (now validates against regex pattern)
  - Added strict validation for `Invoke-Expression` in PowerShell (validates PATH command format)
  - Changed from unsafe `echo "$output"` to safe `printf '%s\n' "$output"` 
  - All export commands are now validated before execution
- Enhanced input validation across all shell SetupCommands functions
- Improved regex patterns to prevent injection through malformed PATH values

## [1.0.0] - 2025-11-01

### Added
- 🎉 First public release of GOVMAN (Go Version Manager)
- Core Go version management functionality
- Install, uninstall, and switch between Go versions
- Project-specific version support
- Cross-platform compatibility (Windows, macOS, Linux, ARM)
- Command-line interface with Cobra framework
- Configuration management with Viper
- Comprehensive test coverage for all core components
- Multi-shell support (Bash, Zsh, Fish, PowerShell, Command Prompt)
- Automatic Go version switching with `.govman-version` files
- Parallel downloads with resume capability
- Cross-platform symlink management
- Intelligent caching system with configurable expiry
- Progress bars for download operations
- Verbose and quiet logging modes
- Self-update functionality
- Complete shell integration with auto-switching hooks
- Version information and metadata display
- Cache management and cleanup tools
- Go releases API integration
- Download resumption support
- Multi-format archive extraction (tar.gz, zip)