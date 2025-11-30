# Team Onboarding

Guide for onboarding team members to use govman in a collaborative development environment.

## For Team Leads

### Setting Up govman for Your Team

#### 1. Choose Standard Go Version

```bash
# Decide on team's Go version
cd /path/to/project
govman use 1.25.1 --local

# This creates .govman-goversion file
git add .govman-goversion
git commit -m "Set Go version to 1.25.1"
git push
```

#### 2. Create Team Documentation

Create `DEVELOPMENT.md` in your repository:

````markdown
# Development Setup

## Go Version Management

This project uses govman for Go version management.

### Quick Setup

1. Install govman:
   ```bash
   curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
   ```

2. Initialize shell integration:
   ```bash
   govman init
   source ~/.bashrc
   ```

3. Install project Go version:
   ```bash
   govman install $(cat .govman-goversion)
   ```

4. Verify:
   ```bash
   go version
   ```

The correct Go version should automatically activate when you enter this directory.
````

#### 3. Add to README

````markdown
## Prerequisites

- [govman](https://github.com/justjundana/govman) for Go version management

## Setup

```bash
# Install govman (if not already installed)
curl -sSL https://install.script | bash

# Install project dependencies
govman install $(cat .govman-goversion)
go mod download
```
````

#### 4. CI/CD Configuration

**GitHub Actions** (`.github/workflows/ci.yml`):

```yaml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install govman
        run: |
          curl -sSL https://install.script | bash
          echo "$HOME/.govman/bin" >> $GITHUB_PATH
      
      - name: Install Go
        run: |
          govman install $(cat .govman-goversion)
          govman use $(cat .govman-goversion)
      
      - name: Verify Go version
        run: go version
      
      - name: Run tests
        run: go test -v ./...
```

**GitLab CI** (`.gitlab-ci.yml`):

```yaml
image: ubuntu:latest

before_script:
  - apt-get update && apt-get install -y curl tar gzip
  - curl -sSL https://install.script | bash
  - export PATH="$HOME/.govman/bin:$PATH"
  - govman install $(cat .govman-goversion)
  - govman use $(cat .govman-goversion)

test:
  script:
    - go version
    - go test -v ./...
```

## For Team Members

### First-Time Setup

#### 1. Install govman

**Linux/macOS**:

```bash
curl -sSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
```

**Windows (PowerShell)**:

```powershell
irm https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.ps1 | iex
```

#### 2. Initialize Shell Integration

```bash
govman init
source ~/.bashrc  # or ~/.zshrc on macOS
```

**Restart your terminal** or run the source command.

#### 3. Clone Project

```bash
git clone https://github.com/yourteam/project.git
cd project
```

#### 4. Install Go Version

```bash
# Auto-reads from .govman-goversion
govman install $(cat .govman-goversion)

# Or manually if you know the version
govman install 1.25.1
```

#### 5. Verify Setup

```bash
go version
# Should show the version from .govman-goversion

which go
# Should show path under ~/.govman/versions/
```

### Daily Workflow

With shell integration enabled, version switching is automatic:

```bash
cd ~/projects/project-a  # Uses Go 1.25.1 (auto-switches)
go version

cd ~/projects/project-b  # Uses Go 1.24.0 (auto-switches)
go version
```

## Common Team Scenarios

### Scenario 1: Upgrading Go Version

**Team Lead**:

```bash
# Test new version
govman install 1.26.0
govman use 1.26.0

# Run full test suite
go test ./...

# If all tests pass, update project
govman use 1.26.0 --local
git add .govman-goversion
git commit -m "Upgrade to Go 1.26.0"
git push
```

**Team Members**:

```bash
# Pull latest changes
git pull

# Install new version
govman install $(cat .govman-goversion)

# Verify (auto-switches when in project directory)
go version
```

### Scenario 2: Multiple Projects with Different Versions

```bash
# Project A uses 1.25.1
cd ~/projects/project-a
govman install 1.25.1
go version  # 1.25.1 (auto-switched)

# Project B uses 1.24.0
cd ~/projects/project-b
govman install 1.24.0
go version  # 1.24.0 (auto-switched)

# Personal project uses latest
cd ~/projects/personal
govman install latest
govman use latest --default
go version  # Latest version
```

### Scenario 3: Testing Across Multiple Go Versions

```bash
# Install versions
govman install 1.25.1 1.24.0 1.23.5

# Test script
#!/bin/bash
for version in 1.25.1 1.24.0 1.23.5; do
  echo "Testing with Go $version"
  govman use $version
  go test ./... || echo "Failed on $version"
done
```

## Best Practices for Teams

### 1. Always Commit .govman-goversion

```bash
# Good
echo "1.25.1" > .govman-goversion
git add .govman-goversion

# Bad
# Forgetting to commit, team members use different versions
```

### 2. Pin Exact Versions

```bash
# Good
echo "1.25.1" > .govman-goversion

# Avoid (ambiguous)
echo "1.25" > .govman-goversion  # Which patch version?
echo "latest" > .govman-goversion  # Changes over time
```

### 3. Document in README

Always mention govman in project README:

- Link to installation instructions
- Explain auto-switching behavior
- Provide troubleshooting tips

### 4. Use in CI/CD

Ensure CI/CD uses same Go version:

- Install govman in CI
- Use version from `.govman-goversion`
- Verify with `go version`

### 5. Onboarding Checklist

Provide new team members with:

```markdown
- [ ] Install govman
- [ ] Run `govman init` and reload shell
- [ ] Clone project repository
- [ ] Run `govman install $(cat .govman-goversion)`
- [ ] Verify `go version` shows correct version
- [ ] Run `go mod download` and `go build`
- [ ] Run tests: `go test ./...`
```

## Troubleshooting for Teams

### Team Member Has Different Go Version

**Problem**: Developer's `go version` doesn't match `.govman-goversion`.

**Solution**:

```bash
# Ensure in project directory
cd /path/to/project

# Check .govman-goversion
cat .govman-goversion

# Install that version
govman install $(cat .govman-goversion)

# If auto-switch not working
govman use $(cat .govman-goversion)

# Or refresh
govman refresh
```

### CI Build Fails with "Go version mismatch"

**Problem**: CI using different Go version than local development.

**Solution**:

```yaml
# In CI config, use .govman-goversion
- govman install $(cat .govman-goversion)
- govman use $(cat .govman-goversion)
- go version  # Verify
```

### New Team Member: "govman: command not found"

**Problem**: govman not in PATH.

**Solution**:

```bash
# Check if installed
ls ~/.govman/bin/govman

# If not installed
curl -sSL https://install.script | bash

# If installed but not in PATH
govman init --force
source ~/.bashrc
```

## Training Resources

### Quick Start Session (15 minutes)

1. **Install govman** (5 min)
   - Run installation script
   - Initialize shell integration

2. **Basic Commands** (5 min)
   - `govman list` - See installed versions
   - `govman install X` - Install version
   - `govman use X` - Switch version

3. **Project Setup** (5 min)
   - Clone project
   - Install version from `.govman-goversion`
   - Verify auto-switching

### Team Workshop (45 minutes)

1. **Introduction** (10 min)
   - Why version management?
   - govman benefits
   - Team workflow

2. **Hands-on Setup** (15 min)
   - Install govman
   - Practice commands
   - Set up first project

3. **Advanced Topics** (10 min)
   - Multiple projects
   - Version upgrades
   - Troubleshooting

4. **Q&A** (10 min)
   - Common questions
   - Team-specific concerns

## Communication Templates

### Announcement: Adopting govman

```
Subject: New Tool: govman for Go Version Management

Hi Team,

We're adopting govman to manage Go versions across our projects.

Benefits:
- Automatic version switching per project
- No version conflicts between projects
- Same Go version in dev and CI/CD

Action Required:
1. Install govman: [link to docs]
2. Run setup for each project: [link to guide]
3. Deadline: [date]

Questions? Check our team docs or ask in #dev-help.

Thanks!
```

### Announcement: Go Version Upgrade

```
Subject: Upgrading Project X to Go 1.26.0

Hi Team,

We're upgrading Project X to Go 1.26.0 for [reasons].

What You Need To Do:
1. Pull latest main branch
2. Run: govman install 1.26.0
3. Verify: go version (in project directory)
4. Run tests: go test ./...

The version will auto-switch when you cd into the project.

Rollback Plan:
If issues arise, we can revert .govman-goversion to 1.25.1.

Questions? Reply here or in #project-x.
```

## Metrics and Success Criteria

Track onboarding success:

- [ ] 100% of team has govman installed
- [ ] All developers can build projects locally
- [ ] CI/CD uses same Go versions as development
- [ ] No "version mismatch" build failures
- [ ] New hires onboarded in < 30 minutes

## Support Channels

Set up team support:

1. **Documentation**: Internal wiki or README
2. **Slack/Discord**: #govman-help channel
3. **Office Hours**: Weekly Q&A sessions
4. **Champions**: Designate govman experts on team

## Advanced: Custom Team Workflows

### Pre-commit Hook

Ensure `.govman-goversion` matches `go.mod`:

```bash
#!/bin/bash
# .git/hooks/pre-commit

GOVMAN_VERSION=$(cat .govman-goversion)
GOMOD_VERSION=$(grep "^go " go.mod | awk '{print $2}')

if [ "$GOVMAN_VERSION" != "$GOMOD_VERSION" ]; then
  echo "Error: .govman-goversion ($GOVMAN_VERSION) doesn't match go.mod ($GOMOD_VERSION)"
  exit 1
fi
```

### Makefile Integration

```makefile
.PHONY: setup
setup:
	@echo "Installing Go version..."
	@govman install $$(cat .govman-goversion)
	@govman use $$(cat .govman-goversion)
	@echo "Downloading dependencies..."
	@go mod download
	@echo "Setup complete!"

.PHONY: verify
verify:
	@echo "Verifying Go version..."
	@go version
	@echo "Expected: $$(cat .govman-goversion)"
```

Usage:

```bash
make setup    # Set up development environment
make verify   # Verify correct Go version
```