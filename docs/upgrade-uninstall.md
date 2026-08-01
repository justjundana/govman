# Upgrade and uninstall

## Self-update

```bash
govman selfupdate --check
govman selfupdate
```

`--prerelease` includes eligible prerelease metadata. `--force` reinstalls the same eligible release; it does not authorize an automatic downgrade.

The update workflow:

1. fetches bounded release metadata;
2. validates SemVer, prerelease policy, and exact platform asset names;
3. downloads `checksums.txt` and the binary;
4. checks size and SHA-256;
5. executes the temp binary with `--version`;
6. replaces the installed binary while retaining rollback state;
7. validates the installed result before deleting the backup.

On Windows, a detached PowerShell helper waits for govman to exit before replacing the locked executable. Legacy installs that used `govman.exe` directly are migrated to the wrapper/backend layout and restored if migration fails.

Development builds report that self-update is unavailable. Rebuild or reinstall them explicitly.

## Standalone upgrade

Running the current platform installer is also safe: it verifies release checksums and validation before replacement. An already valid installation is a successful no-op, so use self-update for normal upgrades or uninstall first when a clean reinstall is required.

## Unix uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/uninstall.sh | bash
```

The script offers minimal and complete removal. Before deleting the binary, it validates every candidate shell profile block. A malformed, duplicate, incomplete, or symlinked profile is not modified and blocks destructive progress. Valid changes preserve mode, create a timestamped backup, and use a same-directory atomic replacement.

Minimal removal deletes govman executable/wrapper, PATH integration, and exact shell blocks while retaining config, cache, and installed Go toolchains. Complete removal deletes the full `~/.govman` tree after confirmation.

## Windows uninstall

PowerShell:

```powershell
irm https://raw.githubusercontent.com/justjundana/govman/main/scripts/uninstall.ps1 | iex
```

Command Prompt users run `scripts/uninstall.bat` from the repository.

Windows uninstallers remove normalized exact user-PATH entries without substring replacement, preserve unrelated entries and registry value type, and clean the PowerShell profile marker block transactionally. Failures are reported rather than followed by a false success message.

## Post-uninstall checks

Open a new terminal, then verify that govman is absent and inspect whether retained state matches the selected mode:

```bash
command -v govman
test -d ~/.govman/versions && govman_versions_retained=yes
```

Do not manually use broad range deletion on shell profiles. Remove only one complete block from `# GOVMAN - Go Version Manager` through `# END GOVMAN`, and keep the generated backup until the profile has been loaded successfully.
