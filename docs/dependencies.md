# Dependencies

govman's external dependencies and their purposes.

## Go Module Dependencies

govman uses minimal external dependencies to maintain security and simplicity.

### Direct Dependencies

#### github.com/spf13/cobra (v1.8.0+)

**Purpose**: CLI framework

**Usage**:
- Command definition and parsing
- Flag handling
- Subcommand support
- Help text generation

**Why chosen**:
- Industry standard for Go CLI applications
- Well-maintained and stable
- Excellent documentation
- Rich feature set

**License**: Apache 2.0

#### github.com/spf13/viper (v1.18.0+)

**Purpose**: Configuration management

**Usage**:
- YAML configuration file parsing
- Environment variable binding
- Configuration validation
- Default value handling

**Why chosen**:
- Works seamlessly with Cobra
- Supports multiple configuration formats
- Flexible configuration sources
- Widely adopted

**License**: MIT

## Standard Library Usage

govman extensively uses Go's standard library:

### Major Standard Library Packages

| Package            | Purpose                           |
|--------------------|-----------------------------------|
| `net/http`         | HTTP client for downloads         |
| `archive/tar`      | Tar archive extraction            |
| `archive/zip`      |Zip archive extraction            |
| `compress/gzip`    | Gzip decompression                |
| `crypto/sha256`    | Checksum verification             |
| `encoding/json`    | JSON parsing (API responses)      |
| `os`               | File system operations            |
| `os/exec`          | External command execution        |
| `path/filepath`    | Cross-platform path handling      |
| `regexp`           | Regular expressions               |
| `text/template`    | Shell integration code generation |
| `time`             | Timing and duration handling      |

## Dependency Tree

```
govman
├── github.com/spf13/cobra
│   ├── github.com/inconshreveable/mousetrap (Windows only)
│   └── github.com/spf13/pflag
└── github.com/spf13/viper
    ├── github.com/fsnotify/fsnotify
    ├── github.com/hashicorp/hcl
    ├── github.com/magiconair/properties
    ├── github.com/mitchellh/mapstructure
    ├── github.com/pelletier/go-toml/v2
    ├── github.com/sagikazarmark/locafero
    ├── github.com/sagikazarmark/slog-shim
    ├── github.com/sourcegraph/conc
    ├── github.com/spf13/afero
    ├── github.com/spf13/cast
    ├── github.com/spf13/pflag
    ├── github.com/subosito/gotenv
    ├── gopkg.in/ini.v1
    └── gopkg.in/yaml.v3
```

## No External Dependencies For

govman implements these features without external dependencies:

- **Download management**: Uses `net/http` directly
- **Progress bars**: Custom implementation
- **Shell detection**: Uses `os` and `os/exec`
- **Version comparison**: Custom semver implementation
- **Checksum verification**: Uses `crypto/sha256`
- **Archive extraction**: Uses `archive/tar` and `archive/zip`

## Dependency Management

### Version Pinning

All dependencies are pinned to specific versions in `go.mod`:

```go
require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.0
)
```

### Update Strategy

- **Patch updates**: Applied automatically for security fixes
- **Minor updates**: Reviewed and tested before adoption
- **Major updates**: Carefully evaluated for breaking changes

### Security Scanning

Dependencies are scanned for vulnerabilities using:
- `go list -m all | govulncheck`
- GitHub Dependabot alerts
- Regular dependency audits

## Build Dependencies

### Development Tools

Not included in runtime binary:

- **`golangci-lint`**: Comprehensive linter
- **`gofmt`**: Code formatting
- **`go vet`**: Static analysis
- **`delve`**: Debugging

## Optional Dependencies

### For Building from Source

- Go 1.21+
- Make (Unix/macOS)
- Git

### For Running Installation Scripts

- `curl` or `wget`
- `tar` and `gzip` (Unix/macOS)
- PowerShell 5.1+ (Windows)

## Reducing Dependencies

govman minimizes dependencies by:

1. **Standard library first**: Prefer stdlib over external packages
2. **Single purpose**: Each dependency serves a clear purpose
3. **No redundancy**: Avoid overlapping functionality
4. **Direct usage**: Avoid dependency chains where possible

## Dependency Justification

### Why Cobra?

**Alternatives considered**:
- `flag` (stdlib): Too basic, no subcommand support
- `urfave/cli`: Less feature-rich than Cobra
- Custom parser: Reinventing the wheel

**Decision**: Cobra provides the best balance of features and stability.

### Why Viper?

**Alternatives considered**:
- `encoding/json` + `gopkg.in/yaml.v3`: Manual implementation
- `koanf`: Similar features but less mature
- Custom config parser: Too much work

**Decision**: Viper integrates seamlessly with Cobra and handles complex configuration needs.

## License Compatibility

All dependencies use permissive licenses compatible with Apache 2.0:

| Dependency | License    | Compatible |
|-----------|-----------|------------|
| cobra     | Apache 2.0 | ✅         |
| viper     | MIT        | ✅         |
| pflag     | BSD-3     | ✅         |

## Dependency Updates

### Checking for Updates

```bash
# List available updates
go list -u -m all

# Update all dependencies
go get -u ./...
go mod tidy
```

### Testing After Updates

```bash
# Run full test suite
make test

# Run linters
make lint

# Build for all platforms
make release
```

## Vendoring (Optional)

govman supports dependency vendoring:

```bash
# Vendor dependencies
go mod vendor

# Build with vendor
go build -mod=vendor ./cmd/govman
```

**Benefits**:
- Reproducible builds
- Offline compilation
- Protection against dependency disappearance

**Drawbacks**:
- Larger repository size
- Manual vendor updates required

## Future Dependency Strategy

- **Minimize additions**: Add dependencies only when truly necessary
- **Evaluate alternatives**: Consider stdlib and custom implementation first
- **Monitor health**: Track maintainer activity and security
- **Gradual adoption**: Test thoroughly before upgrading major versions

## Security Considerations

### Supply Chain Security

- All dependencies fetched from trusted sources (pkg.go.dev)
- Checksums verified via `go.sum`
- Regular vulnerability scanning
- Limited dependency count reduces attack surface

### Dependency Pinning

- Exact versions specified in `go.mod`
- No `latest` or version ranges
- Updates done deliberately and tested

## Quick Reference

```bash
# View dependencies
go list -m all

# Dependency graph
go mod graph

# Why is a dependency included?
go mod why github.com/spf13/cobra

# Check for vulnerabilities
govulncheck ./...

# Tidy dependencies
go mod tidy

# Download dependencies
go mod download
```
