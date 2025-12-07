package cli

import (
	"fmt"
	"strings"

	cobra "github.com/spf13/cobra"

	_logger "github.com/justjundana/govman/internal/logger"
	_manager "github.com/justjundana/govman/internal/manager"
	_util "github.com/justjundana/govman/internal/util"
)

// newInstallCmd creates the 'install' Cobra command to download and install one or more Go versions.
// Versions are provided as positional args (e.g., latest, 1.25.1). Returns a *cobra.Command that installs each version and reports results.
func newInstallCmd() *cobra.Command {
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

Examples:
  govman install latest              # Latest stable release
  govman install 1.25.1              # Specific version
  govman install 1.25.1 1.20.12      # Multiple versions
  govman install 1.22rc1             # Pre-release version`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := _manager.New(getConfig())

			_logger.Info("Starting installation of %d Go version(s)...", len(args))
			_logger.Progress("Preparing downloads and verifying version availability")

			var errors []string
			var successful []string
			for i, version := range args {
				_logger.Info("[%d/%d] Installing Go %s...", i+1, len(args), version)
				if err := mgr.Install(version); err != nil {
					errors = append(errors, fmt.Sprintf("Go %s: %v", version, err))
					_logger.Warning("Failed to install Go %s: %v", version, err)
					continue
				}

				successful = append(successful, version)
				_logger.Success("Successfully installed Go %s", version)
			}

			_logger.Info(strings.Repeat("─", 50))

			if len(successful) > 0 {
				_logger.Success("Successfully installed %d version(s):", len(successful))
				for _, version := range successful {
					_logger.Info("  • Go %s", version)
				}
			}

			if len(errors) > 0 {
				_logger.ErrorWithHelp("Failed to install %d version(s):", "Review the errors below and try installing problematic versions individually for more details.", len(errors))
				for _, err := range errors {
					_logger.Info("  %s", err)
				}
				_logger.Info("Common solutions:")
				_logger.Info("  • Check your internet connection")
				_logger.Info("  • Verify version exists with 'govman list --remote'")
				_logger.Info("  • Try again with verbose mode: govman install <version> --verbose")
				return fmt.Errorf("failed to install %d version(s)", len(errors))
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

	return cmd
}

// newUninstallCmd creates the 'uninstall' Cobra command to remove one or more installed Go versions.
// Versions are provided as positional args. Returns a *cobra.Command that uninstalls each version and reports results.
func newUninstallCmd() *cobra.Command {
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

The uninstalled versions will no longer appear in 'govman list'.

Examples:
  govman uninstall 1.24.1              # Single version
  govman uninstall 1.24.1 1.24.2       # Multiple versions
  govman rm 1.21.1 1.22.0 1.23.0       # Using alias`,
		Aliases: []string{"remove", "rm"},
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := _manager.New(getConfig())

			_logger.Info("Starting uninstallation of %d Go version(s)...", len(args))
			_logger.Progress("Validating versions and checking installation status")

			current, _ := mgr.Current()
			var errors []string
			var successful []string
			var totalFreedSpace int64

			for i, version := range args {
				_logger.Info("[%d/%d] Uninstalling Go %s...", i+1, len(args), version)

				// Resolve alias to concrete version if needed
				originalVersion := version
				if version == "latest" || version == "stable" {
					installedVersions, err := mgr.ListInstalled()
					if err != nil {
						_logger.Verbose("Failed to list installed versions: %v", err)
					}
					if len(installedVersions) > 0 {
						version = installedVersions[0] // installed versions are sorted in descending order
						_logger.Verbose("Resolved alias %s to installed version %s", originalVersion, version)
					}
				} else if strings.Count(version, ".") == 1 {
					// Partial version: resolve to best match
					installedVersions, err := mgr.ListInstalled()
					if err != nil {
						_logger.Verbose("Failed to list installed versions: %v", err)
					}
					if len(installedVersions) > 0 {
						if matchedVersion, err := _util.FindBestMatchingVersion(version, installedVersions); err == nil {
							_logger.Verbose("Resolved %s to installed version %s", version, matchedVersion)
							version = matchedVersion
						}
					}
				}

				// Check if version is currently active
				if current == version {
					_logger.Warning("Cannot uninstall currently active Go version %s", version)
					errors = append(errors, fmt.Sprintf("Go %s: cannot uninstall active version", version))
					continue
				}

				// Get version info before uninstalling to track disk space
				info, err := mgr.Info(version)
				if err != nil {
					_logger.Warning("Go version %s is not installed or information is unavailable", version)
					errors = append(errors, fmt.Sprintf("Go %s: %v", version, err))
					continue
				}

				// Perform uninstallation
				_logger.Progress("Removing installation directory and associated files")
				err = mgr.Uninstall(version)
				if err != nil {
					_logger.Warning("Failed to uninstall Go %s: %v", version, err)
					errors = append(errors, fmt.Sprintf("Go %s: %v", version, err))
					continue
				}

				successful = append(successful, version)
				totalFreedSpace += info.Size
				_logger.Success("Successfully uninstalled Go %s", version)
			}

			_logger.Info(strings.Repeat("─", 50))

			if len(successful) > 0 {
				_logger.Success("Successfully uninstalled %d version(s):", len(successful))
				for _, version := range successful {
					_logger.Info("  • Go %s", version)
				}
				_logger.Info("Total disk space freed: %s", _util.FormatBytes(totalFreedSpace))
			}

			if len(errors) > 0 {
				_logger.ErrorWithHelp("Failed to uninstall %d version(s):", "Review the errors below and address any issues.", len(errors))
				for _, err := range errors {
					_logger.Info("  %s", err)
				}
				_logger.Info("Common solutions:")
				_logger.Info("  • Switch to a different version if trying to uninstall active version")
				_logger.Info("  • Verify version is installed with 'govman list'")
				_logger.Info("  • Ensure no processes are using the Go installation")
				return fmt.Errorf("failed to uninstall %d version(s)", len(errors))
			}

			if len(successful) > 0 {
				_logger.Success("All uninstallations completed successfully!")
				_logger.Info("View remaining versions with: govman list")
			}

			return nil
		},
	}

	return cmd
}
