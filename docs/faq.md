# FAQ (Frequently Asked Questions)

## General Questions

### What is govman?

govman is a Go version manager that allows you to easily install, manage, and switch between multiple Go versions on your system without requiring admin/sudo privileges.

### Why use govman instead of installing Go directly?

- **Multiple versions**: Manage multiple Go versions simultaneously
- **Easy switching**: Quickly switch between versions per-project
- **No sudo required**: Fully userspace installation
- **Project-specific versions**: Use `.govman-goversion` files for automatic switching
- **Offline mode**: Intelligent caching with offline support
- **Cross-platform**: Works on Linux, macOS, and Windows

### Is govman compatible with other Go version managers?

govman works independently. It's recommended to uninstall other version managers (like `gvm`, `goenv`, `asdf`) before using govman to avoid conflicts in PATH management.

## Installation & Setup

### Do I need admin/sudo privileges to install govman?

No. govman is designed to work entirely in userspace without requiring admin or sudo privileges on any platform.

### Where does govman install Go versions?

-  **Linux/macOS**: `~/.govman/versions/`
- **Windows**: `%USERPROFILE%\.govman\versions\`

### Can I change the installation directory?

Yes, edit `~/.govman/config.yaml`:

```yaml
install_dir: /custom/path/to/versions
```

### How much disk space does govman need?

- govman binary: ~10-20 MB
- Per Go version: ~100-500 MB
- Recommended minimum: 1 GB free

You can clean the download cache anytime with `govman clean`.

## Version Management

### What does "latest" mean when installing?

`latest` refers to the newest **stable** release of Go. Pre-release versions (beta, rc) are not included unless explicitly specified.

```bash
govman install latest      # Latest stable
govman install 1.25rc1     # Specific pre-release
```

### Can I install beta or release candidate versions?

Yes:

```bash
govman install 1.25rc1     # Install RC version
govman list --remote --beta # List pre-releases
```

### How do I see all available Go versions?

```bash
govman list --remote         # Stable versions only
govman list --remote --beta  # Include pre-releases
```

### Can I use partial version numbers?

Yes, govman will resolve to the latest patch version:

```bash
govman install 1.25    # Installs latest 1.25.x (e.g., 1.25.1)
```

### What's the difference between `use`, `use --default`, and `use --local`?

- `govman use 1.25.1`: Temporary (session-only)
- `govman use 1.25.1 --default`: System-wide default (persistent)
- `govman use 1.25.1 --local`: Project-specific (creates `.govman-goversion`)

## Shell Integration

### Why doesn't `govman use` work in my current shell session?

Ensure you've run `govman init` and reloaded your shell:

```bash
govman init
source ~/.bashrc  # or ~/.zshrc, etc.
```

The `govman` wrapper function must be loaded for PATH updates to work.

### What is `.govman-goversion` and how does it work?

`.govman-goversion` is a project file containing the required Go version:

```
1.25.1
```

When you navigate to that directory, govman automatically switches to the specified version (if shell integration is enabled).

### Does auto-switching work in subdirectories?

Yes. govman searches for `.govman-goversion` in the current directory and walks up the directory tree until it finds one.

### Can I disable auto-switching temporarily?

Yes, set in `~/.govman/config.yaml`:

```yaml
auto_switch:
  enabled: false
```

### Which shells support auto-switching?

- **Full support**: Bash, Zsh, Fish, PowerShell
- **Limited support**: Command Prompt (cmd.exe) - no auto-switching

## Compatibility & Platform

### Does govman work on Apple Silicon (M1/M2/M3)?

Yes, govman fully supports Apple Silicon. For Go versions before 1.16, govman automatically uses amd64 binaries (which run via Rosetta 2).

### Can I use govman in Docker containers?

Yes, but note:
- You'll need to run `govman init` in the container
- For Dockerfiles, use explicit `govman use` commands
- Ensure download dependencies (`curl`, `tar`) are installed

### Does govman work with WSL?

Yes, use the Linux installation method within WSL. Shell integration works as on native Linux.

### Can I use govman in CI/CD pipelines?

Yes:

```bash
# Install govman (in CI)
curl -sSL https://govman.example.com/install.sh | bash

# Install specific Go version
govman install 1.25.1
govman use 1.25.1

# Run build
go build ./...
```

Alternatively, use `.govman-goversion` in your repository for consistency.

## Troubleshooting

### "go: command not found" after installation

Ensure:
1. Shell integration is set up: `govman init`
2. Shell config is reloaded: `source ~/.bashrc`
3. `~/.govman/bin` is in PATH
4. You've activated a Go version: `govman use latest --default`

### "Permission denied" errors

govman should not require sudo. If you see permission errors:
- Check `~/.govman` directory permissions
- Ensure you're not trying to install to system directories
- Try: `chmod -R u+w ~/.govman`

### Download failures or slow downloads

1. Check your internet connection
2. Try again (govman auto-retries failed downloads)
3. Use a mirror if in a restricted region:
   ```yaml
   mirror:
     enabled: true
     url: https://golang.google.cn/dl/
   ```
4. Increase timeout in config:
   ```yaml
   download:
     timeout: 600s
   ```

### "Checksum verification failed"

This indicates a corrupted download:
1. Run `govman clean` to remove cached files
2. Try installing again
3. Check your internet connection stability

### Multiple Go versions in PATH

Ensure other Go installation methods are removed or ensure `~/.govman/bin` appears first in PATH:

```bash
echo $PATH | tr ':' '\n'  # Check PATH order
```

### Auto-switching isn't working

1. Verify shell integration: `type govman_auto_switch`
2. Check config: `cat ~/.govman/config.yaml | grep auto_switch`
3. Verify `.govman-goversion` file: `cat .govman-goversion`
4. Check if auto-switch is enabled in config

### "No Go version is currently active"

Activate a version:

```bash
govman list                    # See installed versions
govman use 1.25.1 --default    # Set as default
```

## Updates & Maintenance

### How do I update govman itself?

```bash
govman selfupdate
```

### How do I update my Go versions?

govman doesn't automatically update Go installations. To upgrade:

```bash
# Check for new versions
govman list --remote

# Install newer version
govman install 1.26.0

# Switch to it
govman use 1.26.0 --default

# (Optional) Remove old version
govman uninstall 1.25.1
```

### How do I clean up disk space?

```bash
# Remove download cache only (keeps installed versions)
govman clean

# Remove unused Go versions
govman list
govman uninstall 1.old.0
```

### What happens to my Go versions if I uninstall govman?

- **Minimal uninstall**: Go versions are preserved
- **Complete uninstall**: Everything including Go versions is removed

You can choose during uninstallation.

## Advanced Usage

### Can I use govman with Docker multi-stage builds?

Yes:

```dockerfile
FROM golang:1.25 AS builder

# Install govman
RUN curl -sSL https://govman.example.com/install.sh | bash

# Use specific version
RUN govman install 1.25.1
RUN govman use 1.25.1

# Build
COPY . /app
WORKDIR /app
RUN go build -o myapp
```

### Can I use mirrors for faster downloads?

Yes, edit `~/.govman/config.yaml`:

```yaml
mirror:
  enabled: true
  url: https://golang.google.cn/dl/
```

### How do I use govman behind a corporate proxy?

Set standard environment variables:

```bash
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080
export NO_PROXY=localhost,127.0.0.1
```

### Can I install Go versions offline?

Not directly, as govman downloads from official sources. However:
1. Cache is preserved after first download
2. You can manually place archives in `~/.govman/cache/`
3. Set up a local mirror server

## Comparison with Other Tools

### govman vs gvm

- govman: Cross-platform (including Windows), simpler setup, no shell-specific dependencies
- gvm: Linux/macOS only, requires bash, more complex setup

### govman vs goenv

- govman: Native Go implementation, faster, built-in shell integration
- goenv: Shell script based, requires external dependencies (git, bash)

### govman vs asdf-golang

- govman: Dedicated Go version manager, optimized for Go workflows
- asdf: General-purpose version manager for multiple languages

### govman vs official Go installation

- govman: Multiple versions, easy switching, project-specific versions
- Official: Single version, manual management, system-wide only

## Security & Privacy

### Does govman phone home or track usage?

No. govman only contacts:
- `go.dev` - to fetch available Go releases
- `github.com` - for self-updates (only when you run `govman selfupdate`)

### Are downloads verified?

Yes. govman verifies SHA-256 checksums for all Go downloads against official releases.

### Where does govman store data?

All data is stored in `~/.govman/` (or `%USERPROFILE%\.govman\` on Windows). govman never writes to system directories.

## Getting Help

### Where can I report bugs?

Open an issue on the GitHub repository with:
- govman version (`govman --version`)
- Operating system and version
- Shell type and version
- Complete error message
- Steps to reproduce

### How do I request a feature?

Open a feature request on GitHub describing:
- The use case
- Expected behavior
- Why it would be useful

### Where can I find more documentation?

- [Commands Reference](commands.md) - All commands and flags
- [Shell Integration](shell-integration.md) - Advanced shell features
- [Configuration](configuration.md) - Config file options
- [Troubleshooting](troubleshooting.md) - Common issues and solutions
