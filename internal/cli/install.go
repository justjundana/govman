package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cobra "github.com/spf13/cobra"

	_logger "github.com/justjundana/govman/internal/logger"
	_manager "github.com/justjundana/govman/internal/manager"
	_util "github.com/justjundana/govman/internal/util"
)

// installVersions performs the installation loop and returns results.
func installVersions(mgr *_manager.Manager, versions []string) (successful []string, failures []error) {
	for i, version := range versions {
		_logger.Info("[%d/%d] Installing Go %s...", i+1, len(versions), version)
		if err := mgr.Install(version); err != nil {
			failures = append(failures, fmt.Errorf("Go %s: %w", version, err))
			_logger.Warning("Failed to install Go %s: %v", version, err)
			continue
		}
		successful = append(successful, version)
		_logger.Success("Successfully installed Go %s", version)
	}
	return
}

// uninstallVersions performs the uninstallation loop and returns results.
func uninstallVersions(mgr *_manager.Manager, versions []string, current string) (successful []string, totalFreedSpace int64, failures []error) {
	for i, version := range versions {
		_logger.Info("[%d/%d] Uninstalling Go %s...", i+1, len(versions), version)

		if current == version {
			_logger.Warning("Cannot uninstall currently active Go version %s", version)
			failures = append(failures, fmt.Errorf("Go %s: cannot uninstall active version", version))
			continue
		}

		info, err := mgr.Info(version)
		if err != nil {
			_logger.Warning("Go version %s is not installed or information is unavailable", version)
			failures = append(failures, fmt.Errorf("Go %s: %w", version, err))
			continue
		}

		_logger.Progress("Removing installation directory and associated files")
		if err = mgr.Uninstall(version); err != nil {
			_logger.Warning("Failed to uninstall Go %s: %v", version, err)
			failures = append(failures, fmt.Errorf("Go %s: %w", version, err))
			continue
		}

		successful = append(successful, version)
		totalFreedSpace += info.Size
		_logger.Success("Successfully uninstalled Go %s", version)
	}
	return
}

// newInstallCmd creates the 'install' Cobra command to download and install one or more Go versions.
// Versions are provided as positional args (e.g., latest, 1.25.1). Returns a *cobra.Command that installs each version and reports results.
func newInstallCmd() *cobra.Command {
	var includeUnstable bool
	var skipConfirm bool

	cmd := &cobra.Command{
		Use:   "install [version...]",
		Short: "Install Go versions with intelligent download management",
		Long: `Download and install one or more Go versions from official releases.

Features:
  • Lightning-fast parallel downloads with resume capability
  • Automatic integrity verification and checksum validation
  • Smart caching to avoid re-downloading existing archives
  • Support for latest, stable, and pre-release versions
  • Batch installation with detailed progress tracking
  • Automatic cleanup of temporary files on completion
  • Wildcard pattern support for batch installation (e.g., 1.14.*)

Examples:
  govman install latest              # Latest stable release
  govman install 1.25.1              # Specific version
  govman install 1.25.1 1.20.12      # Multiple versions
  govman install 1.22rc1             # Pre-release version
  govman install '1.14.*'            # All 1.14.x stable versions (quote the pattern!)
  govman install '1.14.*' --unstable # All 1.14.x versions including beta/rc`,
		Args: usageArgs(cobra.MinimumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := _manager.New(getConfig())

			// Expand wildcard patterns in args
			expandedVersions, err := expandInstallPatterns(args, mgr, includeUnstable)
			if err != nil {
				return err
			}

			if len(expandedVersions) == 0 {
				_logger.Warning("No versions matched the specified pattern(s)")
				return fmt.Errorf("no versions to install")
			}

			// Show confirmation for pattern-based installation
			if hasWildcardPattern(args) && !skipConfirm {
				if getConfig().Quiet {
					return withUsageHelp(cmd, fmt.Errorf("quiet batch installation requires --yes"), "Pass --yes to confirm without an interactive prompt.")
				}
				versionType := "stable"
				if includeUnstable {
					versionType = "unstable/prerelease"
				}

				_logger.Info("The following %d %s version(s) will be installed:", len(expandedVersions), versionType)
				for _, v := range expandedVersions {
					_logger.Info("  • Go %s", v)
				}
				_logger.Info("")

				if !confirmAction("Proceed with installation?") {
					_logger.Info("Installation cancelled.")
					return nil
				}
			}

			_logger.Info("Starting installation of %d Go version(s)...", len(expandedVersions))
			_logger.Progress("Preparing downloads and verifying version availability")

			successful, failures := installVersions(mgr, expandedVersions)

			_logger.Info(strings.Repeat("─", 50))

			if len(successful) > 0 {
				_logger.Success("Successfully installed %d version(s):", len(successful))
				for _, version := range successful {
					_logger.Info("  • Go %s", version)
				}
			}

			if len(failures) > 0 {
				return withHelp(
					fmt.Errorf("failed to install %d version(s): %w", len(failures), errors.Join(failures...)),
					"Check the network and release endpoint, then retry a failed version individually with --verbose.",
				)
			}

			if len(successful) > 0 {
				_logger.Success("All installations completed successfully!")
				if len(successful) == 1 {
					_logger.Info("Activate it with: govman use %s", successful[0])
				} else {
					_logger.Info("List all versions: govman list")
					_logger.Info("Activate any version: govman use <version>")
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&includeUnstable, "unstable", false, "Include unstable versions (beta, rc) when using wildcard patterns")
	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt for batch operations")

	return cmd
}

// newUninstallCmd creates the 'uninstall' Cobra command to remove one or more installed Go versions.
// Versions are provided as positional args. Returns a *cobra.Command that uninstalls each version and reports results.
func newUninstallCmd() *cobra.Command {
	var skipConfirm bool

	cmd := &cobra.Command{
		Use:   "uninstall [version...]",
		Short: "Safely remove Go versions with cleanup",
		Long: `Completely remove one or more installed Go versions from your system.

Safety features:
  • Prevents removal of currently active versions
  • Confirms version exists before attempting removal
  • Complete cleanup of binaries and associated files
  • Automatic recalculation of disk space
  • Preserves other installed versions safely
  • Batch uninstallation with detailed progress tracking
  • Wildcard pattern support for batch uninstallation (e.g., 1.14.*)

The uninstalled versions will no longer appear in 'govman list'.

Examples:
  govman uninstall 1.24.1              # Single version
  govman uninstall 1.24.1 1.24.2       # Multiple versions
  govman rm 1.21.1 1.22.0 1.23.0       # Using alias
  govman uninstall '1.14.*'            # All 1.14.x versions (quote the pattern!)`,
		Aliases: []string{"remove", "rm"},
		Args:    usageArgs(cobra.MinimumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := _manager.New(getConfig())

			// Expand wildcard patterns for installed versions
			expandedVersions, err := expandUninstallPatterns(args, mgr)
			if err != nil {
				return err
			}

			if len(expandedVersions) == 0 {
				_logger.Warning("No installed versions matched the specified pattern(s)")
				return fmt.Errorf("no versions to uninstall")
			}

			// Show confirmation for pattern-based uninstallation
			if hasWildcardPattern(args) && !skipConfirm {
				if getConfig().Quiet {
					return withUsageHelp(cmd, fmt.Errorf("quiet batch uninstallation requires --yes"), "Pass --yes to confirm without an interactive prompt.")
				}
				_logger.Info("The following %d version(s) will be uninstalled:", len(expandedVersions))
				for _, v := range expandedVersions {
					_logger.Info("  • Go %s", v)
				}
				_logger.Info("")

				if !confirmAction("Proceed with uninstallation?") {
					_logger.Info("Uninstallation cancelled.")
					return nil
				}
			}

			_logger.Info("Starting uninstallation of %d Go version(s)...", len(expandedVersions))
			_logger.Progress("Validating versions and checking installation status")

			current, currentErr := mgr.Current()
			if currentErr != nil && !errors.Is(currentErr, _manager.ErrNoActiveVersion) && !errors.Is(currentErr, _manager.ErrUnmanagedGo) {
				return fmt.Errorf("failed to determine active Go version: %w", currentErr)
			}
			successful, totalFreedSpace, failures := uninstallVersions(mgr, expandedVersions, current)

			_logger.Info(strings.Repeat("─", 50))

			if len(successful) > 0 {
				_logger.Success("Successfully uninstalled %d version(s):", len(successful))
				for _, version := range successful {
					_logger.Info("  • Go %s", version)
				}
				_logger.Info("Total disk space freed: %s", _util.FormatBytes(totalFreedSpace))
			}

			if len(failures) > 0 {
				return withHelp(
					fmt.Errorf("failed to uninstall %d version(s): %w", len(failures), errors.Join(failures...)),
					"Switch away from active versions and verify filesystem permissions before retrying.",
				)
			}

			if len(successful) > 0 {
				_logger.Success("All uninstallations completed successfully!")
				_logger.Info("View remaining versions with: govman list")
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt for batch operations")

	return cmd
}

// hasWildcardPattern checks if any of the provided args contains a wildcard pattern.
func hasWildcardPattern(args []string) bool {
	for _, arg := range args {
		if _util.IsWildcardPattern(arg) {
			return true
		}
	}
	return false
}

func validateGlobPattern(pattern string) error {
	_, err := filepath.Match(pattern, pattern)
	return err
}

// expandInstallPatterns expands wildcard patterns in version arguments using remote available versions.
// When includeUnstable is true, both stable and prerelease versions are returned.
// Returns a deduplicated, sorted list of concrete versions to install.
func expandInstallPatterns(args []string, mgr *_manager.Manager, includeUnstable bool) ([]string, error) {
	var allVersions []string
	seenVersions := make(map[string]bool)
	var remoteVersions []string
	for _, arg := range args {
		if _util.IsWildcardPattern(arg) {
			if err := validateGlobPattern(arg); err != nil {
				return nil, fmt.Errorf("invalid version pattern %q: %w", arg, err)
			}
		}
	}
	if hasWildcardPattern(args) {
		var err error
		remoteVersions, err = mgr.ListRemote(includeUnstable)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch remote versions: %w", err)
		}
	}

	for _, arg := range args {
		if _util.IsWildcardPattern(arg) {
			_logger.Progress("Fetching available versions for pattern '%s'...", arg)

			matched, err := _util.MatchVersionPattern(arg, remoteVersions)
			if err != nil {
				return nil, fmt.Errorf("failed to match install pattern %q: %w", arg, err)
			}
			if len(matched) == 0 {
				_logger.Warning("No versions matched pattern '%s'", arg)
				continue
			}

			for _, v := range matched {
				if !seenVersions[v] {
					seenVersions[v] = true
					allVersions = append(allVersions, v)
				}
			}
		} else {
			// Regular version, add as-is
			if !seenVersions[arg] {
				seenVersions[arg] = true
				allVersions = append(allVersions, arg)
			}
		}
	}

	return allVersions, nil
}

// isPrerelease checks if a version string is a prerelease version (beta, rc, or alpha).
func isPrerelease(version string) bool {
	return strings.Contains(version, "rc") ||
		strings.Contains(version, "beta") ||
		strings.Contains(version, "alpha")
}

// expandUninstallPatterns expands wildcard patterns in version arguments using installed versions.
// Returns a deduplicated list of concrete versions to uninstall.
func expandUninstallPatterns(args []string, mgr *_manager.Manager) ([]string, error) {
	var allVersions []string
	seenVersions := make(map[string]bool)
	for _, arg := range args {
		if _util.IsWildcardPattern(arg) {
			if err := validateGlobPattern(arg); err != nil {
				return nil, fmt.Errorf("invalid version pattern %q: %w", arg, err)
			}
		}
	}

	installedVersions, err := mgr.ListInstalled()
	if err != nil {
		return nil, fmt.Errorf("failed to list installed versions: %w", err)
	}

	for _, arg := range args {
		if _util.IsWildcardPattern(arg) {
			// Filter installed versions by pattern
			matched, err := _util.MatchVersionPattern(arg, installedVersions)
			if err != nil {
				return nil, fmt.Errorf("failed to match uninstall pattern %q: %w", arg, err)
			}
			if len(matched) == 0 {
				_logger.Warning("No installed versions matched pattern '%s'", arg)
				continue
			}

			for _, v := range matched {
				if !seenVersions[v] {
					seenVersions[v] = true
					allVersions = append(allVersions, v)
				}
			}
		} else {
			// Regular version - resolve alias if needed
			version, err := resolveInstalledVersionFromList(arg, installedVersions)
			if err != nil {
				return nil, err
			}

			if !seenVersions[version] {
				seenVersions[version] = true
				allVersions = append(allVersions, version)
			}
		}
	}

	return allVersions, nil
}

// confirmAction prompts the user for confirmation and returns true if they respond with 'y' or 'yes'.
func confirmAction(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", prompt)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
