# Configuration

govman stores its configuration in a YAML file located at `~/.govman/config.yaml` (Linux/macOS) or `%USERPROFILE%\.govman\config.yaml` (Windows).

## Configuration File Structure

```yaml
# Default Version
default_version: ""

# Installation and Cache Directories  
install_dir: ~/.govman/versions
cache_dir: ~/.govman/cache

# Logging
quiet: false
verbose: false

# Download Settings
download:
  parallel: true
  max_connections: 4
  timeout: 300s
  retry_count: 3
  retry_delay: 5s

# Mirror Configuration
mirror:
  enabled: false
  url: https://golang.google.cn/dl/

# Auto-Switch Settings
auto_switch:
  enabled: true
  project_file: .govman-goversion

# Shell Integration
shell:
  auto_detect: true
  completion: true

# Go Releases API
go_releases:
  api_url: https://go.dev/dl/?mode=json&include=all
  download_url: https://go.dev/dl/%s
  cache_expiry: 10m

# Self-Update Settings
self_update:
  github_api_url: https://api.github.com/repos/justjundana/govman/releases/latest
  github_releases_url: https://api.github.com/repos/justjundana/govman/releases?per_page=1
```

## Configuration Options

### Default Version

```yaml
default_version: "1.25.1"
```

The Go version to use as the system default. Set via `govman use <version> --default`.

### Installation Directories

```yaml
install_dir: ~/.govman/versions
cache_dir: ~/.govman/cache
```

- `install_dir`: Where Go versions are installed
- `cache_dir`: Where downloaded archives are cached

**Note**: Paths support `~` expansion for the home directory.

###Logging

```yaml
quiet: false
verbose: false
```

- `quiet`: Suppress all output except errors
- `verbose`: Show detailed debug information

Can also be controlled via CLI flags:
```bash
govman --quiet list
govman --verbose install latest
```

### Download Settings

```yaml
download:
  parallel: true          # Enable parallel downloads
  max_connections: 4      # Maximum concurrent connections
  timeout: 300s           # Download timeout (seconds)
  retry_count: 3          # Number of retry attempts
  retry_delay: 5s         # Delay between retries (seconds)
```

**Download Features**:
- Parallel downloads for faster speeds
- Automatic resume on failure
- Configurable retry logic
- Progress bars with ETA

### Mirror Configuration

```yaml
mirror:
  enabled: false
  url: https://golang.google.cn/dl/
```

Use a mirror for downloading Go releases (useful in regions with restricted access to golang.org):

```yaml
mirror:
  enabled: true
  url: https://golang.google.cn/dl/
```

### Auto-Switch Settings

```yaml
auto_switch:
  enabled: true
  project_file: .govman-goversion
```

- `enabled`: Enable/disable automatic version switching
- `project_file`: Name of the project version file

When enabled, govman automatically switches Go versions when you navigate to directories containing `.govman-goversion` files.

**Note**: Requires shell integration (`govman init`).

### Shell Integration

```yaml
shell:
  auto_detect: true       # Automatically detect shell
  completion: true        # Enable shell completion
```

### Go Releases API

```yaml
go_releases:
  api_url: https://go.dev/dl/?mode=json&include=all
  download_url: https://go.dev/dl/%s
  cache_expiry: 10m       # How long to cache release data
```

- `api_url`: Endpoint for fetching Go release information
- `download_url`: Template for download URLs
- `cache_expiry`: Duration to cache release data (reduces API calls)

### Self-Update Settings

```yaml
self_update:
  github_api_url: https://api.github.com/repos/justjundana/govman/releases/latest
  github_releases_url: https://api.github.com/repos/justjundana/govman/releases?per_page=1
```

Endpoints for self-update feature. Modify if using a fork or custom release mechanism.

## Creating/Editing Configuration

### Initial Configuration

Config is automatically created on first run with default values.

### Manual Editing

```bash
# Linux/macOS
nano ~/.govman/config.yaml

# Windows
notepad %USERPROFILE%\.govman\config.yaml
```

### Resetting Configuration

Delete the config file to reset to defaults:

```bash
# Linux/macOS
rm ~/.govman/config.yaml

# Windows
del %USERPROFILE%\.govman\config.yaml
```

The config will be recreated with defaults on the next govman command.

## Environment Variables

govman respects standard environment variables:

### Proxy Settings

```bash
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080
export NO_PROXY=localhost,127.0.0.1
```

### Custom Config Path

```bash
govman --config /path/to/custom/config.yaml list
```

## Path Expansion

Configuration paths support tilde (`~`) expansion:

```yaml
install_dir: ~/custom/go/versions    # Expands to /home/user/custom/go/versions
cache_dir: ~/.cache/govman          # Expands to /home/user/.cache/govman
```

**Security**: Paths are validated to prevent directory traversal attacks.

## Best Practices

1. **Backup Configuration**: Before making major changes, backup your config:
   ```bash
   cp ~/.govman/config.yaml ~/.govman/config.yaml.backup
   ```

2. **Use Mirrors in Restricted Regions**: If you have slow access to golang.org:
   ```yaml
   mirror:
     enabled: true
     url: https://golang.google.cn/dl/
   ```

3. **Adjust Timeout for Slow Connections**:
   ```yaml
   download:
     timeout: 600s      # 10 minutes
     retry_count: 5
   ```

4. **Disable Auto-Switch for CI/CD**:
   ```yaml
   auto_switch:
     enabled: false
   ```

5. **Increase Cache Expiry to Reduce API Calls**:
   ```yaml
   go_releases:
     cache_expiry: 60m    # 1 hour
   ```

## Configuration Validation

govman validates configuration on load:

- **Path Validation**: Ensures paths don't contain `..` (directory traversal)
- **Type Validation**: Ensures correct data types (bool, int, duration)
- **Required Fields**: Ensures essential fields are present

Invalid configuration will result in errors with helpful messages:

```
Error: failed to load config: invalid path format: paths starting with ~ must be followed by / or \
```

## Advanced Configuration

### Custom Install Location

```yaml
install_dir: /opt/go/versions
```

Ensure the directory exists and you have write permissions.

### Network Timeouts for Slow Connections

```yaml
download:
  timeout: 900s          # 15 minutes
  retry_count: 5
  retry_delay: 10s
```

### Disable Parallel Downloads

```yaml
download:
  parallel: false
  max_connections: 1
```

Useful for debugging download issues or on systems with limited bandwidth.
