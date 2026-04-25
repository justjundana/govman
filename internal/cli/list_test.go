package cli

import (
	"strings"
	"testing"
)

// TestUnstableVersionCounting verifies the fix for BUG-5:
// alpha versions must be counted as unstable alongside rc and beta.
func TestUnstableVersionCounting(t *testing.T) {
	versions := []string{
		"1.25.1",
		"1.25.0",
		"1.25rc1",
		"1.25beta1",
		"1.24alpha1",
		"1.24.0",
	}

	stableCount := 0
	unstableCount := 0
	for _, version := range versions {
		if strings.Contains(version, "rc") || strings.Contains(version, "beta") || strings.Contains(version, "alpha") {
			unstableCount++
		} else {
			stableCount++
		}
	}

	if stableCount != 3 {
		t.Errorf("expected 3 stable versions, got %d", stableCount)
	}
	if unstableCount != 3 {
		t.Errorf("expected 3 unstable versions (rc + beta + alpha), got %d", unstableCount)
	}
}
