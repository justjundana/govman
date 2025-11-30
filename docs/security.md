# Security

Security policies and best practices for govman.

## Reporting Security Issues

**DO NOT** open public GitHub issues for security vulnerabilities.

Instead, report security issues privately to:
- Email: security@govman.example.com (if available)
- GitHub Security Advisories: Use the "Security" tab on GitHub

Expected response time:
- Initial response: within 48 hours
- Status update: within 7 days
- Fix timeline: depends on severity

## Security Features

### Download Verification

All Go downloads are verified using SHA-256 checksums:

1. Fetches official checksums from go.dev
2. Computes SHA-256 of downloaded file
3. Compares with official checksum
4. Rejects mismatched downloads

```bash
# Automatic verification on every install
govman install 1.25.1
# Output includes: "Checksum verified"
```

### Path Validation

govman validates all user  configuration paths:

- **Prevents directory traversal**: Rejects paths containing `..`
- **Validates absolute paths**: Ensures paths resolve correctly
- **Checks permissions**: Verifies write access before operations

### Binary Verification

For self-updates:

- Downloads from official GitHub releases only
- Uses HTTPS for all connections
- Verifies GitHub's SSL certificate
- Creates backup before replacing binary

### No Elevated Privileges

govman never requires or requests:
- `sudo` on Linux/macOS
- Administrator rights on Windows

All operations are user-space only.

## Secure Usage

### Shell Integration

govman adds code to shell configuration files:

**What is added:**
- PATH modifications
- Wrapper functions
- Auto-switch logic

**Security considerations:**
- Code is clearly marked with `# GOVMAN` delimiters
- Can be reviewed before sourcing
- Removed completely during uninstall

**Review integration code:**
```bash
grep -A 50 "GOVMAN" ~/.bashrc
```

### Configuration File

Location: `~/.govman/config.yaml`

**Permissions:**
- Owned by your user account
- Not world-readable
- Contains no sensitive data
- Plain text YAML format

**Secure defaults:**
```yaml
# Official sources only
go_releases:
  api_url: https://go.dev/dl/?mode=json&include=all
  download_url: https://go.dev/dl/%s
```

### Network Security

**Connections made by govman:**

| Destination               | Purpose                    | Frequency        |
|---------------------------|----------------------------|------------------|
| `go.dev`                  | Fetch Go release info      | Per install/list |
| `golang.org`              | Download Go archives       | Per install      |
| `api.github.com`          | Self-update checks         | On selfupdate    |
| `github.com`              | Download govman updates    | On selfupdate    |

**Security measures:**
- All connections use HTTPS/TLS
- Certificate validation enabled
- No telemetry or tracking
- No third-party analytics

### Proxy Support

govman respects standard proxy settings:

```bash
export HTTPS_PROXY=https://proxy.example.com:8080
export HTTP_PROXY=http://proxy.example.com:8080
```

**Corporate environments:**
- Works with MITM SSL proxies
- Trusts system certificate store
- No proxy credentials stored

## Threat Model

### What govman protects against:

- ✅ **Corrupted downloads**: SHA-256 verification
- ✅ **MITM attacks**: HTTPS with certificate validation
- ✅ **Directory traversal**: Path validation
- ✅ **Unauthorized writes**: Userspace only, permission checks
- ✅ **Binary tampering**: Backup and rollback

### What govman does NOT protect against:

- ❌ **Compromised official sources**: Trusts go.dev and github.com
- ❌ **Local system compromise**: If attacker has user access
- ❌ **Supply chain attacks**: Trusts official Go binaries
- ❌ **Network-level attacks**: Relies on OS/system security

## Best Practices

### For Users

1. **Verify installation script:**
   ```bash
   # Download and review before running
   curl -O https://install.script
   less install.sh
   bash install.sh
   ```

2. **Use official sources:**
   - Install govman from official GitHub repository
   - Don't modify mirror URLs unless necessary

3. **Keep govman updated:**
   ```bash
   govman selfupdate
   ```

4. **Review shell integration:**
   ```bash
   govman init
   grep -A 50 "GOVMAN" ~/.bashrc  # Review before sourcing
   ```

5. **Check installed Go versions:**
   ```bash
   govman list
   govman info 1.25.1
   ```

### For Developers/Maintainers

1. **Sign releases**: Use GPG-signed commits and tags
2. **Pin dependencies**: Use `go.mod` with specific versions
3. **Run security scanners**: Regular vulnerability scans
4. **Audit dependencies**: Review third-party packages
5. **Minimal dependencies**: Reduce attack surface

## Dependency Security

govman has minimal external dependencies:

```
github.com/spf13/cobra    # CLI framework
github.com/spf13/viper    # Configuration
```

**Security measures:**
- Dependencies are vendored (optional)
- Specific versions pinned in go.mod
- Regular updates for security patches

### Checking Vulnerabilities

```bash
# Scan for known vulnerabilities
go list -json -m all | go run golang.org/x/vuln/cmd/govulncheck@latest
```

## Incident Response

If a security incident occurs:

1. **Notification**: Users notified via:
   - GitHub Security Advisories
   - Release notes
   - govman tool itself (if applicable)

2. **Patch release**: Security fixes in patch release (e.g., 1.0.1)

3. **Upgrade guidance**: Clear instructions for mitigation

4. **Disclosure timeline**:
   - Private disclosure: Security team notified
   - Fix developed and tested
   - Coordinated public disclosure with patch release

## Secure Defaults

govman ships with secure defaults:

```yaml
# No unencrypted connections
go_releases:
  api_url: https://go.dev/dl/?mode=json&include=all  # HTTPS
  download_url: https://go.dev/dl/%s                  # HTTPS

# Official sources only
mirror:
  enabled: false
  url: https://golang.google.cn/dl/  # HTTPS (if enabled)

# Sensible download limits
download:
  timeout: 300s       # Prevents indefinite hangs
  retry_count: 3      # Limits retry attempts
  retry_delay: 5s     # Rate limiting
```

## Permissions

### File System Permissions

```bash
# govman binary
~/.govman/bin/govman          # 755 (rwxr-xr-x)

# Configuration
~/.govman/config.yaml         # 644 (rw-r--r--)

# Installed Go versions
~/.govman/versions/*/         # 755 (rwxr-xr-x)

# Cache
~/.govman/cache/              # 755 (rwxr-xr-x)
```

### Required Permissions

- Read/write to `~/.govman/`
- Read/write to shell config files (`~/.bashrc`, etc.)
- Network access to HTTPS endpoints

### Unnecessary Permissions

- ❌ Root/sudo
- ❌ System directory access
- ❌ Other users' files
- ❌ Kernel modules
- ❌ Network configuration

## Code Security

### Static Analysis

govman code is analyzed using:
- `go vet`: Go's official code analyzer
- `golangci-lint`: Comprehensive linter suite
- `gosec`: Security-focused static analyzer

### Code Review

All changes require:
- Code review approval
- Automated tests passing
- Security implications considered

### Testing

Security-relevant tests:
- Path traversal prevention
- Input validation
- Configuration parsing
- Download verification
- Shell injection prevention

## Privacy

govman respects user privacy:

- **No telemetry**: No usage tracking
- **No analytics**: No user behavior data collected
- **No advertising IDs**: No device fingerprinting
- **Local-first**: All data stored locally

**Network requests only for:**
- Fetching Go release information
- Downloading Go binaries
- Self-update checks (explicit user action)

## Compliance

govman is designed to work in:
- Corporate environments with security policies
- Air-gapped networks (with pre-downloaded archives)
- Restricted regions (with mirrors)
- Compliance-focused organizations

## Security Checklist for Users

Before using govman in production:

- [ ] Downloaded from official source
- [ ] Reviewed installation script
- [ ] Configured appropriate mirrors/proxies (if needed)
- [ ] Reviewed shell integration code
- [ ] Verified checksum verification is working
- [ ] Tested in non-production environment first
- [ ] Documented version management policy
- [ ] Trained team on secure usage

## Future Security Enhancements

Planned security improvements:
- GPG signature verification for releases
- Support for private Go module proxies
- Enhanced audit logging
- Integration with security scanning tools
