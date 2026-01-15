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

func TestIsWildcardPattern(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{
			name:     "pattern with asterisk at end",
			version:  "1.14.*",
			expected: true,
		},
		{
			name:     "pattern with asterisk in middle",
			version:  "1.*.0",
			expected: true,
		},
		{
			name:     "just asterisk",
			version:  "*",
			expected: true,
		},
		{
			name:     "normal version",
			version:  "1.14.0",
			expected: false,
		},
		{
			name:     "partial version",
			version:  "1.14",
			expected: false,
		},
		{
			name:     "empty string",
			version:  "",
			expected: false,
		},
		{
			name:     "latest keyword",
			version:  "latest",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWildcardPattern(tt.version)
			if result != tt.expected {
				t.Errorf("IsWildcardPattern(%q) = %v, want %v", tt.version, result, tt.expected)
			}
		})
	}
}

func TestMatchVersionPattern(t *testing.T) {
	versions := []string{"1.14", "1.14.0", "1.14.1", "1.14.2", "1.14rc1", "1.14beta1", "1.15", "1.15.0", "1.3", "1.3.1", "1.3.2", "1.3.3"}

	tests := []struct {
		name            string
		pattern         string
		versions        []string
		expectedMatches []string
	}{
		{
			name:            "match 1.14.* pattern includes prerelease",
			pattern:         "1.14.*",
			versions:        versions,
			expectedMatches: []string{"1.14.2", "1.14.1", "1.14", "1.14.0", "1.14rc1", "1.14beta1"},
		},
		{
			name:            "match 1.3.* pattern",
			pattern:         "1.3.*",
			versions:        versions,
			expectedMatches: []string{"1.3.3", "1.3.2", "1.3.1", "1.3"},
		},
		{
			name:            "match all with *",
			pattern:         "*",
			versions:        []string{"1.14.0", "1.15.0", "1.16.0"},
			expectedMatches: []string{"1.16.0", "1.15.0", "1.14.0"},
		},
		{
			name:            "no matches",
			pattern:         "1.99.*",
			versions:        versions,
			expectedMatches: []string{},
		},
		{
			name:            "empty versions list",
			pattern:         "1.14.*",
			versions:        []string{},
			expectedMatches: []string{},
		},
		{
			name:            "match 1.15.* pattern (fewer matches)",
			pattern:         "1.15.*",
			versions:        versions,
			expectedMatches: []string{"1.15", "1.15.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchVersionPattern(tt.pattern, tt.versions)

			if len(result) != len(tt.expectedMatches) {
				t.Errorf("MatchVersionPattern(%q, %v) returned %d matches, want %d\nGot: %v\nWant: %v",
					tt.pattern, tt.versions, len(result), len(tt.expectedMatches), result, tt.expectedMatches)
				return
			}

			for i, v := range result {
				if v != tt.expectedMatches[i] {
					t.Errorf("MatchVersionPattern(%q, %v)[%d] = %q, want %q",
						tt.pattern, tt.versions, i, v, tt.expectedMatches[i])
				}
			}
		})
	}
}

func TestMatchesPrefix(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		prefix   string
		expected bool
	}{
		// Empty prefix
		{
			name:     "empty prefix matches everything",
			version:  "1.14.0",
			prefix:   "",
			expected: true,
		},
		// Prefix ending with dot
		{
			name:     "prefix with dot matches version starting with prefix",
			version:  "1.14.0",
			prefix:   "1.14.",
			expected: true,
		},
		{
			name:     "prefix with dot matches prerelease rc",
			version:  "1.14rc1",
			prefix:   "1.14.",
			expected: true,
		},
		{
			name:     "prefix with dot matches prerelease beta",
			version:  "1.14beta1",
			prefix:   "1.14.",
			expected: true,
		},
		{
			name:     "prefix with dot matches exact base version",
			version:  "1.14",
			prefix:   "1.14.",
			expected: true,
		},
		{
			name:     "prefix with dot does not match different version",
			version:  "1.15.0",
			prefix:   "1.14.",
			expected: false,
		},
		// Exact match
		{
			name:     "exact version match",
			version:  "1.14",
			prefix:   "1.14",
			expected: true,
		},
		// Prefix followed by dot
		{
			name:     "version starts with prefix followed by dot",
			version:  "1.14.5",
			prefix:   "1.14",
			expected: true,
		},
		// Prerelease versions (prefix followed by non-digit)
		{
			name:     "prerelease version matches prefix",
			version:  "1.14rc1",
			prefix:   "1.14",
			expected: true,
		},
		{
			name:     "beta version matches prefix",
			version:  "1.14beta2",
			prefix:   "1.14",
			expected: true,
		},
		// Version with digit after prefix (should NOT match)
		{
			name:     "version with digit continuation does not match",
			version:  "1.141",
			prefix:   "1.14",
			expected: false,
		},
		// No match cases
		{
			name:     "completely different version",
			version:  "2.0.0",
			prefix:   "1.14",
			expected: false,
		},
		{
			name:     "partial prefix mismatch",
			version:  "1.15.0",
			prefix:   "1.14",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesPrefix(tt.version, tt.prefix)
			if result != tt.expected {
				t.Errorf("matchesPrefix(%q, %q) = %v, want %v", tt.version, tt.prefix, result, tt.expected)
			}
		})
	}
}

func TestIsDigit(t *testing.T) {
	tests := []struct {
		name     string
		char     byte
		expected bool
	}{
		{"digit 0", '0', true},
		{"digit 5", '5', true},
		{"digit 9", '9', true},
		{"letter a", 'a', false},
		{"letter Z", 'Z', false},
		{"dot", '.', false},
		{"dash", '-', false},
		{"space", ' ', false},
		{"underscore", '_', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDigit(tt.char)
			if result != tt.expected {
				t.Errorf("isDigit(%q) = %v, want %v", tt.char, result, tt.expected)
			}
		})
	}
}

func TestSortVersionsDescending(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "already sorted",
			input:    []string{"1.21.0", "1.20.0", "1.19.0"},
			expected: []string{"1.21.0", "1.20.0", "1.19.0"},
		},
		{
			name:     "reverse order",
			input:    []string{"1.19.0", "1.20.0", "1.21.0"},
			expected: []string{"1.21.0", "1.20.0", "1.19.0"},
		},
		{
			name:     "mixed order",
			input:    []string{"1.20.0", "1.21.0", "1.19.0"},
			expected: []string{"1.21.0", "1.20.0", "1.19.0"},
		},
		{
			name:     "single element",
			input:    []string{"1.20.0"},
			expected: []string{"1.20.0"},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "with patch versions",
			input:    []string{"1.20.1", "1.20.5", "1.20.3"},
			expected: []string{"1.20.5", "1.20.3", "1.20.1"},
		},
		{
			name:     "with prerelease",
			input:    []string{"1.21.0", "1.21rc1", "1.21beta1"},
			expected: []string{"1.21.0", "1.21rc1", "1.21beta1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the original
			input := make([]string, len(tt.input))
			copy(input, tt.input)

			sortVersionsDescending(input)

			if len(input) != len(tt.expected) {
				t.Errorf("sortVersionsDescending(%v) resulted in %d elements, want %d",
					tt.input, len(input), len(tt.expected))
				return
			}

			for i, v := range input {
				if v != tt.expected[i] {
					t.Errorf("sortVersionsDescending(%v)[%d] = %q, want %q",
						tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}
