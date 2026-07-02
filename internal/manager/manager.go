package manager

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	_config "github.com/justjundana/govman/internal/config"
	_downloader "github.com/justjundana/govman/internal/downloader"
	_golang "github.com/justjundana/govman/internal/golang"
	_logger "github.com/justjundana/govman/internal/logger"
	_shell "github.com/justjundana/govman/internal/shell"
	_symlink "github.com/justjundana/govman/internal/symlink"
	_util "github.com/justjundana/govman/internal/util"
)

// VersionFormatRegex validates Go version format for security.
// Matches: 1.25.4, 1.25, 1.25rc1, 1.25.4-beta1, latest, stable
var VersionFormatRegex = regexp.MustCompile(`^(latest|stable|\d+\.\d+(\.\d+)?(-?(rc|beta|alpha)\d*)?)$`)

// ConcreteVersionRegex matches version identifiers that may safely be used as
// managed directory names. Aliases must be resolved before reaching a
// filesystem operation.
var ConcreteVersionRegex = regexp.MustCompile(`^\d+\.\d+(?:\.\d+)?(?:-?(?:rc|beta|alpha)\d*)?$`)

// ErrUnmanagedGo indicates that PATH resolves to a Go executable outside the
// configured govman installation root.
var ErrUnmanagedGo = errors.New("active Go executable is not managed by govman")

// ErrNoActiveVersion indicates that no managed session, local, or global Go version is active.
var ErrNoActiveVersion = errors.New("no managed Go version is active")

type Manager struct {
	config     *_config.Config
	downloader *_downloader.Downloader
	logger     *_logger.Logger
	shell      _shell.Shell
}

// New constructs a Manager with the provided configuration.
// It initializes a downloader and detects the user's shell.
func New(cfg *_config.Config) *Manager {
	return NewWithLogger(cfg, _logger.Get())
}

// NewWithLogger constructs a Manager whose service output is isolated to logger.
func NewWithLogger(cfg *_config.Config, logger *_logger.Logger) *Manager {
	if logger == nil {
		logger = _logger.New()
	}
	return &Manager{
		config:     cfg,
		downloader: _downloader.NewWithLogger(cfg, logger),
		logger:     logger,
		shell:      _shell.Detect(),
	}
}

// Install downloads and installs the specified Go version.
// version may be an exact string or "latest". Returns an error if resolution, download, or installation fails.
func (m *Manager) Install(version string) error {
	// Validate version format for security
	if !VersionFormatRegex.MatchString(version) {
		return fmt.Errorf("invalid version format: %s", version)
	}

	timer := m.logger.StartTimer("version resolution")
	resolvedVersion, err := m.ResolveVersion(version)
	if err != nil {
		m.logger.StopTimer(timer)
		return fmt.Errorf("failed to resolve version %s: %w", version, err)
	}
	m.logger.StopTimer(timer)

	installDir, err := m.versionDir(resolvedVersion)
	if err != nil {
		return err
	}

	m.logger.InternalProgress("Checking if version is already installed")
	if m.IsInstalled(resolvedVersion) {
		return fmt.Errorf("go version %s is already installed", resolvedVersion)
	}
	if _, err := os.Lstat(installDir); err == nil {
		return fmt.Errorf("go version %s has a corrupted or incomplete installation at %s; remove it with 'govman uninstall %s' before retrying", resolvedVersion, installDir, resolvedVersion)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect installation path for Go %s: %w", resolvedVersion, err)
	}

	m.logger.Info("Installing Go %s...", resolvedVersion)

	timer = m.logger.StartTimer("download URL retrieval")
	downloadURL, err := _golang.GetDownloadURLWithConfig(resolvedVersion,
		m.config.GoReleases.APIURL,
		m.config.GoReleases.CacheExpiry,
		m.config.GoReleases.DownloadURL)
	if err != nil {
		m.logger.StopTimer(timer)
		return fmt.Errorf("failed to get download URL: %w", err)
	}
	m.logger.StopTimer(timer)

	timer = m.logger.StartTimer("download and installation")
	if err := m.downloader.Download(downloadURL, installDir, resolvedVersion); err != nil {
		m.logger.StopTimer(timer)
		return fmt.Errorf("failed to download and install: %w", err)
	}
	if err := _golang.WriteInstallMetadata(installDir, resolvedVersion, time.Now()); err != nil {
		m.logger.StopTimer(timer)
		metadataErr := fmt.Errorf("failed to record installation metadata: %w", err)
		if cleanupErr := os.RemoveAll(installDir); cleanupErr != nil {
			return errors.Join(metadataErr, fmt.Errorf("failed to roll back installation: %w", cleanupErr))
		}
		return metadataErr
	}
	m.logger.StopTimer(timer)

	m.logger.Success("Go %s installed successfully", resolvedVersion)
	return nil
}

// Uninstall removes an installed Go version.
// Returns an error if the version is not installed, is active, or removal fails.
func (m *Manager) Uninstall(version string) error {
	installDir, err := m.versionDir(version)
	if err != nil {
		return err
	}

	m.logger.InternalProgress("Checking if version is installed")
	info, err := os.Lstat(installDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("go version %s is not installed", version)
		}
		return fmt.Errorf("failed to inspect Go %s installation: %w", version, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("go version %s is not installed", version)
	}

	m.logger.InternalProgress("Checking if version is currently active")
	current, err := m.Current()
	if err == nil && current == version {
		return fmt.Errorf("cannot uninstall currently active version %s", version)
	}
	if err != nil && !errors.Is(err, ErrNoActiveVersion) && !errors.Is(err, ErrUnmanagedGo) {
		return fmt.Errorf("failed to determine active Go version: %w", err)
	}
	if m.config.DefaultVersion == version {
		return fmt.Errorf("cannot uninstall default version %s; activate a different default version first", version)
	}
	localVersion, err := m.resolveLocalVersion()
	if err != nil {
		return fmt.Errorf("failed to read project-local version: %w", err)
	}
	if localVersion == version {
		return fmt.Errorf("cannot uninstall project-local version %s; change or remove %s first", version, m.config.AutoSwitch.ProjectFile)
	}

	m.logger.InternalProgress("Removing installation directory: %s", installDir)
	timer := m.logger.StartTimer("uninstallation")
	if err := os.RemoveAll(installDir); err != nil {
		m.logger.StopTimer(timer)
		return fmt.Errorf("failed to remove installation directory: %w", err)
	}
	m.logger.StopTimer(timer)

	m.logger.Success("Go %s uninstalled successfully", version)
	return nil
}

// Use activates a Go version for the current session, as default, or for the local project.
// setDefault sets it globally; setLocal writes a project version file. Returns an error if activation fails.
func (m *Manager) Use(version string, setDefault, setLocal bool) error {
	if setDefault && setLocal {
		return fmt.Errorf("default and local activation scopes are mutually exclusive")
	}
	if version == "default" {
		if m.config.DefaultVersion == "" {
			return fmt.Errorf("no default Go version is configured")
		}
		version = m.config.DefaultVersion
	} else {
		if !ConcreteVersionRegex.MatchString(version) {
			return fmt.Errorf("invalid concrete version format: %s", version)
		}
		// Validate version is installed
		m.logger.InternalProgress("Checking if version is installed")
		if !m.IsInstalled(version) {
			return fmt.Errorf("go version %s is not installed. Run 'govman install %s' first", version, version)
		}
	}

	versionDir, err := m.versionDir(version)
	if err != nil {
		return err
	}
	versionBinPath := filepath.Join(versionDir, "bin")

	switch {
	case setLocal:
		m.logger.InternalProgress("Setting local version for project")
		snapshot, err := snapshotFile(m.config.AutoSwitch.ProjectFile)
		if err != nil {
			return fmt.Errorf("failed to snapshot local version state: %w", err)
		}
		if err := m.setLocalVersion(version); err != nil {
			return fmt.Errorf("failed to set local version: %w", err)
		}
		if err := m.shell.ExecutePathCommand(versionBinPath); err != nil {
			if rollbackErr := restoreFile(m.config.AutoSwitch.ProjectFile, snapshot); rollbackErr != nil {
				return fmt.Errorf("failed to update PATH: %w; failed to restore local version: %v", err, rollbackErr)
			}
			return fmt.Errorf("failed to update PATH; local version was restored: %w", err)
		}
		m.logger.Success("Set Go %s as local version for this project", version)
		return nil

	case setDefault:
		m.logger.InternalProgress("Setting as system default version")
		oldDefault := m.config.DefaultVersion
		links, err := m.snapshotToolchainLinks(version)
		if err != nil {
			return err
		}
		m.logger.InternalProgress("Activating toolchain links for Go %s", version)
		timer := m.logger.StartTimer("symlink creation")
		if err := m.createSymlink(version); err != nil {
			m.logger.StopTimer(timer)
			if rollbackErr := restoreLinks(links); rollbackErr != nil {
				return fmt.Errorf("failed to activate toolchain links: %w; rollback failed: %v", err, rollbackErr)
			}
			return fmt.Errorf("failed to activate toolchain links: %w", err)
		}
		m.logger.StopTimer(timer)

		m.config.DefaultVersion = version
		if err := m.config.Save(); err != nil {
			m.config.DefaultVersion = oldDefault
			if rollbackErr := restoreLinks(links); rollbackErr != nil {
				return fmt.Errorf("failed to save default version: %w; failed to restore toolchain links: %v", err, rollbackErr)
			}
			return fmt.Errorf("failed to save default version: %w", err)
		}
		if err := m.shell.ExecutePathCommand(versionBinPath); err != nil {
			m.config.DefaultVersion = oldDefault
			configRollbackErr := m.config.Save()
			linkRollbackErr := restoreLinks(links)
			if configRollbackErr != nil || linkRollbackErr != nil {
				return fmt.Errorf("failed to update PATH: %w; config rollback: %v; link rollback: %v", err, configRollbackErr, linkRollbackErr)
			}
			return fmt.Errorf("failed to update PATH; default activation was restored: %w", err)
		}
		return nil

	default:
		return m.shell.ExecutePathCommand(versionBinPath)
	}
}

// Current returns the currently active Go version, checking session, local project, or global symlink.
// Returns the version string or an error if none is active or validation fails.
func (m *Manager) Current() (string, error) {
	sessionVersion, err := m.getCurrentSessionVersion()
	if err != nil {
		if errors.Is(err, ErrUnmanagedGo) {
			return "", err
		}
		if !errors.Is(err, exec.ErrNotFound) {
			return "", err
		}
		m.logger.Verbose("No Go executable found in PATH")
	} else if sessionVersion != "" {
		return sessionVersion, nil
	}

	localVersion, localErr := m.resolveLocalVersion()
	if localErr != nil {
		return "", fmt.Errorf("failed to read project-local version: %w", localErr)
	}
	if localVersion != "" {
		if !m.IsInstalled(localVersion) {
			return "", fmt.Errorf("local version %s specified in %s is not installed - run 'govman install %s' to install it",
				localVersion, m.config.AutoSwitch.ProjectFile, localVersion)
		}

		return localVersion, nil
	}

	// Check if there's a raw local version that doesn't have a matching installed version
	rawLocalVersion, rawLocalErr := m.ReadLocalVersionRaw()
	if rawLocalErr != nil {
		return "", fmt.Errorf("failed to read project-local version: %w", rawLocalErr)
	}
	if rawLocalVersion != "" {
		installedVersions, err := m.ListInstalled()
		if err != nil {
			return "", fmt.Errorf("failed to list installed versions: %w", err)
		}
		if len(installedVersions) > 0 {
			return "", fmt.Errorf("no installed version matches %s (from %s) - install a version with matching major.minor (e.g., 'govman install %s')",
				rawLocalVersion, m.config.AutoSwitch.ProjectFile, rawLocalVersion)
		}
		return "", fmt.Errorf("local version %s specified in %s but no Go versions are installed - run 'govman install %s' to install it",
			rawLocalVersion, m.config.AutoSwitch.ProjectFile, rawLocalVersion)
	}

	version, err := m.CurrentGlobal()
	if err != nil {
		return "", err
	}

	return version, nil
}

// CurrentGlobal resolves the active global version from the symlink and validates installation integrity.
// Returns the version or an error for missing/corrupt symlink or installation.
func (m *Manager) CurrentGlobal() (string, error) {
	symlinkPath := m.config.GetCurrentSymlink()

	// On Windows, the symlink for the current go binary is created with .exe suffix.
	// Mirror that here to check/read the correct path.
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(symlinkPath, ".exe") {
			symlinkPath += ".exe"
		}
	}

	linkInfo, err := os.Lstat(symlinkPath)
	if err != nil {
		if os.IsNotExist(err) {
			if m.config.DefaultVersion != "" {
				if m.IsInstalled(m.config.DefaultVersion) {
					return "", fmt.Errorf("no active Go version found - default version %s is configured but symlink is missing. Run 'govman use %s' to activate it",
						m.config.DefaultVersion, m.config.DefaultVersion)
				} else {
					return "", fmt.Errorf("no active Go version found - default version %s is configured but not installed. Run 'govman install %s' first, then 'govman use %s'",
						m.config.DefaultVersion, m.config.DefaultVersion, m.config.DefaultVersion)
				}
			}

			return "", fmt.Errorf("%w: no symlink found at %s and no default version configured", ErrNoActiveVersion, symlinkPath)
		}

		return "", fmt.Errorf("failed to check symlink at %s: %w - this may indicate a permissions issue or corrupted installation",
			symlinkPath, err)
	}

	if linkInfo.Mode()&os.ModeSymlink == 0 {
		return "", fmt.Errorf("expected symlink at %s but found %s instead - this may indicate a corrupted govman installation. Try running 'govman use <version>' to recreate the symlink",
			symlinkPath, linkInfo.Mode().Type().String())
	}

	target, err := os.Readlink(symlinkPath)
	if err != nil {
		return "", fmt.Errorf("failed to read symlink target from %s: %w - the symlink may be corrupted",
			symlinkPath, err)
	}

	// Use regex to extract version from the symlink target path
	// This is more robust than path manipulation across platforms
	matches := _golang.VersionExtractRegex.FindStringSubmatch(target)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract version from symlink target: %s - the symlink may be corrupted", target)
	}
	version := matches[1]

	expectedVersionDir, err := m.versionDir(version)
	if err != nil {
		return "", fmt.Errorf("invalid version in global symlink target: %w", err)
	}
	goExecutable, err := m.validateInstallation(version)
	if err != nil {
		return "", err
	}

	targetPath := target
	if !filepath.IsAbs(targetPath) {
		targetPath = filepath.Join(filepath.Dir(symlinkPath), targetPath)
	}
	targetPath, err = filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve global symlink target: %w", err)
	}
	expectedPath, err := filepath.Abs(goExecutable)
	if err != nil {
		return "", fmt.Errorf("failed to resolve expected Go executable: %w", err)
	}
	canonicalTarget, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve global symlink target: %w", err)
	}
	canonicalExpected, err := filepath.EvalSymlinks(expectedPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve expected Go executable: %w", err)
	}
	pathsEqual := filepath.Clean(canonicalTarget) == filepath.Clean(canonicalExpected)
	if runtime.GOOS == "windows" {
		pathsEqual = strings.EqualFold(filepath.Clean(canonicalTarget), filepath.Clean(canonicalExpected))
	}
	if !pathsEqual {
		return "", fmt.Errorf("global symlink target %s does not match managed executable %s", targetPath, expectedPath)
	}
	if _, err := os.Stat(expectedVersionDir); err != nil {
		return "", fmt.Errorf("failed to verify installation directory %s for Go %s: %w", expectedVersionDir, version, err)
	}

	return version, nil
}

// ListInstalled returns installed Go versions sorted in descending order.
// Returns the slice of versions or an error if the install directory cannot be read.
func (m *Manager) ListInstalled() ([]string, error) {
	entries, err := os.ReadDir(m.config.InstallDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}

		return nil, fmt.Errorf("failed to read install directory: %w", err)
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "go") {
			version := entry.Name()[2:]
			if ConcreteVersionRegex.MatchString(version) && m.IsInstalled(version) {
				versions = append(versions, version)
			}
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		comparison, _ := _golang.CompareVersions(versions[i], versions[j])
		return comparison > 0
	})

	return versions, nil
}

// ListRemote fetches available remote Go versions.
// includeUnstable controls inclusion of beta/rc versions. Returns the list or an error.
func (m *Manager) ListRemote(includeUnstable bool) ([]string, error) {
	return _golang.GetAvailableVersionsWithConfig(includeUnstable,
		m.config.GoReleases.APIURL,
		m.config.GoReleases.CacheExpiry)
}

// IsInstalled reports whether a given version is installed by checking its directory.
// Returns true if installed; false otherwise.
func (m *Manager) IsInstalled(version string) bool {
	_, err := m.validateInstallation(version)
	return err == nil
}

// Info returns metadata about an installed version.
// Returns VersionInfo or an error if the version is not installed or info retrieval fails.
func (m *Manager) Info(version string) (*_golang.VersionInfo, error) {
	if _, err := m.validateInstallation(version); err != nil {
		return nil, err
	}
	installDir, err := m.versionDir(version)
	if err != nil {
		return nil, err
	}
	return _golang.GetVersionInfo(installDir)
}

// Clean removes and recreates the cache directory.
// Returns an error if cleanup fails; nil on success.
func (m *Manager) Clean() error {
	if err := os.RemoveAll(m.config.CacheDir); err != nil {
		return fmt.Errorf("failed to clean cache: %w", err)
	}

	if err := os.MkdirAll(m.config.CacheDir, 0755); err != nil {
		return fmt.Errorf("failed to recreate cache directory: %w", err)
	}

	m.logger.Success("Cache cleaned successfully")
	return nil
}

// ResolveVersion resolves aliases and partial versions to a concrete version.
// "latest" becomes the newest stable; "major.minor" expands to the latest patch. Returns the resolved version or an error.
func (m *Manager) ResolveVersion(version string) (string, error) {
	if version == "latest" || version == "stable" {
		versions, err := m.ListRemote(false)
		if err != nil {
			return "", err
		}

		if len(versions) == 0 {
			return "", fmt.Errorf("no stable versions available")
		}

		return versions[0], nil
	}

	if strings.Count(version, ".") == 1 {
		versions, err := m.ListRemote(true)
		if err != nil {
			return "", err
		}

		prefix := version + "."
		for _, v := range versions {
			if strings.HasPrefix(v, prefix) {
				return v, nil
			}
		}
		return "", fmt.Errorf("no patch version found for %s", version)
	}

	if !ConcreteVersionRegex.MatchString(version) {
		return "", fmt.Errorf("invalid version format: %s", version)
	}

	return version, nil
}

type linkState struct {
	path   string
	exists bool
	target string
}

type fileState struct {
	exists bool
	data   []byte
	mode   os.FileMode
}

// createSymlink activates every executable in the selected Go toolchain's bin
// directory. This keeps go, gofmt, and any future official toolchain binaries
// on the same version.
func (m *Manager) createSymlink(version string) error {
	if _, err := m.validateInstallation(version); err != nil {
		return err
	}
	links, err := m.desiredToolchainLinks(version)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(m.config.GetBinPath(), 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}
	for destination := range links {
		info, err := os.Lstat(destination)
		if err == nil && info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("refusing to replace non-symlink toolchain path: %s", destination)
		}
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to inspect toolchain path %s: %w", destination, err)
		}
	}
	for destination, target := range links {
		if err := _symlink.Create(target, destination); err != nil {
			return fmt.Errorf("failed to activate %s: %w", filepath.Base(destination), err)
		}
	}
	if err := m.removeStaleToolchainLinks(links); err != nil {
		return err
	}
	return nil
}

func (m *Manager) desiredToolchainLinks(version string) (map[string]string, error) {
	versionDir, err := m.versionDir(version)
	if err != nil {
		return nil, err
	}
	binDir := filepath.Join(versionDir, "bin")
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read Go %s toolchain directory: %w", version, err)
	}
	links := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || entry.Name() == "govman" || entry.Name() == "govman.exe" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("failed to inspect toolchain executable %s: %w", entry.Name(), err)
		}
		if !info.Mode().IsRegular() || (runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0) {
			continue
		}
		links[filepath.Join(m.config.GetBinPath(), entry.Name())] = filepath.Join(binDir, entry.Name())
	}
	goName := "go"
	if runtime.GOOS == "windows" {
		goName += ".exe"
	}
	if _, ok := links[filepath.Join(m.config.GetBinPath(), goName)]; !ok {
		return nil, fmt.Errorf("Go %s toolchain does not contain %s", version, goName)
	}
	return links, nil
}

func (m *Manager) isManagedToolchainTarget(target string) bool {
	if !filepath.IsAbs(target) {
		return false
	}
	root, err := filepath.Abs(m.config.InstallDir)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != "." && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func (m *Manager) removeStaleToolchainLinks(desired map[string]string) error {
	entries, err := os.ReadDir(m.config.GetBinPath())
	if err != nil {
		return fmt.Errorf("failed to read govman bin directory: %w", err)
	}
	for _, entry := range entries {
		path := filepath.Join(m.config.GetBinPath(), entry.Name())
		if _, keep := desired[path]; keep || entry.Type()&os.ModeSymlink == 0 {
			continue
		}
		target, err := os.Readlink(path)
		if err != nil {
			return fmt.Errorf("failed to inspect existing toolchain link %s: %w", path, err)
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		if m.isManagedToolchainTarget(filepath.Clean(target)) {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove stale toolchain link %s: %w", path, err)
			}
		}
	}
	return nil
}

func (m *Manager) snapshotToolchainLinks(version string) ([]linkState, error) {
	desired, err := m.desiredToolchainLinks(version)
	if err != nil {
		return nil, err
	}
	paths := make(map[string]struct{}, len(desired))
	for path := range desired {
		paths[path] = struct{}{}
	}
	entries, err := os.ReadDir(m.config.GetBinPath())
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to inspect existing toolchain links: %w", err)
	}
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink == 0 {
			continue
		}
		path := filepath.Join(m.config.GetBinPath(), entry.Name())
		target, readErr := os.Readlink(path)
		if readErr != nil {
			return nil, readErr
		}
		absoluteTarget := target
		if !filepath.IsAbs(absoluteTarget) {
			absoluteTarget = filepath.Join(filepath.Dir(path), absoluteTarget)
		}
		if m.isManagedToolchainTarget(filepath.Clean(absoluteTarget)) {
			paths[path] = struct{}{}
		}
	}
	states := make([]linkState, 0, len(paths))
	for path := range paths {
		state := linkState{path: path}
		info, statErr := os.Lstat(path)
		if statErr == nil {
			if info.Mode()&os.ModeSymlink == 0 {
				return nil, fmt.Errorf("toolchain activation would overwrite a non-symlink path: %s", path)
			}
			state.exists = true
			state.target, statErr = os.Readlink(path)
		}
		if statErr != nil && !os.IsNotExist(statErr) {
			return nil, fmt.Errorf("failed to snapshot toolchain path %s: %w", path, statErr)
		}
		states = append(states, state)
	}
	return states, nil
}

func restoreLinks(states []linkState) error {
	var restoreErrors []error
	for _, state := range states {
		if err := os.Remove(state.path); err != nil && !os.IsNotExist(err) {
			restoreErrors = append(restoreErrors, err)
			continue
		}
		if state.exists {
			if err := os.Symlink(state.target, state.path); err != nil {
				restoreErrors = append(restoreErrors, err)
			}
		}
	}
	return errors.Join(restoreErrors...)
}

// setLocalVersion writes the project's autoswitch file with the specified version.
// Returns an error if the file write fails.
func (m *Manager) setLocalVersion(version string) error {
	if !ConcreteVersionRegex.MatchString(version) {
		return fmt.Errorf("invalid concrete version format: %s", version)
	}
	filename := m.config.AutoSwitch.ProjectFile
	return writeFileAtomic(filename, []byte(version+"\n"), 0644)
}

func snapshotFile(path string) (fileState, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return fileState{}, nil
	}
	if err != nil {
		return fileState{}, err
	}
	if !info.Mode().IsRegular() {
		return fileState{}, fmt.Errorf("refusing to modify non-regular file: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fileState{}, err
	}
	return fileState{exists: true, data: data, mode: info.Mode().Perm()}, nil
}

func restoreFile(path string, state fileState) error {
	if !state.exists {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return writeFileAtomic(path, state.data, state.mode)
}

func writeFileAtomic(path string, data []byte, defaultMode os.FileMode) (resultErr error) {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}
	mode := defaultMode
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refusing to replace non-regular file: %s", path)
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}
	tempFile, err := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	closed := false
	defer func() {
		if !closed {
			if closeErr := tempFile.Close(); resultErr == nil && closeErr != nil {
				resultErr = closeErr
			}
		}
		if resultErr != nil {
			if removeErr := os.Remove(tempPath); removeErr != nil && !os.IsNotExist(removeErr) {
				resultErr = errors.Join(resultErr, fmt.Errorf("failed to clean temporary file: %w", removeErr))
			}
		}
	}()
	if err := tempFile.Chmod(mode); err != nil {
		return err
	}
	if _, err := tempFile.Write(data); err != nil {
		return err
	}
	if err := tempFile.Sync(); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	closed = true
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	return nil
}

// versionDir validates a concrete version and returns an absolute path that is
// guaranteed to remain within the configured installation root.
func (m *Manager) versionDir(version string) (string, error) {
	if m == nil || m.config == nil {
		return "", fmt.Errorf("manager configuration is not initialized")
	}
	if !ConcreteVersionRegex.MatchString(version) {
		return "", fmt.Errorf("invalid concrete version format: %s", version)
	}

	root, err := filepath.Abs(m.config.InstallDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve install directory: %w", err)
	}
	candidate := filepath.Join(root, "go"+version)
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return "", fmt.Errorf("failed to validate version path: %w", err)
	}
	if rel == "." || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("version path escapes install directory: %s", version)
	}
	return candidate, nil
}

// validateInstallation verifies that a managed version directory contains a
// usable Go executable and is not itself a symlink.
func (m *Manager) validateInstallation(version string) (string, error) {
	installDir, err := m.versionDir(version)
	if err != nil {
		return "", err
	}

	dirInfo, err := os.Lstat(installDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("go version %s is not installed", version)
		}
		return "", fmt.Errorf("failed to inspect Go %s installation: %w", version, err)
	}
	if !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("go version %s installation root is not a managed directory", version)
	}

	goExecutable := filepath.Join(installDir, "bin", "go")
	if runtime.GOOS == "windows" {
		goExecutable += ".exe"
	}
	binInfo, err := os.Lstat(goExecutable)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("go %s installation is incomplete: executable not found at %s", version, goExecutable)
		}
		return "", fmt.Errorf("failed to inspect Go %s executable: %w", version, err)
	}
	if !binInfo.Mode().IsRegular() {
		return "", fmt.Errorf("go %s executable is not a regular file: %s", version, goExecutable)
	}
	if runtime.GOOS != "windows" && binInfo.Mode().Perm()&0111 == 0 {
		return "", fmt.Errorf("go %s executable is not executable: %s", version, goExecutable)
	}

	return goExecutable, nil
}

// getLocalVersionRaw reads the project's autoswitch file and returns the raw version string.
// Returns an empty string if the file does not exist or cannot be read.
func (m *Manager) getLocalVersionRaw() string {
	version, _ := m.ReadLocalVersionRaw()
	return version
}

// ReadLocalVersionRaw reads the project version file, distinguishing absence
// from permission and I/O failures.
func (m *Manager) ReadLocalVersionRaw() (string, error) {
	filename := m.config.AutoSwitch.ProjectFile
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// GetLocalVersionRaw returns the raw version string from the project's autoswitch file.
// Returns an empty string if the file does not exist or cannot be read.
func (m *Manager) GetLocalVersionRaw() string {
	return m.getLocalVersionRaw()
}

// getLocalVersion reads the project's autoswitch file and returns the best matching installed version.
// It uses flexible version matching based on major.minor version (e.g., "1.25" matches "1.25.4").
// Returns an empty string if the file does not exist or no matching version is installed.
func (m *Manager) getLocalVersion() string {
	version, _ := m.resolveLocalVersion()
	return version
}

func (m *Manager) resolveLocalVersion() (string, error) {
	rawVersion, err := m.ReadLocalVersionRaw()
	if err != nil {
		return "", err
	}
	if rawVersion == "" {
		return "", nil
	}

	// Get all installed versions
	installedVersions, err := m.ListInstalled()
	if err != nil || len(installedVersions) == 0 {
		return "", err
	}

	// Find a matching version based on major.minor
	matchedVersion, err := _util.FindBestMatchingVersion(rawVersion, installedVersions)
	if err != nil {
		return "", nil
	}

	return matchedVersion, nil
}

// DefaultVersion returns the configured default version string.
func (m *Manager) DefaultVersion() string {
	return m.config.DefaultVersion
}

// CurrentActivationMethod returns the activation method for the currently active Go version.
// Returns "session-only", "project-local", or "system-default" based on how the current version is activated.
func (m *Manager) CurrentActivationMethod() string {
	sessionVersion, err := m.getCurrentSessionVersion()
	if err == nil && sessionVersion != "" {
		if localVersion := m.getLocalVersion(); localVersion != "" && localVersion == sessionVersion {
			return "project-local"
		}

		globalVersion, err := m.CurrentGlobal()
		if err == nil && globalVersion == sessionVersion {
			return "system-default"
		}

		return "session-only"
	}

	if localVersion := m.getLocalVersion(); localVersion != "" {
		return "project-local"
	}

	return "system-default"
}

// getCurrentSessionVersion resolves the actual executable selected by PATH and
// only reports it when its canonical location belongs to a valid managed
// installation.
func (m *Manager) getCurrentSessionVersion() (string, error) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("failed to find Go executable in PATH: %w", err)
	}
	canonicalPath, err := filepath.EvalSymlinks(goPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve active Go executable %s: %w", goPath, err)
	}
	canonicalPath, err = filepath.Abs(canonicalPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve active Go executable path: %w", err)
	}
	root, err := filepath.Abs(m.config.InstallDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve managed installation root: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("failed to canonicalize managed installation root: %w", err)
	}
	relativePath, err := filepath.Rel(root, canonicalPath)
	if err != nil || filepath.IsAbs(relativePath) || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("%w: %s", ErrUnmanagedGo, canonicalPath)
	}
	components := strings.Split(filepath.Clean(relativePath), string(os.PathSeparator))
	if len(components) != 3 || components[1] != "bin" || (components[2] != "go" && components[2] != "go.exe") || !strings.HasPrefix(components[0], "go") {
		return "", fmt.Errorf("%w: %s", ErrUnmanagedGo, canonicalPath)
	}
	version := strings.TrimPrefix(components[0], "go")
	expectedExecutable, err := m.validateInstallation(version)
	if err != nil {
		return "", err
	}
	expectedCanonical, err := filepath.EvalSymlinks(expectedExecutable)
	if err != nil {
		return "", err
	}
	pathsEqual := filepath.Clean(expectedCanonical) == filepath.Clean(canonicalPath)
	if runtime.GOOS == "windows" {
		pathsEqual = strings.EqualFold(filepath.Clean(expectedCanonical), filepath.Clean(canonicalPath))
	}
	if !pathsEqual {
		return "", fmt.Errorf("%w: %s", ErrUnmanagedGo, canonicalPath)
	}
	return version, nil
}
