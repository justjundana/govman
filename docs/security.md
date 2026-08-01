# Security

## Reporting a vulnerability

Do not open a public issue for an undisclosed vulnerability. Use the repository's [private security advisory form](https://github.com/justjundana/govman/security/advisories/new).

Include the affected version, platform, reproduction steps, impact, and any proposed mitigation. Response and remediation time depend on severity and maintainer availability; this project does not promise a fixed response SLA.

## Integrity controls

### Go toolchains

govman obtains archive filename, size, platform, and SHA-256 metadata from the configured Go releases API. The archive is streamed into a unique cache partial, its final size is checked, and its SHA-256 must match before extraction.

### govman installers and self-update

Release automation produces seven raw platform binaries and `checksums.txt`. Standalone installers and `govman selfupdate` select an exact asset name and require a unique matching SHA-256 entry. A downloaded binary is size-bounded and must report the requested version before replacement.

On Unix-like systems, replacement keeps a backup until post-install validation succeeds and rolls back on failure. Windows schedules a detached helper that waits for the running process to exit, replaces the locked executable, validates it, and restores the previous binary on failure.

Configured endpoints are trusted inputs. HTTPS is required for normal self-update assets; loopback HTTP is accepted only to support local testing. govman does not implement certificate pinning, signatures, or SLSA provenance verification.

## Filesystem controls

- Concrete version strings are parsed before a managed path is formed.
- Joined version paths are checked to remain under `install_dir`.
- Archive paths reject absolute, traversal, backslash traversal, volume/UNC, NUL, symlink, and hard-link entries.
- Extraction uses `os.Root`, bounded entry/file/total limits, sanitized modes, and a staging directory.
- Existing regular files are not replaced by symlink activation.
- Config and shell integration refuse symlink/non-regular destinations.
- Config, local-version, shell integration, install, and activation mutations use same-directory temp files and rollback where needed.

## Shell and PATH controls

Generated wrappers accept only one validated PATH command from a successful `use` or `refresh` invocation. Shell blocks use exact markers and transactional replacement. Unix uninstallation refuses malformed or duplicate marker blocks. Windows installers and uninstallers normalize PATH entries and avoid substring deletion or delayed-expansion corruption.

Shell integration sets `GOTOOLCHAIN=local` so an activated Go command does not automatically fetch another toolchain through Go's toolchain selection mechanism.

## Privilege and storage scope

The default install is user-scoped under `~/.govman` or `%USERPROFILE%\.govman`. Custom `install_dir`, `cache_dir`, `--config`, and shell profile locations may place data elsewhere; govman uses the invoking user's permissions and does not sandbox those explicit choices.

No administrator privilege is required for the default layout. Some Windows symlink operations or custom protected paths may be restricted by operating-system policy.

## Network and privacy

There is no telemetry. Network access occurs when release metadata or archives are needed, and during explicit self-update checks. Standard proxy environment variables are honored by Go's HTTP transport.

govman does not provide a supported offline or air-gapped install mode. A cache hit can avoid transferring an existing archive, but metadata resolution may still require the configured API and cache files must exactly match expected release metadata.

## Reserved configuration

`mirror.*`, `download.parallel`, `download.max_connections`, `shell.auto_detect`, and `shell.completion` remain in the schema only for compatibility in v1.3.4. They do not provide mirror routing, parallel transfer, detection policy, or completion generation. Use explicitly controlled `go_releases.*` endpoints when organizational policy requires an internal source.

## Recommended practice

- Keep govman and the operating system trust store updated.
- Review custom endpoint and proxy ownership before use.
- Protect `~/.govman/config.yaml` as user-only data.
- Keep project `.govman-goversion` files under source control.
- Run CI with exact full Go versions when reproducibility matters.
- Report checksum, path-containment, archive, or rollback failures rather than bypassing them.
