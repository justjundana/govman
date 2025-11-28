package util

import (
	"fmt"
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
