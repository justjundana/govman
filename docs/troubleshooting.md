# Troubleshooting Guide

Solutions to common issues when using **govman**.

## Table of Contents

- [Installation Issues](#installation-issues)
- [Version Switching Issues](#version-switching-issues)
- [Auto-Switch Issues](#auto-switch-issues)
- [Shell Integration Issues](#shell-integration-issues)
- [Download Issues](#download-issues)
- [Permission Issues](#permission-issues)
- [Platform-Specific Issues](#platform-specific-issues)
- [Configuration Issues](#configuration-issues)

---

## Installation Issues

### "command not found: govman"

**Cause**: govman binary not in PATH.

**Solutions**:

1. **Check if govman is installed:**
   ```bash
   ls -la ~/.govman/bin/govman
   ```

2. **Verify PATH contains govman:**
   ```bash
   echo $PATH | grep govman
   ```

3. **Add to PATH manually:**
   ```bash
   # Bash/Zsh
   echo 'export PATH="$HOME/.govman/bin:$PATH"' >> ~/.bashrc
   source ~/.bashrc
   
   # Fish
   fish_add_path ~/.govman/bin
   ```

4. **Re-run installation:**
   ```bash
   curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
   ```

### Installation Fails with "Permission Denied"

**Cause**: Insufficient permissions.

**Solutions**:

1. **Make binary executable:**
   ```bash
   chmod +x ~/.govman/bin/govman
   ```

2. **Check directory permissions:**
   ```bash
   ls -ld ~/.govman/bin
   # Should show: drwxr-xr-x
   ```

3. **Fix ownership:**
   ```bash
   sudo chown -R $(whoami) ~/.govman
   ```

### "Certificate verification failed"

**Cause**: Outdated CA certificates.

**Solutions**:

```bash
# macOS
brew install openssl

# Ubuntu/Debian
sudo apt-get update && sudo apt-get install ca-certificates

# CentOS/RHEL
sudo yum install ca-certificates
```

---

## Version Switching Issues

### "Go version X is not installed"

**Cause**: Trying to use a version that isn't installed.

**Solutions**:

1. **Check installed versions:**
   ```bash
   govman list
   ```

2. **Install the version:**
   ```bash
   govman install 1.21.5
   ```

3. **Then switch:**
   ```bash
   govman use 1.21.5
   ```

### "Cannot uninstall currently active version"

**Cause**: Attempting to remove the version you're currently using.

**Solutions**:

1. **Switch to different version first:**
   ```bash
   govman use 1.20.12
   ```

2. **Then uninstall:**
   ```bash
   govman uninstall 1.21.5
   ```

### Version Switch Doesn't Persist

**Cause**: Using session-only activation instead of default.

**Solutions**:

1. **Set as default:**
   ```bash
   govman use 1.21.5 --default
   ```

2. **Verify default is set:**
   ```bash
   cat ~/.govman/config.yaml | grep default_version
   ```

3. **Check in new terminal:**
   ```bash
   # Open new terminal
   go version
   ```

---

## Auto-Switch Issues

### Auto-Switch Not Working

**Cause**: Auto-switch disabled or shell integration not loaded.

**Diagnosis**:

```bash
# Check if auto-switch is enabled
cat ~/.govman/config.yaml | grep -A 2 auto_switch

# Check shell integration
type govman_auto_switch  # Should show it's a function
```

**Solutions**:

1. **Enable auto-switch in config:**
   ```yaml
   auto_switch:
     enabled: true
   ```

2. **Re-initialize shell:**
   ```bash
   govman init --force
   source ~/.bashrc  # Or ~/.zshrc, etc.
   ```

3. **Test manually:**
   ```bash
   govman refresh
   ```

### `.govman-version` File Ignored

**Cause**: Invalid file format or content.

**Solutions**:

1. **Check file content:**
   ```bash
   cat .govman-version
   # Should contain just: 1.21.5
   ```

2. **Fix format (no quotes, no prefix):**
   ```bash
   echo "1.21.5" > .govman-version  # ❌ Wrong (has quotes)
   echo 1.21.5 > .govman-version     # ✅ Correct
   ```

3. **Remove whitespace:**
   ```bash
   echo "1.21.5" | tr -d ' \n\r' > .govman-version
   ```

4. **Verify version is installed:**
   ```bash
   govman list | grep 1.21.5
   govman install 1.21.5  # If missing
   ```

### Wrong Version After `cd`

**Cause**: Multiple `.govman-version` files or cached state.

**Solutions**:

1. **Check for parent directory files:**
   ```bash
   find . -name ".govman-version" -type f
   ```

2. **Force refresh:**
   ```bash
   govman refresh
   ```

3. **Verify current:**
   ```bash
   govman current
   go version
   ```

---

## Shell Integration Issues

### Shell Integration Not Loading

**Cause**: Configuration not in shell RC file or syntax error.

**Diagnosis**:

```bash
# Check if configuration exists
grep -A 5 "GOVMAN" ~/.bashrc  # Or ~/.zshrc, etc.

# Check for syntax errors
bash -n ~/.bashrc  # For Bash
zsh -n ~/.zshrc    # For Zsh
```

**Solutions**:

1. **Re-initialize:**
   ```bash
   govman init --force
   ```

2. **Manually reload:**
   ```bash
   source ~/.bashrc  # Or appropriate RC file
   ```

3. **Check for conflicts:**
   ```bash
   # Look for duplicate GOVMAN sections
   grep -n "GOVMAN" ~/.bashrc
   ```

### `govman` Command is Not a Function

**Cause**: Wrapper function not loaded.

**Diagnosis**:

```bash
type govman
# Should show: govman is a function
# If shows: govman is /path/to/binary - wrapper not loaded
```

**Solutions**:

1. **Reload shell configuration:**
   ```bash
   source ~/.bashrc
   ```

2. **Check function definition:**
   ```bash
   declare -f govman  # Should show function code
   ```

3. **Re-initialize shell:**
   ```bash
   govman init --force
   ```

### Auto-Switch Triggers Too Often

**Cause**: Hook firing on every prompt.

**Solutions**:

1. **Disable auto-switch:**
   ```yaml
   auto_switch:
     enabled: false
   ```

2. **Use manual switching:**
   ```bash
   govman use 1.21.5
   ```

---

## Download Issues

### Download Fails or Times Out

**Cause**: Network issues, firewall, or slow connection.

**Solutions**:

1. **Increase timeout:**
   ```yaml
   download:
     timeout: 1800s  # 30 minutes
     retry_count: 10
   ```

2. **Use mirror:**
   ```yaml
   mirror:
     enabled: true
     url: "https://golang.google.cn/dl/"
   ```

3. **Check connectivity:**
   ```bash
   curl -I https://go.dev/dl/
   ```

4. **Configure proxy:**
   ```bash
   export HTTP_PROXY=http://proxy.example.com:8080
   export HTTPS_PROXY=http://proxy.example.com:8080
   ```

### "Checksum verification failed"

**Cause**: Corrupted download or man-in-the-middle attack.

**Solutions**:

1. **Clear cache and retry:**
   ```bash
   govman clean
   govman install 1.21.5
   ```

2. **Check network security:**
   - Verify you're not behind a corporate proxy that modifies downloads
   - Check for antivirus interference
   - Try different network

3. **Manual download:**
   - Download from https://go.dev/dl/
   - Verify checksum manually
   - Extract to `~/.govman/versions/`

### Downloads are Slow

**Cause**: Limited bandwidth or server issues.

**Solutions**:

1. **Use mirror closer to your region:**
   ```yaml
   mirror:
     enabled: true
     url: "https://golang.google.cn/dl/"  # For Asia
   ```

2. **Adjust connection settings:**
   ```yaml
   download:
     parallel: true
     max_connections: 2  # Lower for slower networks
   ```

3. **Resume interrupted downloads:**
   - govman automatically resumes partial downloads
   - Just run the install command again

---

## Permission Issues

### "Permission denied" When Creating Directories

**Cause**: Insufficient permissions in home directory.

**Solutions**:

1. **Fix home directory permissions:**
   ```bash
   chmod 755 ~
   ```

2. **Fix govman directory permissions:**
   ```bash
   mkdir -p ~/.govman
   chmod 755 ~/.govman
   ```

3. **Check ownership:**
   ```bash
   ls -ld ~/.govman
   # Should be owned by you, not root
   ```

4. **Fix ownership:**
   ```bash
   sudo chown -R $(whoami) ~/.govman
   ```

### "Operation not permitted" on macOS

**Cause**: macOS Gatekeeper blocking binary.

**Solutions**:

1. **Remove quarantine attribute:**
   ```bash
   xattr -d com.apple.quarantine ~/.govman/bin/govman
   ```

2. **Allow in System Preferences:**
   - Go to **System Preferences** → **Security & Privacy**
   - Click "Allow" for govman

3. **Disable Gatekeeper temporarily:**
   ```bash
   sudo spctl --master-disable
   # Run govman
   sudo spctl --master-enable
   ```

### SELinux Issues on Linux

**Cause**: SELinux preventing execution.

**Solutions**:

1. **Check SELinux status:**
   ```bash
   getenforce
   ```

2. **Set proper context:**
   ```bash
   chcon -t bin_t ~/.govman/bin/govman
   ```

3. **Or temporarily disable:**
   ```bash
   sudo setenforce 0
   # Run govman
   sudo setenforce 1
   ```

---

## Platform-Specific Issues

### macOS: "Developer Cannot be Verified"

**Cause**: Apple's security blocking unsigned binary.

**Solutions**:

1. **Right-click and open:**
   - Right-click govman binary
   - Select "Open"
   - Click "Open" in dialog

2. **Command line:**
   ```bash
   xattr -d com.apple.quarantine ~/.govman/bin/govman
   ```

### Windows: "Windows Defender Blocked This App"

**Cause**: SmartScreen blocking unknown binary.

**Solutions**:

1. **Add exclusion:**
   ```powershell
   Add-MpPreference -ExclusionPath "$env:USERPROFILE\.govman"
   ```

2. **Click "Run anyway":**
   - Click "More info"
   - Click "Run anyway"

3. **Disable SmartScreen temporarily:**
   - Only if you trust the source

### Windows: PowerShell Execution Policy

**Cause**: PowerShell blocking scripts.

**Solutions**:

```powershell
# Check current policy
Get-ExecutionPolicy

# Set policy for current user
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser

# Or bypass for single session
powershell -ExecutionPolicy Bypass
```

### Linux: "No such file or directory" Despite Binary Existing

**Cause**: Missing shared libraries or wrong architecture.

**Solutions**:

1. **Check architecture:**
   ```bash
   uname -m
   file ~/.govman/bin/govman
   ```

2. **Install missing libraries:**
   ```bash
   # Ubuntu/Debian
   sudo apt-get install libc6

   # CentOS/RHEL
   sudo yum install glibc
   ```

---

## Configuration Issues

### Configuration Changes Not Taking Effect

**Cause**: Config not reloaded or using flags.

**Solutions**:

1. **Check config location:**
   ```bash
   ls -la ~/.govman/config.yaml
   ```

2. **Validate YAML syntax:**
   ```bash
   # Use online YAML validator or
   govman --verbose list  # Shows config loading
   ```

3. **Remove and regenerate:**
   ```bash
   mv ~/.govman/config.yaml ~/.govman/config.yaml.backup
   govman list  # Regenerates with defaults
   ```

### Custom Config File Not Loaded

**Cause**: Not specifying config path.

**Solutions**:

```bash
# Use --config flag
govman --config /path/to/config.yaml list

# Or set environment variable
export GOVMAN_CONFIG=/path/to/config.yaml
govman list
```

---

## Debugging Tips

### Enable Verbose Mode

Get detailed information about what govman is doing:

```bash
govman --verbose install 1.21.5
govman --verbose use 1.21.5
```

### Check govman Version

```bash
govman --version
```

### Check Go Environment

```bash
go env
go version -v
```

### Test in Clean Environment

```bash
# Start new shell without config
bash --norc --noprofile
# Or
zsh -f

# Test govman commands
```

### Check Logs

govman outputs to stderr. Capture for analysis:

```bash
govman install 1.21.5 2>&1 | tee govman-install.log
```

---

## Getting Help

### Before Opening an Issue

1. ✅ Check this troubleshooting guide
2. ✅ Search [existing issues](https://github.com/justjundana/govman/issues)
3. ✅ Try with `--verbose` flag
4. ✅ Test in clean environment

### When Opening an Issue

Include:

- **OS and version**: `uname -a` (Unix) or `systeminfo` (Windows)
- **Shell**: `echo $SHELL`
- **govman version**: `govman --version`
- **Go version**: `go version` (if applicable)
- **Complete error message**: Use `--verbose`
- **Steps to reproduce**
- **Expected vs actual behavior**

### Community Support

- 📖 [Documentation](https://github.com/justjundana/govman)
- 💬 [GitHub Discussions](https://github.com/justjundana/govman/discussions)
- 🐛 [Report Issues](https://github.com/justjundana/govman/issues/new)

---

## Still Having Issues?

If this guide didn't solve your problem:

1. Check the [Commands Reference](commands.md) for detailed usage
2. Review [Configuration Guide](configuration.md) for settings
3. See [Shell Integration](shell-integration.md) for setup issues
4. Open a [GitHub Issue](https://github.com/justjundana/govman/issues/new) with details

---

We're here to help! 🤝
