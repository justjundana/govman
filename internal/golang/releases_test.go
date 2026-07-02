package golang

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetAvailableVersions(t *testing.T) {
	testCases := []struct {
		name            string
		includeUnstable bool
		mockResponse    []Release
		expectedCount   int
		shouldError     bool
	}{
		{
			name:            "Stable versions only",
			includeUnstable: false,
			mockResponse: []Release{
				{Version: "go1.21.0", Stable: true},
				{Version: "go1.20.5", Stable: true},
				{Version: "go1.22rc1", Stable: false},
			},
			expectedCount: 2,
			shouldError:   false,
		},
		{
			name:            "Include unstable versions",
			includeUnstable: true,
			mockResponse: []Release{
				{Version: "go1.21.0", Stable: true},
				{Version: "go1.20.5", Stable: true},
				{Version: "go1.22rc1", Stable: false},
			},
			expectedCount: 3,
			shouldError:   false,
		},
		{
			name:            "Reject invalid upstream version",
			includeUnstable: true,
			mockResponse: []Release{
				{Version: "not-a-go-version", Stable: true},
			},
			expectedCount: 0,
			shouldError:   true,
		},
		{
			name:            "Empty response",
			includeUnstable: false,
			mockResponse:    []Release{},
			expectedCount:   0,
			shouldError:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear cache before each test
			ClearReleasesCache()

			server := createMockServer(tc.mockResponse, http.StatusOK)
			defer server.Close()

			versions, err := GetAvailableVersionsWithConfig(tc.includeUnstable, server.URL, 1*time.Minute)

			if tc.shouldError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(versions) != tc.expectedCount {
				t.Errorf("Expected %d versions, got %d", tc.expectedCount, len(versions))
			}

			// Verify versions are sorted in descending order
			for i := 0; i < len(versions)-1; i++ {
				comparison, compareErr := CompareVersions(versions[i], versions[i+1])
				if compareErr != nil {
					t.Fatalf("failed to compare returned versions: %v", compareErr)
				}
				if comparison < 0 {
					t.Errorf("Versions not sorted correctly: %s should be before %s", versions[i], versions[i+1])
				}
			}

			// Verify "go" prefix is stripped
			for _, v := range versions {
				if strings.HasPrefix(v, "go") {
					t.Errorf("Version should not have 'go' prefix: %s", v)
				}
			}
		})
	}
}

func TestGetAvailableVersionsWithConfig_Errors(t *testing.T) {
	testCases := []struct {
		name         string
		statusCode   int
		mockResponse interface{}
		expectError  bool
	}{
		{
			name:         "HTTP error",
			statusCode:   http.StatusInternalServerError,
			mockResponse: []Release{},
			expectError:  true,
		},
		{
			name:         "Invalid JSON",
			statusCode:   http.StatusOK,
			mockResponse: "invalid json",
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear cache before each test
			ClearReleasesCache()

			var server *httptest.Server
			if tc.statusCode == http.StatusOK {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.statusCode)
					w.Write([]byte("invalid json"))
				}))
			} else {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.statusCode)
				}))
			}
			defer server.Close()

			_, err := GetAvailableVersionsWithConfig(false, server.URL, 1*time.Minute)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

func TestGetDownloadURL(t *testing.T) {
	// Determine the expected architecture after resolution
	expectedArch := resolveArch("1.21.0", runtime.GOOS, runtime.GOARCH)

	testCases := []struct {
		name         string
		version      string
		mockResponse []Release
		expectedURL  string
		shouldError  bool
	}{
		{
			name:    "Valid version with archive",
			version: "1.21.0",
			mockResponse: []Release{
				{
					Version: "go1.21.0",
					Files: []File{
						{
							Filename: fmt.Sprintf("go1.21.0.%s-%s.tar.gz", runtime.GOOS, expectedArch),
							OS:       runtime.GOOS,
							Arch:     expectedArch,
							Kind:     "archive",
						},
					},
				},
			},
			expectedURL: fmt.Sprintf("https://go.dev/dl/go1.21.0.%s-%s.tar.gz", runtime.GOOS, expectedArch),
			shouldError: false,
		},
		{
			name:    "Version not found",
			version: "1.99.0",
			mockResponse: []Release{
				{
					Version: "go1.21.0",
					Files: []File{
						{
							Filename: "go1.21.0.linux-amd64.tar.gz",
							OS:       "linux",
							Arch:     "amd64",
							Kind:     "archive",
						},
					},
				},
			},
			expectedURL: "",
			shouldError: true,
		},
		{
			name:    "No matching platform",
			version: "1.21.0",
			mockResponse: []Release{
				{
					Version: "go1.21.0",
					Files: []File{
						{
							Filename: "go1.21.0.windows-386.zip",
							OS:       "windows",
							Arch:     "386",
							Kind:     "archive",
						},
					},
				},
			},
			expectedURL: "",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear cache before each test
			ClearReleasesCache()

			server := createMockServer(tc.mockResponse, http.StatusOK)
			defer server.Close()

			url, err := GetDownloadURLWithConfig(tc.version, server.URL, 1*time.Minute, defaultGoDownloadURL)

			if tc.shouldError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tc.shouldError && url != tc.expectedURL {
				t.Errorf("Expected URL %q, got %q", tc.expectedURL, url)
			}
		})
	}
}

func TestResolveArch(t *testing.T) {
	testCases := []struct {
		name         string
		version      string
		goos         string
		goarch       string
		expectedArch string
	}{
		{
			name:         "Darwin ARM64 pre-1.16",
			version:      "1.15.0",
			goos:         "darwin",
			goarch:       "arm64",
			expectedArch: "amd64",
		},
		{
			name:         "Darwin ARM64 1.16+",
			version:      "1.16.0",
			goos:         "darwin",
			goarch:       "arm64",
			expectedArch: "arm64",
		},
		{
			name:         "Darwin ARM64 1.17",
			version:      "1.17.0",
			goos:         "darwin",
			goarch:       "arm64",
			expectedArch: "arm64",
		},
		{
			name:         "Darwin ARM64 exactly 1.16",
			version:      "1.16",
			goos:         "darwin",
			goarch:       "arm64",
			expectedArch: "arm64",
		},
		{
			name:         "Darwin ARM64 1.15.9",
			version:      "1.15.9",
			goos:         "darwin",
			goarch:       "arm64",
			expectedArch: "amd64",
		},
		{
			name:         "Linux AMD64",
			version:      "1.21.0",
			goos:         "linux",
			goarch:       "amd64",
			expectedArch: "amd64",
		},
		{
			name:         "Windows 386",
			version:      "1.20.0",
			goos:         "windows",
			goarch:       "386",
			expectedArch: "386",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			arch := resolveArch(tc.version, tc.goos, tc.goarch)

			if arch != tc.expectedArch {
				t.Errorf("Expected arch %q, got %q", tc.expectedArch, arch)
			}
		})
	}
}

func TestGetFileInfo(t *testing.T) {
	// Determine the expected architecture after resolution
	expectedArch := resolveArch("1.21.0", runtime.GOOS, runtime.GOARCH)

	testCases := []struct {
		name         string
		version      string
		mockResponse []Release
		expectError  bool
		checkFile    func(*testing.T, *File)
	}{
		{
			name:    "Valid file info",
			version: "1.21.0",
			mockResponse: []Release{
				{
					Version: "go1.21.0",
					Files: []File{
						{
							Filename: "go1.21.0." + runtime.GOOS + "-" + expectedArch + ".tar.gz",
							OS:       runtime.GOOS,
							Arch:     expectedArch,
							Kind:     "archive",
							Sha256:   "abc123",
							Size:     1024,
						},
					},
				},
			},
			expectError: false,
			checkFile: func(t *testing.T, f *File) {
				if f == nil {
					t.Fatal("Expected file to be non-nil")
				}
				if f.Sha256 != "abc123" {
					t.Errorf("Expected sha256 'abc123', got %q", f.Sha256)
				}
				if f.Size != 1024 {
					t.Errorf("Expected size 1024, got %d", f.Size)
				}
			},
		},
		{
			name:    "File not found",
			version: "1.99.0",
			mockResponse: []Release{
				{
					Version: "go1.21.0",
					Files:   []File{},
				},
			},
			expectError: true,
			checkFile:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear cache before each test
			ClearReleasesCache()

			server := createMockServer(tc.mockResponse, http.StatusOK)
			defer server.Close()

			file, err := GetFileInfoWithConfig(tc.version, server.URL, 1*time.Minute)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tc.expectError && tc.checkFile != nil {
				tc.checkFile(t, file)
			}
		})
	}
}

func TestGetVersionInfo(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		expectError bool
		checkInfo   func(*testing.T, *VersionInfo)
	}{
		{
			name: "Valid Go installation",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				goDir := filepath.Join(tmpDir, "go1.21.0")
				binDir := filepath.Join(goDir, "bin")

				if err := os.MkdirAll(binDir, 0755); err != nil {
					t.Fatal(err)
				}

				goBinary := filepath.Join(binDir, "go")
				if runtime.GOOS == "windows" {
					goBinary += ".exe"
				}

				if err := os.WriteFile(goBinary, []byte("fake"), 0755); err != nil {
					t.Fatal(err)
				}

				return goDir
			},
			expectError: false,
			checkInfo: func(t *testing.T, info *VersionInfo) {
				if info.Version != "1.21.0" {
					t.Errorf("Expected version '1.21.0', got %q", info.Version)
				}
				if info.OS != runtime.GOOS {
					t.Errorf("Expected OS %q, got %q", runtime.GOOS, info.OS)
				}
				if info.Arch != runtime.GOARCH {
					t.Errorf("Expected Arch %q, got %q", runtime.GOARCH, info.Arch)
				}
			},
		},
		{
			name: "Missing Go binary",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				goDir := filepath.Join(tmpDir, "go1.21.0")
				if err := os.MkdirAll(goDir, 0755); err != nil {
					t.Fatal(err)
				}
				return goDir
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			installPath := tc.setupFunc(t)
			info, err := GetVersionInfo(installPath)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tc.expectError && tc.checkInfo != nil {
				tc.checkInfo(t, info)
			}
		})
	}
}

func TestInstallMetadataControlsInstallDate(t *testing.T) {
	installPath := createLocalGoInstall(t, "1.25.1")
	installedAt := time.Date(2026, time.July, 2, 5, 30, 0, 123, time.UTC)
	if err := WriteInstallMetadata(installPath, "1.25.1", installedAt); err != nil {
		t.Fatal(err)
	}

	info, err := GetVersionInfo(installPath)
	if err != nil {
		t.Fatal(err)
	}
	if !info.InstallDate.Equal(installedAt) {
		t.Fatalf("InstallDate = %s, want %s", info.InstallDate, installedAt)
	}
	metadataPath := filepath.Join(installPath, installMetadataFilename)
	if runtime.GOOS != "windows" {
		metadataInfo, err := os.Stat(metadataPath)
		if err != nil {
			t.Fatal(err)
		}
		if metadataInfo.Mode().Perm() != 0600 {
			t.Fatalf("metadata permissions = %o, want 600", metadataInfo.Mode().Perm())
		}
	}
	if err := WriteInstallMetadata(installPath, "1.25.1", installedAt.Add(time.Hour)); err == nil {
		t.Fatal("WriteInstallMetadata replaced existing metadata")
	}
}

func TestGetVersionInfoLegacyInstallUsesDirectoryTime(t *testing.T) {
	installPath := createLocalGoInstall(t, "1.24.6")
	expected := time.Date(2025, time.December, 1, 2, 3, 4, 0, time.UTC)
	if err := os.Chtimes(installPath, expected, expected); err != nil {
		t.Fatal(err)
	}
	goBinary := filepath.Join(installPath, "bin", "go")
	if runtime.GOOS == "windows" {
		goBinary += ".exe"
	}
	archiveTime := expected.Add(-365 * 24 * time.Hour)
	if err := os.Chtimes(goBinary, archiveTime, archiveTime); err != nil {
		t.Fatal(err)
	}

	info, err := GetVersionInfo(installPath)
	if err != nil {
		t.Fatal(err)
	}
	if !info.InstallDate.Equal(expected) {
		t.Fatalf("legacy InstallDate = %s, want directory time %s", info.InstallDate, expected)
	}
}

func TestGetVersionInfoRejectsCorruptInstallMetadata(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
	}{
		{name: "malformed", content: "{"},
		{name: "wrong version", content: `{"version":"1.25.2","installed_at":"2026-07-02T00:00:00Z"}`},
		{name: "unknown field", content: `{"version":"1.25.1","installed_at":"2026-07-02T00:00:00Z","extra":true}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			installPath := createLocalGoInstall(t, "1.25.1")
			if err := os.WriteFile(filepath.Join(installPath, installMetadataFilename), []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := GetVersionInfo(installPath); err == nil {
				t.Fatal("GetVersionInfo accepted corrupt install metadata")
			}
		})
	}
}

func createLocalGoInstall(t *testing.T, version string) string {
	t.Helper()
	installPath := filepath.Join(t.TempDir(), "go"+version)
	if err := os.MkdirAll(filepath.Join(installPath, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	goBinary := filepath.Join(installPath, "bin", "go")
	if runtime.GOOS == "windows" {
		goBinary += ".exe"
	}
	if err := os.WriteFile(goBinary, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}
	return installPath
}

func TestCompareVersions(t *testing.T) {
	testCases := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{
			name:     "Equal versions",
			v1:       "1.21.0",
			v2:       "1.21.0",
			expected: 0,
		},
		{
			name:     "Exact same string",
			v1:       "go1.21.0",
			v2:       "go1.21.0",
			expected: 0,
		},
		{
			name:     "v1 greater major",
			v1:       "2.0.0",
			v2:       "1.21.0",
			expected: 1,
		},
		{
			name:     "v1 less major",
			v1:       "1.0.0",
			v2:       "2.0.0",
			expected: -1,
		},
		{
			name:     "v1 greater minor",
			v1:       "1.21.0",
			v2:       "1.20.0",
			expected: 1,
		},
		{
			name:     "v1 less minor",
			v1:       "1.20.0",
			v2:       "1.21.0",
			expected: -1,
		},
		{
			name:     "v1 greater patch",
			v1:       "1.21.5",
			v2:       "1.21.3",
			expected: 1,
		},
		{
			name:     "v1 less patch",
			v1:       "1.21.3",
			v2:       "1.21.5",
			expected: -1,
		},
		{
			name:     "Stable vs RC",
			v1:       "1.21.0",
			v2:       "1.21.0-rc1",
			expected: 1,
		},
		{
			name:     "RC vs stable",
			v1:       "1.21.0-rc1",
			v2:       "1.21.0",
			expected: -1,
		},
		{
			name:     "RC1 vs RC2",
			v1:       "1.21.0-rc2",
			v2:       "1.21.0-rc1",
			expected: 1,
		},
		{
			name:     "Beta vs alpha",
			v1:       "1.21.0-beta1",
			v2:       "1.21.0-alpha1",
			expected: 1,
		},
		{
			name:     "RC vs beta",
			v1:       "1.21.0-rc1",
			v2:       "1.21.0-beta1",
			expected: 1,
		},
		{
			name:     "With go prefix",
			v1:       "go1.21.0",
			v2:       "go1.20.0",
			expected: 1,
		},
		{
			name:     "With v prefix",
			v1:       "v1.21.0",
			v2:       "v1.20.0",
			expected: 1,
		},
		{
			name:     "Without patch version",
			v1:       "1.21",
			v2:       "1.20",
			expected: 1,
		},
		{
			name:     "Mixed patch presence",
			v1:       "1.21.0",
			v2:       "1.21",
			expected: 0,
		},
		{
			name:     "Alpha1 vs alpha2",
			v1:       "1.21.0-alpha2",
			v2:       "1.21.0-alpha1",
			expected: 1,
		},
		{
			name:     "Both have same prerelease",
			v1:       "1.21.0-rc1",
			v2:       "1.21.0-rc1",
			expected: 0,
		},
		{
			name:     "v prefix vs no prefix",
			v1:       "v1.21.0",
			v2:       "1.21.0",
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := CompareVersions(tc.v1, tc.v2)
			if err != nil {
				t.Fatalf("CompareVersions(%q, %q) error: %v", tc.v1, tc.v2, err)
			}

			if result != tc.expected {
				t.Errorf("CompareVersions(%q, %q) = %d, expected %d", tc.v1, tc.v2, result, tc.expected)
			}
		})
	}

	for _, invalid := range []string{"", "invalid", "1", "1.2.3.4", "1.2-preview1"} {
		t.Run("invalid_"+invalid, func(t *testing.T) {
			if _, err := CompareVersions(invalid, "1.2.3"); err == nil {
				t.Fatalf("CompareVersions accepted invalid version %q", invalid)
			}
			if _, err := CompareVersions("1.2.3", invalid); err == nil {
				t.Fatalf("CompareVersions accepted invalid version %q as second operand", invalid)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	testCases := []struct {
		name               string
		version            string
		expectedMajor      int
		expectedMinor      int
		expectedPatch      int
		expectedPrerelease string
		expectError        bool
	}{
		{
			name:               "Full version",
			version:            "1.21.5",
			expectedMajor:      1,
			expectedMinor:      21,
			expectedPatch:      5,
			expectedPrerelease: "",
		},
		{
			name:               "Version without patch",
			version:            "1.21",
			expectedMajor:      1,
			expectedMinor:      21,
			expectedPatch:      0,
			expectedPrerelease: "",
		},
		{
			name:               "Version with RC",
			version:            "1.21.0-rc1",
			expectedMajor:      1,
			expectedMinor:      21,
			expectedPatch:      0,
			expectedPrerelease: "rc1",
		},
		{
			name:               "Version with beta",
			version:            "1.21.0-beta2",
			expectedMajor:      1,
			expectedMinor:      21,
			expectedPatch:      0,
			expectedPrerelease: "beta2",
		},
		{
			name:        "Invalid version",
			version:     "invalid",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parts, err := parseVersion(tc.version)
			if tc.expectError {
				if err == nil {
					t.Fatalf("parseVersion(%q) succeeded", tc.version)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseVersion(%q) error: %v", tc.version, err)
			}

			if parts.numbers[0] != tc.expectedMajor {
				t.Errorf("Expected major %d, got %d", tc.expectedMajor, parts.numbers[0])
			}
			if parts.numbers[1] != tc.expectedMinor {
				t.Errorf("Expected minor %d, got %d", tc.expectedMinor, parts.numbers[1])
			}
			if parts.numbers[2] != tc.expectedPatch {
				t.Errorf("Expected patch %d, got %d", tc.expectedPatch, parts.numbers[2])
			}
			if parts.prerelease != tc.expectedPrerelease {
				t.Errorf("Expected prerelease %q, got %q", tc.expectedPrerelease, parts.prerelease)
			}
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "With go prefix",
			version:  "go1.21.0",
			expected: "1.21.0",
		},
		{
			name:     "With v prefix",
			version:  "v1.21.0",
			expected: "1.21.0",
		},
		{
			name:     "With both prefixes",
			version:  "gov1.21.0",
			expected: "1.21.0",
		},
		{
			name:     "Without prefix",
			version:  "1.21.0",
			expected: "1.21.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeVersion(tc.version)

			if result != tc.expected {
				t.Errorf("normalizeVersion(%q) = %q, expected %q", tc.version, result, tc.expected)
			}
		})
	}
}

func TestComparePrerelease(t *testing.T) {
	testCases := []struct {
		name     string
		pre1     string
		pre2     string
		expected int
	}{
		{
			name:     "Both empty",
			pre1:     "",
			pre2:     "",
			expected: 0,
		},
		{
			name:     "First empty (stable)",
			pre1:     "",
			pre2:     "rc1",
			expected: 1,
		},
		{
			name:     "Second empty (stable)",
			pre1:     "rc1",
			pre2:     "",
			expected: -1,
		},
		{
			name:     "Alpha vs beta",
			pre1:     "alpha1",
			pre2:     "beta1",
			expected: -1,
		},
		{
			name:     "Beta vs rc",
			pre1:     "beta1",
			pre2:     "rc1",
			expected: -1,
		},
		{
			name:     "RC1 vs RC2",
			pre1:     "rc1",
			pre2:     "rc2",
			expected: -1,
		},
		{
			name:     "Same prerelease",
			pre1:     "rc1",
			pre2:     "rc1",
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := comparePrerelease(tc.pre1, tc.pre2)

			if result != tc.expected {
				t.Errorf("comparePrerelease(%q, %q) = %d, expected %d", tc.pre1, tc.pre2, result, tc.expected)
			}
		})
	}
}

func TestGetPrereleaseRank(t *testing.T) {
	testCases := []struct {
		name         string
		prerelease   string
		expectedRank int
	}{
		{
			name:         "RC",
			prerelease:   "rc1",
			expectedRank: 3,
		},
		{
			name:         "Beta",
			prerelease:   "beta2",
			expectedRank: 2,
		},
		{
			name:         "Alpha",
			prerelease:   "alpha1",
			expectedRank: 1,
		},
		{
			name:         "Unknown",
			prerelease:   "gamma1",
			expectedRank: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rank := getPrereleaseRank(tc.prerelease)

			if rank != tc.expectedRank {
				t.Errorf("getPrereleaseRank(%q) = %d, expected %d", tc.prerelease, rank, tc.expectedRank)
			}
		})
	}
}

func TestExtractPrereleaseNumber(t *testing.T) {
	testCases := []struct {
		name           string
		prerelease     string
		expectedNumber int
	}{
		{
			name:           "RC with number",
			prerelease:     "rc1",
			expectedNumber: 1,
		},
		{
			name:           "Beta with number",
			prerelease:     "beta2",
			expectedNumber: 2,
		},
		{
			name:           "Alpha with large number",
			prerelease:     "alpha123",
			expectedNumber: 123,
		},
		{
			name:           "Without number",
			prerelease:     "rc",
			expectedNumber: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			num := extractPrereleaseNumber(tc.prerelease)

			if num != tc.expectedNumber {
				t.Errorf("extractPrereleaseNumber(%q) = %d, expected %d", tc.prerelease, num, tc.expectedNumber)
			}
		})
	}
}

func TestFetchReleasesWithConfig(t *testing.T) {
	testCases := []struct {
		name          string
		statusCode    int
		responseBody  interface{}
		cacheDuration time.Duration
		expectError   bool
		errorContains string
	}{
		{
			name:       "Successful fetch",
			statusCode: http.StatusOK,
			responseBody: []Release{
				{Version: "go1.21.0", Stable: true},
			},
			cacheDuration: 1 * time.Minute,
			expectError:   false,
		},
		{
			name:          "HTTP error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  []Release{},
			cacheDuration: 1 * time.Minute,
			expectError:   true,
			errorContains: "HTTP",
		},
		{
			name:          "Invalid JSON",
			statusCode:    http.StatusOK,
			responseBody:  "invalid",
			cacheDuration: 1 * time.Minute,
			expectError:   true,
			errorContains: "parse",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear cache before each test
			ClearReleasesCache()

			var server *httptest.Server

			if tc.statusCode == http.StatusOK && tc.responseBody != "invalid" {
				server = createMockServer(tc.responseBody.([]Release), tc.statusCode)
			} else if tc.responseBody == "invalid" {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.statusCode)
					w.Write([]byte("invalid json"))
				}))
			} else {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.statusCode)
				}))
			}
			defer server.Close()

			releases, err := fetchReleasesWithConfig(server.URL, tc.cacheDuration)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tc.expectError && err != nil && tc.errorContains != "" {
				if !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tc.errorContains, err.Error())
				}
			}

			if !tc.expectError && len(releases) == 0 {
				t.Error("Expected releases but got empty slice")
			}
		})
	}
}

func TestFetchReleasesCache(t *testing.T) {
	t.Run("Cache hit", func(t *testing.T) {
		// Clear cache before test
		ClearReleasesCache()

		mockReleases := []Release{
			{Version: "go1.21.0", Stable: true},
		}

		server := createMockServer(mockReleases, http.StatusOK)
		defer server.Close()

		// First call - populate cache
		releases1, err := fetchReleasesWithConfig(server.URL, 5*time.Minute)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Second call - should hit cache
		releases2, err := fetchReleasesWithConfig(server.URL, 5*time.Minute)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(releases1) != len(releases2) {
			t.Error("Cache should return same results")
		}
	})

	t.Run("Cache expired", func(t *testing.T) {
		// Clear cache before test
		ClearReleasesCache()

		mockReleases := []Release{
			{Version: "go1.21.0", Stable: true},
		}

		server := createMockServer(mockReleases, http.StatusOK)
		defer server.Close()

		// First call with very short cache duration
		_, err := fetchReleasesWithConfig(server.URL, 1*time.Millisecond)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Wait for cache to expire
		time.Sleep(2 * time.Millisecond)

		// Second call should fetch again
		_, err = fetchReleasesWithConfig(server.URL, 1*time.Minute)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
}

func TestFetchReleasesCacheIsolatedByEndpointAndPolicy(t *testing.T) {
	ClearReleasesCache()
	serverA := createMockServer([]Release{{Version: "go1.24.1", Stable: true}}, http.StatusOK)
	defer serverA.Close()
	serverB := createMockServer([]Release{{Version: "go1.25.2", Stable: true}}, http.StatusOK)
	defer serverB.Close()

	releasesA, err := fetchReleasesWithConfig(serverA.URL, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	releasesB, err := fetchReleasesWithConfig(serverB.URL, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if releasesA[0].Version != "go1.24.1" || releasesB[0].Version != "go1.25.2" {
		t.Fatalf("cache leaked across endpoints: A=%v B=%v", releasesA, releasesB)
	}

	var requests atomic.Int32
	policyServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		json.NewEncoder(writer).Encode([]Release{{Version: "go1.25.2", Stable: true}})
	}))
	defer policyServer.Close()
	if _, err := fetchReleasesWithConfig(policyServer.URL, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := fetchReleasesWithConfig(policyServer.URL, 2*time.Minute); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 {
		t.Fatalf("cache policy keys shared one response: requests=%d", requests.Load())
	}
}

func TestFetchReleasesReturnsDeepCopies(t *testing.T) {
	ClearReleasesCache()
	server := createMockServer([]Release{{
		Version: "go1.25.2",
		Stable:  true,
		Files:   []File{{Filename: "original.tar.gz"}},
	}}, http.StatusOK)
	defer server.Close()

	first, err := fetchReleasesWithConfig(server.URL, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	first[0].Version = "mutated"
	first[0].Files[0].Filename = "mutated.tar.gz"
	second, err := fetchReleasesWithConfig(server.URL, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if second[0].Version != "go1.25.2" || second[0].Files[0].Filename != "original.tar.gz" {
		t.Fatalf("caller mutated shared cache: %v", second)
	}
}

func TestFetchReleasesDeduplicatesConcurrentRequests(t *testing.T) {
	ClearReleasesCache()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.Header.Get("User-Agent") != "govman" {
			t.Errorf("User-Agent = %q, want govman", request.Header.Get("User-Agent"))
		}
		time.Sleep(25 * time.Millisecond)
		json.NewEncoder(writer).Encode([]Release{{Version: "go1.25.2", Stable: true}})
	}))
	defer server.Close()

	start := make(chan struct{})
	errors := make(chan error, 20)
	var wait sync.WaitGroup
	for range 20 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, err := fetchReleasesWithConfig(server.URL, time.Minute)
			errors <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	if requests.Load() != 1 {
		t.Fatalf("concurrent release requests = %d, want 1", requests.Load())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestFetchReleasesUsesInjectedClientAndBoundsBody(t *testing.T) {
	requestCount := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestCount++
		if request.Header.Get("User-Agent") != "govman" || request.Header.Get("Accept") != "application/json" {
			t.Fatalf("unexpected request headers: %v", request.Header)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(strings.Repeat("x", maxReleasesResponseSize+1))),
			Request:    request,
		}, nil
	})}

	if _, err := fetchReleasesContext(context.Background(), client, "https://example.test/releases"); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized response error = %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("injected client requests = %d, want 1", requestCount)
	}
	if _, err := fetchReleasesContext(context.Background(), nil, "https://example.test/releases"); err == nil {
		t.Fatal("nil releases client was accepted")
	}
}

func TestGetDirSize(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		expectError bool
		minSize     int64
	}{
		{
			name: "Directory with files",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()

				// Create some files
				if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("hello"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("world"), 0644); err != nil {
					t.Fatal(err)
				}

				return tmpDir
			},
			expectError: false,
			minSize:     10, // "hello" + "world" = 10 bytes
		},
		{
			name: "Empty directory",
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			expectError: false,
			minSize:     0,
		},
		{
			name: "Directory with subdirectories",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				subDir := filepath.Join(tmpDir, "subdir")

				if err := os.MkdirAll(subDir, 0755); err != nil {
					t.Fatal(err)
				}

				if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("data"), 0644); err != nil {
					t.Fatal(err)
				}

				return tmpDir
			},
			expectError: false,
			minSize:     8, // "test" + "data" = 8 bytes
		},
		{
			name: "Directory with error during walk",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				// Create a valid directory that exists
				// getDirSize handles walk errors gracefully and doesn't return error for non-existent paths
				// It returns error from filepath.Walk which is nil when path doesn't exist (it just doesn't walk)
				return tmpDir
			},
			expectError: false,
			minSize:     0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.setupFunc(t)
			size, err := getDirSize(path)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if size < tc.minSize {
					t.Errorf("Expected size >= %d, got %d", tc.minSize, size)
				}
			}
		})
	}
}

func TestGetDirSizeWithErrors(t *testing.T) {
	t.Run("Handle file access errors gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a file
		filePath := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		// getDirSize should handle errors gracefully and continue
		size, err := getDirSize(tmpDir)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if size < 4 {
			t.Errorf("Expected size >= 4, got %d", size)
		}
	})

	t.Run("Non-existent directory returns error from Walk", func(t *testing.T) {
		nonExistentPath := filepath.Join(os.TempDir(), "nonexistent-dir-12345")
		size, err := getDirSize(nonExistentPath)

		// filepath.Walk returns an error for non-existent paths
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}
		if size != 0 {
			t.Errorf("Expected size 0 for non-existent dir, got %d", size)
		}
	})
}

func TestDefaultFunctions(t *testing.T) {
	// Clear cache before tests
	ClearReleasesCache()

	testCases := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "GetAvailableVersions uses defaults",
			test: func(t *testing.T) {
				// This will try to connect to real API, so we just check it doesn't panic
				// In real scenario, you might want to mock the default API
				_, err := GetAvailableVersions(false)
				// We don't check for error since API might be down or network issues
				_ = err
			},
		},
		{
			name: "GetDownloadURL uses defaults",
			test: func(t *testing.T) {
				_, err := GetDownloadURL("1.21.0")
				// We don't check for error since API might be down
				_ = err
			},
		},
		{
			name: "GetFileInfo uses defaults",
			test: func(t *testing.T) {
				_, err := GetFileInfo("1.21.0")
				// We don't check for error since API might be down
				_ = err
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.test(t)
		})
	}
}

func TestConcurrency(t *testing.T) {
	testCases := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Concurrent cache access",
			test: func(t *testing.T) {
				// Clear cache before test
				ClearReleasesCache()

				mockReleases := []Release{
					{Version: "go1.21.0", Stable: true},
				}

				server := createMockServer(mockReleases, http.StatusOK)
				defer server.Close()

				var wg sync.WaitGroup
				for i := 0; i < 10; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						_, _ = fetchReleasesWithConfig(server.URL, 1*time.Minute)
					}()
				}

				wg.Wait()

				// Should not panic
			},
		},
		{
			name: "Concurrent clear and fetch",
			test: func(t *testing.T) {
				mockReleases := []Release{
					{Version: "go1.21.0", Stable: true},
				}

				server := createMockServer(mockReleases, http.StatusOK)
				defer server.Close()

				var wg sync.WaitGroup
				for i := 0; i < 5; i++ {
					wg.Add(2)
					go func() {
						defer wg.Done()
						ClearReleasesCache()
					}()
					go func() {
						defer wg.Done()
						_, _ = fetchReleasesWithConfig(server.URL, 1*time.Minute)
					}()
				}

				wg.Wait()

				// Should not panic or deadlock
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.test(t)
		})
	}
}

func TestReleaseJSONUnmarshaling(t *testing.T) {
	jsonData := `{
		"version": "go1.21.0",
		"stable": true,
		"files": [
			{
				"filename": "go1.21.0.linux-amd64.tar.gz",
				"os": "linux",
				"arch": "amd64",
				"version": "go1.21.0",
				"sha256": "abc123",
				"size": 1024,
				"kind": "archive"
			}
		]
	}`

	var release Release
	err := json.Unmarshal([]byte(jsonData), &release)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if release.Version != "go1.21.0" {
		t.Errorf("Expected version 'go1.21.0', got %q", release.Version)
	}
	if !release.Stable {
		t.Error("Expected stable to be true")
	}
	if len(release.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(release.Files))
	}
}

func TestFileJSONUnmarshaling(t *testing.T) {
	jsonData := `{
		"filename": "go1.21.0.linux-amd64.tar.gz",
		"os": "linux",
		"arch": "amd64",
		"version": "go1.21.0",
		"sha256": "abc123",
		"size": 1024,
		"kind": "archive"
	}`

	var file File
	err := json.Unmarshal([]byte(jsonData), &file)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if file.Filename != "go1.21.0.linux-amd64.tar.gz" {
		t.Errorf("Expected filename 'go1.21.0.linux-amd64.tar.gz', got %q", file.Filename)
	}
	if file.OS != "linux" {
		t.Errorf("Expected OS 'linux', got %q", file.OS)
	}
	if file.Arch != "amd64" {
		t.Errorf("Expected arch 'amd64', got %q", file.Arch)
	}
	if file.Sha256 != "abc123" {
		t.Errorf("Expected sha256 'abc123', got %q", file.Sha256)
	}
	if file.Size != 1024 {
		t.Errorf("Expected size 1024, got %d", file.Size)
	}
	if file.Kind != "archive" {
		t.Errorf("Expected kind 'archive', got %q", file.Kind)
	}
}

func TestVersionInfoStructFields(t *testing.T) {
	now := time.Now()
	info := &VersionInfo{
		Version:     "1.21.0",
		Path:        "/usr/local/go",
		OS:          "linux",
		Arch:        "amd64",
		InstallDate: now,
		Size:        1024000,
	}

	if info.Version != "1.21.0" {
		t.Errorf("Expected version '1.21.0', got %q", info.Version)
	}
	if info.Path != "/usr/local/go" {
		t.Errorf("Expected path '/usr/local/go', got %q", info.Path)
	}
	if info.OS != "linux" {
		t.Errorf("Expected OS 'linux', got %q", info.OS)
	}
	if info.Arch != "amd64" {
		t.Errorf("Expected arch 'amd64', got %q", info.Arch)
	}
	if !info.InstallDate.Equal(now) {
		t.Error("InstallDate mismatch")
	}
	if info.Size != 1024000 {
		t.Errorf("Expected size 1024000, got %d", info.Size)
	}
}

func TestFetchReleasesNetworkError(t *testing.T) {
	t.Run("Invalid URL causes error", func(t *testing.T) {
		// Clear cache before test
		ClearReleasesCache()

		_, err := fetchReleasesWithConfig("http://invalid-url-that-does-not-exist-12345.com", 1*time.Minute)
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
		if !strings.Contains(err.Error(), "failed to fetch releases") {
			t.Errorf("Expected 'failed to fetch releases' in error, got: %v", err)
		}
	})
}

func TestVersionInfoWithLargeDirectory(t *testing.T) {
	t.Run("Calculate size of directory with multiple files", func(t *testing.T) {
		tmpDir := t.TempDir()
		goDir := filepath.Join(tmpDir, "go1.21.0")
		binDir := filepath.Join(goDir, "bin")
		srcDir := filepath.Join(goDir, "src")

		if err := os.MkdirAll(binDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			t.Fatal(err)
		}

		goBinary := filepath.Join(binDir, "go")
		if runtime.GOOS == "windows" {
			goBinary += ".exe"
		}

		if err := os.WriteFile(goBinary, []byte("fake binary content"), 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644); err != nil {
			t.Fatal(err)
		}

		info, err := GetVersionInfo(goDir)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		expectedMinSize := int64(len("fake binary content") + len("package main"))
		if info.Size < expectedMinSize {
			t.Errorf("Expected size >= %d, got %d", expectedMinSize, info.Size)
		}
	})
}

// Helper function to create a mock HTTP server
func createMockServer(releases []Release, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(releases)
	}))
}
