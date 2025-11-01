package cli

import (
	"fmt"
	"os"
	"strings"
	"sync"

	cobra "github.com/spf13/cobra"

	_config "github.com/justjundana/govman/internal/config"
	_version "github.com/justjundana/govman/internal/version"
)

var (
	cfgFile  string
	cfg      *_config.Config
	cfgMutex sync.Mutex
	cfgOnce  sync.Once
)

var rootCmd = &cobra.Command{
	Use:     "govman",
	Short:   "Go Version Manager - Install and manage multiple Go versions",
	Long:    createLongDescription(),
	Version: _version.BuildVersion(),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
}

// createLongDescription returns a formatted long description string for the root CLI command.
// It assembles key features into a multi-line string and returns it.
func createLongDescription() string {
	features := []string{
		"⚡ Lightning-fast installation and switching between Go versions",
		"🎯 Zero configuration - works out of the box, no setup required",
		"📁 Project-specific versions with .govman-version file support",
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

// initConfig lazily loads the application configuration once using sync.Once.
// Returns an error if configuration loading fails; otherwise nil.
func initConfig() error {
	var initErr error
	cfgOnce.Do(func() {
		var err error
		cfg, err = _config.Load(cfgFile)
		if err != nil {
			initErr = fmt.Errorf("failed to load config: %w", err)
		}
	})
	return initErr
}

// getConfig returns the loaded configuration instance.
// No parameters; returns a pointer to Config.
func getConfig() *_config.Config {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	return cfg
}
