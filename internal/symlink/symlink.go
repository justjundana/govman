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
	// Create a temporary symlink in the same directory
	dir := filepath.Dir(symlinkPath)
	tempLink := filepath.Join(dir, fmt.Sprintf(".govman-symlink-%d", os.Getpid()))

	// Remove any leftover temp symlink from previous failed attempts
	os.Remove(tempLink)

	// Create the symlink at the temporary location
	if err := os.Symlink(target, tempLink); err != nil {
		return fmt.Errorf("failed to create temporary symlink: %w", err)
	}

	// Atomically rename the temp symlink to the final location
	// This replaces any existing symlink in a single operation
	if err := os.Rename(tempLink, symlinkPath); err != nil {
		os.Remove(tempLink) // Cleanup on failure
		return fmt.Errorf("failed to rename symlink to final location: %w", err)
	}

	return nil
}
