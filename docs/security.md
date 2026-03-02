# Security

Security policies and best practices for govman.

## Reporting Security Issues

> [!IMPORTANT]
> **DO NOT** open public GitHub issues for security vulnerabilities.

Instead, report security issues privately to:
- **Email**: -
- **GitHub Security Advisories**: Use the "Security" tab on the official [GitHub repository](https://github.com/justjundana/govman/security/advisories).

Expected response time:
- Initial response: within 48 hours
- Status update: within 7 days
- Fix timeline: depends on severity

---

## Security Features

### Download Verification

All Go downloads are verified using SHA-256 checksums fetched from official sources.

1. Fetches official checksums from `go.dev/dl/` via HTTPS.
2. Computes the SHA-256 hash of the downloaded archive.
3. Compares the hash with the official checksum.
4. **Hard Fail**: If a mismatch is detected, govman wipes the file and rejects the installation.

### Path Validation

govman validates all user-provided configuration paths to prevent common attacks:

- **Directory Traversal**: Strictly rejects any paths containing `..` or illegal characters.
- **Absolute Path Resolution**: Resolves all paths relative to `$HOME` or `%USERPROFILE%`.
- **Permission Enforcement**: Verifies write access before attempting to create directories or move binaries.

### Binary Verification (Self-Update)

For `govman selfupdate`:
- **HTTPS Only**: All update metadata and binary downloads use TLS 1.2+.
- **Official Sources**: Downloads are restricted to official GitHub Release assets.
- **Rollback Mechanism**: govman creates a `.bak` of the current binary before replacement. If the new binary fails to run (e.g., checksum error or execution failure), it automatically restores the previous version.

### Zero Sudo Policy

govman is designed to run entirely in userspace. 
- **Linux/macOS**: No `sudo` required.
- **Windows**: No Administrator rights required.
- 100% of data is stored in `~/.govman` or a user-defined directory.

---

## Technical Security Details

### GOTOOLCHAIN Strategy

> [!TIP]
> To prevent unexpected toolchain updates that could introduce unverified binaries, govman sets `export GOTOOLCHAIN=local` in organized shell integration. 

This ensures that the Go compiler only uses the specific version you activated through govman, rather than attempting to download a new toolchain automatically via the built-in Go 1.21+ toolchain management.

### Supply Chain Security

We prioritize the integrity of the govman release process:
- **SLSA Compliance**: We follow SLSA Level 1 guidelines for build provenance.
- **Dependency Pinning**: All dependencies are locked to specific versions in `go.mod` and audited regularly.
- **Minimal Surface**: govman minimizes third-party dependency usage to reduce the risk of transitive vulnerability exploits.

---

## Threat Model

| Threat | govman Protection |
| :--- | :--- |
| **Malicious Archive** | Mandatory SHA-256 verification against official go.dev records. |
| **MITM Attack** | Certificate pinning and TLS mandatory for all API/Download requests. |
| **Path Traversal** | Sanitization of all file paths and expansion logic. |
| **Shell Injection** | Shell-specific escaping in `govman init` to prevent command execution. |
| **Configuration Hijack** | Path validation prevents overriding system files (e.g., `/etc/shadow`). |

---

## Usage in Production & CI/CD

### For Enterprise Teams

1. **Local Mirror**: In restricted networks, use the `mirror` configuration key to point to an internal Artifactory or Go proxy.
2. **Review Init Code**: Always review the output of `govman init` before applying it to production boxes.
3. **Dedicated User**: For servers, run govman under a service-specific user with limited filesystem scope.

### For Air-Gapped Environments

govman supports air-gapped workflows through its cache layer:
1. Download required Go versions to `~/.govman/cache` on a machine with internet access.
2. Transfer the entire `.govman/cache` folder to the target machine.
3. Run `govman install <version>`. govman will prioritize the local cache and proceed with uninstallation/activation without network requests.

---

## Permissions Checklist

| Path | Purpose | Recommended Mode |
| :--- | :--- | :--- |
| `~/.govman/bin` | Execute binaries | `755` (rwxr-xr-x) |
| `~/.govman/config.yaml` | Application settings | `600` (rw-------) |
| `~/.govman/versions` | Go SDKs | `755` (rwxr-xr-x) |
| `~/.govman/cache` | Temporaries | `700` (rwx------) |

---

## Privacy Policy

- **No Telemetry**: govman does not phone home with telemetry, usage stats, or error reports.
- **No Tracking**: We do not collect OS information or user IDs.
- **Minimal Networking**: Network calls only occur during `list --remote`, `install`, and `selfupdate`.

---

## Compliance & Auditing

govman is suitable for use in industries requiring strict auditing (FinTech, Healthcare, Gov):
- **Predictable Environment**: Documentation of all file modification sites.
- **Audit Logs**: Redirect `govman --verbose` to a log file for a complete record of SDK management actions.
- **Open Source**: The entire logic is open for security auditing by your internal teams.
