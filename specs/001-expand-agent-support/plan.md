# Implementation Plan: Expand Agent Support

**Branch**: `001-expand-agent-support` | **Date**: 2026-01-12 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-expand-agent-support/spec.md`

## Summary

Expand the existing agent support in `internal/llmsetup/` from 3 agents (Claude Code, Cursor, GitHub Copilot) to 10 agents by adding 6 new coding agents (Gemini CLI, OpenCode, OpenAI Codex, Amp, Goose, Factory). Each agent follows the established pattern: a struct implementing the `Agent` interface with project-level and user-level skill directory paths, sharing the same embedded skill files.

## Technical Context

**Language/Version**: Go 1.24+
**Primary Dependencies**: github.com/alecthomas/kong (CLI framework)
**Storage**: N/A (file-based skill installation)
**Testing**: Standard Go testing (`go test`)
**Target Platform**: Linux, macOS, Windows (cross-platform CLI)
**Project Type**: Single project
**Performance Goals**: N/A (one-time installation tool)
**Constraints**: Must maintain backward compatibility with existing agents
**Scale/Scope**: 7 new agent implementations following established pattern

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution is not yet defined for this project (template placeholder). Proceeding without gate violations.

## Project Structure

### Documentation (this feature)

```text
specs/001-expand-agent-support/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
internal/llmsetup/
├── agent.go             # Agent interface + registry (modify: add new agents)
├── claudecode.go        # Existing: ClaudeCodeAgent
├── cursor.go            # Existing: CursorAgent
├── copilot.go           # Existing: CopilotAgent
├── gemini.go            # NEW: GeminiCLIAgent
├── opencode.go          # NEW: OpenCodeAgent
├── codex.go             # NEW: CodexAgent (OpenAI Codex)
├── amp.go               # NEW: AmpAgent
├── goose.go             # NEW: GooseAgent
├── factory.go           # NEW: FactoryAgent
├── llmsetup.go          # LLMSetupCmd (modify: add new subcommands)
├── llmsetup_test.go     # Tests (modify: add new agent tests)
├── install.go           # Install function (no changes needed)
├── install_test.go      # Install tests (no changes needed)
├── path.go              # Path resolution (no changes needed)
├── path_test.go         # Path tests (no changes needed)
└── skills/kessoku-di/   # Embedded skill files (shared by all agents)
```

**Structure Decision**: Follow existing single-file-per-agent pattern. All 7 new agents are added as separate files in `internal/llmsetup/`, registered in the `agents` slice in `agent.go`, and exposed via CLI in `llmsetup.go`.

## Complexity Tracking

No constitution violations to justify.
