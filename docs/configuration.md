# Configuration

govman reads YAML from `~/.govman/config.yaml` on Unix-like systems or `%USERPROFILE%\.govman\config.yaml` on Windows. Use `--config <path>` to select another file. A missing file is created with mode `0600`.

## Complete structure

```yaml
default_version: ""
install_dir: ~/.govman/versions
cache_dir: ~/.govman/cache
quiet: false
verbose: false

download:
  parallel: true          # Reserved compatibility field; no effect in v1.3.4
  max_connections: 4      # Reserved compatibility field; no effect in v1.3.4
  timeout: 300s
  retry_count: 3
  retry_delay: 5s

mirror:
  enabled: false          # Reserved compatibility field; no effect in v1.3.4
  url: https://golang.google.cn/dl/

auto_switch:
  enabled: true
  project_file: .govman-goversion

shell:
  auto_detect: true       # Reserved compatibility field; no effect in v1.3.4
  completion: true        # Reserved compatibility field; no effect in v1.3.4

go_releases:
  api_url: https://go.dev/dl/?mode=json&include=all
  download_url: https://go.dev/dl/%s
  cache_expiry: 10m

self_update:
  github_api_url: https://api.github.com/repos/justjundana/govman/releases/latest
  github_releases_url: https://api.github.com/repos/justjundana/govman/releases?per_page=1
```

Reserved fields remain readable and writable so v1.3.3 config files stay compatible. They must not be treated as active parallel download, mirror routing, shell detection, or completion settings in v1.3.4.

## Active options

### Paths

- `install_dir`: parent of managed directories such as `go1.25.4`.
- `cache_dir`: downloaded Go archives and resumable partial files.

`~` expands to the user home. Relative paths resolve against the config file directory. Install and cache paths must not be equal or nested inside one another.

### Output

- `quiet`: errors only; progress is disabled.
- `verbose`: detailed diagnostics and timers.

The values cannot both be true. Explicit `--quiet` or `--verbose` flags override file values for that command.

### Download

- `timeout`: positive HTTP client timeout.
- `retry_count`: at least one request attempt.
- `retry_delay`: non-negative delay used when the response does not provide a valid `Retry-After`.

Downloads are sequential. Resume requires a valid `206 Content-Range`; inconsistent range or size metadata causes a safe restart or failure.

### Auto-switch

- `enabled`: generated Bash, Zsh, Fish, and PowerShell integrations enable or disable directory-change switching.
- `project_file`: a filename only, without path separators. The integration reads it from the current directory.

Command Prompt does not provide directory-change auto-switching.

### Go release endpoints

- `api_url`: JSON release metadata endpoint.
- `download_url`: archive URL template containing exactly one `%s` filename placeholder.
- `cache_expiry`: positive in-memory release metadata lifetime.

To use a controlled internal source, configure both `go_releases.api_url` and `go_releases.download_url`. govman still requires metadata filename, size, platform, and SHA-256 values to match the downloaded archive.

### Self-update endpoints

The two `self_update` URLs support official GitHub release metadata or a compatible fork. Update asset names and `checksums.txt` entries must exactly match the current platform asset.

## Validation

Configuration loading uses strict decoding: unknown YAML keys and wrong types are errors. Validation also rejects:

- empty/invalid paths or overlapping install/cache roots;
- zero/negative timeout, retry count, or cache expiry;
- negative retry delay or `max_connections` below one;
- simultaneous quiet and verbose modes;
- project filenames containing separators, NUL, `.` or `..`;
- endpoint URLs that are not absolute HTTP(S), include credentials, or have an invalid download template.

Although reserved URLs are not used for routing, they must remain syntactically valid for compatibility and safe future migration.

## Proxy settings

Go's standard HTTP transport honors conventional proxy environment variables:

```bash
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080
export NO_PROXY=localhost,127.0.0.1
```

## Reset and backup

```bash
cp ~/.govman/config.yaml ~/.govman/config.yaml.backup
rm ~/.govman/config.yaml
govman list
```

The next command recreates a default config. govman refuses to load or replace a config path that is a symlink or non-regular file.
