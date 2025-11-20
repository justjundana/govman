package symlink

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCreate_NewSymlink(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "target.txt")
	symlink := filepath.Join(tempDir, "link.txt")

	os.WriteFile(target, []byte("ok"), 0644)

	err := Create(target, symlink)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	resolved, err := os.Readlink(symlink)
	if err != nil {
		t.Fatalf("os.Readlink failed: %v", err)
	}
	if resolved != target {
		t.Errorf("expected %q, got %q", target, resolved)
	}
}

func TestCreate_OverwriteExisting(t *testing.T) {
	tempDir := t.TempDir()

	target1 := filepath.Join(tempDir, "file1.txt")
	symlink := filepath.Join(tempDir, "mylink")

	os.WriteFile(target1, []byte("v1"), 0644)
	if err := Create(target1, symlink); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	target2 := filepath.Join(tempDir, "file2.txt")
	os.WriteFile(target2, []byte("v2"), 0644)
	if err := Create(target2, symlink); err != nil {
		t.Fatalf("overwrite Create failed: %v", err)
	}

	resolved, _ := os.Readlink(symlink)
	if resolved != target2 {
		t.Errorf("expected %q, got %q", target2, resolved)
	}
}

func TestCreate_Error_ReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows: chmod 0500 behaves differently")
	}

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "target.txt")
	os.WriteFile(target, []byte("data"), 0644)

	readonlyDir := filepath.Join(tempDir, "readonly")
	os.Mkdir(readonlyDir, 0500)
	defer os.Chmod(readonlyDir, 0700)

	symlink := filepath.Join(readonlyDir, "link")

	err := Create(target, symlink)
	if err == nil {
		t.Error("expected error when creating symlink in read-only dir, got nil")
	}
}

func TestCreate_ErrorOnRemove(t *testing.T) {
	tempDir := t.TempDir()

	target := filepath.Join(tempDir, "target.txt")
	os.WriteFile(target, []byte("data"), 0644)

	blockDir := filepath.Join(tempDir, "blocked")
	if err := os.Mkdir(blockDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	blockFile := filepath.Join(blockDir, "file.txt")
	if err := os.WriteFile(blockFile, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to create file inside blockDir: %v", err)
	}

	err := Create(target, blockDir)
	if err == nil {
		t.Error("expected error when os.Remove fails on non-empty directory, got nil")
	}
}
