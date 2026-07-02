//go:build !windows

package cli

import "fmt"

func startWindowsUpdateHelper(_ string, _ []string) error {
	return fmt.Errorf("Windows update helper is unavailable on this platform")
}
