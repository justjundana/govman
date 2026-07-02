//go:build !windows

package symlink

import "os"

func replaceSymlink(source, destination string) error {
	return os.Rename(source, destination)
}
