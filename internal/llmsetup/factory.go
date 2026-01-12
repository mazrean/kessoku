package llmsetup

import "embed"

// FactoryAgent implements the Agent interface for Factory.
type FactoryAgent struct{}

// Name returns the subcommand name.
func (a *FactoryAgent) Name() string {
	return "factory"
}

// Description returns the help text.
func (a *FactoryAgent) Description() string {
	return "Install Factory skills"
}

// SkillsFS returns the embedded skill files.
func (a *FactoryAgent) SkillsFS() embed.FS {
	return defaultSkillsFS
}

// SkillsSrcDir returns the source directory path in the embedded filesystem.
func (a *FactoryAgent) SkillsSrcDir() string {
	return "skills/kessoku-di"
}

// SkillsDirName returns the skill directory name for installation.
func (a *FactoryAgent) SkillsDirName() string {
	return "kessoku-di"
}

// ProjectSubPath returns the subpath for project-level installation.
func (a *FactoryAgent) ProjectSubPath() string {
	return ".factory/skills"
}

// UserSubPath returns the subpath for user-level installation.
func (a *FactoryAgent) UserSubPath() string {
	return ".factory/skills"
}
