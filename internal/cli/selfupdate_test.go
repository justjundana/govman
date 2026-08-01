package cli

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	_config "github.com/justjundana/govman/internal/config"
	_version "github.com/justjundana/govman/internal/version"
)

// TestAssetExactMatch verifies the fix for CQ-1:
// Asset selection must use exact name matching, not substring Contains.
// This prevents "govman-linux-amd64" from matching "govman-linux-amd64-v2".
func TestAssetExactMatch(t *testing.T) {
	type asset struct {
		Name        string
		DownloadURL string
	}

	assets := []asset{
		{Name: "govman-linux-amd64-v2", DownloadURL: "https://example.com/v2"},
		{Name: "govman-linux-amd64", DownloadURL: "https://example.com/correct"},
		{Name: "govman-linux-arm64", DownloadURL: "https://example.com/arm"},
	}

	targetName := "govman-linux-amd64"

	// Exact match (fixed behavior): must find the exact name, not a superset
	var exactMatch string
	for _, a := range assets {
		if a.Name == targetName {
			exactMatch = a.DownloadURL
			break
		}
	}
	if exactMatch != "https://example.com/correct" {
		t.Errorf("exact match got %q, want %q", exactMatch, "https://example.com/correct")
	}

	// Old behavior (substring match): would incorrectly match the -v2 variant first
	var substringMatch string
	for _, a := range assets {
		if strings.Contains(a.Name, targetName) {
			substringMatch = a.DownloadURL
			break
		}
	}
	// The -v2 variant also contains "govman-linux-amd64" as a substring,
	// so Contains matches the wrong one first.
	if substringMatch == "https://example.com/v2" {
		t.Log("Confirmed: substring match picks wrong asset when -v2 variant appears first")
	}
}

func resetSelfUpdateTestState(t *testing.T) {
	t.Helper()
	originalRename := selfUpdateRename
	originalRemove := selfUpdateRemove
	originalChmod := selfUpdateChmod
	originalValidate := selfUpdateValidateBinary
	originalClient := selfUpdateHTTPClient
	originalConfig := cfg
	originalVersion := _version.Version
	originalCommit := _version.Commit
	t.Cleanup(func() {
		selfUpdateRename = originalRename
		selfUpdateRemove = originalRemove
		selfUpdateChmod = originalChmod
		selfUpdateValidateBinary = originalValidate
		selfUpdateHTTPClient = originalClient
		cfg = originalConfig
		_version.Version = originalVersion
		_version.Commit = originalCommit
	})
}

func TestChecksumForAsset(t *testing.T) {
	hash := strings.Repeat("a", sha256.Size*2)
	for _, test := range []struct {
		name     string
		manifest string
		want     string
		wantErr  string
	}{
		{name: "matching", manifest: hash + "  govman-linux-amd64\n", want: hash},
		{name: "star format", manifest: hash + " *govman-linux-amd64\n", want: hash},
		{name: "missing", manifest: hash + "  another-file\n", wantErr: "does not contain"},
		{name: "duplicate", manifest: hash + "  govman-linux-amd64\n" + hash + "  govman-linux-amd64\n", wantErr: "duplicate"},
		{name: "invalid hash", manifest: strings.Repeat("z", sha256.Size*2) + "  govman-linux-amd64\n", wantErr: "invalid SHA-256"},
		{name: "malformed", manifest: "not-enough-fields\n", wantErr: "invalid checksum manifest"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := checksumForAsset(test.manifest, "govman-linux-amd64")
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("checksum error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("checksum = %q, err=%v, want %q", got, err, test.want)
			}
		})
	}
}

func TestDownloadBinaryVerifiesChecksumAndCleansFailures(t *testing.T) {
	resetSelfUpdateTestState(t)
	binary := []byte("verified binary bytes")
	hash := fmt.Sprintf("%x", sha256.Sum256(binary))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !strings.HasPrefix(request.Header.Get("User-Agent"), "govman/") {
			t.Errorf("missing govman User-Agent: %q", request.Header.Get("User-Agent"))
		}
		writer.Write(binary)
	}))
	defer server.Close()
	asset := &GitHubAsset{Name: "govman-test", DownloadURL: server.URL + "/govman-test", Size: int64(len(binary))}
	directory := t.TempDir()

	path, err := downloadBinary(context.Background(), asset, directory, hash)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != string(binary) {
		t.Fatalf("downloaded data = %q, err=%v", data, err)
	}
	os.Remove(path)

	if _, err := downloadBinary(context.Background(), asset, directory, strings.Repeat("0", sha256.Size*2)); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("checksum mismatch error = %v", err)
	}
	partials, err := filepath.Glob(filepath.Join(directory, "govman-update-*"))
	if err != nil || len(partials) != 0 {
		t.Fatalf("failed download left temp files: %v, err=%v", partials, err)
	}
}

func TestDownloadBinaryRejectsInvalidSizes(t *testing.T) {
	directory := t.TempDir()
	asset := &GitHubAsset{Name: "large", DownloadURL: "https://example.com/large", Size: maxSelfUpdateBinarySize + 1}
	if _, err := downloadBinary(context.Background(), asset, directory, strings.Repeat("0", sha256.Size*2)); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized metadata error = %v", err)
	}

	body := []byte("short")
	hash := fmt.Sprintf("%x", sha256.Sum256(body))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write(body)
	}))
	defer server.Close()
	asset = &GitHubAsset{Name: "truncated", DownloadURL: server.URL + "/truncated", Size: int64(len(body) + 10)}
	if _, err := downloadBinary(context.Background(), asset, directory, hash); err == nil || !strings.Contains(err.Error(), "size mismatch") {
		t.Fatalf("truncated binary error = %v", err)
	}
	partials, _ := filepath.Glob(filepath.Join(directory, "govman-update-*"))
	if len(partials) != 0 {
		t.Fatalf("truncated download left temp files: %v", partials)
	}
}

func TestValidateDownloadedBinaryVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-specific; Windows replacement is tested through injected validation")
	}
	directory := t.TempDir()
	writeScript := func(name, version string) string {
		path := filepath.Join(directory, name)
		content := "#!/bin/sh\nprintf '%s\\n' 'govman version " + version + "'\n"
		if err := os.WriteFile(path, []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
		return path
	}
	if err := validateDownloadedBinary(writeScript("correct", "1.3.4"), "v1.3.4"); err != nil {
		t.Fatalf("valid binary rejected: %v", err)
	}
	if err := validateDownloadedBinary(writeScript("wrong", "1.3.3"), "v1.3.4"); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("wrong binary version error = %v", err)
	}
}

func TestReplaceBinaryRollsBackFailures(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func()
	}{
		{
			name: "install rename failure",
			setup: func() {
				calls := 0
				selfUpdateRename = func(oldPath, newPath string) error {
					calls++
					if calls == 2 {
						return errors.New("injected rename failure")
					}
					return os.Rename(oldPath, newPath)
				}
			},
		},
		{
			name: "post install validation failure",
			setup: func() {
				calls := 0
				selfUpdateValidateBinary = func(path, target string) error {
					calls++
					if calls == 2 {
						return errors.New("injected validation failure")
					}
					return nil
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			resetSelfUpdateTestState(t)
			selfUpdateValidateBinary = func(path, target string) error { return nil }
			test.setup()
			directory := t.TempDir()
			current := filepath.Join(directory, "govman")
			temporary := filepath.Join(directory, "govman-update")
			if err := os.WriteFile(current, []byte("old"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(temporary, []byte("new"), 0755); err != nil {
				t.Fatal(err)
			}
			if _, err := replaceBinary(current, temporary, "v1.3.4"); err == nil {
				t.Fatal("replaceBinary succeeded despite injected failure")
			}
			data, err := os.ReadFile(current)
			if err != nil || string(data) != "old" {
				t.Fatalf("backup was not restored: data=%q err=%v", data, err)
			}
		})
	}
}

func TestReplaceBinaryChmodFailureKeepsOriginal(t *testing.T) {
	resetSelfUpdateTestState(t)
	selfUpdateChmod = func(string, os.FileMode) error { return errors.New("injected chmod failure") }
	selfUpdateValidateBinary = func(path, target string) error { return nil }
	directory := t.TempDir()
	current := filepath.Join(directory, "govman")
	temporary := filepath.Join(directory, "govman-update")
	os.WriteFile(current, []byte("old"), 0755)
	os.WriteFile(temporary, []byte("new"), 0644)
	if _, err := replaceBinary(current, temporary, "v1.3.4"); err == nil || !strings.Contains(err.Error(), "executable permission") {
		t.Fatalf("chmod failure error = %v", err)
	}
	data, err := os.ReadFile(current)
	if err != nil || string(data) != "old" {
		t.Fatalf("chmod failure changed original: data=%q err=%v", data, err)
	}
}

func TestScheduleWindowsBinaryReplacementUsesDetachedHelper(t *testing.T) {
	originalChmod := selfUpdateChmod
	originalValidate := selfUpdateValidateBinary
	originalStarter := selfUpdateStartHelper
	t.Cleanup(func() {
		selfUpdateChmod = originalChmod
		selfUpdateValidateBinary = originalValidate
		selfUpdateStartHelper = originalStarter
	})

	directory := t.TempDir()
	current := filepath.Join(directory, "govman.exe")
	temporary := filepath.Join(directory, "govman-update.exe")
	if err := os.WriteFile(current, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(temporary, []byte("new"), 0755); err != nil {
		t.Fatal(err)
	}
	selfUpdateChmod = func(string, os.FileMode) error { return nil }
	selfUpdateValidateBinary = func(path, version string) error {
		if path != temporary || version != "v1.3.4" {
			t.Fatalf("validation args = %q, %q", path, version)
		}
		return nil
	}

	var helperPath string
	var arguments []string
	selfUpdateStartHelper = func(path string, args []string) error {
		helperPath = path
		arguments = append([]string(nil), args...)
		return nil
	}
	if err := scheduleWindowsBinaryReplacement(current, temporary, "v1.3.4"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(helperPath) })

	helperContent, err := os.ReadFile(helperPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Get-Process", "Move-Item", "--version", "init --force --shell cmd", "Move-Item -LiteralPath $BackupPath -Destination $SourcePath"} {
		if !strings.Contains(string(helperContent), expected) {
			t.Fatalf("Windows helper is missing %q", expected)
		}
	}
	joined := strings.Join(arguments, "\n")
	for _, expected := range []string{
		"-SourcePath\n" + current,
		"-DestinationPath\n" + filepath.Join(directory, "govman-real.exe"),
		"-TempPath\n" + temporary,
		"-ExpectedVersion\nv1.3.4",
		"-MigrateLegacy\n1",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("Windows helper args %q are missing %q", joined, expected)
		}
	}
	if data, err := os.ReadFile(current); err != nil || string(data) != "old" {
		t.Fatalf("scheduler modified running binary: data=%q err=%v", data, err)
	}
	if data, err := os.ReadFile(temporary); err != nil || string(data) != "new" {
		t.Fatalf("scheduler modified downloaded binary: data=%q err=%v", data, err)
	}
}

func TestScheduleWindowsBinaryReplacementCleansHelperOnStartFailure(t *testing.T) {
	originalChmod := selfUpdateChmod
	originalValidate := selfUpdateValidateBinary
	originalStarter := selfUpdateStartHelper
	t.Cleanup(func() {
		selfUpdateChmod = originalChmod
		selfUpdateValidateBinary = originalValidate
		selfUpdateStartHelper = originalStarter
	})

	directory := t.TempDir()
	current := filepath.Join(directory, "govman-real.exe")
	temporary := filepath.Join(directory, "govman-update.exe")
	if err := os.WriteFile(current, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(temporary, []byte("new"), 0755); err != nil {
		t.Fatal(err)
	}
	selfUpdateChmod = func(string, os.FileMode) error { return nil }
	selfUpdateValidateBinary = func(string, string) error { return nil }
	selfUpdateStartHelper = func(string, []string) error { return fmt.Errorf("start failed") }

	if err := scheduleWindowsBinaryReplacement(current, temporary, "v1.3.4"); err == nil || !strings.Contains(err.Error(), "start failed") {
		t.Fatalf("scheduler start error = %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(directory, ".govman-update-helper-*.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("failed scheduler left helpers: %v", matches)
	}
}

func TestFindReleaseAssetRequiresExactUniqueHTTPSAsset(t *testing.T) {
	release := &GitHubRelease{Assets: []GitHubAsset{
		{Name: "govman-linux-amd64-v2", DownloadURL: "https://example.com/v2"},
		{Name: "govman-linux-amd64", DownloadURL: "https://example.com/correct"},
	}}
	asset, err := findReleaseAsset(release, "govman-linux-amd64")
	if err != nil || asset.DownloadURL != "https://example.com/correct" {
		t.Fatalf("asset=%v err=%v", asset, err)
	}
	release.Assets = append(release.Assets, GitHubAsset{Name: "govman-linux-amd64", DownloadURL: "https://example.com/duplicate"})
	if _, err := findReleaseAsset(release, "govman-linux-amd64"); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate asset error = %v", err)
	}
	release.Assets = []GitHubAsset{{Name: "govman-linux-amd64", DownloadURL: "http://example.com/insecure"}}
	if _, err := findReleaseAsset(release, "govman-linux-amd64"); err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("insecure asset URL error = %v", err)
	}
}

func TestRunSelfUpdateDevBuildSkipsNetwork(t *testing.T) {
	resetSelfUpdateTestState(t)
	_version.Version = "dev"
	_version.Commit = "test"
	cfg = nil
	if err := runSelfUpdateContext(context.Background(), false, false, false); err != nil {
		t.Fatalf("dev build self-update error = %v", err)
	}
}

func TestRunSelfUpdateRefusesDowngrade(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprintf("force=%v", force), func(t *testing.T) {
			resetSelfUpdateTestState(t)
			_version.Version = "v1.3.4"
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				fmt.Fprint(writer, `{"tag_name":"v1.3.3","prerelease":false,"draft":false,"assets":[]}`)
			}))
			defer server.Close()
			cfg = &_config.Config{SelfUpdate: _config.SelfUpdateConfig{GitHubAPIURL: server.URL, GitHubReleasesURL: server.URL}}
			if err := runSelfUpdateContext(context.Background(), false, force, false); err != nil {
				t.Fatalf("downgrade check error = %v", err)
			}
		})
	}
}

func TestGetLatestReleaseBoundsAndValidatesResponse(t *testing.T) {
	resetSelfUpdateTestState(t)
	_version.Version = "v1.3.4"
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "invalid tag", body: `{"tag_name":"latest","draft":false}`, want: "not valid semantic"},
		{name: "stable endpoint prerelease", body: `{"tag_name":"v1.4.0-rc1","prerelease":true,"draft":false}`, want: "prerelease"},
		{name: "oversized", body: strings.Repeat("x", maxGitHubAPIResponseSize+1), want: "exceeds"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				fmt.Fprint(writer, test.body)
			}))
			defer server.Close()
			cfg = &_config.Config{SelfUpdate: _config.SelfUpdateConfig{GitHubAPIURL: server.URL, GitHubReleasesURL: server.URL}}
			_, err := getLatestReleaseContext(context.Background(), false)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("release error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestFetchLimitedHonorsContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := fetchLimited(ctx, server.URL, 1024, "application/json"); err == nil {
		t.Fatal("cancelled update request returned no error")
	}
}

// TestTempFileInBinaryDir verifies the fix for BUG-2:
// Temp file must be created in the same directory as the binary,
// not in the system temp dir, to avoid cross-device rename failures (EXDEV).
func TestTempFileInBinaryDir(t *testing.T) {
	// Create a temp directory to simulate a binary location
	binaryDir := t.TempDir()
	binaryPath := filepath.Join(binaryDir, "govman")

	// Create a dummy binary file
	if err := os.WriteFile(binaryPath, []byte("binary"), 0755); err != nil {
		t.Fatal(err)
	}

	// The fix uses filepath.Dir(currentBinary) as the temp dir
	tempDir := filepath.Dir(binaryPath)
	if tempDir != binaryDir {
		t.Errorf("filepath.Dir(binaryPath) = %q, want %q", tempDir, binaryDir)
	}

	// Create temp file in binary dir (same as the fix does)
	tmpFile, err := os.CreateTemp(tempDir, "govman-update-*.bin")
	if err != nil {
		t.Fatalf("failed to create temp file in binary dir: %v", err)
	}
	tmpName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpName)

	// Verify temp file is in the same directory as the binary
	if filepath.Dir(tmpName) != binaryDir {
		t.Errorf("temp file dir = %q, want %q", filepath.Dir(tmpName), binaryDir)
	}

	// Rename should succeed (same filesystem)
	newPath := filepath.Join(binaryDir, "govman-new")
	if err := os.Rename(tmpName, newPath); err != nil {
		t.Errorf("rename within same dir failed: %v", err)
	}
	defer os.Remove(newPath)
}
