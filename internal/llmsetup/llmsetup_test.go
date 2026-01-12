package llmsetup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

// testAgentCmd is a generic test helper for AgentCmd tests.
func testAgentCmd[T Agent](t *testing.T, cmdName string, newCmd func(path string, stdout, stderr *bytes.Buffer) *AgentCmd[T]) {
	t.Helper()

	t.Run("success output", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-"+cmdName+"-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		var stdout, stderr bytes.Buffer
		cmd := newCmd(tmpDir, &stdout, &stderr)

		if err := cmd.Run(); err != nil {
			t.Fatalf("%sCmd.Run failed: %v", cmdName, err)
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
		cmd := newCmd(tmpFile.Name(), &stdout, &stderr)

		err = cmd.Run()
		if err == nil {
			t.Fatalf("%sCmd.Run should fail when path is a file", cmdName)
		}

		if !strings.Contains(stderr.String(), "Error:") {
			t.Errorf("stderr = %q, want to contain 'Error:'", stderr.String())
		}
	})
}

func TestClaudeCodeCmd(t *testing.T) {
	testAgentCmd(t, "ClaudeCode", func(path string, stdout, stderr *bytes.Buffer) *ClaudeCodeCmd {
		return &ClaudeCodeCmd{Path: path, Stdout: stdout, Stderr: stderr}
	})
}

func TestCursorCmd(t *testing.T) {
	testAgentCmd(t, "Cursor", func(path string, stdout, stderr *bytes.Buffer) *CursorCmd {
		return &CursorCmd{Path: path, Stdout: stdout, Stderr: stderr}
	})
}

func TestCopilotCmd(t *testing.T) {
	testAgentCmd(t, "Copilot", func(path string, stdout, stderr *bytes.Buffer) *CopilotCmd {
		return &CopilotCmd{Path: path, Stdout: stdout, Stderr: stderr}
	})
}

func TestGeminiCLICmd(t *testing.T) {
	testAgentCmd(t, "GeminiCLI", func(path string, stdout, stderr *bytes.Buffer) *GeminiCLICmd {
		return &GeminiCLICmd{Path: path, Stdout: stdout, Stderr: stderr}
	})
}

func TestOpenCodeCmd(t *testing.T) {
	testAgentCmd(t, "OpenCode", func(path string, stdout, stderr *bytes.Buffer) *OpenCodeCmd {
		return &OpenCodeCmd{Path: path, Stdout: stdout, Stderr: stderr}
	})
}

func TestCodexCmd(t *testing.T) {
	testAgentCmd(t, "Codex", func(path string, stdout, stderr *bytes.Buffer) *CodexCmd {
		return &CodexCmd{Path: path, Stdout: stdout, Stderr: stderr}
	})
}

func TestAmpCmd(t *testing.T) {
	testAgentCmd(t, "Amp", func(path string, stdout, stderr *bytes.Buffer) *AmpCmd {
		return &AmpCmd{Path: path, Stdout: stdout, Stderr: stderr}
	})
}

func TestGooseCmd(t *testing.T) {
	testAgentCmd(t, "Goose", func(path string, stdout, stderr *bytes.Buffer) *GooseCmd {
		return &GooseCmd{Path: path, Stdout: stdout, Stderr: stderr}
	})
}

func TestFactoryCmd(t *testing.T) {
	testAgentCmd(t, "Factory", func(path string, stdout, stderr *bytes.Buffer) *FactoryCmd {
		return &FactoryCmd{Path: path, Stdout: stdout, Stderr: stderr}
	})
}

// TestAgents tests all Agent implementations with table-driven tests.
func TestAgents(t *testing.T) {
	tests := []struct {
		name           string
		agent          Agent
		wantName       string
		projectSubPath string
		userSubPath    string
	}{
		{
			name:           "ClaudeCodeAgent",
			agent:          &ClaudeCodeAgent{},
			wantName:       "claude-code",
			projectSubPath: ".claude/skills",
			userSubPath:    ".claude/skills",
		},
		{
			name:           "CursorAgent",
			agent:          &CursorAgent{},
			wantName:       "cursor",
			projectSubPath: ".cursor/rules",
			userSubPath:    ".cursor/rules",
		},
		{
			name:           "CopilotAgent",
			agent:          &CopilotAgent{},
			wantName:       "github-copilot",
			projectSubPath: ".github/skills",
			userSubPath:    ".github/skills",
		},
		{
			name:           "GeminiCLIAgent",
			agent:          &GeminiCLIAgent{},
			wantName:       "gemini-cli",
			projectSubPath: ".gemini/skills",
			userSubPath:    ".gemini/skills",
		},
		{
			name:           "OpenCodeAgent",
			agent:          &OpenCodeAgent{},
			wantName:       "opencode",
			projectSubPath: ".opencode/skill",
			userSubPath:    ".config/opencode/skill",
		},
		{
			name:           "CodexAgent",
			agent:          &CodexAgent{},
			wantName:       "openai-codex",
			projectSubPath: ".codex/skills",
			userSubPath:    ".codex/skills",
		},
		{
			name:           "AmpAgent",
			agent:          &AmpAgent{},
			wantName:       "amp",
			projectSubPath: ".agents/skills",
			userSubPath:    ".config/agents/skills",
		},
		{
			name:           "GooseAgent",
			agent:          &GooseAgent{},
			wantName:       "goose",
			projectSubPath: ".agents/skills",
			userSubPath:    ".config/goose/skills",
		},
		{
			name:           "FactoryAgent",
			agent:          &FactoryAgent{},
			wantName:       "factory",
			projectSubPath: ".factory/skills",
			userSubPath:    ".factory/skills",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.agent.Name(); got != tt.wantName {
				t.Errorf("Name() = %q, want %q", got, tt.wantName)
			}
			if got := tt.agent.Description(); got == "" {
				t.Error("Description() should not be empty")
			}
			if got := tt.agent.SkillsSrcDir(); got != "skills/kessoku-di" {
				t.Errorf("SkillsSrcDir() = %q, want %q", got, "skills/kessoku-di")
			}
			if got := tt.agent.SkillsDirName(); got != "kessoku-di" {
				t.Errorf("SkillsDirName() = %q, want %q", got, "kessoku-di")
			}
			if got := tt.agent.ProjectSubPath(); got != tt.projectSubPath {
				t.Errorf("ProjectSubPath() = %q, want %q", got, tt.projectSubPath)
			}
			if got := tt.agent.UserSubPath(); got != tt.userSubPath {
				t.Errorf("UserSubPath() = %q, want %q", got, tt.userSubPath)
			}

			// Test SkillsFS
			skillsFS := tt.agent.SkillsFS()
			entries, err := skillsFS.ReadDir(tt.agent.SkillsSrcDir())
			if err != nil {
				t.Errorf("SkillsFS().ReadDir() failed: %v", err)
			}
			if len(entries) == 0 {
				t.Error("SkillsFS() should contain at least one file")
			}
			if _, err := skillsFS.ReadFile(tt.agent.SkillsSrcDir() + "/SKILL.md"); err != nil {
				t.Errorf("SKILL.md not found in SkillsFS: %v", err)
			}
		})
	}

	// Test that all agents share the same SkillsFS
	t.Run("SkillsFS shared across agents", func(t *testing.T) {
		claudeEntries, _ := (&ClaudeCodeAgent{}).SkillsFS().ReadDir("skills/kessoku-di")
		for _, tt := range tests {
			entries, _ := tt.agent.SkillsFS().ReadDir(tt.agent.SkillsSrcDir())
			if len(entries) != len(claudeEntries) {
				t.Errorf("%s SkillsFS should have same files as ClaudeCodeAgent", tt.name)
			}
		}
	})
}

func TestGetAgent(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		wantOK    bool
	}{
		{"known agent claude-code", "claude-code", true},
		{"known agent cursor", "cursor", true},
		{"known agent github-copilot", "github-copilot", true},
		{"known agent gemini-cli", "gemini-cli", true},
		{"known agent opencode", "opencode", true},
		{"known agent openai-codex", "openai-codex", true},
		{"known agent amp", "amp", true},
		{"known agent goose", "goose", true},
		{"known agent factory", "factory", true},
		{"unknown agent", "unknown-agent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, ok := GetAgent(tt.agentName)
			if ok != tt.wantOK {
				t.Errorf("GetAgent(%q) ok = %v, want %v", tt.agentName, ok, tt.wantOK)
			}
			if tt.wantOK && agent == nil {
				t.Errorf("GetAgent(%q) returned nil agent", tt.agentName)
			}
			if !tt.wantOK && agent != nil {
				t.Errorf("GetAgent(%q) should return nil agent", tt.agentName)
			}
		})
	}
}

func TestListAgents(t *testing.T) {
	agents := ListAgents()
	if len(agents) == 0 {
		t.Error("ListAgents should return at least one agent")
	}

	expectedAgents := []string{
		"claude-code", "cursor", "github-copilot",
		"gemini-cli", "opencode", "openai-codex",
		"amp", "goose", "factory",
	}
	for _, expected := range expectedAgents {
		found := false
		for _, a := range agents {
			if a.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ListAgents should include %s", expected)
		}
	}
}

func TestLLMSetupCmd(t *testing.T) {
	cmd := &LLMSetupCmd{}

	// All subcommands should have default values
	subcommands := []struct {
		name string
		path string
		user bool
	}{
		{"ClaudeCode", cmd.ClaudeCode.Path, cmd.ClaudeCode.User},
		{"Cursor", cmd.Cursor.Path, cmd.Cursor.User},
		{"GithubCopilot", cmd.GithubCopilot.Path, cmd.GithubCopilot.User},
		{"GeminiCLI", cmd.GeminiCLI.Path, cmd.GeminiCLI.User},
		{"OpenCode", cmd.OpenCode.Path, cmd.OpenCode.User},
		{"OpenAICodex", cmd.OpenAICodex.Path, cmd.OpenAICodex.User},
		{"Amp", cmd.Amp.Path, cmd.Amp.User},
		{"Goose", cmd.Goose.Path, cmd.Goose.User},
		{"Factory", cmd.Factory.Path, cmd.Factory.User},
	}

	for _, sc := range subcommands {
		if sc.user != false {
			t.Errorf("%s.User should default to false", sc.name)
		}
		if sc.path != "" {
			t.Errorf("%s.Path should default to empty", sc.name)
		}
	}
}

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

	// Verify SKILL.md exists at root of skill directory
	skillMDPath := filepath.Join(skillDir, "SKILL.md")
	if _, statErr := os.Stat(skillMDPath); statErr != nil {
		t.Errorf("SKILL.md not found at %s: %v", skillMDPath, statErr)
	}

	// Verify references subdirectory exists
	referencesDir := filepath.Join(skillDir, "references")
	refInfo, err := os.Stat(referencesDir)
	if err != nil {
		t.Errorf("references directory not found: %v", err)
	} else if !refInfo.IsDir() {
		t.Error("references should be a directory")
	}

	// Verify reference files exist
	expectedRefFiles := []string{"PATTERNS.md", "MIGRATION.md", "TROUBLESHOOTING.md"}
	for _, refFile := range expectedRefFiles {
		refPath := filepath.Join(referencesDir, refFile)
		if _, err := os.Stat(refPath); err != nil {
			t.Errorf("reference file %s not found: %v", refFile, err)
		}
	}

	// Verify file contents match embedded FS
	skillsFS := agent.SkillsFS()
	srcDir := agent.SkillsSrcDir()

	// Check SKILL.md content
	expectedContent, _ := skillsFS.ReadFile(srcDir + "/SKILL.md")
	installedContent, _ := os.ReadFile(skillMDPath)
	if string(installedContent) != string(expectedContent) {
		t.Error("installed SKILL.md content does not match embedded content")
	}

	// Check a reference file content
	expectedPatterns, _ := skillsFS.ReadFile(srcDir + "/references/PATTERNS.md")
	installedPatterns, _ := os.ReadFile(filepath.Join(referencesDir, "PATTERNS.md"))
	if string(installedPatterns) != string(expectedPatterns) {
		t.Error("installed PATTERNS.md content does not match embedded content")
	}
}
