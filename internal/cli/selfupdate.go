package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	cobra "github.com/spf13/cobra"

	_golang "github.com/justjundana/govman/internal/golang"
	_logger "github.com/justjundana/govman/internal/logger"
	_version "github.com/justjundana/govman/internal/version"
)

type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	Assets      []GitHubAsset `json:"assets"`
	Draft       bool          `json:"draft"`
	PublishedAt time.Time     `json:"published_at"`
	Prerelease  bool          `json:"prerelease"`
}

type GitHubAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

const (
	maxGitHubAPIResponseSize = 4 << 20
	maxChecksumManifestSize  = 4 << 20
	maxSelfUpdateBinarySize  = 200 << 20
	maxVersionOutputSize     = 64 << 10
)

var releaseTagRegex = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:-(?:alpha|beta|rc)\d*)?$`)

// selfUpdateHTTPClient is a shared HTTP client for connection reuse
var selfUpdateHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(request *http.Request, _ []*http.Request) error {
		return validateReleaseURL(request.URL.String())
	},
}

// newSelfUpdateCmd creates the 'selfupdate' Cobra command.
// It defines flags: checkOnly (only check for updates), force (reinstall even if on latest),
// and prerelease (include pre-release versions). Returns the configured *cobra.Command that runs runSelfUpdate.
func newSelfUpdateCmd() *cobra.Command {
	var (
		checkOnly  bool
		force      bool
		prerelease bool
	)

	cmd := &cobra.Command{
		Use:   "selfupdate",
		Short: "Update govman to the latest version with smart management",
		Long: `Automatically check for and install the latest version of govman.

Smart Update Features:
  • Automatic platform detection and binary selection
  • Safe backup and rollback on failure
  • Integrity verification and secure downloads
  • Support for stable and pre-release versions
  • Non-disruptive updates with permission handling
  • Detailed release notes and changelog display

Examples:
  govman selfupdate                    # Update to latest stable
  govman selfupdate --check            # Check without installing
  govman selfupdate --prerelease       # Include pre-releases
  govman selfupdate --force            # Force update even if latest`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSelfUpdateContext(cmd.Context(), checkOnly, force, prerelease)
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Check for updates without installing (dry run)")
	cmd.Flags().BoolVar(&force, "force", false, "Force update even if already on latest version")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "Include pre-release versions (beta, rc)")

	return cmd
}

// findAssetURL finds the download URL for the current platform from the release assets.
func findAssetURL(latest *GitHubRelease) (string, error) {
	asset, err := findReleaseAsset(latest, platformAssetName())
	if err != nil {
		return "", err
	}
	return asset.DownloadURL, nil
}

func platformAssetName() string {
	assetName := fmt.Sprintf("govman-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		assetName += ".exe"
	}
	return assetName
}

func findReleaseAsset(release *GitHubRelease, assetName string) (*GitHubAsset, error) {
	if release == nil {
		return nil, fmt.Errorf("release metadata is missing")
	}
	var match *GitHubAsset
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			if match != nil {
				return nil, fmt.Errorf("release contains duplicate asset %q", assetName)
			}
			assetCopy := asset
			match = &assetCopy
		}
	}
	if match == nil {
		return nil, fmt.Errorf("release asset %q was not found", assetName)
	}
	if err := validateReleaseURL(match.DownloadURL); err != nil {
		return nil, fmt.Errorf("invalid URL for release asset %q: %w", assetName, err)
	}
	return match, nil
}

func downloadChecksumManifest(ctx context.Context, release *GitHubRelease, assetName string) (string, error) {
	manifestAsset, err := findReleaseAsset(release, "checksums.txt")
	if err != nil {
		return "", fmt.Errorf("release cannot be verified: %w", err)
	}
	body, err := fetchLimited(ctx, manifestAsset.DownloadURL, maxChecksumManifestSize, "text/plain")
	if err != nil {
		return "", fmt.Errorf("failed to download checksum manifest: %w", err)
	}
	return checksumForAsset(string(body), assetName)
}

func checksumForAsset(manifest, assetName string) (string, error) {
	var checksum string
	for lineNumber, line := range strings.Split(manifest, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return "", fmt.Errorf("invalid checksum manifest line %d", lineNumber+1)
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name != assetName {
			continue
		}
		if checksum != "" {
			return "", fmt.Errorf("checksum manifest contains duplicate entry for %s", assetName)
		}
		if len(fields[0]) != sha256.Size*2 {
			return "", fmt.Errorf("invalid SHA-256 checksum for %s", assetName)
		}
		if _, err := hex.DecodeString(fields[0]); err != nil {
			return "", fmt.Errorf("invalid SHA-256 checksum for %s", assetName)
		}
		checksum = strings.ToLower(fields[0])
	}
	if checksum == "" {
		return "", fmt.Errorf("checksum manifest does not contain %s", assetName)
	}
	return checksum, nil
}

// downloadBinary downloads, bounds, and verifies a release binary before it is executable.
func downloadBinary(ctx context.Context, asset *GitHubAsset, binaryDir, expectedChecksum string) (string, error) {
	if asset == nil {
		return "", fmt.Errorf("binary release asset is missing")
	}
	if asset.Size < 0 || asset.Size > maxSelfUpdateBinarySize {
		return "", fmt.Errorf("release binary size %d exceeds the allowed limit", asset.Size)
	}
	if err := validateReleaseURL(asset.DownloadURL); err != nil {
		return "", err
	}
	request, err := newSelfUpdateRequest(ctx, asset.DownloadURL, "application/octet-stream")
	if err != nil {
		return "", err
	}
	response, err := selfUpdateHTTPClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to download binary: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return "", fmt.Errorf("failed to download binary: HTTP %d (%s)", response.StatusCode, response.Status)
	}
	if response.ContentLength > maxSelfUpdateBinarySize {
		response.Body.Close()
		return "", fmt.Errorf("release binary exceeds the allowed size")
	}

	pattern := "govman-update-*"
	if runtime.GOOS == "windows" {
		pattern += ".exe"
	}
	tempFile, err := os.CreateTemp(binaryDir, pattern)
	if err != nil {
		response.Body.Close()
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempPath := tempFile.Name()
	succeeded := false
	defer func() {
		if !succeeded {
			os.Remove(tempPath)
		}
	}()

	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(tempFile, hasher), io.LimitReader(response.Body, maxSelfUpdateBinarySize+1))
	responseCloseErr := response.Body.Close()
	fileCloseErr := tempFile.Close()
	if copyErr != nil {
		return "", fmt.Errorf("failed to write binary to temporary file: %w", copyErr)
	}
	if responseCloseErr != nil {
		return "", fmt.Errorf("failed to close binary response: %w", responseCloseErr)
	}
	if fileCloseErr != nil {
		return "", fmt.Errorf("failed to close temporary file: %w", fileCloseErr)
	}
	if written > maxSelfUpdateBinarySize {
		return "", fmt.Errorf("release binary exceeds the allowed size")
	}
	if asset.Size > 0 && written != asset.Size {
		return "", fmt.Errorf("release binary size mismatch: received %d, expected %d", written, asset.Size)
	}
	actualChecksum := fmt.Sprintf("%x", hasher.Sum(nil))
	if !strings.EqualFold(actualChecksum, expectedChecksum) {
		return "", fmt.Errorf("release binary checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}
	if err := selfUpdateChmod(tempPath, 0755); err != nil {
		return "", fmt.Errorf("failed to set executable permission on verified binary: %w", err)
	}
	succeeded = true
	return tempPath, nil
}

var (
	selfUpdateRename         = os.Rename
	selfUpdateRemove         = os.Remove
	selfUpdateChmod          = os.Chmod
	selfUpdateValidateBinary = validateDownloadedBinary
)

func validateReleaseURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("invalid release URL %q", rawURL)
	}
	if parsed.Scheme == "https" {
		return nil
	}
	host := parsed.Hostname()
	ip := net.ParseIP(host)
	if parsed.Scheme == "http" && (strings.EqualFold(host, "localhost") || (ip != nil && ip.IsLoopback())) {
		return nil
	}
	return fmt.Errorf("release URL must use HTTPS")
}

func newSelfUpdateRequest(ctx context.Context, rawURL, accept string) (*http.Request, error) {
	if err := validateReleaseURL(rawURL); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create update request: %w", err)
	}
	request.Header.Set("User-Agent", "govman/"+_version.BuildVersion())
	if accept != "" {
		request.Header.Set("Accept", accept)
	}
	return request, nil
}

func fetchLimited(ctx context.Context, rawURL string, limit int64, accept string) ([]byte, error) {
	request, err := newSelfUpdateRequest(ctx, rawURL, accept)
	if err != nil {
		return nil, err
	}
	response, err := selfUpdateHTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("HTTP %d (%s)", response.StatusCode, response.Status)
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, limit+1))
	closeErr := response.Body.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return body, nil
}

type limitedCommandOutput struct {
	builder   strings.Builder
	remaining int
	exceeded  bool
}

func (output *limitedCommandOutput) Write(data []byte) (int, error) {
	originalLength := len(data)
	if len(data) > output.remaining {
		data = data[:max(output.remaining, 0)]
		output.exceeded = true
	}
	if len(data) > 0 {
		_, _ = output.builder.Write(data)
		output.remaining -= len(data)
	}
	return originalLength, nil
}

func validateDownloadedBinary(binaryPath, targetVersion string) error {
	if !releaseTagRegex.MatchString(targetVersion) {
		return fmt.Errorf("invalid target release tag %q", targetVersion)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binaryPath, "--version")
	output := &limitedCommandOutput{remaining: maxVersionOutputSize}
	command.Stdout = output
	command.Stderr = output
	if err := command.Run(); err != nil {
		return fmt.Errorf("verified binary failed --version validation: %w", err)
	}
	if output.exceeded {
		return fmt.Errorf("verified binary produced excessive version output")
	}
	expected := strings.TrimPrefix(targetVersion, "v")
	versionMatcher := regexp.MustCompile(`(?:^|[^0-9])v?(\d+\.\d+\.\d+(?:-(?:alpha|beta|rc)\d*)?)(?:$|[^0-9])`)
	matches := versionMatcher.FindStringSubmatch(output.builder.String())
	if len(matches) != 2 || matches[1] != expected {
		return fmt.Errorf("verified binary version output %q does not match target %s", strings.TrimSpace(output.builder.String()), targetVersion)
	}
	return nil
}

// replaceBinary replaces the current binary and restores the backup on every failed validation path.
func replaceBinary(currentBinary, tempFilePath, targetVersion string) (string, error) {
	if err := selfUpdateChmod(tempFilePath, 0755); err != nil {
		return "", fmt.Errorf("failed to set executable permission on verified binary: %w", err)
	}
	if err := selfUpdateValidateBinary(tempFilePath, targetVersion); err != nil {
		return "", err
	}
	backupBinary := currentBinary + ".bak." + fmt.Sprintf("%d", time.Now().UnixNano())
	if err := selfUpdateRename(currentBinary, backupBinary); err != nil {
		_logger.ErrorWithHelp("Failed to create backup of current binary", "Check if you have permission to modify the binary directory.")
		return "", fmt.Errorf("failed to rename current binary to backup: %w", err)
	}

	if err := selfUpdateRename(tempFilePath, currentBinary); err != nil {
		_logger.Warning("Failed to install new binary, restoring backup")
		if restoreErr := selfUpdateRename(backupBinary, currentBinary); restoreErr != nil {
			_logger.ErrorWithHelp("Failed to restore backup binary", "You may need to manually restore the binary from the backup file.")
			return "", fmt.Errorf("failed to install new binary: %w; failed to restore backup binary: %v", err, restoreErr)
		}
		return "", fmt.Errorf("failed to move downloaded binary to current binary path: %w", err)
	}

	if err := selfUpdateValidateBinary(currentBinary, targetVersion); err != nil {
		if removeErr := selfUpdateRemove(currentBinary); removeErr != nil && !os.IsNotExist(removeErr) {
			return "", fmt.Errorf("installed binary validation failed: %w; failed to remove invalid binary: %v", err, removeErr)
		}
		if restoreErr := selfUpdateRename(backupBinary, currentBinary); restoreErr != nil {
			return "", fmt.Errorf("installed binary validation failed: %w; failed to restore backup: %v", err, restoreErr)
		}
		return "", fmt.Errorf("installed binary validation failed and previous binary was restored: %w", err)
	}

	return backupBinary, nil
}

// cleanupBackupFiles removes old backup files from the binary directory.
func cleanupBackupFiles(currentBinary, backupBinary string) {
	if runtime.GOOS == "windows" {
		_logger.Verbose("Skipping backup cleanup on Windows - will clean up on next startup")
		return
	}

	_logger.Verbose("Cleaning up backup files")
	os.Remove(backupBinary)

	dir := filepath.Dir(currentBinary)
	baseName := filepath.Base(currentBinary)
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), baseName+".bak.") {
			oldBackup := filepath.Join(dir, entry.Name())
			os.Remove(oldBackup)
			_logger.Verbose("Removed old backup: %s", entry.Name())
		}
	}
}

// runSelfUpdate orchestrates the self-update workflow.
// Parameters: checkOnly (perform a dry run and do not install), force (reinstall even if already on latest),
// prerelease (include pre-release versions when checking). Returns nil on success or an error if any step fails.
func runSelfUpdate(checkOnly, force, prerelease bool) error {
	return runSelfUpdateContext(context.Background(), checkOnly, force, prerelease)
}

func runSelfUpdateContext(ctx context.Context, checkOnly, force, prerelease bool) error {
	_logger.Info("Checking for govman updates...")
	versionInfo := _version.Get()
	current := versionInfo.Version
	if current == "dev" {
		_logger.Warning("Development version detected - updates are not available")
		_logger.Info("You're using a development build. Update manually from source.")
		return nil
	}
	if !releaseTagRegex.MatchString(current) {
		return fmt.Errorf("current build version %q is not a valid release version", current)
	}

	_logger.Progress("Contacting GitHub API for latest release information")
	_logger.Verbose("Retrieving latest release information from GitHub")
	latest, err := getLatestReleaseContext(ctx, prerelease)
	if err != nil {
		_logger.ErrorWithHelp("Unable to fetch update information", "Verify your internet connection and that GitHub API is accessible.")
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	if err := validateReleaseMetadata(latest, prerelease); err != nil {
		return err
	}

	_logger.Info("Version Information:")
	_logger.Info("  Current: %s", current)
	_logger.Info("  Latest:  %s", latest.TagName)

	if latest.PublishedAt.After(time.Time{}) {
		_logger.Info("  Released: %s", latest.PublishedAt.Format("January 2, 2006"))
	}

	comparison := _golang.CompareVersions(latest.TagName, current)
	if comparison < 0 {
		_logger.Warning("Installed version %s is newer than latest eligible release %s; refusing to downgrade", current, latest.TagName)
		return nil
	}
	if !force && comparison == 0 {
		_logger.Success("You are already using the latest version!")
		_logger.Info("Use --force to reinstall the current version")
		return nil
	}

	if checkOnly {
		if comparison > 0 {
			_logger.Info("A new version is available: %s → %s", current, latest.TagName)
			if latest.Body != "" {
				_logger.Info("Release Notes:")
				_logger.Info(strings.Repeat("─", 40))
				_logger.Info("%s", latest.Body)
				_logger.Info(strings.Repeat("─", 40))
			}
			_logger.Info("Run 'govman selfupdate' to install this version")
		} else {
			_logger.Success("No updates available - you're on the latest version")
		}
		return nil
	}

	assetName := platformAssetName()
	binaryAsset, err := findReleaseAsset(latest, assetName)
	if err != nil {
		return err
	}
	expectedChecksum, err := downloadChecksumManifest(ctx, latest, assetName)
	if err != nil {
		return err
	}

	_logger.Download("Downloading %s...", latest.TagName)
	_logger.Verbose("Getting current binary path")
	currentBinary, err := os.Executable()
	if err != nil {
		_logger.ErrorWithHelp("Failed to get current binary path", "Check if the binary has proper permissions.")
		return fmt.Errorf("failed to get current binary path: %w", err)
	}
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return fmt.Errorf("failed to resolve current binary path: %w", err)
	}
	currentInfo, err := os.Lstat(currentBinary)
	if err != nil || !currentInfo.Mode().IsRegular() {
		return fmt.Errorf("current binary path is not a regular file: %s", currentBinary)
	}

	_logger.Verbose("Downloading binary")
	tempFilePath, err := downloadBinary(ctx, binaryAsset, filepath.Dir(currentBinary), expectedChecksum)
	if err != nil {
		return err
	}
	defer func() {
		if _, err := os.Stat(tempFilePath); err == nil {
			os.Remove(tempFilePath)
		}
	}()

	_logger.Verbose("Installing new binary")
	backupBinary, err := replaceBinary(currentBinary, tempFilePath, latest.TagName)
	if err != nil {
		return err
	}

	cleanupBackupFiles(currentBinary, backupBinary)

	_logger.Success("Update completed successfully!")
	return nil
}

// getLatestRelease queries GitHub for release information.
// Parameter includePrerelease: when true, it reads the releases list (including prereleases) and returns
// the first eligible release; otherwise it fetches the latest stable release endpoint.
// Returns a *GitHubRelease on success or an error if the request or JSON parsing fails.
func getLatestRelease(includePrerelease bool) (*GitHubRelease, error) {
	return getLatestReleaseContext(context.Background(), includePrerelease)
}

func getLatestReleaseContext(ctx context.Context, includePrerelease bool) (*GitHubRelease, error) {
	cfg := getConfig()
	if cfg == nil {
		return nil, fmt.Errorf("configuration is not initialized")
	}
	url := cfg.SelfUpdate.GitHubAPIURL
	if includePrerelease {
		url = cfg.SelfUpdate.GitHubReleasesURL
	}

	body, err := fetchLimited(ctx, url, maxGitHubAPIResponseSize, "application/vnd.github+json")
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}

	if includePrerelease {
		var releases []GitHubRelease
		if err := json.Unmarshal(body, &releases); err != nil {
			return nil, err
		}
		for index := range releases {
			if !releases[index].Draft {
				if err := validateReleaseMetadata(&releases[index], true); err != nil {
					return nil, err
				}
				return &releases[index], nil
			}
		}
		return nil, fmt.Errorf("no eligible releases found")
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, err
	}
	if err := validateReleaseMetadata(&release, false); err != nil {
		return nil, err
	}
	return &release, nil
}

func validateReleaseMetadata(release *GitHubRelease, includePrerelease bool) error {
	if release == nil {
		return fmt.Errorf("release metadata is missing")
	}
	if release.Draft {
		return fmt.Errorf("release %q is still a draft", release.TagName)
	}
	if !releaseTagRegex.MatchString(release.TagName) {
		return fmt.Errorf("release tag %q is not valid semantic version metadata", release.TagName)
	}
	if release.Prerelease && !includePrerelease {
		return fmt.Errorf("GitHub returned prerelease %s for a stable update request", release.TagName)
	}
	tagIsPrerelease := strings.Contains(release.TagName, "-alpha") || strings.Contains(release.TagName, "-beta") || strings.Contains(release.TagName, "-rc")
	if tagIsPrerelease != release.Prerelease {
		return fmt.Errorf("release tag %s and prerelease metadata are inconsistent", release.TagName)
	}
	return nil
}
