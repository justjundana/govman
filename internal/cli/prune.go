package cli

import (
	"fmt"
	"strings"

	cobra "github.com/spf13/cobra"

	_logger "github.com/justjundana/govman/internal/logger"
	_manager "github.com/justjundana/govman/internal/manager"
	_util "github.com/justjundana/govman/internal/util"
)

// newPruneCmd creates the 'prune' Cobra command to remove all unused Go versions.
// It keeps the currently active version, the system default, and the local project version.
// Returns a *cobra.Command that prunes unused versions and reports freed disk space.
func newPruneCmd() *cobra.Command {
	var skipConfirm bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove all unused Go versions to reclaim disk space",
		Long: `Uninstall all Go versions except those currently in use.

Protected versions (will NOT be removed):
  • Currently active version (session or global)
  • System default version (from config)
  • Project-local version (from .govman-goversion)

This is a convenient way to reclaim disk space by removing
versions you no longer need, without manually identifying them.

Examples:
  govman prune              # Interactive confirmation
  govman prune --yes        # Skip confirmation prompt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := _manager.New(getConfig())

			// Get all installed versions
			installed, err := mgr.ListInstalled()
			if err != nil {
				_logger.ErrorWithHelp("Unable to list installed versions", "Verify that ~/.govman/versions exists and you have sufficient permissions.", "")
				return fmt.Errorf("failed to list installed versions: %w", err)
			}

			if len(installed) == 0 {
				_logger.Info("No Go versions are installed")
				return nil
			}

			// Determine which versions are protected
			protected := make(map[string]string) // version -> reason

			// Currently active version
			if current, err := mgr.Current(); err == nil && current != "" {
				protected[current] = "currently active"
			}

			// System default version
			if defaultVersion := mgr.DefaultVersion(); defaultVersion != "" {
				if _, exists := protected[defaultVersion]; !exists {
					protected[defaultVersion] = "system default"
				}
			}

			// Local project version (from .govman-goversion)
			cfg := getConfig()
			if cfg != nil && cfg.AutoSwitch.ProjectFile != "" {
				localVersion := mgr.GetLocalVersionRaw()
				if localVersion != "" {
					// Find the best matching installed version for partial versions
					for _, v := range installed {
						if v == localVersion || strings.HasPrefix(v, localVersion) {
							if _, exists := protected[v]; !exists {
								protected[v] = "project-local (.govman-goversion)"
							}
						}
					}
				}
			}

			// Determine which versions to remove
			var toRemove []string
			for _, version := range installed {
				if _, isProtected := protected[version]; !isProtected {
					toRemove = append(toRemove, version)
				}
			}

			if len(toRemove) == 0 {
				_logger.Success("No unused versions to prune")
				_logger.Info("All %d installed version(s) are currently in use:", len(installed))
				for version, reason := range protected {
					_logger.Info("  • Go %s (%s)", version, reason)
				}
				return nil
			}

			// Show what will be removed
			_logger.Info("Protected versions (will be kept):")
			for version, reason := range protected {
				_logger.Info("  ✓ Go %s (%s)", version, reason)
			}
			_logger.Info("")
			_logger.Info("The following %d version(s) will be removed:", len(toRemove))
			for _, version := range toRemove {
				_logger.Info("  ✗ Go %s", version)
			}
			_logger.Info("")

			// Ask for confirmation
			if !skipConfirm {
				if !confirmAction("Proceed with pruning?") {
					_logger.Info("Pruning cancelled.")
					return nil
				}
			}

			// Perform uninstallation
			_logger.Info("Pruning %d unused Go version(s)...", len(toRemove))
			_logger.Progress("Removing unused installations")

			var errors []string
			var successful []string
			var totalFreedSpace int64

			for i, version := range toRemove {
				_logger.Info("[%d/%d] Removing Go %s...", i+1, len(toRemove), version)

				// Get version info before uninstalling to track disk space
				info, err := mgr.Info(version)
				if err != nil {
					_logger.Warning("Failed to get info for Go %s: %v", version, err)
					errors = append(errors, fmt.Sprintf("Go %s: %v", version, err))
					continue
				}

				// Perform uninstallation
				if err := mgr.Uninstall(version); err != nil {
					_logger.Warning("Failed to remove Go %s: %v", version, err)
					errors = append(errors, fmt.Sprintf("Go %s: %v", version, err))
					continue
				}

				successful = append(successful, version)
				totalFreedSpace += info.Size
				_logger.Success("Removed Go %s", version)
			}

			_logger.Info(strings.Repeat("─", 50))

			if len(successful) > 0 {
				_logger.Success("Successfully pruned %d version(s):", len(successful))
				for _, version := range successful {
					_logger.Info("  • Go %s", version)
				}
				_logger.Info("Total disk space freed: %s", _util.FormatBytes(totalFreedSpace))
			}

			if len(errors) > 0 {
				_logger.ErrorWithHelp("Failed to remove %d version(s):", "Review the errors below and address any issues.", len(errors))
				for _, err := range errors {
					_logger.Info("  %s", err)
				}
				return fmt.Errorf("failed to prune %d version(s)", len(errors))
			}

			_logger.Success("Pruning completed successfully!")
			_logger.Info("Remaining installed versions:")
			remaining, _ := mgr.ListInstalled()
			for _, version := range remaining {
				reason := protected[version]
				_logger.Info("  • Go %s (%s)", version, reason)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
