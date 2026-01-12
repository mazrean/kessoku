package llmsetup

import "embed"

// OpenCodeAgent implements the Agent interface for OpenCode.
type OpenCodeAgent struct{}

// Name returns the subcommand name.
func (a *OpenCodeAgent) Name() string {
	return "opencode"
}

// Description returns the help text.
func (a *OpenCodeAgent) Description() string {
	return "Install OpenCode skills"
}

// SkillsFS returns the embedded skill files.
func (a *OpenCodeAgent) SkillsFS() embed.FS {
	return claudeCodeSkillsFS
}

// SkillsSrcDir returns the source directory path in the embedded filesystem.
func (a *OpenCodeAgent) SkillsSrcDir() string {
	return "skills/kessoku-di"
}

// SkillsDirName returns the skill directory name for installation.
func (a *OpenCodeAgent) SkillsDirName() string {
	return "kessoku-di"
}

// ProjectSubPath returns the subpath for project-level installation.
func (a *OpenCodeAgent) ProjectSubPath() string {
	return ".opencode/skill"
}

// UserSubPath returns the subpath for user-level installation.
func (a *OpenCodeAgent) UserSubPath() string {
	return ".config/opencode/skill"
}
