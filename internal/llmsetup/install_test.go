package llmsetup

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed testdata/mock-skill/*
var testMockSkillsFS embed.FS

// testMockAgent implements the Agent interface for testing Install.
type testMockAgent struct {
	name           string
	description    string
	skillsFS       embed.FS
	skillsSrcDir   string
	skillsDirName  string
	projectSubPath string
	userSubPath    string
}

func (m *testMockAgent) Name() string           { return m.name }
func (m *testMockAgent) Description() string    { return m.description }
func (m *testMockAgent) SkillsFS() embed.FS     { return m.skillsFS }
func (m *testMockAgent) SkillsSrcDir() string   { return m.skillsSrcDir }
func (m *testMockAgent) SkillsDirName() string  { return m.skillsDirName }
func (m *testMockAgent) ProjectSubPath() string { return m.projectSubPath }
func (m *testMockAgent) UserSubPath() string    { return m.userSubPath }

func newTestMockAgent() *testMockAgent {
	return &testMockAgent{
		name:           "test-agent",
		description:    "Test agent",
		skillsFS:       testMockSkillsFS,
		skillsSrcDir:   "testdata/mock-skill",
		skillsDirName:  "test-skill",
		projectSubPath: ".test/skills",
		userSubPath:    ".test/skills",
	}
}

// TestInstallFile tests the InstallFile function with various scenarios.
func TestInstallFile(t *testing.T) {
	t.Run("creates directory with 0755 permissions", func(t *testing.T) {
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

		// Verify directory was created with correct permissions
		info, err := os.Stat(targetDir)
		if err != nil {
			t.Fatalf("directory was not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("path is not a directory")
		}
		if perm := info.Mode().Perm(); perm != 0755 {
			t.Errorf("directory permissions = %o, want 0755", perm)
		}

		// Verify file was created with correct permissions and content
		filePath := filepath.Join(targetDir, "test.md")
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("file was not created: %v", err)
		}
		if filePerm := fileInfo.Mode().Perm(); filePerm != 0644 {
			t.Errorf("file permissions = %o, want 0644", filePerm)
		}

		readContent, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(readContent) != string(content) {
			t.Errorf("file content = %q, want %q", string(readContent), string(content))
		}
	})

	t.Run("atomic overwrite of existing file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-install-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		fileName := "test.md"
		filePath := filepath.Join(tmpDir, fileName)

		// Create initial file
		if writeErr := os.WriteFile(filePath, []byte("initial content"), 0644); writeErr != nil {
			t.Fatalf("failed to create initial file: %v", writeErr)
		}

		// Overwrite with new content
		newContent := []byte("new content")
		if installErr := InstallFile(tmpDir, fileName, newContent); installErr != nil {
			t.Fatalf("InstallFile failed: %v", installErr)
		}

		// Verify new content
		readContent, readErr := os.ReadFile(filePath)
		if readErr != nil {
			t.Fatalf("failed to read file: %v", readErr)
		}
		if string(readContent) != string(newContent) {
			t.Errorf("file content = %q, want %q", string(readContent), string(newContent))
		}
	})

	t.Run("idempotent installation", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-install-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		fileName := "test.md"
		content := []byte("test content")

		// Install twice
		for i := 0; i < 2; i++ {
			if installErr := InstallFile(tmpDir, fileName, content); installErr != nil {
				t.Fatalf("InstallFile #%d failed: %v", i+1, installErr)
			}
		}

		// Verify content is correct
		readContent, readErr := os.ReadFile(filepath.Join(tmpDir, fileName))
		if readErr != nil {
			t.Fatalf("failed to read file: %v", readErr)
		}
		if string(readContent) != string(content) {
			t.Errorf("file content = %q, want %q", string(readContent), string(content))
		}
	})

	t.Run("cleans up on error", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-install-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		targetDir := filepath.Join(tmpDir, "readonly")
		if mkdirErr := os.MkdirAll(targetDir, 0755); mkdirErr != nil {
			t.Fatalf("failed to create target dir: %v", mkdirErr)
		}

		// Make directory read-only
		if chmodErr := os.Chmod(targetDir, 0555); chmodErr != nil {
			t.Fatalf("failed to make dir readonly: %v", chmodErr)
		}
		defer func() { _ = os.Chmod(targetDir, 0755) }()

		installErr := InstallFile(targetDir, "test.md", []byte("test content"))
		if installErr == nil {
			t.Skip("test requires permission enforcement (skipping on permissive systems)")
		}

		// Verify no temp files left behind
		entries, _ := os.ReadDir(targetDir)
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".tmp") {
				t.Errorf("temp file left behind: %s", entry.Name())
			}
		}
	})

	t.Run("permission denied error", func(t *testing.T) {
		err := InstallFile("/root/cannot-write-here", "test.md", []byte("test"))
		if err == nil {
			t.Skip("test requires root to be protected (skipping)")
		}
		// Error occurred as expected
	})
}

// TestInstall tests the Install function with various scenarios.
func TestInstall(t *testing.T) {
	tests := []struct {
		setup      func(t *testing.T) (agent *testMockAgent, customPath string, cleanup func())
		name       string
		errContain string
		wantErr    bool
		skipOnNil  bool // Skip if no error occurs (for permission-dependent tests)
	}{
		{
			name: "success",
			setup: func(t *testing.T) (*testMockAgent, string, func()) {
				tmpDir, err := os.MkdirTemp("", "test-install-*")
				if err != nil {
					t.Fatalf("failed to create temp dir: %v", err)
				}
				return newTestMockAgent(), tmpDir, func() { _ = os.RemoveAll(tmpDir) }
			},
			wantErr: false,
		},
		{
			name: "ReadDir error with non-existent srcDir",
			setup: func(t *testing.T) (*testMockAgent, string, func()) {
				tmpDir, err := os.MkdirTemp("", "test-install-*")
				if err != nil {
					t.Fatalf("failed to create temp dir: %v", err)
				}
				agent := &testMockAgent{
					name:           "test-agent",
					description:    "Test agent",
					skillsFS:       testMockSkillsFS,
					skillsSrcDir:   "non-existent-dir",
					skillsDirName:  "test-skill",
					projectSubPath: ".test/skills",
					userSubPath:    ".test/skills",
				}
				return agent, tmpDir, func() { _ = os.RemoveAll(tmpDir) }
			},
			wantErr:    true,
			errContain: "cannot read skills directory",
		},
		{
			name: "ValidatePath error when path is a file",
			setup: func(t *testing.T) (*testMockAgent, string, func()) {
				tmpFile, err := os.CreateTemp("", "test-file-*")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				_ = tmpFile.Close()
				return newTestMockAgent(), tmpFile.Name(), func() { _ = os.Remove(tmpFile.Name()) }
			},
			wantErr:    true,
			errContain: "path is a file",
		},
		{
			name: "InstallFile error with readonly directory",
			setup: func(t *testing.T) (*testMockAgent, string, func()) {
				tmpDir, err := os.MkdirTemp("", "test-install-*")
				if err != nil {
					t.Fatalf("failed to create temp dir: %v", err)
				}
				agent := newTestMockAgent()
				skillDir := filepath.Join(tmpDir, agent.SkillsDirName())
				if err := os.MkdirAll(skillDir, 0755); err != nil {
					t.Fatalf("failed to create skill dir: %v", err)
				}
				if err := os.Chmod(skillDir, 0555); err != nil {
					t.Fatalf("failed to make dir readonly: %v", err)
				}
				return agent, tmpDir, func() {
					_ = os.Chmod(skillDir, 0755)
					_ = os.RemoveAll(tmpDir)
				}
			},
			wantErr:   true,
			skipOnNil: true, // Permission may not be enforced
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, customPath, cleanup := tt.setup(t)
			defer cleanup()

			skillPath, err := Install(agent, customPath, false)
			if tt.skipOnNil && err == nil {
				t.Skip("test requires permission enforcement (skipping on permissive systems)")
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Install() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errContain != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContain)
				}
			}
			if !tt.wantErr {
				expectedPath := filepath.Join(customPath, agent.SkillsDirName())
				if skillPath != expectedPath {
					t.Errorf("skillPath = %q, want %q", skillPath, expectedPath)
				}
				// Verify SKILL.md was installed
				if _, statErr := os.Stat(filepath.Join(skillPath, "SKILL.md")); statErr != nil {
					t.Errorf("SKILL.md was not installed: %v", statErr)
				}
			}
		})
	}
}
