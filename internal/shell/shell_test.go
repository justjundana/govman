package shell

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name         string
		goos         string
		shellEnv     string
		mockCommands map[string]bool
		expectedType interface{}
	}{
		{
			name:         "Windows with pwsh available",
			goos:         "windows",
			mockCommands: map[string]bool{"pwsh": true},
			expectedType: &PowerShell{},
		},
		{
			name:         "Windows with powershell available",
			goos:         "windows",
			mockCommands: map[string]bool{"powershell": true},
			expectedType: &PowerShell{},
		},
		{
			name:         "Windows fallback to cmd",
			goos:         "windows",
			mockCommands: map[string]bool{},
			expectedType: &CmdShell{},
		},
		{
			name:         "Unix with zsh in SHELL",
			goos:         "linux",
			shellEnv:     "/usr/bin/zsh",
			mockCommands: map[string]bool{"zsh": true},
			expectedType: &ZshShell{},
		},
		{
			name:         "Unix with fish in SHELL",
			goos:         "linux",
			shellEnv:     "/usr/bin/fish",
			mockCommands: map[string]bool{"fish": true},
			expectedType: &FishShell{},
		},
		{
			name:         "Unix with bash in SHELL",
			goos:         "linux",
			shellEnv:     "/bin/bash",
			mockCommands: map[string]bool{"bash": true},
			expectedType: &BashShell{},
		},
		{
			name:         "Unix with unknown SHELL",
			goos:         "linux",
			shellEnv:     "/bin/unknown",
			mockCommands: map[string]bool{"zsh": true},
			expectedType: &ZshShell{},
		},
		{
			name:         "Unix with no SHELL env",
			goos:         "linux",
			mockCommands: map[string]bool{"zsh": true},
			expectedType: &ZshShell{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock currentGOOS
			originalGOOS := currentGOOS
			defer func() { currentGOOS = originalGOOS }()
			currentGOOS = tc.goos

			// Mock SHELL environment variable
			if tc.shellEnv != "" {
				originalShell := os.Getenv("SHELL")
				defer func() { os.Setenv("SHELL", originalShell) }()
				os.Setenv("SHELL", tc.shellEnv)
			}

			// Mock exec.LookPath
			originalLookPath := execLookPath
			defer func() { execLookPath = originalLookPath }()
			execLookPath = func(cmd string) (string, error) {
				if tc.mockCommands[cmd] {
					return "/usr/bin/" + cmd, nil
				}
				return "", exec.ErrNotFound
			}

			shell := Detect()

			if shell == nil {
				t.Fatal("Detect() returned nil")
			}

			// Check type
			switch tc.expectedType.(type) {
			case *BashShell:
				if _, ok := shell.(*BashShell); !ok {
					t.Errorf("Expected BashShell, got %T", shell)
				}
			case *ZshShell:
				if _, ok := shell.(*ZshShell); !ok {
					t.Errorf("Expected ZshShell, got %T", shell)
				}
			case *FishShell:
				if _, ok := shell.(*FishShell); !ok {
					t.Errorf("Expected FishShell, got %T", shell)
				}
			case *PowerShell:
				if _, ok := shell.(*PowerShell); !ok {
					t.Errorf("Expected PowerShell, got %T", shell)
				}
			case *CmdShell:
				if _, ok := shell.(*CmdShell); !ok {
					t.Errorf("Expected CmdShell, got %T", shell)
				}
			}
		})
	}
}

func TestIsCommandAvailable(t *testing.T) {
	testCases := []struct {
		name       string
		command    string
		mockResult bool
		expected   bool
	}{
		{
			name:       "Command available",
			command:    "bash",
			mockResult: true,
			expected:   true,
		},
		{
			name:       "Command not available",
			command:    "nonexistent",
			mockResult: false,
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock exec.LookPath
			originalLookPath := execLookPath
			defer func() { execLookPath = originalLookPath }()
			execLookPath = func(cmd string) (string, error) {
				if tc.mockResult && cmd == tc.command {
					return "/usr/bin/" + cmd, nil
				}
				return "", exec.ErrNotFound
			}

			result := isCommandAvailable(tc.command)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		isDir    bool
		expected bool
	}{
		{
			name:     "File exists",
			filename: "test_file.txt",
			isDir:    false,
			expected: true,
		},
		{
			name:     "Directory exists",
			filename: "test_dir",
			isDir:    true,
			expected: false, // fileExists returns !info.IsDir()
		},
		{
			name:     "File does not exist",
			filename: "nonexistent.txt",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create the fixtures inside a temp dir instead of the package
			// source directory (the working directory during `go test`), so the
			// suite never writes into the checkout and works read-only.
			path := filepath.Join(t.TempDir(), tc.filename)

			var err error
			if tc.expected || tc.isDir {
				if tc.isDir {
					err = os.Mkdir(path, 0755)
				} else {
					err = os.WriteFile(path, []byte("test"), 0644)
				}
				if err != nil {
					t.Fatalf("Failed to create test file/dir: %v", err)
				}
			}

			result := fileExists(path)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestBashShell(t *testing.T) {
	shell := &BashShell{}

	// Test Name
	if shell.Name() != "bash" {
		t.Errorf("Expected 'bash', got %s", shell.Name())
	}

	// Test DisplayName
	if shell.DisplayName() != "Bash" {
		t.Errorf("Expected 'Bash', got %s", shell.DisplayName())
	}

	// Test IsAvailable
	originalLookPath := execLookPath
	defer func() { execLookPath = originalLookPath }()
	execLookPath = func(cmd string) (string, error) {
		if cmd == "bash" {
			return "/bin/bash", nil
		}
		return "", exec.ErrNotFound
	}

	if !shell.IsAvailable() {
		t.Error("Expected bash to be available")
	}

	// Test ConfigFile
	originalUserHomeDir := userHomeDir
	defer func() { userHomeDir = originalUserHomeDir }()
	testHome := filepath.FromSlash("/test/home")
	userHomeDir = func() (string, error) {
		return testHome, nil
	}

	// Test with ~/.bashrc existing. ConfigFile builds this with filepath.Join, so
	// the expectation has to use the platform separator rather than a literal "/".
	bashrcPath := filepath.Join(testHome, ".bashrc")
	if shell.ConfigFile() != bashrcPath {
		t.Errorf("Expected %s, got %s", bashrcPath, shell.ConfigFile())
	}

	// Test with ~/.bashrc not existing but ~/.bash_profile exists
	// This is already covered by the ConfigFile logic, no need to mock file existence for this test

	// Test PathCommand
	if shell.PathCommand("/usr/local/bin") != "export PATH=\"/usr/local/bin:$PATH\"" {
		t.Errorf("PathCommand output incorrect")
	}

	// Test SetupCommands
	commands := shell.SetupCommands("/usr/local/bin")
	if len(commands) == 0 {
		t.Error("SetupCommands should return commands")
	}
	if !strings.Contains(commands[0], "GOVMAN - Go Version Manager") {
		t.Error("SetupCommands should contain GOVMAN header")
	}

	// Test ExecutePathCommand
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Use current directory which exists
	err := shell.ExecutePathCommand(".")
	wOut.Close()
	wErr.Close()

	if err != nil {
		t.Errorf("ExecutePathCommand failed: %v", err)
	}

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)
	output := string(outBytes) + string(errBytes)

	if !strings.Contains(output, "export PATH") {
		t.Error("ExecutePathCommand should output PATH command")
	}

	os.Stdout = oldStdout
	os.Stderr = oldStderr
}

func TestZshShell(t *testing.T) {
	shell := &ZshShell{}

	// Test Name
	if shell.Name() != "zsh" {
		t.Errorf("Expected 'zsh', got %s", shell.Name())
	}

	// Test DisplayName
	if shell.DisplayName() != "Zsh" {
		t.Errorf("Expected 'Zsh', got %s", shell.DisplayName())
	}

	// Test IsAvailable
	originalLookPath := execLookPath
	defer func() { execLookPath = originalLookPath }()
	execLookPath = func(cmd string) (string, error) {
		if cmd == "zsh" {
			return "/bin/zsh", nil
		}
		return "", exec.ErrNotFound
	}

	if !shell.IsAvailable() {
		t.Error("Expected zsh to be available")
	}

	// Test ConfigFile
	originalUserHomeDir := userHomeDir
	defer func() { userHomeDir = originalUserHomeDir }()

	testHome := t.TempDir()
	userHomeDir = func() (string, error) {
		return testHome, nil
	}

	if shell.ConfigFile() != filepath.Join(testHome, ".zshrc") {
		t.Errorf("ConfigFile output incorrect")
	}

	// Test PathCommand
	if shell.PathCommand("/usr/local/bin") != "export PATH=\"/usr/local/bin:$PATH\"" {
		t.Errorf("PathCommand output incorrect")
	}

	// Test SetupCommands
	commands := shell.SetupCommands("/usr/local/bin")
	if len(commands) == 0 {
		t.Error("SetupCommands should return commands")
	}
	if !strings.Contains(commands[0], "GOVMAN - Go Version Manager") {
		t.Error("SetupCommands should contain GOVMAN header")
	}

	// Test ExecutePathCommand
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Use current directory which exists
	err := shell.ExecutePathCommand(".")
	wOut.Close()
	wErr.Close()

	if err != nil {
		t.Errorf("ExecutePathCommand failed: %v", err)
	}

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)
	output := string(outBytes) + string(errBytes)

	if !strings.Contains(output, "export PATH") {
		t.Error("ExecutePathCommand should output PATH command")
	}

	os.Stdout = oldStdout
	os.Stderr = oldStderr
}

func TestFishShell(t *testing.T) {
	shell := &FishShell{}

	// Test Name
	if shell.Name() != "fish" {
		t.Errorf("Expected 'fish', got %s", shell.Name())
	}

	// Test DisplayName
	if shell.DisplayName() != "Fish" {
		t.Errorf("Expected 'Fish', got %s", shell.DisplayName())
	}

	// Test IsAvailable
	originalLookPath := execLookPath
	defer func() { execLookPath = originalLookPath }()
	execLookPath = func(cmd string) (string, error) {
		if cmd == "fish" {
			return "/usr/bin/fish", nil
		}
		return "", exec.ErrNotFound
	}

	if !shell.IsAvailable() {
		t.Error("Expected fish to be available")
	}

	// Test ConfigFile
	originalUserHomeDir := userHomeDir
	defer func() { userHomeDir = originalUserHomeDir }()

	testHome := t.TempDir()
	userHomeDir = func() (string, error) {
		return testHome, nil
	}

	expected := filepath.Join(testHome, ".config", "fish", "config.fish")
	if shell.ConfigFile() != expected {
		t.Errorf("Expected %s, got %s", expected, shell.ConfigFile())
	}

	// Test PathCommand
	if shell.PathCommand("/usr/local/bin") != `fish_add_path -p "/usr/local/bin"` {
		t.Errorf("PathCommand output incorrect")
	}

	// Test SetupCommands
	commands := shell.SetupCommands("/usr/local/bin")
	if len(commands) == 0 {
		t.Error("SetupCommands should return commands")
	}
	if !strings.Contains(commands[0], "GOVMAN - Go Version Manager") {
		t.Error("SetupCommands should contain GOVMAN header")
	}

	// Test ExecutePathCommand
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Use current directory which exists
	err := shell.ExecutePathCommand(".")
	wOut.Close()
	wErr.Close()

	if err != nil {
		t.Errorf("ExecutePathCommand failed: %v", err)
	}

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)
	output := string(outBytes) + string(errBytes)

	if !strings.Contains(output, "fish_add_path") {
		t.Error("ExecutePathCommand should output PATH command")
	}

	os.Stdout = oldStdout
	os.Stderr = oldStderr
}

func TestPowerShell(t *testing.T) {
	shell := &PowerShell{}

	// Test Name
	if shell.Name() != "powershell" {
		t.Errorf("Expected 'powershell', got %s", shell.Name())
	}

	// Test DisplayName
	if shell.DisplayName() != "PowerShell" {
		t.Errorf("Expected 'PowerShell', got %s", shell.DisplayName())
	}

	// Test IsAvailable
	originalLookPath := execLookPath
	defer func() { execLookPath = originalLookPath }()
	execLookPath = func(cmd string) (string, error) {
		if cmd == "pwsh" || cmd == "powershell" {
			return "/usr/bin/" + cmd, nil
		}
		return "", exec.ErrNotFound
	}

	if !shell.IsAvailable() {
		t.Error("Expected PowerShell to be available")
	}

	// Test ConfigFile
	originalUserHomeDir := userHomeDir
	defer func() { userHomeDir = originalUserHomeDir }()

	testHome := "/Users/testuser"
	userHomeDir = func() (string, error) {
		return testHome, nil
	}

	expected := filepath.Join(testHome, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	if shell.ConfigFile() != expected {
		t.Errorf("Expected %s, got %s", expected, shell.ConfigFile())
	}

	// Test PathCommand
	if shell.PathCommand("/usr/local/bin") != `$env:PATH = "/usr/local/bin;" + $env:PATH` {
		t.Errorf("PathCommand output incorrect")
	}

	// Test SetupCommands
	commands := shell.SetupCommands("/usr/local/bin")
	if len(commands) == 0 {
		t.Error("SetupCommands should return commands")
	}
	if !strings.Contains(commands[0], "GOVMAN - Go Version Manager") {
		t.Error("SetupCommands should contain GOVMAN header")
	}

	// Test ExecutePathCommand
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Use current directory which exists
	err := shell.ExecutePathCommand(".")
	wOut.Close()
	wErr.Close()

	if err != nil {
		t.Errorf("ExecutePathCommand failed: %v", err)
	}

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)
	output := string(outBytes) + string(errBytes)

	if !strings.Contains(output, "$env:PATH") {
		t.Error("ExecutePathCommand should output PATH command")
	}

	os.Stdout = oldStdout
	os.Stderr = oldStderr
}

func TestCmdShell(t *testing.T) {
	shell := &CmdShell{}

	// Test Name
	if shell.Name() != "cmd" {
		t.Errorf("Expected 'cmd', got %s", shell.Name())
	}

	// Test DisplayName
	if shell.DisplayName() != "Command Prompt" {
		t.Errorf("Expected 'Command Prompt', got %s", shell.DisplayName())
	}

	// Test IsAvailable - should be available on Windows
	originalGOOS := currentGOOS
	defer func() { currentGOOS = originalGOOS }()

	currentGOOS = "windows"
	if !shell.IsAvailable() {
		t.Error("Expected cmd to be available on Windows")
	}

	currentGOOS = "linux"
	if shell.IsAvailable() {
		t.Error("Expected cmd to not be available on Linux")
	}

	// Test ConfigFile
	if shell.ConfigFile() != "Environment Variables (System Properties)" {
		t.Errorf("ConfigFile output incorrect")
	}

	// Test PathCommand
	if shell.PathCommand("/usr/local/bin") != `set PATH=/usr/local/bin;%PATH%` {
		t.Errorf("PathCommand output incorrect")
	}

	// Test SetupCommands
	commands := shell.SetupCommands("/usr/local/bin")
	if len(commands) == 0 {
		t.Error("SetupCommands should return commands")
	}
	// CmdShell uses REM for comments
	found := false
	for _, cmd := range commands {
		if strings.Contains(cmd, "GOVMAN - Go Version Manager") {
			found = true
			break
		}
	}
	if !found {
		t.Error("SetupCommands should contain GOVMAN header")
	}

	// Test ExecutePathCommand
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Use current directory which exists
	err := shell.ExecutePathCommand(".")
	wOut.Close()
	wErr.Close()

	if err != nil {
		t.Errorf("ExecutePathCommand failed: %v", err)
	}

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)
	output := string(outBytes) + string(errBytes)

	if !strings.Contains(output, "set PATH") {
		t.Error("ExecutePathCommand should output PATH command")
	}

	os.Stdout = oldStdout
	os.Stderr = oldStderr
}

func TestInitializeShell(t *testing.T) {
	testCases := []struct {
		name        string
		shellName   string
		expectError bool
	}{
		{
			name:        "Valid bash shell",
			shellName:   "bash",
			expectError: false,
		},
		{
			name:        "Valid zsh shell",
			shellName:   "zsh",
			expectError: false,
		},
		{
			name:        "Valid fish shell",
			shellName:   "fish",
			expectError: false,
		},
		{
			name:        "Valid powershell shell",
			shellName:   "powershell",
			expectError: false,
		},
		{
			name:        "Valid pwsh shell",
			shellName:   "pwsh",
			expectError: false,
		},
		{
			name:        "Invalid shell",
			shellName:   "invalid",
			expectError: false, // No longer expect error since we provide a default shell
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var shell Shell
			switch tc.shellName {
			case "bash":
				shell = &BashShell{}
			case "zsh":
				shell = &ZshShell{}
			case "fish":
				shell = &FishShell{}
			case "powershell", "pwsh":
				shell = &PowerShell{}
			default:
				shell = &BashShell{} // For invalid shell test, provide a valid shell to avoid nil pointer
			}

			// Use a temporary directory for config files to avoid conflicts.
			// Overriding HOME alone is not enough: userHomeDir is os.UserHomeDir,
			// which reads %USERPROFILE% on Windows, so every subtest would share
			// the runner's real profile and the second PowerShell subtest would
			// fail with "govman is already configured".
			tempDir := t.TempDir()
			originalUserHomeDir := userHomeDir
			t.Cleanup(func() { userHomeDir = originalUserHomeDir })
			userHomeDir = func() (string, error) { return tempDir, nil }
			t.Setenv("HOME", tempDir)
			t.Setenv("USERPROFILE", tempDir)

			err := InitializeShell(shell, tempDir, false)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestInitializeShellCmd(t *testing.T) {
	// Test InitializeShell with cmd shell (Windows)
	originalGOOS := currentGOOS
	defer func() { currentGOOS = originalGOOS }()

	// Mock Windows environment
	currentGOOS = "windows"

	shell := &CmdShell{}

	// Use a temporary directory to avoid conflicts
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "govman-real.exe"), []byte("test backend"), 0755); err != nil {
		t.Fatal(err)
	}

	err := InitializeShell(shell, tempDir, false)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	wrapperPath := filepath.Join(tempDir, "govman.cmd")
	if _, err := os.Stat(wrapperPath); os.IsNotExist(err) {
		t.Error("Expected wrapper batch file to be created")
	}
	content, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatal(err)
	}
	wrapper := string(content)
	for _, forbidden := range []string{"setlocal", "enabledelayedexpansion", `govman.exe`} {
		if strings.Contains(strings.ToLower(wrapper), forbidden) {
			t.Errorf("wrapper must not contain %q", forbidden)
		}
	}
	if count := strings.Count(wrapper, `"%GOVMAN_BIN%" %*`); count != 1 {
		t.Errorf("backend invocation count = %d, want 1", count)
	}
	for _, expected := range []string{"govman-real.exe", `if /i "%GOVMAN_COMMAND%"=="use"`, `if /i "%GOVMAN_COMMAND%"=="refresh"`, `%RANDOM%-%RANDOM%`} {
		if !strings.Contains(wrapper, expected) {
			t.Errorf("wrapper missing %q", expected)
		}
	}
}

func TestContainsGovmanConfig(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "Contains complete GOVMAN block",
			content:  "# GOVMAN - Go Version Manager\necho test\n# END GOVMAN",
			expected: true,
		},
		{
			name:     "Keyword without markers",
			content:  "govman_auto_switch() {\n  echo test\n}",
			expected: false,
		},
		{
			name:     "PowerShell keyword without markers",
			content:  "function Invoke-GovmanAutoSwitch {\n  Write-Host test\n}",
			expected: false,
		},
		{
			name:     "Fish keyword without markers",
			content:  "function __govman_cd_hook --on-variable PWD\n  echo test\nend",
			expected: false,
		},
		{
			name:     "Opening marker only",
			content:  "# GOVMAN - Go Version Manager\necho test",
			expected: false,
		},
		{
			name:     "No govman config",
			content:  "echo test\nexport PATH=/usr/bin:$PATH",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := containsGovmanConfig(tc.content)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestRemoveExistingConfig(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Remove complete govman block",
			input: `export PATH=/usr/bin:$PATH
# GOVMAN - Go Version Manager
export PATH="/usr/local/bin:$PATH"
export GOTOOLCHAIN=local
govman() {
  echo "test"
}
# END GOVMAN
export PS1="\$ "`,
			expected: `export PATH=/usr/bin:$PATH
export PS1="\$ "`,
		},
		{
			name:     "No govman config to remove",
			input:    "export PATH=/usr/bin:$PATH",
			expected: "export PATH=/usr/bin:$PATH",
		},
		{
			name: "Multiple consecutive newlines collapsed",
			input: `export PATH=/usr/bin:$PATH


# GOVMAN - Go Version Manager
export PATH="/usr/local/bin:$PATH"
# END GOVMAN


export PS1="\$ "`,
			expected: `export PATH=/usr/bin:$PATH

export PS1="\$ "`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := removeExistingConfig(tc.input)
			if result != tc.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tc.expected, result)
			}
		})
	}
}

func TestGetShellInstructions(t *testing.T) {
	testCases := []struct {
		name     string
		shell    Shell
		binPath  string
		expected string
	}{
		{
			name:     "Bash instructions",
			shell:    &BashShell{},
			binPath:  "/usr/local/bin",
			expected: "Manual setup for Bash:",
		},
		{
			name:     "Zsh instructions",
			shell:    &ZshShell{},
			binPath:  "/usr/local/bin",
			expected: "Manual setup for Zsh:",
		},
		{
			name:     "Fish instructions",
			shell:    &FishShell{},
			binPath:  "/usr/local/bin",
			expected: "Manual setup for Fish:",
		},
		{
			name:     "PowerShell instructions",
			shell:    &PowerShell{},
			binPath:  "/usr/local/bin",
			expected: "Manual setup for PowerShell:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetShellInstructions(tc.shell, tc.binPath)

			if !strings.Contains(result, tc.expected) {
				t.Errorf("Expected instructions to contain %q, got %q", tc.expected, result)
			}

			if !strings.Contains(result, tc.binPath) {
				t.Errorf("Expected instructions to contain binPath %q", tc.binPath)
			}

			// Should contain setup commands
			commands := tc.shell.SetupCommands(tc.binPath)
			for _, cmd := range commands {
				if !strings.Contains(result, cmd) {
					t.Errorf("Expected instructions to contain setup command %q", cmd)
				}
			}
		})
	}
}

func TestValidateBinPath(t *testing.T) {
	testCases := []struct {
		name        string
		binPath     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty path",
			binPath:     "",
			expectError: true,
			errorMsg:    "binary path cannot be empty",
		},
		{
			name:        "nonexistent path",
			binPath:     "/nonexistent/path",
			expectError: true,
			errorMsg:    "binary path does not exist",
		},
		{
			name:        "file instead of directory",
			binPath:     "shell_test.go", // This is a file, not a directory
			expectError: true,
			errorMsg:    "binary path is not a directory",
		},
		{
			name:        "valid directory",
			binPath:     ".", // Current directory exists and is a directory
			expectError: false,
		},
		{
			name:        "path with traversal",
			binPath:     "/tmp/../../../etc",
			expectError: true,
			errorMsg:    "invalid binary path (path traversal detected)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBinPath(tc.binPath)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValidateBinPathTraversal(t *testing.T) {
	testCases := []struct {
		name        string
		binPath     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "path with double dots",
			binPath:     "/tmp/../etc",
			expectError: true,
			errorMsg:    "path traversal detected",
		},
		{
			name:        "path with double dots in middle",
			binPath:     "/usr/local/../bin",
			expectError: true,
			errorMsg:    "path traversal detected",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBinPath(tc.binPath)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestDetectWithEmptyShell(t *testing.T) {
	// Mock currentGOOS
	originalGOOS := currentGOOS
	defer func() { currentGOOS = originalGOOS }()
	currentGOOS = "linux"

	// Mock SHELL environment variable to be empty
	originalShell := os.Getenv("SHELL")
	defer func() { os.Setenv("SHELL", originalShell) }()
	os.Setenv("SHELL", "")

	// Mock exec.LookPath to return bash available
	originalLookPath := execLookPath
	defer func() { execLookPath = originalLookPath }()
	execLookPath = func(cmd string) (string, error) {
		if cmd == "bash" {
			return "/bin/bash", nil
		}
		return "", exec.ErrNotFound
	}

	shell := Detect()
	if shell == nil {
		t.Fatal("Detect() returned nil")
	}

	if _, ok := shell.(*BashShell); !ok {
		t.Errorf("Expected BashShell when SHELL is empty, got %T", shell)
	}
}

func TestInitializeUnixShellReadError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mode 0000 only sets FILE_ATTRIBUTE_READONLY on Windows; reads still succeed so os.ReadFile cannot return permission denied")
	}

	shell := &BashShell{}
	tempDir := t.TempDir()

	// Create a config file but make it unreadable
	configFile := filepath.Join(tempDir, ".bashrc")
	if err := os.WriteFile(configFile, []byte("test"), 0000); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Mock userHomeDir to return our temp dir
	originalUserHomeDir := userHomeDir
	defer func() { userHomeDir = originalUserHomeDir }()
	userHomeDir = func() (string, error) {
		return tempDir, nil
	}

	// Try to initialize - should fail due to permission error
	err := initializeUnixShell(shell, tempDir, false)
	if err == nil {
		t.Fatal("Expected error due to permission denied reading config file")
	}

	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("Expected config file read error, got: %v", err)
	}
}

func TestInitializePowerShellReadError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mode 0000 only sets FILE_ATTRIBUTE_READONLY on Windows; the profile stays readable so os.ReadFile cannot fail")
	}

	shell := &PowerShell{}
	tempDir := t.TempDir()

	// Create a profile file but make it unreadable
	profileDir := filepath.Join(tempDir, "Documents", "WindowsPowerShell")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("Failed to create profile dir: %v", err)
	}

	profileFile := filepath.Join(profileDir, "Microsoft.PowerShell_profile.ps1")
	if err := os.WriteFile(profileFile, []byte("test"), 0000); err != nil {
		t.Fatalf("Failed to create profile file: %v", err)
	}

	// Mock userHomeDir and isCommandAvailable
	originalUserHomeDir := userHomeDir
	defer func() { userHomeDir = originalUserHomeDir }()
	userHomeDir = func() (string, error) {
		return tempDir, nil
	}

	originalLookPath := execLookPath
	defer func() { execLookPath = originalLookPath }()
	execLookPath = func(cmd string) (string, error) {
		if cmd == "powershell" {
			return "/bin/powershell", nil
		}
		return "", exec.ErrNotFound
	}

	// Try to initialize - should fail due to permission error
	err := initializePowerShell(shell, tempDir, false)
	if err == nil {
		t.Fatal("Expected error due to permission denied reading profile")
	}

	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("Expected profile read error, got: %v", err)
	}
}

func TestInitializeCmdShellTemplateErrors(t *testing.T) {
	tempDir := t.TempDir()

	// Test with a path that contains null byte - this should cause an error
	invalidPath := filepath.Join(tempDir, "path with\x00null")

	err := initializeCmdShell(invalidPath, false)
	if err == nil {
		t.Error("Expected error when creating wrapper with invalid path")
	}

	// The error might be from various sources (path validation, template parsing, etc.)
	// We just check that we get an error
	if err != nil {
		t.Logf("Got expected error: %v", err)
	}
}

func TestValidateBinPathEdgeCases(t *testing.T) {
	// Test with a path that contains special characters
	testCases := []struct {
		name        string
		binPath     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "path with spaces",
			binPath:     "/path with spaces",
			expectError: true,
			errorMsg:    "binary path does not exist",
		},
		{
			name:        "relative path",
			binPath:     "./relative/path",
			expectError: true,
			errorMsg:    "binary path does not exist",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBinPath(tc.binPath)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestDetectEdgeCases(t *testing.T) {
	testCases := []struct {
		name         string
		goos         string
		shellEnv     string
		mockCommands map[string]bool
		expectedType interface{}
	}{
		{
			name:         "Unix with sh in SHELL but bash not available",
			goos:         "linux",
			shellEnv:     "/bin/sh",
			mockCommands: map[string]bool{}, // bash not available
			expectedType: &BashShell{},      // fallback to first available (BashShell is always returned as fallback)
		},
		{
			name:         "Unix with zsh in SHELL but zsh not available",
			goos:         "linux",
			shellEnv:     "/usr/bin/zsh",
			mockCommands: map[string]bool{}, // zsh not available
			expectedType: &BashShell{},      // fallback to first available
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock currentGOOS
			originalGOOS := currentGOOS
			defer func() { currentGOOS = originalGOOS }()
			currentGOOS = tc.goos

			// Mock SHELL environment variable
			if tc.shellEnv != "" {
				originalShell := os.Getenv("SHELL")
				defer func() { os.Setenv("SHELL", originalShell) }()
				os.Setenv("SHELL", tc.shellEnv)
			}

			// Mock exec.LookPath
			originalLookPath := execLookPath
			defer func() { execLookPath = originalLookPath }()
			execLookPath = func(cmd string) (string, error) {
				if tc.mockCommands[cmd] {
					return "/usr/bin/" + cmd, nil
				}
				return "", exec.ErrNotFound
			}

			shell := Detect()

			if shell == nil {
				t.Fatal("Detect() returned nil")
			}

			// Check type
			switch tc.expectedType.(type) {
			case *BashShell:
				if _, ok := shell.(*BashShell); !ok {
					t.Errorf("Expected BashShell, got %T", shell)
				}
			case *ZshShell:
				if _, ok := shell.(*ZshShell); !ok {
					t.Errorf("Expected ZshShell, got %T", shell)
				}
			}
		})
	}
}

func TestDetectAvailableShellNoShells(t *testing.T) {
	// Mock exec.LookPath to return no shells
	originalLookPath := execLookPath
	defer func() { execLookPath = originalLookPath }()
	execLookPath = func(cmd string) (string, error) {
		return "", exec.ErrNotFound
	}

	shell := detectAvailableShell()

	// Should return BashShell as fallback
	if _, ok := shell.(*BashShell); !ok {
		t.Errorf("Expected BashShell as fallback, got %T", shell)
	}
}

func TestConfigFileErrors(t *testing.T) {
	testCases := []struct {
		name         string
		shell        Shell
		expectedFile string
	}{
		{
			name:         "BashShell with userHomeDir error",
			shell:        &BashShell{},
			expectedFile: ".bashrc",
		},
		{
			name:         "ZshShell with userHomeDir error",
			shell:        &ZshShell{},
			expectedFile: ".zshrc",
		},
		{
			name:         "FishShell with userHomeDir error",
			shell:        &FishShell{},
			expectedFile: "config.fish",
		},
		{
			name:         "PowerShell with userHomeDir error",
			shell:        &PowerShell{},
			expectedFile: "$PROFILE",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalUserHomeDir := userHomeDir
			defer func() { userHomeDir = originalUserHomeDir }()

			// Mock userHomeDir to return an error
			userHomeDir = func() (string, error) {
				return "", fmt.Errorf("mock error")
			}

			if tc.shell.ConfigFile() != tc.expectedFile {
				t.Errorf("Expected %s, got %s", tc.expectedFile, tc.shell.ConfigFile())
			}
		})
	}
}

func TestExecutePathCommandErrors(t *testing.T) {
	testCases := []struct {
		name  string
		shell Shell
	}{
		{
			name:  "BashShell",
			shell: &BashShell{},
		},
		{
			name:  "ZshShell",
			shell: &ZshShell{},
		},
		{
			name:  "FishShell",
			shell: &FishShell{},
		},
		{
			name:  "PowerShell",
			shell: &PowerShell{},
		},
		{
			name:  "CmdShell",
			shell: &CmdShell{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with invalid path
			err := tc.shell.ExecutePathCommand("")
			if err == nil {
				t.Error("Expected error for empty path")
			}
		})
	}
}

func TestInitializeShellErrors(t *testing.T) {
	testCases := []struct {
		name        string
		shell       Shell
		binPath     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Invalid bin path",
			shell:       &BashShell{},
			binPath:     "",
			expectError: true,
			errorMsg:    "binary path cannot be empty",
		},
		{
			name:        "Non-existent bin path",
			shell:       &BashShell{},
			binPath:     "/nonexistent/path",
			expectError: true,
			errorMsg:    "binary path does not exist",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := InitializeShell(tc.shell, tc.binPath, false)
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestInitializeShellForce(t *testing.T) {
	// Test force flag with existing config
	tempDir := t.TempDir()
	originalUserHomeDir := userHomeDir
	defer func() { userHomeDir = originalUserHomeDir }()

	userHomeDir = func() (string, error) {
		return tempDir, nil
	}

	shell := &BashShell{}

	// First initialization
	err := InitializeShell(shell, tempDir, false)
	if err != nil {
		t.Fatalf("First initialization failed: %v", err)
	}

	// Second initialization without force should fail
	err = InitializeShell(shell, tempDir, false)
	if err == nil {
		t.Error("Expected error for second initialization without force")
	}

	// Second initialization with force should succeed
	err = InitializeShell(shell, tempDir, true)
	if err != nil {
		t.Errorf("Second initialization with force failed: %v", err)
	}
}

func TestPowerShellConfigFileNoPwsh(t *testing.T) {
	originalUserHomeDir := userHomeDir
	defer func() { userHomeDir = originalUserHomeDir }()

	testHome := "/Users/testuser"
	userHomeDir = func() (string, error) {
		return testHome, nil
	}

	// Mock exec.LookPath to return false for pwsh
	originalLookPath := execLookPath
	defer func() { execLookPath = originalLookPath }()
	execLookPath = func(cmd string) (string, error) {
		if cmd == "pwsh" {
			return "", exec.ErrNotFound
		}
		if cmd == "powershell" {
			return "/usr/bin/powershell", nil
		}
		return "", exec.ErrNotFound
	}

	shell := &PowerShell{}
	expected := filepath.Join(testHome, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
	if shell.ConfigFile() != expected {
		t.Errorf("Expected %s, got %s", expected, shell.ConfigFile())
	}
}

func TestGetShellInstructionsCmd(t *testing.T) {
	// Test GetShellInstructions with CmdShell (not covered in the main test)
	shell := &CmdShell{}
	binPath := "/usr/local/bin"

	result := GetShellInstructions(shell, binPath)

	if !strings.Contains(result, "Manual setup for Command Prompt:") {
		t.Errorf("Expected instructions to contain 'Manual setup for Command Prompt:', got %q", result)
	}

	if !strings.Contains(result, binPath) {
		t.Errorf("Expected instructions to contain binPath %q", binPath)
	}

	// Should contain setup commands
	commands := shell.SetupCommands(binPath)
	for _, cmd := range commands {
		if !strings.Contains(result, cmd) {
			t.Errorf("Expected instructions to contain setup command %q", cmd)
		}
	}
}

func TestBashShellConfigFileWithExistingFiles(t *testing.T) {
	// Test BashShell.ConfigFile with existing files
	shell := &BashShell{}

	originalUserHomeDir := userHomeDir
	defer func() { userHomeDir = originalUserHomeDir }()

	tempDir := t.TempDir()
	userHomeDir = func() (string, error) {
		return tempDir, nil
	}

	// Test with .bash_profile existing (should be preferred over .profile)
	bashProfilePath := filepath.Join(tempDir, ".bash_profile")
	os.WriteFile(bashProfilePath, []byte("test"), 0644)
	defer os.Remove(bashProfilePath)

	if shell.ConfigFile() != bashProfilePath {
		t.Errorf("Expected %s, got %s", bashProfilePath, shell.ConfigFile())
	}

	// Test with only .profile existing
	os.Remove(bashProfilePath)
	profilePath := filepath.Join(tempDir, ".profile")
	os.WriteFile(profilePath, []byte("test"), 0644)
	defer os.Remove(profilePath)

	if shell.ConfigFile() != profilePath {
		t.Errorf("Expected %s, got %s", profilePath, shell.ConfigFile())
	}
}

func TestEscapeFunctions(t *testing.T) {
	// Test escape functions
	testCases := []struct {
		name     string
		input    string
		expected string
		function func(string) string
	}{
		{
			name:     "escapeBashPath with special chars",
			input:    `/path/with"special$chars\`,
			expected: `/path/with\"special\$chars\\`,
			function: escapeBashPath,
		},
		{
			name:     "escapeFishPath with special chars",
			input:    `/path/with"special$chars\`,
			expected: `/path/with\"special\$chars\\`,
			function: escapeFishPath,
		},
		{
			name:     "escapePowerShellPath with special chars",
			input:    "/path/with\"special$chars",
			expected: "/path/with`\"special`$chars",
			function: escapePowerShellPath,
		},
		{
			name:     "escapeCmdPath with percent",
			input:    `/path/with%percent`,
			expected: `/path/with%%percent`,
			function: escapeCmdPath,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.function(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestValidateBinPathMoreEdgeCases(t *testing.T) {
	// Test more edge cases for validateBinPath
	testCases := []struct {
		name        string
		binPath     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "path with symlinks",
			binPath:     ".",
			expectError: false,
		},
		{
			name:        "path that exists but is not a directory",
			binPath:     "shell_test.go",
			expectError: true,
			errorMsg:    "binary path is not a directory",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBinPath(tc.binPath)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestDetectMoreEdgeCases(t *testing.T) {
	// Test more edge cases for Detect
	testCases := []struct {
		name         string
		goos         string
		shellEnv     string
		mockCommands map[string]bool
		expectedType interface{}
	}{
		{
			name:         "Windows with neither pwsh nor powershell",
			goos:         "windows",
			mockCommands: map[string]bool{},
			expectedType: &CmdShell{},
		},
		{
			name:         "Unix with bash in SHELL path not available",
			goos:         "linux",
			shellEnv:     "/bin/bash",
			mockCommands: map[string]bool{"bash": false, "zsh": true},
			expectedType: &ZshShell{}, // fallback to next available
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock currentGOOS
			originalGOOS := currentGOOS
			defer func() { currentGOOS = originalGOOS }()
			currentGOOS = tc.goos

			// Mock SHELL environment variable
			if tc.shellEnv != "" {
				originalShell := os.Getenv("SHELL")
				defer func() { os.Setenv("SHELL", originalShell) }()
				os.Setenv("SHELL", tc.shellEnv)
			}

			// Mock exec.LookPath
			originalLookPath := execLookPath
			defer func() { execLookPath = originalLookPath }()
			execLookPath = func(cmd string) (string, error) {
				if tc.mockCommands[cmd] {
					return "/usr/bin/" + cmd, nil
				}
				return "", exec.ErrNotFound
			}

			shell := Detect()

			if shell == nil {
				t.Fatal("Detect() returned nil")
			}

			// Check type
			switch tc.expectedType.(type) {
			case *BashShell:
				if _, ok := shell.(*BashShell); !ok {
					t.Errorf("Expected BashShell, got %T", shell)
				}
			case *ZshShell:
				if _, ok := shell.(*ZshShell); !ok {
					t.Errorf("Expected ZshShell, got %T", shell)
				}
			case *FishShell:
				if _, ok := shell.(*FishShell); !ok {
					t.Errorf("Expected FishShell, got %T", shell)
				}
			case *PowerShell:
				if _, ok := shell.(*PowerShell); !ok {
					t.Errorf("Expected PowerShell, got %T", shell)
				}
			case *CmdShell:
				if _, ok := shell.(*CmdShell); !ok {
					t.Errorf("Expected CmdShell, got %T", shell)
				}
			}
		})
	}
}

// testInitializeUnixShell tests InitializeShell for a Unix shell (Bash, Zsh, Fish).
func testInitializeUnixShell(t *testing.T, shell Shell, existingCfg string, force bool, expectError bool, errorMsg string) {
	t.Helper()
	tempDir := t.TempDir()

	originalUserHomeDir := userHomeDir
	defer func() { userHomeDir = originalUserHomeDir }()
	userHomeDir = func() (string, error) {
		return tempDir, nil
	}

	configFile := filepath.Join(tempDir, ".bashrc")
	if _, ok := shell.(*ZshShell); ok {
		configFile = filepath.Join(tempDir, ".zshrc")
	} else if _, ok := shell.(*FishShell); ok {
		fishDir := filepath.Join(tempDir, ".config", "fish")
		os.MkdirAll(fishDir, 0755)
		configFile = filepath.Join(fishDir, "config.fish")
	}

	if existingCfg != "" {
		os.WriteFile(configFile, []byte(existingCfg), 0644)
	}

	err := InitializeShell(shell, tempDir, force)
	if expectError {
		if err == nil {
			t.Errorf("Expected error but got none")
		} else if !strings.Contains(err.Error(), errorMsg) {
			t.Errorf("Expected error containing %q, got %q", errorMsg, err.Error())
		}
	} else {
		if err != nil {
			t.Errorf("Expected no error but got: %v", err)
		}
	}
}

// testInitializePowerShell tests InitializeShell for PowerShell.
func testInitializePowerShell(t *testing.T, shell *PowerShell, existingCfg string, force bool, expectError bool, errorMsg string) {
	t.Helper()
	tempDir := t.TempDir()

	originalUserHomeDir := userHomeDir
	defer func() { userHomeDir = originalUserHomeDir }()
	userHomeDir = func() (string, error) {
		return tempDir, nil
	}

	// ConfigFile picks Documents/PowerShell when pwsh is on PATH and
	// Documents/WindowsPowerShell otherwise. Pin the flavour so the profile the
	// helper writes is always the one the code reads: every GitHub runner ships
	// PowerShell Core, so an unpinned lookup wrote the existing config where
	// InitializeShell never looked and the "already configured" error never fired.
	originalLookPath := execLookPath
	defer func() { execLookPath = originalLookPath }()
	execLookPath = func(cmd string) (string, error) {
		if cmd == "powershell" {
			return "/usr/bin/powershell", nil
		}
		return "", exec.ErrNotFound
	}

	profileFile := shell.ConfigFile()
	if err := os.MkdirAll(filepath.Dir(profileFile), 0755); err != nil {
		t.Fatalf("Failed to create profile dir: %v", err)
	}

	if existingCfg != "" {
		if err := os.WriteFile(profileFile, []byte(existingCfg), 0644); err != nil {
			t.Fatalf("Failed to write existing config: %v", err)
		}
	}

	err := InitializeShell(shell, tempDir, force)
	if expectError {
		if err == nil {
			t.Errorf("Expected error but got none")
		} else if !strings.Contains(err.Error(), errorMsg) {
			t.Errorf("Expected error containing %q, got %q", errorMsg, err.Error())
		}
	} else {
		if err != nil {
			t.Errorf("Expected no error but got: %v", err)
		}
	}
}

// testInitializeCmdShell tests InitializeShell for CMD.
func testInitializeCmdShell(t *testing.T, shell *CmdShell, force bool, expectError bool, errorMsg string) {
	t.Helper()
	tempDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, "govman-real.exe"), []byte("test backend"), 0755); err != nil {
		t.Fatal(err)
	}
	wrapperPath := filepath.Join(tempDir, "govman.cmd")
	os.WriteFile(wrapperPath, []byte("@echo off"), 0644)

	err := InitializeShell(shell, tempDir, force)
	if expectError {
		if err == nil {
			t.Errorf("Expected error but got none")
		} else if !strings.Contains(err.Error(), errorMsg) {
			t.Errorf("Expected error containing %q, got %q", errorMsg, err.Error())
		}
	} else {
		if err != nil {
			t.Errorf("Expected no error but got: %v", err)
		}
	}
}

func TestInitializeShellWithExistingConfig(t *testing.T) {
	// Test InitializeShell with existing configuration
	testCases := []struct {
		name        string
		shell       Shell
		existingCfg string
		force       bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Unix shell with existing config without force",
			shell:       &BashShell{},
			existingCfg: "# GOVMAN - Go Version Manager\nexport PATH=/test\n# END GOVMAN",
			force:       false,
			expectError: true,
			errorMsg:    "govman is already configured",
		},
		{
			name:        "Unix shell with existing config with force",
			shell:       &BashShell{},
			existingCfg: "# GOVMAN - Go Version Manager\nexport PATH=/test\n# END GOVMAN",
			force:       true,
			expectError: false,
		},
		{
			name:        "PowerShell with existing config without force",
			shell:       &PowerShell{},
			existingCfg: "# GOVMAN - Go Version Manager\n$env:PATH = \"/test\"\n# END GOVMAN",
			force:       false,
			expectError: true,
			errorMsg:    "govman is already configured",
		},
		{
			name:        "PowerShell with existing config with force",
			shell:       &PowerShell{},
			existingCfg: "# GOVMAN - Go Version Manager\n$env:PATH = \"/test\"\n# END GOVMAN",
			force:       true,
			expectError: false,
		},
		{
			name:        "CMD shell with existing wrapper without force",
			shell:       &CmdShell{},
			force:       false,
			expectError: true,
			errorMsg:    "wrapper already exists",
		},
		{
			name:        "CMD shell with existing wrapper with force",
			shell:       &CmdShell{},
			force:       true,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			switch s := tc.shell.(type) {
			case *BashShell, *ZshShell, *FishShell:
				testInitializeUnixShell(t, s, tc.existingCfg, tc.force, tc.expectError, tc.errorMsg)
			case *PowerShell:
				testInitializePowerShell(t, s, tc.existingCfg, tc.force, tc.expectError, tc.errorMsg)
			case *CmdShell:
				testInitializeCmdShell(t, s, tc.force, tc.expectError, tc.errorMsg)
			}
		})
	}
}

// TestAutoSwitchPartialVersionComparison verifies the fix for BUG-4:
// Shell auto-switch scripts must use compare_version to handle partial version
// matching (e.g. required "1.25" should match current "1.25.3").
func TestAutoSwitchPartialVersionComparison(t *testing.T) {
	testCases := []struct {
		name    string
		shell   Shell
		keyword string
	}{
		{"Bash", &BashShell{}, "compare_version"},
		{"Zsh", &ZshShell{}, "compare_version"},
		{"Fish", &FishShell{}, "compare_version"},
		{"PowerShell", &PowerShell{}, "compareVersion"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			commands := tc.shell.SetupCommands("/usr/local/bin")
			joined := strings.Join(commands, "\n")
			if !strings.Contains(joined, tc.keyword) {
				t.Errorf("%s SetupCommands should contain %q for partial version comparison", tc.name, tc.keyword)
			}
		})
	}
}

func writeMockGovman(t *testing.T, binDir, mode string) string {
	t.Helper()
	path := filepath.Join(binDir, "govman")
	script := `#!/usr/bin/env bash
printf '%s\n' "$*" >> "$GOVMAN_TEST_LOG"
if [[ "` + mode + `" == "missing-path" ]]; then
    echo "command completed without a path"
else
    echo 'export PATH="/managed/go/bin:$PATH"'
fi
`
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGeneratedBashWrapperExecutesOnce(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}
	for _, mode := range []string{"valid", "missing-path"} {
		t.Run(mode, func(t *testing.T) {
			binDir := t.TempDir()
			writeMockGovman(t, binDir, mode)
			logPath := filepath.Join(t.TempDir(), "calls.log")
			shell := &BashShell{options: &IntegrationOptions{BinPath: binDir, AutoSwitch: false}}
			script := strings.Join(shell.SetupCommands(binDir), "\n") + "\n" +
				`govman --verbose --config "/config path/config.yaml" use 1.25` + "\n"
			command := exec.Command("bash", "--noprofile", "--norc", "-c", script)
			command.Env = append(os.Environ(), "GOVMAN_TEST_LOG="+logPath)
			output, err := command.CombinedOutput()
			if mode == "valid" && err != nil {
				t.Fatalf("generated wrapper failed: %v\n%s", err, output)
			}
			if mode == "missing-path" && err == nil {
				t.Fatalf("wrapper accepted success without a PATH command: %s", output)
			}
			calls, readErr := os.ReadFile(logPath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			lines := strings.Split(strings.TrimSpace(string(calls)), "\n")
			if len(lines) != 1 {
				t.Fatalf("govman invocation count = %d, want 1; calls=%q", len(lines), calls)
			}
			if !strings.Contains(lines[0], "use 1.25") {
				t.Fatalf("global flags prevented use detection: %q", lines[0])
			}
		})
	}
}

func TestGeneratedZshWrapperExecutesOnce(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh is not available")
	}
	binDir := t.TempDir()
	writeMockGovman(t, binDir, "valid")
	logPath := filepath.Join(t.TempDir(), "calls.log")
	shell := &ZshShell{options: &IntegrationOptions{BinPath: binDir, AutoSwitch: false}}
	script := strings.Join(shell.SetupCommands(binDir), "\n") + "\n" +
		`govman --verbose --config "/config path/config.yaml" use 1.25` + "\n"
	command := exec.Command("zsh", "-f", "-c", script)
	command.Env = append(os.Environ(), "GOVMAN_TEST_LOG="+logPath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated Zsh wrapper failed: %v\n%s", err, output)
	}
	calls, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.Split(strings.TrimSpace(string(calls)), "\n")) != 1 {
		t.Fatalf("Zsh wrapper invoked govman more than once: %q", calls)
	}
}

func TestGeneratedBashAutoSwitchAndDefaultRestore(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}
	for _, requiredVersion := range []string{"1.25", "1.25.4", "1.26rc1", "stable", "latest"} {
		t.Run(requiredVersion, func(t *testing.T) {
			root := t.TempDir()
			binDir := filepath.Join(root, "bin")
			projectDir := filepath.Join(root, "project")
			outsideDir := filepath.Join(root, "outside")
			installDir := filepath.Join(root, "versions")
			for _, directory := range []string{binDir, projectDir, outsideDir, installDir} {
				if err := os.MkdirAll(directory, 0755); err != nil {
					t.Fatal(err)
				}
			}
			writeMockGovman(t, binDir, "valid")
			if err := os.WriteFile(filepath.Join(projectDir, ".custom-go-version"), []byte(requiredVersion+"\n"), 0644); err != nil {
				t.Fatal(err)
			}
			logPath := filepath.Join(root, "calls.log")
			shell := &BashShell{options: &IntegrationOptions{
				BinPath:        binDir,
				ProjectFile:    ".custom-go-version",
				InstallDir:     installDir,
				DefaultVersion: "1.24.1",
				AutoSwitch:     true,
			}}
			script := `cd "` + projectDir + `"` + "\n" + strings.Join(shell.SetupCommands(binDir), "\n") + "\n" +
				`cd "` + outsideDir + `"` + "\n" + `govman_auto_switch` + "\n"
			command := exec.Command("bash", "--noprofile", "--norc", "-c", script)
			command.Env = append(os.Environ(), "GOVMAN_TEST_LOG="+logPath)
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("generated auto-switch failed: %v\n%s", err, output)
			}
			calls, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatal(err)
			}
			callText := string(calls)
			if !strings.Contains(callText, "use "+requiredVersion) {
				t.Fatalf("custom project version was not activated: %q", callText)
			}
			if !strings.Contains(callText, "use default") {
				t.Fatalf("default was not restored after leaving project: %q", callText)
			}
		})
	}
}

func TestConfiguredShellUsesEffectiveConfig(t *testing.T) {
	options := IntegrationOptions{
		BinPath:        "/opt/govman/bin",
		ConfigPath:     "/config path/custom.yaml",
		ProjectFile:    ".custom-version",
		InstallDir:     "/toolchains/go versions",
		DefaultVersion: "1.25.4",
		AutoSwitch:     true,
	}
	for _, shell := range []Shell{&BashShell{}, &ZshShell{}, &FishShell{}, &PowerShell{}} {
		Configure(shell, options)
		generated := strings.Join(shell.SetupCommands(options.BinPath), "\n")
		for _, expected := range []string{"custom.yaml", ".custom-version", "go versions"} {
			if !strings.Contains(generated, expected) {
				t.Errorf("%s integration omitted effective %q", shell.Name(), expected)
			}
		}
	}
}

func TestShellConfigMarkerSafetyAndAtomicProperties(t *testing.T) {
	t.Run("malformed marker is preserved", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ".bashrc")
		original := "before\n# GOVMAN - Go Version Manager\nuser content\n"
		if err := os.WriteFile(path, []byte(original), 0600); err != nil {
			t.Fatal(err)
		}
		err := writeShellIntegration(path, (&BashShell{}).SetupCommands(t.TempDir()), true, "\n")
		if err == nil || !strings.Contains(err.Error(), "malformed") {
			t.Fatalf("malformed marker error = %v", err)
		}
		data, _ := os.ReadFile(path)
		if string(data) != original {
			t.Fatalf("malformed config was modified: %q", data)
		}
	})

	t.Run("mode and CRLF are preserved", func(t *testing.T) {
		directory := t.TempDir()
		path := filepath.Join(directory, "profile.ps1")
		original := "user-line\r\n"
		if err := os.WriteFile(path, []byte(original), 0600); err != nil {
			t.Fatal(err)
		}
		commands := (&PowerShell{}).SetupCommands(directory)
		if err := writeShellIntegration(path, commands, false, "\r\n"); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ReplaceAll(string(data), "\r\n", ""), "\n") {
			t.Fatalf("mixed newline style in PowerShell profile")
		}
		// Windows synthesizes the mode from FILE_ATTRIBUTE_READONLY and can only
		// report 0444 or 0666 (os/types_windows.go), so 0600 is unreachable there.
		if runtime.GOOS != "windows" {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0600 {
				t.Fatalf("profile mode = %v", info.Mode().Perm())
			}
		}
		backups, err := filepath.Glob(filepath.Join(directory, ".govman-backup-*"))
		if err != nil || len(backups) != 0 {
			t.Fatalf("shell config backup was not cleaned: %v, err=%v", backups, err)
		}
	})

	t.Run("symlink config is rejected", func(t *testing.T) {
		directory := t.TempDir()
		target := filepath.Join(directory, "target")
		link := filepath.Join(directory, ".bashrc")
		if err := os.WriteFile(target, []byte("preserve"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if err := writeShellIntegration(link, (&BashShell{}).SetupCommands(directory), false, "\n"); err == nil {
			t.Fatal("symlink shell config was replaced")
		}
		data, _ := os.ReadFile(target)
		if string(data) != "preserve" {
			t.Fatalf("symlink target changed: %q", data)
		}
	})
}
