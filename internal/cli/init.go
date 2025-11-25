package cli

import (
	"fmt"
	"strings"

	cobra "github.com/spf13/cobra"

	_logger "github.com/justjundana/govman/internal/logger"
	_shell "github.com/justjundana/govman/internal/shell"
)

// newInitCmd creates the 'init' Cobra command to set up shell integration.
// Flags: force (overwrite existing configuration) and shellName (target shell).
// Returns a *cobra.Command whose RunE detects or uses the specified shell and initializes integration.
func newInitCmd() *cobra.Command {
	var (
		force     bool
		shellName string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize smart shell integration for seamless Go version switching",
		Long: `Set up intelligent shell integration for automatic Go version management.

Integration Features:
  • Automatic Go version switching based on .govman-goversion files
  • Smart PATH management and environment variable handling
  • Support for bash, zsh, fish, and PowerShell
  • Non-intrusive configuration with easy removal
  • Project-aware version detection
  • Seamless integration with existing shell setups

Supported Shells:
  • Bash (.bashrc, .bash_profile)
  • Zsh (.zshrc)
  • Fish (config.fish)
  • PowerShell (profile)

After initialization, govman will automatically activate the correct
Go version when you navigate to different projects.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var sh _shell.Shell

			if shellName != "" {
				sh = getShellByName(shellName)
				if sh == nil {
					_logger.ErrorWithHelp("Unsupported shell: %s", "Supported shells: bash, zsh, fish, powershell. Use --shell flag to specify.", shellName)
					return fmt.Errorf("unsupported shell: %s", shellName)
				}
				_logger.Info("Using manually specified shell: %s", sh.Name())
			} else {
				sh = _shell.Detect()
				_logger.Info("Auto-detected shell: %s", sh.Name())
			}

			cfg := getConfig()
			binPath := cfg.GetBinPath()

			_logger.Info("Initializing shell integration for %s...", sh.Name())
			_logger.Progress("Configuring PATH and environment variables")

			_logger.Verbose("Setting up shell integration with binary path: %s", binPath)
			if err := _shell.InitializeShell(sh, binPath, force); err != nil {
				_logger.ErrorWithHelp("Failed to configure shell integration", "Ensure you have write permissions to your shell configuration file and try again.", "")
				return err
			}

			_logger.Success("Shell integration configured successfully!")
			_logger.Info("Configuration file: %s", sh.ConfigFile())
			_logger.Info(strings.Repeat("─", 50))
			_logger.Info("Next Steps:")
			_logger.Info("  1. Restart your terminal or run: source %s", sh.ConfigFile())
			_logger.Info("  2. Navigate to a project directory")
			_logger.Info("  3. Create a .govman-goversion file with your desired Go version")
			_logger.Info("  4. govman will automatically switch versions for you!")
			_logger.Info(strings.Repeat("─", 50))
			_logger.Info("Happy Go development!")

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force re-initialization (overwrite existing configuration)")
	cmd.Flags().StringVar(&shellName, "shell", "", "Target specific shell (bash, zsh, fish, powershell)")

	return cmd
}

// getShellByName maps a shell name to its Shell implementation.
// Supported values: bash, zsh, fish, powershell/pwsh. Returns nil if unsupported.
func getShellByName(name string) _shell.Shell {
	switch name {
	case "bash":
		return &_shell.BashShell{}
	case "zsh":
		return &_shell.ZshShell{}
	case "fish":
		return &_shell.FishShell{}
	case "powershell", "pwsh":
		return &_shell.PowerShell{}
	default:
		return nil
	}
}
