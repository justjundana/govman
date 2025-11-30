# Installation

govman can be installed on Linux, macOS, and Windows using automated installation scripts.

## Linux and macOS

The installation script automatically detects your platform and installs the appropriate binary.

### One-Line Installation

```bash
curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
```

### Installation with Options

```bash
# Quiet mode (minimal output)
curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash -s -- --quiet

# Install specific version
curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash -s -- --version v1.0.0

# Show help
curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash -s -- --help
```

### Manual Installation

1. Download the installation script:
   ```bash
   curl -O https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh
   chmod +x install.sh
   ```

2. Run the script:
   ```bash
   ./install.sh
   ```

## Windows

govman supports both PowerShell and Command Prompt on Windows.

### PowerShell (Recommended)

```powershell
# One-line installation
irm https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.ps1 | iex

# Install with options
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.ps1))) -Quiet
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.ps1))) -Version "v1.0.0"
```

### Command Prompt

Download the `install.bat` script from the repository and run it:

```cmd
install.bat
```

## Installation Directories

govman installs to the following directories:

- **Linux/macOS**: `~/.govman/`
  - Binary: `~/.govman/bin/govman`
  - Go versions: `~/.govman/versions/`
  - Cache: `~/.govman/cache/`
  - Config: `~/.govman/config.yaml`

- **Windows**: `%USERPROFILE%\.govman\`
  - Binary: `%USERPROFILE%\.govman\bin\govman.exe`
  - Go versions: `%USERPROFILE%\.govman\versions\`
  - Cache: `%USERPROFILE%\.govman\cache\`
  - Config: `%USERPROFILE%\.govman\config.yaml`

## Post-Installation Steps

### 1. Initialize Shell Integration

Run the following command to configure your shell:

```bash
govman init
```

This will automatically configure:
- Bash (`.bashrc`, `.bash_profile`)
- Zsh (`.zshrc`)
- Fish (`config.fish`)
- PowerShell (`$PROFILE`)

### 2. Reload Your Shell

```bash
# Bash
source ~/.bashrc

# Zsh
source ~/.zshrc

# Fish
source ~/.config/fish/config.fish
```

Or simply restart your terminal.

### 3. Verify Installation

```bash
govman --version
```

## Platform Support

govman supports the following platforms:

| OS      | Architecture | Supported |
|---------|--------------|-----------|
| Linux   | amd64        | ✅         |
| Linux   | arm64        | ✅         |
| macOS   | amd64        | ✅         |
| macOS   | arm64 (M1+)  | ✅         |
| Windows | amd64        | ✅         |
| Windows | arm64        | ✅         |

## Troubleshooting

### Permission Denied

If you encounter permission errors during installation:

```bash
# Make the script executable
chmod +x install.sh

# Run with appropriate permissions
./install.sh
```

### curl or wget Not Found

The installation script requires either `curl` or `wget`:

```bash
# Install curl (Debian/Ubuntu)
sudo apt-get install curl

# Install wget (Debian/Ubuntu)
sudo apt-get install wget
```

### Existing Installation Detected

If govman is already installed, the installer will notify you and exit. To reinstall:

1. Uninstall first using the uninstall script
2. Run the installer again

### Installation Fails on Windows

Windows older than Windows 10 may not support ANSI colors. The installer will still work but without colored output.

For PowerShell, ensure you have execution policy configured:

```powershell
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser
```
