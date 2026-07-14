package llmsetup

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
)

// LLMSetupCmd is the parent command for coding agent setup.
type LLMSetupCmd struct {
	Usage         UsageCmd      `kong:"cmd,default='1',hidden"`
	ClaudeCode    ClaudeCodeCmd `kong:"cmd,name='claude-code',help='Install Claude Code skills'"`
	Cursor        CursorCmd     `kong:"cmd,name='cursor',help='Install Cursor skills'"`
	GithubCopilot CopilotCmd    `kong:"cmd,name='github-copilot',help='Install GitHub Copilot skills'"`
	Amp           AmpCmd        `kong:"cmd,name='amp',help='Install Amp skills'"`
	Factory       FactoryCmd    `kong:"cmd,name='factory',help='Install Factory skills'"`
	GeminiCLI     GeminiCLICmd  `kong:"cmd,name='gemini-cli',help='Install Gemini CLI skills'"`
	Goose         GooseCmd      `kong:"cmd,name='goose',help='Install Goose skills'"`
	OpenAICodex   CodexCmd      `kong:"cmd,name='openai-codex',help='Install OpenAI Codex skills'"`
	OpenCode      OpenCodeCmd   `kong:"cmd,name='opencode',help='Install OpenCode skills'"`
}

// UsageCmd is a hidden default command that prints the llm-setup parent usage (FR-003).
type UsageCmd struct{}

// Run prints the llm-setup usage with the full agent subcommand list.
//
// ctx.Selected() points to this hidden "usage" leaf node when called, so we
// pop it from ctx.Path before calling PrintUsage so that ctx.Selected() returns
// the llm-setup parent node and the correct usage — including all agent
// subcommands — is displayed.
func (c *UsageCmd) Run(ctx *kong.Context) error {
	if len(ctx.Path) > 0 {
		ctx.Path = ctx.Path[:len(ctx.Path)-1]
	}
	return ctx.PrintUsage(false)
}

// AgentCmd is a generic command for installing agent skills.
type AgentCmd[T Agent] struct {
	// Stdout and Stderr for output (defaults to os.Stdout/os.Stderr)
	Stdout io.Writer `kong:"-"`
	Stderr io.Writer `kong:"-"`
	Path   string    `kong:"short='p',help='Custom installation directory'"`
	User   bool      `kong:"help='Install to user-level directory'"`
}

// Run installs the agent skills.
func (c *AgentCmd[T]) Run() error {
	var agent T

	installedPath, err := Install(agent, c.Path, c.User)
	if err != nil {
		return err
	}

	c.writeSuccess(installedPath)
	return nil
}

func (c *AgentCmd[T]) stdout() io.Writer {
	if c.Stdout != nil {
		return c.Stdout
	}
	return os.Stdout
}

func (c *AgentCmd[T]) stderr() io.Writer {
	if c.Stderr != nil {
		return c.Stderr
	}
	return os.Stderr
}

func (c *AgentCmd[T]) writeSuccess(path string) {
	_, _ = fmt.Fprintf(c.stdout(), "Skills installed to: %s\n", path)
}

// ClaudeCodeCmd is the subcommand for installing Claude Code Skills.
type ClaudeCodeCmd = AgentCmd[*ClaudeCodeAgent]

// CursorCmd is the subcommand for installing Cursor Skills.
type CursorCmd = AgentCmd[*CursorAgent]

// CopilotCmd is the subcommand for installing GitHub Copilot Skills.
type CopilotCmd = AgentCmd[*CopilotAgent]

// AmpCmd is the subcommand for installing Amp Skills.
type AmpCmd = AgentCmd[*AmpAgent]

// CodexCmd is the subcommand for installing OpenAI Codex Skills.
type CodexCmd = AgentCmd[*CodexAgent]

// FactoryCmd is the subcommand for installing Factory Skills.
type FactoryCmd = AgentCmd[*FactoryAgent]

// GeminiCLICmd is the subcommand for installing Gemini CLI Skills.
type GeminiCLICmd = AgentCmd[*GeminiCLIAgent]

// GooseCmd is the subcommand for installing Goose Skills.
type GooseCmd = AgentCmd[*GooseAgent]

// OpenCodeCmd is the subcommand for installing OpenCode Skills.
type OpenCodeCmd = AgentCmd[*OpenCodeAgent]
