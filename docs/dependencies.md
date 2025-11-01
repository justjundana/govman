# Dependencies

Complete reference for govman's external and internal dependencies.

## Direct Dependencies

### CLI Framework

#### cobra (github.com/spf13/cobra)
- **Version**: v1.8.0+
- **Purpose**: Command-line interface framework
- **Usage**: Command structure, flags, subcommands
- **License**: Apache 2.0

**Key Features Used**:
- Root command and subcommands
- Flag parsing (global and local flags)
- Command aliases
- Help generation
- Completion generation

**Example**:
```go
rootCmd := &cobra.Command{
    Use:   "govman",
    Short: "Go Version Manager",
    Run:   func(cmd *cobra.Command, args []string) {},
}
```

#### viper (github.com/spf13/viper)
- **Version**: v1.18.0+
- **Purpose**: Configuration management
- **Usage**: YAML config file reading/writing
- **License**: MIT

**Key Features Used**:
- YAML configuration
- Environment variable binding
- Default values
- Configuration validation

**Example**:
```go
viper.SetConfigFile("config.yaml")
viper.SetDefault("install_dir", "~/.govman/versions")
viper.ReadInConfig()
```

### Compression and Archives

#### compress/gzip (standard library)
- **Purpose**: Gzip decompression for .tar.gz files
- **Usage**: Extract downloaded Go distributions

#### archive/tar (standard library)
- **Purpose**: TAR archive extraction
- **Usage**: Extract .tar.gz archives (Unix/Linux/macOS)

#### archive/zip (standard library)
- **Purpose**: ZIP archive extraction
- **Usage**: Extract .zip archives (Windows)

### HTTP Client

#### net/http (standard library)
- **Purpose**: HTTP downloads
- **Usage**: Download Go distributions and release data

**Features Used**:
- GET requests with custom headers
- Range requests for resume
- Progress tracking
- Timeout handling
- Redirect following

**Example**:
```go
req, _ := http.NewRequest("GET", url, nil)
req.Header.Set("User-Agent", "govman/1.0.0")
if resumeFrom > 0 {
    req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeFrom))
}
```

### Cryptography

#### crypto/sha256 (standard library)
- **Purpose**: SHA-256 checksum verification
- **Usage**: Verify downloaded file integrity

**Example**:
```go
hasher := sha256.New()
io.Copy(hasher, file)
calculatedHash := hex.EncodeToString(hasher.Sum(nil))
```

### JSON Processing

#### encoding/json (standard library)
- **Purpose**: JSON parsing
- **Usage**: Parse Go releases API response

**Example**:
```go
var releases []Release
json.Unmarshal(data, &releases)
```

### File System Operations

#### os (standard library)
- **Purpose**: File and directory operations
- **Usage**: File creation, deletion, symlinks

#### path/filepath (standard library)
- **Purpose**: Path manipulation
- **Usage**: Path joining, cleaning, directory traversal

#### io (standard library)
- **Purpose**: I/O operations
- **Usage**: File copying, reading, writing

### Process Execution

#### os/exec (standard library)
- **Purpose**: Execute external commands
- **Usage**: Run shell commands for PATH setup

**Example**:
```go
cmd := exec.Command("bash", "-c", "source ~/.bashrc && govman --version")
```

### Concurrency

#### sync (standard library)
- **Purpose**: Synchronization primitives
- **Usage**: Singleton logger, mutex for thread safety

**Example**:
```go
var once sync.Once
once.Do(func() {
    globalLogger = New()
})
```

### Runtime Detection

#### runtime (standard library)
- **Purpose**: Platform detection
- **Usage**: Determine OS and architecture

**Example**:
```go
goos := runtime.GOOS    // "linux", "darwin", "windows"
goarch := runtime.GOARCH // "amd64", "arm64", "386"
```

### Version Comparison

#### golang.org/x/mod/semver
- **Version**: v0.14.0+
- **Purpose**: Semantic version comparison
- **Usage**: Compare Go versions (1.20.1 vs 1.21.0)
- **License**: BSD-3-Clause

**Example**:
```go
import "golang.org/x/mod/semver"

if semver.Compare("v1.21.0", "v1.20.5") > 0 {
    // v1.21.0 is newer
}
```

## Development Dependencies

### Testing

#### testing (standard library)
- **Purpose**: Unit tests
- **Usage**: All `*_test.go` files

**Example**:
```go
func TestInstall(t *testing.T) {
    // Test code
}
```

#### testify (github.com/stretchr/testify)
- **Version**: v1.8.0+ (optional)
- **Purpose**: Test assertions and mocking
- **Usage**: Enhanced test readability
- **License**: MIT

**Example**:
```go
import "github.com/stretchr/testify/assert"

assert.Equal(t, expected, actual)
assert.NoError(t, err)
```

### Build Tools

#### Make
- **Purpose**: Build automation
- **Usage**: Makefile for common tasks

**Targets**:
- `make build` - Build binary
- `make test` - Run tests
- `make install` - Install locally
- `make clean` - Clean build artifacts

## Indirect Dependencies

These are dependencies of our direct dependencies (transitive):

### From cobra/viper:
- `github.com/inconshreveable/mousetrap` - Windows compatibility
- `github.com/spf13/pflag` - POSIX/GNU-style flags
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/fsnotify/fsnotify` - File watching
- `github.com/hashicorp/hcl` - HCL support
- `github.com/magiconair/properties` - Properties file support
- `github.com/pelletier/go-toml` - TOML support
- `github.com/subosito/gotenv` - .env file support

### From golang.org/x/mod:
- `golang.org/x/xerrors` - Error handling

## Standard Library Dependencies

Complete list of standard library packages used:

| Package | Purpose |
|---------|---------|
| `archive/tar` | TAR extraction |
| `archive/zip` | ZIP extraction |
| `compress/gzip` | Gzip decompression |
| `context` | Context management |
| `crypto/sha256` | SHA-256 hashing |
| `encoding/hex` | Hexadecimal encoding |
| `encoding/json` | JSON parsing |
| `errors` | Error handling |
| `fmt` | Formatting |
| `io` | I/O operations |
| `io/ioutil` | I/O utilities |
| `net/http` | HTTP client |
| `os` | OS interface |
| `os/exec` | Process execution |
| `path/filepath` | Path manipulation |
| `runtime` | Runtime info |
| `strings` | String utilities |
| `sync` | Synchronization |
| `time` | Time operations |

## Dependency Management

### go.mod

```go
module github.com/justjundana/govman

go 1.20

require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.0
    golang.org/x/mod v0.14.0
)
```

### Updating Dependencies

```bash
# Update all dependencies
go get -u ./...

# Update specific dependency
go get -u github.com/spf13/cobra

# Tidy dependencies
go mod tidy

# Verify dependencies
go mod verify
```

### Vendoring (Optional)

```bash
# Vendor dependencies
go mod vendor

# Build with vendored dependencies
go build -mod=vendor ./cmd/govman
```

## External APIs

### Go Download API

**Endpoint**: `https://go.dev/dl/?mode=json`

**Purpose**: Fetch available Go versions

**Response Format**:
```json
[
  {
    "version": "go1.21.5",
    "stable": true,
    "files": [
      {
        "filename": "go1.21.5.linux-amd64.tar.gz",
        "os": "linux",
        "arch": "amd64",
        "sha256": "...",
        "size": 67108864
      }
    ]
  }
]
```

**Rate Limiting**: None specified, but cached locally

**Caching**: 1 hour (configurable)

### Go Binary Downloads

**Endpoint**: `https://go.dev/dl/<filename>`

**Example**: `https://go.dev/dl/go1.21.5.linux-amd64.tar.gz`

**Mirror Support**: Configurable mirror URLs

## Dependency Security

### Vulnerability Scanning

```bash
# Check for known vulnerabilities
go list -json -m all | nancy sleuth

# Or use govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### Checksum Verification

All dependencies are verified via `go.sum`:

```
github.com/spf13/cobra v1.8.0 h1:...
github.com/spf13/cobra v1.8.0/go.mod h1:...
```

### Supply Chain Security

- All dependencies are from trusted sources
- Go module proxy provides integrity verification
- HTTPS for all downloads
- SHA-256 checksums for Go distributions

## Dependency Licenses

| Package | License | Commercial Use |
|---------|---------|----------------|
| cobra | Apache 2.0 | ✅ Yes |
| viper | MIT | ✅ Yes |
| golang.org/x/mod | BSD-3-Clause | ✅ Yes |
| Go standard library | BSD-3-Clause | ✅ Yes |

All dependencies are permissively licensed and safe for commercial use.

## Minimal Dependency Philosophy

govman follows a minimal dependency approach:

### Why Minimal Dependencies?

1. **Security**: Fewer dependencies = smaller attack surface
2. **Reliability**: Less risk of breakage from updates
3. **Performance**: Smaller binary size
4. **Maintenance**: Fewer updates needed

### Dependencies We Avoided

- **UI frameworks**: Used standard output instead
- **HTTP libraries**: Standard `net/http` is sufficient
- **Logging frameworks**: Built custom lightweight logger
- **Configuration libraries**: Viper handles our needs
- **Progress bars**: Built custom implementation

### When We Add Dependencies

Only when:
1. Feature requires complex implementation
2. Standard library doesn't provide functionality
3. Dependency is well-maintained and widely used
4. License is permissive

## Binary Size Impact

| Component | Size Contribution |
|-----------|-------------------|
| Go standard library | ~1.2 MB |
| cobra | ~500 KB |
| viper | ~800 KB |
| golang.org/x/mod | ~100 KB |
| govman code | ~300 KB |
| **Total** | **~3 MB** |

Stripped and compressed: **~2 MB**

## Version Compatibility

### Go Version Requirements

- **Minimum**: Go 1.20
- **Recommended**: Go 1.21+
- **Tested**: Go 1.20, 1.21, 1.22

### Dependency Version Strategy

- Use stable versions (no pre-releases)
- Pin major versions to avoid breaking changes
- Test before upgrading major versions
- Keep dependencies reasonably up-to-date for security

## See Also

- [Project Structure](project-structure.md) - Code organization
- [Architecture](architecture.md) - System design
- [Getting Started](getting-started.md) - Development setup

---

Understanding dependencies helps with security, maintenance, and troubleshooting! 📦
