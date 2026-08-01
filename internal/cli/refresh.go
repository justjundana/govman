package cli

import (
	"fmt"
	"os"
	"strings"

	cobra "github.com/spf13/cobra"

	_logger "github.com/justjundana/govman/internal/logger"
	_manager "github.com/justjundana/govman/internal/manager"
)

// newRefreshCmd creates the 'refresh' Cobra command to re-evaluate the current directory for a .govman-goversion file.
// Returns a *cobra.Command whose RunE switches to the local version if present, otherwise to the default; errors if the required version isn't installed.
func newRefreshCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh Go version based on current directory context",
		Args:  usageArgs(cobra.NoArgs),
		Long: `Manually trigger version switching based on the current directory.

Purpose:
  • Re-evaluate the current directory for .govman-goversion files
  • Switch to the appropriate version (local or default)
  • Useful after adding/removing .govman-goversion files

Examples:
  govman refresh                    # Re-evaluate current directory

Behavior:
  • If .govman-goversion exists: switch to that version
  • If no .govman-goversion: switch to default version
  • Equivalent to the auto-switch that happens on 'cd'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := _manager.New(getConfig())

			cfg := getConfig()
			filename := cfg.AutoSwitch.ProjectFile
			data, readErr := os.ReadFile(filename)
			if readErr != nil && !os.IsNotExist(readErr) {
				return fmt.Errorf("failed to read local version file %s: %w", filename, readErr)
			}
			if readErr == nil {
				version := strings.TrimSpace(string(data))

				// Validate version format
				if version == "" {
					_logger.Warning("Empty version file: %s", filename)
					_logger.Info("Switching to default Go version")
					return mgr.Use("default", false, false)
				}

				if !_manager.VersionFormatRegex.MatchString(version) {
					return withHelp(fmt.Errorf("invalid version format in %s: %s", filename, version), "Version should be like '1.25', '1.25.4', or 'latest'.")
				}

				_logger.Info("Found local version file: %s", filename)

				// Resolve aliases (latest/stable) to concrete version
				if version == "latest" || version == "stable" {
					resolvedVersion, err := mgr.ResolveVersion(version)
					if err != nil {
						return withHelp(fmt.Errorf("failed to resolve version %s: %w", version, err), "Check your internet connection or specify an exact installed version.")
					}
					_logger.Verbose("Resolved alias %s to %s", version, resolvedVersion)
					version = resolvedVersion
				} else if strings.Count(version, ".") == 1 {
					resolvedVersion, err := resolveInstalledVersion(mgr, version)
					if err != nil {
						return err
					}
					version = resolvedVersion
				}

				_logger.Info("Switching to Go %s", version)

				if !mgr.IsInstalled(version) {
					helpMsg := fmt.Sprintf("Install it first with 'govman install %s'", version)
					return withHelp(fmt.Errorf("go version %s is not installed", version), helpMsg)
				}

				return mgr.Use(version, false, false)
			}

			_logger.Info("No local version file found")
			_logger.Info("Switching to default Go version")

			return mgr.Use("default", false, false)
		},
	}

	return cmd
}
