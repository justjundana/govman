# Migration Guide

Guide for migrating from other Go version managers to govman.

## From System Go Installation

### Current Setup

You have Go installed system-wide (e.g., via apt, brew, or manual installation).

### Migration Steps

```bash
# 1. Check current Go version
go version

# 2. Install govman
curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash

# 3. Initialize shell integration
govman init
source ~/.bashrc  # or ~/.zshrc

# 4. Install the same version with govman
govman install 1.25.1  # Replace with your current version

# 5. Activate it
govman use 1.25.1 --default

# 6. Verify
go version
which go  # Should show ~/.govman/versions/go1.25.1/bin/go

# 7. (Optional) Remove system Go
# Ubuntu/Debian:
sudo apt remove golang-go
sudo rm -rf /usr/local/go

# macOS:
brew uninstall go
# or manually: sudo rm -rf /usr/local/go

# Windows:
# Uninstall via Control Panel or Settings
```

### What Changes

- **Go binaries**: Now in `~/.govman/versions/` instead of `/usr/local/go` or system paths
- **PATH**: `~/.govman/bin` added to PATH
- **GOROOT**: Automatically managed (don't set manually)
- **Modules & workspace**: No changes, continue working normally

### Preserving Installed Packages

Your globally installed Go tools are preserved:

```bash
# Before migration
which gopls
# /home/user/go/bin/gopls

# After migration (same location)
which gopls
# /home/user/go/bin/gopls
```

`$GOPATH/bin` (typically `~/go/bin`) is automatically added to PATH by shell integration.

## From gvm

### Current Setup

Using [gvm](https://github.com/moovweb/gvm) for Go version management.

### Migration Steps

```bash
# 1. List currently installed versions
gvm list

# 2. Note your default version
gvm listdefault

# 3. Install govman
curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash

# 4. Initialize
govman init --force  # --force to override gvm's shell config
source ~/.bashrc

# 5. Install same versions
govman install 1.25.1 1.24.0  # Your versions from gvm

# 6. Set default
govman use 1.25.1 --default

# 7. Verify
go version

# 8. Uninstall gvm
rm -rf ~/.gvm
# Remove gvm lines from ~/.bashrc or ~/.zshrc
```

### Key Differences

| Feature                | gvm                        | govman                     |
|------------------------|----------------------------|----------------------------|
| Installation           | Requires bash, git         | Self-contained binary      |
| Windows support        | No                         | Yes                        |
| Shell support          | Bash only                  | Bash, Zsh, Fish, PowerShell|
| Download source        | Compiles from source       | Official pre-built binaries|
| Installation speed     | Slow (compilation)         | Fast (binary download)     |
| Version switching      | Shell function             | Binary + shell wrapper     |
| Project-local versions | Via .gvmrc                 | Via .govman-goversion        |

### Converting .gvmrc Files

```bash
# Old (.gvmrc):
echo "go1.25.1" > .gvmrc

# New (.govman-goversion):
echo "1.25.1" > .govman-goversion
```

## From goenv

### Current Setup

Using [goenv](https://github.com/syndbg/goenv) for version management.

### Migration Steps

```bash
# 1. Check installed versions
goenv versions

# 2. Note global version
goenv global

# 3. Install govman
curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash

# 4. Initialize (will update shell config)
govman init --force
source ~/.bashrc

# 5. Install versions
# goenv stores versions like "1.25.1", same as govman
govman install 1.25.1 1.24.0

# 6. Set global
govman use 1.25.1 --default

# 7. Convert local version files
find . -name ".go-version" -exec sh -c 'cp "$1" "$(dirname "$1")/.govman-goversion"' _ {} \;

# 8. Uninstall goenv
rm -rf ~/.goenv
# Remove goenv lines from shell config
```

### Key Differences

| Feature          | goenv                | govman                  |
|------------------|----------------------|-------------------------|
| Implementation   | Shell scripts        | Go binary               |
| Speed            | Moderate             | Fast                    |
| Windows          | Limited (WSL)        | Full support            |
| Local version file | .go-version        | .govman-goversion         |
| Shell integration | eval "$(goenv init -)" | govman init (automatic) |

### Converting .go-version Files

```bash
# Convert all .go-version to .govman-goversion
find . -name ".go-version" | while read file; do
    version=$(cat "$file")
    echo "$version" > "$(dirname "$file")/.govman-goversion"
    echo "Converted: $file"
done
```

## From asdf-golang

### Current Setup

Using [asdf](https://asdf-vm.com/) with golang plugin.

### Migration Steps

```bash
# 1. List versions
asdf list golang

# 2. Check global version
asdf current golang

# 3. Install govman
curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash

# 4. Initialize
govman init --force
source ~/.bashrc

# 5. Install versions
govman install 1.25.1 1.24.0

# 6. Set global
govman use 1.25.1 --default

# 7. Convert .tool-versions
# Extract golang version from .tool-versions and create .govman-goversion
awk '/^golang/ {print $2}' .tool-versions > .govman-goversion

# 8. (Optional) Uninstall asdf golang plugin
asdf plugin remove golang
```

### Key Differences

| Feature          | asdf-golang              | govman                     |
|------------------|--------------------------|----------------------------|
| Purpose          | Multi-language manager   | Go-specific                |
| Configuration    | .tool-versions           | .govman-goversion + config   |
| Performance      | Moderate                 | Fast (optimized for Go)    |
| Features         | General                  | Go-specific optimizations  |

## From Docker-based Workflow

### Current Setup

Using Docker containers with different Go versions.

### Migration to govman

```bash
# 1. Install govman
curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash

# 2. Initialize
govman init
source ~/.bashrc

# 3. Install Go versions you need
govman install 1.25.1 1.24.0

# 4. Use project-local versions
cd /project-using-1.25
govman use 1.25.1 --local

cd /project-using-1.24
govman use 1.24.0 --local

# 5. Auto-switches when cd'ing between projects
```

### Hybrid Approach

You can use both govman and Docker:

```dockerfile
# Dockerfile with govman
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y curl tar gzip

# Install govman
RUN curl -sSL https://install.script | bash

# Install Go version
RUN ~/.govman/bin/govman install 1.25.1
RUN ~/.govman/bin/govman use 1.25.1

ENV PATH="/root/.govman/bin:$PATH"
```

## General Migration Checklist

- [ ] **Backup:**Back up shell configuration files
- [ ] **List versions:** Note all currently installed Go versions
- [ ] **Note default:** Record your default/global Go version
- [ ] **Install govman:** Run installation script
- [ ] **Initialize:** Run `govman init`
- [ ] **Reload shell:** `source ~/.bashrc` (or restart terminal)
- [ ] **Install versions:** Install needed Go versions with govman
- [ ] **Set default:** `govman use <version> --default`
- [ ] **Verify:** `go version` and `which go`
- [ ] **Convert project files:** Migrate .gvmrc, .go-version, etc.
- [ ] **Test builds:** Ensure projects still build correctly
- [ ] **Uninstall old:** Remove previous version manager
- [ ] **Clean up:** Remove old shell configuration lines

## Troubleshooting Migration

### "command not found" After Migration

**Cause:** Shell not reloaded or PATH not updated.

**Fix:**
```bash
source ~/.bashrc  # or ~/.zshrc
# Or restart terminal
```

### Multiple Version Managers Conflicting

**Cause:** Old version manager's PATH entries interfering.

**Fix:**
```bash
# Edit shell config
nano ~/.bashrc

# Remove lines from old version manager (between # gvm/goenv/asdf markers)
# Keep only GOVMAN section

source ~/.bashrc
```

### Go Version Unchanged

**Cause:** System Go still in PATH before govman.

**Fix:**
```bash
# Check PATH order
echo $PATH | tr ':' '\n'

# Ensure ~/.govman/bin comes first
# Edit ~/.bashrc to put govman PATH at top

export PATH="$HOME/.govman/bin:$PATH"  # Must be near top of file
```

### Old Project Version Files Not Working

**Cause:** govman doesn't recognize `.gvmrc`, `.go-version`, or `.tool-versions`.

**Fix:**
```bash
# Convert old files
echo "$(cat .gvmrc)" > .govman-goversion      # From gvm
echo "$(cat .go-version)" > .govman-goversion # From goenv
awk '/^golang/ {print $2}' .tool-versions > .govman-goversion  # From asdf
```

## Post-Migration Verification

```bash
# 1. Check govman is active
which govman
# Should show: ~/.govman/bin/govman

# 2. Check Go is from govman
which go
# Should show: ~/.govman/versions/go1.x.x/bin/go

# 3. Verify version
go version

# 4. Test version switching
govman use 1.24.0
go version  # Should change

govman use 1.25.1
go version  # Should change back

# 5. Test auto-switching
cd /project-with-govman-version
go version  # Should match .govman-goversion content

# 6. Test builds
cd /your-project
go build ./...
go test ./...
```

##Rollback (If Needed)

If migration doesn't work:

```bash
# 1. Remove govman
curl -sSL https://uninstall.script | bash

# 2. Reinstall old version manager
# (Follow their installation instructions)

# 3. Restore shell config from backup
cp ~/.bashrc.backup ~/.bashrc
source ~/.bashrc
```

## Getting Help

If you encounter issues during migration:

1. Check [Troubleshooting](troubleshooting.md)
2. Review [FAQ](faq.md)
3. Open an issue on GitHub with:
   - Previous version manager used
   - Operating system
   - Shell type
   - Error messages
   - Steps taken so far
