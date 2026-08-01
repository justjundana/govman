# Architecture

govman is a single Go CLI. Commands orchestrate configuration, release metadata, downloads, managed installations, and shell integration; there is no daemon or background service.

## Package responsibilities

| Package | Responsibility |
|---|---|
| `cmd/govman` | Process entry point and final exit status |
| `internal/cli` | Cobra commands, argument validation, output ownership, self-update |
| `internal/config` | Isolated Viper decoding, defaults, validation, transactional persistence |
| `internal/manager` | Version resolution, install/uninstall, activation, local/default state |
| `internal/downloader` | HTTP retry/resume, cache commit, checksum, safe extraction |
| `internal/golang` | Go release API, SemVer comparison, install metadata |
| `internal/shell` | Shell detection, generated wrappers, transactional profile updates |
| `internal/symlink` | Refuse non-link destinations and atomically replace managed links |
| `internal/progress` | Writer-injected, quiet-aware transfer progress |
| `internal/logger` | Thread-safe output abstraction; injected into manager/downloader services |
| `internal/util` | Formatting and installed-version matching |
| `internal/version` | Build metadata supplied by release linker flags |

## Install flow

```text
CLI input
  -> strict version/alias resolution
  -> release metadata lookup
  -> unique cache partial + cross-process lock
  -> bounded retry/resume protocol
  -> size and SHA-256 verification
  -> unique sibling staging directory
  -> bounded link-free extraction through os.Root
  -> required executable validation
  -> atomic rename to final version directory
  -> atomic install metadata write
```

A failed download may retain one bounded partial for a future resume. A failed extraction removes staging. The final install directory appears only after validation.

## Activation flow

Session activation prints one shell-specific PATH command. The shell wrapper validates and evaluates that command exactly once.

Project-local activation snapshots `.govman-goversion`, writes the new exact version atomically, and restores the old file if PATH command generation fails.

Default activation snapshots managed toolchain links and the old config value, updates links for every regular executable in the selected Go `bin` directory, persists config, and rolls back both state sets if a later step fails.

## Managed state

- `config.yaml`: effective settings and default version, mode `0600`.
- `versions/go<version>`: committed Go toolchains.
- `versions/go<version>/.govman-install.json`: UTC install completion metadata.
- `cache`: verified archives, resumable partials, and short-lived lock files.
- `bin`: govman executable/wrapper and managed active toolchain links.
- `.govman-goversion`: current-directory project version pin.
- shell profile block: generated integration between exact govman markers.

Each config load owns a separate Viper instance. Release metadata cache is the only shared service cache; it is keyed by endpoint/policy, deduplicates concurrent fetches, and returns clones.

## Version semantics

- `latest` and `stable` resolve to the newest eligible stable release for install.
- Installed alias resolution selects the newest installed version.
- `major.minor` selects the highest matching installed patch where installed resolution is required.
- Full stable and prerelease versions are exact pins.
- Only strict concrete versions may reach managed path construction.

## Network and concurrency

Archives are downloaded sequentially. `download.parallel` and `download.max_connections` are reserved compatibility fields in v1.3.4.

HTTP retries cover transport failures and transient `408`, `429`, and `5xx` responses. Resume requires an exact byte range. Cache writes use unique partials, a portable lock file, and atomic commit so concurrent processes cannot merge content.

## Platform boundaries

- Unix symlink replacement uses same-directory rename.
- Windows uses `MoveFileEx` with replace-existing and write-through flags.
- Unix self-update performs in-process backup, replacement, validation, and rollback.
- Windows self-update starts a detached PowerShell helper because the running executable is locked.
- Shell generation supports Bash, Zsh, Fish, PowerShell, and Command Prompt. Directory-change auto-switch is not generated for Command Prompt.

## Error and output ownership

Production functions wrap and return errors. Cobra is configured to suppress duplicate automatic rendering; the CLI entry point renders each error once and includes usage only for argument/flag errors. Normal logs use stderr so stdout remains available for PATH commands. Quiet mode disables informational logs and progress.

## Security boundaries

- Version-derived paths must remain under `install_dir`.
- Config and shell profile destinations must be regular files, not symlinks.
- Archive paths, types, sizes, counts, and output modes are constrained.
- Installer and self-update binaries require release checksums and version validation.
- Custom API/download endpoints are explicit trusted configuration, not an implicit mirror feature.

See [Security](security.md) for limitations and trust assumptions.

## Testing and release

Unit tests sit beside implementation. Command-level tests use temp HOME/config/install roots and local HTTP servers. Helpers under `test/` are local-only, untracked, and are not run by CI.

CI gates include:

- Go 1.25 and 1.26 on Linux, macOS, and Windows;
- race detection;
- format, tidy, vet, pinned lint/security tools;
- total statement coverage of at least 80% and CLI coverage of at least 70%;
- Bash syntax checks on Linux and macOS, ShellCheck on Linux, and PSScriptAnalyzer on Windows;
- Docker amd64/arm64 build and native health check;
- GoReleaser snapshot with exact artifact names and checksums.

Release workflows build only from tags and do not treat marker branches as release inputs.
