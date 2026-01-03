package llmsetup

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
)

// LLMSetupCmd is the parent command for coding agent setup.
type LLMSetupCmd struct {
	Usage      UsageCmd      `kong:"cmd,default='1',hidden"`
	ClaudeCode ClaudeCodeCmd `kong:"cmd,name='claude-code',help='Install Claude Code skills'"`
}

// UsageCmd is a hidden default command that prints usage (FR-003).
type UsageCmd struct{}

// Run prints usage information and exits with code 0.
func (c *UsageCmd) Run(ctx *kong.Context) error {
	return ctx.PrintUsage(false)
}

// ClaudeCodeCmd is the subcommand for installing Claude Code Skills.
type ClaudeCodeCmd struct {
	// Stdout and Stderr for output (defaults to os.Stdout/os.Stderr)
	Stdout io.Writer `kong:"-"`
	Stderr io.Writer `kong:"-"`
	Path   string    `kong:"short='p',help='Custom installation directory'"`
	User   bool      `kong:"help='Install to user-level directory'"`
}

// Run installs the Claude Code Skills file.
func (c *ClaudeCodeCmd) Run() error {
	agent := &ClaudeCodeAgent{}

	installedPath, err := Install(agent, c.Path, c.User)
	if err != nil {
		c.writeError(err)
		return err
	}

	c.writeSuccess(installedPath)
	return nil
}

func (c *ClaudeCodeCmd) stdout() io.Writer {
	if c.Stdout != nil {
		return c.Stdout
	}
	return os.Stdout
}

func (c *ClaudeCodeCmd) stderr() io.Writer {
	if c.Stderr != nil {
		return c.Stderr
	}
	return os.Stderr
}

func (c *ClaudeCodeCmd) writeSuccess(path string) {
	_, _ = fmt.Fprintf(c.stdout(), "Skills installed to: %s\n", path)
}

func (c *ClaudeCodeCmd) writeError(err error) {
	_, _ = fmt.Fprintf(c.stderr(), "Error: %v\n", err)
}
