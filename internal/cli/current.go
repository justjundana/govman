package cli

import (
	"fmt"
	"strings"

	cobra "github.com/spf13/cobra"

	_logger "github.com/justjundana/govman/internal/logger"
	_manager "github.com/justjundana/govman/internal/manager"
	_util "github.com/justjundana/govman/internal/util"
)

// newCurrentCmd creates the 'current' Cobra command to display details of the active Go version.
// It returns a *cobra.Command whose RunE queries the Manager for the current version and prints environment info.
func newCurrentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "current",
		Short: "Display comprehensive current Go version information",
		Args:  usageArgs(cobra.NoArgs),
		Long: `Show detailed information about the currently active Go version.

Information displayed:
  • Version number and release status
  • Installation path and size
  • Platform architecture details
  • Installation date and source
  • Activation method (system, project, or session)

Use this to verify your environment and troubleshoot version issues.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := _manager.New(getConfig())

			_logger.Verbose("Detecting currently active Go version")
			current, err := mgr.Current()
			if err != nil {
				return withHelp(fmt.Errorf("unable to determine the active Go version: %w", err), "Install a Go version with 'govman install latest', then activate it with 'govman use <version>'.")
			}

			info, err := mgr.Info(current)
			if err != nil {
				return fmt.Errorf("active Go %s has unreadable installation details: %w", current, err)
			}

			_logger.Info("Current Go Environment:")
			_logger.Info(strings.Repeat("─", 50))
			_logger.Info("Version:         Go %s", info.Version)
			_logger.Info("Install Path:    %s", info.Path)
			_logger.Info("Platform:        %s/%s", info.OS, info.Arch)
			_logger.Info("Installed:       %s", info.InstallDate.Format("2006-01-02 15:04:05 MST"))
			_logger.Info("Disk Usage:      %s", _util.FormatBytes(info.Size))

			activationMethod := mgr.CurrentActivationMethod()

			_logger.Info("Activation:      %s", activationMethod)
			_logger.Info(strings.Repeat("─", 50))
			_logger.Info("Run 'go version' to verify your Go installation")

			return nil
		},
	}

	return cmd
}
