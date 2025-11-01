# Internals Troubleshooting

Developer guide for debugging and troubleshooting govman internals.

## Development Environment Issues

### Build Failures

#### Problem: Go version mismatch

```bash
$ make build
go: directive requires go >= 1.20
```

**Solution**:
```bash
# Check Go version
go version

# Install Go 1.20+
# Then rebuild
make clean
make build
```

#### Problem: Missing dependencies

```bash
$ go build ./cmd/govman
package github.com/spf13/cobra: cannot find package
```

**Solution**:
```bash
# Download dependencies
go mod download

# Verify modules
go mod verify

# Rebuild
go build ./cmd/govman
```

#### Problem: Build errors after git pull

```bash
$ make build
./internal/cli/cli.go:15:2: undefined: newFunction
```

**Solution**:
```bash
# Clean build cache
go clean -cache -modcache

# Re-download dependencies
go mod tidy
go mod download

# Rebuild
make build
```

### Test Failures

#### Problem: Tests fail in CI but pass locally

```bash
=== RUN   TestDownload
--- FAIL: TestDownload (0.00s)
    downloader_test.go:45: timeout exceeded
```

**Possible Causes**:
1. Network connectivity in CI
2. Different OS/architecture
3. Race conditions
4. File permission differences

**Solution**:
```bash
# Run with verbose output
go test -v ./internal/downloader

# Run with race detector
go test -race ./...

# Run specific test
go test -run TestDownload ./internal/downloader

# Run integration tests
go test -tags=integration ./...
```

#### Problem: Flaky tests

**Diagnosis**:
```bash
# Run test multiple times
go test -count=100 -run TestAutoSwitch ./internal/shell

# With race detector
go test -race -count=100 -run TestAutoSwitch ./internal/shell
```

**Solution**:
```go
// Add proper synchronization
func TestAutoSwitch(t *testing.T) {
    var wg sync.WaitGroup
    wg.Add(1)
    
    go func() {
        defer wg.Done()
        // Test code
    }()
    
    wg.Wait()
}

// Add timeouts
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

#### Problem: Test coverage issues

```bash
$ make test
coverage: 65.2% of statements
```

**Improve Coverage**:
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View in browser
go tool cover -html=coverage.out

# Check per-package coverage
go test -cover ./internal/...
```

## Runtime Issues

### Debugging Crashes

#### Problem: Panic in production

```
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x1234567]
```

**Debug Steps**:

1. **Enable stack traces**:
```bash
# Set environment variable
export GOTRACEBACK=all

# Run with debug symbols
go build -gcflags="all=-N -l" -o govman-debug ./cmd/govman
./govman-debug install 1.21.5
```

2. **Add panic recovery**:
```go
func (m *Manager) Install(version string) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic recovered: %v\n%s", r, debug.Stack())
        }
    }()
    
    // Function logic
}
```

3. **Use delve debugger**:
```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug
dlv exec ./govman -- install 1.21.5

# In delve
(dlv) break internal/manager/manager.go:45
(dlv) continue
(dlv) print version
(dlv) locals
```

### Memory Leaks

#### Problem: High memory usage

```bash
$ ps aux | grep govman
user  1234  50.0  2.5GB govman install
```

**Diagnosis**:

1. **Profile memory**:
```go
// Add to main.go
import (
    "runtime"
    "runtime/pprof"
)

func main() {
    f, _ := os.Create("mem.prof")
    defer f.Close()
    defer pprof.WriteHeapProfile(f)
    
    cli.Execute()
}
```

```bash
# Run and analyze
go build ./cmd/govman
./govman install 1.21.5
go tool pprof mem.prof

# In pprof
(pprof) top10
(pprof) list Manager.Install
```

2. **Common causes**:
```go
// BAD: Goroutine leak
go func() {
    // Never returns
    for {
        // Work without exit condition
    }
}()

```go
// GOOD: Proper cleanup
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            // Work
        }
    }
}()
```

```go
// BAD: Growing slice
var data []byte
for {
    data = append(data, moreData...) // Unbounded growth
}

// GOOD: Bounded buffer
buf := make([]byte, 0, maxSize)
for {
    if len(buf) >= maxSize {
        break
    }
    buf = append(buf, moreData...)
}
```
```

```go
// BAD: Growing slice
var data []byte
for {
    data = append(data, moreData...) // Unbounded growth
}

// GOOD: Bounded buffer
buf := make([]byte, 0, maxSize)
for {
    if len(buf) >= maxSize {
        break
    }
    buf = append(buf, moreData...)
}
```

### Performance Issues

#### Problem: Slow downloads

**Debug**:
```go
// Add timing instrumentation
import "time"

func (d *Downloader) Download(url string) error {
    start := time.Now()
    defer func() {
        _logger.Info("Download took: %v", time.Since(start))
    }()
    
    // Download logic
}
```

**Profile**:
```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=BenchmarkDownload ./internal/downloader
go tool pprof cpu.prof
```

**Optimize**:
```go
// Use larger buffer
const bufferSize = 64 * 1024 // 64KB instead of 32KB
buf := make([]byte, bufferSize)

// Parallel downloads (if multiple files)
var wg sync.WaitGroup
for _, file := range files {
    wg.Add(1)
    go func(f File) {
        defer wg.Done()
        download(f)
    }(file)
}
wg.Wait()
```

## Debugging Techniques

### Logging Strategies

#### Add Debug Logging

```go
// internal/logger/logger.go
const (
    LevelQuiet   = 0
    LevelNormal  = 1
    LevelVerbose = 2
    LevelDebug   = 3  // Add debug level
)

func (l *Logger) Debug(format string, args ...interface{}) {
    if l.level >= LevelDebug {
        l.log("DEBUG", fmt.Sprintf(format, args...))
    }
}
```

**Usage**:
```go
func (m *Manager) Install(version string) error {
    _logger.Debug("Install called with version: %s", version)
    _logger.Debug("Config: %+v", m.config)
    
    release, err := m.golang.GetRelease(version)
    _logger.Debug("Got release: %+v, err: %v", release, err)
    
    // More logic
}
```

**Enable**:
```bash
govman --log-level=debug install 1.21.5
```

### Tracing Execution

#### Add trace logging

```go
// internal/util/trace.go
package util

import (
    "fmt"
    "runtime"
    "time"
)

func Trace() func() {
    pc, _, _, _ := runtime.Caller(1)
    fn := runtime.FuncForPC(pc)
    name := fn.Name()
    
    start := time.Now()
    fmt.Printf("→ Enter: %s\n", name)
    
    return func() {
        fmt.Printf("← Exit:  %s (took %v)\n", name, time.Since(start))
    }
}
```

**Usage**:
```go
func (m *Manager) Install(version string) error {
    defer util.Trace()()
    
    // Function logic
}
```

**Output**:
```
→ Enter: github.com/justjundana/govman/internal/manager.(*Manager).Install
→ Enter: github.com/justjundana/govman/internal/downloader.(*Downloader).Download
← Exit:  github.com/justjundana/govman/internal/downloader.(*Downloader).Download (took 5.2s)
← Exit:  github.com/justjundana/govman/internal/manager.(*Manager).Install (took 8.5s)
```

### Interactive Debugging

#### Using delve

```bash
# Start debugging
dlv debug ./cmd/govman -- install 1.21.5

# Set breakpoints
(dlv) break Manager.Install
(dlv) break downloader.go:45

# Run
(dlv) continue

# Inspect variables
(dlv) print version
(dlv) print m.config
(dlv) print err

# Step through
(dlv) next  # Next line
(dlv) step  # Step into function
(dlv) stepout  # Step out of function

# View stack
(dlv) stack

# List goroutines
(dlv) goroutines

# Switch goroutines
(dlv) goroutine 5
```

#### Using print debugging

```go
// Quick debug prints
func (m *Manager) Install(version string) error {
    fmt.Printf("DEBUG: Install called\n")
    fmt.Printf("DEBUG: version=%q\n", version)
    fmt.Printf("DEBUG: config=%+v\n", m.config)
    
    // Logic
    
    fmt.Printf("DEBUG: Install complete\n")
    return nil
}
```

## Common Internal Bugs

### Symlink Issues

#### Problem: Broken symlinks on Windows

```go
// BAD: Unix-only
os.Symlink(target, linkName)

// GOOD: Cross-platform
func CreateSymlink(target, linkName string) error {
    if runtime.GOOS == "windows" {
        // Use junction or directory symlink on Windows
        return createWindowsSymlink(target, linkName)
    }
    return os.Symlink(target, linkName)
}
```

### Path Issues

#### Problem: Path traversal vulnerability

```go
// BAD: Unsafe
extractPath := filepath.Join(destDir, entry.Name)

// GOOD: Validated
func SafeJoin(base, target string) (string, error) {
    result := filepath.Join(base, target)
    rel, err := filepath.Rel(base, result)
    if err != nil || strings.HasPrefix(rel, "..") {
        return "", fmt.Errorf("invalid path: %s", target)
    }
    return result, nil
}
```

### Concurrency Bugs

#### Problem: Race condition in cache

```go
// BAD: Race condition
var cache map[string][]Release

func GetReleases() []Release {
    if val, ok := cache["releases"]; ok {  // Read
        return val
    }
    
    releases := fetchReleases()
    cache["releases"] = releases  // Write
    return releases
}

// GOOD: Thread-safe
var (
    cache   map[string][]Release
    cacheMu sync.RWMutex
)

func GetReleases() []Release {
    cacheMu.RLock()
    if val, ok := cache["releases"]; ok {
        cacheMu.RUnlock()
        return val
    }
    cacheMu.RUnlock()
    
    cacheMu.Lock()
    defer cacheMu.Unlock()
    
    // Double-check after acquiring write lock
    if val, ok := cache["releases"]; ok {
        return val
    }
    
    releases := fetchReleases()
    cache["releases"] = releases
    return releases
}
```

### Error Handling

#### Problem: Swallowed errors

```go
// BAD: Silent failure
result, _ := doSomething()

// GOOD: Proper handling
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

#### Problem: Panic instead of error

```go
// BAD: Panic on error
if err != nil {
    panic(err)
}

// GOOD: Return error
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

## Testing Strategies

### Unit Test Isolation

```go
// Use test helpers
func setupTest(t *testing.T) *Manager {
    t.Helper()
    
    // Create temp directory
    tmpDir := t.TempDir()
    
    // Create test config
    cfg := &Config{
        InstallDir: filepath.Join(tmpDir, "versions"),
        CacheDir:   filepath.Join(tmpDir, "cache"),
    }
    
    // Create manager
    return New(cfg)
}

func TestInstall(t *testing.T) {
    mgr := setupTest(t)
    
    // Test with isolated manager
    err := mgr.Install("1.21.5")
    if err != nil {
        t.Fatalf("Install failed: %v", err)
    }
}
```

### Integration Tests

```go
//go:build integration

package manager_test

import (
    "testing"
    "time"
)

func TestRealDownload(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    mgr := setupTest(t)
    
    // Real network call
    err := mgr.Install("1.21.5")
    if err != nil {
        t.Fatalf("Real install failed: %v", err)
    }
}
```

**Run**:
```bash
# Skip integration tests
go test -short ./...

# Run only integration tests
go test -tags=integration ./...
```

### Mock External Dependencies

```go
// Define interface
type GolangAPI interface {
    GetReleases() ([]Release, error)
}

// Real implementation
type RealGolangAPI struct{}

func (r *RealGolangAPI) GetReleases() ([]Release, error) {
    // Real API call
}

// Mock for tests
type MockGolangAPI struct {
    Releases []Release
    Error    error
}

func (m *MockGolangAPI) GetReleases() ([]Release, error) {
    return m.Releases, m.Error
}

// Test with mock
func TestWithMock(t *testing.T) {
    mockAPI := &MockGolangAPI{
        Releases: []Release{{Version: "go1.21.5"}},
    }
    
    mgr := &Manager{golang: mockAPI}
    // Test
}
```

## Profiling and Optimization

### CPU Profiling

```bash
# Profile a command
go build -o govman ./cmd/govman
govman install 1.21.5 -cpuprofile=cpu.prof

# Analyze
go tool pprof cpu.prof
(pprof) top10
(pprof) list Manager.Install
(pprof) web  # Generate SVG graph
```

### Memory Profiling

```bash
# Profile memory
govman install 1.21.5 -memprofile=mem.prof

# Analyze
go tool pprof mem.prof
(pprof) top10
(pprof) list
```

### Benchmark Tests

```go
func BenchmarkInstall(b *testing.B) {
    mgr := setupTest(b)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        mgr.Install("1.21.5")
    }
}
```

**Run**:
```bash
# Run benchmarks
go test -bench=. ./internal/manager

# With memory profiling
go test -bench=. -benchmem ./internal/manager

# Compare before/after
go test -bench=. -benchmem > old.txt
# Make changes
go test -bench=. -benchmem > new.txt
benchstat old.txt new.txt
```

## CI/CD Debugging

### GitHub Actions Issues

#### Problem: Tests fail only in CI

**Debug**:
```yaml
# .github/workflows/test.yml
- name: Run tests with verbose output
  run: |
    go test -v -race ./...
  env:
    GOVMAN_DEBUG: "1"
```

#### Problem: Build fails on specific OS

```yaml
# Test locally with act
act -j test

# Or use Docker
docker run --rm -v $PWD:/work -w /work golang:1.21 go test ./...
```

## Getting Help

### Collect Diagnostic Information

```bash
# Create diagnostics script
cat > diagnose.sh << 'EOF'
#!/bin/bash
echo "=== System Info ==="
uname -a
echo ""

echo "=== Go Version ==="
go version
echo ""

echo "=== govman Version ==="
govman --version
echo ""

echo "=== govman Config ==="
cat ~/.govman/config.yaml
echo ""

echo "=== Installed Versions ==="
ls -la ~/.govman/versions/
echo ""

echo "=== Last Error ==="
tail -n 50 ~/.govman/govman.log
EOF

chmod +x diagnose.sh
./diagnose.sh > diagnostics.txt
```

### Report Issues

Include in bug reports:
1. Diagnostics output
2. Exact command that failed
3. Expected vs actual behavior
4. Stack trace if available
5. OS and Go version

## See Also

- [Getting Started](getting-started.md) - Development setup
- [Architecture](architecture.md) - System design
- [Troubleshooting](troubleshooting.md) - User-facing issues

---

Effective debugging makes development faster and more enjoyable! 🐛
