# Installation

Standalone installers download an exact platform asset plus `checksums.txt`, verify SHA-256, validate the binary, and only then replace an existing installation. Download or validation failure preserves the old binary.

## Linux and macOS

```bash
curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
```

Options can be passed to the downloaded script:

```bash
curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash -s -- --quiet
curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash -s -- --version v1.3.4
```

The script supports amd64 and arm64 on Linux/macOS and selects the matching raw release binary.

## Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.ps1 | iex
```

To pass options:

```powershell
$script = [scriptblock]::Create((irm https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.ps1))
& $script -Version 'v1.3.4'
```

## Windows Command Prompt

Download `scripts/install.bat` from the repository and run it from Command Prompt. The Batch installer supports Windows amd64, arm64, and 386 assets.

## Default layout

| Purpose | Unix-like systems | Windows |
|---|---|---|
| govman binary/wrapper | `~/.govman/bin` | `%USERPROFILE%\.govman\bin` |
| Go toolchains | `~/.govman/versions` | `%USERPROFILE%\.govman\versions` |
| Download cache | `~/.govman/cache` | `%USERPROFILE%\.govman\cache` |
| Configuration | `~/.govman/config.yaml` | `%USERPROFILE%\.govman\config.yaml` |

No root/administrator permission is required for this layout.

## Initialize and verify

The installer attempts shell initialization. It reports a partial failure and exits nonzero if required initialization fails; rerun it explicitly after fixing profile permissions:

```bash
govman init --force
```

Restart the terminal or source the profile path printed by the command, then:

```bash
govman --version
govman install latest
govman use latest --default
go version
gofmt -h
```

PowerShell users reload `$PROFILE`. Command Prompt users open a new prompt so user PATH and the generated wrapper are visible.

## Existing installations

Detecting an already installed valid binary is a successful no-op. Use `govman selfupdate` for an in-place verified update, or uninstall before intentionally reinstalling from scratch.

## Manual binary installation

If an installer cannot be used, download the exact asset and `checksums.txt` from the same GitHub release, verify its SHA-256 entry, run the downloaded binary with `--version`, and place it in the platform bin directory. Do not install a binary from an unmatched release or bypass a checksum failure.
