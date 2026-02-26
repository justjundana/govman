package util

import (
	"fmt"
	"sort"
	"strings"

	_golang "github.com/justjundana/govman/internal/golang"
)

// ExtractMajorMinor extracts the major.minor version from a version string.
// Examples:
//   - "1.25.4" -> "1.25"
//   - "1.25" -> "1.25"
//   - "1" -> "1"
func ExtractMajorMinor(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return version
}

// FindBestMatchingVersion finds the best matching installed version for a requested version.
// It matches based on major.minor version (e.g., "1.25" matches "1.25.1", "1.25.4", etc.).
// If multiple versions match, it returns the highest (latest patch) version.
//
// Parameters:
//   - requestedVersion: The version requested (can be partial like "1.25" or full like "1.25.4")
//   - installedVersions: List of installed versions to search from
//
// Returns:
//   - The best matching version, or an error if no match is found
//
// Examples:
//   - requestedVersion="1.25", installedVersions=["1.25.1", "1.25.4", "1.26.0"] -> "1.25.4"
//   - requestedVersion="1.25.4", installedVersions=["1.25.1", "1.24.3"] -> "1.25.1"
//   - requestedVersion="1.25", installedVersions=["1.24.5", "1.26.0"] -> error
func FindBestMatchingVersion(requestedVersion string, installedVersions []string) (string, error) {
	if len(installedVersions) == 0 {
		return "", fmt.Errorf("no versions installed")
	}

	requestedMajorMinor := ExtractMajorMinor(requestedVersion)

	// Find all versions that match the major.minor
	var matchingVersions []string
	for _, installed := range installedVersions {
		installedMajorMinor := ExtractMajorMinor(installed)
		if installedMajorMinor == requestedMajorMinor {
			matchingVersions = append(matchingVersions, installed)
		}
	}

	if len(matchingVersions) == 0 {
		return "", fmt.Errorf("no installed version matches %s (major.minor: %s)", requestedVersion, requestedMajorMinor)
	}

	// If there's only one match, return it
	if len(matchingVersions) == 1 {
		return matchingVersions[0], nil
	}

	// If multiple matches, return the highest version
	bestVersion := matchingVersions[0]
	for _, v := range matchingVersions[1:] {
		if _golang.CompareVersions(v, bestVersion) > 0 {
			bestVersion = v
		}
	}

	return bestVersion, nil
}

// IsWildcardPattern checks if a version string contains a wildcard pattern.
// Supports patterns like "1.14.*" where * matches any suffix.
func IsWildcardPattern(version string) bool {
	return strings.Contains(version, "*")
}

// MatchVersionPattern filters versions that match a given wildcard pattern.
// Pattern examples:
//   - "1.14.*" matches "1.14", "1.14.0", "1.14.1", "1.14.2", "1.14rc1", etc.
//   - "*" matches all versions
//
// Parameters:
//   - pattern: The wildcard pattern (e.g., "1.14.*")
//   - versions: List of available versions to filter
//
// Returns:
//   - Slice of versions that match the pattern, sorted in descending order
func MatchVersionPattern(pattern string, versions []string) []string {
	if pattern == "*" {
		// Match all versions
		result := make([]string, len(versions))
		copy(result, versions)
		sortVersionsDescending(result)
		return result
	}

	// Remove the wildcard suffix to get the prefix
	// "1.14.*" -> "1.14."
	// "1.14*" -> "1.14"
	prefix := strings.TrimSuffix(pattern, "*")

	var matched []string
	for _, v := range versions {
		if matchesPrefix(v, prefix) {
			matched = append(matched, v)
		}
	}

	sortVersionsDescending(matched)
	return matched
}

// matchesPrefix checks if a version matches a given prefix pattern.
// Handles edge cases like:
//   - prefix "1.14." matches "1.14.0", "1.14.1", "1.14", "1.14rc1", "1.14beta1", etc.
//   - prefix "1.14" matches "1.14", "1.14.0", "1.14rc1", etc.
func matchesPrefix(version, prefix string) bool {
	if prefix == "" {
		return true
	}

	// If prefix ends with ".", we need to handle both stable and prerelease patterns
	// e.g., "1.14." should match "1.14", "1.14.0", "1.14.1", "1.14rc1", "1.14beta1"
	if strings.HasSuffix(prefix, ".") {
		basePrefix := strings.TrimSuffix(prefix, ".")

		// Exact match with base prefix (e.g., "1.14" matches prefix "1.14.")
		if version == basePrefix {
			return true
		}

		// Starts with prefix (e.g., "1.14.1" matches prefix "1.14.")
		if strings.HasPrefix(version, prefix) {
			return true
		}

		// Prerelease versions (e.g., "1.14rc1" matches prefix "1.14.")
		if strings.HasPrefix(version, basePrefix) {
			remaining := strings.TrimPrefix(version, basePrefix)
			// Remaining should start with non-digit (like "rc", "beta") or "."
			if len(remaining) > 0 && (remaining[0] == '.' || !isDigit(remaining[0])) {
				return true
			}
		}

		return false
	}

	// Otherwise, match if version equals prefix or starts with "prefix." or "prefix<non-digit>"
	if version == prefix {
		return true
	}

	if strings.HasPrefix(version, prefix+".") {
		return true
	}

	// Also match prerelease versions like "1.14rc1" for pattern "1.14*"
	if strings.HasPrefix(version, prefix) {
		remaining := strings.TrimPrefix(version, prefix)
		// Remaining should start with non-digit (like "rc", "beta") or "."
		if len(remaining) > 0 && (remaining[0] == '.' || !isDigit(remaining[0])) {
			return true
		}
	}

	return false
}

// isDigit checks if a byte is a digit character.
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// sortVersionsDescending sorts versions in descending order (newest first).
func sortVersionsDescending(versions []string) {
	sort.Slice(versions, func(i, j int) bool {
		return _golang.CompareVersions(versions[i], versions[j]) > 0
	})
}
