package llmsetup

import "embed"

// AmpAgent implements the Agent interface for Amp.
type AmpAgent struct{}

// Name returns the subcommand name.
func (a *AmpAgent) Name() string {
	return "amp"
}

// Description returns the help text.
func (a *AmpAgent) Description() string {
	return "Install Amp skills"
}

// SkillsFS returns the embedded skill files.
func (a *AmpAgent) SkillsFS() embed.FS {
	return claudeCodeSkillsFS
}

// SkillsSrcDir returns the source directory path in the embedded filesystem.
func (a *AmpAgent) SkillsSrcDir() string {
	return "skills/kessoku-di"
}

// SkillsDirName returns the skill directory name for installation.
func (a *AmpAgent) SkillsDirName() string {
	return "kessoku-di"
}

// ProjectSubPath returns the subpath for project-level installation.
func (a *AmpAgent) ProjectSubPath() string {
	return ".agents/skills"
}

// UserSubPath returns the subpath for user-level installation.
func (a *AmpAgent) UserSubPath() string {
	return ".config/agents/skills"
}
