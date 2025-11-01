# Shell Integration

Complete guide to setting up **govman** shell integration for automatic Go version switching.

## Overview

govman's shell integration provides:

- ✅ **Automatic PATH management** - Go binaries available instantly
- ✅ **Auto-switching** - Change Go versions when entering project directories
- ✅ **Seamless activation** - Versions activate automatically with `.govman-version` files
- ✅ **Non-intrusive** - Minimal impact on shell startup time
- ✅ **Easy removal** - Clean uninstall when needed

## Quick Setup

### Automatic Initialization

Run the initialization command for your shell:

```bash
govman init
```

This will:
1. Detect your current shell
2. Add govman configuration to your shell's RC file
3. Set up PATH and environment variables
4. Enable auto-switching functionality

Then restart your terminal or reload your shell configuration.

## Supported Shells

| Shell | OS Support | Auto-Switch | Status |
|-------|-----------|-------------|--------|
| **Bash** | macOS, Linux, Windows (Git Bash) | ✅ Yes | Fully Supported |
| **Zsh** | macOS, Linux | ✅ Yes | Fully Supported |
| **Fish** | macOS, Linux | ✅ Yes | Fully Supported |
| **PowerShell** | Windows, macOS, Linux | ✅ Yes | Fully Supported |
| **Command Prompt** | Windows | ❌ No | Limited Support |

## Shell-Specific Instructions

### Bash

#### Auto-Initialize

```bash
govman init
source ~/.bashrc
```

#### Manual Setup

Add to `~/.bashrc`:

```bash
# GOVMAN - Go Version Manager
export PATH="$HOME/.govman/bin:$PATH"
export GOTOOLCHAIN=local

# Ensure GOBIN and GOPATH/bin are available
if [ -n "$GOBIN" ]; then export PATH="$GOBIN:$PATH"; fi
if command -v go >/dev/null 2>&1; then export PATH="$(go env GOPATH)/bin:$PATH"; fi
export PATH="$HOME/go/bin:$PATH"

# Wrapper function for automatic PATH execution
govman() {
    local govman_bin="$HOME/.govman/bin/govman"
    if [[ "$1" == "use" && "$#" -ge 2 && "$2" != "--help" && "$2" != "-h" ]]; then
        local output
        output="$("$govman_bin" "$@" 2>&1)"
        local exit_code=$?
        if [[ $exit_code -eq 0 ]]; then
            local export_cmd=$(echo "$output" | grep -E '^export PATH=')
            if [[ -n "$export_cmd" ]]; then
                eval "$export_cmd"
                echo "✓ Go version switched successfully"
                return 0
            fi
        else
            echo "$output" >&2
            return $exit_code
        fi
    fi
    "$govman_bin" "$@"
}

# Auto-switch Go versions based on .govman-version file
govman_auto_switch() {
    if [[ -f .govman-version ]]; then
        local required_version=$(cat .govman-version 2>/dev/null | tr -d '\n\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        if [[ -n "$required_version" ]]; then
            if ! command -v go >/dev/null 2>&1; then
                echo "Go not found. Switching to Go $required_version..."
                govman use "$required_version" >/dev/null 2>&1
                return
            fi
            
            local current_version=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')
            if [[ "$current_version" != "$required_version" ]]; then
                echo "Auto-switching to Go $required_version (required by .govman-version)"
                govman use "$required_version" >/dev/null 2>&1
            fi
        fi
    fi
}

# Hook into PROMPT_COMMAND for directory changes
__govman_prev_pwd="$PWD"
__govman_check_dir_change() {
    if [[ "$PWD" != "$__govman_prev_pwd" ]]; then
        __govman_prev_pwd="$PWD"
        govman_auto_switch
    fi
}

if [[ -z "$PROMPT_COMMAND" ]]; then
    PROMPT_COMMAND="__govman_check_dir_change"
else
    PROMPT_COMMAND="__govman_check_dir_change;$PROMPT_COMMAND"
fi

# Run auto-switch on shell startup
govman_auto_switch
# END GOVMAN
```

Then reload:
```bash
source ~/.bashrc
```

### Zsh

#### Auto-Initialize

```bash
govman init
source ~/.zshrc
```

#### Manual Setup

Add to `~/.zshrc`:

```bash
# GOVMAN - Go Version Manager
export PATH="$HOME/.govman/bin:$PATH"
export GOTOOLCHAIN=local

# Ensure GOBIN and GOPATH/bin are available
if [ -n "$GOBIN" ]; then export PATH="$GOBIN:$PATH"; fi
if command -v go >/dev/null 2>&1; then export PATH="$(go env GOPATH)/bin:$PATH"; fi
export PATH="$HOME/go/bin:$PATH"

# Wrapper function for automatic PATH execution
govman() {
    local govman_bin="$HOME/.govman/bin/govman"
    if [[ "$1" == "use" && "$#" -ge 2 && "$2" != "--help" && "$2" != "-h" ]]; then
        local output
        output="$("$govman_bin" "$@" 2>&1)"
        local exit_code=$?
        if [[ $exit_code -eq 0 ]]; then
            local export_cmd=$(echo "$output" | grep -E '^export PATH=')
            if [[ -n "$export_cmd" ]]; then
                eval "$export_cmd"
                echo "✓ Go version switched successfully"
                return 0
            fi
        else
            echo "$output" >&2
            return $exit_code
        fi
    fi
    "$govman_bin" "$@"
}

# Auto-switch Go versions based on .govman-version file
govman_auto_switch() {
    if [[ -f .govman-version ]]; then
        local required_version=$(cat .govman-version 2>/dev/null | tr -d '\n\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        if [[ -n "$required_version" ]]; then
            if ! command -v go >/dev/null 2>&1; then
                echo "Go not found. Switching to Go $required_version..."
                govman use "$required_version" >/dev/null 2>&1
                return
            fi
            
            local current_version=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')
            if [[ "$current_version" != "$required_version" ]]; then
                echo "Auto-switching to Go $required_version (required by .govman-version)"
                govman use "$required_version" >/dev/null 2>&1
            fi
        fi
    fi
}

# Zsh-specific: Hook into chpwd for directory changes
autoload -U add-zsh-hook
add-zsh-hook chpwd govman_auto_switch

# Run auto-switch on shell startup
govman_auto_switch
# END GOVMAN
```

Then reload:
```bash
source ~/.zshrc
```

### Fish

#### Auto-Initialize

```bash
govman init
source ~/.config/fish/config.fish
```

#### Manual Setup

Add to `~/.config/fish/config.fish`:

```fish
# GOVMAN - Go Version Manager
fish_add_path -p "$HOME/.govman/bin"
set -gx GOTOOLCHAIN local

# Ensure GOBIN and GOPATH/bin are available
if test -n "$GOBIN"; and test -d "$GOBIN"; fish_add_path -p "$GOBIN"; end
if type -q go; set -l gopath (go env GOPATH 2>/dev/null); if test -n "$gopath"; and test -d "$gopath/bin"; fish_add_path -p "$gopath/bin"; end; end
set -l homegobin "$HOME/go/bin"; if test -d "$homegobin"; fish_add_path -p "$homegobin"; end

# Wrapper function for automatic PATH execution
function govman
    set govman_bin "$HOME/.govman/bin/govman"
    if test "$argv[1]" = "use"; and test (count $argv) -ge 2; and test "$argv[2]" != "--help"; and test "$argv[2]" != "-h"
        set output ($govman_bin $argv 2>&1)
        set exit_code $status
        if test $exit_code -eq 0
            for line in $output
                if string match -qr '^fish_add_path' -- $line
                    eval $line
                    echo "✓ Go version switched successfully"
                    return 0
                end
            end
        else
            for line in $output
                echo $line >&2
            end
            return $exit_code
        end
    end
    $govman_bin $argv
end

# Auto-switch Go versions based on .govman-version file
function govman_auto_switch
    if test -f .govman-version
        set required_version (string trim < .govman-version)
        if test -n "$required_version"
            if not command -v go >/dev/null 2>&1
                echo "Go not found. Switching to Go $required_version..."
                govman use "$required_version" >/dev/null 2>&1
                return
            end
            
            set current_version (go version 2>/dev/null | awk '{print $3}' | sed 's/go//')
            if test "$current_version" != "$required_version"
                echo "Auto-switching to Go $required_version (required by .govman-version)"
                govman use "$required_version" >/dev/null 2>&1
            end
        end
    end
end

# Fish-specific: Hook into directory changes
function __govman_cd_hook --on-variable PWD
    govman_auto_switch
end

# Run auto-switch on shell startup
govman_auto_switch
# END GOVMAN
```

Then reload:
```fish
source ~/.config/fish/config.fish
```

### PowerShell

#### Auto-Initialize

```powershell
govman init
. $PROFILE
```

#### Manual Setup

Add to your PowerShell profile (`$PROFILE`):

```powershell
# GOVMAN - Go Version Manager
$env:PATH = "$env:USERPROFILE\.govman\bin;" + $env:PATH
$env:GOTOOLCHAIN = 'local'

# Ensure GOPATH\bin and GOBIN are available
if ($env:GOBIN) { $env:PATH = "$env:GOBIN;" + $env:PATH }
$goCmd = Get-Command go -ErrorAction SilentlyContinue
if ($goCmd) { 
    $gopath = (& go env GOPATH 2>$null)
    if ($gopath) { $env:PATH = "$gopath\bin;" + $env:PATH }
}
$homeGoBin = Join-Path $env:USERPROFILE "go\bin"
if (Test-Path $homeGoBin) { $env:PATH = "$homeGoBin;" + $env:PATH }

# Wrapper function for automatic PATH execution
function govman {
    $govman_bin = "$env:USERPROFILE\.govman\bin\govman.exe"
    if ($args.Count -ge 2 -and $args[0] -eq 'use' -and $args[1] -ne '--help' -and $args[1] -ne '-h') {
        try {
            $output = & $govman_bin @args 2>&1
            if ($LASTEXITCODE -eq 0) {
                $pathCmd = $output | Where-Object { $_ -match '^\$env:PATH = ' }
                if ($pathCmd) {
                    Invoke-Expression $pathCmd
                    Write-Host '✓ Go version switched successfully' -ForegroundColor Green
                    return
                }
            } else {
                $output | ForEach-Object { Write-Error $_ }
                return
            }
        } catch {
            Write-Error $_.Exception.Message
            return
        }
    }
    & $govman_bin @args
}

# Auto-switch Go versions based on .govman-version file
function Invoke-GovmanAutoSwitch {
    if (Test-Path .govman-version) {
        try {
            $requiredVersion = (Get-Content .govman-version -Raw -ErrorAction Stop).Trim()
        } catch {
            return
        }

        if ($requiredVersion) {
            $currentVersion = $null
            try {
                $goVersionOutput = go version 2>$null
                if ($LASTEXITCODE -eq 0 -and $goVersionOutput) {
                    if ($goVersionOutput -match 'go version go([\d\.]+)') {
                        $currentVersion = $matches[1]
                    }
                }
            } catch {}

            if (-not $currentVersion) {
                Write-Host "Go not found. Switching to Go $requiredVersion..." -ForegroundColor Yellow
                govman use $requiredVersion *>$null
                return
            }

            if ($currentVersion -ne $requiredVersion) {
                Write-Host "Auto-switching to Go $requiredVersion (required by .govman-version)" -ForegroundColor Yellow
                govman use $requiredVersion *>$null
            }
        }
    }
}

# PowerShell-specific: Hook into location changes
$Global:GovmanPreviousLocation = $PWD.Path

function Global:Invoke-GovmanLocationCheck {
    if ($PWD.Path -ne $Global:GovmanPreviousLocation) {
        $Global:GovmanPreviousLocation = $PWD.Path
        Invoke-GovmanAutoSwitch
    }
}

# Hook into prompt for auto-switching
if (Get-Command prompt -ErrorAction SilentlyContinue) {
    $Global:GovmanOriginalPrompt = $function:prompt
    function global:prompt {
        Invoke-GovmanLocationCheck
        if ($Global:GovmanOriginalPrompt) {
            & $Global:GovmanOriginalPrompt
        } else {
            "PS $($executionContext.SessionState.Path.CurrentLocation)$('>' * ($nestedPromptLevel + 1)) "
        }
    }
}

# Run auto-switch on shell startup
Invoke-GovmanAutoSwitch
# END GOVMAN
```

Then reload:
```powershell
. $PROFILE
```

### Command Prompt (Windows)

Command Prompt has limited auto-switching capabilities. Use PowerShell for better experience.

#### Basic Setup

After running `govman init`, the PATH is configured. Manual version switching is required:

```cmd
govman use 1.21.5
```

**Limitations:**
- ❌ No automatic version switching
- ❌ No directory change hooks
- ✅ Manual switching works

**Recommendation:** Use PowerShell instead for full functionality.

## Features

### Auto-Switching

When enabled, govman automatically switches Go versions when you enter a directory containing a `.govman-version` file.

#### Enable/Disable

In `~/.govman/config.yaml`:

```yaml
auto_switch:
  enabled: true  # Set to false to disable
  project_file: ".govman-version"
```

#### How It Works

1. You enter a directory: `cd ~/my-project`
2. govman checks for `.govman-version`
3. If found, it reads the version number
4. If different from current, it switches automatically
5. You see: `Auto-switching to Go 1.21.5 (required by .govman-version)`

#### Example

```bash
# Project A
cd ~/projects/project-a
cat .govman-version  # Shows: 1.21.5
# Output: Auto-switching to Go 1.21.5 (required by .govman-version)

# Project B
cd ~/projects/project-b
cat .govman-version  # Shows: 1.20.12
# Output: Auto-switching to Go 1.20.12 (required by .govman-version)

# Verify
go version  # Shows: go version go1.20.12 ...
```

### PATH Management

govman automatically manages multiple PATH entries:

1. **govman bin**: `~/.govman/bin` - The govman binary
2. **Active Go**: `~/.govman/versions/go1.21.5/bin` - Current Go version
3. **GOBIN**: `$GOBIN` - User-installed Go tools
4. **GOPATH**: `$GOPATH/bin` - Project-installed tools
5. **Default Go bin**: `~/go/bin` - Default Go workspace

### Environment Variables

govman sets/manages:

- `PATH` - Includes govman and active Go version
- `GOTOOLCHAIN` - Set to `local` (prevents automatic downloads)
- `GOBIN` - If set, added to PATH
- `GOPATH` - Binary path added to PATH

## Advanced Configuration

### Custom Project File Name

Use a different filename instead of `.govman-version`:

```yaml
auto_switch:
  enabled: true
  project_file: ".govman-version"  # Default name
```

### Multiple Shell Support

If you use multiple shells, initialize each:

```bash
# Bash
govman init --shell bash

# Zsh
govman init --shell zsh

# Fish
govman init --shell fish
```

### Force Re-initialization

Overwrite existing configuration:

```bash
govman init --force
```

## Troubleshooting

### Auto-Switch Not Working

**Check if enabled:**
```bash
cat ~/.govman/config.yaml | grep -A 2 auto_switch
```

**Verify .govman-version exists:**
```bash
cat .govman-version
```

**Test manually:**
```bash
govman refresh
```

### Wrong Version After cd

**Check file content:**
```bash
cat .govman-version
# Should contain just: 1.21.5
```

**Verify version is installed:**
```bash
govman list | grep 1.21.5
```

**Install if missing:**
```bash
govman install 1.21.5
```

### Shell Integration Not Loaded

**Verify configuration exists:**
```bash
grep -A 5 "GOVMAN" ~/.bashrc  # Or ~/.zshrc, etc.
```

**Reload shell:**
```bash
source ~/.bashrc  # Or appropriate RC file
```

**Re-initialize:**
```bash
govman init --force
```

### PATH Not Updated

**Check PATH:**
```bash
echo $PATH | tr ':' '\n' | grep govman
```

**Verify wrapper function:**
```bash
type govman  # Should show it's a function
```

**Manual reload:**
```bash
eval "$(govman use 1.21.5)"
```

## Uninstalling Shell Integration

### Remove Configuration

Edit your shell RC file and remove the section between:
```bash
# GOVMAN - Go Version Manager
...
# END GOVMAN
```

**Files to check:**
- Bash: `~/.bashrc` or `~/.bash_profile`
- Zsh: `~/.zshrc`
- Fish: `~/.config/fish/config.fish`
- PowerShell: `$PROFILE`

### Reload Shell

```bash
source ~/.bashrc  # Or appropriate RC file
```

### Clean Removal Script

Use the uninstall script:

```bash
curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/uninstall.sh | bash
```

## Best Practices

### 1. Commit .govman-version

Always commit `.govman-version` to ensure team consistency:

```bash
git add .govman-version
git commit -m "Pin Go version to 1.21.5"
```

### 2. Use Auto-Switch in Development

Keep auto-switch enabled for seamless development:

```yaml
auto_switch:
  enabled: true
```

### 3. Disable in CI/CD

For reproducible builds, disable auto-switch and use explicit versions:

```yaml
auto_switch:
  enabled: false
```

```bash
# In CI script
govman use 1.21.5 --default
```

### 4. Test Shell Integration

After setup, test in a new terminal:

```bash
# Create test project
mkdir -p /tmp/test-govman
cd /tmp/test-govman
echo "1.21.5" > .govman-version

# Should auto-switch
cd /tmp/test-govman
go version  # Should show 1.21.5
```

## See Also

- [Quick Start](quick-start.md) - Get started with govman
- [Configuration](configuration.md) - Configure govman behavior
- [Commands](commands.md) - All available commands
- [Troubleshooting](troubleshooting.md) - Common issues and solutions

---

Enjoy seamless Go version management! 🚀
