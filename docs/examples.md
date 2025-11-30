# Examples

Common workflows and usage patterns for govman.

## Basic Workflows

### First-Time Setup

```bash
# 1. Install govman (already done)

# 2. Initialize shell integration
govman init

# 3. Reload shell
source ~/.bashrc  # or ~/.zshrc

# 4. Install latest Go
govman install latest

# 5. Activate it as default
govman use latest --default

# 6. Verify
go version
```

### Installing Specific Versions

```bash
# Install latest stable
govman install latest

# Install specific version
govman install 1.25.1

# Install latest patch of a minor version
govman install 1.24

# Install pre-release
govman install 1.26rc1
```

### Switching Between Versions

```bash
# Temporary (current session only)
govman use 1.25.1

# Set as system default
govman use 1.25.1 --default

# Set for current project
govman use 1.25.1 --local

# Switch back to default
govman use default
```

## Project-Based Workflows

### Setting Up a New Project

```bash
# Create project directory
mkdir my-go-project
cd my-go-project

# Set Go version for this project
govman use 1.25.1 --local

# Verify .govman-goversion was created
cat .govman-goversion
# Output: 1.25.1

# Add to git
git init
git add .govman-goversion
git commit -m "Set Go version to 1.25.1"

#Initialize Go module
go mod init github.com/username/my-go-project
```

### Cloning a Project with .govman-goversion

```bash
# Clone repository
git clone https://github.com/username/project.git
cd project

# Check required version
cat .govman-goversion
# Output: 1.24.0

# Install if not already installed
govman install 1.24.0

# Version automatically switches with shell integration
# Or manually switch:
govman use 1.24.0

# Verify
go version
```

### Team Development

```bash
# Team lead sets version
govman use 1.25.1 --local
git add .govman-goversion
git commit -m "Set Go 1.25.1 for project"
git push

# Team members clone and install
git pull
govman install $(cat .govman-goversion)
# Auto-switches when entering directory
```

## Multi-Version Development

### Testing Across Multiple Go Versions

```bash
# Install multiple versions
govman install 1.25.1 1.24.0 1.23.5

# Test with each version
for version in 1.25.1 1.24.0 1.23.5; do
  echo "Testing with Go $version"
  govman use $version
  go test ./...
done

# Back to default
govman use default
```

### Version-Specific Build

```bash
# Build with specific Go version
govman use 1.25.1
go build -o myapp-go1.25 ./cmd/myapp

govman use 1.24.0
go build -o myapp-go1.24 ./cmd/myapp

# Compare binaries
ls -lh myapp-*
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Build
on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      # Install govman
      - name: Install govman
        run: |
          curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
          echo "$HOME/.govman/bin" >> $GITHUB_PATH
      
      # Install Go version from .govman-goversion
      - name: Install Go
        run: |
          govman install $(cat .govman-goversion)
          govman use $(cat .govman-goversion)
      
      # Build
      - name: Build
        run: go build ./...
      
      # Test
      - name: Test
        run: go test ./...
```

### GitLab CI

```yaml
build:
  image: ubuntu:latest
  before_script:
    # Install dependencies
    - apt-get update && apt-get install -y curl tar gzip git
    
    # Install govman
    - curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
    - export PATH="$HOME/.govman/bin:$PATH"
    
    # Install and use Go version
    - govman install $(cat .govman-goversion)
    - govman use $(cat .govman-goversion)
  
  script:
    - go version
    - go build ./...
    - go test ./...
```

### Docker

```dockerfile
FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y curl tar gzip

# Install govman
RUN curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash

# Add govman to PATH
ENV PATH="/root/.govman/bin:$PATH"

# Set working directory
WORKDIR /app

# Copy project files
COPY . .

# Install Go version from .govman-goversion
RUN govman install $(cat .govman-goversion) && \
    govman use $(cat .govman-goversion)

# Build application
RUN go build -o myapp ./cmd/myapp

CMD ["./myapp"]
```

## Version Management

### Listing and Exploring Versions

```bash
# List installed versions
govman list

# List available remote versions
govman list --remote

# Filter by pattern
govman list --remote --pattern "1.25*"

# Include beta/RC versions
govman list --remote --beta

# Get detailed info about a version
govman info 1.25.1
```

### Upgrading Go

```bash
# Check for new versions
govman list --remote

# Install newer version
govman install 1.26.0

# Test with new version
govman use 1.26.0
go test ./...

# If all good, set as default
govman use 1.26.0 --default

# Remove old version
govman uninstall 1.25.1
```

### Downgrading Go

```bash
# List installed versions
govman list

# Switch to older version
govman use 1.24.0 --default

# Or install older version if not present
govman install 1.24.0
govman use 1.24.0 --default
```

## Maintenance

### Cleaning Up Disk Space

```bash
# Check disk usage
govman list  # Shows size of each version

# Remove multiple old versions at once
govman uninstall 1.23.0 1.22.0 1.21.1

# Clean download cache
govman clean

# View freed space
govman list
```

### Backing Up Configuration

```bash
# Backup config and version list
cp ~/.govman/config.yaml ~/.govman/config.yaml.backup

# Save list of installed versions
govman list > ~/govman-versions-backup.txt

# Restore later: install versions from list
cat ~/govman-versions-backup.txt | grep "Installed" | awk '{print $2}' | while read version; do
  govman install $version
done
```

## Advanced Workflows

### Pre-Release Testing

```bash
# Install release candidate
govman install 1.26rc1

# Test in isolated environment
cd /tmp/test-project
go mod init test
govman use 1 .26rc1

# Run tests
go test std

# Report issues if found
```

### Custom Installation Directory

```bash
# Edit config
nano ~/.govman/config.yaml

# Change install directory
install_dir: /opt/go/versions

# Install new version (goes to new directory)
govman install 1.25.1
```

### Using Mirrors

```bash
# For users in restricted regions
nano ~/.govman/config.yaml

# Enable mirror
mirror:
  enabled: true
  url: https://golang.google.cn/dl/

# Install will now use mirror
govman install latest
```

### Corporate Proxy Setup

```bash
# Set proxy environment variables
export HTTP_PROXY=http://proxy.corp.com:8080
export HTTPS_PROXY=http://proxy.corp.com:8080
export NO_PROXY=localhost,127.0.0.1,.corp.com

# Now use govman normally
govman install latest
```

## Shell-Specific Workflows

### Bash/Zsh

```bash
# Add to ~/.bashrc or ~/.zshrc for quick aliases
alias goi='govman install'
alias gou='govman use'
alias gol='govman list'
alias goc='govman current'

# Use aliases
goi latest
gou latest --default
gol
goc
```

### Fish

```fish
# Add to ~/.config/fish/config.fish
abbr -a goi govman install
abbr -a gou govman use
abbr -a gol govman list
abbr -a goc govman current

# Use abbreviations
goi latest
gou latest --default
```

### PowerShell

```powershell
# Add to $PROFILE
function goi { govman install $args }
function gou { govman use $args }
function gol { govman list $args }
function goc { govman current }

# Use functions
goi latest
gou latest --default
```

## Troubleshooting Workflows

### Version Not Switching

```bash
# Check if shell integration is loaded
type govman_auto_switch

# Re-initialize if needed
govman init --force
source ~/.bashrc

# Manually trigger auto-switch
govman_auto_switch

# Or use refresh command
govman refresh
```

### Download Failures

```bash
# Clean cache and retry
govman clean
govman install 1.25.1

# Check with verbose mode
govman --verbose install 1.25.1

# Try with increased timeout
nano ~/.govman/config.yaml
# Set: download.timeout: 600s
govman install 1.25.1
```

### Checking Current Setup

```bash
# Comprehensive setup check
echo "=== govman Version ===" govman --version

echo "=== Currently Active ==="
govman current

echo "=== Installed Versions ==="
govman list

echo "=== Go Version ==="
go version

echo "=== Go Environment ==="
go env

echo "=== PATH ==="
echo $PATH | tr ':' '\n' | grep -E "(govman|go)"
```

## Migration Workflows

### From System Go to govman

```bash
# Check current Go version
go version  # e.g., go1.24.0

# Install same version with govman
govman install 1.24.0
govman use 1.24.0 --default

# Remove system Go (if desired)
# sudo rm -rf /usr/local/go  # or appropriate path

# Verify
which go  # Should show govman path
go version
```

### From Other Version Managers

```bash
# Example: migrating from gvm

# List current gvm versions
gvm list

# Install same versions with govman
govman install 1.25.1 1.24.0 1.23.5

# Set default
govman use 1.25.1 --default

# Uninstall gvm
# (Follow gvm-specific uninstall instructions)

# Verify
govman list
go version
```
