package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	cobra "github.com/spf13/cobra"

	_logger "github.com/justjundana/govman/internal/logger"
	_manager "github.com/justjundana/govman/internal/manager"
	_util "github.com/justjundana/govman/internal/util"
)

// getProtectedVersions determines which installed versions should be kept during pruning.
func getProtectedVersions(mgr *_manager.Manager, installed []string) (map[string]string, error) {
	protected := make(map[string]string) // version -> reason

	if current, err := mgr.Current(); err == nil && current != "" {
		protected[current] = "currently active"
	} else if err != nil && !errors.Is(err, _manager.ErrNoActiveVersion) && !errors.Is(err, _manager.ErrUnmanagedGo) {
		return nil, fmt.Errorf("failed to determine active Go version: %w", err)
	}

	if defaultVersion := mgr.DefaultVersion(); defaultVersion != "" {
		if _, exists := protected[defaultVersion]; !exists {
			protected[defaultVersion] = "system default"
		}
	}

	cfg := getConfig()
	if cfg != nil && cfg.AutoSwitch.ProjectFile != "" {
		localVersion, err := mgr.ReadLocalVersionRaw()
		if err != nil {
			return nil, fmt.Errorf("failed to read project-local version: %w", err)
		}
		if localVersion != "" {
			if matchedVersion, err := _util.FindBestMatchingVersion(localVersion, installed); err == nil {
				if _, exists := protected[matchedVersion]; !exists {
					protected[matchedVersion] = "project-local (.govman-goversion)"
				}
			}
		}
	}

	return protected, nil
}

// executePrune removes the specified versions and returns results.
func executePrune(mgr *_manager.Manager, toRemove []string) (successful []string, totalFreedSpace int64, failures []error) {
	for i, version := range toRemove {
		_logger.Info("[%d/%d] Removing Go %s...", i+1, len(toRemove), version)

		info, err := mgr.Info(version)
		if err != nil {
			_logger.Warning("Failed to get info for Go %s: %v", version, err)
			failures = append(failures, fmt.Errorf("Go %s: %w", version, err))
			continue
		}

		if err := mgr.Uninstall(version); err != nil {
			_logger.Warning("Failed to remove Go %s: %v", version, err)
			failures = append(failures, fmt.Errorf("Go %s: %w", version, err))
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
func reportPruneResults(successful []string, totalFreedSpace int64, failures []error, protected map[string]string) error {
	_logger.Info(strings.Repeat("─", 50))

	if len(successful) > 0 {
		_logger.Success("Successfully pruned %d version(s):", len(successful))
		for _, version := range successful {
			_logger.Info("  • Go %s", version)
		}
		_logger.Info("Total disk space freed: %s", _util.FormatBytes(totalFreedSpace))
	}

	if len(failures) > 0 {
		return withHelp(fmt.Errorf("failed to prune %d version(s): %w", len(failures), errors.Join(failures...)), "Review active versions and filesystem permissions, then retry.")
	}

	_logger.Success("Pruning completed successfully!")
	_logger.Info("Remaining installed versions:")
	for _, version := range sortedVersionKeys(protected) {
		reason := protected[version]
		_logger.Info("  • Go %s (%s)", version, reason)
	}

	return nil
}

func sortedVersionKeys(versions map[string]string) []string {
	keys := make([]string, 0, len(versions))
	for version := range versions {
		keys = append(keys, version)
	}
	sort.Strings(keys)
	return keys
}

func newPruneCmd() *cobra.Command {
	var skipConfirm bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove all unused Go versions to reclaim disk space",
		Args:  usageArgs(cobra.NoArgs),
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
				return withHelp(fmt.Errorf("failed to list installed versions: %w", err), "Verify that the configured install directory is accessible.")
			}

			if len(installed) == 0 {
				_logger.Info("No Go versions are installed")
				return nil
			}

			protected, err := getProtectedVersions(mgr, installed)
			if err != nil {
				return err
			}

			var toRemove []string
			for _, version := range installed {
				if _, isProtected := protected[version]; !isProtected {
					toRemove = append(toRemove, version)
				}
			}

			if len(toRemove) == 0 {
				_logger.Success("No unused versions to prune")
				_logger.Info("All %d installed version(s) are currently in use:", len(installed))
				for _, version := range sortedVersionKeys(protected) {
					reason := protected[version]
					_logger.Info("  • Go %s (%s)", version, reason)
				}
				return nil
			}

			_logger.Info("Protected versions (will be kept):")
			for _, version := range sortedVersionKeys(protected) {
				reason := protected[version]
				_logger.Info("  ✓ Go %s (%s)", version, reason)
			}
			_logger.Info("")
			_logger.Info("The following %d version(s) will be removed:", len(toRemove))
			for _, version := range toRemove {
				_logger.Info("  ✗ Go %s", version)
			}
			_logger.Info("")

			if !skipConfirm {
				if getConfig().Quiet {
					return withUsageHelp(cmd, fmt.Errorf("quiet pruning requires --yes"), "Pass --yes to confirm without an interactive prompt.")
				}
				if !confirmAction("Proceed with pruning?") {
					_logger.Info("Pruning cancelled.")
					return nil
				}
			}

			_logger.Info("Pruning %d unused Go version(s)...", len(toRemove))
			_logger.Progress("Removing unused installations")

			successful, totalFreedSpace, failures := executePrune(mgr, toRemove)

			return reportPruneResults(successful, totalFreedSpace, failures, protected)
		},
	}

	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
