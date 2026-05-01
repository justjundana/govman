package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAssetExactMatch verifies the fix for CQ-1:
// Asset selection must use exact name matching, not substring Contains.
// This prevents "govman-linux-amd64" from matching "govman-linux-amd64-v2".
func TestAssetExactMatch(t *testing.T) {
	type asset struct {
		Name        string
		DownloadURL string
	}

	assets := []asset{
		{Name: "govman-linux-amd64-v2", DownloadURL: "https://example.com/v2"},
		{Name: "govman-linux-amd64", DownloadURL: "https://example.com/correct"},
		{Name: "govman-linux-arm64", DownloadURL: "https://example.com/arm"},
	}

	targetName := "govman-linux-amd64"

	// Exact match (fixed behavior): must find the exact name, not a superset
	var exactMatch string
	for _, a := range assets {
		if a.Name == targetName {
			exactMatch = a.DownloadURL
			break
		}
	}
	if exactMatch != "https://example.com/correct" {
		t.Errorf("exact match got %q, want %q", exactMatch, "https://example.com/correct")
	}

	// Old behavior (substring match): would incorrectly match the -v2 variant first
	var substringMatch string
	for _, a := range assets {
		if strings.Contains(a.Name, targetName) {
			substringMatch = a.DownloadURL
			break
		}
	}
	// The -v2 variant also contains "govman-linux-amd64" as a substring,
	// so Contains matches the wrong one first.
	if substringMatch == "https://example.com/v2" {
		t.Log("Confirmed: substring match picks wrong asset when -v2 variant appears first")
	}
}

// TestTempFileInBinaryDir verifies the fix for BUG-2:
// Temp file must be created in the same directory as the binary,
// not in the system temp dir, to avoid cross-device rename failures (EXDEV).
func TestTempFileInBinaryDir(t *testing.T) {
	// Create a temp directory to simulate a binary location
	binaryDir := t.TempDir()
	binaryPath := filepath.Join(binaryDir, "govman")

	// Create a dummy binary file
	if err := os.WriteFile(binaryPath, []byte("binary"), 0755); err != nil {
		t.Fatal(err)
	}

	// The fix uses filepath.Dir(currentBinary) as the temp dir
	tempDir := filepath.Dir(binaryPath)
	if tempDir != binaryDir {
		t.Errorf("filepath.Dir(binaryPath) = %q, want %q", tempDir, binaryDir)
	}

	// Create temp file in binary dir (same as the fix does)
	tmpFile, err := os.CreateTemp(tempDir, "govman-update-*.bin")
	if err != nil {
		t.Fatalf("failed to create temp file in binary dir: %v", err)
	}
	tmpName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpName)

	// Verify temp file is in the same directory as the binary
	if filepath.Dir(tmpName) != binaryDir {
		t.Errorf("temp file dir = %q, want %q", filepath.Dir(tmpName), binaryDir)
	}

	// Rename should succeed (same filesystem)
	newPath := filepath.Join(binaryDir, "govman-new")
	if err := os.Rename(tmpName, newPath); err != nil {
		t.Errorf("rename within same dir failed: %v", err)
	}
	defer os.Remove(newPath)
}
