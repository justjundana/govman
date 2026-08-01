package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	viper "github.com/spf13/viper"
)

type Config struct {
	InstallDir     string           `mapstructure:"install_dir"`
	CacheDir       string           `mapstructure:"cache_dir"`
	DefaultVersion string           `mapstructure:"default_version"`
	Download       DownloadConfig   `mapstructure:"download"`
	Mirror         MirrorConfig     `mapstructure:"mirror"`
	AutoSwitch     AutoSwitchConfig `mapstructure:"auto_switch"`
	Shell          ShellConfig      `mapstructure:"shell"`
	GoReleases     GoReleasesConfig `mapstructure:"go_releases"`
	SelfUpdate     SelfUpdateConfig `mapstructure:"self_update"`
	Quiet          bool             `mapstructure:"quiet"`
	Verbose        bool             `mapstructure:"verbose"`
	configPath     string
	decoder        *viper.Viper
}

type DownloadConfig struct {
	Parallel       bool          `mapstructure:"parallel"`
	MaxConnections int           `mapstructure:"max_connections"`
	Timeout        time.Duration `mapstructure:"timeout"`
	RetryCount     int           `mapstructure:"retry_count"`
	RetryDelay     time.Duration `mapstructure:"retry_delay"`
}

type MirrorConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	URL     string `mapstructure:"url"`
}

type AutoSwitchConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	ProjectFile string `mapstructure:"project_file"`
}

type ShellConfig struct {
	AutoDetect bool `mapstructure:"auto_detect"`
	Completion bool `mapstructure:"completion"`
}

type GoReleasesConfig struct {
	APIURL      string        `mapstructure:"api_url"`
	DownloadURL string        `mapstructure:"download_url"`
	CacheExpiry time.Duration `mapstructure:"cache_expiry"`
}

type SelfUpdateConfig struct {
	GitHubAPIURL      string `mapstructure:"github_api_url"`
	GitHubReleasesURL string `mapstructure:"github_releases_url"`
}

// Load loads configuration from a YAML file.
// If configFile is empty, it defaults to ~/.govman/config.yaml.
// It applies defaults, reads/unmarshals the file, expands paths, ensures directories, and returns the Config or an error.
func Load(configFile string) (*Config, error) {
	cfg := &Config{}
	cfg.setDefaults()

	if configFile != "" {
		absolutePath, err := filepath.Abs(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve config file path: %w", err)
		}
		cfg.configPath = absolutePath
	} else {
		homeDir, err := getHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		cfg.configPath = filepath.Join(homeDir, ".govman", "config.yaml")
	}

	if _, err := os.Lstat(cfg.configPath); os.IsNotExist(err) {
		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("failed to create config file with default values: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to inspect config file: %w", err)
	} else if info, err := os.Lstat(cfg.configPath); err != nil {
		return nil, fmt.Errorf("failed to inspect config file: %w", err)
	} else if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("config file must be a regular file and not a symlink: %s", cfg.configPath)
	}
	if err := os.Chmod(cfg.configPath, 0600); err != nil {
		return nil, fmt.Errorf("failed to restrict config file permissions: %w", err)
	}

	decoder := newDecoder(cfg.configPath, cfg)
	if err := decoder.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := decoder.UnmarshalExact(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	cfg.decoder = decoder

	if err := cfg.expandPaths(); err != nil {
		return nil, fmt.Errorf("failed to expand paths: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if err := cfg.createDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}
	return cfg, nil
}

func newDecoder(configPath string, defaults *Config) *viper.Viper {
	decoder := viper.New()
	decoder.SetConfigFile(configPath)
	decoder.SetConfigType("yaml")
	decoder.SetDefault("install_dir", defaults.InstallDir)
	decoder.SetDefault("cache_dir", defaults.CacheDir)
	decoder.SetDefault("default_version", defaults.DefaultVersion)
	decoder.SetDefault("quiet", defaults.Quiet)
	decoder.SetDefault("verbose", defaults.Verbose)
	decoder.SetDefault("download.parallel", defaults.Download.Parallel)
	decoder.SetDefault("download.max_connections", defaults.Download.MaxConnections)
	decoder.SetDefault("download.timeout", defaults.Download.Timeout)
	decoder.SetDefault("download.retry_count", defaults.Download.RetryCount)
	decoder.SetDefault("download.retry_delay", defaults.Download.RetryDelay)
	decoder.SetDefault("mirror.enabled", defaults.Mirror.Enabled)
	decoder.SetDefault("mirror.url", defaults.Mirror.URL)
	decoder.SetDefault("auto_switch.enabled", defaults.AutoSwitch.Enabled)
	decoder.SetDefault("auto_switch.project_file", defaults.AutoSwitch.ProjectFile)
	decoder.SetDefault("shell.auto_detect", defaults.Shell.AutoDetect)
	decoder.SetDefault("shell.completion", defaults.Shell.Completion)
	decoder.SetDefault("go_releases.api_url", defaults.GoReleases.APIURL)
	decoder.SetDefault("go_releases.download_url", defaults.GoReleases.DownloadURL)
	decoder.SetDefault("go_releases.cache_expiry", defaults.GoReleases.CacheExpiry)
	decoder.SetDefault("self_update.github_api_url", defaults.SelfUpdate.GitHubAPIURL)
	decoder.SetDefault("self_update.github_releases_url", defaults.SelfUpdate.GitHubReleasesURL)
	return decoder
}

// setDefaults initializes default values for all Config fields:
// install/cache directories, download behavior, mirror, autoswitch, shell, releases API, and self-update endpoints.
func (c *Config) setDefaults() {
	homeDir, err := getHomeDir()
	if err != nil {
		homeDir = "."
	}
	govmanDir := filepath.Join(homeDir, ".govman")

	c.InstallDir = filepath.Join(govmanDir, "versions")
	c.CacheDir = filepath.Join(govmanDir, "cache")
	c.DefaultVersion = ""
	c.Quiet = false
	c.Verbose = false

	c.Download = DownloadConfig{
		Parallel:       true,
		MaxConnections: 4,
		Timeout:        300 * time.Second,
		RetryCount:     3,
		RetryDelay:     5 * time.Second,
	}

	c.Mirror = MirrorConfig{
		Enabled: false,
		URL:     "https://golang.google.cn/dl/",
	}

	c.AutoSwitch = AutoSwitchConfig{
		Enabled:     true,
		ProjectFile: ".govman-goversion",
	}

	c.Shell = ShellConfig{
		AutoDetect: true,
		Completion: true,
	}

	c.GoReleases = GoReleasesConfig{
		APIURL:      "https://go.dev/dl/?mode=json&include=all",
		DownloadURL: "https://go.dev/dl/%s",
		CacheExpiry: 10 * time.Minute,
	}

	c.SelfUpdate = SelfUpdateConfig{
		GitHubAPIURL:      "https://api.github.com/repos/justjundana/govman/releases/latest",
		GitHubReleasesURL: "https://api.github.com/repos/justjundana/govman/releases?per_page=1",
	}
}

// expandPaths expands and validates configured paths (e.g., handles ~), preventing traversal outside HOME.
// Returns an error if expansion/validation fails.
func (c *Config) expandPaths() error {
	var err error
	baseDir := filepath.Dir(c.configPath)
	if c.configPath == "" {
		baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to determine base directory: %w", err)
		}
	}

	c.InstallDir, err = expandPath(c.InstallDir, baseDir)
	if err != nil {
		return fmt.Errorf("failed to expand install_dir: %w", err)
	}

	c.CacheDir, err = expandPath(c.CacheDir, baseDir)
	if err != nil {
		return fmt.Errorf("failed to expand cache_dir: %w", err)
	}

	return nil
}

// createDirectories ensures required directories (install and cache) exist, creating them if necessary.
// Returns an error on filesystem failures.
func (c *Config) createDirectories() error {
	dirs := []string{c.InstallDir, c.CacheDir}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// Save writes the current Config to disk at configPath using viper.
// Uses atomic write (temp file + rename) to prevent corruption on crash.
// Returns an error if the config directory cannot be created or the file cannot be written.
func (c *Config) Save() error {
	if c.configPath == "" {
		return fmt.Errorf("config path is empty")
	}
	if err := c.expandPaths(); err != nil {
		return err
	}
	if err := c.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	configDir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	if info, err := os.Lstat(c.configPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("config file must be a regular file and not a symlink: %s", c.configPath)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect config file: %w", err)
	}

	decoder := viper.New()
	decoder.SetConfigType("yaml")
	decoder.Set("default_version", c.DefaultVersion)
	decoder.Set("install_dir", c.InstallDir)
	decoder.Set("cache_dir", c.CacheDir)
	decoder.Set("quiet", c.Quiet)
	decoder.Set("verbose", c.Verbose)
	decoder.Set("download.parallel", c.Download.Parallel)
	decoder.Set("download.max_connections", c.Download.MaxConnections)
	decoder.Set("download.timeout", c.Download.Timeout)
	decoder.Set("download.retry_count", c.Download.RetryCount)
	decoder.Set("download.retry_delay", c.Download.RetryDelay)
	decoder.Set("mirror.enabled", c.Mirror.Enabled)
	decoder.Set("mirror.url", c.Mirror.URL)
	decoder.Set("auto_switch.enabled", c.AutoSwitch.Enabled)
	decoder.Set("auto_switch.project_file", c.AutoSwitch.ProjectFile)
	decoder.Set("shell.auto_detect", c.Shell.AutoDetect)
	decoder.Set("shell.completion", c.Shell.Completion)
	decoder.Set("go_releases.api_url", c.GoReleases.APIURL)
	decoder.Set("go_releases.download_url", c.GoReleases.DownloadURL)
	decoder.Set("go_releases.cache_expiry", c.GoReleases.CacheExpiry)
	decoder.Set("self_update.github_api_url", c.SelfUpdate.GitHubAPIURL)
	decoder.Set("self_update.github_releases_url", c.SelfUpdate.GitHubReleasesURL)

	tempFile, err := os.CreateTemp(configDir, ".govman-config-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temporary config file: %w", err)
	}
	tempPath := tempFile.Name()
	closed := false
	defer func() {
		if !closed {
			_ = tempFile.Close()
		}
		_ = os.Remove(tempPath)
	}()
	if err := tempFile.Chmod(0600); err != nil {
		return fmt.Errorf("failed to secure temporary config file: %w", err)
	}
	if err := decoder.WriteConfigTo(tempFile); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync config file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close config file: %w", err)
	}
	closed = true
	if err := os.Rename(tempPath, c.configPath); err != nil {
		return fmt.Errorf("failed to save config file: %w", err)
	}
	c.decoder = decoder

	return nil
}

// Validate rejects values that would make downloads, path management, or endpoint resolution unsafe.
func (c *Config) Validate() error {
	if c.Download.Timeout <= 0 {
		return fmt.Errorf("download.timeout must be greater than zero")
	}
	if c.Download.RetryCount < 1 {
		return fmt.Errorf("download.retry_count must be at least 1")
	}
	if c.Download.RetryDelay < 0 {
		return fmt.Errorf("download.retry_delay cannot be negative")
	}
	if c.Download.MaxConnections < 1 {
		return fmt.Errorf("download.max_connections must be at least 1")
	}
	if c.GoReleases.CacheExpiry <= 0 {
		return fmt.Errorf("go_releases.cache_expiry must be greater than zero")
	}
	if c.Quiet && c.Verbose {
		return fmt.Errorf("quiet and verbose cannot both be enabled")
	}
	if err := validateProjectFilename(c.AutoSwitch.ProjectFile); err != nil {
		return err
	}
	if pathsOverlap(c.InstallDir, c.CacheDir) {
		return fmt.Errorf("install_dir and cache_dir must not overlap")
	}
	for name, value := range map[string]string{
		"mirror.url":                      c.Mirror.URL,
		"go_releases.api_url":             c.GoReleases.APIURL,
		"self_update.github_api_url":      c.SelfUpdate.GitHubAPIURL,
		"self_update.github_releases_url": c.SelfUpdate.GitHubReleasesURL,
	} {
		if err := validateHTTPURL(name, value); err != nil {
			return err
		}
	}
	if strings.Count(c.GoReleases.DownloadURL, "%s") != 1 {
		return fmt.Errorf("go_releases.download_url must contain exactly one %%s placeholder")
	}
	if err := validateHTTPURL("go_releases.download_url", strings.Replace(c.GoReleases.DownloadURL, "%s", "archive", 1)); err != nil {
		return err
	}
	return nil
}

func validateHTTPURL(name, value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return fmt.Errorf("%s must be an absolute http(s) URL without credentials", name)
	}
	return nil
}

func validateProjectFilename(value string) error {
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, `/\`) || strings.ContainsRune(value, '\x00') || filepath.Base(value) != value {
		return fmt.Errorf("auto_switch.project_file must be a filename without path separators")
	}
	return nil
}

func pathsOverlap(first, second string) bool {
	for _, pair := range [][2]string{{first, second}, {second, first}} {
		relative, err := filepath.Rel(pair[0], pair[1])
		if err == nil && (relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))) {
			return true
		}
	}
	return false
}

// GetVersionDir returns the installation directory for a given Go version, e.g., ~/.govman/versions/go1.25.1.
func (c *Config) GetVersionDir(version string) string {
	return filepath.Join(c.InstallDir, fmt.Sprintf("go%s", version))
}

// GetBinPath returns the path to the govman bin directory, typically ~/.govman/bin.
func (c *Config) GetBinPath() string {
	homeDir, err := getHomeDir()
	if err != nil {
		homeDir = "."
	}

	return filepath.Join(homeDir, ".govman", "bin")
}

// GetCurrentSymlink returns the path to the global "go" symlink inside the bin directory.
func (c *Config) GetCurrentSymlink() string {
	return filepath.Join(c.GetBinPath(), "go")
}

// ConfigPath returns the effective configuration file path.
func (c *Config) ConfigPath() string {
	return c.configPath
}

// getHomeDir returns the current user's HOME directory (USERPROFILE on Windows).
// Returns an error if it cannot be determined.
func getHomeDir() (string, error) {
	var homeDir string
	if runtime.GOOS == "windows" {
		homeDir = os.Getenv("USERPROFILE")
	} else {
		homeDir = os.Getenv("HOME")
	}

	if homeDir == "" {
		return "", fmt.Errorf("unable to determine home directory: HOME/USERPROFILE environment variable is not set")
	}

	return homeDir, nil
}

// expandPath expands a leading ~ to the home directory and validates the result against traversal outside HOME.
// Returns the expanded path or an error for invalid formats or traversal attempts.
func expandPath(path, baseDir string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("empty path provided")
	}
	if strings.ContainsRune(path, '\x00') {
		return "", fmt.Errorf("path contains NUL")
	}
	if path[0] == '~' {
		homeDir, err := getHomeDir()
		if err != nil {
			return "", err
		}

		if len(path) > 1 && path[1] != '/' && path[1] != '\\' {
			return "", fmt.Errorf("invalid path format: paths starting with ~ must be followed by / or \\")
		}

		remainder := strings.TrimLeft(path[1:], `/\`)
		expandedPath := filepath.Join(homeDir, remainder)

		rel, err := filepath.Rel(homeDir, expandedPath)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate relative path: %w", err)
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("path traversal detected: expanded path is outside home directory")
		}

		return filepath.Clean(expandedPath), nil
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	for _, component := range strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' }) {
		if component == ".." {
			return "", fmt.Errorf("relative path traversal is not allowed")
		}
	}
	return filepath.Abs(filepath.Join(baseDir, path))
}
