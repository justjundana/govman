package cli

import (
	"testing"
)

func TestIsPrerelease(t *testing.T) {
	testCases := []struct {
		version  string
		expected bool
	}{
		{"1.25.1", false},
		{"1.25.0", false},
		{"1.25", false},
		{"1.25rc1", true},
		{"1.25beta1", true},
		{"1.25alpha1", true},
		{"1.25rc2", true},
		{"1.25beta2", true},
	}

	for _, tc := range testCases {
		t.Run(tc.version, func(t *testing.T) {
			result := isPrerelease(tc.version)
			if result != tc.expected {
				t.Errorf("isPrerelease(%q) = %v, want %v", tc.version, result, tc.expected)
			}
		})
	}
}
