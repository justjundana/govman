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
	currentGOOS  = runtime.GOOS
	execLookPath = exec.LookPath
	userHomeDir  = os.UserHomeDir
	newlineRegex = regexp.MustCompile(`\n{3,}`)
)

const (
	govmanMarkerStart = "# GOVMAN - Go Version Manager"
	govmanMarkerEnd   = "# END GOVMAN"
)

type IntegrationOptions struct {
	BinPath        string
	ConfigPath     string
	ProjectFile    string
	InstallDir     string
	DefaultVersion string
	AutoSwitch     bool
}

type Shell interface {
	Name() string
	DisplayName() string
	ConfigFile() string
	PathCommand(path string) string
	SetupCommands(binPath string) []string
	IsAvailable() bool
	ExecutePathCommand(path string) error
}

type BashShell struct{ options *IntegrationOptions }
type ZshShell struct{ options *IntegrationOptions }
type FishShell struct{ options *IntegrationOptions }
type PowerShell struct{ options *IntegrationOptions }
type CmdShell struct{}

func Configure(shell Shell, options IntegrationOptions) Shell {
	optionsCopy := options
	switch configured := shell.(type) {
	case *BashShell:
		configured.options = &optionsCopy
	case *ZshShell:
		configured.options = &optionsCopy
	case *FishShell:
		configured.options = &optionsCopy
	case *PowerShell:
		configured.options = &optionsCopy
	}
	return shell
}

func effectiveOptions(binPath string, configured *IntegrationOptions) IntegrationOptions {
	options := IntegrationOptions{
		BinPath:     binPath,
		ConfigPath:  filepath.Join(filepath.Dir(binPath), "config.yaml"),
		ProjectFile: ".govman-goversion",
		InstallDir:  filepath.Join(filepath.Dir(binPath), "versions"),
		AutoSwitch:  true,
	}
	if configured != nil {
		options = *configured
		if options.BinPath == "" {
			options.BinPath = binPath
		}
		if options.ProjectFile == "" {
			options.ProjectFile = ".govman-goversion"
		}
		if options.InstallDir == "" {
			options.InstallDir = filepath.Join(filepath.Dir(options.BinPath), "versions")
		}
		if options.ConfigPath == "" {
			options.ConfigPath = filepath.Join(filepath.Dir(options.BinPath), "config.yaml")
		}
	}
	return options
}

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

// ==================== BASH SHELL ====================
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
	options := effectiveOptions(binPath, s.options)
	escapedPath := escapeBashPath(options.BinPath)
	escapedProjectFile := escapeBashPath(options.ProjectFile)
	escapedInstallDir := escapeBashPath(options.InstallDir)
	escapedConfigPath := escapeBashPath(options.ConfigPath)
	autoSwitchEnabled := "false"
	if options.AutoSwitch {
		autoSwitchEnabled = "true"
	}

	commands := []string{
		"# GOVMAN - Go Version Manager",
		fmt.Sprintf(`export PATH="%s:$PATH"`, escapedPath),
		"# Ensure GOBIN and GOPATH/bin are available",
		`if [ -n "$GOBIN" ]; then export PATH="$PATH:$GOBIN"; fi`,
		`if command -v go >/dev/null 2>&1; then export PATH="$PATH:$(go env GOPATH)/bin"; fi`,
		`export PATH="$PATH:$HOME/go/bin"`,
		"export GOTOOLCHAIN=local",
		"",
		"# Wrapper function for automatic PATH execution",
		"govman() {",
		fmt.Sprintf(`    local govman_bin="%s/govman"`, escapedPath),
		`    local command_name="" arg_index=1 expect_config=false`,
		`    for arg in "$@"; do`,
		`        if $expect_config; then expect_config=false; arg_index=$((arg_index + 1)); continue; fi`,
		`        case "$arg" in`,
		`            --config) expect_config=true ;;`,
		`            --config=*|--quiet|-q|--verbose|-V) ;;`,
		`            --) ;;`,
		`            -*) ;;`,
		`            *) command_name="$arg"; break ;;`,
		`        esac`,
		`        arg_index=$((arg_index + 1))`,
		`    done`,
		`    if [[ "$command_name" == "use" || "$command_name" == "refresh" ]]; then`,
		"        local output",
		`        output="$("$govman_bin" "$@" 2>&1)"`,
		"        local exit_code=$?",
		"        if [[ $exit_code -ne 0 ]]; then",
		`            echo "$output" >&2`,
		"            return $exit_code",
		"        fi",
		`        local export_cmd export_count`,
		`        export_cmd=$(printf '%s\n' "$output" | grep -E '^export PATH="[^"]*:\$PATH"$' || true)`,
		`        export_count=$(printf '%s\n' "$export_cmd" | awk 'NF { count++ } END { print count+0 }')`,
		`        if [[ "$export_count" -ne 1 ]]; then`,
		`            echo "govman returned success without one valid PATH command" >&2`,
		`            return 1`,
		`        fi`,
		`        eval "$export_cmd"`,
		`        printf '%s\n' "$output" | grep -Ev '^export PATH=' || true`,
		`        return 0`,
		"    fi",
		`    "$govman_bin" "$@"`,
		"}",
		"",
		"# Auto-switch Go versions based on .govman-goversion file",
		"govman_auto_switch() {",
		fmt.Sprintf(`    local auto_switch_enabled="%s"`, autoSwitchEnabled),
		fmt.Sprintf(`    local config_file="%s"`, escapedConfigPath),
		`    if [[ "$auto_switch_enabled" != "true" ]]; then`,
		"        return 0",
		"    fi",
		"",
		"    # Check file exists and is non-empty (-s), handle permission errors",
		fmt.Sprintf(`    local project_file="%s"`, escapedProjectFile),
		`    if [[ -s "$project_file" ]]; then`,
		`        local required_version`,
		`        required_version=$(cat "$project_file" 2>/dev/null | tr -d '\n\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')`,
		`        if [[ $? -ne 0 ]] || [[ -z "$required_version" ]]; then`,
		"            return 0",
		"        fi",
		"",
		"        # Validate version format (e.g., 1.25, 1.25.1, 1.25rc1)",
		`        if [[ ! "$required_version" =~ ^(latest|stable|[0-9]+\.[0-9]+(\.[0-9]+)?(-?(rc|beta|alpha)[0-9]*)?)$ ]]; then`,
		`            echo "Warning: Invalid version format in $project_file: $required_version" >&2`,
		"            return 0",
		"        fi",
		"",
		"        # Skip go version call if we already matched this version",
		`        local switch_key="$PWD|$required_version|$(command -v go 2>/dev/null)"`,
		`        if [[ "$switch_key" == "$__govman_last_switch" ]]; then`,
		"            return 0",
		"        fi",
		"",
		"        if ! command -v go >/dev/null 2>&1; then",
		`            echo "Go not found. Switching to Go $required_version..."`,
		`            if govman --config "$config_file" use "$required_version" >/dev/null 2>&1; then`,
		`                __govman_project_active=1`,
		`                __govman_last_switch="$PWD|$required_version|$(command -v go 2>/dev/null)"`,
		`            else`,
		`                echo "Warning: Failed to switch to Go $required_version. Install it with 'govman install $required_version'" >&2`,
		`            fi`,
		"            return",
		"        fi",
		"",
		`        local current_version=$(go version 2>/dev/null | awk '{print $3}' | sed -E 's/^go//; s/([0-9]+\.[0-9]+(\.[0-9]+)?).*/\1/')`,
		fmt.Sprintf(`        case "$(command -v go 2>/dev/null)" in "%s"/go*/bin/go|"%s"/go*/bin/go.exe) ;; *) current_version="" ;; esac`, escapedInstallDir, escapedInstallDir),
		`        if [[ ! "$current_version" =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?$ ]]; then current_version=""; fi`,
		"        # If required_version is major.minor only, truncate current_version for comparison",
		`        local compare_version="$current_version"`,
		`        if [[ "$required_version" =~ ^[0-9]+\.[0-9]+$ ]]; then compare_version="${current_version%.*}"; fi`,
		`        if [[ -z "$current_version" || "$compare_version" != "$required_version" ]]; then`,
		`            echo "Auto-switching to Go $required_version (required by .govman-goversion)"`,
		`            if govman --config "$config_file" use "$required_version" >/dev/null 2>&1; then`,
		`                __govman_project_active=1`,
		`                __govman_last_switch="$PWD|$required_version|$(command -v go 2>/dev/null)"`,
		`            else`,
		`                echo "Warning: Failed to switch to Go $required_version. Install it with 'govman install $required_version'" >&2`,
		`            fi`,
		`        elif [[ -n "$current_version" ]]; then`,
		`            __govman_last_switch="$switch_key"`,
		`            __govman_project_active=1`,
		"        fi",
		`    elif [[ "$__govman_project_active" == "1" ]]; then`,
		`        govman --config "$config_file" use default >/dev/null 2>&1 || echo "Warning: Failed to restore the default Go version" >&2`,
		`        __govman_project_active=0`,
		`        __govman_last_switch=""`,
		"    fi",
		"}",
		"",
		"# Bash-specific: Hook into PROMPT_COMMAND for directory changes",
		`__govman_prev_pwd="$PWD"`,
		`__govman_last_switch=""`,
		`__govman_project_active=0`,
		"__govman_check_dir_change() {",
		`    if [[ "$PWD" != "$__govman_prev_pwd" ]]; then`,
		`        __govman_prev_pwd="$PWD"`,
		"        govman_auto_switch",
		"    fi",
		"}",
		"",
		"# Add to PROMPT_COMMAND (preserves existing commands)",
		`if [[ ! "$PROMPT_COMMAND" =~ __govman_check_dir_change ]]; then`,
		`    if [[ -z "$PROMPT_COMMAND" ]]; then`,
		`        PROMPT_COMMAND="__govman_check_dir_change"`,
		"    else",
		`        PROMPT_COMMAND="__govman_check_dir_change;$PROMPT_COMMAND"`,
		"    fi",
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

// ==================== ZSH SHELL ====================
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
	options := effectiveOptions(binPath, s.options)
	escapedPath := escapeBashPath(options.BinPath)
	escapedProjectFile := escapeBashPath(options.ProjectFile)
	escapedInstallDir := escapeBashPath(options.InstallDir)
	escapedConfigPath := escapeBashPath(options.ConfigPath)
	autoSwitchEnabled := "false"
	if options.AutoSwitch {
		autoSwitchEnabled = "true"
	}

	commands := []string{
		"# GOVMAN - Go Version Manager",
		fmt.Sprintf(`export PATH="%s:$PATH"`, escapedPath),
		"# Ensure GOBIN and GOPATH/bin are available",
		`if [ -n "$GOBIN" ]; then export PATH="$PATH:$GOBIN"; fi`,
		`if command -v go >/dev/null 2>&1; then export PATH="$PATH:$(go env GOPATH)/bin"; fi`,
		`export PATH="$PATH:$HOME/go/bin"`,
		"export GOTOOLCHAIN=local",
		"",
		"# Wrapper function for automatic PATH execution",
		"govman() {",
		fmt.Sprintf(`    local govman_bin="%s/govman"`, escapedPath),
		`    local command_name="" expect_config=false`,
		`    for arg in "$@"; do`,
		`        if $expect_config; then expect_config=false; continue; fi`,
		`        case "$arg" in`,
		`            --config) expect_config=true ;;`,
		`            --config=*|--quiet|-q|--verbose|-V|--|-*) ;;`,
		`            *) command_name="$arg"; break ;;`,
		`        esac`,
		`    done`,
		`    if [[ "$command_name" == "use" || "$command_name" == "refresh" ]]; then`,
		"        local output",
		`        output="$("$govman_bin" "$@" 2>&1)"`,
		"        local exit_code=$?",
		"        if [[ $exit_code -ne 0 ]]; then",
		`            echo "$output" >&2`,
		"            return $exit_code",
		"        fi",
		`        local export_cmd export_count`,
		`        export_cmd=$(printf '%s\n' "$output" | grep -E '^export PATH="[^"]*:\$PATH"$' || true)`,
		`        export_count=$(printf '%s\n' "$export_cmd" | awk 'NF { count++ } END { print count+0 }')`,
		`        if [[ "$export_count" -ne 1 ]]; then echo "govman returned success without one valid PATH command" >&2; return 1; fi`,
		`        eval "$export_cmd"`,
		`        printf '%s\n' "$output" | grep -Ev '^export PATH=' || true`,
		`        return 0`,
		"    fi",
		`    "$govman_bin" "$@"`,
		"}",
		"",
		"# Auto-switch Go versions based on .govman-goversion file",
		"govman_auto_switch() {",
		fmt.Sprintf(`    local auto_switch_enabled="%s"`, autoSwitchEnabled),
		fmt.Sprintf(`    local config_file="%s"`, escapedConfigPath),
		`    if [[ "$auto_switch_enabled" != "true" ]]; then`,
		"        return 0",
		"    fi",
		"",
		"    # Check file exists and is non-empty (-s), handle permission errors",
		fmt.Sprintf(`    local project_file="%s"`, escapedProjectFile),
		`    if [[ -s "$project_file" ]]; then`,
		`        local required_version`,
		`        required_version=$(cat "$project_file" 2>/dev/null | tr -d '\n\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')`,
		`        if [[ $? -ne 0 ]] || [[ -z "$required_version" ]]; then`,
		"            return 0",
		"        fi",
		"",
		"        # Validate version format (e.g., 1.25, 1.25.1, 1.25rc1)",
		`        if [[ ! "$required_version" =~ ^(latest|stable|[0-9]+\.[0-9]+(\.[0-9]+)?(-?(rc|beta|alpha)[0-9]*)?)$ ]]; then`,
		`            echo "Warning: Invalid version format in $project_file: $required_version" >&2`,
		"            return 0",
		"        fi",
		"",
		"        # Skip go version call if we already matched this version",
		`        local switch_key="$PWD|$required_version|$(command -v go 2>/dev/null)"`,
		`        if [[ "$switch_key" == "$__govman_last_switch" ]]; then`,
		"            return 0",
		"        fi",
		"",
		"        if ! command -v go >/dev/null 2>&1; then",
		`            echo "Go not found. Switching to Go $required_version..."`,
		`            if govman --config "$config_file" use "$required_version" >/dev/null 2>&1; then`,
		`                __govman_project_active=1`,
		`                __govman_last_switch="$PWD|$required_version|$(command -v go 2>/dev/null)"`,
		`            else`,
		`                echo "Warning: Failed to switch to Go $required_version. Install it with 'govman install $required_version'" >&2`,
		`            fi`,
		"            return",
		"        fi",
		"",
		`        local current_version=$(go version 2>/dev/null | awk '{print $3}' | sed -E 's/^go//; s/([0-9]+\.[0-9]+(\.[0-9]+)?).*/\1/')`,
		fmt.Sprintf(`        case "$(command -v go 2>/dev/null)" in "%s"/go*/bin/go|"%s"/go*/bin/go.exe) ;; *) current_version="" ;; esac`, escapedInstallDir, escapedInstallDir),
		`        if [[ ! "$current_version" =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?$ ]]; then current_version=""; fi`,
		"        # If required_version is major.minor only, truncate current_version for comparison",
		`        local compare_version="$current_version"`,
		`        if [[ "$required_version" =~ ^[0-9]+\.[0-9]+$ ]]; then compare_version="${current_version%.*}"; fi`,
		`        if [[ -z "$current_version" || "$compare_version" != "$required_version" ]]; then`,
		`            echo "Auto-switching to Go $required_version (required by .govman-goversion)"`,
		`            if govman --config "$config_file" use "$required_version" >/dev/null 2>&1; then`,
		`                __govman_last_switch="$PWD|$required_version|$(command -v go 2>/dev/null)"`,
		`                __govman_project_active=1`,
		`            else`,
		`                echo "Warning: Failed to switch to Go $required_version. Install it with 'govman install $required_version'" >&2`,
		`            fi`,
		`        elif [[ -n "$current_version" ]]; then`,
		`            __govman_last_switch="$switch_key"`,
		`            __govman_project_active=1`,
		"        fi",
		`    elif [[ "$__govman_project_active" == "1" ]]; then`,
		`        govman --config "$config_file" use default >/dev/null 2>&1 || echo "Warning: Failed to restore the default Go version" >&2`,
		`        __govman_project_active=0`,
		`        __govman_last_switch=""`,
		"    fi",
		"}",
		"",
		"# Zsh-specific: Hook into chpwd for directory changes",
		`__govman_last_switch=""`,
		`__govman_project_active=0`,
		"autoload -U add-zsh-hook",
		`if [[ ! "${chpwd_functions[(r)govman_auto_switch]}" ]]; then`,
		"    add-zsh-hook chpwd govman_auto_switch",
		"fi",
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

// ==================== FISH SHELL ====================
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
	options := effectiveOptions(binPath, s.options)
	escapedPath := escapeFishPath(options.BinPath)
	escapedProjectFile := escapeFishPath(options.ProjectFile)
	escapedInstallDir := escapeFishPath(options.InstallDir)
	escapedConfigPath := escapeFishPath(options.ConfigPath)
	autoSwitchEnabled := "false"
	if options.AutoSwitch {
		autoSwitchEnabled = "true"
	}

	commands := []string{
		"# GOVMAN - Go Version Manager",
		fmt.Sprintf(`fish_add_path -p "%s"`, escapedPath),
		"set -gx GOTOOLCHAIN local",
		"",
		"# Ensure GOBIN and GOPATH/bin are available",
		`if test -n "$GOBIN"; and test -d "$GOBIN"; fish_add_path -a "$GOBIN"; end`,
		`if type -q go; set -l gopath (go env GOPATH 2>/dev/null); if test -n "$gopath"; and test -d "$gopath/bin"; fish_add_path -a "$gopath/bin"; end; end`,
		`set -l homegobin "$HOME/go/bin"; if test -d "$homegobin"; fish_add_path -a "$homegobin"; end`,
		"",
		"# Wrapper function for automatic PATH execution",
		"function govman",
		fmt.Sprintf(`    set govman_bin "%s/govman"`, escapedPath),
		`    set -l command_name ""`,
		`    set -l expect_config 0`,
		`    for arg in $argv`,
		`        if test $expect_config -eq 1; set expect_config 0; continue; end`,
		`        switch $arg`,
		`            case --config; set expect_config 1`,
		`            case '--config=*' --quiet -q --verbose -V -- '-*'`,
		`            case '*'; set command_name $arg; break`,
		`        end`,
		`    end`,
		`    if test "$command_name" = "refresh"; or test "$command_name" = "use"`,
		"        set output ($govman_bin $argv 2>&1)",
		"        set exit_code $status",
		"        if test $exit_code -ne 0",
		"            for line in $output",
		"                echo $line >&2",
		"            end",
		"            return $exit_code",
		"        end",
		`        set -l path_lines (string match -r '^fish_add_path -p "[^"]+"$' -- $output)`,
		`        if test (count $path_lines) -ne 1`,
		`            echo "govman returned success without one valid PATH command" >&2`,
		`            return 1`,
		`        end`,
		`        eval $path_lines[1]`,
		`        for line in $output; if not string match -qr '^fish_add_path' -- $line; echo $line; end; end`,
		`        return 0`,
		"    end",
		"    $govman_bin $argv",
		"end",
		"",
		"# Auto-switch Go versions based on .govman-goversion file",
		"function govman_auto_switch",
		fmt.Sprintf(`    set auto_switch_enabled "%s"`, autoSwitchEnabled),
		fmt.Sprintf(`    set config_file "%s"`, escapedConfigPath),
		`    if test "$auto_switch_enabled" != "true"`,
		"        return 0",
		"    end",
		"",
		"    # Check file exists and is non-empty (-s), handle permission/empty errors",
		fmt.Sprintf(`    set project_file "%s"`, escapedProjectFile),
		`    if test -s "$project_file"`,
		`        set required_version (string trim < "$project_file" 2>/dev/null)`,
		`        if test -z "$required_version"`,
		"            return 0",
		"        end",
		"",
		"        # Validate version format (e.g., 1.25, 1.25.1, 1.25rc1)",
		`        if not string match -qr '^(latest|stable|[0-9]+\.[0-9]+(\.[0-9]+)?(-?(rc|beta|alpha)[0-9]*)?)$' -- "$required_version"`,
		`            echo "Warning: Invalid version format in $project_file: $required_version" >&2`,
		"            return 0",
		"        end",
		"",
		"        # Skip go version call if we already matched this version",
		`        set switch_key "$PWD|$required_version|"(command -v go 2>/dev/null)`,
		`        if set -q __govman_last_switch; and test "$switch_key" = "$__govman_last_switch"`,
		"            return 0",
		"        end",
		"",
		"        if not command -v go >/dev/null 2>&1",
		`            echo "Go not found. Switching to Go $required_version..."`,
		`            if govman --config "$config_file" use "$required_version" >/dev/null 2>&1`,
		`                set -g __govman_project_active 1`,
		`                set -g __govman_last_switch "$PWD|$required_version|"(command -v go 2>/dev/null)`,
		`            else`,
		`                echo "Warning: Failed to switch to Go $required_version. Install it with 'govman install $required_version'" >&2`,
		`            end`,
		"            return",
		"        end",
		"",
		"        set current_version (go version 2>/dev/null | awk '{print $3}' | sed -E 's/^go//; s/([0-9]+\\.[0-9]+(\\.[0-9]+)?).*/\\1/')",
		fmt.Sprintf(`        if not string match -q "%s/go*/bin/go*" -- (command -v go 2>/dev/null); set current_version ""; end`, escapedInstallDir),
		`        if not string match -qr '^[0-9]+\.[0-9]+(\.[0-9]+)?$' -- "$current_version"; set current_version ""; end`,
		"        # If required_version is major.minor only, truncate current_version for comparison",
		`        set compare_version $current_version`,
		`        if not string match -q '*.*.*' -- "$required_version"; set compare_version (string replace -r '(\d+\.\d+).*' '$1' -- $current_version); end`,
		`        if test -z "$current_version"; or test "$compare_version" != "$required_version"`,
		"            echo \"Auto-switching to Go $required_version (required by .govman-goversion)\"",
		`            govman --config "$config_file" use "$required_version" >/dev/null 2>&1; or begin`,
		`                echo "Warning: Failed to switch to Go $required_version. Install it with 'govman install $required_version'" >&2`,
		"            end",
		`            set -g __govman_last_switch "$PWD|$required_version|"(command -v go 2>/dev/null)`,
		`            set -g __govman_project_active 1`,
		`        else if test -n "$current_version"`,
		`            set -g __govman_last_switch "$switch_key"`,
		`            set -g __govman_project_active 1`,
		"        end",
		`    else if set -q __govman_project_active; and test "$__govman_project_active" = "1"`,
		`        govman --config "$config_file" use default >/dev/null 2>&1; or echo "Warning: Failed to restore the default Go version" >&2`,
		`        set -g __govman_project_active 0`,
		`        set -e __govman_last_switch`,
		"    end",
		"end",
		"",
		"# Fish-specific: Hook into directory changes",
		"functions -q __govman_cd_hook; and functions -e __govman_cd_hook",
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

// ==================== POWER SHELL ====================
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
	options := effectiveOptions(binPath, s.options)
	escapedPath := escapePowerShellPath(options.BinPath)
	escapedProjectFile := escapePowerShellPath(options.ProjectFile)
	escapedInstallDir := escapePowerShellPath(options.InstallDir)
	escapedConfigPath := escapePowerShellPath(options.ConfigPath)
	autoSwitchEnabled := "$false"
	if options.AutoSwitch {
		autoSwitchEnabled = "$true"
	}

	commands := []string{
		"# GOVMAN - Go Version Manager",
		fmt.Sprintf(`$env:PATH = "%s;" + $env:PATH`, escapedPath),
		"$env:GOTOOLCHAIN = 'local'",
		"",
		"# Ensure GOPATH\\bin and GOBIN are available",
		`if ($env:GOBIN) { $env:PATH = $env:PATH + ";$env:GOBIN" }`,
		`$goCmd = Get-Command go -ErrorAction SilentlyContinue; if ($goCmd) { $gopath = (& go env GOPATH 2>$null); if ($gopath) { $env:PATH = $env:PATH + ";$gopath\bin" } }`,
		`$homeGoBin = Join-Path $env:USERPROFILE "go\bin"; if (Test-Path $homeGoBin) { $env:PATH = $env:PATH + ";$homeGoBin" }`,
		"",
		"# Wrapper function for automatic PATH execution",
		"function govman {",
		fmt.Sprintf(`    $govman_bin = "%s\govman.exe"`, escapedPath),
		"    $commandName = $null",
		"    $expectConfig = $false",
		"    foreach ($argument in $args) {",
		"        if ($expectConfig) { $expectConfig = $false; continue }",
		"        if ($argument -eq '--config') { $expectConfig = $true; continue }",
		"        if ($argument -match '^(--config=|--quiet$|-q$|--verbose$|-V$|-)') { continue }",
		"        $commandName = $argument; break",
		"    }",
		"    if ($commandName -eq 'refresh' -or $commandName -eq 'use') {",
		"        try {",
		"            $output = & $govman_bin @args 2>&1",
		"            $exitCode = $LASTEXITCODE",
		"            if ($exitCode -ne 0) {",
		"                $output | ForEach-Object { Write-Error $_ }",
		"                $global:LASTEXITCODE = $exitCode",
		"                return",
		"            }",
		"            $pathCommands = @($output | Where-Object { $_ -match '^\\$env:PATH\\s*=\\s*\"[^\"]+;\"\\s*\\+\\s*\\$env:PATH$' })",
		"            if ($pathCommands.Count -ne 1) { Write-Error 'govman returned success without one valid PATH command'; $global:LASTEXITCODE = 1; return }",
		"            Invoke-Expression $pathCommands[0]",
		"            $output | Where-Object { $_ -notmatch '^\\$env:PATH' } | ForEach-Object { Write-Output $_ }",
		"            $global:LASTEXITCODE = 0",
		"            return",
		"        } catch {",
		"            Write-Error $_.Exception.Message",
		"            $global:LASTEXITCODE = 1",
		"            return",
		"        }",
		"    }",
		"    & $govman_bin @args",
		"}",
		"",
		"# Auto-switch Go versions based on .govman-goversion file",
		"function Invoke-GovmanAutoSwitch {",
		fmt.Sprintf("    $autoSwitchEnabled = %s", autoSwitchEnabled),
		fmt.Sprintf(`    $configFile = "%s"`, escapedConfigPath),
		"    if (-not $autoSwitchEnabled) { return }",
		"",
		fmt.Sprintf(`    $projectFile = "%s"`, escapedProjectFile),
		"    if (Test-Path -LiteralPath $projectFile) {",
		"        # Check file is non-empty and readable",
		"        try {",
		"            $fileInfo = Get-Item -LiteralPath $projectFile -ErrorAction Stop",
		"            if ($fileInfo.Length -eq 0) { return }",
		"            $requiredVersion = (Get-Content -LiteralPath $projectFile -Raw -ErrorAction Stop).Trim()",
		"        } catch {",
		"            return",
		"        }",
		"",
		"        if ($requiredVersion -and $requiredVersion -match '^(latest|stable|[0-9]+\\.[0-9]+(\\.[0-9]+)?(-?(rc|beta|alpha)[0-9]*)?)$') {",
		"            # Skip go version call if we already matched this version",
		"            $switchKey = \"$($PWD.Path)|$requiredVersion|$((Get-Command go -ErrorAction SilentlyContinue).Source)\"",
		"            if ($Global:GovmanLastSwitch -eq $switchKey) { return }",
		"",
		"            $currentVersion = $null",
		"            try {",
		"                $goVersionOutput = go version 2>$null",
		"                if ($LASTEXITCODE -eq 0 -and $goVersionOutput) {",
		"                    if ($goVersionOutput -match 'go version go(\\d+\\.\\d+(?:\\.\\d+)?)') {",
		"                        $currentVersion = $matches[1]",
		"                    }",
		fmt.Sprintf(`                    $activeGo = (Get-Command go -ErrorAction SilentlyContinue).Source; if (-not $activeGo.StartsWith("%s", [StringComparison]::OrdinalIgnoreCase)) { $currentVersion = $null }`, escapedInstallDir),
		"                }",
		"            } catch {}",
		"",
		"            if (-not $currentVersion) {",
		"                Write-Host \"Go not found. Switching to Go $requiredVersion...\" -ForegroundColor Yellow",
		"                govman --config $configFile use $requiredVersion *>$null",
		"                if ($LASTEXITCODE -ne 0) {",
		"                    Write-Warning \"Failed to switch to Go $requiredVersion. Install it with 'govman install $requiredVersion'\"",
		"                } else {",
		"                    $Global:GovmanProjectActive = $true",
		"                    $Global:GovmanLastSwitch = \"$($PWD.Path)|$requiredVersion|$((Get-Command go -ErrorAction SilentlyContinue).Source)\"",
		"                }",
		"                return",
		"            }",
		"",
		"            # If requiredVersion is major.minor only, truncate currentVersion for comparison",
		"            $compareVersion = $currentVersion",
		"            if ($requiredVersion -notmatch '^\\d+\\.\\d+\\.\\d+') { $compareVersion = ($currentVersion -replace '^(\\d+\\.\\d+).*', '$1') }",
		"            if ($compareVersion -ne $requiredVersion) {",
		"                Write-Host \"Auto-switching to Go $requiredVersion (required by .govman-goversion)\" -ForegroundColor Yellow",
		"                govman --config $configFile use $requiredVersion *>$null",
		"                if ($LASTEXITCODE -ne 0) {",
		"                    Write-Warning \"Failed to switch to Go $requiredVersion. Install it with 'govman install $requiredVersion'\"",
		"                }",
		"                $Global:GovmanLastSwitch = \"$($PWD.Path)|$requiredVersion|$((Get-Command go -ErrorAction SilentlyContinue).Source)\"",
		"                $Global:GovmanProjectActive = $true",
		"            } else {",
		"                $Global:GovmanLastSwitch = $switchKey",
		"                $Global:GovmanProjectActive = $true",
		"            }",
		"        } elseif ($requiredVersion) {",
		"            Write-Warning \"Invalid version format in $projectFile: $requiredVersion\"",
		"        }",
		"    } elseif ($Global:GovmanProjectActive) {",
		"        govman --config $configFile use default *>$null",
		"        if ($LASTEXITCODE -ne 0) { Write-Warning 'Failed to restore the default Go version' }",
		"        $Global:GovmanProjectActive = $false",
		"        $Global:GovmanLastSwitch = $null",
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
		"if ((Get-Command prompt -ErrorAction SilentlyContinue) -and -not $Global:GovmanPromptInjected) {",
		"    $Global:GovmanOriginalPrompt = $function:prompt",
		"    function global:prompt {",
		"        Invoke-GovmanLocationCheck",
		"        if ($Global:GovmanOriginalPrompt) {",
		"            & $Global:GovmanOriginalPrompt",
		"        } else {",
		"            \"PS $($executionContext.SessionState.Path.CurrentLocation)$('>' * ($nestedPromptLevel + 1)) \"",
		"        }",
		"    }",
		"    $Global:GovmanPromptInjected = $true",
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

// ==================== CMD SHELL ====================
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
	escapedPath := escapeCmdPath(binPath)

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
		"REM Note: Auto-switching (.govman-goversion) is not available in Command Prompt",
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
		return initializeCmdShell(binPath, force)
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

	setupCommands := shell.SetupCommands(binPath)
	if err := writeShellIntegration(configFile, setupCommands, force, "\n"); err != nil {
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

	setupCommands := shell.SetupCommands(binPath)
	if err := writeShellIntegration(profilePath, setupCommands, force, "\r\n"); err != nil {
		return fmt.Errorf("failed to write PowerShell profile: %w", err)
	}

	fmt.Printf("✅ Successfully configured PowerShell\n")
	fmt.Printf("📝 Configuration added to: %s\n", profilePath)
	fmt.Printf("🔄 Reload PowerShell or run: . $PROFILE\n")

	return nil
}

// initializeCmdShell creates a batch wrapper for Command Prompt.
func initializeCmdShell(binPath string, force bool) error {
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
	fmt.Println("• No automatic version switching (.govman-goversion not supported)")
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

type markerBlock struct {
	start int
	end   int
	found bool
}

func inspectGovmanBlock(content string) (markerBlock, error) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	starts := make([]int, 0, 1)
	ends := make([]int, 0, 1)
	for index, line := range lines {
		switch line {
		case govmanMarkerStart:
			starts = append(starts, index)
		case govmanMarkerEnd:
			ends = append(ends, index)
		}
	}
	if len(starts) == 0 && len(ends) == 0 {
		return markerBlock{}, nil
	}
	if len(starts) != 1 || len(ends) != 1 || starts[0] >= ends[0] {
		return markerBlock{}, fmt.Errorf("malformed or duplicate govman marker block")
	}
	return markerBlock{start: starts[0], end: ends[0], found: true}, nil
}

// containsGovmanConfig only recognizes one complete exact marker block.
func containsGovmanConfig(content string) bool {
	block, err := inspectGovmanBlock(content)
	return err == nil && block.found
}

func removeExistingConfigStrict(content string) (string, error) {
	newline := "\n"
	if strings.Contains(content, "\r\n") {
		newline = "\r\n"
	}
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	block, err := inspectGovmanBlock(normalized)
	if err != nil {
		return "", err
	}
	if !block.found {
		return content, nil
	}
	lines := strings.Split(normalized, "\n")
	lines = append(lines[:block.start], lines[block.end+1:]...)
	cleaned := strings.Join(lines, "\n")
	cleaned = newlineRegex.ReplaceAllString(cleaned, "\n\n")
	cleaned = strings.Trim(cleaned, "\n")
	return strings.ReplaceAll(cleaned, "\n", newline), nil
}

// removeExistingConfig preserves malformed content instead of deleting an
// open-ended range. Initialization uses the strict variant and reports errors.
func removeExistingConfig(content string) string {
	cleaned, err := removeExistingConfigStrict(content)
	if err != nil {
		return content
	}
	return cleaned
}

func writeShellIntegration(configPath string, setupCommands []string, force bool, defaultNewline string) error {
	newline := defaultNewline
	mode := os.FileMode(0644)
	existing := ""
	info, err := os.Lstat(configPath)
	if err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refusing to replace non-regular shell config: %s", configPath)
		}
		mode = info.Mode().Perm()
		data, readErr := os.ReadFile(configPath)
		if readErr != nil {
			return readErr
		}
		existing = string(data)
		if strings.Contains(existing, "\r\n") {
			newline = "\r\n"
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	block, err := inspectGovmanBlock(existing)
	if err != nil {
		return err
	}
	if block.found && !force {
		return fmt.Errorf("govman is already configured (use --force to override)")
	}
	if block.found {
		existing, err = removeExistingConfigStrict(existing)
		if err != nil {
			return err
		}
	}

	base := strings.TrimRight(existing, "\r\n")
	generated := strings.Join(setupCommands, newline)
	finalContent := generated + newline
	if base != "" {
		finalContent = base + newline + newline + generated + newline
	}
	return atomicWriteShellFile(configPath, []byte(finalContent), mode)
}

func atomicWriteShellFile(path string, content []byte, mode os.FileMode) (resultErr error) {
	directory := filepath.Dir(path)
	backupPath := ""
	if existing, err := os.ReadFile(path); err == nil {
		backupFile, createErr := os.CreateTemp(directory, ".govman-backup-*")
		if createErr != nil {
			return createErr
		}
		backupPath = backupFile.Name()
		if chmodErr := backupFile.Chmod(mode); chmodErr != nil {
			backupFile.Close()
			os.Remove(backupPath)
			return chmodErr
		}
		if _, writeErr := backupFile.Write(existing); writeErr != nil {
			backupFile.Close()
			os.Remove(backupPath)
			return writeErr
		}
		if syncErr := backupFile.Sync(); syncErr != nil {
			backupFile.Close()
			os.Remove(backupPath)
			return syncErr
		}
		if closeErr := backupFile.Close(); closeErr != nil {
			os.Remove(backupPath)
			return closeErr
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	defer func() {
		if backupPath != "" {
			os.Remove(backupPath)
		}
	}()
	tempFile, err := os.CreateTemp(directory, ".govman-config-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	closed := false
	defer func() {
		if !closed {
			if closeErr := tempFile.Close(); resultErr == nil && closeErr != nil {
				resultErr = closeErr
			}
		}
		if resultErr != nil {
			os.Remove(tempPath)
		}
	}()
	if err := tempFile.Chmod(mode); err != nil {
		return err
	}
	if _, err := tempFile.Write(content); err != nil {
		return err
	}
	if err := tempFile.Sync(); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	closed = true
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	return nil
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
