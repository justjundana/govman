package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func setTestEnv(key, value string) func() {
	oldValue, existed := os.LookupEnv(key)
	_ = os.Setenv(key, value)
	return func() {
		if existed {
			_ = os.Setenv(key, oldValue)
			return
		}
		_ = os.Unsetenv(key)
	}
}

func unsetTestHome() func() {
	restoreHome := setTestEnv("HOME", "")
	restoreUserProfile := setTestEnv("USERPROFILE", "")
	_ = os.Unsetenv("HOME")
	_ = os.Unsetenv("USERPROFILE")
	return func() {
		restoreUserProfile()
		restoreHome()
	}
}

// setTestHomeEnv points both home-directory environment variables at tempHome and
// returns an exact restore func. getHomeDir reads USERPROFILE on Windows and HOME
// everywhere else, so setting only one of them would leave the real user profile
// in play on the other platform.
func setTestHomeEnv(tempHome string) func() {
	restoreHome := setTestEnv("HOME", tempHome)
	restoreUserProfile := setTestEnv("USERPROFILE", tempHome)
	return func() {
		restoreUserProfile()
		restoreHome()
	}
}

// setTestHome redirects the home directory to tempHome for the duration of the test.
func setTestHome(t *testing.T, tempHome string) {
	t.Helper()
	t.Cleanup(setTestHomeEnv(tempHome))
}

// blockGovmanConfigDir plants a regular file where <tempHome>/.govman has to be a
// directory, so Load cannot produce a config file. This replaces a chmod 0444 on
// that directory: on Windows os.Chmod only toggles FILE_ATTRIBUTE_READONLY, which
// is ignored for directories, so a read-only .govman would still accept MkdirAll
// and CreateTemp and Load would succeed.
//
// On linux/darwin os.Lstat(<tempHome>/.govman/config.yaml) fails with ENOTDIR,
// which is not os.IsNotExist, so Load reports "failed to inspect config file".
// On Windows the same Lstat reports ERROR_PATH_NOT_FOUND, which maps to
// fs.ErrNotExist, so Load falls through to Save() where os.MkdirAll stats the
// regular file, sees !IsDir and returns ENOTDIR, and Load reports "failed to
// create config file with default values". Both are errors, on every platform.
func blockGovmanConfigDir(t *testing.T, tempHome string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(tempHome, ".govman"), []byte("not a directory\n"), 0600); err != nil {
		t.Fatalf("Failed to plant blocking file: %v", err)
	}
}

// occupyGovmanConfigPath creates a directory at the exact path Load expects the
// config file to be. os.Lstat then succeeds while reporting a non-regular file,
// which Load rejects identically on linux, darwin and windows.
func occupyGovmanConfigPath(t *testing.T, tempHome string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(tempHome, ".govman", "config.yaml"), 0755); err != nil {
		t.Fatalf("Failed to create directory at config path: %v", err)
	}
}

func TestLoad(t *testing.T) {
	testCases := []struct {
		name        string
		configFile  string
		setup       func(t *testing.T) string
		expectError bool
		cleanup     func(string)
		validate    func(t *testing.T, cfg *Config, configPath string)
	}{
		{
			name: "Load default config",
			setup: func(t *testing.T) string {
				setTestHome(t, t.TempDir())
				return ""
			},
			expectError: false,
		},
		{
			name: "Load custom config file",
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "custom.yaml")

				configContent := `install_dir: "/tmp/custom/install"
cache_dir: "/tmp/custom/cache"
default_version: "1.21.0"`
				if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}

				return configPath
			},
			expectError: false,
		},
		{
			name: "Config file with invalid YAML",
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "invalid.yaml")

				if err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644); err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}

				return configPath
			},
			expectError: true,
		},
		{
			name: "Home directory not accessible",
			setup: func(t *testing.T) string {
				t.Cleanup(unsetTestHome())
				return ""
			},
			expectError: true,
		},
		{
			name: "Config directory path occupied by a regular file",
			setup: func(t *testing.T) string {
				tempHome := t.TempDir()
				setTestHome(t, tempHome)
				blockGovmanConfigDir(t, tempHome)
				return ""
			},
			expectError: true,
		},
		{
			name: "Successful config creation when file doesn't exist",
			setup: func(t *testing.T) string {
				setTestHome(t, t.TempDir())
				return ""
			},
			expectError: false,
		},
		{
			name: "Config path occupied by a directory",
			setup: func(t *testing.T) string {
				tempHome := t.TempDir()
				setTestHome(t, tempHome)
				occupyGovmanConfigPath(t, tempHome)
				return ""
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var configPath string
			if tc.setup != nil {
				configPath = tc.setup(t)
			}

			cfg, err := Load(configPath)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error but got: %v", err)
			}

			if cfg == nil {
				t.Fatal("Config should not be nil")
			}

			// Verify some default values
			if cfg.Download.Timeout != 300*time.Second {
				t.Errorf("Expected download timeout 300s, got %v", cfg.Download.Timeout)
			}
			if cfg.GoReleases.APIURL != "https://go.dev/dl/?mode=json&include=all" {
				t.Errorf("Expected Go releases API URL, got %s", cfg.GoReleases.APIURL)
			}

			if tc.validate != nil {
				tc.validate(t, cfg, configPath)
			}
		})
	}
}

func TestSetDefaults(t *testing.T) {
	// Set up fake home directory
	tempHome := t.TempDir()
	defer setTestHomeEnv(tempHome)()

	cfg := &Config{}
	cfg.setDefaults()

	// Check default values
	expectedInstallDir := filepath.Join(tempHome, ".govman", "versions")
	if cfg.InstallDir != expectedInstallDir {
		t.Errorf("Expected install dir %s, got %s", expectedInstallDir, cfg.InstallDir)
	}

	expectedCacheDir := filepath.Join(tempHome, ".govman", "cache")
	if cfg.CacheDir != expectedCacheDir {
		t.Errorf("Expected cache dir %s, got %s", expectedCacheDir, cfg.CacheDir)
	}

	if cfg.DefaultVersion != "" {
		t.Errorf("Expected empty default version, got %s", cfg.DefaultVersion)
	}

	if cfg.Download.Timeout != 300*time.Second {
		t.Errorf("Expected download timeout 300s, got %v", cfg.Download.Timeout)
	}

	if cfg.GoReleases.CacheExpiry != 10*time.Minute {
		t.Errorf("Expected cache expiry 10m, got %v", cfg.GoReleases.CacheExpiry)
	}
}

func TestExpandPaths(t *testing.T) {
	testCases := []struct {
		name        string
		installDir  string
		cacheDir    string
		expectError bool
		setup       func() func()
	}{
		{
			name:        "Valid absolute paths",
			installDir:  "/tmp/test/install",
			cacheDir:    "/tmp/test/cache",
			expectError: false,
		},
		{
			name:        "Valid relative paths",
			installDir:  "versions",
			cacheDir:    "cache",
			expectError: false,
		},
		{
			name:        "Tilde expansion",
			installDir:  "~/test/install",
			cacheDir:    "~/test/cache",
			expectError: false,
		},
		{
			name:        "Path traversal attempt",
			installDir:  "~/../../../etc",
			cacheDir:    "~/test/cache",
			expectError: true,
		},
		{
			name:        "InstallDir expansion fails",
			installDir:  "",
			cacheDir:    "~/test/cache",
			expectError: true,
		},
		{
			name:        "CacheDir expansion fails",
			installDir:  "~/test/install",
			cacheDir:    "",
			expectError: true,
		},
		{
			name:        "Both expansions fail",
			installDir:  "",
			cacheDir:    "",
			expectError: true,
		},
		{
			name:        "Invalid tilde format",
			installDir:  "~invalid",
			cacheDir:    "/tmp/cache",
			expectError: true,
		},
		{
			name:        "GetHomeDir fails",
			installDir:  "~/test/install",
			cacheDir:    "~/test/cache",
			expectError: true,
			setup: func() func() {
				return unsetTestHome()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				cleanup := tc.setup()
				defer cleanup()
			}

			cfg := &Config{
				InstallDir: tc.installDir,
				CacheDir:   tc.cacheDir,
			}

			err := cfg.expandPaths()

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Verify paths were expanded
			if strings.HasPrefix(cfg.InstallDir, "~") {
				t.Error("InstallDir should not contain tilde after expansion")
			}
			if strings.HasPrefix(cfg.CacheDir, "~") {
				t.Error("CacheDir should not contain tilde after expansion")
			}
		})
	}
}

func TestCreateDirectories(t *testing.T) {
	testCases := []struct {
		name         string
		installDir   string
		cacheDir     string
		blockInstall bool
		blockCache   bool
		expectError  bool
	}{
		{
			name:        "Valid directories",
			installDir:  "install",
			cacheDir:    "cache",
			expectError: false,
		},
		{
			name:         "Install directory creation fails",
			installDir:   "install",
			cacheDir:     "cache",
			blockInstall: true,
			expectError:  true,
		},
		{
			name:        "Cache directory creation fails",
			installDir:  "install",
			cacheDir:    "cache",
			blockCache:  true,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			cfg := &Config{
				InstallDir: filepath.Join(tempDir, tc.installDir),
				CacheDir:   filepath.Join(tempDir, tc.cacheDir),
			}

			// For error cases, route the directory through a regular file so that
			// os.MkdirAll fails with ENOTDIR on linux, darwin and windows alike.
			// A hardcoded POSIX path such as "/invalid/path/..." is not portable:
			// on Windows it resolves against the current drive, where MkdirAll can
			// legitimately succeed.
			if tc.blockInstall || tc.blockCache {
				blocker := filepath.Join(tempDir, "blocker")
				if err := os.WriteFile(blocker, []byte("not a directory\n"), 0600); err != nil {
					t.Fatalf("Failed to create blocker file: %v", err)
				}
				if tc.blockInstall {
					cfg.InstallDir = filepath.Join(blocker, tc.installDir)
				}
				if tc.blockCache {
					cfg.CacheDir = filepath.Join(blocker, tc.cacheDir)
				}
			}

			err := cfg.createDirectories()

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Failed to create directories: %v", err)
			}

			// Verify directories were created
			if _, err := os.Stat(cfg.InstallDir); os.IsNotExist(err) {
				t.Error("Install directory was not created")
			}
			if _, err := os.Stat(cfg.CacheDir); os.IsNotExist(err) {
				t.Error("Cache directory was not created")
			}
		})
	}
}

func TestSave(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	cfg := &Config{
		configPath:     configPath,
		DefaultVersion: "1.21.0",
		InstallDir:     "/tmp/install",
		CacheDir:       "/tmp/cache",
		Quiet:          true,
		Verbose:        false,
	}

	cfg.setDefaults() // Set other defaults
	cfg.DefaultVersion = "1.21.0"

	err := cfg.Save()
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Verify we can load it back
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedCfg.DefaultVersion != cfg.DefaultVersion {
		t.Errorf("Expected default version %s, got %s", cfg.DefaultVersion, loadedCfg.DefaultVersion)
	}
}

func TestSaveFailure(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(t *testing.T) *Config
		expectError bool
	}{
		{
			name: "Save fails on invalid path",
			setup: func(t *testing.T) *Config {
				cfg := &Config{
					configPath:     "/invalid/path/that/does/not/exist/config.yaml",
					DefaultVersion: "1.21.0",
				}
				return cfg
			},
			expectError: true,
		},
		{
			name: "Save fails on write error",
			setup: func(t *testing.T) *Config {
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "test-config.yaml")
				cfg := &Config{
					configPath:     configPath,
					DefaultVersion: "1.21.0",
				}
				// Make the directory read-only to cause WriteConfigAs to fail
				err := os.Chmod(tempDir, 0444)
				if err != nil {
					t.Fatalf("Failed to make temp dir read-only: %v", err)
				}
				t.Cleanup(func() {
					os.Chmod(tempDir, 0755)
				})
				return cfg
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.setup(t)
			err := cfg.Save()
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestLoadUsesIsolatedViperInstances(t *testing.T) {
	home := t.TempDir()
	setTestHome(t, home)

	firstDir := t.TempDir()
	secondDir := t.TempDir()
	firstPath := filepath.Join(firstDir, "first.yaml")
	secondPath := filepath.Join(secondDir, "second.yaml")
	if err := os.WriteFile(firstPath, []byte("default_version: 1.21.1\ninstall_dir: first-versions\ncache_dir: first-cache\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondPath, []byte("default_version: 1.22.2\ninstall_dir: second-versions\ncache_dir: second-cache\n"), 0600); err != nil {
		t.Fatal(err)
	}

	first, err := Load(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Load(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if first.decoder == nil || second.decoder == nil || first.decoder == second.decoder {
		t.Fatal("config loads must retain distinct decoder instances")
	}
	if first.DefaultVersion != "1.21.1" || second.DefaultVersion != "1.22.2" {
		t.Fatalf("config values leaked across instances: first=%q second=%q", first.DefaultVersion, second.DefaultVersion)
	}
	if first.InstallDir != filepath.Join(firstDir, "first-versions") {
		t.Fatalf("first relative install path = %q", first.InstallDir)
	}
	if second.CacheDir != filepath.Join(secondDir, "second-cache") {
		t.Fatalf("second relative cache path = %q", second.CacheDir)
	}
}

func TestLoadRejectsUnknownKeysAndSymlinks(t *testing.T) {
	setTestHome(t, t.TempDir())

	t.Run("unknown top-level key", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(path, []byte("download_typo: true\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "invalid keys") {
			t.Fatalf("expected strict unmarshal error, got %v", err)
		}
	})

	t.Run("unknown nested key", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(path, []byte("download:\n  retries: 3\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "invalid keys") {
			t.Fatalf("expected strict nested-key error, got %v", err)
		}
	})

	if runtime.GOOS != "windows" {
		t.Run("symlink", func(t *testing.T) {
			directory := t.TempDir()
			target := filepath.Join(directory, "target.yaml")
			link := filepath.Join(directory, "config.yaml")
			if err := os.WriteFile(target, []byte("{}\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(link); err == nil || !strings.Contains(err.Error(), "not a symlink") {
				t.Fatalf("expected symlink rejection, got %v", err)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	newValidConfig := func(t *testing.T) *Config {
		t.Helper()
		setTestHome(t, t.TempDir())
		config := &Config{}
		config.setDefaults()
		return config
	}

	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"zero timeout", func(c *Config) { c.Download.Timeout = 0 }, "download.timeout"},
		{"zero retries", func(c *Config) { c.Download.RetryCount = 0 }, "download.retry_count"},
		{"negative retry delay", func(c *Config) { c.Download.RetryDelay = -time.Second }, "download.retry_delay"},
		{"zero max connections", func(c *Config) { c.Download.MaxConnections = 0 }, "download.max_connections"},
		{"zero cache expiry", func(c *Config) { c.GoReleases.CacheExpiry = 0 }, "cache_expiry"},
		{"overlapping paths", func(c *Config) { c.CacheDir = filepath.Join(c.InstallDir, "cache") }, "must not overlap"},
		{"path project filename", func(c *Config) { c.AutoSwitch.ProjectFile = "nested/version" }, "project_file"},
		{"windows project filename", func(c *Config) { c.AutoSwitch.ProjectFile = `nested\version` }, "project_file"},
		{"conflicting output", func(c *Config) { c.Quiet, c.Verbose = true, true }, "cannot both"},
		{"invalid mirror URL", func(c *Config) { c.Mirror.URL = "ftp://example.com" }, "mirror.url"},
		{"credentialed URL", func(c *Config) { c.SelfUpdate.GitHubAPIURL = "https://user:pass@example.com/releases" }, "github_api_url"},
		{"missing download placeholder", func(c *Config) { c.GoReleases.DownloadURL = "https://go.dev/dl/archive" }, "exactly one"},
		{"duplicate download placeholder", func(c *Config) { c.GoReleases.DownloadURL = "https://go.dev/%s/%s" }, "exactly one"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := newValidConfig(t)
			test.mutate(config)
			if err := config.Validate(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Validate() error = %v, want substring %q", err, test.want)
			}
		})
	}

	t.Run("prefix paths do not overlap", func(t *testing.T) {
		config := newValidConfig(t)
		config.InstallDir = filepath.Join(t.TempDir(), "go")
		config.CacheDir = config.InstallDir + "-cache"
		if err := config.Validate(); err != nil {
			t.Fatalf("prefix-like paths should be valid: %v", err)
		}
	})
}

func TestConfigFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix permission bits")
	}
	setTestHome(t, t.TempDir())
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if permissions := info.Mode().Perm(); permissions != 0600 {
		t.Fatalf("config permissions = %o, want 600", permissions)
	}
}

func TestGetVersionDir(t *testing.T) {
	cfg := &Config{
		InstallDir: "/opt/govman/versions",
	}

	version := "1.21.0"
	expected := filepath.Join(cfg.InstallDir, "go"+version)
	result := cfg.GetVersionDir(version)

	if result != expected {
		t.Errorf("Expected version dir %s, got %s", expected, result)
	}
}

func TestGetBinPath(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func() func()
		expectError bool
		mockEnv     func(string) string
	}{
		{
			name: "Valid HOME on Unix",
			setup: func() func() {
				return setTestHomeEnv(t.TempDir())
			},
			expectError: false,
		},
		{
			name: "Valid USERPROFILE on Windows",
			setup: func() func() {
				return setTestHomeEnv(t.TempDir())
			},
			expectError: false,
		},
		{
			name: "Fallback when home directory not found",
			setup: func() func() {
				return unsetTestHome()
			},
			expectError: false, // GetBinPath doesn't return error, it falls back to "."
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := tc.setup()
			defer cleanup()

			cfg := &Config{}
			result := cfg.GetBinPath()

			// Verify result is not empty
			if result == "" {
				t.Error("Bin path should not be empty")
			}

			// Verify it contains the expected structure (except for fallback case)
			if tc.name != "Fallback when home directory not found" {
				if !strings.Contains(result, ".govman") || !strings.Contains(result, "bin") {
					t.Errorf("Bin path should contain .govman/bin, got: %s", result)
				}
			} else {
				// For fallback case, it should contain "bin" but not necessarily ".govman"
				if !strings.Contains(result, "bin") {
					t.Errorf("Fallback bin path should contain bin, got: %s", result)
				}
			}
		})
	}
}

func TestGetCurrentSymlink(t *testing.T) {
	testCases := []struct {
		name  string
		setup func() func()
	}{
		{
			name: "Valid HOME on Unix",
			setup: func() func() {
				return setTestHomeEnv(t.TempDir())
			},
		},
		{
			name: "Valid USERPROFILE on Windows",
			setup: func() func() {
				return setTestHomeEnv(t.TempDir())
			},
		},
		{
			name: "Fallback when home directory not found",
			setup: func() func() {
				return unsetTestHome()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := tc.setup()
			defer cleanup()

			cfg := &Config{}
			result := cfg.GetCurrentSymlink()

			// Verify result is not empty
			if result == "" {
				t.Error("Symlink path should not be empty")
			}

			// Verify it contains the expected structure
			if !strings.Contains(result, ".govman") || !strings.Contains(result, "bin") || !strings.HasSuffix(result, "go") {
				t.Errorf("Symlink path should contain .govman/bin/go, got: %s", result)
			}
		})
	}
}
