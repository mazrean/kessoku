package llmsetup

import "embed"

// GooseAgent implements the Agent interface for Goose.
type GooseAgent struct{}

// Name returns the subcommand name.
func (a *GooseAgent) Name() string {
	return "goose"
}

// Description returns the help text.
func (a *GooseAgent) Description() string {
	return "Install Goose skills"
}

// SkillsFS returns the embedded skill files.
func (a *GooseAgent) SkillsFS() embed.FS {
	return defaultSkillsFS
}

// SkillsSrcDir returns the source directory path in the embedded filesystem.
func (a *GooseAgent) SkillsSrcDir() string {
	return "skills/kessoku-di"
}

// SkillsDirName returns the skill directory name for installation.
func (a *GooseAgent) SkillsDirName() string {
	return "kessoku-di"
}

// ProjectSubPath returns the subpath for project-level installation.
func (a *GooseAgent) ProjectSubPath() string {
	return ".agents/skills"
}

// UserSubPath returns the subpath for user-level installation.
func (a *GooseAgent) UserSubPath() string {
	return ".config/goose/skills"
}
