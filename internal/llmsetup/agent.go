// Package llmsetup provides functionality for setting up coding agent Skills.
package llmsetup

import "embed"

// Agent defines configuration for a coding agent.
type Agent interface {
	Name() string           // Subcommand name (e.g., "claude-code")
	Description() string    // Help text
	SkillsFS() embed.FS     // Embedded skill files (directory with SKILL.md and support files)
	SkillsSrcDir() string   // Source directory path in embed.FS (e.g., "skills/kessoku-di")
	SkillsDirName() string  // Skill directory name for installation (e.g., "kessoku-di")
	ProjectSubPath() string // Subpath for project-level (e.g., ".claude/skills")
	UserSubPath() string    // Subpath for user-level (e.g., ".claude/skills")
}

// agents is the registry of all supported coding agents.
var agents = []Agent{
	&ClaudeCodeAgent{},
	&CursorAgent{},
	&CopilotAgent{},
	&AmpAgent{},
	&CodexAgent{},
	&FactoryAgent{},
	&GeminiCLIAgent{},
	&GooseAgent{},
	&OpenCodeAgent{},
}

// GetAgent returns the agent with the given name, if it exists.
func GetAgent(name string) (Agent, bool) {
	for _, a := range agents {
		if a.Name() == name {
			return a, true
		}
	}
	return nil, false
}

// ListAgents returns all registered agents in a consistent order.
func ListAgents() []Agent {
	return agents
}
