package cli

import (
	"testing"

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
