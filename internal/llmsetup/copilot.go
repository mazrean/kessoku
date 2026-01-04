package llmsetup

import "embed"

// CopilotAgent implements the Agent interface for GitHub Copilot.
type CopilotAgent struct{}

// Name returns the subcommand name.
func (a *CopilotAgent) Name() string {
	return "github-copilot"
}

// Description returns the help text.
func (a *CopilotAgent) Description() string {
	return "Install GitHub Copilot skills"
}

// SkillsFS returns the embedded skill files (shared with Claude Code).
func (a *CopilotAgent) SkillsFS() embed.FS {
	return claudeCodeSkillsFS
}

// SkillsSrcDir returns the source directory path in the embedded filesystem.
func (a *CopilotAgent) SkillsSrcDir() string {
	return "skills/kessoku-di"
}

// SkillsDirName returns the skill directory name for installation.
func (a *CopilotAgent) SkillsDirName() string {
	return "kessoku-di"
}

// ProjectSubPath returns the subpath for project-level installation.
func (a *CopilotAgent) ProjectSubPath() string {
	return ".github/skills"
}

// UserSubPath returns the subpath for user-level installation.
func (a *CopilotAgent) UserSubPath() string {
	return ".github/skills"
}
