//go:build !windows

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	_config "github.com/justjundana/govman/internal/config"
	_golang "github.com/justjundana/govman/internal/golang"
	_logger "github.com/justjundana/govman/internal/logger"
	_manager "github.com/justjundana/govman/internal/manager"
	_version "github.com/justjundana/govman/internal/version"
)

func setupCLICommandTest(t *testing.T) (*_config.Config, *bytes.Buffer) {
	t.Helper()
	home := t.TempDir()
	project := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/bash")
	t.Setenv("PATH", "")

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDirectory) })

	configFile := filepath.Join(home, ".govman", "config.yaml")
	loaded, err := _config.Load(configFile)
	if err != nil {
		t.Fatal(err)
	}
	loaded.InstallDir = filepath.Join(home, ".govman", "versions")
	loaded.CacheDir = filepath.Join(home, ".govman", "cache")
	loaded.DefaultVersion = ""
	loaded.AutoSwitch.Enabled = true
	loaded.AutoSwitch.ProjectFile = ".govman-goversion"
	loaded.Download.Timeout = 5 * time.Second
	loaded.Download.RetryCount = 1
	loaded.Download.RetryDelay = time.Millisecond
	loaded.GoReleases.CacheExpiry = 0
	if err := os.MkdirAll(loaded.InstallDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(loaded.CacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(loaded.GetBinPath(), 0755); err != nil {
		t.Fatal(err)
	}

	originalConfig := cfg
	cfg = loaded
	t.Cleanup(func() { cfg = originalConfig })

	log := _logger.Get()
	originalLevel := log.Level()
	originalNormal := log.NormalWriter()
	originalVerbose := log.VerboseWriter()
	output := &bytes.Buffer{}
	log.SetLevel(_logger.VerboseLevel)
	log.SetNormalWriter(output)
	log.SetVerboseWriter(output)
	t.Cleanup(func() {
		log.SetLevel(originalLevel)
		log.SetNormalWriter(originalNormal)
		log.SetVerboseWriter(originalVerbose)
	})

	_golang.ClearReleasesCache()
	t.Cleanup(_golang.ClearReleasesCache)
	return loaded, output
}

func createCLIInstalledVersion(t *testing.T, config *_config.Config, version string, installedAt time.Time) string {
	t.Helper()
	binDirectory := filepath.Join(config.InstallDir, "go"+version, "bin")
	if err := os.MkdirAll(binDirectory, 0755); err != nil {
		t.Fatal(err)
	}
	goBinary := filepath.Join(binDirectory, "go")
	script := fmt.Sprintf("#!/bin/sh\nprintf 'go version go%s test/amd64\\n'\n", version)
	if err := os.WriteFile(goBinary, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDirectory, "gofmt"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := _golang.WriteInstallMetadata(filepath.Dir(binDirectory), version, installedAt); err != nil {
		t.Fatal(err)
	}
	return goBinary
}

func executeStandaloneCommand(command *cobra.Command, args ...string) error {
	command.SilenceErrors = true
	command.SilenceUsage = true
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	return command.Execute()
}

func releaseServer(t *testing.T, payload string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(writer, payload)
	}))
}

func TestCLIListInfoAndCurrentCommands(t *testing.T) {
	config, output := setupCLICommandTest(t)
	if err := executeStandaloneCommand(newListCmd()); err != nil {
		t.Fatalf("empty list: %v", err)
	}

	old := time.Now().Add(-200 * 24 * time.Hour)
	createCLIInstalledVersion(t, config, "1.25.4", old)
	activeBinary := createCLIInstalledVersion(t, config, "1.24.3", time.Now())
	if err := executeStandaloneCommand(newListCmd()); err != nil {
		t.Fatalf("installed list: %v", err)
	}
	if !strings.Contains(output.String(), "1.25.4") {
		t.Fatalf("list output did not include installed version: %q", output.String())
	}

	if err := executeStandaloneCommand(newInfoCmd(), "1.25"); err != nil {
		t.Fatalf("partial info: %v", err)
	}
	if !strings.Contains(output.String(), "over 6 months old") {
		t.Fatalf("old installation warning missing: %q", output.String())
	}

	t.Setenv("PATH", filepath.Dir(activeBinary))
	if err := executeStandaloneCommand(newCurrentCmd()); err != nil {
		t.Fatalf("current command: %v", err)
	}
	if !strings.Contains(output.String(), "Go 1.24.3") {
		t.Fatalf("current output missing active version: %q", output.String())
	}

	server := releaseServer(t, `[
        {"version":"go1.26rc1","stable":false,"files":[]},
        {"version":"go1.25.5","stable":true,"files":[]},
        {"version":"go1.25.4","stable":true,"files":[]}
    ]`)
	defer server.Close()
	config.GoReleases.APIURL = server.URL
	if err := executeStandaloneCommand(newListCmd(), "--remote", "--beta", "--pattern", "1.26"); err != nil {
		t.Fatalf("remote list: %v", err)
	}
	if !strings.Contains(output.String(), "release candidate") {
		t.Fatalf("remote list did not label prerelease: %q", output.String())
	}
	if err := executeStandaloneCommand(newListCmd(), "--pattern", "1.25"); err == nil {
		t.Fatal("list accepted --pattern without --remote")
	}
}

func TestCLIUseRefreshInitAndCleanCommands(t *testing.T) {
	config, output := setupCLICommandTest(t)
	createCLIInstalledVersion(t, config, "1.25.4", time.Now())

	if err := executeStandaloneCommand(newUseCmd(), "1.25", "--local"); err != nil {
		t.Fatalf("local use: %v", err)
	}
	data, err := os.ReadFile(config.AutoSwitch.ProjectFile)
	if err != nil || strings.TrimSpace(string(data)) != "1.25.4" {
		t.Fatalf("local version file=%q err=%v", data, err)
	}
	if err := executeStandaloneCommand(newRefreshCmd()); err != nil {
		t.Fatalf("refresh local: %v", err)
	}

	config.DefaultVersion = "1.25.4"
	if err := os.WriteFile(config.AutoSwitch.ProjectFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := executeStandaloneCommand(newRefreshCmd()); err != nil {
		t.Fatalf("refresh empty local file: %v", err)
	}
	if err := os.Remove(config.AutoSwitch.ProjectFile); err != nil {
		t.Fatal(err)
	}
	if err := executeStandaloneCommand(newRefreshCmd()); err != nil {
		t.Fatalf("refresh default: %v", err)
	}

	cacheFile := filepath.Join(config.CacheDir, "archive.tar.gz")
	if err := os.WriteFile(cacheFile, []byte("cache"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := executeStandaloneCommand(newCleanCmd()); err != nil {
		t.Fatalf("clean command: %v", err)
	}
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		t.Fatalf("cache file still exists: %v", err)
	}

	if err := executeStandaloneCommand(newInitCmd(), "--shell", "bash", "--force"); err != nil {
		t.Fatalf("init bash: %v", err)
	}
	bashConfig, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".bashrc"))
	if err != nil || !bytes.Contains(bashConfig, []byte("# GOVMAN - Go Version Manager")) || !bytes.Contains(bashConfig, []byte("# END GOVMAN")) {
		t.Fatalf("bash integration missing: err=%v content=%q", err, bashConfig)
	}
	if err := executeStandaloneCommand(newInitCmd(), "--shell", "unsupported"); err == nil {
		t.Fatal("init accepted unsupported shell")
	}

	if !strings.Contains(output.String(), config.GetBinPath()) {
		t.Fatalf("init output did not include govman bin path: %q", output.String())
	}
}

func TestCLIUninstallAndPruneCommands(t *testing.T) {
	config, output := setupCLICommandTest(t)
	createCLIInstalledVersion(t, config, "1.25.4", time.Now())
	createCLIInstalledVersion(t, config, "1.24.3", time.Now())
	createCLIInstalledVersion(t, config, "1.23.9", time.Now())

	if err := executeStandaloneCommand(newUninstallCmd(), "1.23.9", "--yes"); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if _, err := os.Stat(filepath.Join(config.InstallDir, "go1.23.9")); !os.IsNotExist(err) {
		t.Fatalf("uninstall left version directory: %v", err)
	}
	createCLIInstalledVersion(t, config, "1.23.9", time.Now())

	config.DefaultVersion = "1.25.4"
	if err := os.WriteFile(config.AutoSwitch.ProjectFile, []byte("1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := executeStandaloneCommand(newPruneCmd(), "--yes"); err != nil {
		t.Fatalf("prune: %v", err)
	}
	if _, err := os.Stat(filepath.Join(config.InstallDir, "go1.23.9")); !os.IsNotExist(err) {
		t.Fatalf("prune left unused version: %v", err)
	}
	if err := executeStandaloneCommand(newPruneCmd(), "--yes"); err != nil {
		t.Fatalf("second prune: %v", err)
	}

	failure := fmt.Errorf("injected failure")
	if err := reportPruneResults(nil, 0, []error{failure}, map[string]string{}); err == nil {
		t.Fatal("prune report swallowed removal failure")
	}
	keys := sortedVersionKeys(map[string]string{"1.25.4": "default", "1.24.3": "local"})
	if strings.Join(keys, ",") != "1.24.3,1.25.4" {
		t.Fatalf("protected versions are not sorted: %v", keys)
	}
	if !strings.Contains(output.String(), "Successfully pruned") {
		t.Fatalf("prune summary missing: %q", output.String())
	}
}

func TestCLIInstallExpansionAndFailurePaths(t *testing.T) {
	config, _ := setupCLICommandTest(t)
	server := releaseServer(t, `[
        {"version":"go1.26rc1","stable":false,"files":[]},
        {"version":"go1.25.4","stable":true,"files":[]},
        {"version":"go1.25.3","stable":true,"files":[]}
    ]`)
	defer server.Close()
	config.GoReleases.APIURL = server.URL
	mgr := _manager.New(config)

	expanded, err := expandInstallPatterns([]string{"1.25.*", "1.25.4", "1.26*"}, mgr, true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(expanded, ",") != "1.25.4,1.25.3,1.26rc1" {
		t.Fatalf("expanded versions=%v", expanded)
	}
	if _, err := expandInstallPatterns([]string{"[*"}, mgr, false); err == nil {
		t.Fatal("invalid install glob was accepted")
	}
	if !hasWildcardPattern([]string{"1.25", "1.26*"}) || hasWildcardPattern([]string{"1.25.4"}) {
		t.Fatal("wildcard detection is incorrect")
	}

	successful, failures := installVersions(mgr, []string{"invalid"})
	if len(successful) != 0 || len(failures) != 1 {
		t.Fatalf("install failure result: success=%v failures=%v", successful, failures)
	}
	if err := executeStandaloneCommand(newInstallCmd(), "invalid"); err == nil {
		t.Fatal("install command swallowed invalid version")
	}
	config.Quiet = true
	if err := executeStandaloneCommand(newInstallCmd(), "1.25.*"); err == nil || !strings.Contains(err.Error(), "requires --yes") {
		t.Fatalf("quiet wildcard confirmation error=%v", err)
	}
}

func TestCLIConfirmationInput(t *testing.T) {
	originalStdin := os.Stdin
	t.Cleanup(func() { os.Stdin = originalStdin })

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = reader
	if _, err := writer.WriteString("yes\n"); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	if !confirmAction("Proceed?") {
		t.Fatal("yes confirmation was rejected")
	}
	_ = reader.Close()

	reader, writer, err = os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = reader
	_ = writer.Close()
	if confirmAction("Proceed?") {
		t.Fatal("EOF confirmation was accepted")
	}
	_ = reader.Close()
}

func TestCLISelfUpdateCheckWorkflowAndHelpers(t *testing.T) {
	setupCLICommandTest(t)
	resetSelfUpdateTestState(t)
	_version.Version = "v1.3.4"
	_version.Commit = "test"

	server := releaseServer(t, `{
        "tag_name":"v1.3.5",
        "prerelease":false,
        "draft":false,
        "body":"reliability fixes",
        "published_at":"2026-07-03T00:00:00Z",
        "assets":[]
    }`)
	defer server.Close()
	cfg.SelfUpdate.GitHubAPIURL = server.URL
	cfg.SelfUpdate.GitHubReleasesURL = server.URL
	if err := runSelfUpdateContext(context.Background(), true, false, false); err != nil {
		t.Fatalf("self-update check: %v", err)
	}

	prereleaseServer := releaseServer(t, `[{
        "tag_name":"v1.4.0-rc1",
        "prerelease":true,
        "draft":false,
        "assets":[]
    }]`)
	defer prereleaseServer.Close()
	cfg.SelfUpdate.GitHubReleasesURL = prereleaseServer.URL
	if release, err := getLatestRelease(true); err != nil || release.TagName != "v1.4.0-rc1" {
		t.Fatalf("prerelease lookup=%v err=%v", release, err)
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte("binary")))
	manifestServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = fmt.Fprintf(writer, "%s  %s\n", hash, platformAssetName())
	}))
	defer manifestServer.Close()
	release := &GitHubRelease{Assets: []GitHubAsset{{Name: "checksums.txt", DownloadURL: manifestServer.URL}}}
	if checksum, err := downloadChecksumManifest(context.Background(), release, platformAssetName()); err != nil || checksum != hash {
		t.Fatalf("manifest checksum=%q err=%v", checksum, err)
	}
	if _, err := findAssetURL(&GitHubRelease{Assets: []GitHubAsset{{Name: platformAssetName(), DownloadURL: "https://example.test/binary"}}}); err != nil {
		t.Fatalf("findAssetURL: %v", err)
	}

	directory := t.TempDir()
	current := filepath.Join(directory, "govman")
	backup := current + ".bak.1"
	oldBackup := current + ".bak.2"
	for _, path := range []string{backup, oldBackup} {
		if err := os.WriteFile(path, []byte("backup"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cleanupBackupFiles(current, backup)
	for _, path := range []string{backup, oldBackup} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("backup cleanup left %s: %v", path, err)
		}
	}

	_version.Version = "dev"
	if err := runSelfUpdate(false, false, false); err != nil {
		t.Fatalf("runSelfUpdate dev wrapper: %v", err)
	}
	if err := startWindowsUpdateHelper("ignored", nil); err == nil {
		t.Fatal("non-Windows helper unexpectedly succeeded")
	}
}

func TestCLIFormattingAndErrorBranches(t *testing.T) {
	if got := getActivationMode(false, true); got != "project-local" {
		t.Fatalf("local activation mode=%q", got)
	}
	if got := getActivationMode(true, false); got != "system-default" {
		t.Fatalf("default activation mode=%q", got)
	}
	if got := getActivationMode(false, false); got != "session-only" {
		t.Fatalf("session activation mode=%q", got)
	}
	if got := formatVersionTypeDesc(true, 3, 2); got != "versions (3 stable, 2 pre-release)" {
		t.Fatalf("unstable description=%q", got)
	}
	if got := formatVersionTypeDesc(false, 3, 0); got != "stable versions" {
		t.Fatalf("stable description=%q", got)
	}
	if resolveFullVersion("1.25.4") != "1.25.4" {
		t.Fatal("full version resolution floated an exact pin")
	}
	if !isDecimal("123") || isDecimal("") || isDecimal("12a") {
		t.Fatal("decimal backup suffix validation is incorrect")
	}
	if err := flagUsageError(&cobra.Command{}, fmt.Errorf("bad flag")); err == nil {
		t.Fatal("flagUsageError returned nil")
	}
	showBanner()
}
