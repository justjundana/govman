# FAQ

## General

### What is govman?

govman installs and switches multiple Go toolchains in user-owned directories. It supports Linux, macOS, and Windows without administrator privileges for the default layout.

### Where is data stored?

Defaults are under `~/.govman` or `%USERPROFILE%\.govman`. Explicit config, install, cache, and shell-profile paths may be elsewhere.

### Does govman collect telemetry?

No. Network requests are limited to configured release metadata/download endpoints and explicit self-update checks.

## Versions

### What does `latest` mean?

For installation, it means the newest stable release returned by the configured Go release API. Pre-releases require an exact version or an unstable wildcard operation.

### What is the difference between partial and full versions?

`1.25` is flexible and selects the highest matching patch. `1.25.4` and prereleases such as `1.26rc1` are exact pins and are never substituted with another patch.

### How do I protect a version from `prune`?

Make it active, set it as default, or reference it from the current directory's `.govman-goversion`. `prune` keeps all three categories.

## Shell integration

### Why did `govman use` not change my current shell?

A child process cannot mutate its parent environment. Run `govman init`, reload the shell config, and use the generated wrapper. For a non-interactive POSIX shell:

```bash
eval "$(govman use 1.25.4)"
```

### Where is `.govman-goversion` searched?

The generated integration and `refresh` read the configured project filename in the current directory. They do not walk parent directories in v1.3.4.

### Which shells auto-switch?

Bash, Zsh, Fish, and PowerShell generate directory-change hooks. Command Prompt supports manual `use` through its wrapper but does not auto-switch on directory changes.

### How do I leave a project version?

With shell integration loaded, leaving a directory that activated a project version restores the configured default. `govman refresh` manually re-evaluates the current directory.

## Downloads

### Are downloads verified?

Yes. Go archives require matching filename, size, platform, and SHA-256 metadata. govman installer/self-update binaries require an exact entry in release `checksums.txt` and must report the target version.

### Are downloads parallel?

No. They are sequential in v1.3.4. `download.parallel` and `download.max_connections` remain in the config only for compatibility.

### Can an interrupted download resume?

Yes, when the server returns a valid byte range consistent with expected metadata. Otherwise govman restarts safely or reports an error.

### Does govman have an offline mode?

No supported offline or air-gapped workflow is provided. A valid cache hit can avoid retransferring an archive, but version resolution may still require the configured API.

### Does `mirror.enabled` reroute downloads?

No. `mirror.*` is reserved/no-op in v1.3.4. Organizations with a controlled compatible source must configure `go_releases.api_url` and `go_releases.download_url` explicitly.

### How do proxies work?

Use standard `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment variables.

## Troubleshooting

### `go` or `gofmt` is not found

1. Run `govman init --force`.
2. Reload the shell profile shown by the command.
3. Install a version and set it as default.
4. Verify that the selected toolchain `bin` directory precedes other Go paths.

```bash
govman install latest
govman use latest --default
command -v go
command -v gofmt
```

### A checksum failed

Do not bypass it. Run `govman clean`, retry, and inspect configured endpoints/proxies. Repeated mismatches should be reported with verbose output.

### Can I uninstall the active/default/local version?

Not until the reference is changed. Activate another default, leave or update the project file, then uninstall.

### How do I reclaim space?

`govman clean` removes cached archives. `govman prune` removes unused toolchains. `govman uninstall <version>` removes a specific unprotected toolchain.

## Updates and compatibility

### How do I update govman?

```bash
govman selfupdate --check
govman selfupdate
```

Development builds do not self-update. The update refuses automatic downgrade and requires checksum/version validation.

### Will v1.3.4 read my v1.3.3 config?

Yes. Reserved fields remain accepted, installed toolchain directory names remain compatible, and existing project files/shell marker blocks are migrated in place.

### Is `GOVMAN_HOME` supported?

No. Use `install_dir`, `cache_dir`, and `--config`.

## Help

Open a GitHub issue for non-sensitive bugs with the govman version, operating system, shell, config fields relevant to the problem, complete error, and reproduction steps. Use a private security advisory for vulnerabilities.
