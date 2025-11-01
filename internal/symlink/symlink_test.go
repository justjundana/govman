package symlink

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReadLink_AbsoluteAndRelative(t *testing.T) {
	tempDir := t.TempDir()

	targetPath := filepath.Join(tempDir, "file.txt")
	if err := os.WriteFile(targetPath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	symlinkAbs := filepath.Join(tempDir, "abs_link")
	if err := os.Symlink(targetPath, symlinkAbs); err != nil {
		t.Skipf("Skipping test: os.Symlink failed (maybe not supported on this OS): %v", err)
	}

	resultAbs, err := ReadLink(symlinkAbs)
	if err != nil {
		t.Fatalf("ReadLink absolute failed: %v", err)
	}
	if resultAbs != targetPath {
		t.Errorf("expected %q, got %q", targetPath, resultAbs)
	}

	relTarget := "file.txt"
	symlinkRel := filepath.Join(tempDir, "rel_link")
	if err := os.Symlink(relTarget, symlinkRel); err != nil {
		t.Skipf("Skipping test: os.Symlink failed (maybe not supported): %v", err)
	}

	resultRel, err := ReadLink(symlinkRel)
	if err != nil {
		t.Fatalf("ReadLink relative failed: %v", err)
	}
	expectedRel := filepath.Join(tempDir, relTarget)
	if resultRel != expectedRel {
		t.Errorf("expected %q, got %q", expectedRel, resultRel)
	}
}

func TestReadLink_Error_NotASymlink(t *testing.T) {
	tempDir := t.TempDir()
	regularFile := filepath.Join(tempDir, "regular.txt")

	if err := os.WriteFile(regularFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadLink(regularFile)
	if err == nil {
		t.Error("expected error when reading non-symlink file, got nil")
	}
}

func TestReadLink_Error_NotExist(t *testing.T) {
	_, err := ReadLink("/path/does/not/exist")
	if err == nil {
		t.Error("expected error for non-existent path, got nil")
	}
}

func TestCreate_NewSymlink(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "target.txt")
	symlink := filepath.Join(tempDir, "link.txt")

	os.WriteFile(target, []byte("ok"), 0644)

	err := Create(target, symlink)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	resolved, err := ReadLink(symlink)
	if err != nil {
		t.Fatalf("ReadLink failed: %v", err)
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

	resolved, _ := ReadLink(symlink)
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
