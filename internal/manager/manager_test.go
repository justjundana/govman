package manager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_config "github.com/justjundana/govman/internal/config"
	_downloader "github.com/justjundana/govman/internal/downloader"
	_golang "github.com/justjundana/govman/internal/golang"
)

// mockShell implements Shell interface for testing
type mockShell struct {
	name         string
	displayName  string
	configFile   string
	pathCommand  string
	setupCommand []string
	available    bool
}

func (m *mockShell) Name() string {
	return m.name
}

func (m *mockShell) DisplayName() string {
	return m.displayName
}

func (m *mockShell) ConfigFile() string {
	return m.configFile
}

func (m *mockShell) PathCommand(path string) string {
	return m.pathCommand
}

func (m *mockShell) SetupCommands(binPath string) []string {
	return m.setupCommand
}

func (m *mockShell) IsAvailable() bool {
	return m.available
}

func (m *mockShell) ExecutePathCommand(path string) error {
	fmt.Printf(`export PATH="%s:$PATH"`+"\n", path)
	return nil
}

func createTestConfig(t *testing.T) *_config.Config {
	tempDir := t.TempDir()

	// Mock HOME directory to ensure GetBinPath uses the temporary directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
	})

	// Create config file first
	configFile := filepath.Join(tempDir, "config.yaml")
	config := &_config.Config{
		InstallDir:     filepath.Join(tempDir, "versions"),
		CacheDir:       filepath.Join(tempDir, "cache"),
		DefaultVersion: "",
		GoReleases: _config.GoReleasesConfig{
			APIURL:      "https://api.github.com/repos/golang/go/releases",
			CacheExpiry: 3600,
			DownloadURL: "",
		},
		AutoSwitch: _config.AutoSwitchConfig{
			ProjectFile: filepath.Join(tempDir, ".govman-version"),
		},
	}

	// Create directories
	os.MkdirAll(config.InstallDir, 0755)
	os.MkdirAll(config.CacheDir, 0755)
	os.MkdirAll(config.GetBinPath(), 0755)

	// Create empty config file to enable saving
	os.WriteFile(configFile, []byte(""), 0644)

	return config
}

func createTestManager(t *testing.T, config *_config.Config) *Manager {
	return &Manager{
		config:     config,
		downloader: _downloader.New(config),
		shell: &mockShell{
			name:         "bash",
			displayName:  "Bash",
			configFile:   "~/.bashrc",
			pathCommand:  `export PATH="$1:$PATH"`,
			setupCommand: []string{"# GOVMAN"},
			available:    true,
		},
	}
}

func TestManagerNew(t *testing.T) {
	tests := []struct {
		name           string
		setup          func() *_config.Config
		validateResult func(*testing.T, *Manager)
		wantErr        bool
	}{
		{
			name: "successful initialization",
			setup: func() *_config.Config {
				return createTestConfig(t)
			},
			validateResult: func(t *testing.T, m *Manager) {
				if m == nil {
					t.Fatal("New() returned nil")
				}
				if m.downloader == nil {
					t.Error("Manager downloader not initialized")
				}
				if m.shell == nil {
					t.Error("Manager shell not detected")
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.setup()
			manager := New(config)
			tt.validateResult(t, manager)
		})
	}
}

func TestManager_IsInstalled(t *testing.T) {
	tests := []struct {
		name    string
		version string
		setup   func(*_config.Config)
		want    bool
		wantErr bool
	}{
		{
			name:    "version not installed",
			version: "1.20.0",
			setup:   func(c *_config.Config) {},
			want:    false,
			wantErr: false,
		},
		{
			name:    "version installed",
			version: "1.20.0",
			setup: func(c *_config.Config) {
				versionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(versionDir, 0755)
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			// Clean up any existing directories
			versionDir := config.GetVersionDir(tt.version)
			os.RemoveAll(versionDir)

			tt.setup(config)

			got := manager.IsInstalled(tt.version)
			if got != tt.want {
				t.Errorf("IsInstalled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_ListInstalled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*_config.Config)
		want    []string
		wantErr bool
	}{
		{
			name:    "no versions installed",
			setup:   func(c *_config.Config) {},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "multiple versions installed",
			setup: func(c *_config.Config) {
				versions := []string{"1.19.0", "1.20.0", "1.18.0"}
				for _, version := range versions {
					versionDir := c.GetVersionDir(version)
					os.MkdirAll(versionDir, 0755)
				}
			},
			want:    []string{"1.20.0", "1.19.0", "1.18.0"}, // Should be sorted descending
			wantErr: false,
		},
		{
			name: "versions installed out of order",
			setup: func(c *_config.Config) {
				versions := []string{"1.19.0", "1.21.0", "1.20.0"}
				for _, v := range versions {
					os.MkdirAll(c.GetVersionDir(v), 0755)
				}
			},
			want:    []string{"1.21.0", "1.20.0", "1.19.0"},
			wantErr: false,
		},
		{
			name: "install directory read error",
			setup: func(c *_config.Config) {
				// Create install dir without read permissions
				os.Chmod(c.InstallDir, 0000)
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "mixed directories and files",
			setup: func(c *_config.Config) {
				// Create a directory that starts with "go"
				os.MkdirAll(filepath.Join(c.InstallDir, "go1.20.0"), 0755)
				// Create a file that starts with "go"
				os.WriteFile(filepath.Join(c.InstallDir, "go.mod"), []byte("test"), 0644)
				// Create a directory that doesn't start with "go"
				os.MkdirAll(filepath.Join(c.InstallDir, "cache"), 0755)
			},
			want:    []string{"1.20.0"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			// Clean up install directory
			os.RemoveAll(config.InstallDir)
			os.MkdirAll(config.InstallDir, 0755)

			tt.setup(config)

			// Cleanup permissions after test
			t.Cleanup(func() {
				os.Chmod(config.InstallDir, 0755)
			})

			got, err := manager.ListInstalled()
			if (err != nil) != tt.wantErr {
				t.Errorf("ListInstalled() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("ListInstalled() = %v, want %v", got, tt.want)
				return
			}

			for i, want := range tt.want {
				if i >= len(got) || got[i] != want {
					t.Errorf("ListInstalled() version at index %d = %s, want %s", i, got[i], want)
				}
			}
		})
	}
}

func TestManager_Current(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*_config.Config)
		want    string
		wantErr bool
	}{
		{
			name: "no active version - uses system go",
			setup: func(c *_config.Config) {
				// No setup needed, uses system Go
			},
			want:    "SYSTEM_GO", // Will be resolved dynamically
			wantErr: false,
		},
		{
			name: "global version active",
			setup: func(c *_config.Config) {
				version := "1.20.0"
				versionDir := c.GetVersionDir(version)
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)

				// Create symlink
				symlinkPath := c.GetCurrentSymlink()
				targetPath := filepath.Join(versionDir, "bin", "go")
				os.Symlink(targetPath, symlinkPath)

				// Create a go binary that reports the correct version
				goPath := filepath.Join(versionDir, "bin", "go")
				os.WriteFile(goPath, []byte("#!/bin/bash\necho 'go version go1.20.0 darwin/arm64'"), 0755)

				// Temporarily replace PATH to use the test go binary
				os.Setenv("PATH", filepath.Join(versionDir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
			},
			want:    "1.20.0",
			wantErr: false,
		},
		{
			name: "local version specified",
			setup: func(c *_config.Config) {
				version := "1.19.0"
				versionDir := c.GetVersionDir(version)
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)

				// Create a go binary that reports the correct version
				goPath := filepath.Join(versionDir, "bin", "go")
				os.WriteFile(goPath, []byte("#!/bin/bash\necho 'go version go1.19.0 darwin/arm64'"), 0755)

				// Temporarily replace PATH to use the test go binary
				os.Setenv("PATH", filepath.Join(versionDir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))

				// Write local version file
				os.WriteFile(c.AutoSwitch.ProjectFile, []byte(version), 0644)
			},
			want:    "1.19.0",
			wantErr: false,
		},
		{
			name: "session version check fails",
			setup: func(c *_config.Config) {
				// Set PATH to non-existent directory so go command fails
				os.Setenv("PATH", "/nonexistent/path")
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "local version not installed",
			setup: func(c *_config.Config) {
				version := "1.19.0"
				// Write local version file but don't install the version
				os.WriteFile(c.AutoSwitch.ProjectFile, []byte(version), 0644)
				// Set PATH to non-existent directory so session check fails
				os.Setenv("PATH", "/nonexistent/path")
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "session version not managed by govman",
			setup: func(c *_config.Config) {
				// Create a fake go that returns a version not in GOVMAN
				binDir := filepath.Join(c.GetBinPath(), "systemgo")
				os.MkdirAll(binDir, 0755)
				goPath := filepath.Join(binDir, "go")
				os.WriteFile(goPath, []byte("#!/bin/bash\necho 'go version go1.99.0 darwin/arm64'"), 0755)
				os.Chmod(goPath, 0755)
				os.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			},
			want:    "1.99.0",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			// Clean up
			os.RemoveAll(config.InstallDir)
			os.RemoveAll(config.GetBinPath())
			os.Remove(config.AutoSwitch.ProjectFile)
			os.MkdirAll(config.InstallDir, 0755)
			os.MkdirAll(config.GetBinPath(), 0755)

			tt.setup(config)

			got, err := manager.Current()
			if (err != nil) != tt.wantErr {
				t.Errorf("Current() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			want := tt.want
			if want == "SYSTEM_GO" {
				// Get system Go version dynamically
				out, err := exec.Command("go", "version").Output()
				if err == nil {
					parts := strings.Fields(string(out))
					if len(parts) >= 3 {
						want = strings.TrimPrefix(parts[2], "go")
					}
				}
			}

			if got != want {
				t.Errorf("Current() = %v, want %v", got, want)
			}
		})
	}
}

func TestManager_CurrentGlobal(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*_config.Config)
		want    string
		wantErr bool
	}{
		{
			name:    "no symlink exists",
			setup:   func(c *_config.Config) {},
			want:    "",
			wantErr: true,
		},
		{
			name: "valid symlink",
			setup: func(c *_config.Config) {
				version := "1.20.0"
				versionDir := c.GetVersionDir(version)
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)

				// Create go executable
				goPath := filepath.Join(versionDir, "bin", "go")
				if runtime.GOOS == "windows" {
					goPath += ".exe"
				}
				os.WriteFile(goPath, []byte("#!/bin/bash\necho 'go version go1.20.0'"), 0755)

				// Create symlink
				symlinkPath := c.GetCurrentSymlink()
				if runtime.GOOS == "windows" {
					symlinkPath += ".exe"
				}
				targetPath := filepath.Join(versionDir, "bin", "go")
				if runtime.GOOS == "windows" {
					targetPath += ".exe"
				}
				os.Symlink(targetPath, symlinkPath)
			},
			want:    "1.20.0",
			wantErr: false,
		},
		{
			name: "symlink points to non-existent version",
			setup: func(c *_config.Config) {
				version := "1.20.0"
				versionDir := c.GetVersionDir(version)
				targetPath := filepath.Join(versionDir, "bin", "go")

				// Create symlink but don't create the version directory
				symlinkPath := c.GetCurrentSymlink()
				os.Symlink(targetPath, symlinkPath)
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "symlink is not a symlink",
			setup: func(c *_config.Config) {
				symlinkPath := c.GetCurrentSymlink()
				// Create a regular file instead of a symlink
				os.WriteFile(symlinkPath, []byte("not a symlink"), 0644)
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "symlink target format invalid",
			setup: func(c *_config.Config) {
				// Create symlink pointing to invalid path
				symlinkPath := c.GetCurrentSymlink()
				os.Symlink("/invalid/path/go", symlinkPath)
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "go executable missing",
			setup: func(c *_config.Config) {
				version := "1.20.0"
				versionDir := c.GetVersionDir(version)
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)

				// Create symlink but don't create the go executable
				symlinkPath := c.GetCurrentSymlink()
				targetPath := filepath.Join(versionDir, "bin", "go")
				os.Symlink(targetPath, symlinkPath)
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "no symlink and no default version",
			setup: func(c *_config.Config) {
				// Ensure no symlink exists
				os.Remove(c.GetCurrentSymlink())
				c.DefaultVersion = ""
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "default version not installed",
			setup: func(c *_config.Config) {
				// Set default version but do not install it
				c.DefaultVersion = "1.20.0"
				os.Remove(c.GetCurrentSymlink())
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "default version installed but symlink missing",
			setup: func(c *_config.Config) {
				version := "1.20.0"
				c.DefaultVersion = version
				versionDir := c.GetVersionDir(version)
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)
				os.Remove(c.GetCurrentSymlink())
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			// Clean up
			os.RemoveAll(config.InstallDir)
			os.RemoveAll(config.GetBinPath())
			os.MkdirAll(config.InstallDir, 0755)
			os.MkdirAll(config.GetBinPath(), 0755)

			tt.setup(config)

			got, err := manager.CurrentGlobal()
			if (err != nil) != tt.wantErr {
				t.Errorf("CurrentGlobal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("CurrentGlobal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_Use(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		setDefault bool
		setLocal   bool
		setup      func(*_config.Config)
		wantErr    bool
	}{
		{
			name:       "use for session only",
			version:    "1.20.0",
			setDefault: false,
			setLocal:   false,
			setup: func(c *_config.Config) {
				// Install version first
				versionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)
			},
			wantErr: false,
		},
		{
			name:       "set as default",
			version:    "1.20.0",
			setDefault: true,
			setLocal:   false,
			setup: func(c *_config.Config) {
				// Install version first
				versionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)
			},
			wantErr: false,
		},
		{
			name:       "set local version",
			version:    "1.20.0",
			setDefault: false,
			setLocal:   true,
			setup: func(c *_config.Config) {
				// Install version first
				versionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)
			},
			wantErr: false,
		},
		{
			name:       "use non-installed version",
			version:    "1.19.0",
			setDefault: false,
			setLocal:   false,
			setup:      func(c *_config.Config) {},
			wantErr:    true,
		},
		{
			name:       "use default when CurrentGlobal fails",
			version:    "default",
			setDefault: false,
			setLocal:   false,
			setup:      func(c *_config.Config) {},
			wantErr:    true,
		},
		{
			name:       "set local version with write failure",
			version:    "1.20.0",
			setDefault: false,
			setLocal:   true,
			setup: func(c *_config.Config) {
				// Install version first
				versionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)

				// Make directory read-only to cause write failure
				projectDir := filepath.Dir(c.AutoSwitch.ProjectFile)
				os.Chmod(projectDir, 0444)
			},
			wantErr: true,
		},
		{
			name:       "set as default with createSymlink failure",
			version:    "1.20.0",
			setDefault: true,
			setLocal:   false,
			setup: func(c *_config.Config) {
				// Install version first
				versionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)

				// Make bin directory read-only to cause symlink creation failure
				os.Chmod(c.GetBinPath(), 0444)
			},
			wantErr: true,
		},
		{
			name:       "use default that is installed",
			version:    "default",
			setDefault: false,
			setLocal:   false,
			setup: func(c *_config.Config) {
				c.DefaultVersion = "1.20.0"
				versionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)

				// Create go executable
				goPath := filepath.Join(versionDir, "bin", "go")
				os.WriteFile(goPath, []byte("#!/bin/bash\necho 'go version go1.20.0'"), 0755)

				// Create symlink so CurrentGlobal() works
				symlinkPath := c.GetCurrentSymlink()
				targetPath := filepath.Join(versionDir, "bin", "go")
				os.Symlink(targetPath, symlinkPath)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			// Clean up
			os.Remove(config.AutoSwitch.ProjectFile)

			tt.setup(config)

			// Cleanup permissions after test
			t.Cleanup(func() {
				os.Chmod(filepath.Dir(config.AutoSwitch.ProjectFile), 0755)
				os.Chmod(config.GetBinPath(), 0755)
			})

			err := manager.Use(tt.version, tt.setDefault, tt.setLocal)
			if (err != nil) != tt.wantErr {
				t.Errorf("Use() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify local version was set if requested and no error
			if tt.setLocal && !tt.wantErr {
				localVersion := manager.getLocalVersion()
				if localVersion != tt.version {
					t.Errorf("Local version = %v, want %v", localVersion, tt.version)
				}
			}
		})
	}
}

func TestManager_Install(t *testing.T) {
	tests := []struct {
		name    string
		version string
		setup   func(*_config.Config)
		wantErr bool
	}{
		{
			name:    "install new version (network failure expected)",
			version: "1.20.0",
			setup:   func(c *_config.Config) {},
			wantErr: true, // Will fail because no actual download URL available in test
		},
		{
			name:    "install already installed version",
			version: "1.20.0",
			setup: func(c *_config.Config) {
				versionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(versionDir, 0755)
			},
			wantErr: true,
		},
		{
			name:    "install with resolveVersion failure",
			version: "latest",
			setup: func(c *_config.Config) {
				c.GoReleases.APIURL = "invalid://url"
			},
			wantErr: true,
		},
		{
			name:    "install with GetDownloadURL failure",
			version: "1.19.0",
			setup:   func(c *_config.Config) {},
			wantErr: true,
		},
		{
			name:    "install exact version that is not already installed",
			version: "1.21.0",
			setup:   func(c *_config.Config) {},
			wantErr: true, // Fails due to no download URL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			tt.setup(config)

			err := manager.Install(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("Install() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_Uninstall(t *testing.T) {
	tests := []struct {
		name    string
		version string
		setup   func(*_config.Config)
		wantErr bool
	}{
		{
			name:    "uninstall installed version",
			version: "1.20.0",
			setup: func(c *_config.Config) {
				versionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(versionDir, 0755)
			},
			wantErr: false,
		},
		{
			name:    "uninstall non-installed version",
			version: "1.20.0",
			setup:   func(c *_config.Config) {},
			wantErr: true,
		},
		{
			name:    "uninstall currently active version",
			version: "1.20.0",
			setup: func(c *_config.Config) {
				versionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)

				// Create symlink to make it current
				symlinkPath := c.GetCurrentSymlink()
				targetPath := filepath.Join(versionDir, "bin", "go")
				os.Symlink(targetPath, symlinkPath)

				// Create go binary
				goPath := filepath.Join(versionDir, "bin", "go")
				os.WriteFile(goPath, []byte("#!/bin/bash\necho 'go version go1.20.0 darwin/arm64'"), 0755)

				// Set as current by mocking go command path
				os.Setenv("PATH", filepath.Join(versionDir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			// Clean up
			versionDir := config.GetVersionDir(tt.version)
			os.RemoveAll(versionDir)

			tt.setup(config)

			err := manager.Uninstall(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("Uninstall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_Clean(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*_config.Config)
		wantErr bool
	}{
		{
			name: "clean cache successfully",
			setup: func(c *_config.Config) {
				// Create some files in cache
				cacheFile := filepath.Join(c.CacheDir, "test.txt")
				os.WriteFile(cacheFile, []byte("test"), 0644)
			},
			wantErr: false,
		},
		{
			name: "clean cache with recreation failure",
			setup: func(c *_config.Config) {
				// Make parent directory of cache read-only
				parentDir := filepath.Dir(c.CacheDir)
				os.Chmod(parentDir, 0444)
			},
			wantErr: true,
		},
		{
			name: "clean empty cache directory",
			setup: func(c *_config.Config) {
				// Cache directory already exists but is empty
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			// Clean up
			os.RemoveAll(config.CacheDir)
			parentDir := filepath.Dir(config.CacheDir)
			os.MkdirAll(parentDir, 0755)
			os.MkdirAll(config.CacheDir, 0755)

			tt.setup(config)

			// Cleanup permissions after test
			t.Cleanup(func() {
				os.Chmod(parentDir, 0755)
				os.Chmod(config.CacheDir, 0755)
			})

			err := manager.Clean()
			if (err != nil) != tt.wantErr {
				t.Errorf("Clean() error = %v, wantErr %v", err, tt.wantErr)
			}

			// For success case, verify cache directory exists and is empty
			if !tt.wantErr {
				if _, err := os.Stat(config.CacheDir); os.IsNotExist(err) {
					t.Error("Cache directory was not recreated")
				}
			}
		})
	}
}

func TestManager_DefaultVersion(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *_config.Config
		want    string
		wantErr bool
	}{
		{
			name: "get default version",
			setup: func() *_config.Config {
				config := createTestConfig(t)
				config.DefaultVersion = "1.20.0"
				return config
			},
			want:    "1.20.0",
			wantErr: false,
		},
		{
			name: "no default version set",
			setup: func() *_config.Config {
				config := createTestConfig(t)
				config.DefaultVersion = ""
				return config
			},
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.setup()
			manager := createTestManager(t, config)

			got := manager.DefaultVersion()
			if got != tt.want {
				t.Errorf("DefaultVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_setLocalVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		setup   func(*_config.Config)
		wantErr bool
	}{
		{
			name:    "set local version successfully",
			version: "1.20.0",
			setup:   func(c *_config.Config) {},
			wantErr: false,
		},
		{
			name:    "set local version with write permission failure",
			version: "1.20.0",
			setup: func(c *_config.Config) {
				// Make directory read-only
				projectDir := filepath.Dir(c.AutoSwitch.ProjectFile)
				os.Chmod(projectDir, 0444)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			tt.setup(config)

			// Cleanup permissions after test
			t.Cleanup(func() {
				os.Chmod(filepath.Dir(config.AutoSwitch.ProjectFile), 0755)
			})

			err := manager.setLocalVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("setLocalVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check that file was created with correct content
			if !tt.wantErr {
				data, err := os.ReadFile(config.AutoSwitch.ProjectFile)
				if err != nil {
					t.Errorf("Failed to read local version file: %v", err)
				}

				if strings.TrimSpace(string(data)) != tt.version {
					t.Errorf("File content = %s, want %s", string(data), tt.version)
				}
			}
		})
	}
}

func TestManager_getLocalVersion(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*_config.Config)
		want    string
		wantErr bool
	}{
		{
			name:    "no local version file",
			setup:   func(c *_config.Config) {},
			want:    "",
			wantErr: false,
		},
		{
			name: "local version file exists",
			setup: func(c *_config.Config) {
				version := "1.19.0"
				os.WriteFile(c.AutoSwitch.ProjectFile, []byte(version), 0644)
			},
			want:    "1.19.0",
			wantErr: false,
		},
		{
			name: "local version file with whitespace",
			setup: func(c *_config.Config) {
				version := "  1.19.0  \n"
				os.WriteFile(c.AutoSwitch.ProjectFile, []byte(version), 0644)
			},
			want:    "1.19.0",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			// Clean up
			os.Remove(config.AutoSwitch.ProjectFile)

			tt.setup(config)

			got := manager.getLocalVersion()
			if got != tt.want {
				t.Errorf("getLocalVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_CurrentActivationMethod(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*_config.Config)
		want    string
		wantErr bool
	}{
		{
			name: "no active version",
			setup: func(c *_config.Config) {
				// Set PATH to non-existent so session check fails
				os.Setenv("PATH", "/nonexistent/path")
			},
			want:    "system-default",
			wantErr: false,
		},
		{
			name: "local version set",
			setup: func(c *_config.Config) {
				os.WriteFile(c.AutoSwitch.ProjectFile, []byte("1.20.0"), 0644)
				// Set PATH to include a fake go binary that will return an error, so session check fails
				os.Setenv("PATH", "/nonexistent/path")
			},
			want:    "project-local",
			wantErr: false,
		},
		{
			name: "system default active",
			setup: func(c *_config.Config) {
				version := "1.20.0"
				versionDir := c.GetVersionDir(version)
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)

				// Create symlink
				symlinkPath := c.GetCurrentSymlink()
				targetPath := filepath.Join(versionDir, "bin", "go")
				os.Symlink(targetPath, symlinkPath)

				// Create go binary
				goPath := filepath.Join(versionDir, "bin", "go")
				os.WriteFile(goPath, []byte("#!/bin/bash\necho 'go version go1.20.0 darwin/arm64'"), 0755)

				// Temporarily replace PATH to make this version active
				os.Setenv("PATH", filepath.Join(versionDir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
			},
			want:    "system-default",
			wantErr: false,
		},
		{
			name: "session-only version active",
			setup: func(c *_config.Config) {
				version := "1.20.0"
				versionDir := c.GetVersionDir(version)
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)

				// Create go binary
				goPath := filepath.Join(versionDir, "bin", "go")
				os.WriteFile(goPath, []byte("#!/bin/bash\necho 'go version go1.20.0 darwin/arm64'"), 0755)

				// Set PATH to include this version but don't create symlink
				os.Setenv("PATH", filepath.Join(versionDir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
			},
			want:    "session-only",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			// Clean up
			os.RemoveAll(config.InstallDir)
			os.RemoveAll(config.GetBinPath())
			os.Remove(config.AutoSwitch.ProjectFile)
			os.MkdirAll(config.InstallDir, 0755)
			os.MkdirAll(config.GetBinPath(), 0755)

			tt.setup(config)

			got := manager.CurrentActivationMethod()
			if got != tt.want {
				t.Errorf("CurrentActivationMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_Info(t *testing.T) {
	tests := []struct {
		name    string
		version string
		setup   func(*_config.Config)
		wantErr bool
	}{
		{
			name:    "version not installed",
			version: "1.20.0",
			setup:   func(c *_config.Config) {},
			wantErr: true,
		},
		{
			name:    "version installed",
			version: "1.20.0",
			setup: func(c *_config.Config) {
				versionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)
				// Create a mock go binary
				goPath := filepath.Join(versionDir, "bin", "go")
				os.WriteFile(goPath, []byte("#!/bin/bash\necho 'go version go1.20.0 darwin/arm64'"), 0755)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			tt.setup(config)

			got, err := manager.Info(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("Info() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got == nil {
				t.Error("Expected VersionInfo but got nil")
			}
		})
	}
}

func TestManager_ListRemote(t *testing.T) {
	tests := []struct {
		name       string
		forceCheck bool
		setup      func(*_config.Config)
		wantErr    bool
	}{
		{
			name:       "list remote with invalid API URL",
			forceCheck: false,
			setup: func(c *_config.Config) {
				c.GoReleases.APIURL = "invalid://url"
			},
			wantErr: true,
		},
		{
			name:       "list remote with force check and invalid URL",
			forceCheck: true,
			setup: func(c *_config.Config) {
				c.GoReleases.APIURL = "invalid://url"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			tt.setup(config)

			_, err := manager.ListRemote(tt.forceCheck)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListRemote() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_ResolveVersion(t *testing.T) {
	_golang.ClearReleasesCache()

	tests := []struct {
		name    string
		input   string
		setup   func(*_config.Config)
		want    string
		wantErr bool
	}{
		{
			name:    "resolve exact version",
			input:   "1.20.0",
			setup:   func(c *_config.Config) {},
			want:    "1.20.0",
			wantErr: false,
		},
		{
			name:  "resolve latest with API failure",
			input: "latest",
			setup: func(c *_config.Config) {
				c.GoReleases.APIURL = "invalid://url"
			},
			want:    "",
			wantErr: true,
		},
		{
			name:  "resolve partial version with API failure",
			input: "1.20",
			setup: func(c *_config.Config) {
				c.GoReleases.APIURL = "invalid://url"
			},
			want:    "",
			wantErr: true,
		},
		{
			name:    "resolve version with no dots",
			input:   "1",
			setup:   func(c *_config.Config) {},
			want:    "1",
			wantErr: false,
		},
		{
			name:  "resolve version with one dot that doesn't match",
			input: "99.99",
			setup: func(c *_config.Config) {
				c.GoReleases.APIURL = "invalid://url"
			},
			want:    "",
			wantErr: true,
		},
		{
			name:  "resolve latest with empty versions list",
			input: "latest",
			setup: func(c *_config.Config) {
				c.GoReleases.APIURL = "invalid://url"
			},
			want:    "",
			wantErr: true,
		},
		{
			name:    "resolve version with three dots",
			input:   "1.20.5",
			setup:   func(c *_config.Config) {},
			want:    "1.20.5",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			tt.setup(config)

			got, err := manager.ResolveVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want && tt.want != "" {
				t.Errorf("ResolveVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_createSymlink(t *testing.T) {
	tests := []struct {
		name    string
		version string
		setup   func(*_config.Config)
		wantErr bool
	}{
		{
			name:    "create symlink successfully",
			version: "1.20.0",
			setup: func(c *_config.Config) {
				versionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)
			},
			wantErr: false,
		},
		{
			name:    "create symlink with bin directory creation failure",
			version: "1.20.0",
			setup: func(c *_config.Config) {
				versionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)

				// Make parent directory read-only
				parentDir := filepath.Dir(c.GetBinPath())
				os.Chmod(parentDir, 0444)
			},
			wantErr: true,
		},
		{
			name:    "create symlink with symlink creation failure",
			version: "1.20.0",
			setup: func(c *_config.Config) {
				versionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)

				// Make bin directory read-only
				os.Chmod(c.GetBinPath(), 0444)
			},
			wantErr: true,
		},
		{
			name:    "create symlink with non-empty directory at symlink location",
			version: "1.20.0",
			setup: func(c *_config.Config) {
				versionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(filepath.Join(versionDir, "bin"), 0755)

				// Create a non-empty directory at the symlink location
				symlinkPath := c.GetCurrentSymlink()
				os.MkdirAll(symlinkPath, 0755)
				// Add a file inside to make removal fail (non-empty dir)
				os.WriteFile(filepath.Join(symlinkPath, "file.txt"), []byte("data"), 0644)
			},
			wantErr: true,
		},
		{
			name:    "replace existing symlink",
			version: "1.20.0",
			setup: func(c *_config.Config) {
				// Create old symlink
				oldVersion := "1.19.0"
				oldVersionDir := c.GetVersionDir(oldVersion)
				os.MkdirAll(filepath.Join(oldVersionDir, "bin"), 0755)
				symlinkPath := c.GetCurrentSymlink()
				oldTargetPath := filepath.Join(oldVersionDir, "bin", "go")
				os.Symlink(oldTargetPath, symlinkPath)

				// Create new version
				newVersionDir := c.GetVersionDir("1.20.0")
				os.MkdirAll(filepath.Join(newVersionDir, "bin"), 0755)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			// Clean up
			os.RemoveAll(config.GetBinPath())
			os.MkdirAll(config.GetBinPath(), 0755)

			tt.setup(config)

			// Cleanup permissions after test
			t.Cleanup(func() {
				parentDir := filepath.Dir(config.GetBinPath())
				os.Chmod(parentDir, 0755)
				os.Chmod(config.GetBinPath(), 0755)
				os.RemoveAll(config.GetCurrentSymlink())
			})

			err := manager.createSymlink(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("createSymlink() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_getCurrentSessionVersion(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*_config.Config)
		want    string
		wantErr bool
	}{
		{
			name: "get current session version successfully",
			setup: func(c *_config.Config) {
				// Create a fake go binary that outputs a version string
				binDir := filepath.Join(c.GetBinPath(), "fakego")
				os.MkdirAll(binDir, 0755)
				goPath := filepath.Join(binDir, "go")
				os.WriteFile(goPath, []byte("#!/bin/bash\necho 'go version go1.21.0 linux/amd64'"), 0755)
				// Prepend fake bin directory to PATH
				originalPath := os.Getenv("PATH")
				os.Setenv("PATH", binDir+string(os.PathListSeparator)+originalPath)
			},
			want:    "1.21.0",
			wantErr: false,
		},
		{
			name: "get current session version with error",
			setup: func(c *_config.Config) {
				// Set PATH to a non-existent directory
				os.Setenv("PATH", "/nonexistent/path")
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "get current session version with invalid output format",
			setup: func(c *_config.Config) {
				// Create a fake go binary that outputs invalid format
				binDir := filepath.Join(c.GetBinPath(), "fakego")
				os.MkdirAll(binDir, 0755)
				goPath := filepath.Join(binDir, "go")
				os.WriteFile(goPath, []byte("#!/bin/bash\necho 'invalid output'"), 0755)
				os.Chmod(goPath, 0755)
				originalPath := os.Getenv("PATH")
				os.Setenv("PATH", binDir+string(os.PathListSeparator)+originalPath)
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "get current session version with empty version string",
			setup: func(c *_config.Config) {
				// Create a fake go binary that outputs version without 'go' prefix
				binDir := filepath.Join(c.GetBinPath(), "fakego")
				os.MkdirAll(binDir, 0755)
				goPath := filepath.Join(binDir, "go")
				os.WriteFile(goPath, []byte("#!/bin/bash\necho 'go version  linux/amd64'"), 0755)
				os.Chmod(goPath, 0755)
				originalPath := os.Getenv("PATH")
				os.Setenv("PATH", binDir+string(os.PathListSeparator)+originalPath)
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(t)
			manager := createTestManager(t, config)

			// Save original PATH
			originalPath := os.Getenv("PATH")
			t.Cleanup(func() {
				os.Setenv("PATH", originalPath)
			})

			tt.setup(config)

			got, err := manager.getCurrentSessionVersion()
			if (err != nil) != tt.wantErr {
				t.Errorf("getCurrentSessionVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("getCurrentSessionVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
