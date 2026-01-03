package llmsetup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

// TestClaudeCodeCmd tests the ClaudeCodeCmd command with various scenarios.
func TestClaudeCodeCmd(t *testing.T) {
	t.Run("success output", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-claudecode-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		var stdout, stderr bytes.Buffer
		cmd := &ClaudeCodeCmd{
			Path:   tmpDir,
			User:   false,
			Stdout: &stdout,
			Stderr: &stderr,
		}

		if err := cmd.Run(); err != nil {
			t.Fatalf("ClaudeCodeCmd.Run failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "Skills installed to:") {
			t.Errorf("stdout = %q, want to contain 'Skills installed to:'", output)
		}

		expectedPath := filepath.Join(tmpDir, "kessoku-di")
		if !strings.Contains(output, expectedPath) {
			t.Errorf("stdout = %q, want to contain path %q", output, expectedPath)
		}

		if stderr.Len() > 0 {
			t.Errorf("stderr should be empty, got %q", stderr.String())
		}
	})

	t.Run("error output when path is a file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-file-*")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		_ = tmpFile.Close()
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		var stdout, stderr bytes.Buffer
		cmd := &ClaudeCodeCmd{
			Path:   tmpFile.Name(),
			User:   false,
			Stdout: &stdout,
			Stderr: &stderr,
		}

		err = cmd.Run()
		if err == nil {
			t.Fatal("ClaudeCodeCmd.Run should fail when path is a file")
		}

		if !strings.Contains(stderr.String(), "Error:") {
			t.Errorf("stderr = %q, want to contain 'Error:'", stderr.String())
		}
	})

	t.Run("stdout/stderr returns defaults when nil", func(t *testing.T) {
		cmd := &ClaudeCodeCmd{
			Stdout: nil,
			Stderr: nil,
		}

		if cmd.stdout() != os.Stdout {
			t.Error("stdout() should return os.Stdout when Stdout is nil")
		}
		if cmd.stderr() != os.Stderr {
			t.Error("stderr() should return os.Stderr when Stderr is nil")
		}
	})
}

// TestClaudeCodeAgent tests that ClaudeCodeAgent implements Agent correctly.
func TestClaudeCodeAgent(t *testing.T) {
	var _ Agent = &ClaudeCodeAgent{} // Compile-time check

	agent := &ClaudeCodeAgent{}

	tests := []struct {
		name   string
		got    string
		want   string
		notNil bool // If true, just check that got is not empty
	}{
		{"Name", agent.Name(), "claude-code", false},
		{"Description", agent.Description(), "", true},
		{"SkillsSrcDir", agent.SkillsSrcDir(), "skills", false},
		{"SkillsDirName", agent.SkillsDirName(), "kessoku-di", false},
		{"ProjectSubPath", agent.ProjectSubPath(), ".claude/skills", false},
		{"UserSubPath", agent.UserSubPath(), ".claude/skills", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.notNil {
				if tt.got == "" {
					t.Errorf("%s() should not be empty", tt.name)
				}
			} else {
				if tt.got != tt.want {
					t.Errorf("%s() = %q, want %q", tt.name, tt.got, tt.want)
				}
			}
		})
	}

	// Test SkillsFS separately as it requires more complex verification
	t.Run("SkillsFS", func(t *testing.T) {
		skillsFS := agent.SkillsFS()
		entries, err := skillsFS.ReadDir(agent.SkillsSrcDir())
		if err != nil {
			t.Errorf("SkillsFS().ReadDir() failed: %v", err)
		}
		if len(entries) == 0 {
			t.Error("SkillsFS() should contain at least one file")
		}

		if _, err := skillsFS.ReadFile(agent.SkillsSrcDir() + "/SKILL.md"); err != nil {
			t.Errorf("SKILL.md not found in SkillsFS: %v", err)
		}
	})
}

// TestGetAgent tests the GetAgent function with various agent names.
func TestGetAgent(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		wantOK    bool
		wantNil   bool
	}{
		{
			name:      "known agent claude-code",
			agentName: "claude-code",
			wantOK:    true,
			wantNil:   false,
		},
		{
			name:      "unknown agent",
			agentName: "unknown-agent",
			wantOK:    false,
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, ok := GetAgent(tt.agentName)
			if ok != tt.wantOK {
				t.Errorf("GetAgent(%q) ok = %v, want %v", tt.agentName, ok, tt.wantOK)
			}
			if (agent == nil) != tt.wantNil {
				t.Errorf("GetAgent(%q) agent nil = %v, want %v", tt.agentName, agent == nil, tt.wantNil)
			}
		})
	}
}

// TestListAgents tests the ListAgents function.
func TestListAgents(t *testing.T) {
	agents := ListAgents()
	if len(agents) == 0 {
		t.Error("ListAgents should return at least one agent")
	}

	// Verify claude-code is in the list
	found := false
	for _, a := range agents {
		if a.Name() == "claude-code" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ListAgents should include claude-code")
	}
}

// TestLLMSetupCmd tests LLMSetupCmd structure and defaults.
func TestLLMSetupCmd(t *testing.T) {
	cmd := &LLMSetupCmd{}

	if cmd.ClaudeCode.User != false {
		t.Error("ClaudeCode.User should default to false")
	}
	if cmd.ClaudeCode.Path != "" {
		t.Error("ClaudeCode.Path should default to empty")
	}
}

// TestUsageCmd_Run tests that UsageCmd.Run prints usage and returns nil.
func TestUsageCmd_Run(t *testing.T) {
	var stdout, stderr bytes.Buffer

	cmd := &LLMSetupCmd{}
	parser, err := kong.New(cmd,
		kong.Writers(&stdout, &stderr),
		kong.Exit(func(int) {}),
	)
	if err != nil {
		t.Fatalf("failed to create kong parser: %v", err)
	}

	ctx, err := parser.Parse([]string{})
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	usageCmd := &UsageCmd{}
	if err := usageCmd.Run(ctx); err != nil {
		t.Errorf("UsageCmd.Run() returned error: %v", err)
	}

	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("usage output should contain 'Usage:', got: %q", stdout.String())
	}
}

// TestInstall_Integration tests the complete Install function with real ClaudeCodeAgent.
func TestInstall_Integration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-install-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	agent := &ClaudeCodeAgent{}

	skillDir, err := Install(agent, tmpDir, false)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify skill directory was created
	expectedSkillDir := filepath.Join(tmpDir, agent.SkillsDirName())
	if skillDir != expectedSkillDir {
		t.Errorf("Install returned wrong path: got %s, want %s", skillDir, expectedSkillDir)
	}

	info, err := os.Stat(skillDir)
	if err != nil {
		t.Fatalf("skill directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("skill path is not a directory")
	}

	// Verify all files from embedded FS were installed
	skillsFS := agent.SkillsFS()
	entries, err := skillsFS.ReadDir(agent.SkillsSrcDir())
	if err != nil {
		t.Fatalf("failed to read embedded dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		expectedContent, readErr := skillsFS.ReadFile(agent.SkillsSrcDir() + "/" + entry.Name())
		if readErr != nil {
			t.Fatalf("failed to read embedded file %s: %v", entry.Name(), readErr)
		}

		installedContent, readErr := os.ReadFile(filepath.Join(skillDir, entry.Name()))
		if readErr != nil {
			t.Fatalf("installed file %s not found: %v", entry.Name(), readErr)
		}

		if string(installedContent) != string(expectedContent) {
			t.Errorf("installed content of %s does not match embedded content", entry.Name())
		}
	}
}
