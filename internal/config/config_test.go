package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

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
				// Create temp home directory
				tempHome := t.TempDir()
				oldHome := os.Getenv("HOME")
				if runtime.GOOS == "windows" {
					os.Setenv("USERPROFILE", tempHome)
					t.Cleanup(func() { os.Setenv("USERPROFILE", oldHome) })
				} else {
					os.Setenv("HOME", tempHome)
					t.Cleanup(func() { os.Setenv("HOME", oldHome) })
				}
				return ""
			},
			expectError: false,
		},
		{
			name: "Load custom config file",
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "custom.yaml")

				// Create a custom config file
				configContent := `install_dir: "/tmp/custom/install"
cache_dir: "/tmp/custom/cache"
default_version: "1.21.0"`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				if err != nil {
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

				// Create invalid YAML
				err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
				if err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}

				return configPath
			},
			expectError: true,
		},
		{
			name: "Home directory not accessible",
			setup: func(t *testing.T) string {
				oldHome := os.Getenv("HOME")
				oldUserProfile := os.Getenv("USERPROFILE")
				os.Unsetenv("HOME")
				os.Unsetenv("USERPROFILE")
				t.Cleanup(func() {
					os.Setenv("HOME", oldHome)
					if oldUserProfile != "" {
						os.Setenv("USERPROFILE", oldUserProfile)
					}
				})
				return ""
			},
			expectError: true,
		},
		{
			name: "Config save fails during initial creation",
			setup: func(t *testing.T) string {
				tempHome := t.TempDir()
				oldHome := os.Getenv("HOME")
				if runtime.GOOS == "windows" {
					os.Setenv("USERPROFILE", tempHome)
					t.Cleanup(func() { os.Setenv("USERPROFILE", oldHome) })
				} else {
					os.Setenv("HOME", tempHome)
					t.Cleanup(func() { os.Setenv("HOME", oldHome) })
				}
				// Make the .govman directory read-only to cause Save to fail
				govmanDir := filepath.Join(tempHome, ".govman")
				err := os.MkdirAll(govmanDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create govman dir: %v", err)
				}
				err = os.Chmod(govmanDir, 0444) // Read-only
				if err != nil {
					t.Fatalf("Failed to make govman dir read-only: %v", err)
				}
				t.Cleanup(func() {
					os.Chmod(govmanDir, 0755) // Restore permissions for cleanup
				})
				return ""
			},
			expectError: true,
		},
		{
			name: "Successful config creation when file doesn't exist",
			setup: func(t *testing.T) string {
				tempHome := t.TempDir()
				oldHome := os.Getenv("HOME")
				if runtime.GOOS == "windows" {
					os.Setenv("USERPROFILE", tempHome)
					t.Cleanup(func() { os.Setenv("USERPROFILE", oldHome) })
				} else {
					os.Setenv("HOME", tempHome)
					t.Cleanup(func() { os.Setenv("HOME", oldHome) })
				}
				// Don't create the config file - let Load create it
				return ""
			},
			expectError: false,
		},
		{
			name: "Home directory accessible but config creation fails",
			setup: func(t *testing.T) string {
				tempHome := t.TempDir()
				oldHome := os.Getenv("HOME")
				if runtime.GOOS == "windows" {
					os.Setenv("USERPROFILE", tempHome)
					t.Cleanup(func() { os.Setenv("USERPROFILE", oldHome) })
				} else {
					os.Setenv("HOME", tempHome)
					t.Cleanup(func() { os.Setenv("HOME", oldHome) })
				}
				// Make the .govman directory read-only to cause Save to fail
				govmanDir := filepath.Join(tempHome, ".govman")
				err := os.MkdirAll(govmanDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create govman dir: %v", err)
				}
				err = os.Chmod(govmanDir, 0444) // Read-only
				if err != nil {
					t.Fatalf("Failed to make govman dir read-only: %v", err)
				}
				t.Cleanup(func() {
					os.Chmod(govmanDir, 0755) // Restore permissions for cleanup
				})
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
	oldHome := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", tempHome)
		defer os.Setenv("USERPROFILE", oldHome)
	} else {
		os.Setenv("HOME", tempHome)
		defer os.Setenv("HOME", oldHome)
	}

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
				oldHome := os.Getenv("HOME")
				oldUserProfile := os.Getenv("USERPROFILE")
				os.Unsetenv("HOME")
				os.Unsetenv("USERPROFILE")
				return func() {
					if oldHome != "" {
						os.Setenv("HOME", oldHome)
					}
					if oldUserProfile != "" {
						os.Setenv("USERPROFILE", oldUserProfile)
					}
				}
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
		name        string
		installDir  string
		cacheDir    string
		expectError bool
	}{
		{
			name:        "Valid directories",
			installDir:  "install",
			cacheDir:    "cache",
			expectError: false,
		},
		{
			name:        "Install directory creation fails",
			installDir:  "/invalid/path/that/does/not/exist/install",
			cacheDir:    "cache",
			expectError: true,
		},
		{
			name:        "Cache directory creation fails",
			installDir:  "install",
			cacheDir:    "/invalid/path/that/does/not/exist/cache",
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

			// For error cases, use absolute paths that will fail
			if tc.expectError && strings.Contains(tc.installDir, "invalid") {
				cfg.InstallDir = tc.installDir
			}
			if tc.expectError && strings.Contains(tc.cacheDir, "invalid") {
				cfg.CacheDir = tc.cacheDir
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
				tempHome := t.TempDir()
				oldHome := os.Getenv("HOME")
				if runtime.GOOS != "windows" {
					os.Setenv("HOME", tempHome)
					return func() { os.Setenv("HOME", oldHome) }
				}
				os.Setenv("USERPROFILE", tempHome)
				return func() { os.Setenv("USERPROFILE", oldHome) }
			},
			expectError: false,
		},
		{
			name: "Valid USERPROFILE on Windows",
			setup: func() func() {
				tempHome := t.TempDir()
				oldHome := os.Getenv("USERPROFILE")
				if runtime.GOOS == "windows" {
					os.Setenv("USERPROFILE", tempHome)
					return func() { os.Setenv("USERPROFILE", oldHome) }
				}
				os.Setenv("HOME", tempHome)
				return func() { os.Setenv("HOME", oldHome) }
			},
			expectError: false,
		},
		{
			name: "Fallback when home directory not found",
			setup: func() func() {
				oldHome := os.Getenv("HOME")
				oldUserProfile := os.Getenv("USERPROFILE")
				os.Unsetenv("HOME")
				os.Unsetenv("USERPROFILE")
				return func() {
					if oldHome != "" {
						os.Setenv("HOME", oldHome)
					}
					if oldUserProfile != "" {
						os.Setenv("USERPROFILE", oldUserProfile)
					}
				}
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
				tempHome := t.TempDir()
				oldHome := os.Getenv("HOME")
				if runtime.GOOS != "windows" {
					os.Setenv("HOME", tempHome)
					return func() { os.Setenv("HOME", oldHome) }
				}
				os.Setenv("USERPROFILE", tempHome)
				return func() { os.Setenv("USERPROFILE", oldHome) }
			},
		},
		{
			name: "Valid USERPROFILE on Windows",
			setup: func() func() {
				tempHome := t.TempDir()
				oldHome := os.Getenv("USERPROFILE")
				if runtime.GOOS == "windows" {
					os.Setenv("USERPROFILE", tempHome)
					return func() { os.Setenv("USERPROFILE", oldHome) }
				}
				os.Setenv("HOME", tempHome)
				return func() { os.Setenv("HOME", oldHome) }
			},
		},
		{
			name: "Fallback when home directory not found",
			setup: func() func() {
				oldHome := os.Getenv("HOME")
				oldUserProfile := os.Getenv("USERPROFILE")
				os.Unsetenv("HOME")
				os.Unsetenv("USERPROFILE")
				return func() {
					if oldHome != "" {
						os.Setenv("HOME", oldHome)
					}
					if oldUserProfile != "" {
						os.Setenv("USERPROFILE", oldUserProfile)
					}
				}
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
