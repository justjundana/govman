# Requirements

## Supported release artifacts

| Operating system | Architectures |
|---|---|
| Linux | amd64, arm64 |
| macOS | amd64, arm64 |
| Windows | amd64, arm64, 386 |

The build and release pipeline compiles these seven targets. Support is validated by CI runners for current Linux, macOS, and Windows images; no minimum kernel or historical operating-system version is promised.

## Runtime

The govman binary is self-contained and uses Go's standard libraries to download and extract Go archives. It does not invoke external `tar`, `gzip`, `git`, or `curl` for normal `govman install` operations.

The standalone installer needs:

- Linux/macOS: `curl` or `wget`, a POSIX environment, and a supported interactive shell for integration.
- Windows PowerShell installer: PowerShell 5.1 or newer.
- Windows Batch installer: Command Prompt and `curl.exe`.

PowerShell is also required to complete a Windows self-update after the running executable exits.

## Filesystem and permissions

The default layout needs write access to:

- `~/.govman` or `%USERPROFILE%\.govman`;
- the selected shell profile;
- the current project directory when using `--local`.

Administrator/root privileges are not required for the default layout. Custom paths may require additional permissions. Windows policy may restrict symbolic links, although the Command Prompt wrapper does not depend on directory-change auto-switching.

Allow enough free space for a cached archive, an extraction staging directory, and the final Go toolchain during installation. Exact size depends on the selected Go release and platform.

## Network

Default endpoints require HTTPS access to:

- `go.dev` for Go release metadata and archives;
- `api.github.com` and GitHub release assets for explicit self-update;
- `raw.githubusercontent.com` when running a standalone installer directly from the repository.

Standard `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment variables are supported. There is no supported offline mode in v1.3.4.

## Shell support

| Shell | Manual `use` wrapper | Directory-change auto-switch |
|---|---:|---:|
| Bash | Yes | Yes |
| Zsh | Yes | Yes |
| Fish | Yes | Yes |
| PowerShell | Yes | Yes |
| Command Prompt | Yes | No |

The project file is read from the current directory. Shell integration must be initialized and loaded for a child govman process to update its parent shell environment.

## Building from source

- Go 1.25 or newer;
- Git for source checkout and release metadata;
- Make for documented project targets;
- pinned analysis/release tools installed by `make dev-setup` when running the complete local gate.

Docker is optional and only required for container validation.
