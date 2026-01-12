# Research: Expand Agent Support

**Phase 0 Output** | **Date**: 2026-01-12

## Summary

Research to verify agent skill directory paths and implementation patterns for 7 new coding agents.

## Decisions

### 1. Agent Implementation Pattern

**Decision**: Follow existing single-file-per-agent pattern
**Rationale**:
- Existing agents (Claude Code, Cursor, Copilot) each have dedicated files
- Pattern is proven and well-tested
- Maintains code organization consistency
- Each agent struct is small (~40 lines), making separate files appropriate

**Alternatives Considered**:
- Single file with all agents: Rejected (would become too large with 10 agents)
- Config-driven agents: Rejected (adds unnecessary complexity for static configuration)

### 2. Skill File Sharing

**Decision**: All agents share the same embedded `claudeCodeSkillsFS`
**Rationale**:
- Existing pattern already shares embed.FS across agents
- No agent-specific skill customization required per spec
- Reduces binary size by avoiding duplicate embeddings

**Alternatives Considered**:
- Separate embed.FS per agent: Rejected (wasteful, no benefit)
- Agent-specific SKILL.md files: Out of scope per spec

### 3. Agent Skill Directory Paths

Paths verified from official documentation (see spec.md Path Verification Status):

| Agent | Project Path | User Path | Source |
|-------|-------------|-----------|--------|
| Gemini CLI | `.gemini/skills/` | `~/.gemini/skills/` | [Gemini CLI Docs](https://geminicli.com/docs/cli/skills/) |
| OpenCode | `.opencode/skill/` | `~/.config/opencode/skill/` | [OpenCode Docs](https://opencode.ai/docs/skills/) |
| OpenAI Codex | `.codex/skills/` | `~/.codex/skills/` | [Codex Docs](https://developers.openai.com/codex/skills) |
| Amp | `.agents/skills/` | `~/.config/agents/skills/` | [Amp Docs](https://ampcode.com/news/agent-skills) |
| Goose | `.agents/skills/` | `~/.config/goose/skills/` | [Goose GitHub](https://github.com/block/goose) |
| Factory | `.factory/skills/` | `~/.factory/skills/` | [Factory Docs](https://docs.factory.ai/cli/configuration/skills) |

### 4. Shared Project-Level Directory Handling

**Decision**: Amp and Goose share `.agents/skills/` at project level but have distinct user paths
**Rationale**:
- Both agents follow the Agent Skills standard convention for project-level
- User-level directories remain distinct for configuration isolation
- No special handling needed - existing implementation supports this naturally

### 5. CLI Subcommand Naming

**Decision**: Use lowercase hyphenated names matching agent branding

| Agent | Subcommand |
|-------|------------|
| Gemini CLI | `gemini-cli` |
| OpenCode | `opencode` |
| OpenAI Codex | `openai-codex` |
| Amp | `amp` |
| Goose | `goose` |
| Factory | `factory` |

**Rationale**: Consistent with existing agents (`claude-code`, `github-copilot`)

## Open Questions Resolved

All technical questions from spec clarification phase have been resolved:

1. **Shared directories**: No special handling needed; works with existing implementation
2. **Skill file customization**: Out of scope; all agents share same files

## Implementation Notes

1. **File structure**: 6 new files in `internal/llmsetup/`
2. **Registry update**: Add 6 new agents to `agents` slice in `agent.go`
3. **CLI update**: Add 6 new type aliases and fields in `llmsetup.go`
4. **Test update**: Extend table-driven tests to cover new agents
