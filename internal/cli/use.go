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

// resolveAlias resolves an alias like "latest" or "stable" to a concrete version.
func resolveAlias(mgr *_manager.Manager, version string) (string, error) {
	installedVersions, err := mgr.ListInstalled()
	if err != nil {
		return "", fmt.Errorf("failed to list installed versions: %w", err)
	}
	if len(installedVersions) > 0 {
		resolved := installedVersions[0] // installed versions are sorted in descending order
		_logger.Verbose("Resolved alias to installed version %s", resolved)
		return resolved, nil
	}
	return mgr.ResolveVersion(version)
}

// resolvePartialVersion resolves a partial version (e.g., "1.24") to a concrete installed version.
func resolvePartialVersion(mgr *_manager.Manager, version string) (string, error) {
	installedVersions, err := mgr.ListInstalled()
	if err != nil {
		return "", fmt.Errorf("failed to list installed versions: %w", err)
	}
	if len(installedVersions) > 0 {
		if matchedVersion, err := _util.FindBestMatchingVersion(version, installedVersions); err == nil {
			_logger.Verbose("Resolved %s to installed version %s", version, matchedVersion)
			return matchedVersion, nil
		}
	}
	return mgr.ResolveVersion(version)
}

// resolveFullVersion preserves exact full-version requests; patch versions are never substituted.
func resolveFullVersion(version string) string {
	return version
}

// resolveVersionForUse resolves a version argument to a concrete installed version.
func resolveVersionForUse(mgr *_manager.Manager, version string) (string, error) {
	isAlias := version == "latest" || version == "stable"
	isPartialVersion := strings.Count(version, ".") == 1

	var err error
	if isAlias {
		version, err = resolveAlias(mgr, version)
	} else if isPartialVersion {
		version, err = resolvePartialVersion(mgr, version)
	} else {
		version = resolveFullVersion(version)
	}
	if err != nil {
		return "", fmt.Errorf("failed to resolve version %s: %w", version, err)
	}
	return version, nil
}

// resolveInstalledVersion resolves aliases and partial versions only against
// installed versions. Full versions remain exact.
func resolveInstalledVersion(mgr *_manager.Manager, requested string) (string, error) {
	installed, err := mgr.ListInstalled()
	if err != nil {
		return "", fmt.Errorf("failed to list installed versions: %w", err)
	}
	return resolveInstalledVersionFromList(requested, installed)
}

func resolveInstalledVersionFromList(requested string, installed []string) (string, error) {
	if requested == "latest" || requested == "stable" {
		if len(installed) == 0 {
			return "", fmt.Errorf("cannot resolve %s: no Go versions are installed", requested)
		}
		_logger.Verbose("Resolved %s to installed version %s", requested, installed[0])
		return installed[0], nil
	}
	if strings.Count(requested, ".") == 1 {
		matched, err := _util.FindBestMatchingVersion(requested, installed)
		if err != nil {
			return "", fmt.Errorf("no installed version matches %s: %w", requested, err)
		}
		_logger.Verbose("Resolved %s to installed version %s", requested, matched)
		return matched, nil
	}
	return requested, nil
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
		Args: usageArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := args[0]
			mgr := _manager.New(getConfig())

			if version != "default" {
				resolved, err := resolveVersionForUse(mgr, version)
				if err != nil {
					return err
				}
				version = resolved

				if !mgr.IsInstalled(version) {
					helpMsg := fmt.Sprintf("Install it first with 'govman install %s', or check available versions with 'govman list'.", version)
					return withHelp(fmt.Errorf("go version %s is not installed", version), helpMsg)
				}
			}

			_logger.Verbose("Activating Go %s with mode: %s", version, getActivationMode(setDefault, setLocal))

			err := mgr.Use(version, setDefault, setLocal)
			if err != nil {
				return withHelp(fmt.Errorf("failed to activate Go %s: %w", version, err), "Ensure the version is properly installed and you have sufficient permissions.")
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
	cmd.MarkFlagsMutuallyExclusive("default", "local")

	return cmd
}
