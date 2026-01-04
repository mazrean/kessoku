package llmsetup

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
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

// TestResolvePath tests path resolution with various configurations.
func TestResolvePath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	home := os.Getenv("HOME")

	agent := newMockAgent()

	tests := []struct {
		skip       func() bool
		name       string
		customPath string
		want       string
		userFlag   bool
		wantErr    bool
	}{
		{
			name:       "project-level (default)",
			customPath: "",
			userFlag:   false,
			want:       filepath.Join(cwd, agent.ProjectSubPath()),
		},
		{
			name:       "user-level with HOME set",
			customPath: "",
			userFlag:   true,
			skip:       func() bool { return home == "" },
			want:       filepath.Join(home, agent.UserSubPath()),
		},
		{
			name:       "custom path overrides user flag",
			customPath: "/custom/path",
			userFlag:   true,
			want:       "/custom/path",
		},
		{
			name:       "relative path converted to absolute",
			customPath: "relative/path",
			userFlag:   false,
			want:       filepath.Join(cwd, "relative/path"),
		},
		{
			name:       "project-level works on any OS",
			customPath: "",
			userFlag:   false,
			want:       filepath.Join(cwd, agent.ProjectSubPath()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip != nil && tt.skip() {
				t.Skip("skip condition met")
			}

			got, err := ResolvePath(tt.customPath, tt.userFlag, agent)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolvePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolvePath() = %q, want %q", got, tt.want)
			}
		})
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

	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

// TestValidatePath tests path validation with various scenarios.
func TestValidatePath(t *testing.T) {
	// Create temp dir for tests
	tmpDir, err := os.MkdirTemp("", "test-validate-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a temp file
	tmpFile, err := os.CreateTemp(tmpDir, "test-file-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	_ = tmpFile.Close()

	tests := []struct {
		name       string
		path       string
		errContain string
		wantErr    bool
	}{
		{
			name:    "non-existent path is OK",
			path:    "/non/existent/path/that/does/not/exist",
			wantErr: false,
		},
		{
			name:    "existing directory is OK",
			path:    tmpDir,
			wantErr: false,
		},
		{
			name:       "path is a file",
			path:       tmpFile.Name(),
			wantErr:    true,
			errContain: "path is a file, not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errContain != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContain)
				}
			}
		})
	}
}

// TestValidatePath_PermissionDenied tests error when path cannot be accessed.
func TestValidatePath_PermissionDenied(t *testing.T) {
	// Create a directory with a subdirectory that has no permissions
	tmpDir, err := os.MkdirTemp("", "test-validate-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.Chmod(filepath.Join(tmpDir, "protected"), 0755)
		_ = os.RemoveAll(tmpDir)
	}()

	// Create protected directory
	protectedDir := filepath.Join(tmpDir, "protected")
	if mkdirErr := os.MkdirAll(protectedDir, 0000); mkdirErr != nil {
		t.Fatalf("failed to create protected dir: %v", mkdirErr)
	}

	// Try to validate a path inside the protected directory
	testPath := filepath.Join(protectedDir, "test")
	err = ValidatePath(testPath)

	// On systems with strict permissions, this should fail
	if err != nil {
		if !strings.Contains(err.Error(), "cannot access path") {
			t.Errorf("error = %q, want to contain 'cannot access path'", err.Error())
		}
	}
	// If err is nil, permissions are not enforced - that's OK
}
