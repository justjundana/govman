package symlink

import (
	"os"
	"path/filepath"
)

// ReadLink reads the target of a symlink at symlinkPath.
// It resolves relative targets against the symlink's directory and returns the absolute path or an error.
func ReadLink(symlinkPath string) (string, error) {
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		return "", err
	}

	if !filepath.IsAbs(target) {
		dir := filepath.Dir(symlinkPath)
		target = filepath.Join(dir, target)
	}

	return target, nil
}

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
