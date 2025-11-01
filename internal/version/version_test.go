package version

import (
	"testing"
)

func setTestValues(v, c, d, b string) {
	Version = v
	Commit = c
	Date = d
	BuildBy = b
}

func TestGet(t *testing.T) {
	testCases := []struct {
		name    string
		version string
		commit  string
		date    string
		buildBy string
	}{
		{
			name:    "Default values",
			version: "dev",
			commit:  "none",
			date:    "unknown",
			buildBy: "unknown",
		},
		{
			name:    "Custom values",
			version: "v1.2.3",
			commit:  "abc123",
			date:    "2025-09-13",
			buildBy: "tester",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setTestValues(tc.version, tc.commit, tc.date, tc.buildBy)

			info := Get()

			if info.Version != tc.version {
				t.Errorf("expected Version: %s, got: %s", tc.version, info.Version)
			}
			if info.Commit != tc.commit {
				t.Errorf("expected Commit: %s, got: %s", tc.commit, info.Commit)
			}
			if info.Date != tc.date {
				t.Errorf("expected Date: %s, got: %s", tc.date, info.Date)
			}
			if info.BuildBy != tc.buildBy {
				t.Errorf("expected BuildBy: %s, got: %s", tc.buildBy, info.BuildBy)
			}
			if info.GoVersion == "" {
				t.Errorf("expected GoVersion to be non-empty")
			}
			if info.Platform == "" {
				t.Errorf("expected Platform to be non-empty")
			}
		})
	}
}

func TestBuildVersion(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		commit   string
		expected string
	}{
		{
			name:     "Dev version with commit",
			version:  "dev",
			commit:   "abc123",
			expected: "dev-abc123",
		},
		{
			name:     "Release version",
			version:  "v1.2.3",
			commit:   "ignored",
			expected: "v1.2.3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setTestValues(tc.version, tc.commit, "", "")

			got := BuildVersion()
			if got != tc.expected {
				t.Errorf("expected BuildVersion: %s, got: %s", tc.expected, got)
			}
		})
	}
}
