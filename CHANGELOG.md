# Changelog

All notable changes to GOVMAN (Go Version Manager) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [1.3.3] - 2026-05-01

### 🔧 Patch Release - Code Quality & Stability Improvements

This release reduces cyclomatic complexity across all flagged functions, fixes an ineffectual assignment, and replaces the broken interactive progress bar with a clean log-style output that never garbles on terminal resize.

### Changed
- Replaced interactive single-line progress bar with log-style percentage output
  - Previous implementation garbled on terminal resize due to ANSI escape / line-wrapping issues
  - New implementation prints one line per progress update (throttled to 1 per second)
  - Displays percentage, downloaded/total size, speed, and ETA per line
  - Skips duplicate percentage lines to keep output concise
  - Removed dependency on `golang.org/x/term` (no longer needed)
  - Zero risk of terminal garbling — uses plain `fmt.Printf` with newlines

### Fixed
- Reduced cyclomatic complexity in `newPruneCmd` by extracting `getProtectedVersions`, `executePrune`, and `reportPruneResults` helpers
- Reduced cyclomatic complexity in `newUseCmd` by extracting `resolveAlias`, `resolvePartialVersion`, `resolveFullVersion`, and `resolveVersionForUse` helpers
- Reduced cyclomatic complexity in `listRemoteVersions` by extracting `countVersionStats` and `formatVersionTypeDesc` helpers
- Fixed ineffectual assignment to `versionTypeDesc` in `listRemoteVersions`
- Reduced cyclomatic complexity in `newInstallCmd` and `newUninstallCmd` by extracting `installVersions` and `uninstallVersions` helpers
- Reduced cyclomatic complexity in `runSelfUpdate` by extracting `findAssetURL`, `downloadBinary`, `replaceBinary`, and `cleanupBackupFiles` helpers
- Reduced cyclomatic complexity in `downloadFile` by extracting `downloadWithRetry`, `handleResumeResponse`, and `setupProgressReader` helpers
- Reduced cyclomatic complexity in `extractTarGz` and `extractZip` by extracting `validateArchivePath`, `ensureParentDir`, `extractTarEntry`, and `extractZipFile` helpers
- Reduced cyclomatic complexity in `TestGlobalLogger` by splitting into `TestGlobalLogger` and `TestGlobalLoggerExtended`
- Reduced cyclomatic complexity in `TestDownloader_extractTarGz` by extracting `createTestTarGz` and `verifyExtractedFile` helpers
- Reduced cyclomatic complexity in `TestInitializeShellWithExistingConfig` by extracting `testInitializeUnixShell`, `testInitializePowerShell`, and `testInitializeCmdShell` helpers
- Reduced cyclomatic complexity in `TestReleaseAndFileStructs` by splitting into `TestReleaseJSONUnmarshaling`, `TestFileJSONUnmarshaling`, and `TestVersionInfoStructFields`
- Reduced cyclomatic complexity in `TestLoad` by extracting `setTestHome` and `makeGovmanDirReadOnly` helpers

## [1.3.2] - 2026-05-01

### 🔧 Patch Release - Bug Fixes & Code Quality Improvements

This release fixes several edge-case bugs across install, selfupdate, shell auto-switch, progress bar, and version switching. No new features are added.

### Fixed
- Fixed `install` wildcard expansion with `--unstable` flag stripping stable versions
  - `govman install '1.14.*' --unstable` previously returned only prerelease versions
  - Now correctly returns both stable and prerelease versions matching the pattern
- Fixed `list --remote --beta` not counting alpha versions as unstable
  - Alpha versions were missing from the unstable version counter in summary output
- Fixed selfupdate cross-device rename failure on Linux
  - Temp file was created in system temp dir (`/tmp`), causing `EXDEV` error when renaming to binary dir on a different filesystem
  - Now creates temp file in the same directory as the current binary
- Fixed selfupdate asset matching using substring instead of exact name
  - `strings.Contains` could match `govman-linux-amd64-v2` when looking for `govman-linux-amd64`
  - Now uses exact name comparison
- Fixed symlink replacement not being fully atomic
  - `manager.createSymlink` explicitly removed the old symlink before calling `symlink.Create`
  - This created a window where no symlink existed
  - Removed the redundant `os.Remove` since `symlink.Create` already performs atomic replacement via temp-symlink + `os.Rename`
- Fixed shell auto-switch hooks triggering redundant version switches with partial versions
  - When `.govman-goversion` contains a partial version like `1.25`, and the active version is `1.25.3`, the comparison `"1.25.3" != "1.25"` would trigger an unnecessary switch
  - Auto-switch hooks in all 4 shells (Bash, Zsh, Fish, PowerShell) now truncate the current version to match the format of the required version before comparison
- Fixed `use` command allowing `--default` and `--local` flags simultaneously
  - These flags are mutually exclusive but were not validated
  - Now enforced via Cobra's `MarkFlagsMutuallyExclusive`
- Fixed progress bar `Set()` method not clamping negative values
  - Negative values are now clamped to `0` (upper bound was already clamped to `total`)
- Added missing `cmd` shell support in `init --shell` flag
  - `govman init --shell cmd` was not recognized despite `CmdShell` being fully implemented
- Cleaned up unnecessary trailing empty string arguments in `ErrorWithHelp` calls across CLI commands

## [1.3.1] - 2026-04-01

### 🔧 Patch Release - Bug Fixes & Code Quality Improvements

This release focuses on fixing bugs, improving code quality, and ensuring consistent behavior across commands. No new features are added.

### Changed
- `list --remote` now defaults to showing only stable versions, consistent with `install` behavior
  - Removed redundant `--stable-only` flag since stable is now the default
  - Use `--beta` flag to include pre-release versions
- `install` command's `--unstable` flag description corrected from "Show only" to "Include unstable versions"

### Fixed
- **Critical:** Fixed `refresh` command failing with aliases (`latest`/`stable`) and partial versions (`1.25`)
  - Now resolves aliases via `ResolveVersion()` and partial versions via `FindBestMatchingVersion()`
  - Consistent with `use`, `info`, and `install` commands
- **Critical:** Fixed selfupdate binary download not validating HTTP status code
  - A 404 or error response would silently corrupt the binary
  - Now validates status code before writing response body to temp file
- **Critical:** Fixed download resume producing corrupted archives
  - When a `Range` header was sent but server responded with `200 OK` instead of `206 Partial Content`, the full file was appended to the existing partial file
  - Now truncates the file and restarts download from scratch when server does not support resume
- Fixed selfupdate version comparison using string equality instead of SemVer
  - `v1.3.0` vs `1.3.0` would incorrectly report updates available
  - Now uses `CompareVersions` with prefix normalization
- Fixed `prune` command using overly broad `strings.HasPrefix` for local version protection
  - Version `1.2` in `.govman-goversion` would incorrectly protect `1.20.x`, `1.21.x`, etc.
  - Now uses `FindBestMatchingVersion` for precise major.minor matching
- Fixed misleading indentation in auto-switch hooks for Bash, Zsh, and Fish shells
  - Go version check and switch logic appeared to be inside a non-existent block
- Fixed `.govman-goversion` file not ending with trailing newline
  - `setLocalVersion` now writes version with `\n` suffix per text file conventions
- Removed duplicate "Updated Makefile" entry from v1.3.0 CHANGELOG

### Performance
- Moved HTTP request outside write lock in releases cache (`fetchReleasesWithConfig`)
  - Previously blocked all goroutines for up to 30 seconds during slow network requests
  - Now only acquires write lock when updating the cache

## [1.3.0] - 2026-03-01

### 🚀 Minor Release - Prune Command & Shell Performance

This release introduces a new `prune` command for cleaning up unused Go versions, optimizes auto-switch shell hooks for better performance, hardens `.govman-goversion` parsers across all shells, and fixes backup cleanup issues on Windows.

### Added
- New `prune` command to remove unused Go versions
  - Automatically identifies and removes Go versions that are no longer in use
  - Helps keep the system clean by freeing disk space from stale installations
### Changed
- Optimized auto-switch shell hook to skip `go version` call when unnecessary
  - Reduces shell startup and directory-change overhead
  - Hook now only invokes `go version` when a version switch is actually needed
- Updated `Makefile` to use dynamic `$(HOME)` path instead of hardcoded static path for `install-local` target
- Replaced `filepath.Walk` with `filepath.WalkDir` in `getDirSize` for better performance
  - Avoids unnecessary `os.Stat` calls on directory entries
- Deduplicated `versionFormatRegex` across `cli/refresh.go` and `manager/manager.go`
  - Exported as `VersionFormatRegex` from manager package for reuse
  - Removed duplicate definition and unused `regexp` import from CLI refresh command

### Fixed
- Hardened `.govman-goversion` parsers for edge cases across all shells
  - Improved robustness when handling malformed or unexpected file content
  - Better handling of whitespace, empty lines, and special characters
- Gracefully handle backup cleanup on Windows with startup routine in selfupdate
  - Resolves issues where locked backup files could not be removed immediately
  - Adds a deferred cleanup mechanism via startup routine
- Added `stable` alias support to `Manager.ResolveVersion`
  - `govman install stable` now works like `govman install latest`
  - Previously only `latest` was handled, `stable` was silently ignored
- **Critical:** Fixed `govman refresh` not actually switching Go version
  - Shell wrapper function only intercepted `govman use`, not `govman refresh`
  - `refresh` output was printed to stdout but never eval'd in the current shell
  - Added `refresh` to wrapper intercept in all shells: Bash, Zsh, Fish, PowerShell

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