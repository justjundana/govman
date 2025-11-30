# Requirements

govman is designed to work with minimal dependencies on all supported platforms.

## System Requirements

### Operating System

govman supports:
- **Linux** (kernel  2.6+ recommended)
- **macOS** 10.12 (Sierra) or later
- **Windows** 10 or later (Windows 7/8 may work with limitations)

### Architecture

- **amd64** (x86_64)
- **arm64** (ARM 64-bit, including Apple Silicon)

### Disk Space

- **govman binary**: ~10-20 MB
- **Per Go version**: ~100-500 MB (varies by version and platform)
- **Download cache**: Varies (can be cleaned with `govman clean`)

Recommended minimum: **1 GB free** for comfortable usage with multiple Go versions.

## Software Dependencies

### Linux

**Required**:
- One of: `curl` or `wget` (for downloading)
- `tar` and `gzip` (for extracting archives)
- A supported shell: `bash`, `zsh`, or `fish`

**Optional**:
- `git` (for development workflows)

Installation on Debian/Ubuntu:
```bash
sudo apt-get install curl tar gzip
```

Installation on RHEL/CentOS/Fedora:
```bash
sudo yum install curl tar gzip
```

### macOS

**Required**:
- `curl` (pre-installed on macOS)
- `tar` (pre-installed on macOS)
- A supported shell: `bash`, `zsh`, or `fish` (zsh is default on macOS 10.15+)

**Optional**:
- Homebrew (for alternative installation methods)
- `git` (typically installed via Xcode Command Line Tools)

### Windows

**Required**:
- PowerShell 5.1+ or PowerShell Core 7+ (recommended)
- OR Command Prompt (with limited features)

**Optional**:
- `curl.exe` (included in Windows 10 1803+)
- PowerShell 7+ for better experience
- Windows Terminal (recommended for better UI)
- Git Bash or WSL (for bash shell integration)

## Network Requirements

govman requires internet access for:
- Downloading govman binary during installation
- Fetching available Go release information
- Downloading Go versions
- Updating govman itself

**Firewall Configuration**:
Ensure access to:
- `https://go.dev` (Go official releases and API)
- `https://golang.org` (Go downloads)
- `https://github.com` (govman releases and updates)
- `https://api.github.com` (self-update feature)

**Proxy Support**:
govman respects standard `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment variables.

## Shell Compatibility

govman provides shell integration for automatic version switching:

### Fully Supported Shells

| Shell      | Platform        | Auto-Switch | Wrapper Function | Notes                |
|------------|-----------------|-------------|------------------|----------------------|
| Bash       | Linux/macOS     | ✅           | ✅                | Uses PROMPT_COMMAND  |
| Zsh        | Linux/macOS     | ✅           | ✅                | Uses chpwd hook      |
| Fish       | Linux/macOS     | ✅           | ✅                | Native fish support  |
| PowerShell | Windows         | ✅           | ✅                | PowerShell 5.1+      |

### Limited Support

| Shell | Platform | Auto-Switch | Notes                                      |
|-------|----------|-------------|--------------------------------------------|
| Cmd   | Windows  | ❌           | Basic wrapper only, no auto-switch         |
| Sh    |Linux/macOS| Partial    | Basic PATH management, limited integration |

## Permissions

### Linux/macOS

- **No root/sudo required** for installation and usage
- Write access to `~/.govman/` directory
- Write access to shell configuration files (`.bashrc`, `.zshrc`, etc.)

### Windows

- **No administrator privileges required**
- Write access to `%USERPROFILE%\.govman\`
- Ability to modify user PATH environment variable
- PowerShell execution policy must allow running scripts

To configure PowerShell execution policy:
```powershell
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser
```

## Optional Dependencies

### For Development

If you plan to build govman from source:
- Go 1.25 or later
- Git
- Make (on Linux/macOS)

### For Enhanced Features

- **Git**: For project-based workflows and version control integration
- **jq**: For parsing JSON configuration (though not required by govman)
- **direnv**: Can work alongside govman for environment management

## Compatibility Notes

### Darwin/ARM64 (Apple Silicon)

- govman fully supports Apple Silicon (M1, M2, M3)
- For Go versions before 1.16, govman automatically falls back to amd64 binaries (which run via Rosetta 2)

### WSL (Windows Subsystem for Linux)

- govman works fully in WSL
- Use the Linux installation method within WSL
- Shell integration works as on native Linux

### Docker/Containers

govman can be used in containers, but note:
- Shell auto-switching requires shell integration setup
- Consider using explicit `govman use` commands in Dockerfiles
- May need to install download dependencies (`curl`, `tar`) in base images

## Known Limitations

1. **Command Prompt (Windows)**: No support for automatic version switching (.govman-goversion files)
2. **Network Isolation**: govman requires internet connectivity for most operations
3. **Concurrent Installations**: Multiple simultaneous `govman install` commands may conflict
4. **Symlink Support**: Some restricted Windows environments may have symlink limitations

## Verification

To verify your system meets the requirements:

```bash
# Check shell
echo $SHELL  # Linux/macOS
echo $0      # Current shell

# Check curl/wget
curl --version
wget --version

# Check tar/gzip
tar --version
gzip --version

# Check available disk space
df -h ~/.govman  # Linux/macOS
dir %USERPROFILE%\.govman  # Windows
```
