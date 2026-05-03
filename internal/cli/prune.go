package cli

import (
	"fmt"
	"strings"

	cobra "github.com/spf13/cobra"

	_logger "github.com/justjundana/govman/internal/logger"
	_manager "github.com/justjundana/govman/internal/manager"
	_util "github.com/justjundana/govman/internal/util"
)

// getProtectedVersions determines which installed versions should be kept during pruning.
func getProtectedVersions(mgr *_manager.Manager, installed []string) map[string]string {
	protected := make(map[string]string) // version -> reason

	if current, err := mgr.Current(); err == nil && current != "" {
		protected[current] = "currently active"
	}

	if defaultVersion := mgr.DefaultVersion(); defaultVersion != "" {
		if _, exists := protected[defaultVersion]; !exists {
			protected[defaultVersion] = "system default"
		}
	}

	cfg := getConfig()
	if cfg != nil && cfg.AutoSwitch.ProjectFile != "" {
		localVersion := mgr.GetLocalVersionRaw()
		if localVersion != "" {
			if matchedVersion, err := _util.FindBestMatchingVersion(localVersion, installed); err == nil {
				if _, exists := protected[matchedVersion]; !exists {
					protected[matchedVersion] = "project-local (.govman-goversion)"
				}
			}
		}
	}

	return protected
}

// executePrune removes the specified versions and returns results.
func executePrune(mgr *_manager.Manager, toRemove []string) (successful []string, totalFreedSpace int64, errors []string) {
	for i, version := range toRemove {
		_logger.Info("[%d/%d] Removing Go %s...", i+1, len(toRemove), version)

		info, err := mgr.Info(version)
		if err != nil {
			_logger.Warning("Failed to get info for Go %s: %v", version, err)
			errors = append(errors, fmt.Sprintf("Go %s: %v", version, err))
			continue
		}

		if err := mgr.Uninstall(version); err != nil {
			_logger.Warning("Failed to remove Go %s: %v", version, err)
			errors = append(errors, fmt.Sprintf("Go %s: %v", version, err))
			continue
		}

		successful = append(successful, version)
		totalFreedSpace += info.Size
		_logger.Success("Removed Go %s", version)
	}
	return
}

// newPruneCmd creates the 'prune' Cobra command to remove all unused Go versions.
// It keeps the currently active version, the system default, and the local project version.
// Returns a *cobra.Command that prunes unused versions and reports freed disk space.

// reportPruneResults prints the summary of the prune operation and returns an error if any removals failed.
func reportPruneResults(successful []string, totalFreedSpace int64, errors []string, protected map[string]string) error {
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
	for version, reason := range protected {
		_logger.Info("  • Go %s (%s)", version, reason)
	}

	return nil
}

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

			installed, err := mgr.ListInstalled()
			if err != nil {
				_logger.ErrorWithHelp("Unable to list installed versions", "Verify that ~/.govman/versions exists and you have sufficient permissions.")
				return fmt.Errorf("failed to list installed versions: %w", err)
			}

			if len(installed) == 0 {
				_logger.Info("No Go versions are installed")
				return nil
			}

			protected := getProtectedVersions(mgr, installed)

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

			if !skipConfirm {
				if !confirmAction("Proceed with pruning?") {
					_logger.Info("Pruning cancelled.")
					return nil
				}
			}

			_logger.Info("Pruning %d unused Go version(s)...", len(toRemove))
			_logger.Progress("Removing unused installations")

			successful, totalFreedSpace, errors := executePrune(mgr, toRemove)

			return reportPruneResults(successful, totalFreedSpace, errors, protected)
		},
	}

	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
