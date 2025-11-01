# Configuration Guide

Complete reference for configuring **govman** to suit your workflow.

## Configuration File

govman stores its configuration in `~/.govman/config.yaml`. This file is automatically created on first run with sensible defaults.

### Location

- **macOS/Linux**: `~/.govman/config.yaml`
- **Windows**: `%USERPROFILE%\.govman\config.yaml`

### Default Configuration

```yaml
# Installation directory for Go versions
install_dir: ~/.govman/versions

# Cache directory for downloads
cache_dir: ~/.govman/cache

# Default Go version (empty = none set)
default_version: ""

# Quiet mode (errors only)
quiet: false

# Verbose mode (detailed output)
verbose: false

# Download configuration
download:
  parallel: true
  max_connections: 4
  timeout: 300s
  retry_count: 3
  retry_delay: 5s

# Mirror configuration
mirror:
  enabled: false
  url: "https://golang.google.cn/dl/"

# Auto-switch configuration
auto_switch:
  enabled: true
  project_file: ".govman-version"

# Shell configuration
shell:
  auto_detect: true
  completion: true

# Go releases API configuration
go_releases:
  api_url: "https://go.dev/dl/?mode=json&include=all"
  download_url: "https://go.dev/dl/%s"
  cache_expiry: 10m

# Self-update configuration
self_update:
  github_api_url: "https://api.github.com/repos/justjundana/govman/releases/latest"
  github_releases_url: "https://api.github.com/repos/justjundana/govman/releases?per_page=1"
```

## Configuration Options

### Basic Settings

#### `install_dir`
**Type**: `string`  
**Default**: `~/.govman/versions`

Directory where Go versions are installed.

```yaml
install_dir: ~/custom/go/versions
```

#### `cache_dir`
**Type**: `string`  
**Default**: `~/.govman/cache`

Directory for caching downloaded archives.

```yaml
cache_dir: /tmp/govman-cache
```

#### `default_version`
**Type**: `string`  
**Default**: `""` (empty)

The default Go version to use when no project-specific version is set.

```yaml
default_version: "1.21.5"
```

This is automatically set when you run:
```bash
govman use 1.21.5 --default
```

#### `quiet`
**Type**: `boolean`  
**Default**: `false`

Suppress all output except errors.

```yaml
quiet: true
```

Or use the command-line flag:
```bash
govman --quiet install 1.21.5
```

#### `verbose`
**Type**: `boolean`  
**Default**: `false`

Show detailed debug information.

```yaml
verbose: true
```

Or use the command-line flag:
```bash
govman --verbose install 1.21.5
```

### Download Configuration

#### `download.parallel`
**Type**: `boolean`  
**Default**: `true`

Enable parallel downloads for faster installation.

```yaml
download:
  parallel: true
```

#### `download.max_connections`
**Type**: `integer`  
**Default**: `4`

Maximum number of concurrent download connections.

```yaml
download:
  max_connections: 8
```

#### `download.timeout`
**Type**: `duration`  
**Default**: `300s` (5 minutes)

Timeout for download operations.

```yaml
download:
  timeout: 600s  # 10 minutes
```

#### `download.retry_count`
**Type**: `integer`  
**Default**: `3`

Number of retry attempts for failed downloads.

```yaml
download:
  retry_count: 5
```

#### `download.retry_delay`
**Type**: `duration`  
**Default**: `5s`

Delay between retry attempts.

```yaml
download:
  retry_delay: 10s
```

### Mirror Configuration

Use a mirror for faster downloads or when the official site is blocked.

#### `mirror.enabled`
**Type**: `boolean`  
**Default**: `false`

Enable download mirror.

```yaml
mirror:
  enabled: true
```

#### `mirror.url`
**Type**: `string`  
**Default**: `"https://golang.google.cn/dl/"`

Mirror URL for downloading Go distributions.

```yaml
mirror:
  enabled: true
  url: "https://golang.google.cn/dl/"  # China mirror
```

**Available Mirrors:**
- `https://golang.google.cn/dl/` - Google China
- `https://go.dev/dl/` - Official (default)
- Custom mirrors must follow the same URL structure

### Auto-Switch Configuration

#### `auto_switch.enabled`
**Type**: `boolean`  
**Default**: `true`

Enable automatic version switching when entering directories with `.govman-version` files.

```yaml
auto_switch:
  enabled: true
```

Disable if you prefer manual version switching:
```yaml
auto_switch:
  enabled: false
```

#### `auto_switch.project_file`
**Type**: `string`  
**Default**: `".govman-version"`

Name of the project version file.

```yaml
auto_switch:
  project_file: ".govman-version"  # Default filename
```

### Shell Configuration

#### `shell.auto_detect`
**Type**: `boolean`  
**Default**: `true`

Automatically detect the current shell.

```yaml
shell:
  auto_detect: true
```

#### `shell.completion`
**Type**: `boolean`  
**Default**: `true`

Enable shell completion features.

```yaml
shell:
  completion: true
```

### Go Releases Configuration

#### `go_releases.api_url`
**Type**: `string`  
**Default**: `"https://go.dev/dl/?mode=json&include=all"`

API endpoint for fetching available Go versions.

```yaml
go_releases:
  api_url: "https://go.dev/dl/?mode=json&include=all"
```

#### `go_releases.download_url`
**Type**: `string`  
**Default**: `"https://go.dev/dl/%s"`

Template URL for downloading Go distributions. `%s` is replaced with the filename.

```yaml
go_releases:
  download_url: "https://go.dev/dl/%s"
```

#### `go_releases.cache_expiry`
**Type**: `duration`  
**Default**: `10m`

How long to cache the list of available Go versions.

```yaml
go_releases:
  cache_expiry: 30m  # Cache for 30 minutes
```

### Self-Update Configuration

#### `self_update.github_api_url`
**Type**: `string`  
**Default**: `"https://api.github.com/repos/justjundana/govman/releases/latest"`

GitHub API URL for checking govman updates.

```yaml
self_update:
  github_api_url: "https://api.github.com/repos/justjundana/govman/releases/latest"
```

#### `self_update.github_releases_url`
**Type**: `string`  
**Default**: `"https://api.github.com/repos/justjundana/govman/releases?per_page=1"`

GitHub API URL for fetching release list (including pre-releases).

```yaml
self_update:
  github_releases_url: "https://api.github.com/repos/justjundana/govman/releases?per_page=1"
```

## Command-Line Flags

Configuration can be overridden using command-line flags:

### Global Flags

```bash
# Use custom config file
govman --config /path/to/config.yaml list

# Verbose output
govman --verbose install 1.21.5

# Quiet mode
govman --quiet install 1.21.5
```

## Environment Variables

Some settings can be controlled via environment variables:

### Network Proxy

```bash
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080
export NO_PROXY=localhost,127.0.0.1
```

### Custom Config Location

```bash
export GOVMAN_CONFIG=~/.config/govman/config.yaml
govman list
```

## Configuration Examples

### Optimized for Fast Networks

```yaml
download:
  parallel: true
  max_connections: 8
  timeout: 600s
  retry_count: 5
  retry_delay: 3s

go_releases:
  cache_expiry: 30m
```

### Optimized for Slow/Unreliable Networks

```yaml
download:
  parallel: false
  max_connections: 2
  timeout: 1800s  # 30 minutes
  retry_count: 10
  retry_delay: 15s
```

### Corporate Environment with Proxy

```yaml
mirror:
  enabled: true
  url: "https://internal-mirror.company.com/go/"

download:
  timeout: 900s
  retry_count: 5
```

### Minimal Disk Usage

```yaml
cache_dir: /tmp/govman-cache  # Use temp directory

go_releases:
  cache_expiry: 5m  # Shorter cache
```

### Multiple Go Versions Development

```yaml
auto_switch:
  enabled: true
  project_file: ".govman-version"

download:
  parallel: true
  max_connections: 6
```

### CI/CD Environment

```yaml
quiet: false
verbose: true

download:
  parallel: true
  max_connections: 4
  timeout: 600s
  retry_count: 3

auto_switch:
  enabled: false  # Manual control in CI
```

## Project-Specific Configuration

### Using `.govman-version`

Create a `.govman-version` file in your project root:

```bash
echo "1.21.5" > .govman-version
```

govman will automatically switch to this version when you `cd` into the directory (if auto-switch is enabled).

**File format:**
```
1.21.5
```

Just the version number, no prefix or quotes needed.

### Committing `.govman-version`

Add to your repository to ensure all team members use the same Go version:

```bash
git add .govman-version
git commit -m "Pin Go version to 1.21.5"
```

### `.gitignore` Recommendations

You typically **don't** want to ignore `.govman-version`:

```gitignore
# Don't add this:
# .govman-version
```

But you may want to ignore local overrides if you create them:

```gitignore
.govman-version.local
```

## Configuration Best Practices

### 1. Version Control Your Project Configuration

Always commit `.govman-version` to ensure consistent Go versions across your team.

### 2. Use Mirrors in Restricted Networks

If you're behind a firewall or in a region with poor connectivity to `go.dev`:

```yaml
mirror:
  enabled: true
  url: "https://golang.google.cn/dl/"
```

### 3. Adjust Timeouts for Your Network

For slow connections:
```yaml
download:
  timeout: 1800s
  retry_count: 10
```

For fast connections:
```yaml
download:
  timeout: 300s
  retry_count: 3
```

### 4. Disable Auto-Switch When Testing

If you're frequently testing different versions:

```yaml
auto_switch:
  enabled: false
```

Then manually switch:
```bash
govman use 1.21.5
```

### 5. Keep Cache on Fast Storage

For SSD-equipped systems:
```yaml
cache_dir: ~/.govman/cache  # On SSD
```

For systems with limited SSD space:
```yaml
cache_dir: /mnt/hdd/govman-cache  # On HDD
```

## Resetting Configuration

### Reset to Defaults

```bash
rm ~/.govman/config.yaml
govman list  # Will recreate with defaults
```

### Backup Configuration

```bash
cp ~/.govman/config.yaml ~/.govman/config.yaml.backup
```

### Restore from Backup

```bash
cp ~/.govman/config.yaml.backup ~/.govman/config.yaml
```

## Configuration Validation

govman validates configuration on startup. Invalid settings will show errors:

```bash
govman list
# Error: failed to load config: invalid timeout value
```

Check configuration syntax:

```bash
govman --verbose list
```

## Troubleshooting Configuration

### Configuration Not Loading

**Check file exists:**
```bash
ls -la ~/.govman/config.yaml
```

**Check permissions:**
```bash
chmod 644 ~/.govman/config.yaml
```

### Syntax Errors

YAML is whitespace-sensitive. Use 2 spaces for indentation:

❌ **Wrong:**
```yaml
download:
    parallel: true  # 4 spaces
```

✅ **Correct:**
```yaml
download:
  parallel: true  # 2 spaces
```

### Values Not Taking Effect

1. Verify the config file location
2. Check for command-line flag overrides
3. Restart your terminal after changes
4. Use `--verbose` to see what's being loaded

## See Also

- [Quick Start Guide](quick-start.md) - Get started quickly
- [Installation Guide](installation.md) - Install govman
- [Shell Integration](shell-integration.md) - Set up your shell
- [Commands Reference](commands.md) - All available commands
- [Troubleshooting](troubleshooting.md) - Common issues

---

Need help? Check the [troubleshooting guide](troubleshooting.md) or [open an issue](https://github.com/justjundana/govman/issues).
