# Troubleshooting

Common issues and their solutions.

## Installation Issues

### Permission Denied During Installation

**Symptoms:**
```
Error: Permission denied: cannot write to /usr/local/bin
Error: failed to install: permission denied
```

**Solution:**

govman does NOT require sudo. It installs to your home directory:

```bash
# Correct installation (no sudo):
curl -sSL https://install.script | bash

# NOT this:
# sudo curl -sSL https://install.script | bash
```

If you get permission errors in `~/.govman`:
```bash
chmod -R u+w ~/.govman
```

### curl or wget Not Found

**Symptoms:**
```
bash: curl: command not found
bash: wget: command not found
```

**Solution:**

Install curl or wget:

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install curl

# RHEL/CentOS/Fedora
sudo yum install curl

# macOS (if somehow missing)
brew install curl
```

### Installation Script Fails to Download

**Symptoms:**
```
Failed to download installation script
Connection refused
```

**Solution:**

1. Check internet connection
2. Verify firewall allows HTTPS to github.com
3. Try alternative download method:
   ```bash
   wget https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh
   bash install.sh
   ```

## Shell Integration Issues

### govman use Doesn't Update PATH in Current Session

**Symptoms:**
```bash
govman use 1.25.1
go version  # Still shows old version
```

**Cause:** Shell wrapper function not loaded.

**Solution:**

1. Initialize shell integration:
   ```bash
   govman init
   ```

2. Reload your shell:
   ```bash
   source ~/.bashrc  # Bash
   source ~/.zshrc   # Zsh
   source ~/.config/fish/config.fish  # Fish
   . $PROFILE  # PowerShell
   ```

3. Verify wrapper exists:
   ```bash
   type govman  # Should show it's a function
   ```

### Auto-Switching Not Working

**Symptoms:**
```bash
cd project-with-govman-version
go version  # Doesn't switch automatically
```

**Diagnosis:**

1. Check if auto-switch function exists:
   ```bash
   type govman_auto_switch
   ```

2. Check config:
   ```bash
   cat ~/.govman/config.yaml | grep -A 3 auto_switch
   ```

3. Verify `.govman-goversion` format:
   ```bash
   cat .govman-goversion  # Should contain only version number, e.g., "1.25.1"
   ```

**Solution:**

1. Ensure shell integration is initialized:
   ```bash
   govman init --force
   source ~/.bashrc  # or appropriate config
   ```

2. Enable auto-switch in config:
   ```yaml
   auto_switch:
     enabled: true
   ```

3. Test manually:
   ```bash
   govman_auto_switch
   ```

### Command Not Found After Installation

**Symptoms:**
```bash
govman: command not found
```

**Solution:**

1. Check if binary exists:
   ```bash
   ls -la ~/.govman/bin/govman
   ```

2. Add to PATH manually:
   ```bash
   export PATH="$HOME/.govman/bin:$PATH"
   ```

3. Make permanent (add to shell config):
   ```bash
   echo 'export PATH="$HOME/.govman/bin:$PATH"' >> ~/.bashrc
   source ~/.bashrc
   ```

4. Or reinitialize:
   ```bash
   govman init
   ```

## Version Management Issues

### "No Go version is currently active"

**Symptoms:**
```bash
govman current
# Error: no Go version is currently active
```

**Solution:**

```bash
# List installed versions
govman list

# If no versions installed:
govman install latest

# Activate a version
govman use latest --default
```

### "Go version X is not installed"

**Symptoms:**
```bash
govman use 1.25.1
# Error: Go version 1.25.1 is not installed
```

**Solution:**

```bash
# Install the version first
govman install 1.25.1

# Then use it
govman use 1.25.1
```

### Cannot Uninstall Currently Active Version

**Symptoms:**
```bash
govman uninstall 1.25.1
# Error: cannot uninstall currently active version 1.25.1
```

**Solution:**

Switch to a different version first:

```bash
# Switch to another installed version
govman use 1.24.0 --default

# Or install and switch to a new version
govman install latest
govman use latest --default

# Now uninstall
govman uninstall 1.25.1
```

## Download Issues

### Download Fails or Times Out

**Symptoms:**
```
Error: failed to download: context deadline exceeded
Error: download failed with status 503
```

**Solution:**

1. Check internet connection
2. Retry (govman auto-retries):
   ```bash
   govman install 1.25.1
   ```

3. Increase timeout:
   ```yaml
   # ~/.govman/config.yaml
   download:
     Timeout: 600s  # 10 minutes
     retry_count: 5
   ```

4. Use a mirror if in restricted region:
   ```yaml
   mirror:
     enabled: true
     url: https://golang.google.cn/dl/
   ```

### Checksum Verification Failed

**Symptoms:**
```
Error: checksum verification failed
Error: checksum mismatch: expected abc123, got def456
```

**Cause:** Corrupted download.

**Solution:**

```bash
# Clean cache and retry
govman clean
govman install 1.25.1
```

### Slow Downloads

**Solution:**

1. Enable parallel downloads:
   ```yaml
   download:
     parallel: true
     max_connections: 4
   ```

2. Use a geographically closer mirror:
   ```yaml
   mirror:
     enabled: true
     url: https://golang.google.cn/dl/  # For users in China
   ```

3. Check network congestion
4. Try at a different time

## PATH and Environment Issues

### Multiple Go Installations in PATH

**Symptoms:**
```bash
which go
# /usr/local/go/bin/go  (not govman's)

go version
# Not the expected version
```

**Solution:**

Ensure `~/.govman/bin` appears first in PATH:

```bash
# Check PATH order
echo $PATH | tr ':' '\n'

# Prepend govman to PATH in shell config
export PATH="$HOME/.govman/bin:$PATH"

# Not this:
# export PATH="$PATH:$HOME/.govman/bin"  # Wrong! Goes at end
```

### GOPATH/GOROOT Conflicts

**Symptoms:**
```
Warning: GOROOT environment variable is set
go: cannot find GOROOT directory
```

**Solution:**

1. Unset GOROOT (govman manages it):
   ```bash
   unset GOROOT
   ```

2. Remove from shell config if manually set:
   ```bash
   # Remove these lines from ~/.bashrc
   # export GOROOT=/usr/local/go
   # export GOPATH=$HOME/go
   ```

3. Let Go manage GOPATH automatically

### go env Shows Wrong GO ROOT

**Symptoms:**
```bash
go env GOROOT
# /usr/local/go  (not govman's Go)
```

**Solution:**

```bash
# Check which go binary is being used
which go

# Should be: ~/.govman/versions/goX.X.X/bin/go

# If not, fix PATH order
export PATH="$HOME/.govman/bin:$PATH"
```

## Windows-Specific Issues

### "Set-ExecutionPolicy" Error (PowerShell)

**Symptoms:**
```
cannot be loaded because running scripts is disabled
```

**Solution:**

```powershell
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### govman Not Found After Installation (Windows)

**Symptoms:**
```cmd
'govman' is not recognized as an internal or external command
```

**Solution:**

1. Add to PATH manually:
   - Open "Environment Variables"
   - Edit user PATH
   - Add: `%USERPROFILE%\.govman\bin`

2. Restart Command Prompt/PowerShell

3. Or use installer's automatic PATH setup:
   ```powershell
   govman init
   ```

### Auto-Switch Not Working in Command Prompt

**Expected:** Command Prompt (cmd.exe) does not support auto-switching.

**Solution:**

Use PowerShell instead, or manually run:
```cmd
govman use <version>
```

## macOS-Specific Issues

### "govman" Cannot Be Opened (macOS Security)

**Symptoms:**
```
"govman" cannot be opened because it is from an unidentified developer
```

**Solution:**

```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine ~/.govman/bin/govman

# Or allow in System Preferences
# System Preferences → Security & Privacy → General → Allow
```

###Rosetta Issues on Apple Silicon

**Symptoms:**
```
Bad CPU type in executable
```

**Solution:**

govman automatically handles this. If issues persist:

```bash
# Reinstall govman
curl -sSL https://install.script | bash

# Verify architecture
file ~/.govman/bin/govman
# Should show: Mach-O 64-bit executable arm64
```

## CI/CD Issues

### govman Not Found in CI

**Cause:** PATH not set or govman not installed.

**Solution:**

```yaml
#GitHub Actions example
- name: Install govman
  run: |
    curl -sSL https://install.script | bash
    echo "$HOME/.govman/bin" >> $GITHUB_PATH

- name: Verify
  run: govman --version
```

### Permission Errors in Docker

**Solution:**

Run as non-root user:

```dockerfile
# Create non-root user
RUN useradd -m govman
USER govman

# Install to user directory
RUN curl -sSL https://install.script | bash
```

## Network and Proxy Issues

### Corporate Proxy Blocks Downloads

**Solution:**

Set proxy environment variables:

```bash
export HTTP_PROXY=http://proxy.corp.com:8080
export HTTPS_PROXY=http://proxy.corp.com:8080
export NO_PROXY=localhost,127.0.0.1,.corp.local

govman install latest
```

### SSL Certificate Errors

**Symptoms:**
```
Error: x509: certificate signed by unknown authority
```

**Solution:**

1. Update CA certificates:
   ```bash
   # Ubuntu
   sudo apt-get update
   sudo apt-get install ca-certificates

   # macOS
   # Usually not needed; check system time is correct
   ```

2. If behind corporate proxy with SSL inspection, contact IT

## Debugging Commands

### Verbose Mode

```bash
govman --verbose install 1.25.1
govman --verbose list --remote
```

### Check Configuration

```bash
cat ~/.govman/config.yaml
```

### Verify Shell Integration

```bash
# Check if functions are loaded
type govman
type govman_auto_switch

# Check shell config
grep -A 50 "GOVMAN" ~/.bashrc
```

### Inspect Cache

```bash
ls -lah ~/.govman/cache/
```

### Check Symlinks

```bash
ls -la ~/.govman/bin/go
```

## Getting Help

### Collect Debug Information

When reporting issues, include:

```bash
# govman version
govman --version

# OS and version
uname -a  # Linux/macOS
ver       # Windows

# Shell type and version
echo $SHELL
$SHELL --version

# Go version
go version

# PATH
echo $PATH

# Config
cat ~/.govman/config.yaml

# Installed versions
govman list

# Permissions
ls -la ~/.govman/
```

### Common Log Locations

```bash
# Installation logs (if saved)
~/.govman/install.log

# Shell config
~/.bashrc, ~/.zshrc, ~/.config/fish/config.fish, $PROFILE
```

### Reinstall govman

If all else fails:

```bash
# Uninstall (keeps Go versions)
curl -sSL https://uninstall.script | bash  # Choose minimal removal

# Reinstall
curl -sSL https://install.script | bash

# Reinitialize
govman init
source ~/.bashrc
```

### Reset to Defaults

```bash
# Backup current config
cp ~/.govman/config.yaml ~/.govman/config.yaml.backup

# Remove config (will recreate with defaults)
rm ~/.govman/config.yaml

# Next govman command recreates it
govman --version
```
