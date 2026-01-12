# Data Model: Expand Agent Support

**Phase 1 Output** | **Date**: 2026-01-12

## Entities

### Agent (Interface)

Defines configuration for a coding agent that supports skill installation.

```go
type Agent interface {
    Name() string           // Subcommand name (e.g., "gemini-cli")
    Description() string    // Help text for CLI
    SkillsFS() embed.FS     // Embedded skill files
    SkillsSrcDir() string   // Source directory in embed.FS
    SkillsDirName() string  // Installation directory name
    ProjectSubPath() string // Project-level path (e.g., ".gemini/skills")
    UserSubPath() string    // User-level path (e.g., ".gemini/skills")
}
```

**Validation Rules**:
- `Name()` must be unique across all agents
- `Name()` must be lowercase with hyphens (CLI convention)
- `ProjectSubPath()` and `UserSubPath()` must not be empty
- `SkillsSrcDir()` must exist in `SkillsFS()`

### New Agent Implementations

| Struct Name | Name() | ProjectSubPath() | UserSubPath() |
|-------------|--------|------------------|---------------|
| `GeminiCLIAgent` | `gemini-cli` | `.gemini/skills` | `.gemini/skills` |
| `OpenCodeAgent` | `opencode` | `.opencode/skill` | `.config/opencode/skill` |
| `CodexAgent` | `openai-codex` | `.codex/skills` | `.codex/skills` |
| `AmpAgent` | `amp` | `.agents/skills` | `.config/agents/skills` |
| `GooseAgent` | `goose` | `.agents/skills` | `.config/goose/skills` |
| `FactoryAgent` | `factory` | `.factory/skills` | `.factory/skills` |

**Common Fields** (same for all new agents):
- `SkillsFS()`: Returns `claudeCodeSkillsFS` (shared)
- `SkillsSrcDir()`: Returns `"skills/kessoku-di"`
- `SkillsDirName()`: Returns `"kessoku-di"`
- `Description()`: Returns `"Install {AgentName} skills"`

### Agent Registry

Global slice maintaining ordered list of all supported agents.

```go
var agents = []Agent{
    // Existing
    &ClaudeCodeAgent{},
    &CursorAgent{},
    &CopilotAgent{},
    // New (alphabetical order within new additions)
    &AmpAgent{},
    &CodexAgent{},
    &FactoryAgent{},
    &GeminiCLIAgent{},
    &GooseAgent{},
    &OpenCodeAgent{},
}
```

**Invariants**:
- Order determines help display order
- All agents must be registered to be accessible via `GetAgent()` and `ListAgents()`

### CLI Command Structure

```go
type LLMSetupCmd struct {
    // Existing
    Usage         UsageCmd      `kong:"cmd,default='1',hidden"`
    ClaudeCode    ClaudeCodeCmd `kong:"cmd,name='claude-code',help='...'"`
    Cursor        CursorCmd     `kong:"cmd,name='cursor',help='...'"`
    GithubCopilot CopilotCmd    `kong:"cmd,name='github-copilot',help='...'"`
    // New
    Amp           AmpCmd        `kong:"cmd,name='amp',help='Install Amp skills'"`
    Factory       FactoryCmd    `kong:"cmd,name='factory',help='Install Factory skills'"`
    GeminiCLI     GeminiCLICmd  `kong:"cmd,name='gemini-cli',help='Install Gemini CLI skills'"`
    Goose         GooseCmd      `kong:"cmd,name='goose',help='Install Goose skills'"`
    OpenAICodex   CodexCmd      `kong:"cmd,name='openai-codex',help='Install OpenAI Codex skills'"`
    OpenCode      OpenCodeCmd   `kong:"cmd,name='opencode',help='Install OpenCode skills'"`
}
```

## Relationships

```
┌─────────────────────────────────────────────────────────────────┐
│                         LLMSetupCmd                             │
│  (kong parent command)                                          │
└─────────────────────────┬───────────────────────────────────────┘
                          │ has subcommands
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                    AgentCmd[T Agent]                            │
│  (generic command type)                                         │
│  - Path: string                                                 │
│  - User: bool                                                   │
└─────────────────────────┬───────────────────────────────────────┘
                          │ instantiated as
                          ▼
┌───────────────┬───────────────┬───────────────┬─────────────────┐
│ ClaudeCodeCmd │ CursorCmd     │ CopilotCmd    │ ... 7 new cmds  │
│ = AgentCmd    │ = AgentCmd    │ = AgentCmd    │                 │
│ [*Claude...]  │ [*Cursor...]  │ [*Copilot...] │                 │
└───────────────┴───────────────┴───────────────┴─────────────────┘
                          │ uses
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Agent interface                            │
│  Implemented by: ClaudeCodeAgent, CursorAgent, ..., FactoryAgent│
└─────────────────────────┬───────────────────────────────────────┘
                          │ shares
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                    claudeCodeSkillsFS                           │
│  (embed.FS containing skills/kessoku-di/*)                      │
└─────────────────────────────────────────────────────────────────┘
```

## State Transitions

N/A - This feature has no state management. Skill installation is idempotent.
