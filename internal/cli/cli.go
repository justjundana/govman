package cli

import (
	"fmt"
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
	Use:     "govman",
	Short:   "Go Version Manager - Install and manage multiple Go versions",
	Long:    createLongDescription(),
	Version: _version.BuildVersion(),
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
		"💾 Intelligent caching with offline mode support",
		"📦 Parallel downloads with automatic resume on failure",
		"🌍 Cross-platform support (Windows, macOS, Linux, ARM)",
		"🧹 Built-in cleanup tools to manage disk space efficiently",
	}

	var sb strings.Builder
	sb.WriteString("\nKey Features:\n")
	for _, feature := range features {
		sb.WriteString(fmt.Sprintf("  %s\n", feature))
	}
	return sb.String()
}

// Execute runs the root Cobra command.
// It shows an ASCII banner when no CLI arguments are provided and returns any execution error.
func Execute() error {

	if len(os.Args) <= 1 {
		showBanner()
	}
	return rootCmd.Execute()
}

// showBanner prints a colored ASCII banner to stdout.
// It has no parameters and no return value.
func showBanner() {
	fmt.Println()
	banner := `
	 ██████╗  ██████╗ ██╗   ██╗███╗   ███╗ █████╗ ███╗   ██╗
	██╔════╝ ██╔═══██╗██║   ██║████╗ ████║██╔══██╗████╗  ██║
	██║  ███╗██║   ██║██║   ██║██╔████╔██║███████║██╔██╗ ██║
	██║   ██║██║   ██║╚██╗ ██╔╝██║╚██╔╝██║██╔══██║██║╚██╗██║
	╚██████╔╝╚██████╔╝ ╚████╔╝ ██║ ╚═╝ ██║██║  ██║██║ ╚████║
	 ╚═════╝  ╚═════╝   ╚═══╝  ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝`

	lines := strings.Split(banner, "\n")

	const (
		color = "\033[38;5;75m"
		bold  = "\033[1m"
		reset = "\033[0m"
	)

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			fmt.Printf("%s%s%s%s\n", color, bold, line, reset)
		}
	}
	fmt.Println()
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
		if strings.HasPrefix(entry.Name(), baseName+".bak.") {
			oldBackup := filepath.Join(dir, entry.Name())
			os.Remove(oldBackup) // Best-effort, ignore errors
		}
	}
}
