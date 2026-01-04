package llmsetup

import "embed"

// CursorAgent implements the Agent interface for Cursor.
type CursorAgent struct{}

// Name returns the subcommand name.
func (a *CursorAgent) Name() string {
	return "cursor"
}

// Description returns the help text.
func (a *CursorAgent) Description() string {
	return "Install Cursor skills"
}

// SkillsFS returns the embedded skill files (shared with Claude Code).
func (a *CursorAgent) SkillsFS() embed.FS {
	return claudeCodeSkillsFS
}

// SkillsSrcDir returns the source directory path in the embedded filesystem.
func (a *CursorAgent) SkillsSrcDir() string {
	return "skills"
}

// SkillsDirName returns the skill directory name for installation.
func (a *CursorAgent) SkillsDirName() string {
	return "kessoku-di"
}

// ProjectSubPath returns the subpath for project-level installation.
func (a *CursorAgent) ProjectSubPath() string {
	return ".cursor/rules"
}

// UserSubPath returns the subpath for user-level installation.
func (a *CursorAgent) UserSubPath() string {
	return ".cursor/rules"
}
