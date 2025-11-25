package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	cobra "github.com/spf13/cobra"

	_logger "github.com/justjundana/govman/internal/logger"
	_manager "github.com/justjundana/govman/internal/manager"
	_util "github.com/justjundana/govman/internal/util"
)

// min returns the smaller of two integers a and b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// newListCmd creates the 'list' Cobra command to display installed or remote Go versions.
// Flags: --remote, --stable-only, --beta, and --pattern control the output. Returns a *cobra.Command.
func newListCmd() *cobra.Command {
	var (
		remote     bool
		stableOnly bool
		beta       bool
		pattern    string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List and manage Go versions with detailed information",
		Long: `Display comprehensive information about Go versions on your system.

Features:
  • View all installed Go versions with size information
  • Browse available remote versions for installation
  • Filter versions by patterns and stability level
  • See which version is currently active
  • Get installation status for each version

Pro Tips:
  • Use --remote to explore available versions before installing
  • Combine --pattern with --remote to find specific version ranges
  • The * marker indicates your currently active version`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := _manager.New(getConfig())

			if remote {
				return listRemoteVersions(mgr, !stableOnly || beta, pattern)
			}

			return listInstalledVersions(mgr)
		},
	}

	cmd.Flags().BoolVarP(&remote, "remote", "r", false, "List available versions from Go's official releases")
	cmd.Flags().BoolVar(&stableOnly, "stable-only", false, "Show only stable, production-ready versions (remote only)")
	cmd.Flags().BoolVar(&beta, "beta", false, "Include beta/rc versions for early testing (remote only)")
	cmd.Flags().StringVar(&pattern, "pattern", "", "Filter versions using glob patterns like '1.25*' or '1.2?' (remote only)")

	return cmd
}

// listInstalledVersions lists installed Go versions with size, install date, and active/default markers.
// Parameter mgr is the Manager used to query versions and metadata. Returns an error if listing fails.
func listInstalledVersions(mgr *_manager.Manager) error {
	_logger.Verbose("Scanning installation directory for Go versions")
	versions, err := mgr.ListInstalled()
	if err != nil {
		_logger.ErrorWithHelp("Unable to scan for installed Go versions", "Verify that ~/.govman/versions exists and is accessible.", "")
		return fmt.Errorf("failed to list installed versions: %w", err)
	}

	if len(versions) == 0 {
		_logger.Info("No Go versions are currently installed")
		_logger.Info("Quick start: Run 'govman install latest' to get the newest stable version")
		_logger.Info("Or browse available versions with 'govman list --remote'")
		return nil
	}

	current, _ := mgr.Current()
	defaultVersion := mgr.DefaultVersion()

	_logger.Info("Installed Go Versions (%d total):", len(versions))
	_logger.Info(strings.Repeat("─", 60))

	totalSize := int64(0)
	for _, version := range versions {
		marker := "  "
		statusIcon := "Installed"
		if version == current {
			marker = "→ "
			statusIcon = "Active"
		}

		info, err := mgr.Info(version)
		if err != nil {
			_logger.Info("%s%s %s (unable to read installation info)", marker, statusIcon, version)
			continue
		}

		versionDisplay := version
		if version == defaultVersion && defaultVersion != "" {
			versionDisplay = version + " [default]"
		}

		size := _util.FormatBytes(info.Size)
		totalSize += info.Size
		installDate := info.InstallDate.Format("2006-01-02")
		_logger.Info("%s%s %-25s %8s   installed: %s", marker, statusIcon, versionDisplay, size, installDate)
	}

	_logger.Info(strings.Repeat("─", 60))
	_logger.Info("Total disk usage: %s across %d versions", _util.FormatBytes(totalSize), len(versions))

	if current != "" {
		_logger.Info("Currently active: Go %s", current)
	} else {
		_logger.Warning("No version is currently active")
		_logger.Info("Activate a version with: govman use <version>")
	}

	return nil
}

// listRemoteVersions fetches and displays available remote Go versions.
// Parameters: mgr (Manager), includeUnstable (include beta/rc), pattern (glob filter). Returns an error on fetch failures.
func listRemoteVersions(mgr *_manager.Manager, includeUnstable bool, pattern string) error {
	_logger.Verbose("Fetching available versions from Go's official release API")
	versions, err := mgr.ListRemote(includeUnstable)
	if err != nil {
		_logger.ErrorWithHelp("Unable to fetch remote Go versions", "Check your internet connection and verify that golang.org is accessible.", "")
		return fmt.Errorf("failed to list remote versions: %w", err)
	}

	if pattern != "" {
		originalCount := len(versions)

		if !strings.ContainsAny(pattern, "*?[]") {
			pattern = pattern + "*"
			_logger.Verbose("Pattern auto-expanded to '%s' for broader matching", pattern)
		}

		var filtered []string
		for _, version := range versions {
			if matched, _ := filepath.Match(pattern, version); matched {
				filtered = append(filtered, version)
			}
		}
		versions = filtered
		_logger.Verbose("Pattern '%s' matched %d of %d available versions", pattern, len(versions), originalCount)
	}

	if len(versions) == 0 {
		if pattern != "" {
			_logger.Info("No versions found matching pattern '%s'", pattern)
			_logger.Info("Try a broader pattern like '%s*' or remove the pattern filter", pattern[:min(len(pattern), 4)])
		} else {
			_logger.Info("No versions found")
			_logger.Info("This might be a temporary issue - try again in a moment")
		}
		return nil
	}

	stableCount := 0
	unstableCount := 0
	installedCount := 0

	for _, version := range versions {
		if strings.Contains(version, "rc") || strings.Contains(version, "beta") {
			unstableCount++
		} else {
			stableCount++
		}
		if mgr.IsInstalled(version) {
			installedCount++
		}
	}

	versionTypeDesc := "versions"
	if includeUnstable {
		versionTypeDesc = fmt.Sprintf("versions (%d stable, %d pre-release)", stableCount, unstableCount)
	} else {
		versionTypeDesc = "stable versions"
	}

	_logger.Info("Available Go %s (%d total, %d already installed):", versionTypeDesc, len(versions), installedCount)
	_logger.Info(strings.Repeat("─", 60))

	for _, version := range versions {
		installed := mgr.IsInstalled(version)
		statusIcon := "Available"
		statusText := "available"
		marker := "  "
		if installed {
			statusIcon = "Installed"
			statusText = "installed"
			marker = "✓ "
		}

		versionType := ""
		if strings.Contains(version, "rc") {
			versionType = " [release candidate]"
		} else if strings.Contains(version, "beta") {
			versionType = " [beta]"
		}

		_logger.Info("%s%s %-15s %s%s", marker, statusIcon, version, statusText, versionType)
	}

	_logger.Info(strings.Repeat("─", 60))
	if installedCount > 0 {
		_logger.Info("%d versions already installed (marked with ✓)", installedCount)
	}
	_logger.Info("Install any version with: govman install <version>")
	if !includeUnstable && unstableCount > 0 {
		_logger.Info("Add --beta flag to see %d pre-release versions", unstableCount)
	}

	return nil
}
