# Upgrade & Uninstall

## Upgrading govman

govman includes a built-in self-update mechanism to keep itself up-to-date.

### Check for Updates

```bash
govman selfupdate --check
```

This shows:
- Current version
- Latest available version
- Release date
- Release notes

### Update to Latest Version

```bash
govman selfupdate
```

Features:
- Automatic platform detection and binary selection
- Safe backup and rollback on failure
- Integrity verification and secure downloads
- Detailed release notes and changelog display

### Update to Specific Version

```bash
# Include pre-release versions
govman selfupdate --prerelease

# Force re-installation
govman selfupdate --force
```

### Self-Update Process

1. Checks GitHub for the latest release
2. Downloads the appropriate binary for your platform
3. Creates a backup of the current binary
4. Replaces the old binary with the new one
5. Verifies the installation
6. Cleans up temporary files

If the update fails, the backup is automatically restored.

### Manual Upgrade

If `govman selfupdate` fails, you can manually upgrade:

1. Download the latest binary from [GitHub Releases](https://github.com/justjundana/govman/releases)
2. Replace the existing binary:
   ```bash
   # Linux/macOS
   mv govman ~/.govman/bin/govman
   chmod +x ~/.govman/bin/govman
   
   # Windows (PowerShell)
   Move-Item govman.exe "$env:USERPROFILE\.govman\bin\govman.exe" -Force
   ```

## Uninstalling govman

govman provides uninstall scripts for clean removal.

### Linux/macOS

```bash
curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/uninstall.sh | bash
```

Or download and run the script:

```bash
curl -O https://raw.githubusercontent.com/justjundana/govman/main/scripts/uninstall.sh
chmod +x uninstall.sh
./uninstall.sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/justjundana/govman/main/scripts/uninstall.ps1 | iex
```

### Windows (Command Prompt)

Download and run `uninstall.bat` from the repository.

### Uninstall Options

The uninstall script offers two modes:

#### 1. Minimal Removal (Recommended)

Removes:
- govman binary and executable
- PATH configuration
- Shell integration

Keeps:
- Downloaded Go versions (`~/.govman/versions`)
- Configuration file (`~/.govman/config.yaml`)
- Cache directory

This allows you to reinstall govman later without re-downloading Go versions.

#### 2. Complete Removal

Removes:
- govman binary and executable
- PATH configuration
- Shell integration
- **All downloaded Go versions**
- **Entire .govman directory**

⚠️ **Warning**: This permanently deletes all Go versions and cannot be undone.

### Manual Uninstall

If the uninstall script is unavailable, you can manually remove govman:

#### Linux/macOS

```bash
# 1. Remove govman directory
rm -rf ~/.govman

# 2. Remove shell configuration
# Edit your shell config file and remove the GOVMAN section
nano ~/.bashrc  # or ~/.zshrc, ~/.config/fish/config.fish

# Remove lines between:
# # GOVMAN - Go Version Manager
# ...
# # END GOVMAN
```

#### Windows

```powershell
# 1. Remove govman directory
Remove-Item -Recurse -Force "$env:USERPROFILE\.govman"

# 2. Remove from PATH
# Open System Properties > Environment Variables
# Edit user PATH and remove: %USERPROFILE%\.govman\bin

# 3. Remove from PowerShell profile (if configured)
notepad $PROFILE
# Remove the GOVMAN section
```

### Post-Uninstall Verification

```bash
# Verify govman is removed
which govman  # Should return nothing
govman --version  # Should show command not found

# Verify PATH is clean
echo $PATH | grep govman  # Should return nothing

# Check for remaining files
ls -la ~/.govman  # Linux/macOS
dir %USERPROFILE%\.govman  # Windows
```

### Reinstallation

After uninstalling, you can reinstall govman anytime using the installation script. If you used minimal removal, your Go versions will still be available after reinstallation.

## Troubleshooting

### Self-Update Fails

If `govman selfupdate` fails:

1. Check your internet connection
2. Verify access to `https://api.github.com`
3. Check permissions for the govman binary directory
4. Try manual upgrade method

### Development Version

If you're running a development build (`dev`), self-update is not available:

```bash
$ govman --version
dev-abc123

$ govman selfupdate
Warning: Development version detected - updates are not available
You're using a development build. Update manually from source.
```

To update a development build, rebuild from source:

```bash
cd /path/to/govman/source
git pull
make build
make install
```

### Uninstall Script Not Found

If the uninstall script fails to download:

1. Check your internet connection
2. Download manually from GitHub
3. Use the manual uninstall method above

### Permission Denied During Uninstall

```bash
# Linux/macOS: Ensure you have write access
ls -la ~/.govman

# If needed, remove with appropriate permissions
rm -rf ~/.govman

# Windows: Run PowerShell as Administrator if needed
```

### Shell Configuration Not Cleaned

After uninstall, if shell integration remains:

1. Open your shell configuration file
2. Manually remove the GOVMAN section:
   ```bash
   # Bash
   nano ~/.bashrc
   
   # Zsh
   nano ~/.zshrc
   
   # Fish
   nano ~/.config/fish/config.fish
   ```
3. Save and restart your terminal
