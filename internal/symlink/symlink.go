package symlink

import (
	"fmt"
	"os"
	"path/filepath"
)

// Create creates a symlink at symlinkPath pointing to target.
// Uses atomic replacement pattern: creates a temp symlink and renames it.
// This avoids TOCTOU race conditions between check and create operations.
func Create(target, symlinkPath string) error {
	dir := filepath.Dir(symlinkPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create symlink directory: %w", err)
	}
	if info, err := os.Lstat(symlinkPath); err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("refusing to replace non-symlink path: %s", symlinkPath)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect existing symlink: %w", err)
	}

	tempFile, err := os.CreateTemp(dir, ".govman-symlink-*")
	if err != nil {
		return fmt.Errorf("failed to reserve temporary symlink path: %w", err)
	}
	tempLink := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		os.Remove(tempLink)
		return fmt.Errorf("failed to close temporary symlink reservation: %w", err)
	}
	if err := os.Remove(tempLink); err != nil {
		return fmt.Errorf("failed to prepare temporary symlink path: %w", err)
	}

	if err := os.Symlink(target, tempLink); err != nil {
		return fmt.Errorf("failed to create temporary symlink: %w", err)
	}

	if err := os.Rename(tempLink, symlinkPath); err != nil {
		os.Remove(tempLink)
		return fmt.Errorf("failed to rename symlink to final location: %w", err)
	}

	return nil
}
