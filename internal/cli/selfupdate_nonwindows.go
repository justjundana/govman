//go:build !windows

package cli

import "fmt"

func startWindowsUpdateHelper(_ string, _ []string) error {
	return fmt.Errorf("windows update helper is unavailable on this platform")
}
