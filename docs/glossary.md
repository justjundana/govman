# Glossary

Common terms and concepts used in govman documentation.

## General Terms

### govman
The Go Version Manager - a command-line tool for installing and managing multiple Go versions.

### Go Version
A specific release of the Go programming language (e.g., 1.25.1, 1.24.0).

### Version String
The identifier for a Go version, optionally including "go" prefix (e.g., "1.25.1" or "go1.25.1").

## Version Types

### Stable Release
Official production-ready Go version with full support. Recommended for most users.
Example: 1.25.1, 1.24.0

### Pre-release
Beta or release candidate versions for testing. Not recommended for production.
Examples: 1.26beta1, 1.26rc1

### Latest
Refers to the most recent stable Go release. Pre-releases are excluded unless explicitly requested.

### Patch Version
The third number in a version (e.g., the "1" in 1.25.1). Contains bug fixes only.

### Minor Version
The second number in a version (e.g., the "25" in 1.25.1). May include new features.

### Major Version
The first number in a version (currently always "1" for Go).

## Installation Concepts

### Install Directory
Where Go versions are installed.
Default: `~/.govman/versions/`

### Cache Directory
Where downloaded Go archives are stored for reuse.
Default: `~/.govman/cache/`

### Checksum
SHA-256 hash used to verify download integrity. Ensures files aren't corrupted or tampered with.

### Archive
Compressed file containing a Go distribution (.tar.gz on Unix, .zip on Windows).

### Extraction
Decompressing an archive and copying files to the install directory.

## Activation Concepts

### Active Version
The Go version currently in use (in PATH).

### Session-only Activation
Temporary version activation for the current shell session only. Not persistent across shell restarts.

### System Default
The Go version used by default across all new shell sessions. Persistent.

### Project-local Version
Go version tied to a specific project directory via `.govman-goversion` file.

### Activation Method
How a version was activated: session-only, system-default, or project-local.

## Version Management

### Installed Version
A Go version that has been downloaded and extracted to the install directory.

### Remote Version
A Go version available for download from official sources but not yet installed.

### Current Version
The Go version currently active in your environment.

### Default Version
The system-wide default Go version (set with `govman use <version> --default`).

### Version Resolution
The process of converting a version string (like "latest" or "1.25") to a specific version number (like "1.25.1").

## Shell Integration

### Shell Integration
Code added to shell configuration files to enable automatic version switching and PATH management.

### Shell Hook
Mechanism that triggers automatic actions on directory change (e.g., `chpwd` in Zsh, `PROMPT_COMMAND` in Bash).

### Wrapper Function
Shell function that intercepts `govman` commands to update PATH in the current session.

### Auto-switch
Automatic version switching when navigating to directories with `.govman-goversion` files.

### `.govman-goversion` File
Project file containing the required Go version. Enables automatic version switching.

### Shell Configuration File
File that configures your shell environment.
Examples: `~/.bashrc`, `~/.zshrc`, `~/.config/fish/config.fish`, `$PROFILE`

## PATH and Environment

### PATH
Environment variable listing directories where the shell looks for executables.

### GOROOT
Environment variable pointing to the Go installation directory. Managed automatically by govman.

### GOPATH
Environment variable for Go workspace. Not managed by govman.

### GOBIN
Directory for user-installed Go binaries. Added to PATH by shell integration.

### GOTOOLCHAIN
Environment variable controlling Go toolchain selection. Set to "local" by govman.

### Symlink
A symbolic link pointing from `~/.govman/bin/go` to the active version's Go binary.

## Configuration

### Configuration File
YAML file at `~/.govman/config.yaml` storing govman settings.

### Mirror
Alternative download source for Go releases. Useful in regions with restricted access to golang.org.

### Download Config
Settings controlling download behavior (parallel, timeout, retries).

### Auto-switch Config
Settings controlling automatic version switching behavior.

## Network and Downloads

### go.dev API
Official Go releases API providing version metadata and download links.

### Parallel Download
Downloading a file using multiple simultaneous connections for faster speed.

### Download Resume
Continuing an incomplete download from where it left off.

### Retry Logic
Automatically retrying failed operations (e.g., downloads) before giving up.

### Cache Hit
Successfully using a previously downloaded file from the cache, avoiding re-download.

### Cache Miss
Not finding a requested file in the cache, requiring a fresh download.

## govman Operations

### Install
Download and set up a Go version in the install directory.

### Uninstall
Remove an installed Go version from the install directory.

###Use
Activate a specific Go version (make it the active version).

### Switch
Change from one Go version to another (synonym for "use").

### List
Display installed versions (`govman list`) or available versions (`govman list --remote`).

### Info
Show detailed information about a specific Go version.

### Clean
Remove downloaded archives from the cache directory to free disk space.

### Refresh
Manually trigger version switching based on the current directory.

### Self-update
Update govman itself to the latest version.

### Init
Set up shell integration for automatic version switching.

## Platform Terms

### Platform
Operating system and architecture combination (e.g., linux-amd64, darwin-arm64).

### Architecture
CPU architecture (amd64, arm64, 386, etc.).

### Cross-platform
Works on multiple operating systems (Linux, macOS, Windows).

### Rosetta 2
Apple's translation layer for running x86_64 binaries on Apple Silicon Macs.

### Apple Silicon
ARM64-based Apple processors (M1, M2, M3).

## Development Terms

### Binary
Compiled executable program (govman itself or Go installations).

### Release
Published version of govman with pre-built binaries.

### GitHub Release
govman version published on GitHub with downloadable assets.

### Semantic Versioning (SemVer)
Version numbering scheme: MAJOR.MINOR.PATCH (e.g., 1.2.3).

### Development Build
Unreleased version built from source, tagged as "dev".

## Security Terms

### Checksum Verification
Validating downloaded files using SHA-256 hashes to ensure integrity.

### Directory Traversal
Security vulnerability where paths like `../` are used to access unauthorized directories. Prevented by govman.

### HTTPS
Secure HTTP protocol used for all external connections.

### Certificate Validation
Verifying SSL/TLS certificates to prevent man-in-the-middle attacks.

### User Space
Operating system space where user applications run, as opposed to kernel/system space. Requires no admin privileges.

## File System

### Home Directory
User's home folder.
Unix/Linux/macOS: `/home/username` or `~`
Windows: `C:\Users\Username`

### Tilde Expansion
Converting `~` to the actual home directory path.

### Absolute Path
Full path from root directory (e.g., `/home/user/.govman/config.yaml`).

### Relative Path
Path relative to current directory (e.g., `./project/.govman-goversion`).

### Atomic Operation
File system operation that completes entirely or not at all, with no intermediate state.

## Error Handling

### Exit Code
Numeric value returned by a command indicating success (0) or specific error types (1-6).

### Error Wrapping
Adding context to an error message while preserving the original error.

### Retry Attempt
Trying a failed operation again before returning an error.

### Graceful Degradation
Continuing to function with reduced capabilities when somethingfails (e.g., disabling colors on unsupported terminals).

## Logging

### Quiet Mode
Suppress all output except errors (`--quiet` flag).

### Verbose Mode
Show detailed debugging information (`--verbose` flag).

### Log Level
Amount of detail in log output: quiet, normal, or verbose.

### ANSI Colors
Terminal color codes for formatted output.

### Progress Bar
Visual indicator showing completion percentage and estimated time for long-running operations.

## Common Abbreviations

- **API**: Application Programming Interface
- **CLI**: Command-Line Interface
- **ETA**: Estimated Time of Arrival (completion)
- **HTTP(S)**: HyperText Transfer Protocol (Secure)
- **I/O**: Input/Output
- **JSON**: JavaScript Object Notation
- **OS**: Operating System
- **PATH**: System environment variable for executable directories
- **RC**: Release Candidate
- **SHA**: Secure Hash Algorithm
- **SSL/TLS**: Secure Sockets Layer / Transport Layer Security
- **TTL**: Time To Live (cache expiration)
- **UI**: User Interface
- **URL**: Uniform Resource Locator
- **YAML**: YAML Ain't Markup Language (configuration format)

## Version Activation Priority

When multiple version specifications exist, govman uses this priority order (highest to lowest):

1. Session-only (govman use X)
2. Project-local (.govman-goversion file)
3. System default (config.yaml DefaultVersion)

## File Extensions

- `.tar.gz`: Compressed archive for Unix/Linux/macOS
- `.zip`: Compressed archive for Windows
- `.yaml` or `.yml`: YAML configuration file
- `.exe`: Windows executable
- `.sh`: Shell script (Unix/Linux/macOS)
- `.ps1`: PowerShell script
- `.bat`: Windows batch file
