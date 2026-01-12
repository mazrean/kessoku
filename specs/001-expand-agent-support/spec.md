# Feature Specification: Expand Agent Support

**Feature Branch**: `001-expand-agent-support`
**Created**: 2026-01-12
**Status**: Ready for Planning
**Input**: User description: "Expand agent.go to support all Coding Agents listed on agentskills.io"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Install Skills for Any Supported Agent (Priority: P1)

Users want to install kessoku-di skills for their preferred coding agent, regardless of which tool they use. Currently, only Claude Code, Cursor, and GitHub Copilot are supported. The agentskills.io adoption section lists 12 tools, but some share implementations (VS Code/GitHub use Copilot, Claude Desktop uses Claude Code paths), resulting in 9 distinct agent configurations.

**Why this priority**: This is the core value proposition - enabling all users of agentskills.io-compatible tools to benefit from kessoku-di skills.

**Independent Test**: Can be fully tested by running `go tool kessoku <agent-name> --project` for each new agent and verifying the skill files are installed to the correct directory.

**Acceptance Scenarios**:

1. **Given** a user with Gemini CLI installed, **When** they run `go tool kessoku gemini-cli --project`, **Then** skills are installed to `.gemini/skills/kessoku-di/`
2. **Given** a user with OpenCode installed, **When** they run `go tool kessoku opencode --project`, **Then** skills are installed to `.opencode/skill/kessoku-di/`
3. **Given** a user with OpenAI Codex installed, **When** they run `go tool kessoku openai-codex --project`, **Then** skills are installed to `.codex/skills/kessoku-di/`
4. **Given** a user with Amp installed, **When** they run `go tool kessoku amp --project`, **Then** skills are installed to `.agents/skills/kessoku-di/`
5. **Given** a user with Goose installed, **When** they run `go tool kessoku goose --project`, **Then** skills are installed to `.agents/skills/kessoku-di/`
6. **Given** a user with Factory installed, **When** they run `go tool kessoku factory --project`, **Then** skills are installed to `.factory/skills/kessoku-di/`
7. **Given** skills already exist in the target directory, **When** a user runs any agent install command, **Then** existing files are silently overwritten with the new version
8. **Given** the target directory does not exist, **When** a user runs any agent install command, **Then** the directory is created automatically with permissions 0755

---

### User Story 2 - Install Skills at User Level for New Agents (Priority: P2)

Users want to install kessoku-di skills at the user level so they're available across all projects without per-project installation.

**Why this priority**: User-level installation is a convenience feature that follows naturally from project-level support.

**Independent Test**: Can be fully tested by running `go tool kessoku <agent-name> --user` for each new agent and verifying files are installed to the correct user-level directory.

**Acceptance Scenarios**:

1. **Given** a user wants user-level skills, **When** they run `go tool kessoku gemini-cli --user`, **Then** skills are installed to `~/.gemini/skills/kessoku-di/`
2. **Given** a user wants user-level skills, **When** they run `go tool kessoku opencode --user`, **Then** skills are installed to `~/.config/opencode/skill/kessoku-di/`
3. **Given** a user wants user-level skills, **When** they run `go tool kessoku openai-codex --user`, **Then** skills are installed to `~/.codex/skills/kessoku-di/`
4. **Given** a user wants user-level skills, **When** they run `go tool kessoku amp --user`, **Then** skills are installed to `~/.config/agents/skills/kessoku-di/`
5. **Given** a user wants user-level skills, **When** they run `go tool kessoku goose --user`, **Then** skills are installed to `~/.config/goose/skills/kessoku-di/`
6. **Given** a user wants user-level skills, **When** they run `go tool kessoku factory --user`, **Then** skills are installed to `~/.factory/skills/kessoku-di/`
7. **Given** the user-level config directory does not exist, **When** a user runs `--user` install, **Then** the directory is created automatically

---

### User Story 3 - Discover Available Agents (Priority: P3)

Users want to see which agents are supported and choose the appropriate one for their setup.

**Why this priority**: Discoverability enhances user experience but is not essential for core functionality.

**Independent Test**: Can be tested by running `go tool kessoku --help` and verifying all new agents appear in the subcommand list.

**Acceptance Scenarios**:

1. **Given** a user runs `go tool kessoku --help`, **When** viewing the output, **Then** all 9 supported agents are listed (`claude-code`, `cursor`, `github-copilot`, `gemini-cli`, `opencode`, `openai-codex`, `amp`, `goose`, `factory`)
2. **Given** a user runs `go tool kessoku --help`, **When** viewing the output, **Then** each agent has a unique subcommand name with no duplicates
3. **Given** a user runs `go tool kessoku --help`, **When** viewing each agent's entry, **Then** each agent displays a descriptive help text (e.g., "Install Gemini CLI skills")

---

### User Story 4 - Default Installation Behavior (Priority: P1)

Users need clear, predictable behavior when running agent commands without explicit flags.

**Why this priority**: Essential for usability - users must know what happens by default.

**Independent Test**: Can be tested by running `go tool kessoku <agent-name>` without flags and verifying default behavior.

**Acceptance Scenarios**:

1. **Given** a user runs `go tool kessoku gemini-cli` without `--project` or `--user`, **When** the command executes, **Then** skills are installed to the project-level directory (`.gemini/skills/kessoku-di/`) by default
2. **Given** a user runs any agent command without flags, **When** the command executes, **Then** installation defaults to project-level (same as `--project`)
3. **Given** a user runs `go tool kessoku unknown-agent`, **When** the command executes, **Then** the CLI displays an error message listing valid agent names and exits with non-zero status

---

### User Story 5 - Shared Directory Behavior (Priority: P2)

Users of Amp and Goose need predictable behavior when both agents target the same project-level directory.

**Why this priority**: Prevents confusion when multiple agents share paths.

**Independent Test**: Can be tested by running both `amp --project` and `goose --project` in sequence.

**Acceptance Scenarios**:

1. **Given** a user runs `go tool kessoku amp --project`, **When** they subsequently run `go tool kessoku goose --project`, **Then** skills are overwritten (idempotent, same content)
2. **Given** Amp and Goose share `.agents/skills/` at project level, **When** either agent installs, **Then** the resulting `kessoku-di/` directory contains identical files regardless of which agent installed last
3. **Given** skills were installed by Amp, **When** Goose installs to the same directory, **Then** no error occurs and the operation completes successfully

---

### Edge Cases

- What happens when a user tries to install to an existing skill directory? → **FR-013** (silent overwrite)
- How does the system handle agents with shared project-level skill directories? → **User Story 5** (idempotent overwrite)
- What happens if the user-level config directory doesn't exist? → **FR-014** (auto-create)
- What happens when no `--project` or `--user` flag is provided? → **FR-015** (default to project)
- What happens with an unknown agent name? → **FR-016** (error with valid names)

## Requirements *(mandatory)*

### Agent Configuration Table

All 9 supported agents with their canonical subcommand names and paths:

| Subcommand | Project Path | User Path | Source | Status |
|------------|--------------|-----------|--------|--------|
| `claude-code` | `.claude/skills/` | `~/.claude/skills/` | [Claude Code Docs](https://code.claude.com/docs/en/skills) | Existing |
| `cursor` | `.cursor/rules/` | `~/.cursor/rules/` | [Cursor Docs](https://docs.cursor.com/context/rules) | Existing |
| `github-copilot` | `.github/skills/` | `~/.github/skills/` | [GitHub Copilot Changelog](https://github.blog/changelog/2025-12-18-github-copilot-now-supports-agent-skills/) | Existing |
| `gemini-cli` | `.gemini/skills/` | `~/.gemini/skills/` | [Gemini CLI Docs](https://geminicli.com/docs/cli/skills/) | **New** |
| `opencode` | `.opencode/skill/` | `~/.config/opencode/skill/` | [OpenCode Docs](https://opencode.ai/docs/skills/) | **New** |
| `openai-codex` | `.codex/skills/` | `~/.codex/skills/` | [Codex Docs](https://developers.openai.com/codex/skills) | **New** |
| `amp` | `.agents/skills/` | `~/.config/agents/skills/` | [Amp Docs](https://ampcode.com/news/agent-skills) | **New** |
| `goose` | `.agents/skills/` | `~/.config/goose/skills/` | [Goose GitHub](https://github.com/block/goose) | **New** |
| `factory` | `.factory/skills/` | `~/.factory/skills/` | [Factory Docs](https://docs.factory.ai/cli/configuration/skills) | **New** |

### Functional Requirements

#### Agent Support

- **FR-001**: System MUST support `gemini-cli` subcommand with project path `.gemini/skills/` and user path `~/.gemini/skills/`
- **FR-002**: System MUST support `opencode` subcommand with project path `.opencode/skill/` and user path `~/.config/opencode/skill/`
- **FR-003**: System MUST support `openai-codex` subcommand with project path `.codex/skills/` and user path `~/.codex/skills/`
- **FR-004**: System MUST support `amp` subcommand with project path `.agents/skills/` and user path `~/.config/agents/skills/`
- **FR-005**: System MUST support `goose` subcommand with project path `.agents/skills/` and user path `~/.config/goose/skills/`
- **FR-006**: System MUST support `factory` subcommand with project path `.factory/skills/` and user path `~/.factory/skills/`

#### Registry and Discovery

- **FR-007**: All new agents MUST be registered in the agents registry and returned by `ListAgents()`
- **FR-008**: All new agents MUST be discoverable via `GetAgent(name)` using their exact subcommand names from the Agent Configuration Table
- **FR-009**: All new agents MUST share the same embedded skill files (`skills/kessoku-di`) as existing agents
- **FR-010**: Each agent MUST have a unique subcommand name for CLI invocation (no duplicates allowed)
- **FR-011**: Each agent MUST have a descriptive help text in format "Install {AgentName} skills"
- **FR-012**: `GetAgent(name)` MUST return `(nil, false)` for unknown agent names

#### Installation Behavior

- **FR-013**: System MUST silently overwrite existing skill files when reinstalling to an existing directory (idempotent operation, no user confirmation required)
- **FR-014**: System MUST automatically create target directories (and parent directories) with permissions 0755 if they do not exist
- **FR-015**: When neither `--project` nor `--user` flag is provided, system MUST default to project-level installation (equivalent to `--project`)
- **FR-016**: When an unknown agent name is provided, CLI MUST display an error message listing all valid agent names and exit with non-zero status code

### Key Entities

- **Agent**: A coding tool that supports the Agent Skills standard. Each agent has a unique subcommand name (per Agent Configuration Table), skill directory paths, and shares the embedded skill files.
- **Skill Directory**: The filesystem location where agent skills are stored. Each agent has both project-level and user-level directories as defined in the Agent Configuration Table.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All 9 agents (3 existing + 6 new) are accessible via `go tool kessoku <agent-name>` commands using exact subcommand names from Agent Configuration Table
- **SC-002**: Running `go tool kessoku --help` displays all 9 agents with their descriptions
- **SC-003**: Skills can be installed to correct project-level directories for all agents (per Agent Configuration Table)
- **SC-004**: Skills can be installed to correct user-level directories for all agents (per Agent Configuration Table)
- **SC-005**: All existing tests continue to pass after adding new agents
- **SC-006**: `GetAgent(name)` returns the correct agent for all 9 agent subcommand names
- **SC-007**: Running `go tool kessoku <agent>` without flags installs to project-level directory
- **SC-008**: Running `go tool kessoku unknown-agent` displays error listing valid agents and exits non-zero
- **SC-009**: Running `go tool kessoku amp --project` followed by `go tool kessoku goose --project` in same directory succeeds with identical resulting files (shared directory idempotence)

## Assumptions

- All agents use the same embedded skill files (no agent-specific customization needed)
- The existing Agent interface provides all necessary methods for new agents
- VS Code and GitHub are covered by the existing GitHub Copilot agent implementation
- Claude Desktop uses the same paths as Claude Code

## Path Verification Status

### Existing Agents (already implemented)

- **Claude Code**: `.claude/skills/` and `~/.claude/skills/` - [Claude Code Docs](https://code.claude.com/docs/en/skills)
- **Cursor**: `.cursor/rules/` and `~/.cursor/rules/` - [Cursor Docs](https://docs.cursor.com/context/rules)
- **GitHub Copilot**: `.github/skills/` and `~/.github/skills/` - [GitHub Copilot Changelog](https://github.blog/changelog/2025-12-18-github-copilot-now-supports-agent-skills/)

### New Agents - Fully Verified (project + user paths from official docs)

- **Gemini CLI**: `.gemini/skills/` and `~/.gemini/skills/` - [Gemini CLI Docs](https://geminicli.com/docs/cli/skills/)
- **OpenCode**: `.opencode/skill/` and `~/.config/opencode/skill/` - [OpenCode Docs](https://opencode.ai/docs/skills/)
- **OpenAI Codex**: `.codex/skills/` and `~/.codex/skills/` - [Codex Docs](https://developers.openai.com/codex/skills)
- **Amp**: `.agents/skills/` and `~/.config/agents/skills/` - [Amp Docs](https://ampcode.com/news/agent-skills)
- **Goose**: `.agents/skills/` and `~/.config/goose/skills/` - [Goose GitHub](https://github.com/block/goose)
- **Factory**: `.factory/skills/` and `~/.factory/skills/` - [Factory Docs](https://docs.factory.ai/cli/configuration/skills)

## Out of Scope

- Agent-specific skill file customization (all agents share the same SKILL.md)
- New skill file formats (e.g., Factory's AGENTS.md format)
- Auto-detection of installed agents
- Multiple skill directories per agent (some agents support fallback paths)
- **Windows-specific path handling**: This feature targets POSIX systems (Linux, macOS). Windows support is explicitly out of scope. Paths like `~/.config/` are resolved via Go's `os.UserHomeDir()` which returns appropriate home directory per platform, but Windows-specific paths (e.g., `%APPDATA%`) are not supported.
- Custom path resolution for XDG_CONFIG_HOME or similar environment variables
- **Letta**: Excluded due to unverified user-level path. Letta documentation only confirms project path `.skills/` but does not specify a default user-level directory.
