package golang

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	releasesCache []Release
	cacheMutex    sync.RWMutex
	cacheExpiry   time.Time
)

const (
	GoDownloadURLTemplate = "%s"
)

var (
	defaultGoReleasesAPI = "https://go.dev/dl/?mode=json&include=all"
	defaultCacheDuration = 10 * time.Minute
	defaultGoDownloadURL = "https://go.dev/dl/%s"
)

type Release struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
	Files   []File `json:"files"`
}

type File struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	Sha256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Kind     string `json:"kind"`
}

type VersionInfo struct {
	Version     string
	Path        string
	OS          string
	Arch        string
	InstallDate time.Time
	Size        int64
}

// GetAvailableVersions returns all available Go versions, optionally including unstable ones.
// Parameter includeUnstable controls inclusion. Returns a sorted slice of version strings or an error.
func GetAvailableVersions(includeUnstable bool) ([]string, error) {
	return GetAvailableVersionsWithConfig(includeUnstable, defaultGoReleasesAPI, defaultCacheDuration)
}

// GetAvailableVersionsWithConfig fetches available versions using a specific API URL and cache duration.
// Parameters: includeUnstable, apiURL, cacheDuration. Returns a sorted slice of version strings or an error.
func GetAvailableVersionsWithConfig(includeUnstable bool, apiURL string, cacheDuration time.Duration) ([]string, error) {
	releases, err := fetchReleasesWithConfig(apiURL, cacheDuration)
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, release := range releases {
		if !includeUnstable && !release.Stable {
			continue
		}

		version := strings.TrimPrefix(release.Version, "go")
		versions = append(versions, version)
	}

	sort.Slice(versions, func(i, j int) bool {
		return CompareVersions(versions[i], versions[j]) > 0
	})

	return versions, nil
}

// GetDownloadURL returns the archive download URL for a given version using default endpoints.
// Parameter version is the version string. Returns the URL or an error if unavailable for the platform.
func GetDownloadURL(version string) (string, error) {
	return GetDownloadURLWithConfig(version, defaultGoReleasesAPI, defaultCacheDuration, defaultGoDownloadURL)
}

// GetDownloadURLWithConfig computes the archive download URL using custom API and URL template.
// Parameters: version, apiURL, cacheDuration, downloadURL (format string). Returns URL or error.
func GetDownloadURLWithConfig(version string, apiURL string, cacheDuration time.Duration, downloadURL string) (string, error) {
	releases, err := fetchReleasesWithConfig(apiURL, cacheDuration)
	if err != nil {
		return "", err
	}

	targetVersion := "go" + version
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	resolvedArch := resolveArch(version, goos, goarch)

	for _, release := range releases {
		if release.Version != targetVersion {
			continue
		}

		for _, file := range release.Files {
			if file.OS == goos && file.Arch == resolvedArch && file.Kind == "archive" {
				return fmt.Sprintf(downloadURL, file.Filename), nil
			}
		}
	}

	return "", fmt.Errorf("no download available for Go %s on %s/%s", version, goos, goarch)
}

// resolveArch determines the appropriate architecture for downloads (e.g., maps darwin/arm64 to amd64 pre-1.16).
// Parameters: version, goos, goarch. Returns the resolved architecture string.
func resolveArch(version, goos, goarch string) string {
	if goos == "darwin" && goarch == "arm64" {
		if CompareVersions(version, "1.16") < 0 {
			return "amd64"
		}
	}

	return goarch
}

// GetFileInfo returns metadata for the current platform's archive for a version using defaults.
// Parameter version is the version string. Returns *File or an error if not found.
func GetFileInfo(version string) (*File, error) {
	return GetFileInfoWithConfig(version, defaultGoReleasesAPI, defaultCacheDuration)
}

// GetFileInfoWithConfig returns archive metadata using a specific API URL and cache duration.
// Parameters: version, apiURL, cacheDuration. Returns *File or an error.
func GetFileInfoWithConfig(version string, apiURL string, cacheDuration time.Duration) (*File, error) {
	releases, err := fetchReleasesWithConfig(apiURL, cacheDuration)
	if err != nil {
		return nil, err
	}

	targetVersion := "go" + version
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	resolvedArch := resolveArch(version, goos, goarch)

	for _, release := range releases {
		if release.Version != targetVersion {
			continue
		}

		for _, file := range release.Files {
			if file.OS == goos && file.Arch == resolvedArch && file.Kind == "archive" {
				return &file, nil
			}
		}
	}

	return nil, fmt.Errorf("no file info available for Go %s on %s/%s", version, goos, goarch)
}

// GetVersionInfo collects local installation details (version, path, OS/arch, install date, size).
// Parameter installPath is the Go installation root. Returns *VersionInfo or an error if missing binary.
func GetVersionInfo(installPath string) (*VersionInfo, error) {
	goBinary := filepath.Join(installPath, "bin", "go")
	if runtime.GOOS == "windows" {
		goBinary += ".exe"
	}

	stat, err := os.Stat(goBinary)
	if err != nil {
		return nil, fmt.Errorf("go binary not found in %s", installPath)
	}

	version := filepath.Base(installPath)
	version = strings.TrimPrefix(version, "go")

	size, err := getDirSize(installPath)
	if err != nil {
		size = 0
	}

	return &VersionInfo{
		Version:     version,
		Path:        installPath,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		InstallDate: stat.ModTime(),
		Size:        size,
	}, nil
}

// CompareVersions compares two semantic version strings with prerelease awareness.
// Returns 1 if v1 > v2, -1 if v1 < v2, and 0 if equal.
func CompareVersions(v1, v2 string) int {
	// Early return for identical strings
	if v1 == v2 {
		return 0
	}

	// Normalize once and check again
	v1Norm := normalizeVersion(v1)
	v2Norm := normalizeVersion(v2)

	if v1Norm == v2Norm {
		return 0
	}

	// Parse both versions
	parts1 := parseVersion(v1Norm)
	parts2 := parseVersion(v2Norm)

	// Compare version numbers
	for i := 0; i < 3; i++ {
		if parts1.numbers[i] > parts2.numbers[i] {
			return 1
		} else if parts1.numbers[i] < parts2.numbers[i] {
			return -1
		}
	}

	// Compare prerelease tags
	return comparePrerelease(parts1.prerelease, parts2.prerelease)
}

// IsValidVersion validates a version string (optional patch and prerelease tags supported).
// Parameter version. Returns true if valid, false otherwise.
func IsValidVersion(version string) bool {
	pattern := `^\d+\.\d+(?:\.\d+)?(?:-?(?:rc|beta|alpha)\d*)?$`
	matched, _ := regexp.MatchString(pattern, version)
	return matched
}

type versionParts struct {
	numbers    [3]int
	prerelease string
}

// parseVersion parses a normalized version into numeric components and a prerelease tag.
// Parameter version. Returns a versionParts struct.
func parseVersion(version string) versionParts {
	var parts versionParts

	re := regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?(?:-?(rc\d+|beta\d+|alpha\d+))?$`)
	matches := re.FindStringSubmatch(version)

	if len(matches) == 0 {
		return parts
	}

	if len(matches) > 1 {
		if num, err := strconv.Atoi(matches[1]); err == nil {
			parts.numbers[0] = num
		}
	}

	if len(matches) > 2 {
		if num, err := strconv.Atoi(matches[2]); err == nil {
			parts.numbers[1] = num
		}
	}

	if len(matches) > 3 && matches[3] != "" {
		if num, err := strconv.Atoi(matches[3]); err == nil {
			parts.numbers[2] = num
		}
	}

	if len(matches) > 4 && matches[4] != "" {
		parts.prerelease = matches[4]
	}

	return parts
}

// normalizeVersion strips leading "go" or "v" prefixes from version strings.
// Parameter version. Returns the normalized string.
func normalizeVersion(version string) string {
	version = strings.TrimPrefix(version, "go")
	version = strings.TrimPrefix(version, "v")
	return version
}

// comparePrerelease compares prerelease identifiers by type (alpha < beta < rc) and numeric suffix.
// Returns 1, -1, or 0 depending on ordering.
func comparePrerelease(pre1, pre2 string) int {
	if pre1 == "" && pre2 == "" {
		return 0
	}
	if pre1 == "" {
		return 1
	}
	if pre2 == "" {
		return -1
	}

	rank1 := getPrereleaseRank(pre1)
	rank2 := getPrereleaseRank(pre2)

	if rank1 != rank2 {
		return rank1 - rank2
	}

	num1 := extractPrereleaseNumber(pre1)
	num2 := extractPrereleaseNumber(pre2)

	if num1 > num2 {
		return 1
	} else if num1 < num2 {
		return -1
	}

	return 0
}

// getPrereleaseRank assigns an ordering to prerelease types: alpha < beta < rc.
// Parameter prerelease. Returns an integer rank.
func getPrereleaseRank(prerelease string) int {
	if strings.HasPrefix(prerelease, "rc") {
		return 3
	} else if strings.HasPrefix(prerelease, "beta") {
		return 2
	} else if strings.HasPrefix(prerelease, "alpha") {
		return 1
	}
	return 0
}

// extractPrereleaseNumber extracts trailing digits from a prerelease tag.
// Parameter prerelease. Returns the numeric suffix, or 0 if absent.
func extractPrereleaseNumber(prerelease string) int {
	re := regexp.MustCompile(`\d+$`)
	match := re.FindString(prerelease)
	if num, err := strconv.Atoi(match); err == nil {
		return num
	}
	return 0
}

// fetchReleasesWithConfig fetches releases JSON, caches results with expiry, and returns parsed data.
// Parameters: apiURL, cacheDuration. Returns []Release or an error.
func fetchReleasesWithConfig(apiURL string, cacheDuration time.Duration) ([]Release, error) {
	cacheMutex.RLock()
	if time.Now().Before(cacheExpiry) && releasesCache != nil {
		defer cacheMutex.RUnlock()
		return releasesCache, nil
	}
	cacheMutex.RUnlock()

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch releases: HTTP %d (%s)", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var releases []Release
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases: %w", err)
	}

	cacheMutex.Lock()
	releasesCache = releases
	cacheExpiry = time.Now().Add(cacheDuration)
	cacheMutex.Unlock()

	return releases, nil
}

// getDirSize walks a directory and sums file sizes.
// Parameter path. Returns total size in bytes or an error (errors during walk are ignored).
func getDirSize(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// ClearReleasesCache clears the in-memory releases cache and resets its expiry time.
func ClearReleasesCache() {
	cacheMutex.Lock()
	releasesCache = nil
	cacheExpiry = time.Time{}
	cacheMutex.Unlock()
}
