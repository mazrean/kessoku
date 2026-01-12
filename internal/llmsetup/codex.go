package llmsetup

import "embed"

// CodexAgent implements the Agent interface for OpenAI Codex.
type CodexAgent struct{}

// Name returns the subcommand name.
func (a *CodexAgent) Name() string {
	return "openai-codex"
}

// Description returns the help text.
func (a *CodexAgent) Description() string {
	return "Install OpenAI Codex skills"
}

// SkillsFS returns the embedded skill files.
func (a *CodexAgent) SkillsFS() embed.FS {
	return defaultSkillsFS
}

// SkillsSrcDir returns the source directory path in the embedded filesystem.
func (a *CodexAgent) SkillsSrcDir() string {
	return "skills/kessoku-di"
}

// SkillsDirName returns the skill directory name for installation.
func (a *CodexAgent) SkillsDirName() string {
	return "kessoku-di"
}

// ProjectSubPath returns the subpath for project-level installation.
func (a *CodexAgent) ProjectSubPath() string {
	return ".codex/skills"
}

// UserSubPath returns the subpath for user-level installation.
func (a *CodexAgent) UserSubPath() string {
	return ".codex/skills"
}
