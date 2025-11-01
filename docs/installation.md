# Installation Guide

Complete installation instructions for **govman** on all supported platforms.

## System Requirements

- **Operating Systems**: Windows 10+, macOS 10.13+, Linux (any modern distribution)
- **Architectures**: AMD64 (x86_64), ARM64
- **Disk Space**: ~50MB for govman + space for Go versions (~100-150MB per version)
- **Internet Connection**: Required for initial installation and downloading Go versions

## Installation Methods

### Quick Installation

#### macOS / Linux / WSL

```bash
curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
```

#### Windows PowerShell

Open PowerShell as Administrator and run:

```powershell
Set-ExecutionPolicy Bypass -Scope Process -Force
iex (iwr -useb https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.ps1)
```

#### Windows Command Prompt

```cmd
curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.bat -o install.bat && install.bat
```

### Manual Installation

#### 1. Download Binary

Visit the [Releases Page](https://github.com/justjundana/govman/releases/latest) and download the appropriate binary:

**macOS:**
```bash
# Intel Macs
curl -LO https://github.com/justjundana/govman/releases/latest/download/govman-darwin-amd64
mv govman-darwin-amd64 govman
chmod +x govman

# Apple Silicon (M1/M2/M3)
curl -LO https://github.com/justjundana/govman/releases/latest/download/govman-darwin-arm64
mv govman-darwin-arm64 govman
chmod +x govman
```

**Linux:**
```bash
# AMD64
curl -LO https://github.com/justjundana/govman/releases/latest/download/govman-linux-amd64
mv govman-linux-amd64 govman
chmod +x govman

# ARM64
curl -LO https://github.com/justjundana/govman/releases/latest/download/govman-linux-arm64
mv govman-linux-arm64 govman
chmod +x govman
```

**Windows:**
```powershell
# AMD64
Invoke-WebRequest -Uri "https://github.com/justjundana/govman/releases/latest/download/govman-windows-amd64.exe" -OutFile "govman.exe"

# ARM64
Invoke-WebRequest -Uri "https://github.com/justjundana/govman/releases/latest/download/govman-windows-arm64.exe" -OutFile "govman.exe"
```

#### 2. Move to Installation Directory

**macOS / Linux:**
```bash
mkdir -p ~/.govman/bin
mv govman ~/.govman/bin/
```

**Windows (PowerShell):**
```powershell
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\.govman\bin"
Move-Item -Force govman.exe "$env:USERPROFILE\.govman\bin\"
```

#### 3. Add to PATH

**macOS / Linux (Bash):**
```bash
echo 'export PATH="$HOME/.govman/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

**macOS / Linux (Zsh):**
```bash
echo 'export PATH="$HOME/.govman/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

**macOS / Linux (Fish):**
```fish
fish_add_path ~/.govman/bin
```

**Windows:**
```powershell
$path = [Environment]::GetEnvironmentVariable("PATH", "User")
[Environment]::SetEnvironmentVariable("PATH", "$path;$env:USERPROFILE\.govman\bin", "User")
```

Then restart your terminal.

### Build from Source

#### Prerequisites
- Go 1.25 or later
- Git
- Make (optional)

#### Steps

```bash
# Clone repository
git clone https://github.com/justjundana/govman.git
cd govman

# Build
go build -o govman ./cmd/govman

# Or use Make
make build

# Install to ~/.govman/bin
mkdir -p ~/.govman/bin
cp govman ~/.govman/bin/

# Add to PATH (if not already added)
echo 'export PATH="$HOME/.govman/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

## Platform-Specific Instructions

### macOS

#### Using Homebrew (if available in tap)

```bash
brew install govman
```

#### Gatekeeper Issues

If macOS blocks the binary:

```bash
xattr -d com.apple.quarantine ~/.govman/bin/govman
```

Or go to **System Preferences → Security & Privacy** and allow the app.

### Linux

#### Permissions

Make sure the binary is executable:

```bash
chmod +x ~/.govman/bin/govman
```

#### Shell Configuration

For system-wide installation (requires sudo):

```bash
sudo mv govman /usr/local/bin/
```

#### SELinux

If using SELinux, you may need to adjust contexts:

```bash
chcon -t bin_t ~/.govman/bin/govman
```

### Windows

#### PATH Configuration

To add govman to PATH permanently:

1. Open **Start Menu** → Search "Environment Variables"
2. Click **Environment Variables**
3. Under **User variables**, select **Path** → **Edit**
4. Click **New** → Add `%USERPROFILE%\.govman\bin`
5. Click **OK** to save

Or use PowerShell (as Administrator):

```powershell
[Environment]::SetEnvironmentVariable(
    "PATH",
    [Environment]::GetEnvironmentVariable("PATH", "User") + ";$env:USERPROFILE\.govman\bin",
    "User"
)
```

#### Windows Defender

If Windows Defender blocks the binary, add an exclusion:

```powershell
Add-MpPreference -ExclusionPath "$env:USERPROFILE\.govman\bin\govman.exe"
```

#### Execution Policy

If scripts are blocked:

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### WSL (Windows Subsystem for Linux)

Use the Linux installation method:

```bash
curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
```

## Post-Installation

### Verify Installation

```bash
govman --version
```

### Initialize Shell Integration

This enables automatic version switching:

```bash
govman init
```

Then restart your terminal or reload your shell configuration.

### Install Your First Go Version

```bash
govman install latest
govman use latest --default
```

### Verify Go Installation

```bash
go version
```

## Installation Locations

govman uses the following directory structure:

```
~/.govman/
├── bin/              # govman binary
├── versions/         # Installed Go versions
│   ├── go1.21.5/
│   ├── go1.20.12/
│   └── ...
├── cache/           # Downloaded archives
└── config.yaml      # Configuration file
```

## Network Configuration

### Using a Proxy

Set environment variables before installation:

```bash
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080
```

### Using a Mirror

Edit `~/.govman/config.yaml` after installation:

```yaml
mirror:
  enabled: true
  url: "https://golang.google.cn/dl/"  # China mirror
```

### Firewall Requirements

govman needs access to:
- `https://github.com` - For govman updates
- `https://go.dev` - For Go version metadata
- `https://dl.google.com` - For downloading Go distributions

## Upgrading govman

### Automatic Update

```bash
govman selfupdate
```

### Manual Update

Download the latest binary and replace the existing one:

```bash
# Backup current version
cp ~/.govman/bin/govman ~/.govman/bin/govman.backup

# Download new version (see Manual Installation above)
# Replace the binary
mv govman ~/.govman/bin/govman
```

## Uninstalling govman

### Complete Removal

**macOS / Linux:**

```bash
# Run uninstall script
curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/uninstall.sh | bash

# Or manually
rm -rf ~/.govman
# Remove from shell config manually (search for GOVMAN sections)
```

**Windows (PowerShell):**

```powershell
# Run uninstall script
iex (iwr -useb https://raw.githubusercontent.com/justjundana/govman/main/scripts/uninstall.ps1)

# Or manually
Remove-Item -Recurse -Force "$env:USERPROFILE\.govman"
# Remove from PATH via Environment Variables
```

### Keep Configuration and Go Versions

Remove only the govman binary:

```bash
rm ~/.govman/bin/govman
```

## Troubleshooting Installation

### "command not found: govman"

**Solution:** Make sure `~/.govman/bin` is in your PATH:

```bash
echo $PATH | grep govman
```

If not present, add it to your shell configuration and reload.

### Permission Denied

**Solution:** Make the binary executable:

```bash
chmod +x ~/.govman/bin/govman
```

### "Certificate verification failed"

**Solution:** Update CA certificates:

```bash
# Ubuntu/Debian
sudo apt-get update && sudo apt-get install ca-certificates

# macOS
brew install openssl
```

### Installation Fails on Windows

**Solutions:**
1. Run PowerShell as Administrator
2. Check antivirus/Windows Defender settings
3. Try manual installation method
4. Verify download integrity

### Binary Doesn't Run on macOS

**Solution:** Remove quarantine attribute:

```bash
xattr -d com.apple.quarantine ~/.govman/bin/govman
```

## Alternative Installation Methods

### Docker

Run govman in a Docker container:

```dockerfile
FROM golang:1.21
RUN curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
ENV PATH="/root/.govman/bin:${PATH}"
```

### CI/CD Integration

#### GitHub Actions

```yaml
- name: Install govman
  run: |
    curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
    echo "$HOME/.govman/bin" >> $GITHUB_PATH
```

#### GitLab CI

```yaml
before_script:
  - curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
  - export PATH="$HOME/.govman/bin:$PATH"
```

## Getting Help

If you encounter issues during installation:

1. Check the [Troubleshooting Guide](troubleshooting.md)
2. Search [GitHub Issues](https://github.com/justjundana/govman/issues)
3. Open a [new issue](https://github.com/justjundana/govman/issues/new) with:
   - Your OS and version
   - Installation method used
   - Complete error message
   - Output of `uname -a` (Unix) or `systeminfo` (Windows)

## Next Steps

After successful installation:

- 📖 Read the [Quick Start Guide](quick-start.md)
- ⚙️ Configure govman in [Configuration Guide](configuration.md)
- 🐚 Set up [Shell Integration](shell-integration.md)
- 📚 Explore [Commands Reference](commands.md)

---

Welcome to govman! 🎉
