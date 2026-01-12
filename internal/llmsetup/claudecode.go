package llmsetup

import "embed"

// ClaudeCodeAgent implements the Agent interface for Claude Code.
type ClaudeCodeAgent struct{}

// Name returns the subcommand name.
func (a *ClaudeCodeAgent) Name() string {
	return "claude-code"
}

// Description returns the help text.
func (a *ClaudeCodeAgent) Description() string {
	return "Install Claude Code skills"
}

// SkillsFS returns the embedded skill files.
func (a *ClaudeCodeAgent) SkillsFS() embed.FS {
	return defaultSkillsFS
}

// SkillsSrcDir returns the source directory path in the embedded filesystem.
func (a *ClaudeCodeAgent) SkillsSrcDir() string {
	return "skills/kessoku-di"
}

// SkillsDirName returns the skill directory name for installation.
func (a *ClaudeCodeAgent) SkillsDirName() string {
	return "kessoku-di"
}

// ProjectSubPath returns the subpath for project-level installation.
func (a *ClaudeCodeAgent) ProjectSubPath() string {
	return ".claude/skills"
}

// UserSubPath returns the subpath for user-level installation.
func (a *ClaudeCodeAgent) UserSubPath() string {
	return ".claude/skills"
}
