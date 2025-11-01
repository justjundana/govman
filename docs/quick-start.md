# Quick Start Guide

Get up and running with **govman** in minutes!

## What is govman?

**govman** (Go Version Manager) is a fast, lightweight tool for installing and managing multiple Go versions on your system. Switch between Go versions instantly for different projects.

## Installation

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
```

### Windows (PowerShell)

```powershell
iex (iwr -useb https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.ps1)
```

### Windows (Command Prompt)

```cmd
curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.bat -o install.bat && install.bat
```

## First Steps

### 1. Initialize Shell Integration

After installation, set up your shell for automatic version switching:

```bash
govman init
```

Then restart your terminal or reload your shell configuration:

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

### 2. Install Your First Go Version

Install the latest stable Go version:

```bash
govman install latest
```

Or install a specific version:

```bash
govman install 1.21.5
```

### 3. Activate the Go Version

Set it as your default version:

```bash
govman use 1.21.5 --default
```

Or use it just for the current session:

```bash
govman use 1.21.5
```

### 4. Verify Installation

```bash
go version
```

## Essential Commands

### View Available Versions

See all Go versions available for installation:

```bash
govman list --remote
```

See installed versions:

```bash
govman list
```

### Check Current Version

```bash
govman current
```

### Install Multiple Versions

```bash
govman install 1.21.5 1.20.12 1.19.13
```

### Project-Specific Versions

Create a `.govman-version` file in your project:

```bash
echo "1.21.5" > .govman-version
```

govman will automatically switch to this version when you enter the directory!

### Clean Up Cache

Free up disk space by cleaning download cache:

```bash
govman clean
```

## Common Workflows

### Switching Between Projects

```bash
# Project A uses Go 1.21
cd ~/projects/project-a
govman use 1.21.5 --local

# Project B uses Go 1.20
cd ~/projects/project-b
govman use 1.20.12 --local

# Now it auto-switches!
cd ~/projects/project-a
go version  # Shows 1.21.5

cd ~/projects/project-b
go version  # Shows 1.20.12
```

### Testing with Multiple Versions

```bash
# Install multiple versions
govman install 1.21.5 1.20.12 1.19.13

# Test your code
govman use 1.21.5
go test ./...

govman use 1.20.12
go test ./...

govman use 1.19.13
go test ./...
```

### Keeping govman Updated

```bash
# Check for updates
govman selfupdate --check

# Update to latest version
govman selfupdate
```

## Quick Reference Card

| Command | Description |
|---------|-------------|
| `govman install <version>` | Install a Go version |
| `govman use <version>` | Switch to a Go version |
| `govman use <version> --default` | Set as system default |
| `govman use <version> --local` | Set for current project |
| `govman list` | Show installed versions |
| `govman list --remote` | Show available versions |
| `govman current` | Show active version |
| `govman info <version>` | Show version details |
| `govman uninstall <version>` | Remove a version |
| `govman clean` | Clean download cache |
| `govman init` | Setup shell integration |
| `govman selfupdate` | Update govman itself |

## Next Steps

- 📖 Read the [Installation Guide](installation.md) for advanced options
- ⚙️ Learn about [Configuration](configuration.md) options
- 🐚 Explore [Shell Integration](shell-integration.md) features
- 📚 Browse the [Commands Reference](commands.md) for all commands
- ❓ Check [Troubleshooting](troubleshooting.md) if you encounter issues

## Get Help

```bash
# General help
govman --help

# Command-specific help
govman install --help
govman use --help
```

## Features at a Glance

✅ **Lightning-fast** installation and switching  
✅ **Zero configuration** - works out of the box  
✅ **Project-specific** versions with `.govman-version`  
✅ **No admin/sudo** required  
✅ **Intelligent caching** with offline mode  
✅ **Parallel downloads** with resume support  
✅ **Cross-platform** - Windows, macOS, Linux, ARM  
✅ **Built-in cleanup** tools  

---

Happy Go development! 🚀
