package golang

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
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
	releasesCache      = make(map[releasesCacheKey]*releasesCacheEntry)
	cacheMutex         sync.Mutex
	releasesHTTPClient = &http.Client{Timeout: 30 * time.Second}

	// Pre-compiled regex patterns to avoid repeated compilation
	versionParseRegex     = regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?(?:-?(rc\d+|beta\d+|alpha\d+))?$`)
	prereleaseNumberRegex = regexp.MustCompile(`\d+$`)

	// VersionExtractRegex extracts a Go version from paths like ".../go1.25.4/bin/go"
	VersionExtractRegex = regexp.MustCompile(`go(\d+\.\d+(?:\.\d+)?(?:-?(?:rc|beta|alpha)\d*)?)`)
)

type releasesCacheKey struct {
	apiURL        string
	cacheDuration time.Duration
}

type releasesCacheEntry struct {
	releases []Release
	expires  time.Time
	fetching bool
	ready    chan struct{}
}

const (
	installMetadataFilename = ".govman-install.json"
	maxInstallMetadataSize  = 4 << 10
	maxReleasesResponseSize = 32 << 20
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

type installMetadata struct {
	Version     string    `json:"version"`
	InstalledAt time.Time `json:"installed_at"`
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
		if _, err := parseVersion(normalizeVersion(version)); err != nil {
			return nil, fmt.Errorf("Go releases API returned invalid version %q: %w", release.Version, err)
		}
		versions = append(versions, version)
	}

	sort.Slice(versions, func(i, j int) bool {
		comparison, _ := CompareVersions(versions[i], versions[j])
		return comparison > 0
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
		comparison, err := CompareVersions(version, "1.16")
		if err == nil && comparison < 0 {
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

	_, err := os.Stat(goBinary)
	if err != nil {
		return nil, fmt.Errorf("go binary not found in %s", installPath)
	}

	version := filepath.Base(installPath)
	version = strings.TrimPrefix(version, "go")
	if _, err := parseVersion(normalizeVersion(version)); err != nil {
		return nil, fmt.Errorf("invalid installed Go version directory %q: %w", filepath.Base(installPath), err)
	}

	installDate, err := readInstallDate(installPath, version)
	if err != nil {
		return nil, err
	}

	size, err := getDirSize(installPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate Go %s installation size: %w", version, err)
	}

	return &VersionInfo{
		Version:     version,
		Path:        installPath,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		InstallDate: installDate,
		Size:        size,
	}, nil
}

// WriteInstallMetadata records the completion time for a committed Go installation.
// The metadata is written atomically inside the installation directory.
func WriteInstallMetadata(installPath, version string, installedAt time.Time) error {
	normalizedVersion := normalizeVersion(version)
	if _, err := parseVersion(normalizedVersion); err != nil {
		return fmt.Errorf("invalid install metadata version %q: %w", version, err)
	}
	if installedAt.IsZero() {
		return fmt.Errorf("install timestamp cannot be zero")
	}
	info, err := os.Lstat(installPath)
	if err != nil {
		return fmt.Errorf("failed to inspect installation directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("installation path is not a regular directory: %s", installPath)
	}

	metadata := installMetadata{
		Version:     normalizedVersion,
		InstalledAt: installedAt.UTC(),
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to encode install metadata: %w", err)
	}
	encoded = append(encoded, '\n')

	tempFile, err := os.CreateTemp(installPath, ".govman-install-*")
	if err != nil {
		return fmt.Errorf("failed to create install metadata temp file: %w", err)
	}
	tempPath := tempFile.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()

	if err := tempFile.Chmod(0600); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to secure install metadata temp file: %w", err)
	}
	if _, err := tempFile.Write(encoded); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to write install metadata: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to sync install metadata: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close install metadata: %w", err)
	}

	metadataPath := filepath.Join(installPath, installMetadataFilename)
	if _, err := os.Lstat(metadataPath); err == nil {
		return fmt.Errorf("install metadata already exists: %s", metadataPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect install metadata path: %w", err)
	}
	if err := os.Rename(tempPath, metadataPath); err != nil {
		return fmt.Errorf("failed to commit install metadata: %w", err)
	}
	removeTemp = false
	return nil
}

func readInstallDate(installPath, version string) (time.Time, error) {
	metadataPath := filepath.Join(installPath, installMetadataFilename)
	metadataInfo, err := os.Lstat(metadataPath)
	if os.IsNotExist(err) {
		// Legacy installations predate explicit metadata. The installation
		// directory mtime reflects extraction/commit time more accurately than
		// the archive-preserved mtime of bin/go.
		installInfo, statErr := os.Lstat(installPath)
		if statErr != nil {
			return time.Time{}, fmt.Errorf("failed to inspect installation directory: %w", statErr)
		}
		if !installInfo.IsDir() || installInfo.Mode()&os.ModeSymlink != 0 {
			return time.Time{}, fmt.Errorf("installation path is not a regular directory: %s", installPath)
		}
		return installInfo.ModTime(), nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to inspect install metadata: %w", err)
	}
	if !metadataInfo.Mode().IsRegular() || metadataInfo.Mode()&os.ModeSymlink != 0 {
		return time.Time{}, fmt.Errorf("install metadata is not a regular file: %s", metadataPath)
	}
	if metadataInfo.Size() > maxInstallMetadataSize {
		return time.Time{}, fmt.Errorf("install metadata exceeds %d bytes", maxInstallMetadataSize)
	}

	file, err := os.Open(metadataPath)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to open install metadata: %w", err)
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxInstallMetadataSize+1))
	closeErr := file.Close()
	if readErr != nil {
		return time.Time{}, fmt.Errorf("failed to read install metadata: %w", readErr)
	}
	if closeErr != nil {
		return time.Time{}, fmt.Errorf("failed to close install metadata: %w", closeErr)
	}
	if len(data) > maxInstallMetadataSize {
		return time.Time{}, fmt.Errorf("install metadata exceeds %d bytes", maxInstallMetadataSize)
	}

	var metadata installMetadata
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&metadata); err != nil {
		return time.Time{}, fmt.Errorf("failed to decode install metadata: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return time.Time{}, fmt.Errorf("install metadata contains trailing data")
	}
	if metadata.Version != version {
		return time.Time{}, fmt.Errorf("install metadata version %q does not match directory version %q", metadata.Version, version)
	}
	if metadata.InstalledAt.IsZero() {
		return time.Time{}, fmt.Errorf("install metadata timestamp is missing")
	}
	return metadata.InstalledAt, nil
}

// CompareVersions compares two semantic version strings with prerelease awareness.
// Returns 1 if v1 > v2, -1 if v1 < v2, and 0 if equal. Invalid versions return an error.
func CompareVersions(v1, v2 string) (int, error) {
	v1Norm := normalizeVersion(v1)
	v2Norm := normalizeVersion(v2)
	parts1, err := parseVersion(v1Norm)
	if err != nil {
		return 0, fmt.Errorf("invalid version %q: %w", v1, err)
	}
	parts2, err := parseVersion(v2Norm)
	if err != nil {
		return 0, fmt.Errorf("invalid version %q: %w", v2, err)
	}

	// Compare version numbers
	for i := 0; i < 3; i++ {
		if parts1.numbers[i] > parts2.numbers[i] {
			return 1, nil
		} else if parts1.numbers[i] < parts2.numbers[i] {
			return -1, nil
		}
	}

	// Compare prerelease tags
	return comparePrerelease(parts1.prerelease, parts2.prerelease), nil
}

type versionParts struct {
	numbers    [3]int
	prerelease string
}

// parseVersion parses a normalized version into numeric components and a prerelease tag.
// Parameter version. Returns a versionParts struct.
func parseVersion(version string) (versionParts, error) {
	var parts versionParts

	matches := versionParseRegex.FindStringSubmatch(version)

	if len(matches) == 0 {
		return parts, fmt.Errorf("must use major.minor[.patch][-prerelease] format")
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return parts, fmt.Errorf("invalid major version: %w", err)
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return parts, fmt.Errorf("invalid minor version: %w", err)
	}
	parts.numbers[0] = major
	parts.numbers[1] = minor
	if matches[3] != "" {
		patch, err := strconv.Atoi(matches[3])
		if err != nil {
			return parts, fmt.Errorf("invalid patch version: %w", err)
		}
		parts.numbers[2] = patch
	}

	if matches[4] != "" {
		parts.prerelease = matches[4]
	}

	return parts, nil
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
	match := prereleaseNumberRegex.FindString(prerelease)
	if num, err := strconv.Atoi(match); err == nil {
		return num
	}
	return 0
}

// fetchReleasesWithConfig fetches releases JSON, caches results with expiry, and returns parsed data.
// Parameters: apiURL, cacheDuration. Returns []Release or an error.
func fetchReleasesWithConfig(apiURL string, cacheDuration time.Duration) ([]Release, error) {
	parsedURL, err := url.ParseRequestURI(apiURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid Go releases API URL %q", apiURL)
	}
	if cacheDuration < 0 {
		cacheDuration = 0
	}
	key := releasesCacheKey{apiURL: parsedURL.String(), cacheDuration: cacheDuration}

	for {
		cacheMutex.Lock()
		entry := releasesCache[key]
		if entry != nil && !entry.fetching && time.Now().Before(entry.expires) {
			result := cloneReleases(entry.releases)
			cacheMutex.Unlock()
			return result, nil
		}
		if entry != nil && entry.fetching {
			ready := entry.ready
			cacheMutex.Unlock()
			<-ready
			continue
		}
		entry = &releasesCacheEntry{fetching: true, ready: make(chan struct{})}
		releasesCache[key] = entry
		cacheMutex.Unlock()

		fetched, fetchErr := fetchReleases(parsedURL.String())

		cacheMutex.Lock()
		if fetchErr == nil {
			entry.releases = cloneReleases(fetched)
			entry.expires = time.Now().Add(cacheDuration)
		} else if releasesCache[key] == entry {
			delete(releasesCache, key)
		}
		entry.fetching = false
		close(entry.ready)
		cacheMutex.Unlock()

		if fetchErr != nil {
			return nil, fetchErr
		}
		return cloneReleases(fetched), nil
	}
}

func fetchReleases(apiURL string) ([]Release, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return fetchReleasesContext(ctx, releasesHTTPClient, apiURL)
}

func fetchReleasesContext(ctx context.Context, client *http.Client, apiURL string) ([]Release, error) {
	if client == nil {
		return nil, fmt.Errorf("Go releases HTTP client is nil")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create releases request: %w", err)
	}
	request.Header.Set("User-Agent", "govman")
	request.Header.Set("Accept", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to fetch releases: HTTP %d (%s)", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxReleasesResponseSize+1))
	closeErr := resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("failed to close releases response: %w", closeErr)
	}
	if len(body) > maxReleasesResponseSize {
		return nil, fmt.Errorf("Go releases response exceeds %d bytes", maxReleasesResponseSize)
	}

	var releases []Release
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases: %w", err)
	}
	return releases, nil
}

func cloneReleases(releases []Release) []Release {
	if releases == nil {
		return nil
	}
	clone := make([]Release, len(releases))
	for index, release := range releases {
		clone[index] = release
		clone[index].Files = append([]File(nil), release.Files...)
	}
	return clone
}

// getDirSize walks a directory and sums file sizes.
// Uses filepath.WalkDir for better performance (avoids unnecessary os.Stat calls).
// Parameter path. Returns total size in bytes or an error.
func getDirSize(path string) (int64, error) {
	var size int64

	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// ClearReleasesCache clears the in-memory releases cache.
// This is primarily used for testing to ensure a clean state.
func ClearReleasesCache() {
	cacheMutex.Lock()
	releasesCache = make(map[releasesCacheKey]*releasesCacheEntry)
	cacheMutex.Unlock()
}
