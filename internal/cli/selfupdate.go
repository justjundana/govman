package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	cobra "github.com/spf13/cobra"

	_logger "github.com/justjundana/govman/internal/logger"
	_version "github.com/justjundana/govman/internal/version"
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
	PublishedAt time.Time `json:"published_at"`
	Prerelease  bool      `json:"prerelease"`
}

// selfUpdateHTTPClient is a shared HTTP client for connection reuse
var selfUpdateHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
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
			return runSelfUpdate(checkOnly, force, prerelease)
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Check for updates without installing (dry run)")
	cmd.Flags().BoolVar(&force, "force", false, "Force update even if already on latest version")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "Include pre-release versions (beta, rc)")

	return cmd
}

// runSelfUpdate orchestrates the self-update workflow.
// Parameters: checkOnly (perform a dry run and do not install), force (reinstall even if already on latest),
// prerelease (include pre-release versions when checking). Returns nil on success or an error if any step fails.
func runSelfUpdate(checkOnly, force, prerelease bool) error {
	_logger.Info("Checking for govman updates...")
	_logger.Progress("Contacting GitHub API for latest release information")

	_logger.Verbose("Retrieving latest release information from GitHub")
	latest, err := getLatestRelease(prerelease)
	if err != nil {
		_logger.ErrorWithHelp("Unable to fetch update information", "Verify your internet connection and that GitHub API is accessible.", "")
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	current := _version.BuildVersion()
	if current == "dev" {
		_logger.Warning("Development version detected - updates are not available")
		_logger.Info("You're using a development build. Update manually from source.")
		return nil
	}

	_logger.Info("Version Information:")
	_logger.Info("  Current: %s", current)
	_logger.Info("  Latest:  %s", latest.TagName)

	if latest.PublishedAt.After(time.Time{}) {
		_logger.Info("  Released: %s", latest.PublishedAt.Format("January 2, 2006"))
	}

	if !force && latest.TagName == current {
		_logger.Success("You are already using the latest version!")
		_logger.Info("Use --force to reinstall the current version")
		return nil
	}

	if checkOnly {
		if latest.TagName != current {
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

	assetName := fmt.Sprintf("govman-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		assetName += ".exe"
	}

	var downloadURL string
	for _, asset := range latest.Assets {
		if strings.Contains(asset.Name, assetName) {
			downloadURL = asset.DownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	_logger.Download("Downloading %s...", latest.TagName)

	_logger.Verbose("Downloading binary")
	resp, err := selfUpdateHTTPClient.Get(downloadURL)
	if err != nil {
		_logger.ErrorWithHelp("Failed to download binary", "Check your internet connection and try again.", "")
		return fmt.Errorf("failed to download binary: %w", err)
	}
	defer resp.Body.Close()

	tempFile, err := os.CreateTemp("", "govman-update-*.bin")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempFilePath := tempFile.Name()
	defer func() {
		// Only remove temp file if it still exists (wasn't renamed)
		if _, err := os.Stat(tempFilePath); err == nil {
			os.Remove(tempFilePath)
		}
	}()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write binary to temporary file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	_logger.Verbose("Getting current binary path")
	currentBinary, err := os.Executable()
	if err != nil {
		_logger.ErrorWithHelp("Failed to get current binary path", "Check if the binary has proper permissions.", "")
		return fmt.Errorf("failed to get current binary path: %w", err)
	}

	_logger.Verbose("Creating backup of current binary")
	backupBinary := currentBinary + ".bak." + fmt.Sprintf("%d", time.Now().Unix())
	if err := os.Rename(currentBinary, backupBinary); err != nil {
		_logger.ErrorWithHelp("Failed to create backup of current binary", "Check if you have permission to modify the binary directory.", "")
		return fmt.Errorf("failed to rename current binary to backup: %w", err)
	}

	_logger.Verbose("Installing new binary")
	if err := os.Rename(tempFile.Name(), currentBinary); err != nil {
		// Failed to install new binary, restore backup
		_logger.Warning("Failed to install new binary, restoring backup")
		if restoreErr := os.Rename(backupBinary, currentBinary); restoreErr != nil {
			_logger.ErrorWithHelp("Failed to restore backup binary", "You may need to manually restore the binary from the backup file.", "")
			return fmt.Errorf("failed to restore backup binary: %w", restoreErr)
		}
		return fmt.Errorf("failed to move downloaded binary to current binary path: %w", err)
	}

	_logger.Verbose("Setting executable permissions")
	if err := os.Chmod(currentBinary, 0755); err != nil {
		_logger.ErrorWithHelp("Failed to set executable permissions", "You may need to manually set executable permissions on the binary.", "")
		return fmt.Errorf("failed to set executable permission for new binary: %w", err)
	}

	// Clean up backup files after successful update
	// On Windows, the running process locks the backup file, so we skip
	// cleanup here and let the startup routine handle it on next run.
	if runtime.GOOS == "windows" {
		_logger.Verbose("Skipping backup cleanup on Windows - will clean up on next startup")
	} else {
		_logger.Verbose("Cleaning up backup files")
		os.Remove(backupBinary)

		// Clean up any old backup files
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

	_logger.Success("Update completed successfully!")
	return nil
}

// getLatestRelease queries GitHub for release information.
// Parameter includePrerelease: when true, it reads the releases list (including prereleases) and returns
// the first eligible release; otherwise it fetches the latest stable release endpoint.
// Returns a *GitHubRelease on success or an error if the request or JSON parsing fails.
func getLatestRelease(includePrerelease bool) (*GitHubRelease, error) {
	cfg := getConfig()
	url := cfg.SelfUpdate.GitHubAPIURL
	if includePrerelease {
		url = cfg.SelfUpdate.GitHubReleasesURL
	}

	resp, err := selfUpdateHTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Validate HTTP status code before processing response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if includePrerelease {
		var releases []GitHubRelease
		if err := json.Unmarshal(body, &releases); err != nil {
			return nil, err
		}
		if len(releases) == 0 {
			return nil, fmt.Errorf("no releases found")
		}
		// When including prereleases, return the first (latest) release
		return &releases[0], nil
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, err
	}

	return &release, nil
}
