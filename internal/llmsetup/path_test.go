package llmsetup

import (
	"embed"
	"os"
	"path/filepath"
	"testing"
)

//go:embed testdata/mock-skill/*
var mockSkillsFS embed.FS

// mockAgent implements the Agent interface for testing.
type mockAgent struct {
	name           string
	description    string
	skillsFS       embed.FS
	skillsSrcDir   string
	skillsDirName  string
	projectSubPath string
	userSubPath    string
}

func (m *mockAgent) Name() string           { return m.name }
func (m *mockAgent) Description() string    { return m.description }
func (m *mockAgent) SkillsFS() embed.FS     { return m.skillsFS }
func (m *mockAgent) SkillsSrcDir() string   { return m.skillsSrcDir }
func (m *mockAgent) SkillsDirName() string  { return m.skillsDirName }
func (m *mockAgent) ProjectSubPath() string { return m.projectSubPath }
func (m *mockAgent) UserSubPath() string    { return m.userSubPath }

func newMockAgent() *mockAgent {
	return &mockAgent{
		name:           "test-agent",
		description:    "Test agent",
		skillsFS:       mockSkillsFS,
		skillsSrcDir:   "testdata/mock-skill",
		skillsDirName:  "test-skill",
		projectSubPath: ".test/skills",
		userSubPath:    ".test/skills",
	}
}

// TestResolvePath_ProjectLevel tests project-level path resolution (T007).
func TestResolvePath_ProjectLevel(t *testing.T) {
	agent := newMockAgent()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	// Call ResolvePath with no custom path and no user flag (default: project-level)
	path, err := ResolvePath("", false, agent)
	if err != nil {
		t.Fatalf("ResolvePath failed: %v", err)
	}

	expected := filepath.Join(cwd, agent.ProjectSubPath())
	if path != expected {
		t.Errorf("ResolvePath = %q, want %q", path, expected)
	}
}

// TestResolvePath_UserLevel_Linux tests user-level path resolution on Linux (T021).
func TestResolvePath_UserLevel_Linux(t *testing.T) {
	if os.Getenv("HOME") == "" {
		t.Skip("HOME not set")
	}

	agent := newMockAgent()
	home := os.Getenv("HOME")

	path, err := ResolvePath("", true, agent)
	if err != nil {
		t.Fatalf("ResolvePath failed: %v", err)
	}

	expected := filepath.Join(home, agent.UserSubPath())
	if path != expected {
		t.Errorf("ResolvePath = %q, want %q", path, expected)
	}
}

// TestResolvePath_UserLevel_Darwin tests user-level path resolution on Darwin (T022).
// Note: On Darwin (macOS), the behavior is identical to Linux - uses $HOME.
// This test verifies the path structure when running on any Unix-like system.
func TestResolvePath_UserLevel_Darwin(t *testing.T) {
	if os.Getenv("HOME") == "" {
		t.Skip("HOME not set")
	}

	agent := newMockAgent()
	home := os.Getenv("HOME")

	path, err := ResolvePath("", true, agent)
	if err != nil {
		t.Fatalf("ResolvePath failed: %v", err)
	}

	// Darwin uses the same path structure as Linux
	expected := filepath.Join(home, agent.UserSubPath())
	if path != expected {
		t.Errorf("ResolvePath = %q, want %q", path, expected)
	}
}

// TestResolvePath_UserLevel_Windows tests user-level path resolution on Windows (T023).
// Note: On Windows, the behavior uses %USERPROFILE% instead of $HOME.
// This test runs on all platforms but uses HOME when available.
// A true Windows test would require running on Windows where USERPROFILE is set.
func TestResolvePath_UserLevel_Windows(t *testing.T) {
	// On non-Windows, this tests the same code path as Linux/Darwin
	// On Windows, it would use USERPROFILE
	if os.Getenv("HOME") == "" && os.Getenv("USERPROFILE") == "" {
		t.Skip("Neither HOME nor USERPROFILE is set")
	}

	agent := newMockAgent()

	path, err := ResolvePath("", true, agent)
	if err != nil {
		t.Fatalf("ResolvePath failed: %v", err)
	}

	// Just verify we got a valid path back
	if path == "" {
		t.Error("ResolvePath should return a non-empty path")
	}
}

// TestResolvePath_ProjectLevelWorksOnAnyOS tests that project-level path works regardless of OS (T025).
func TestResolvePath_ProjectLevelWorksOnAnyOS(t *testing.T) {
	agent := newMockAgent()

	path, err := ResolvePath("", false, agent)
	if err != nil {
		t.Errorf("project-level ResolvePath failed: %v", err)
	}
	if path == "" {
		t.Error("project-level path should not be empty")
	}
}

// TestResolvePath_HomeNotSet tests error when HOME is not set (T024).
func TestResolvePath_HomeNotSet(t *testing.T) {
	// Save and restore HOME
	origHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHome) }()

	// Unset HOME
	_ = os.Unsetenv("HOME")

	agent := newMockAgent()

	_, err := ResolvePath("", true, agent)
	if err == nil {
		t.Fatal("ResolvePath should fail when HOME is not set")
	}

	// Check error message contains expected text
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

// TestResolvePath_PathOverridesUser tests that --path takes precedence over --user (T034).
func TestResolvePath_PathOverridesUser(t *testing.T) {
	agent := newMockAgent()
	customPath := "/custom/path"

	// Both user flag and custom path specified - custom path should win
	path, err := ResolvePath(customPath, true, agent)
	if err != nil {
		t.Fatalf("ResolvePath failed: %v", err)
	}

	if path != customPath {
		t.Errorf("ResolvePath = %q, want %q (--path should override --user)", path, customPath)
	}
}

// TestResolvePath_RelativePath tests relative path resolution (T036).
func TestResolvePath_RelativePath(t *testing.T) {
	agent := newMockAgent()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	path, err := ResolvePath("relative/path", false, agent)
	if err != nil {
		t.Fatalf("ResolvePath failed: %v", err)
	}

	expected := filepath.Join(cwd, "relative/path")
	if path != expected {
		t.Errorf("ResolvePath = %q, want %q", path, expected)
	}
}

// TestValidatePath_PathIsFile tests error when path is a file (T037).
func TestValidatePath_PathIsFile(t *testing.T) {
	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test-file-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	err = ValidatePath(tmpFile.Name())
	if err == nil {
		t.Fatal("ValidatePath should fail when path is a file")
	}

	expected := "path is a file, not a directory"
	if err.Error() == "" || !contains(err.Error(), expected) {
		t.Errorf("error = %q, want to contain %q", err.Error(), expected)
	}
}

// TestValidatePath_NonExistentPath tests that non-existent path is OK.
func TestValidatePath_NonExistentPath(t *testing.T) {
	err := ValidatePath("/non/existent/path/that/does/not/exist")
	if err != nil {
		t.Errorf("ValidatePath should succeed for non-existent path: %v", err)
	}
}

// TestValidatePath_ExistingDirectory tests that existing directory is OK.
func TestValidatePath_ExistingDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-dir-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	err = ValidatePath(tmpDir)
	if err != nil {
		t.Errorf("ValidatePath should succeed for existing directory: %v", err)
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
