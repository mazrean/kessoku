package llmsetup

import "embed"

// GeminiCLIAgent implements the Agent interface for Gemini CLI.
type GeminiCLIAgent struct{}

// Name returns the subcommand name.
func (a *GeminiCLIAgent) Name() string {
	return "gemini-cli"
}

// Description returns the help text.
func (a *GeminiCLIAgent) Description() string {
	return "Install Gemini CLI skills"
}

// SkillsFS returns the embedded skill files.
func (a *GeminiCLIAgent) SkillsFS() embed.FS {
	return claudeCodeSkillsFS
}

// SkillsSrcDir returns the source directory path in the embedded filesystem.
func (a *GeminiCLIAgent) SkillsSrcDir() string {
	return "skills/kessoku-di"
}

// SkillsDirName returns the skill directory name for installation.
func (a *GeminiCLIAgent) SkillsDirName() string {
	return "kessoku-di"
}

// ProjectSubPath returns the subpath for project-level installation.
func (a *GeminiCLIAgent) ProjectSubPath() string {
	return ".gemini/skills"
}

// UserSubPath returns the subpath for user-level installation.
func (a *GeminiCLIAgent) UserSubPath() string {
	return ".gemini/skills"
}
