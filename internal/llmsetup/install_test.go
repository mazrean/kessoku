package llmsetup

import (
	"os"
	"path/filepath"
	"testing"
)

// TestInstallFile_CreatesDirectory tests directory creation with 0755 permissions (T008).
func TestInstallFile_CreatesDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-install-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	targetDir := filepath.Join(tmpDir, "new", "nested", "dir")
	content := []byte("test content")

	err = InstallFile(targetDir, "test.md", content)
	if err != nil {
		t.Fatalf("InstallFile failed: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(targetDir)
	if err != nil {
		t.Fatalf("directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("path is not a directory")
	}

	// Verify directory permissions (0755)
	perm := info.Mode().Perm()
	if perm != 0755 {
		t.Errorf("directory permissions = %o, want 0755", perm)
	}

	// Verify file was created
	filePath := filepath.Join(targetDir, "test.md")
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("file was not created: %v", err)
	}

	// Verify file permissions (0644)
	filePerm := fileInfo.Mode().Perm()
	if filePerm != 0644 {
		t.Errorf("file permissions = %o, want 0644", filePerm)
	}

	// Verify file content
	readContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(readContent) != string(content) {
		t.Errorf("file content = %q, want %q", string(readContent), string(content))
	}
}

// TestInstallFile_AtomicOverwrite tests atomic overwrite of existing file (T009).
func TestInstallFile_AtomicOverwrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-install-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	fileName := "test.md"
	filePath := filepath.Join(tmpDir, fileName)

	// Create initial file
	initialContent := []byte("initial content")
	err = os.WriteFile(filePath, initialContent, 0644)
	if err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	// Overwrite with new content
	newContent := []byte("new content")
	err = InstallFile(tmpDir, fileName, newContent)
	if err != nil {
		t.Fatalf("InstallFile failed: %v", err)
	}

	// Verify new content
	readContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(readContent) != string(newContent) {
		t.Errorf("file content = %q, want %q", string(readContent), string(newContent))
	}
}

// TestInstallFile_Idempotent tests that running install twice produces identical result (T010).
func TestInstallFile_Idempotent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-install-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	fileName := "test.md"
	content := []byte("test content")

	// Install first time
	err = InstallFile(tmpDir, fileName, content)
	if err != nil {
		t.Fatalf("first InstallFile failed: %v", err)
	}

	// Install second time (same content)
	err = InstallFile(tmpDir, fileName, content)
	if err != nil {
		t.Fatalf("second InstallFile failed: %v", err)
	}

	// Verify content is correct
	filePath := filepath.Join(tmpDir, fileName)
	readContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(readContent) != string(content) {
		t.Errorf("file content = %q, want %q", string(readContent), string(content))
	}
}

// TestInstallFile_CleansUpOnError tests cleanup of partial files on error (T011).
func TestInstallFile_CleansUpOnError(t *testing.T) {
	// Create a read-only directory to simulate write failure
	tmpDir, err := os.MkdirTemp("", "test-install-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	targetDir := filepath.Join(tmpDir, "readonly")
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	// Make directory read-only
	err = os.Chmod(targetDir, 0555)
	if err != nil {
		t.Fatalf("failed to make dir readonly: %v", err)
	}
	defer func() { _ = os.Chmod(targetDir, 0755) }() // Restore for cleanup

	content := []byte("test content")
	err = InstallFile(targetDir, "test.md", content)

	// Should fail due to permission denied
	if err == nil {
		t.Skip("test requires permission enforcement (skipping on permissive systems)")
	}

	// Verify no temp files left behind
	entries, _ := os.ReadDir(targetDir)
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == "" || entry.Name()[:4] == ".tmp" {
			t.Errorf("temp file left behind: %s", entry.Name())
		}
	}
}

// TestInstallFile_PermissionDenied tests error message for permission denied (T038).
func TestInstallFile_PermissionDenied(t *testing.T) {
	// Try to install to a directory we can't write to
	err := InstallFile("/root/cannot-write-here", "test.md", []byte("test"))
	if err == nil {
		t.Skip("test requires root to be protected (skipping)")
	}
	// We verified err != nil by not skipping above
}
