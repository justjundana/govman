package symlink

import (
	"os"
)

// Create creates a symlink at symlinkPath pointing to target.
// If a path already exists at symlinkPath, it removes it first, then creates the new symlink.
func Create(target, symlinkPath string) error {
	if _, err := os.Lstat(symlinkPath); err == nil {
		if err := os.Remove(symlinkPath); err != nil {
			return err
		}
	}

	return os.Symlink(target, symlinkPath)
}
