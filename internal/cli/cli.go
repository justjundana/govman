package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	cobra "github.com/spf13/cobra"

	_config "github.com/justjundana/govman/internal/config"
	_logger "github.com/justjundana/govman/internal/logger"
	_version "github.com/justjundana/govman/internal/version"
)

var (
	cfgFile     string
	quietFlag   bool
	verboseFlag bool
	cfg         *_config.Config
)

var rootCmd = &cobra.Command{
	Use:           "govman",
	Short:         "Go Version Manager - Install and manage multiple Go versions",
	Long:          createLongDescription(),
	Version:       _version.BuildVersion(),
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cleanupOldBackups()
		return initConfig(cmd.Flags().Changed("quiet"), cmd.Flags().Changed("verbose"))
	},
}

// createLongDescription returns a formatted long description string for the root CLI command.
// It assembles key features into a multi-line string and returns it.
func createLongDescription() string {
	features := []string{
		"⚡ Lightning-fast installation and switching between Go versions",
		"🎯 Zero configuration - works out of the box, no setup required",
		"📁 Project-specific versions with .govman-goversion file support",
		"🚫 No admin/sudo required - fully userspace installation",
		"💾 Verified download caching to avoid duplicate transfers",
		"📦 Resumable downloads with integrity verification",
		"🌍 Cross-platform support (Windows, macOS, Linux, ARM)",
		"🧹 Built-in cleanup tools to manage disk space efficiently",
	}

	var sb strings.Builder
	sb.WriteString("\nKey Features:\n")
	for _, feature := range features {
		_, _ = fmt.Fprintf(&sb, "  %s\n", feature)
	}
	return sb.String()
}

// Execute runs the root Cobra command.
// It shows an ASCII banner when no CLI arguments are provided and returns any execution error.
func Execute() error {
	if len(os.Args) <= 1 {
		showBanner()
	}
	err := rootCmd.Execute()
	if err != nil {
		renderCommandError(rootCmd.ErrOrStderr(), rootCmd, err)
	}
	return err
}

// showBanner prints the ASCII banner to stdout and only uses color on a TTY.
// It has no parameters and no return value.
func showBanner() {
	colorEnabled := false
	if os.Getenv("NO_COLOR") == "" {
		if info, err := os.Stdout.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
			colorEnabled = true
		}
	}
	renderBanner(os.Stdout, colorEnabled)
}

func renderBanner(writer io.Writer, colorEnabled bool) {
	_, _ = fmt.Fprintln(writer)
	banner := `
	 ██████╗  ██████╗ ██╗   ██╗███╗   ███╗ █████╗ ███╗   ██╗
	██╔════╝ ██╔═══██╗██║   ██║████╗ ████║██╔══██╗████╗  ██║
	██║  ███╗██║   ██║██║   ██║██╔████╔██║███████║██╔██╗ ██║
	██║   ██║██║   ██║╚██╗ ██╔╝██║╚██╔╝██║██╔══██║██║╚██╗██║
	╚██████╔╝╚██████╔╝ ╚████╔╝ ██║ ╚═╝ ██║██║  ██║██║ ╚████║
	 ╚═════╝  ╚═════╝   ╚═══╝  ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝`

	lines := strings.Split(banner, "\n")

	color, bold, reset := "", "", ""
	if colorEnabled {
		color = "\033[38;5;75m"
		bold = "\033[1m"
		reset = "\033[0m"
	}

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			_, _ = fmt.Fprintf(writer, "%s%s%s%s\n", color, bold, line, reset)
		}
	}
	_, _ = fmt.Fprintln(writer)
}

// initConfig loads a fresh isolated configuration and then applies explicit
// output flags. Failed loads are never cached.
func initConfig(quietChanged, verboseChanged bool) error {
	loaded, err := _config.Load(cfgFile)
	if err != nil {
		cfg = nil
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := applyOutputFlags(
		loaded,
		quietFlag,
		quietChanged,
		verboseFlag,
		verboseChanged,
	); err != nil {
		cfg = nil
		return err
	}
	if err := loaded.Validate(); err != nil {
		cfg = nil
		return fmt.Errorf("invalid effective config: %w", err)
	}

	level := _logger.NormalLevel
	if loaded.Quiet {
		level = _logger.QuietLevel
	} else if loaded.Verbose {
		level = _logger.VerboseLevel
	}
	_logger.Get().SetLevel(level)
	cfg = loaded
	return nil
}

func applyOutputFlags(config *_config.Config, quiet bool, quietChanged bool, verbose bool, verboseChanged bool) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}
	if quietChanged && verboseChanged && quiet && verbose {
		return fmt.Errorf("--quiet and --verbose cannot be enabled together")
	}
	if verboseChanged {
		config.Verbose = verbose
		if verbose {
			config.Quiet = false
		}
	}
	if quietChanged {
		config.Quiet = quiet
		if quiet {
			config.Verbose = false
		}
	}
	return nil
}

// getConfig returns the loaded configuration instance.
// No parameters; returns a pointer to Config.
func getConfig() *_config.Config {
	return cfg
}

// cleanupOldBackups removes leftover .bak files from previous self-updates.
// On Windows, the update process cannot delete the running binary's backup,
// so this routine handles cleanup on the next startup.
func cleanupOldBackups() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}

	dir := filepath.Dir(exePath)
	baseName := filepath.Base(exePath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		prefix := baseName + ".bak."
		if !strings.HasPrefix(entry.Name(), prefix) || !isDecimal(strings.TrimPrefix(entry.Name(), prefix)) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			_logger.Verbose("Failed to inspect old backup %s: %v", entry.Name(), err)
			continue
		}
		if !info.Mode().IsRegular() {
			_logger.Verbose("Skipping non-regular backup candidate: %s", entry.Name())
			continue
		}
		oldBackup := filepath.Join(dir, entry.Name())
		if err := os.Remove(oldBackup); err != nil {
			_logger.Verbose("Failed to remove old backup %s: %v", entry.Name(), err)
		}
	}
}

func isDecimal(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
