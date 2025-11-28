package util

import (
	"testing"
)

func TestExtractMajorMinor(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "full version with patch",
			version:  "1.25.4",
			expected: "1.25",
		},
		{
			name:     "major.minor only",
			version:  "1.25",
			expected: "1.25",
		},
		{
			name:     "major only",
			version:  "1",
			expected: "1",
		},
		{
			name:     "version with multiple dots",
			version:  "1.25.4.1",
			expected: "1.25",
		},
		{
			name:     "empty string",
			version:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractMajorMinor(tt.version)
			if result != tt.expected {
				t.Errorf("ExtractMajorMinor(%q) = %q, want %q", tt.version, result, tt.expected)
			}
		})
	}
}

func TestFindBestMatchingVersion(t *testing.T) {
	tests := []struct {
		name              string
		requestedVersion  string
		installedVersions []string
		expectedVersion   string
		expectError       bool
	}{
		{
			name:              "exact match available",
			requestedVersion:  "1.25.4",
			installedVersions: []string{"1.25.4", "1.24.3", "1.26.0"},
			expectedVersion:   "1.25.4",
			expectError:       false,
		},
		{
			name:              "partial version matches multiple, returns latest",
			requestedVersion:  "1.25",
			installedVersions: []string{"1.25.1", "1.25.4", "1.25.9", "1.24.5"},
			expectedVersion:   "1.25.9",
			expectError:       false,
		},
		{
			name:              "full version matches different patch",
			requestedVersion:  "1.25.4",
			installedVersions: []string{"1.25.1", "1.24.3"},
			expectedVersion:   "1.25.1",
			expectError:       false,
		},
		{
			name:              "no matching version",
			requestedVersion:  "1.25",
			installedVersions: []string{"1.24.5", "1.26.0"},
			expectedVersion:   "",
			expectError:       true,
		},
		{
			name:              "empty installed versions",
			requestedVersion:  "1.25",
			installedVersions: []string{},
			expectedVersion:   "",
			expectError:       true,
		},
		{
			name:              "single matching version",
			requestedVersion:  "1.25",
			installedVersions: []string{"1.25.1"},
			expectedVersion:   "1.25.1",
			expectError:       false,
		},
		{
			name:              "multiple versions, picks highest",
			requestedVersion:  "1.25.2",
			installedVersions: []string{"1.25.9", "1.25.1", "1.25.4"},
			expectedVersion:   "1.25.9",
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FindBestMatchingVersion(tt.requestedVersion, tt.installedVersions)

			if tt.expectError {
				if err == nil {
					t.Errorf("FindBestMatchingVersion(%q, %v) expected error but got none", tt.requestedVersion, tt.installedVersions)
				}
				return
			}

			if err != nil {
				t.Errorf("FindBestMatchingVersion(%q, %v) unexpected error: %v", tt.requestedVersion, tt.installedVersions, err)
				return
			}

			if result != tt.expectedVersion {
				t.Errorf("FindBestMatchingVersion(%q, %v) = %q, want %q", tt.requestedVersion, tt.installedVersions, result, tt.expectedVersion)
			}
		})
	}
}
