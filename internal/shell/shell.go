package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"
)

var (
	currentGOOS        = runtime.GOOS
	execLookPath       = exec.LookPath
	userHomeDir        = os.UserHomeDir
	newlineRegex       = regexp.MustCompile(`\n{3,}`)
	configRemovalRegex = regexp.MustCompile(`(?ms)^[#\s]*(REM\s+)?GOVMAN - Go Version Manager.*?^[#\s]*(REM\s+)?END GOVMAN.*?$\n?`)
)

type Shell interface {
	Name() string
	DisplayName() string
	ConfigFile() string
	PathCommand(path string) string
	SetupCommands(binPath string) []string
	IsAvailable() bool
	ExecutePathCommand(path string) error
}

type BashShell struct{}
type ZshShell struct{}
type FishShell struct{}
type PowerShell struct{}
type CmdShell struct{}

// validateBinPath ensures the binary path is safe and exists
func validateBinPath(binPath string) error {
	if binPath == "" {
		return fmt.Errorf("binary path cannot be empty")
	}

	// Check for path traversal indicators
	if strings.Contains(binPath, "..") {
		return fmt.Errorf("invalid binary path (path traversal detected): %s", binPath)
	}

	// Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(binPath)

	// Convert to absolute path for comparison
	absPath, err := filepath.Abs(binPath)
	if err != nil {
		return fmt.Errorf("unable to resolve absolute path: %w", err)
	}

	absCleanPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("unable to resolve clean absolute path: %w", err)
	}

	// Ensure no path traversal
	if absPath != absCleanPath {
		return fmt.Errorf("invalid binary path (path traversal detected): %s", binPath)
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("binary path does not exist: %s", absPath)
		}
		return fmt.Errorf("unable to access binary path: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("binary path is not a directory: %s", absPath)
	}

	return nil
}

// escapeBashPath properly escapes a path for use in bash/zsh
func escapeBashPath(path string) string {
	// Escape special characters for bash/zsh
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`$`, `\$`,
		"`", "\\`",
		`!`, `\!`,
	)
	return replacer.Replace(path)
}

// escapeFishPath properly escapes a path for use in fish
func escapeFishPath(path string) string {
	// Fish uses different escaping rules - escape backslash, quotes, and dollar signs
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`$`, `\$`,
		`'`, `\'`,
	)
	return replacer.Replace(path)
}

// escapePowerShellPath properly escapes a path for use in PowerShell
func escapePowerShellPath(path string) string {
	// PowerShell escaping: backtick is the escape character
	// Order matters: escape backtick first
	replacer := strings.NewReplacer(
		"`", "``",
		`"`, "`\"",
		`$`, "`$",
	)
	return replacer.Replace(path)
}

// escapeCmdPath properly escapes a path for use in cmd
func escapeCmdPath(path string) string {
	// CMD uses % for variables
	return strings.ReplaceAll(path, "%", "%%")
}

// Detect determines the user's shell based on OS and environment variables,
// falling back to an available default when detection is inconclusive.
func Detect() Shell {
	if currentGOOS == "windows" {
		// Check for PowerShell Core first (preferred)
		if isCommandAvailable("pwsh") {
			return &PowerShell{}
		}
		if isCommandAvailable("powershell") {
			return &PowerShell{}
		}

		// Fallback to Command Prompt
		return &CmdShell{}
	}

	// For Unix-like systems, check SHELL environment variable
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return detectAvailableShell()
	}

	shellName := filepath.Base(shellPath)
	switch shellName {
	case "zsh":
		if isCommandAvailable("zsh") {
			return &ZshShell{}
		}
	case "fish":
		if isCommandAvailable("fish") {
			return &FishShell{}
		}
	case "bash", "sh":
		if isCommandAvailable("bash") {
			return &BashShell{}
		}
	}

	// If the detected shell isn't available, find an alternative
	return detectAvailableShell()
}

// DetectAll returns a slice of supported shells that are available on the current system.
func DetectAll() []Shell {
	var shells []Shell

	if currentGOOS == "windows" {
		// Windows-specific shells
		shells = []Shell{
			&PowerShell{},
			&CmdShell{},
		}
	} else {
		// Unix-like shells
		shells = []Shell{
			&ZshShell{},
			&BashShell{},
			&FishShell{},
		}
	}

	var available []Shell
	for _, shell := range shells {
		if shell.IsAvailable() {
			available = append(available, shell)
		}
	}

	return available
}

// detectAvailableShell returns the first available shell from a prioritized list.
func detectAvailableShell() Shell {
	shells := []Shell{
		&BashShell{},
		&ZshShell{},
		&FishShell{},
	}

	for _, shell := range shells {
		if shell.IsAvailable() {
			return shell
		}
	}

	return &BashShell{}
}

// isCommandAvailable reports whether a command exists in the system PATH.
func isCommandAvailable(command string) bool {
	_, err := execLookPath(command)
	return err == nil
}

// fileExists checks if a file exists and is not a directory.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// Name returns the identifier for Bash.
func (s *BashShell) Name() string {
	return "bash"
}

// DisplayName returns the human-friendly name for Bash.
func (s *BashShell) DisplayName() string {
	return "Bash"
}

// IsAvailable reports whether Bash is present in the system PATH.
func (s *BashShell) IsAvailable() bool {
	return isCommandAvailable("bash")
}

// ConfigFile returns the path to the Bash configuration file.
func (s *BashShell) ConfigFile() string {
	home, err := userHomeDir()
	if err != nil {
		return ".bashrc" // Fallback to relative path
	}

	candidates := []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".profile"),
	}

	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate
		}
	}

	// Default to .bashrc if none exist
	return filepath.Join(home, ".bashrc")
}

// PathCommand returns a Bash-compatible command to prepend binPath to PATH.
func (s *BashShell) PathCommand(path string) string {
	escapedPath := escapeBashPath(path)
	return fmt.Sprintf(`export PATH="%s:$PATH"`, escapedPath)
}

// SetupCommands returns the Bash shell configuration lines to integrate govman.
func (s *BashShell) SetupCommands(binPath string) []string {
	escapedPath := escapeBashPath(binPath)

	commands := []string{
		"# GOVMAN - Go Version Manager",
		fmt.Sprintf(`export PATH="%s:$PATH"`, escapedPath),
		"# Ensure GOBIN and GOPATH/bin are available",
		`if [ -n "$GOBIN" ]; then export PATH="$GOBIN:$PATH"; fi`,
		`if command -v go >/dev/null 2>&1; then export PATH="$(go env GOPATH)/bin:$PATH"; fi`,
		`export PATH="$HOME/go/bin:$PATH"`,
		"export GOTOOLCHAIN=local",
		"",
		"# Wrapper function for automatic PATH execution",
		"govman() {",
		fmt.Sprintf(`    local govman_bin="%s/govman"`, escapedPath),
		`    if [[ "$1" == "use" && "$#" -ge 2 && "$2" != "--help" && "$2" != "-h" ]]; then`,
		"        local output",
		`        output="$("$govman_bin" "$@" 2>&1)"`,
		"        local exit_code=$?",
		"        if [[ $exit_code -eq 0 ]]; then",
		`            local export_cmd=$(echo "$output" | grep -E '^export PATH=')`,
		`            if [[ -n "$export_cmd" ]]; then`,
		`                eval "$export_cmd"`,
		`                echo "✓ Go version switched successfully"`,
		"                return 0",
		"            fi",
		"        else",
		`            echo "$output" >&2`,
		"            return $exit_code",
		"        fi",
		"    fi",
		`    "$govman_bin" "$@"`,
		"}",
		"",
		"# Auto-switch Go versions based on .govman-version file",
		"govman_auto_switch() {",
		"    # Check if auto-switch is enabled in config",
		`    local config_file="$HOME/.govman/config.yaml"`,
		`    if [[ -f "$config_file" ]]; then`,
		`        local auto_switch_enabled=$(grep -E '^auto_switch:' -A 10 "$config_file" 2>/dev/null | grep -E '^[[:space:]]*enabled:' | head -1 | awk '{print $2}' | tr -d '[:space:]')`,
		`        if [[ "$auto_switch_enabled" != "true" ]]; then`,
		"            return 0",
		"        fi",
		"    fi",
		"",
		"    if [[ -f .govman-version ]]; then",
		`        local required_version=$(cat .govman-version 2>/dev/null | tr -d '\n\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')`,
		`        if [[ -n "$required_version" ]]; then`,
		"            if ! command -v go >/dev/null 2>&1; then",
		`                echo "Go not found. Switching to Go $required_version..."`,
		`                govman use "$required_version" >/dev/null 2>&1 || {`,
		`                    echo "Warning: Failed to switch to Go $required_version. Install it with 'govman install $required_version'" >&2`,
		"                }",
		"                return",
		"            fi",
		"",
		`            local current_version=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')`,
		`            if [[ "$current_version" != "$required_version" ]]; then`,
		`                echo "Auto-switching to Go $required_version (required by .govman-version)"`,
		`                govman use "$required_version" >/dev/null 2>&1 || {`,
		`                    echo "Warning: Failed to switch to Go $required_version. Install it with 'govman install $required_version'" >&2`,
		"                }",
		"            fi",
		"        fi",
		"    fi",
		"}",
		"",
		"# Bash-specific: Hook into PROMPT_COMMAND for directory changes",
		`__govman_prev_pwd="$PWD"`,
		"__govman_check_dir_change() {",
		`    if [[ "$PWD" != "$__govman_prev_pwd" ]]; then`,
		`        __govman_prev_pwd="$PWD"`,
		"        govman_auto_switch",
		"    fi",
		"}",
		"",
		"# Add to PROMPT_COMMAND (preserves existing commands)",
		`if [[ -z "$PROMPT_COMMAND" ]]; then`,
		`    PROMPT_COMMAND="__govman_check_dir_change"`,
		"else",
		`    PROMPT_COMMAND="__govman_check_dir_change;$PROMPT_COMMAND"`,
		"fi",
		"",
		"# Run auto-switch on shell startup",
		"govman_auto_switch",
		"# END GOVMAN",
	}

	return commands
}

// ExecutePathCommand outputs the PATH command for automatic execution via eval.
func (s *BashShell) ExecutePathCommand(path string) error {
	if err := validateBinPath(path); err != nil {
		return err
	}

	pathCmd := s.PathCommand(path)

	// Output the command for eval
	fmt.Println(pathCmd)

	// Instructions to stderr so they don't interfere with eval
	fmt.Fprintf(os.Stderr, "# To apply to current session, run:\n")
	fmt.Fprintf(os.Stderr, "# eval \"$(govman use <version>)\"\n")

	return nil
}

// Name returns the identifier for Zsh.
func (s *ZshShell) Name() string {
	return "zsh"
}

// DisplayName returns the human-friendly name for Zsh.
func (s *ZshShell) DisplayName() string {
	return "Zsh"
}

// IsAvailable reports whether Zsh is present in the system PATH.
func (s *ZshShell) IsAvailable() bool {
	return isCommandAvailable("zsh")
}

// ConfigFile returns the path to the Zsh configuration file.
func (s *ZshShell) ConfigFile() string {
	home, err := userHomeDir()
	if err != nil {
		return ".zshrc"
	}
	return filepath.Join(home, ".zshrc")
}

// PathCommand returns a Zsh-compatible command to prepend binPath to PATH.
func (s *ZshShell) PathCommand(path string) string {
	escapedPath := escapeBashPath(path)
	return fmt.Sprintf(`export PATH="%s:$PATH"`, escapedPath)
}

// SetupCommands returns the Zsh configuration lines to integrate govman.
func (s *ZshShell) SetupCommands(binPath string) []string {
	escapedPath := escapeBashPath(binPath)

	commands := []string{
		"# GOVMAN - Go Version Manager",
		fmt.Sprintf(`export PATH="%s:$PATH"`, escapedPath),
		"# Ensure GOBIN and GOPATH/bin are available",
		`if [ -n "$GOBIN" ]; then export PATH="$GOBIN:$PATH"; fi`,
		`if command -v go >/dev/null 2>&1; then export PATH="$(go env GOPATH)/bin:$PATH"; fi`,
		`export PATH="$HOME/go/bin:$PATH"`,
		"export GOTOOLCHAIN=local",
		"",
		"# Wrapper function for automatic PATH execution",
		"govman() {",
		fmt.Sprintf(`    local govman_bin="%s/govman"`, escapedPath),
		`    if [[ "$1" == "use" && "$#" -ge 2 && "$2" != "--help" && "$2" != "-h" ]]; then`,
		"        local output",
		`        output="$("$govman_bin" "$@" 2>&1)"`,
		"        local exit_code=$?",
		"        if [[ $exit_code -eq 0 ]]; then",
		`            local export_cmd=$(echo "$output" | grep -E '^export PATH=')`,
		`            if [[ -n "$export_cmd" ]]; then`,
		`                eval "$export_cmd"`,
		`                echo "✓ Go version switched successfully"`,
		"                return 0",
		"            fi",
		"        else",
		`            echo "$output" >&2`,
		"            return $exit_code",
		"        fi",
		"    fi",
		`    "$govman_bin" "$@"`,
		"}",
		"",
		"# Auto-switch Go versions based on .govman-version file",
		"govman_auto_switch() {",
		"    # Check if auto-switch is enabled in config",
		`    local config_file="$HOME/.govman/config.yaml"`,
		`    if [[ -f "$config_file" ]]; then`,
		`        local auto_switch_enabled=$(grep -E '^auto_switch:' -A 10 "$config_file" 2>/dev/null | grep -E '^[[:space:]]*enabled:' | head -1 | awk '{print $2}' | tr -d '[:space:]')`,
		`        if [[ "$auto_switch_enabled" != "true" ]]; then`,
		"            return 0",
		"        fi",
		"    fi",
		"",
		"    if [[ -f .govman-version ]]; then",
		`        local required_version=$(cat .govman-version 2>/dev/null | tr -d '\n\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')`,
		`        if [[ -n "$required_version" ]]; then`,
		"            if ! command -v go >/dev/null 2>&1; then",
		`                echo "Go not found. Switching to Go $required_version..."`,
		`                govman use "$required_version" >/dev/null 2>&1 || {`,
		`                    echo "Warning: Failed to switch to Go $required_version. Install it with 'govman install $required_version'" >&2`,
		"                }",
		"                return",
		"            fi",
		"",
		`            local current_version=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')`,
		`            if [[ "$current_version" != "$required_version" ]]; then`,
		`                echo "Auto-switching to Go $required_version (required by .govman-version)"`,
		`                govman use "$required_version" >/dev/null 2>&1 || {`,
		`                    echo "Warning: Failed to switch to Go $required_version. Install it with 'govman install $required_version'" >&2`,
		"                }",
		"            fi",
		"        fi",
		"    fi",
		"}",
		"",
		"# Zsh-specific: Hook into chpwd for directory changes",
		"autoload -U add-zsh-hook",
		"add-zsh-hook chpwd govman_auto_switch",
		"",
		"# Run auto-switch on shell startup",
		"govman_auto_switch",
		"# END GOVMAN",
	}

	return commands
}

// ExecutePathCommand outputs the PATH command for automatic execution via eval.
func (s *ZshShell) ExecutePathCommand(path string) error {
	if err := validateBinPath(path); err != nil {
		return err
	}

	pathCmd := s.PathCommand(path)
	fmt.Println(pathCmd)

	fmt.Fprintf(os.Stderr, "# To apply to current session, run:\n")
	fmt.Fprintf(os.Stderr, "# eval \"$(govman use <version>)\"\n")

	return nil
}

// Name returns the identifier for Fish.
func (s *FishShell) Name() string {
	return "fish"
}

// DisplayName returns the human-friendly name for Fish.
func (s *FishShell) DisplayName() string {
	return "Fish"
}

// IsAvailable reports whether Fish is present in the system PATH.
func (s *FishShell) IsAvailable() bool {
	return isCommandAvailable("fish")
}

// ConfigFile returns the path to the Fish configuration file.
func (s *FishShell) ConfigFile() string {
	home, err := userHomeDir()
	if err != nil {
		return "config.fish"
	}
	return filepath.Join(home, ".config", "fish", "config.fish")
}

// PathCommand returns a Fish-compatible command to prepend binPath to PATH.
func (s *FishShell) PathCommand(path string) string {
	escapedPath := escapeFishPath(path)
	return fmt.Sprintf(`fish_add_path -p "%s"`, escapedPath)
}

// SetupCommands returns the Fish configuration lines to integrate govman.
func (s *FishShell) SetupCommands(binPath string) []string {
	escapedPath := escapeFishPath(binPath)

	commands := []string{
		"# GOVMAN - Go Version Manager",
		fmt.Sprintf(`fish_add_path -p "%s"`, escapedPath),
		"set -gx GOTOOLCHAIN local",
		"",
		"# Ensure GOBIN and GOPATH/bin are available",
		`if test -n "$GOBIN"; and test -d "$GOBIN"; fish_add_path -p "$GOBIN"; end`,
		`if type -q go; set -l gopath (go env GOPATH 2>/dev/null); if test -n "$gopath"; and test -d "$gopath/bin"; fish_add_path -p "$gopath/bin"; end; end`,
		`set -l homegobin "$HOME/go/bin"; if test -d "$homegobin"; fish_add_path -p "$homegobin"; end`,
		"",
		"# Wrapper function for automatic PATH execution",
		"function govman",
		fmt.Sprintf(`    set govman_bin "%s/govman"`, escapedPath),
		`    if test "$argv[1]" = "use"; and test (count $argv) -ge 2; and test "$argv[2]" != "--help"; and test "$argv[2]" != "-h"`,
		"        set output ($govman_bin $argv 2>&1)",
		"        set exit_code $status",
		"        if test $exit_code -eq 0",
		"            for line in $output",
		"                if string match -qr '^fish_add_path' -- $line",
		"                    eval $line",
		`                    echo "✓ Go version switched successfully"`,
		"                    return 0",
		"                end",
		"            end",
		"        else",
		"            for line in $output",
		"                echo $line >&2",
		"            end",
		"            return $exit_code",
		"        end",
		"    end",
		"    $govman_bin $argv",
		"end",
		"",
		"# Auto-switch Go versions based on .govman-version file",
		"function govman_auto_switch",
		`    set config_file "$HOME/.govman/config.yaml"`,
		`    if test -f "$config_file"`,
		`        set auto_switch_enabled (grep -E '^auto_switch:' -A 10 "$config_file" 2>/dev/null | grep -E '^[[:space:]]*enabled:' | head -1 | awk '{print $2}' | tr -d '[:space:]')`,
		`        if test "$auto_switch_enabled" != "true"`,
		"            return 0",
		"        end",
		"    end",
		"",
		"    if test -f .govman-version",
		"        set required_version (string trim < .govman-version)",
		`        if test -n "$required_version"`,
		"            if not command -v go >/dev/null 2>&1",
		`                echo "Go not found. Switching to Go $required_version..."`,
		`                govman use "$required_version" >/dev/null 2>&1; or begin`,
		`                    echo "Warning: Failed to switch to Go $required_version. Install it with 'govman install $required_version'" >&2`,
		"                end",
		"                return",
		"            end",
		"",
		"            set current_version (go version 2>/dev/null | awk '{print $3}' | sed 's/go//')",
		`            if test "$current_version" != "$required_version"`,
		`                echo "Auto-switching to Go $required_version (required by .govman-version)"`,
		`                govman use "$required_version" >/dev/null 2>&1; or begin`,
		`                    echo "Warning: Failed to switch to Go $required_version. Install it with 'govman install $required_version'" >&2`,
		"                end",
		"            end",
		"        end",
		"    end",
		"end",
		"",
		"# Fish-specific: Hook into directory changes",
		"function __govman_cd_hook --on-variable PWD",
		"    govman_auto_switch",
		"end",
		"",
		"# Run auto-switch on shell startup",
		"govman_auto_switch",
		"# END GOVMAN",
	}

	return commands
}

// ExecutePathCommand outputs the PATH command for automatic execution via eval.
func (s *FishShell) ExecutePathCommand(path string) error {
	if err := validateBinPath(path); err != nil {
		return err
	}

	pathCmd := s.PathCommand(path)
	fmt.Println(pathCmd)

	fmt.Fprintf(os.Stderr, "# To apply to current session, run:\n")
	fmt.Fprintf(os.Stderr, "# eval (govman use <version>)\n")

	return nil
}

// Name returns the identifier for PowerShell.
func (s *PowerShell) Name() string {
	return "powershell"
}

// DisplayName returns the human-friendly name for PowerShell.
func (s *PowerShell) DisplayName() string {
	return "PowerShell"
}

// IsAvailable reports whether PowerShell is available.
func (s *PowerShell) IsAvailable() bool {
	return isCommandAvailable("pwsh") || isCommandAvailable("powershell")
}

// ConfigFile returns the PowerShell profile path.
func (s *PowerShell) ConfigFile() string {
	home, err := userHomeDir()
	if err != nil {
		return "$PROFILE"
	}

	// Check for PowerShell Core first
	if isCommandAvailable("pwsh") {
		profilePath := filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
		return profilePath
	}

	// Fall back to Windows PowerShell
	profilePath := filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
	return profilePath
}

// PathCommand returns a PowerShell command to prepend binPath to PATH.
func (s *PowerShell) PathCommand(path string) string {
	escapedPath := escapePowerShellPath(path)
	return fmt.Sprintf(`$env:PATH = "%s;" + $env:PATH`, escapedPath)
}

// SetupCommands returns the PowerShell profile lines to integrate govman.
func (s *PowerShell) SetupCommands(binPath string) []string {
	escapedPath := escapePowerShellPath(binPath)

	commands := []string{
		"# GOVMAN - Go Version Manager",
		fmt.Sprintf(`$env:PATH = "%s;" + $env:PATH`, escapedPath),
		"$env:GOTOOLCHAIN = 'local'",
		"",
		"# Ensure GOPATH\\bin and GOBIN are available",
		`if ($env:GOBIN) { $env:PATH = "$env:GOBIN;" + $env:PATH }`,
		`$goCmd = Get-Command go -ErrorAction SilentlyContinue; if ($goCmd) { $gopath = (& go env GOPATH 2>$null); if ($gopath) { $env:PATH = "$gopath\bin;" + $env:PATH } }`,
		`$homeGoBin = Join-Path $env:USERPROFILE "go\bin"; if (Test-Path $homeGoBin) { $env:PATH = "$homeGoBin;" + $env:PATH }`,
		"",
		"# Wrapper function for automatic PATH execution",
		"function govman {",
		fmt.Sprintf(`    $govman_bin = "%s\govman.exe"`, escapedPath),
		"    if ($args.Count -ge 2 -and $args[0] -eq 'use' -and $args[1] -ne '--help' -and $args[1] -ne '-h') {",
		"        try {",
		"            $output = & $govman_bin @args 2>&1",
		"            if ($LASTEXITCODE -eq 0) {",
		"                $pathCmd = $output | Where-Object { $_ -match '^\\$env:PATH = ' }",
		"                if ($pathCmd) {",
		"                    Invoke-Expression $pathCmd",
		"                    Write-Host '✓ Go version switched successfully' -ForegroundColor Green",
		"                    return",
		"                }",
		"            } else {",
		"                $output | ForEach-Object { Write-Error $_ }",
		"                return",
		"            }",
		"        } catch {",
		"            Write-Error $_.Exception.Message",
		"            return",
		"        }",
		"    }",
		"    & $govman_bin @args",
		"}",
		"",
		"# Auto-switch Go versions based on .govman-version file",
		"function Invoke-GovmanAutoSwitch {",
		"    $configFile = \"$env:USERPROFILE\\.govman\\config.yaml\"",
		"    if (Test-Path $configFile) {",
		"        try {",
		"            $autoSwitchEnabled = $false",
		"            $content = Get-Content $configFile -Raw -ErrorAction Stop",
		"            if ($content -match '(?ms)auto_switch:.*?enabled:\\s*(true|false)') {",
		"                $autoSwitchEnabled = ($matches[1] -eq 'true')",
		"            }",
		"            if (-not $autoSwitchEnabled) {",
		"                return",
		"            }",
		"        } catch {",
		"            return",
		"        }",
		"    }",
		"",
		"    if (Test-Path .govman-version) {",
		"        try {",
		"            $requiredVersion = (Get-Content .govman-version -Raw -ErrorAction Stop).Trim()",
		"        } catch {",
		"            return",
		"        }",
		"",
		"        if ($requiredVersion) {",
		"            $currentVersion = $null",
		"            try {",
		"                $goVersionOutput = go version 2>$null",
		"                if ($LASTEXITCODE -eq 0 -and $goVersionOutput) {",
		"                    if ($goVersionOutput -match 'go version go([\\d\\.]+)') {",
		"                        $currentVersion = $matches[1]",
		"                    }",
		"                }",
		"            } catch {}",
		"",
		"            if (-not $currentVersion) {",
		"                Write-Host \"Go not found. Switching to Go $requiredVersion...\" -ForegroundColor Yellow",
		"                govman use $requiredVersion *>$null",
		"                if ($LASTEXITCODE -ne 0) {",
		"                    Write-Warning \"Failed to switch to Go $requiredVersion. Install it with 'govman install $requiredVersion'\"",
		"                }",
		"                return",
		"            }",
		"",
		"            if ($currentVersion -ne $requiredVersion) {",
		"                Write-Host \"Auto-switching to Go $requiredVersion (required by .govman-version)\" -ForegroundColor Yellow",
		"                govman use $requiredVersion *>$null",
		"                if ($LASTEXITCODE -ne 0) {",
		"                    Write-Warning \"Failed to switch to Go $requiredVersion. Install it with 'govman install $requiredVersion'\"",
		"                }",
		"            }",
		"        }",
		"    }",
		"}",
		"",
		"# PowerShell-specific: Hook into location changes",
		"$Global:GovmanPreviousLocation = $PWD.Path",
		"",
		"function Global:Invoke-GovmanLocationCheck {",
		"    if ($PWD.Path -ne $Global:GovmanPreviousLocation) {",
		"        $Global:GovmanPreviousLocation = $PWD.Path",
		"        Invoke-GovmanAutoSwitch",
		"    }",
		"}",
		"",
		"# Hook into prompt for auto-switching",
		"if (Get-Command prompt -ErrorAction SilentlyContinue) {",
		"    $Global:GovmanOriginalPrompt = $function:prompt",
		"    function global:prompt {",
		"        Invoke-GovmanLocationCheck",
		"        if ($Global:GovmanOriginalPrompt) {",
		"            & $Global:GovmanOriginalPrompt",
		"        } else {",
		"            \"PS $($executionContext.SessionState.Path.CurrentLocation)$('>' * ($nestedPromptLevel + 1)) \"",
		"        }",
		"    }",
		"}",
		"",
		"# Run auto-switch on shell startup",
		"Invoke-GovmanAutoSwitch",
		"# END GOVMAN",
	}

	return commands
}

// ExecutePathCommand outputs the PATH command for automatic execution.
func (s *PowerShell) ExecutePathCommand(path string) error {
	if err := validateBinPath(path); err != nil {
		return err
	}

	pathCmd := s.PathCommand(path)
	fmt.Println(pathCmd)

	fmt.Fprintf(os.Stderr, "# To apply to current session, run:\n")
	fmt.Fprintf(os.Stderr, "# govman use <version> | Invoke-Expression\n")

	return nil
}

// Name returns the identifier for Windows Command Prompt.
func (s *CmdShell) Name() string {
	return "cmd"
}

// DisplayName returns the human-friendly name for Command Prompt.
func (s *CmdShell) DisplayName() string {
	return "Command Prompt"
}

// IsAvailable reports whether cmd is available (Windows only).
func (s *CmdShell) IsAvailable() bool {
	return currentGOOS == "windows"
}

// ConfigFile returns a description of where cmd configuration is managed.
func (s *CmdShell) ConfigFile() string {
	return "Environment Variables (System Properties)"
}

// PathCommand returns a cmd.exe command to prepend binPath to PATH.
func (s *CmdShell) PathCommand(path string) string {
	escapedPath := escapeCmdPath(path)
	return fmt.Sprintf(`set PATH=%s;%%PATH%%`, escapedPath)
}

// SetupCommands returns guidance for integrating govman with Command Prompt.
func (s *CmdShell) SetupCommands(binPath string) []string {
	escapedPath := escapeBashPath(binPath)

	commands := []string{
		"@echo off",
		"REM GOVMAN - Go Version Manager",
		fmt.Sprintf(`set "PATH=%s;%%PATH%%"`, escapedPath),
		"set GOTOOLCHAIN=local",
		"",
		"REM Ensure GOBIN and GOPATH\\bin are available",
		`if defined GOBIN set "PATH=%GOBIN%;%PATH%"`,
		"",
		"REM Check for go command and add GOPATH\\bin",
		`where go >nul 2>&1`,
		`if %errorlevel% equ 0 (`,
		`    for /f "delims=" %%i in ('go env GOPATH 2^>nul') do set "GOPATH_BIN=%%i\bin"`,
		`    if defined GOPATH_BIN if exist "%GOPATH_BIN%" set "PATH=%GOPATH_BIN%;%PATH%"`,
		`)`,
		"",
		"REM Add Go's default bin directory",
		`if exist "%USERPROFILE%\go\bin" set "PATH=%USERPROFILE%\go\bin;%PATH%"`,
		"",
		"REM Note: Auto-switching (.govman-version) is not available in Command Prompt",
		"REM Use 'govman use <version>' to switch versions manually",
		"",
		"REM END GOVMAN",
	}

	return commands
}

// ExecutePathCommand outputs the PATH command for Command Prompt.
func (s *CmdShell) ExecutePathCommand(path string) error {
	if err := validateBinPath(path); err != nil {
		return err
	}

	pathCmd := s.PathCommand(path)
	fmt.Println(pathCmd)

	fmt.Fprintln(os.Stderr, "REM To apply to current session, copy and run:")
	fmt.Fprintf(os.Stderr, "REM %s\n", pathCmd)

	return nil
}

// InitializeShell sets up shell integration for govman.
func InitializeShell(shell Shell, binPath string, force bool) error {
	// Validate the binary path first
	if err := validateBinPath(binPath); err != nil {
		return fmt.Errorf("invalid binary path: %w", err)
	}

	switch shell.Name() {
	case "powershell":
		return initializePowerShell(shell, binPath, force)
	case "cmd":
		return initializeCmdShell(shell, binPath, force)
	default:
		return initializeUnixShell(shell, binPath, force)
	}
}

// initializeUnixShell writes govman integration to the shell config file.
func initializeUnixShell(shell Shell, binPath string, force bool) error {
	configFile := shell.ConfigFile()

	// Create config directory if needed
	configDir := filepath.Dir(configFile)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	// Verify we can write to the directory
	testFile := filepath.Join(configDir, ".govman_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("insufficient permissions to write to %s: %w", configDir, err)
	}
	os.Remove(testFile)

	// Read existing content
	var existingContent string
	if content, err := os.ReadFile(configFile); err == nil {
		existingContent = string(content)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Check if govman is already configured
	if containsGovmanConfig(existingContent) {
		if !force {
			return fmt.Errorf("govman is already configured in %s (use --force to override)", configFile)
		}
		existingContent = removeExistingConfig(existingContent)
	}

	// Prepare new configuration
	setupCommands := shell.SetupCommands(binPath)
	newConfig := "\n" + strings.Join(setupCommands, "\n") + "\n"

	// Combine content
	finalContent := strings.TrimSpace(existingContent) + newConfig

	// Write to file with proper permissions
	if err := os.WriteFile(configFile, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", configFile, err)
	}

	fmt.Printf("✅ Successfully configured %s\n", shell.DisplayName())
	fmt.Printf("📝 Configuration added to: %s\n", configFile)
	fmt.Printf("🔄 Reload your shell or run: source %s\n", configFile)

	return nil
}

// initializePowerShell writes configuration to PowerShell profile.
func initializePowerShell(shell Shell, binPath string, force bool) error {
	profilePath := shell.ConfigFile()

	// Create profile directory if needed
	profileDir := filepath.Dir(profilePath)
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	// Verify write permissions
	testFile := filepath.Join(profileDir, ".govman_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("insufficient permissions to write to %s: %w", profileDir, err)
	}
	os.Remove(testFile)

	// Read existing content
	var existingContent string
	if content, err := os.ReadFile(profilePath); err == nil {
		existingContent = string(content)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read profile: %w", err)
	}

	// Check if govman is already configured
	if containsGovmanConfig(existingContent) {
		if !force {
			return fmt.Errorf("govman is already configured in PowerShell profile (use --force to override)")
		}
		existingContent = removeExistingConfig(existingContent)
	}

	// Prepare new configuration
	setupCommands := shell.SetupCommands(binPath)
	newConfig := "\r\n" + strings.Join(setupCommands, "\r\n") + "\r\n"

	// Combine content
	finalContent := strings.TrimSpace(existingContent) + newConfig

	// Write to file
	if err := os.WriteFile(profilePath, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("failed to write PowerShell profile: %w", err)
	}

	fmt.Printf("✅ Successfully configured PowerShell\n")
	fmt.Printf("📝 Configuration added to: %s\n", profilePath)
	fmt.Printf("🔄 Reload PowerShell or run: . $PROFILE\n")

	return nil
}

// initializeCmdShell creates a batch wrapper for Command Prompt.
func initializeCmdShell(shell Shell, binPath string, force bool) error {
	wrapperPath := filepath.Join(binPath, "govman.bat")

	// Check if wrapper exists
	if !force && fileExists(wrapperPath) {
		return fmt.Errorf("wrapper already exists at %s (use --force to override)", wrapperPath)
	}

	// Verify write permissions
	testFile := filepath.Join(binPath, ".govman_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("insufficient permissions to write to %s: %w", binPath, err)
	}
	os.Remove(testFile)

	// Create wrapper content using template for better maintainability
	tmpl := `@echo off
setlocal enabledelayedexpansion

REM GOVMAN Wrapper for Command Prompt
set "GOVMAN_BIN={{.BinPath}}\govman.exe"

REM Check if govman.exe exists
if not exist "%GOVMAN_BIN%" (
    echo Error: govman.exe not found at %GOVMAN_BIN% >&2
    exit /b 1
)

REM Handle 'use' command with special PATH updating logic
if "%~1"=="use" (
    if not "%~2"=="" (
        if not "%~2"=="--help" (
            if not "%~2"=="-h" (
                REM Execute govman use and capture output
                "%GOVMAN_BIN%" %* > "%TEMP%\govman_output.tmp" 2>&1
                set GOVMAN_EXIT_CODE=!errorlevel!
                
                if !GOVMAN_EXIT_CODE! equ 0 (
                    REM Look for PATH export command in output
                    set "PATH_UPDATED="
                    for /f "usebackq delims=" %%i in ("%TEMP%\govman_output.tmp") do (
                        set "LINE=%%i"
                        echo !LINE! | findstr /b /c:"set PATH=" >nul
                        if !errorlevel! equ 0 (
                            REM Execute the PATH update command
                            %%i
                            set "PATH_UPDATED=1"
                        )
                    )
                    del "%TEMP%\govman_output.tmp" 2>nul
                    if defined PATH_UPDATED (
                        echo.
                        echo ✓ Go version switched successfully
                        echo.
                        echo Note: This change only affects the current Command Prompt session.
                        echo To verify, run: go version
                    ) else (
                        echo Warning: No PATH update found in govman output >&2
                    )
                    exit /b 0
                ) else (
                    REM Show error output
                    type "%TEMP%\govman_output.tmp" >&2
                    del "%TEMP%\govman_output.tmp" 2>nul
                    exit /b !GOVMAN_EXIT_CODE!
                )
            )
        )
    )
)

REM For all other commands, just pass through
"%GOVMAN_BIN%" %*
exit /b %errorlevel%
`

	// Parse and execute template
	t, err := template.New("wrapper").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse wrapper template: %w", err)
	}

	var buf strings.Builder
	data := struct {
		BinPath string
	}{
		BinPath: binPath,
	}

	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to generate wrapper: %w", err)
	}

	// Write wrapper file with CRLF line endings for Windows
	content := strings.ReplaceAll(buf.String(), "\n", "\r\n")
	if err := os.WriteFile(wrapperPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create wrapper: %w", err)
	}

	// Print setup instructions (inline, no separate function)
	fmt.Printf("✅ Created govman wrapper: %s\n\n", wrapperPath)
	fmt.Println("📝 SETUP INSTRUCTIONS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Step 1: Add govman to your PATH")
	fmt.Println()
	fmt.Println("   Option A - Permanent (Recommended):")
	fmt.Printf("   setx PATH \"%%PATH%%;%s\"\n", binPath)
	fmt.Println()
	fmt.Println("   Option B - Current session only:")
	fmt.Printf("   set PATH=%%PATH%%;%s\n", binPath)
	fmt.Println()
	fmt.Println("Step 2: Restart Command Prompt (if using Option A)")
	fmt.Println()
	fmt.Println("Step 3: Verify installation")
	fmt.Println("   govman --version")
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("⚠️  COMMAND PROMPT LIMITATIONS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("• No automatic version switching (.govman-version not supported)")
	fmt.Println("• Must manually run 'govman use <version>' in each session")
	fmt.Println("• PATH changes only affect current Command Prompt window")
	fmt.Println()
	fmt.Println("💡 FOR BETTER EXPERIENCE")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Consider using one of these shells for auto-switching:")
	fmt.Println()
	fmt.Println("• PowerShell (Recommended for Windows):")
	fmt.Println("  powershell -Command \"govman init\"")
	fmt.Println()
	fmt.Println("• Git Bash (if installed):")
	fmt.Println("  bash -c 'govman init'")
	fmt.Println()
	fmt.Println("• WSL (Windows Subsystem for Linux):")
	fmt.Println("  wsl -e govman init")
	fmt.Println()

	return nil
}

// containsGovmanConfig checks if content contains govman configuration.
func containsGovmanConfig(content string) bool {
	markers := []string{
		"GOVMAN - Go Version Manager",
		"govman_auto_switch",
		"Invoke-GovmanAutoSwitch",
		"__govman_cd_hook",
	}

	for _, marker := range markers {
		if strings.Contains(content, marker) {
			return true
		}
	}

	return false
}

// removeExistingConfig removes existing govman configuration from content.
func removeExistingConfig(content string) string {
	// Use the pre-compiled regex for better performance
	cleanedContent := configRemovalRegex.ReplaceAllString(content, "")

	// Clean up excessive newlines
	cleanedContent = newlineRegex.ReplaceAllString(cleanedContent, "\n\n")

	return strings.TrimSpace(cleanedContent)
}

// GetShellInstructions returns manual setup instructions for a shell.
func GetShellInstructions(shell Shell, binPath string) string {
	var instructions strings.Builder

	instructions.WriteString(fmt.Sprintf("Manual setup for %s:\n\n", shell.DisplayName()))
	instructions.WriteString(fmt.Sprintf("1. Edit: %s\n\n", shell.ConfigFile()))
	instructions.WriteString("2. Add these lines:\n\n")

	commands := shell.SetupCommands(binPath)
	for _, cmd := range commands {
		instructions.WriteString(fmt.Sprintf("   %s\n", cmd))
	}

	instructions.WriteString("\n3. Reload your shell:\n")

	switch shell.Name() {
	case "fish":
		instructions.WriteString("   source ~/.config/fish/config.fish\n")
	case "powershell":
		instructions.WriteString("   . $PROFILE\n")
	case "cmd":
		instructions.WriteString("   (Restart Command Prompt)\n")
	default:
		instructions.WriteString(fmt.Sprintf("   source %s\n", shell.ConfigFile()))
	}

	return instructions.String()
}
