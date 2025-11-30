# Contributing to govman

Guidelines for contributing to the govman project.

## Welcome Contributors!

Thank you for your interest in contributing to govman. All contributions are welcome:

- Bug reports
- Feature requests
- Documentation improvements
- Code contributions

## Code of Conduct

- Be respectful and inclusive
- Provide constructive feedback
- Focus on what is best for the project and community
- Show empathy towards other community members

## How to Contribute

### Reporting Bugs

1. **Search existing issues** to avoid duplicates
2. **Use the bug report template**
3. **Include**:
   - govman version (`govman --version`)
   - Operating system and version
   - Shell type and version (`echo $SHELL`)
   - Steps to reproduce
   - Expected vs actual behavior
   - Error messages (full output)

**Example**:

```
**govman version**: 1.0.0
**OS**: Ubuntu 22.04
**Shell**: bash 5.1.16

**Steps to reproduce**:
1. govman install 1.25.1
2. govman use 1.25.1
3. go version

**Expected**: go version go1.25.1 linux/amd64
**Actual**: go: command not found

**Error output**:
[paste full error]
```

### Requesting Features

1. **Search existing issues** for similar requests
2. **Describe the use case** clearly
3. **Explain the benefit** to users
4. **Provide examples** of how it would work

**Template**:

````
**Feature**: [Brief description]

**Use case**: [Why is this needed?]

**Proposed behavior**:
```bash
govman new-command --flag
# Expected output
```

**Alternatives considered**: [Other ways to achieve this]

**Additional context**: [Any relevant information]
````

### Improving Documentation

Documentation improvements are always welcome!

1. **Fork the repository**
2. **Edit files in `docs/`**
3. **Follow markdown formatting**:
   - Use headers appropriately
   - Include code examples
   - Add cross-references where helpful
4. **Submit Pull Request**

## Development Setup

### Prerequisites

- Go 1.25 or later
- Git
- Make (Linux/macOS) or equivalent

### Setup Instructions

```bash
# 1. Fork the repository on GitHub

# 2. Clone your fork
git clone https://github.com/YOUR_USERNAME/govman.git
cd govman

# 3. Add upstream remote
git remote add upstream https://github.com/justjundana/govman.git

# 4. Install dependencies
go mod download

# 5. Build
make build

# 6. Run
./govman --help

# 7. Run tests
make test
```

## Making Changes

### Workflow

1. **Create a branch**:
   ```bash
   git checkout -b feature/my-feature
   # or
   git checkout -b fix/bug-description
   ```

2. **Make your changes**
   - Write code
   - Add tests
   - Update documentation

3. **Test your changes**:
   ```bash
   make test
   make lint
   ```

4. **Commit**:
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

5. **Push**:
   ```bash
   git push origin feature/my-feature
   ```

6. **Create Pull Request** on GitHub

### Coding Standards

#### Go Code Style

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for formatting
- Run `go vet` before committing
- Use meaningful variable names
- Add comments for exported functions

**Example**:

```go
// Install downloads and installs the specified Go version.
// It returns an error if the version is not available or installation fails.
func (m *Manager) Install(version string) error {
    resolved, err := m.ResolveVersion(version)
    if err != nil {
        return fmt.Errorf("failed to resolve version: %w", err)
    }
    // ... implementation
}
```

#### Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): description

[optional body]

[optional footer]
```

**Types**:

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test changes
- `refactor`: Code refactoring
- `chore`: Maintenance tasks
- `perf`: Performance improvements

**Examples**:

```
feat: add support for Fish shell
fix: resolve symlink creation on Windows
docs: update installation instructions
test: add tests for version resolution
refactor: simplify download logic
chore: update dependencies
```

### Testing

#### Writing Tests

- Test files: `*_test.go` alongside implementation
- Test functions: `func TestFeatureName(t *testing.T)`
- Use table-driven tests for multiple scenarios
- Test both happy path and error cases

**Example**:

```go
func TestManager_ResolveVersion(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "latest resolves to newest stable",
            input:    "latest",
            expected: "1.25.1",
            wantErr:  false,
        },
        {
            name:     "partial version resolves",
            input:    "1.25",
            expected: "1.25.1",
            wantErr:  false,
        },
        {
            name:     "invalid version errors",
            input:    "99.99.99",
            expected: "",
            wantErr:  true,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

#### Running Tests

```bash
# Run all tests
make test

# Run specific package
go test ./internal/manager/

# Run with coverage
make test-coverage

# Run specific test
go test -run TestManager_Install ./internal/manager/

# Verbose output
go test -v ./...
```

### Documentation

Update documentation for:

- New commands: Update `docs/commands.md`
- Configuration changes: Update `docs/configuration.md`
- New features: Update relevant docs and `docs/examples.md`
- Breaking changes: Update `CHANGELOG.md` and migration guides

## Pull Request Process

### Before Submitting

- [ ] Code follows style guidelines
- [ ] Tests added/updated
- [ ] All tests pass
- [ ] Documentation updated
- [ ] Commit messages follow convention
- [ ] Branch is up-to-date with main

### PR Description

Use this template:

```markdown
## Description
[Clear description of changes]

## Type of Change
- [ ] Bug fix (non-breaking)
- [ ] New feature (non-breaking)
- [ ] Breaking change
- [ ] Documentation update

## Testing
[How you tested your changes]

## Checklist
- [ ] Tests pass
- [ ] Linters pass
- [ ] Documentation updated
- [ ] CHANGELOG.md updated (if applicable)

## Related Issues
Fixes #123
```

### Review Process

1. **Automated checks**: Must pass CI/CD
2. **Code review**: Maintainers will review
3. **Feedback**: Address comments/suggestions
4. **Approval**: At least one maintainer approval required
5. **Merge**: Squash and merge to main

### After Merge

- Your contribution will be in the next release
- You'll be credited in release notes
- Thank you for contributing!

## Areas Needing Help

Good first issues:

- Documentation improvements
- Test coverage improvements
- Error message enhancements
- Platform-specific testing

Medium complexity:

- New shell support
- Performance optimizations
- New commands/features

Advanced:

- Core architecture changes
- Security enhancements
- Cross-platform compatibility

## Development Tips

### Debugging

```bash
# Verbose mode
govman --verbose install 1.25.1

# Using delve debugger
dlv debug ./cmd/govman -- install 1.25.1
(dlv) break main.main
(dlv) continue
```

### Testing Locally

```bash
# Build and test
make build
./govman install 1.25.1

# Test on different platforms (requires Docker)
docker run --rm -v $(pwd):/app -w /app golang:1.21 make build
```

### Useful Commands

```bash
# Format code
make fmt

# Run linters
make lint

# Build for all platforms
make release

# Clean build artifacts
make clean

# View coverage
go tool cover -html=coverage.out
```

## Release Process

(For maintainers)

1. Update version in `internal/version/version.go`
2. Update `CHANGELOG.md`
3. Create tag: `git tag -a v1.0.0 -m "Release v1.0.0"`
4. Push tag: `git push origin v1.0.0`
5. GitHub Actions builds and publishes
6. Create GitHub Release with notes

## Questions?

- **Documentation**: See `docs/` directory
- **Discussions**: Use GitHub Discussions
- **Issues**: Open an issue for bugs/features
- **Contact**: [maintainer email if available]

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.

## Recognition

Contributors are recognized in:

- Release notes
- CONTRIBUTORS file
- Project README (for significant contributions)

Thank you for making govman better!