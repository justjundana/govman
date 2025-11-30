# Quick Start

Get started with govman in under 5 minutes.

## Installation

### Linux/macOS

```bash
curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.ps1 | iex
```

### Windows (Command Prompt)

Download and run the batch script from the repository.

## Initialize Shell Integration

```bash
govman init
```

Then restart your terminal or source your shell configuration:

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

## Install Your First Go Version

```bash
# Install the latest stable version
govman install latest

# Install a specific version
govman install 1.25.1
```

## Activate a Go Version

```bash
# Session-only activation
govman use 1.25.1

# Set as system default
govman use 1.25.1 --default

# Set for current project
govman use 1.25.1 --local
```

## Verify Installation

```bash
govman current
go version
```

## Common Commands

```bash
# List installed Go versions
govman list

# List available remote versions
govman list --remote

# View version information
govman info 1.25.1

# Uninstall a version
govman uninstall 1.24.0

# Clean download cache
govman clean

# Update govman itself
govman selfupdate
```

## Next Steps

- See [Commands Reference](commands.md) for all available commands
- Read [Shell Integration](shell-integration.md) for advanced shell features
- Check [Examples](examples.md) for common workflows
