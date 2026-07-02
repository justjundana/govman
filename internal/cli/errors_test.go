package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cobra "github.com/spf13/cobra"

	_config "github.com/justjundana/govman/internal/config"
)

func TestRenderCommandErrorOwnsOutput(t *testing.T) {
	root := &cobra.Command{Use: "govman"}
	runtimeErr := withHelp(errors.New("network failed"), "Check the configured endpoint.")
	var output bytes.Buffer
	renderCommandError(&output, root, runtimeErr)

	if count := strings.Count(output.String(), "Error:"); count != 1 {
		t.Fatalf("error rendered %d times: %q", count, output.String())
	}
	if !strings.Contains(output.String(), "Help: Check the configured endpoint.") {
		t.Fatalf("help missing from %q", output.String())
	}
	if strings.Contains(output.String(), "Usage:") {
		t.Fatalf("runtime error unexpectedly rendered usage: %q", output.String())
	}
}

func TestRenderBannerDisablesANSIForNonTTY(t *testing.T) {
	var plain bytes.Buffer
	renderBanner(&plain, false)
	if strings.Contains(plain.String(), "\x1b[") {
		t.Fatalf("plain banner contains ANSI: %q", plain.String())
	}

	var colored bytes.Buffer
	renderBanner(&colored, true)
	if !strings.Contains(colored.String(), "\x1b[") {
		t.Fatal("interactive banner is missing ANSI styling")
	}
}

func TestUsageErrorsRenderUsage(t *testing.T) {
	command := &cobra.Command{Use: "info <version>"}
	err := usageArgs(cobra.ExactArgs(1))(command, nil)
	var output bytes.Buffer
	renderCommandError(&output, command, err)
	if !strings.Contains(output.String(), "Usage:") {
		t.Fatalf("argument error did not render usage: %q", output.String())
	}
}

func TestListRejectsRemoteOnlyFlags(t *testing.T) {
	for _, flag := range []string{"--beta", "--pattern=1.25*"} {
		t.Run(flag, func(t *testing.T) {
			command := newListCmd()
			command.SilenceErrors = true
			command.SilenceUsage = true
			command.SetArgs([]string{flag})
			err := command.Execute()
			var typed *commandError
			if err == nil || !errors.As(err, &typed) || !typed.usage {
				t.Fatalf("expected usage error for %s, got %v", flag, err)
			}
		})
	}
}

func TestListRejectsInvalidGlobBeforeFetching(t *testing.T) {
	err := listRemoteVersions(nil, false, "[")
	if err == nil || !strings.Contains(err.Error(), "invalid version pattern") {
		t.Fatalf("expected invalid glob error, got %v", err)
	}
}

func TestBatchCommandsRejectInvalidGlobBeforeLookup(t *testing.T) {
	if _, err := expandInstallPatterns([]string{"1.*["}, nil, false); err == nil || !strings.Contains(err.Error(), "invalid version pattern") {
		t.Fatalf("install pattern error = %v", err)
	}
	if _, err := expandUninstallPatterns([]string{"1.*["}, nil); err == nil || !strings.Contains(err.Error(), "invalid version pattern") {
		t.Fatalf("uninstall pattern error = %v", err)
	}
}

func TestInstalledVersionResolutionIsExactAndConsistent(t *testing.T) {
	installed := []string{"1.25.3", "1.24.9"}
	tests := []struct {
		requested string
		want      string
	}{
		{"latest", "1.25.3"},
		{"stable", "1.25.3"},
		{"1.25", "1.25.3"},
		{"1.25.2", "1.25.2"},
	}
	for _, test := range tests {
		resolved, err := resolveInstalledVersionFromList(test.requested, installed)
		if err != nil {
			t.Fatalf("resolve %s: %v", test.requested, err)
		}
		if resolved != test.want {
			t.Fatalf("resolve %s = %s, want %s", test.requested, resolved, test.want)
		}
	}
}

func TestRefreshPropagatesProjectFileReadError(t *testing.T) {
	directory := t.TempDir()
	originalConfig := cfg
	cfg = &_config.Config{
		InstallDir: filepath.Join(directory, "versions"),
		CacheDir:   filepath.Join(directory, "cache"),
		AutoSwitch: _config.AutoSwitchConfig{ProjectFile: directory},
	}
	t.Cleanup(func() { cfg = originalConfig })

	command := newRefreshCmd()
	command.SilenceErrors = true
	command.SilenceUsage = true
	command.SetArgs(nil)
	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "failed to read local version file") {
		t.Fatalf("expected project-file read error, got %v", err)
	}
	if !errors.Is(err, os.ErrInvalid) && !strings.Contains(strings.ToLower(err.Error()), "directory") {
		t.Logf("platform returned a non-directory-specific read error: %v", err)
	}
}
