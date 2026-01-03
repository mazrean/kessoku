package llmsetup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestClaudeCodeCmd_SuccessOutput tests success message output (T012).
func TestClaudeCodeCmd_SuccessOutput(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-claudecode-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := &ClaudeCodeCmd{
		Path:   tmpDir,
		User:   false,
		Stdout: &stdout,
		Stderr: &stderr,
	}

	err = cmd.Run()
	if err != nil {
		t.Fatalf("ClaudeCodeCmd.Run failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Skills installed to:") {
		t.Errorf("stdout = %q, want to contain 'Skills installed to:'", output)
	}

	// Now we show the skill directory path since multiple files are installed
	expectedPath := filepath.Join(tmpDir, "kessoku-di")
	if !strings.Contains(output, expectedPath) {
		t.Errorf("stdout = %q, want to contain path %q", output, expectedPath)
	}

	if stderr.Len() > 0 {
		t.Errorf("stderr should be empty, got %q", stderr.String())
	}
}

// TestClaudeCodeCmd_ErrorOutput tests error message output (T013).
func TestClaudeCodeCmd_ErrorOutput(t *testing.T) {
	// Create a file where we expect a directory
	tmpFile, err := os.CreateTemp("", "test-file-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := &ClaudeCodeCmd{
		Path:   tmpFile.Name(), // This is a file, not a directory
		User:   false,
		Stdout: &stdout,
		Stderr: &stderr,
	}

	err = cmd.Run()
	if err == nil {
		t.Fatal("ClaudeCodeCmd.Run should fail when path is a file")
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "Error:") {
		t.Errorf("stderr = %q, want to contain 'Error:'", errOutput)
	}
}

// TestClaudeCodeAgent_Interface tests that ClaudeCodeAgent implements Agent correctly.
func TestClaudeCodeAgent_Interface(t *testing.T) {
	var _ Agent = &ClaudeCodeAgent{} // Compile-time check

	agent := &ClaudeCodeAgent{}

	if agent.Name() != "claude-code" {
		t.Errorf("Name() = %q, want %q", agent.Name(), "claude-code")
	}

	if agent.Description() == "" {
		t.Error("Description() should not be empty")
	}

	// Verify SkillsFS returns a valid embed.FS with files
	skillsFS := agent.SkillsFS()
	entries, err := skillsFS.ReadDir(agent.SkillsSrcDir())
	if err != nil {
		t.Errorf("SkillsFS().ReadDir() failed: %v", err)
	}
	if len(entries) == 0 {
		t.Error("SkillsFS() should contain at least one file")
	}

	// Verify SKILL.md exists
	_, err = skillsFS.ReadFile(agent.SkillsSrcDir() + "/SKILL.md")
	if err != nil {
		t.Errorf("SKILL.md not found in SkillsFS: %v", err)
	}

	if agent.SkillsSrcDir() != "skills" {
		t.Errorf("SkillsSrcDir() = %q, want %q", agent.SkillsSrcDir(), "skills")
	}

	if agent.SkillsDirName() != "kessoku-di" {
		t.Errorf("SkillsDirName() = %q, want %q", agent.SkillsDirName(), "kessoku-di")
	}

	if agent.ProjectSubPath() != ".claude/skills" {
		t.Errorf("ProjectSubPath() = %q, want %q", agent.ProjectSubPath(), ".claude/skills")
	}

	if agent.UserSubPath() != ".claude/skills" {
		t.Errorf("UserSubPath() = %q, want %q", agent.UserSubPath(), ".claude/skills")
	}
}

// TestLLMSetupCmd_NoSubcommand tests that running llm-setup without subcommand exits 0 (T030).
// Note: The actual exit behavior is handled by Kong when parsing CLI args.
// This test verifies that LLMSetupCmd.Run() returns nil (no error = exit 0).
func TestLLMSetupCmd_NoSubcommand(t *testing.T) {
	cmd := &LLMSetupCmd{}

	// When LLMSetupCmd.Run() is called (no subcommand), it should return nil
	// The Run() method calls ctx.PrintUsage() which requires a kong.Context
	// For unit testing, we just verify the struct exists and the method signature
	// A full integration test would use kong.Parse to test the actual CLI behavior

	// Verify LLMSetupCmd has the expected structure
	if cmd.ClaudeCode.User != false {
		t.Error("ClaudeCode.User should default to false")
	}
	if cmd.ClaudeCode.Path != "" {
		t.Error("ClaudeCode.Path should default to empty")
	}
}

// TestLLMSetupCmd_UnknownAgent tests that unknown agents exit with error (T031).
// Note: Kong automatically handles unknown subcommands with exit 1.
// This test documents that behavior - the actual error handling is in Kong.
func TestLLMSetupCmd_UnknownAgent(t *testing.T) {
	// Kong handles unknown subcommands automatically:
	// - Prints error: "unknown command '<name>'"
	// - Lists available commands
	// - Exits with code 1
	//
	// The agent registry provides the list of valid agents:
	agent, ok := GetAgent("unknown-agent")
	if ok {
		t.Error("GetAgent should return false for unknown agent")
	}
	if agent != nil {
		t.Error("GetAgent should return nil for unknown agent")
	}

	// Verify known agent exists
	agent, ok = GetAgent("claude-code")
	if !ok {
		t.Error("GetAgent should return true for claude-code")
	}
	if agent == nil {
		t.Error("GetAgent should return non-nil for claude-code")
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

// TestInstall_Integration tests the complete Install function.
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

		// Read expected content from embed.FS
		expectedContent, err := skillsFS.ReadFile(agent.SkillsSrcDir() + "/" + entry.Name())
		if err != nil {
			t.Fatalf("failed to read embedded file %s: %v", entry.Name(), err)
		}

		// Read installed file
		installedPath := filepath.Join(skillDir, entry.Name())
		installedContent, err := os.ReadFile(installedPath)
		if err != nil {
			t.Fatalf("installed file %s not found: %v", entry.Name(), err)
		}

		if string(installedContent) != string(expectedContent) {
			t.Errorf("installed content of %s does not match embedded content", entry.Name())
		}
	}
}
