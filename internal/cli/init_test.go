package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	_config "github.com/justjundana/govman/internal/config"
	_logger "github.com/justjundana/govman/internal/logger"
	_shell "github.com/justjundana/govman/internal/shell"
)

// TestGetShellByName_Cmd verifies the fix for CQ-3:
// getShellByName must recognize "cmd" and return a CmdShell.
func TestGetShellByName_Cmd(t *testing.T) {
	testCases := []struct {
		name     string
		expected bool
	}{
		{"bash", true},
		{"zsh", true},
		{"fish", true},
		{"powershell", true},
		{"pwsh", true},
		{"cmd", true},
		{"unknown", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getShellByName(tc.name)
			if tc.expected && result == nil {
				t.Errorf("getShellByName(%q) returned nil, expected a Shell", tc.name)
			}
			if !tc.expected && result != nil {
				t.Errorf("getShellByName(%q) returned non-nil, expected nil", tc.name)
			}
		})
	}

	// Verify "cmd" returns CmdShell specifically
	shell := getShellByName("cmd")
	if _, ok := shell.(*_shell.CmdShell); !ok {
		t.Errorf("getShellByName(\"cmd\") should return *CmdShell, got %T", shell)
	}
}

func TestApplyOutputFlags(t *testing.T) {
	tests := []struct {
		name                         string
		initialQuiet, initialVerbose bool
		quiet, quietChanged          bool
		verbose, verboseChanged      bool
		wantQuiet, wantVerbose       bool
		wantError                    bool
	}{
		{"file values", true, false, false, false, false, false, true, false, false},
		{"verbose overrides quiet file", true, false, false, false, true, true, false, true, false},
		{"quiet overrides verbose file", false, true, true, true, false, false, true, false, false},
		{"explicit false", true, false, false, true, false, false, false, false, false},
		{"conflicting flags", false, false, true, true, true, true, false, false, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := &_config.Config{Quiet: test.initialQuiet, Verbose: test.initialVerbose}
			err := applyOutputFlags(config, test.quiet, test.quietChanged, test.verbose, test.verboseChanged)
			if (err != nil) != test.wantError {
				t.Fatalf("applyOutputFlags() error = %v", err)
			}
			if !test.wantError && (config.Quiet != test.wantQuiet || config.Verbose != test.wantVerbose) {
				t.Fatalf("effective output flags = quiet:%t verbose:%t", config.Quiet, config.Verbose)
			}
		})
	}
}

func TestInitConfigDoesNotCacheFailureOrPreviousFile(t *testing.T) {
	home := t.TempDir()
	homeKey := "HOME"
	if runtime.GOOS == "windows" {
		homeKey = "USERPROFILE"
	}
	t.Setenv(homeKey, home)

	originalFile, originalQuiet, originalVerbose, originalConfig := cfgFile, quietFlag, verboseFlag, cfg
	t.Cleanup(func() {
		cfgFile, quietFlag, verboseFlag, cfg = originalFile, originalQuiet, originalVerbose, originalConfig
		_logger.Get().SetLevel(_logger.NormalLevel)
	})
	quietFlag, verboseFlag = false, false

	invalidPath := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(invalidPath, []byte("invalid: ["), 0600); err != nil {
		t.Fatal(err)
	}
	cfgFile = invalidPath
	if err := initConfig(false, false); err == nil || cfg != nil {
		t.Fatalf("failed config load was cached incorrectly: cfg=%v err=%v", cfg, err)
	}

	quietPath := filepath.Join(t.TempDir(), "quiet.yaml")
	if err := os.WriteFile(quietPath, []byte("quiet: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfgFile = quietPath
	if err := initConfig(false, false); err != nil {
		t.Fatal(err)
	}
	if cfg == nil || !cfg.Quiet || _logger.Get().Level() != _logger.QuietLevel {
		t.Fatalf("quiet config was not applied after failure: cfg=%+v level=%v", cfg, _logger.Get().Level())
	}

	verbosePath := filepath.Join(t.TempDir(), "verbose.yaml")
	if err := os.WriteFile(verbosePath, []byte("verbose: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfgFile = verbosePath
	if err := initConfig(false, false); err != nil {
		t.Fatal(err)
	}
	if cfg == nil || !cfg.Verbose || cfg.Quiet || cfg.ConfigPath() != verbosePath || _logger.Get().Level() != _logger.VerboseLevel {
		t.Fatalf("second config leaked first config state: cfg=%+v level=%v", cfg, _logger.Get().Level())
	}
}
