package cli

import (
	"fmt"
	"strings"

	cobra "github.com/spf13/cobra"

	_logger "github.com/justjundana/govman/internal/logger"
	_manager "github.com/justjundana/govman/internal/manager"
	_util "github.com/justjundana/govman/internal/util"
)

// getActivationMode returns a human-friendly label for the activation mode.
// Parameters: setDefault (system-wide default), setLocal (project-local).
// Returns "project-local", "system-default", or "session-only" based on flags.
func getActivationMode(setDefault, setLocal bool) string {
	if setLocal {
		return "project-local"
	}
	if setDefault {
		return "system-default"
	}
	return "session-only"
}

// newUseCmd creates the 'use' Cobra command to activate a Go version.
// Flags: setDefault (system default) and setLocal (project-local) control activation scope.
// Returns a *cobra.Command that validates installation, calls Manager.Use, and reports status.
func newUseCmd() *cobra.Command {
	var (
		setDefault bool
		setLocal   bool
	)

	cmd := &cobra.Command{
		Use:   "use <version>",
		Short: "Switch between Go versions with flexible activation options",
		Long: `Activate a specific Go version for your development environment.

Activation Modes:
  • Session-only: Temporary activation for current terminal session
  • System default: Permanent activation across all new sessions
  • Project-local: Version tied to specific project directory

Smart Features:
  • Automatic verification of version installation
  • Shell integration with PATH management
  • Project-specific .govman-goversion file support
  • Seamless switching between versions

Examples:
  govman use 1.25.1                 # Session-only activation
  govman use 1.25.1 --default       # Set as system default
  govman use 1.25.1 --local         # Project-specific version`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := args[0]
			mgr := _manager.New(getConfig())

			if version != "default" {
				// Check if version is an alias like "latest", "stable", etc.
				// Aliases have no dots in them (except for partial versions like "1.24")
				isAlias := version == "latest" || version == "stable"
				isPartialVersion := strings.Count(version, ".") == 1

				if isAlias {
					// Alias (e.g., "latest"): resolve to installed version first
					installedVersions, err := mgr.ListInstalled()
					if err != nil {
						_logger.Verbose("Failed to list installed versions: %v", err)
					}
					if len(installedVersions) > 0 {
						// For "latest", use the newest installed version
						if version == "latest" || version == "stable" {
							version = installedVersions[0] // installed versions are sorted in descending order
							_logger.Verbose("Resolved alias to installed version %s", version)
						}
					} else {
						// No versions installed, resolve from remote
						resolved, err := mgr.ResolveVersion(version)
						if err != nil {
							return fmt.Errorf("failed to resolve version %s: %w", version, err)
						}
						version = resolved
					}
				} else if isPartialVersion {
					// Partial version (e.g., "1.24"): use flexible matching
					installedVersions, err := mgr.ListInstalled()
					if err != nil {
						_logger.Verbose("Failed to list installed versions: %v", err)
					}
					if len(installedVersions) > 0 {
						if matchedVersion, err := _util.FindBestMatchingVersion(version, installedVersions); err == nil {
							_logger.Verbose("Resolved %s to installed version %s", version, matchedVersion)
							version = matchedVersion
						} else {
							// No installed version matches, resolve from remote
							resolved, err := mgr.ResolveVersion(version)
							if err != nil {
								return fmt.Errorf("failed to resolve version %s: %w", version, err)
							}
							version = resolved
						}
					} else {
						// No versions installed, resolve from remote
						resolved, err := mgr.ResolveVersion(version)
						if err != nil {
							return fmt.Errorf("failed to resolve version %s: %w", version, err)
						}
						version = resolved
					}
				} else {
					// Full version (e.g., "1.24.1"): check exact match first
					if !mgr.IsInstalled(version) {
						// Exact version not found, try flexible matching as fallback
						installedVersions, err := mgr.ListInstalled()
						if err != nil {
							_logger.Verbose("Failed to list installed versions: %v", err)
						}
						if len(installedVersions) > 0 {
							if matchedVersion, err := _util.FindBestMatchingVersion(version, installedVersions); err == nil {
								_logger.Verbose("Exact version %s not found, using %s (closest match)", version, matchedVersion)
								version = matchedVersion
							}
						}
					}
				}

				if !mgr.IsInstalled(version) {
					helpMsg := fmt.Sprintf("Install it first with 'govman install %s', or check available versions with 'govman list'.", version)
					_logger.ErrorWithHelp("Go version %s is not installed", helpMsg, version)
					return fmt.Errorf("version %s not installed", version)
				}
			}

			_logger.Verbose("Activating Go %s with mode: %s", version, getActivationMode(setDefault, setLocal))

			err := mgr.Use(version, setDefault, setLocal)
			if err != nil {
				_logger.ErrorWithHelp("Failed to activate Go %s", "Ensure the version is properly installed and you have sufficient permissions.", version)
				return err
			}

			if setLocal {
				_logger.Success("Set Go %s as local version for this project", version)
				_logger.Info("Created/updated .govman-goversion file in current directory")
				_logger.Info("This version will be used automatically when working in this project")
			} else if setDefault {
				_logger.Success("Set Go %s as system default version", version)
				_logger.Info("All new terminal sessions will use this version")
				_logger.Info("Current session updated - run 'go version' to verify")
			} else {
				_logger.Success("Now using Go %s for this session", version)
				_logger.Info("This is temporary - use --default to make it permanent")
				_logger.Info("Run 'go version' to confirm the switch")
			}

			info, err := mgr.Info(version)
			if err == nil {
				_logger.Info("Version details: %s/%s, installed %s", info.OS, info.Arch, info.InstallDate.Format("2006-01-02"))
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&setDefault, "default", "d", false, "Set as system-wide default version (persistent)")
	cmd.Flags().BoolVarP(&setLocal, "local", "l", false, "Set as project-local version (creates .govman-goversion file)")

	return cmd
}
