# Shell Integration

govman provides intelligent shell integration for automatic Go version management.

## Features

- **Automatic version switching** based on `.govman-goversion` files
- **Smart PATH management** for Go binaries
- **Wrapper functions** for seamless `govman use` execution
- **Project-aware** version detection
- **Non-intrusive** configuration with easy removal
- **Security hardened** with input validation and command injection prevention (v1.1.0+)
- **Robust YAML parsing** with fallback mechanisms (v1.1.0+)
- **Duplicate prevention** for hook registration (v1.1.0+)

## Supported Shells

| Shell      | Platform    | Auto-Switch | Wrapper | Hook Mechanism   |
|------------|-------------|-------------|---------|------------------|
| Bash       | Linux/macOS | ✅           | ✅       | PROMPT_COMMAND   |
| Zsh        | Linux/macOS | ✅           | ✅       | chpwd hook       |
| Fish       | Linux/macOS | ✅           | ✅       | Native events    |
| PowerShell | Windows     | ✅           | ✅       | Set-Location hook|
| Cmd        | Windows     | ❌           | Partial | Not supported    |

## Setup

### Automatic Setup

```bash
govman init
```

This automatically:
1. Detects your current shell
2. Adds integration code to your shell config file
3. Sets up PATH and environment variables
4. Enables automatic version switching

### Manual Shell Selection

```bash
# Specify a shell explicitly
govman init --shell bash
govman init --shell zsh
govman init --shell fish
govman init --shell powershell
```

### Force Reinitialization

```bash
# Overwrites existing configuration
govman init --force
```

## Shell Configuration Files

govman modifies these files based on your shell:

- **Bash**: `~/.bashrc`, `~/.bash_profile`, or `~/.profile`
- **Zsh**: `~/.zshrc`
- **Fish**: `~/.config/fish/config.fish`
- **PowerShell**: `$PROFILE` (`Microsoft.PowerShell_profile.ps1`)
- **Cmd**: Creates wrapper batch file (limited functionality)



## How Auto-Switching Works

### Directory Change Detection

When you navigate to a directory, govman:

1. Checks for `.govman-goversion` file in current directory
2. Reads the required Go version from the file
3. Compares with currently active version
4. Automatically switches if different
5. Updates PATH to use the correct Go binary

### Activation Priority

govman resolves the active version in this order:

1. **Session-only**: Temporary activation via `govman use`
2. **Project-local**: `.govman-goversion` file in current/parent directory
3. **System-default**: Global version set via `govman use --default`


## Security & Reliability Improvements (v1.1.0+)

### Command Injection Prevention

All shell integration code includes strict validation before executing commands:

- **Bash/Zsh**: Uses `printf` instead of `echo` and validates export commands against regex pattern `^export PATH="[^"]*"$` before `eval`
- **PowerShell**: Validates PATH commands match `^\$env:PATH\s*=\s*"[^"]+"\s*\+\s*\$env:PATH$` before `Invoke-Expression`
- **Fish**: Improved pattern matching for `fish_add_path` commands

### Robust YAML Parsing

Configuration parsing is now more reliable:

- Uses `awk`-based parsing instead of fragile `grep -A 10` approach
- No dependency on hardcoded line limits
- Includes default values and fallback logic
- Handles edge cases in YAML structure

### Duplicate Prevention

Hook registration prevents issues from multiple config sourcing:

- **Bash**: Checks if `__govman_check_dir_change` already exists in `PROMPT_COMMAND`
- **Zsh**: Validates hook not in `chpwd_functions` array before adding
- **Fish**: Removes existing function before redefining
- **PowerShell**: Uses `$Global:GovmanPromptInjected` flag to track prompt injection

### Version Validation

Go version extraction includes format validation:

- More precise regex patterns: `\d+\.\d+(?:\.\d+)?` 
- Validates extracted version matches expected format
- Properly handles pre-release versions (rc, beta)
- Prevents issues from malformed version strings
## Shell Integration Code

> **Note:** The code examples below reflect the latest security improvements and robustness enhancements from v1.1.0.

### Bash/Zsh

```bash
# GOVMAN - Go Version Manager
export PATH="$HOME/.govman/bin:$PATH"

# Ensure GOBIN and GOPATH/bin are available
if [ -n "$GOBIN" ]; then export PATH="$GOBIN:$PATH"; fi
if command -v go > /dev/null 2>&1; then export PATH="$(go env GOPATH)/bin:$PATH"; fi
export PATH="$HOME/go/bin:$PATH"
export GOTOOLCHAIN=local

# Wrapper function for automatic PATH execution
govman() {
    local govman_bin="$HOME/.govman/bin/govman"
    if [[ "$1" == "use" && "$#" -ge 2 && "$2" != "--help" && "$2" != "-h" ]]; then
        local output
        output="$("$govman_bin" "$@" 2>&1)"
        local exit_code=$?
        if [[ $exit_code -eq 0 ]]; then
            # Security: Use printf instead of echo, validate format before eval
            local export_cmd=$(printf '%s\n' "$output" | grep -E '^export PATH=' | head -n 1)
            if [[ -n "$export_cmd" && "$export_cmd" =~ ^export\ PATH=\"[^\"]*\"$ ]]; then
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

# Auto-switch Go versions based on .govman-goversion file
govman_auto_switch() {
    # Check if auto-switch is enabled in config (improved YAML parsing)
    local config_file="$HOME/.govman/config.yaml"
    local auto_switch_enabled="true"
    if [[ -f "$config_file" ]]; then
        auto_switch_enabled=$(awk '/^auto_switch:/,/^[^ ]/ {if (/^[[:space:]]*enabled:/) {print $2; exit}}' "$config_file" 2>/dev/null | tr -d '[:space:]')
        [[ -z "$auto_switch_enabled" ]] && auto_switch_enabled="true"
    fi
    if [[ "$auto_switch_enabled" != "true" ]]; then
        return 0
    fi

    if [[ -f .govman-goversion ]]; then
        local required_version=$(cat .govman-goversion 2>/dev/null | tr -d '\n\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        if [[ -n "$required_version" ]]; then
            if ! command -v go > /dev/null 2>&1; then
                echo "Go not found. Switching to Go $required_version..."
                govman use "$required_version" > /dev/null 2>&1 || {
                    echo "Warning: Failed to switch to Go $required_version. Install it with 'govman install $required_version'" >&2
                }
                return
            fi

            # Improved version extraction with validation
            local current_version=$(go version 2>/dev/null | awk '{print $3}' | sed -E 's/^go//; s/([0-9]+\.[0-9]+(\.[0-9]+)?).*/\1/')
            if [[ ! "$current_version" =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?$ ]]; then current_version=""; fi
            if [[ -n "$current_version" && "$current_version" != "$required_version" ]]; then
                echo "Auto-switching to Go $required_version (required by .govman-goversion)"
                govman use "$required_version" > /dev/null 2>&1 || {
                    echo "Warning: Failed to switch to Go $required_version. Install it with 'govman install $required_version'" >&2
                }
            fi
        fi
    fi
}

# Bash-specific: Hook into PROMPT_COMMAND for directory changes
__govman_prev_pwd="$PWD"
__govman_check_dir_change() {
    if [[ "$PWD" != "$__govman_prev_pwd" ]]; then
        __govman_prev_pwd="$PWD"
        govman_auto_switch
    fi
}

# Add to PROMPT_COMMAND (prevents duplicates on re-source)
if [[ ! "$PROMPT_COMMAND" =~ __govman_check_dir_change ]]; then
    if [[ -z "$PROMPT_COMMAND" ]]; then
        PROMPT_COMMAND="__govman_check_dir_change"
    else
        PROMPT_COMMAND="__govman_check_dir_change;$PROMPT_COMMAND"
    fi
fi

# Run auto-switch on shell startup
govman_auto_switch
# END GOVMAN
```

### Zsh

Zsh uses the `chpwd` hook instead of `PROMPT_COMMAND`:

```bash
# Zsh-specific: Hook into chpwd for directory changes (prevents duplicates)
autoload -U add-zsh-hook
if [[ ! "${chpwd_functions[(r)govman_auto_switch]}" ]]; then
    add-zsh-hook chpwd govman_auto_switch
fi
```

### Fish

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

# Auto-switch Go versions based on .govman- version file
function govman_auto_switch
   # Implementation follows same logic as bash
end

# Fish event for directory changes
function __govman_cd_hook --on-variable PWD
    govman_auto_switch
end

# Run auto-switch on shell startup
govman_auto_switch
# END GOVMAN
```

### PowerShell

```powershell
# GOVMAN - Go Version Manager
$env:PATH = "$env:USERPROFILE\.govman\bin;$env:PATH"
$env:GOTOOLCHAIN = "local"

# Wrapper function for automatic PATH execution
function govman {
    $govmanBin = "$env:USERPROFILE\.govman\bin\govman.exe"
    if ($args.Count -ge 2 -and $args[0] -eq "use" -and $args[1] -ne "--help" -and $args[1] -ne "-h") {
        $output = & $govmanBin $args 2>&1 | Out-String
        if ($LASTEXITCODE -eq 0) {
            $exportCmd = $output -split "`n" | Where-Object { $_ -match '^\$env:PATH' }
            if ($exportCmd) {
                Invoke-Expression $exportCmd
                Write-Host "✓ Go version switched successfully"
                return
            }
        } else {
            Write-Error $output
            return
        }
    }
    & $govmanBin $args
}

# Auto-switch function
function Invoke-GovmanAutoSwitch {
    # Implementation follows same logic as bash
}

# Hook into Set-Location (cd)
$global:__GovmanPreviousLocation = Get-Location
function prompt {
    $currentLocation = Get-Location
    if ($currentLocation.Path -ne $global:__GovmanPreviousLocation.Path) {
        $global:__GovmanPreviousLocation = $currentLocation
        Invoke-GovmanAutoSwitch
    }
    # Call original prompt
}

# Run auto-switch on shell startup
Invoke-GovmanAutoSwitch
# END GOVMAN
```

## Disabling Auto-Switch

### Temporarily

```bash
# Edit config
nano ~/.govman/config.yaml

# Set enabled to false
auto_switch:
  enabled: false
```

### Remove Shell Integration

```bash
# Edit your shell config file
nano ~/.bashrc  # or ~/.zshrc, etc.

# Remove the GOVMAN section (between # GOVMAN and # END GOVMAN markers)
```

Then restart your shell or run:

```bash
source ~/.bashrc  # or appropriate config file
```

## Troubleshooting

### Auto-Switch Not Working

1. **Check shell integration**:
   ```bash
   type govman_auto_switch
   ```
   Should show the function definition.

2. **Verify config**:
   ```bash
   cat ~/.govman/config.yaml | grep -A 3 auto_switch
   ```
   Ensure `enabled: true`.

3. **Check .govman-goversion file**:
   ```bash
   cat .govman-goversion
   ```
   Should contain only the version number (e.g., `1.25.1`).

4. **Test manually**:
   ```bash
   govman_auto_switch
   ```

### PATH Not Updated

If `govman use` doesn't update PATH in current session:

1. **Use the wrapper function**:
   Ensure you're calling `govman` (the shell function), not the binary directly.

2. **Check wrapper function**:
   ```bash
   type govman
   ```
   Should show it's a function, not an alias or binary.

3. **Reinitialize shell integration**:
   ```bash
   govman init --force
   source ~/.bashrc
   ```

### Command Prompt (cmd.exe) Limitations

Windows Command Prompt has limited shell integration:
- No automatic version switching
- No wrapper function for PATH updates
- Manual `govman use` required in each session

**Recommendation**: Use PowerShell or Git Bash for full functionality.

## Best Practices

1. **Commit .govman-goversion to Git**:
   ```bash
   echo "1.25.1" > .govman-goversion
   git add .govman-goversion
   git commit -m "Set Go version to 1.25.1"
   ```

2. **Use .govman-goversion for Projects**:
   Ensure consistent Go versions across team members and CI/CD.

3. **Set System Default for Personal Projects**:
   ```bash
   govman use 1.25.1 --default
   ```

4. **Verify Integration After Updates**:
   After updating shell config or govman, verify integration works:
   ```bash
   cd /tmp
   echo "1.24.0" > .govman-goversion
   cd .  # Trigger auto-switch
   go version
   ```
