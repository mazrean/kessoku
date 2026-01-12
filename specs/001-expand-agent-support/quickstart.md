# Quickstart: Expand Agent Support

**Phase 1 Output** | **Date**: 2026-01-12

## Overview

Add 6 new coding agent implementations to enable skill installation for Gemini CLI, OpenCode, OpenAI Codex, Amp, Goose, and Factory.

## Implementation Checklist

### Step 1: Create Agent Files

Create 7 new files in `internal/llmsetup/`:

```bash
# Template for each agent file
cat > internal/llmsetup/gemini.go << 'EOF'
package llmsetup

import "embed"

// GeminiCLIAgent implements the Agent interface for Gemini CLI.
type GeminiCLIAgent struct{}

func (a *GeminiCLIAgent) Name() string           { return "gemini-cli" }
func (a *GeminiCLIAgent) Description() string    { return "Install Gemini CLI skills" }
func (a *GeminiCLIAgent) SkillsFS() embed.FS     { return claudeCodeSkillsFS }
func (a *GeminiCLIAgent) SkillsSrcDir() string   { return "skills/kessoku-di" }
func (a *GeminiCLIAgent) SkillsDirName() string  { return "kessoku-di" }
func (a *GeminiCLIAgent) ProjectSubPath() string { return ".gemini/skills" }
func (a *GeminiCLIAgent) UserSubPath() string    { return ".gemini/skills" }
EOF
```

### Step 2: Agent Configuration Reference

| File | Struct | Name | Project Path | User Path |
|------|--------|------|--------------|-----------|
| `gemini.go` | `GeminiCLIAgent` | `gemini-cli` | `.gemini/skills` | `.gemini/skills` |
| `opencode.go` | `OpenCodeAgent` | `opencode` | `.opencode/skill` | `.config/opencode/skill` |
| `codex.go` | `CodexAgent` | `openai-codex` | `.codex/skills` | `.codex/skills` |
| `amp.go` | `AmpAgent` | `amp` | `.agents/skills` | `.config/agents/skills` |
| `goose.go` | `GooseAgent` | `goose` | `.agents/skills` | `.config/goose/skills` |
| `factory.go` | `FactoryAgent` | `factory` | `.factory/skills` | `.factory/skills` |

### Step 3: Register Agents

Update `internal/llmsetup/agent.go`:

```go
var agents = []Agent{
    &ClaudeCodeAgent{},
    &CursorAgent{},
    &CopilotAgent{},
    // Add new agents (alphabetical)
    &AmpAgent{},
    &CodexAgent{},
    &FactoryAgent{},
    &GeminiCLIAgent{},
    &GooseAgent{},
    &OpenCodeAgent{},
}
```

### Step 4: Add CLI Commands

Update `internal/llmsetup/llmsetup.go`:

```go
// Add type aliases
type AmpCmd = AgentCmd[*AmpAgent]
type CodexCmd = AgentCmd[*CodexAgent]
type FactoryCmd = AgentCmd[*FactoryAgent]
type GeminiCLICmd = AgentCmd[*GeminiCLIAgent]
type GooseCmd = AgentCmd[*GooseAgent]
type OpenCodeCmd = AgentCmd[*OpenCodeAgent]

// Add fields to LLMSetupCmd struct
type LLMSetupCmd struct {
    // ... existing fields ...
    Amp         AmpCmd       `kong:"cmd,name='amp',help='Install Amp skills'"`
    Factory     FactoryCmd   `kong:"cmd,name='factory',help='Install Factory skills'"`
    GeminiCLI   GeminiCLICmd `kong:"cmd,name='gemini-cli',help='Install Gemini CLI skills'"`
    Goose       GooseCmd     `kong:"cmd,name='goose',help='Install Goose skills'"`
    OpenAICodex CodexCmd     `kong:"cmd,name='openai-codex',help='Install OpenAI Codex skills'"`
    OpenCode    OpenCodeCmd  `kong:"cmd,name='opencode',help='Install OpenCode skills'"`
}
```

### Step 5: Update Tests

Extend `TestAgents` table in `internal/llmsetup/llmsetup_test.go`:

```go
{
    name:           "GeminiCLIAgent",
    agent:          &GeminiCLIAgent{},
    wantName:       "gemini-cli",
    projectSubPath: ".gemini/skills",
    userSubPath:    ".gemini/skills",
},
// ... repeat for each new agent
```

Add new tests:
- `TestGeminiCLICmd`, `TestOpenCodeCmd`, etc. (using `testAgentCmd` helper)
- Update `TestGetAgent` to include new agent names
- Update `TestListAgents` expected agents slice
- Update `TestLLMSetupCmd` subcommands check

## Verification Commands

```bash
# Run all tests
go test -v ./internal/llmsetup/...

# Verify CLI help shows new agents
go tool kessoku --help

# Test installation for each new agent
go tool kessoku gemini-cli --project -p /tmp/test-gemini
go tool kessoku opencode --project -p /tmp/test-opencode
go tool kessoku openai-codex --project -p /tmp/test-codex
go tool kessoku amp --project -p /tmp/test-amp
go tool kessoku goose --project -p /tmp/test-goose
go tool kessoku factory --project -p /tmp/test-factory

# Lint
go tool lint ./...
```

## Success Criteria Verification

- [ ] `go tool kessoku --help` shows all 10 agents
- [ ] `go test -v ./internal/llmsetup/...` passes
- [ ] Each agent installs skills to correct project directory
- [ ] Each agent installs skills to correct user directory
- [ ] `go tool lint ./...` passes
