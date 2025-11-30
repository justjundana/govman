# Versioning Policy

govman's versioning strategy and compatibility guarantees.

## Semantic Versioning

govman follows [Semantic Versioning 2.0.0](https://semver.org/):

```
MAJOR.MINOR.PATCH
```

- **MAJOR**: Incompatible API changes
- **MINOR**: New functionality (backwards compatible)
- **PATCH**: Bug fixes (backwards compatible)

### Examples

- `1.0.0` → `1.0.1`: Bug fix
- `1.0.0` → `1.1.0`: New feature
- `1.0.0` → `2.0.0`: Breaking change

## Version Components

### govman Version

The version of the govman tool itself:

```bash
govman --version
# Output: 1.0.0
```

### Go Version

The version of Go being managed:

```bash
go version
# Output: go version go1.25.1 linux/amd64
```

## Compatibility Guarantees

### Configuration File

**Minor versions** (1.x → 1.y):
- New fields may be added
- Existing fields remain compatible
- Defaults provided for new fields
- Old configs continue to work

**Major versions** (1.x → 2.x):
- Configuration format may change
- Migration guide provided
- Tool assists with migration

### CLI Commands

**Minor versions**:
- New commands may be added
- New flags may be added to existing commands
- Existing commands remain unchanged
- Deprecated commands show warnings

**Major versions**:
- Commands may be removed (after deprecation)
- Command behavior may change
- Flags may be removed or changed

### Shell Integration

**Minor versions**:
- Integration code remains compatible
- New features may require reinitialization
- `govman init --force` updates to latest

**Major versions**:
- Integration format may change
- `govman init` migrates automatically

## Deprecation Process

1. **Announcement**: Feature marked as deprecated in release notes
2. **Warning**: Deprecated feature shows warning when used
3. **Grace Period**: Minimum one minor version (e.g., 1.1 → 1.2)
4. **Removal**: Removed in next major version

### Example

```
v1.1.0: Feature X announced as deprecated
v1.2.0: Feature X shows warnings
v1.3.0: Feature X still available with warnings
v2.0.0: Feature X removed
```

## Release Cycle

### Stable Releases

- **Patch releases**: As needed for bug fixes
- **Minor releases**: Every 2-3 months
- **Major releases**: Annually or as needed

### Development Builds

- Built from `main` branch
- Tagged as `dev-<commit>`
- Not for production use
- May contain breaking changes

## Version Support

### Current Version

- Full support: Bug fixes, security patches, new features
- Recommended for all users

### Previous Minor Version

- Bug fixes and security patches
- No new features
- Example: 1.2.x when 1.3.x is current

### Older Versions

- Critical security patches only
- Users encouraged to upgrade

### End of Life

Versions reach end of life (EOL) when:
- Two major versions behind (e.g., 1.x when 3.x is current)
- Critical security issues arise that require breaking changes
- Announced 6 months in advance

## Go Version Compatibility

govman supports all official Go releases from golang.org:

- **Stable releases**: Fully supported
- **Pre-releases** (beta, rc): Supported (install with explicit version)
- **Archived releases**: Supported if available from go.dev

### Platform-Specific Go Versions

- **Apple Silicon**: Go 1.16+ native arm64, earlier versions use amd64 (Rosetta)
- **Windows ARM**: Go 1.18+ native arm64
- **Linux ARM**: All official arm64 releases

## Build Version Information

govman embeds build metadata:

```bash
govman --version
# Output includes:
# Version: 1.0.0
# Commit: abc123
# Build Date: 2025-01-15
# Go Version: go1.21.0
```

## Self-Update Policy

```bash
govman selfupdate
```

- Updates to latest **stable** release
- Never updates to pre-release unless `--prerelease` specified
- Creates backup before update
- Automatic rollback on failure

## Beta/RC Releases

Pre-release versions:
- Format: `1.2.0-rc.1`, `1.2.0-beta.2`
- Available via GitHub releases
- Not available through `govman selfupdate` (unless `--prerelease`)
- No compatibility guarantees
- For testing only

## Version Resolution

When you specify:

| Input     | Resolves To                    |
|-----------|--------------------------------|
| `latest`  | Latest stable release          |
| `1.25`    | Latest 1.25.x patch            |
| `1.25.1`  | Exact version 1.25.1           |
| `1.25rc1` | Exact pre-release 1.25rc1      |
| `default` | Configured default version     |

## Breaking Changes

Major version updates may break:

1. **Configuration format**
   - Migration guide provided
   - Tool-assisted migration

2. **CLI interface**
   - Removed commands
   - Changed flags or behavior

3. **Shell integration**
   - Different integration code
   - Automatic migration via `govman init`

4. **File formats**
   - `.govman-goversion` format (unlikely)
   - Cache/metadata formats

## Backwards Compatibility

Within the same major version:

✅ **Guaranteed**:
- Configuration files work across minor versions
- CLI commands remain stable
- `.govman-goversion` files are honored
- Installed Go versions remain accessible

❌ **Not Guaranteed**:
- Internal implementation details
- Undocumented behavior
- Cache format (automatically rebuilt)

## Version Check

Check compatibility:

```bash
# govman version
govman --version

# Minimum required for a project
cat .govman-goversion

# Update govman
govman selfupdate

# Check for breaking changes
govman selfupdate --check
```

## Changelog

Detailed changes in each release:
- GitHub Releases page
-Embedded in `govman --version` output
- Release notes documentation

## Version Pinning

For CI/CD or reproducible builds:

```bash
# Install specific govman version
curl -sSL https://install.script | bash -s -- --version v1.0.0

# Or pin in Dockerfile
FROM ubuntu:22.04
RUN curl -sSL https://install.script | bash -s -- --version v1.0.0
```

## Policy Updates

This versioning policy may be updated with:
- Notice in release notes
- Minimum 30 days before changes take effect
- No mid-version changes to policy
