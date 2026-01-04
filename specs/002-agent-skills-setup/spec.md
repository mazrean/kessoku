# Feature Specification: Agent Skills Setup Subcommand

**Feature Branch**: `002-agent-skills-setup`
**Created**: 2026-01-03
**Status**: Draft
**Input**: User description: "kessokuの使い方をClaude CodeのSkillsとして提供する機能を作成して。ただし、この際今後の拡張性を考え、指定したコーディングエージェント用の設定を行うサブコマンドとして提供する形にして。また、Skillsのファイルはgo:embedでバイナリに埋め込まれるようにして、サブコマンドが実行された際にSkillsの設定ディレクトリへ展開するようにして。"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Install Claude Code Skills (Project-Level) (Priority: P1)

A developer using kessoku wants to install kessoku usage instructions as Claude Code Skills at the project level so that Claude Code can assist them with kessoku-specific tasks like creating providers, setting up dependency injection, and using the migrate command.

**Why this priority**: This is the core functionality requested. Without this, the feature has no value. Claude Code is the primary target agent. Project-level is the default for team collaboration.

**Independent Test**: Can be fully tested by running the llm-setup subcommand and verifying that the Skills file (`kessoku.md`) appears in the project-level skills directory with correct content.

**Acceptance Scenarios**:

1. **Given** a user has kessoku installed in a project, **When** they run `kessoku llm-setup claude-code`, **Then** the `kessoku.md` Skills file is installed to `./.claude/skills/kessoku.md` (project-level) and the command exits with code 0.
2. **Given** Skills are already installed, **When** the user runs `kessoku llm-setup claude-code` again, **Then** the existing `kessoku.md` is atomically replaced with the latest version and the command exits with code 0.
3. **Given** a user runs the llm-setup command, **When** the installation completes successfully, **Then** a success message showing the installation path is printed to stdout.
4. **Given** a user runs the llm-setup command, **When** the installation fails, **Then** an error message is printed to stderr and the command exits with a non-zero exit code.

---

### User Story 2 - Install Claude Code Skills (User-Level) (Priority: P2)

A developer wants to install kessoku Skills at the user level so they are available across all projects without per-project setup.

**Why this priority**: Secondary to project-level, but important for users who prefer global configuration.

**Independent Test**: Can be tested by running `kessoku llm-setup claude-code --user` and verifying the file is installed to the user-level skills directory.

**Acceptance Scenarios**:

1. **Given** a user wants user-level skills, **When** they run `kessoku llm-setup claude-code --user`, **Then** `kessoku.md` is installed to `~/.claude/skills/kessoku.md` (user-level) instead of project-level.
2. **Given** user-level skills are already installed, **When** the user runs `kessoku llm-setup claude-code --user` again, **Then** the existing file is atomically replaced.

---

### User Story 3 - List Available Agents (Priority: P2)

A developer wants to see which coding agents are supported by kessoku's llm-setup command so they can choose the appropriate one for their workflow.

**Why this priority**: Helps discoverability of supported agents, but secondary to actual installation functionality.

**Independent Test**: Can be tested by running `kessoku llm-setup --help` and verifying the list of available agent subcommands is displayed.

**Acceptance Scenarios**:

1. **Given** a user runs `kessoku llm-setup --help`, **When** the help is displayed, **Then** a list of available agent subcommands (e.g., `claude-code`) is shown with descriptions.
2. **Given** a user runs `kessoku llm-setup` without specifying an agent, **When** the command executes, **Then** usage information with available agents is displayed and the command exits with code 0 (not an error).

---

### User Story 4 - Custom Installation Path (Priority: P3)

A developer wants to install Skills to a custom location instead of the default skills directory (e.g., for testing or non-standard setups).

**Why this priority**: Edge case for advanced users; most users will use the default path.

**Independent Test**: Can be tested by running the llm-setup command with a custom path flag and verifying the `kessoku.md` file is installed to the specified location.

**Acceptance Scenarios**:

1. **Given** a user wants to install Skills to a custom path, **When** they run `kessoku llm-setup claude-code --path /custom/path`, **Then** `kessoku.md` is installed to `/custom/path/kessoku.md` instead of the default location.
2. **Given** a custom path that doesn't exist, **When** the user runs the llm-setup command with that path, **Then** the directory is created with mode 0755 and Skills are installed.
3. **Given** a custom path with a relative path like `./skills`, **When** the user runs the llm-setup command, **Then** the path is resolved relative to the current working directory.
4. **Given** a custom path with tilde expansion like `~/my-skills`, **When** the user runs the llm-setup command, **Then** the tilde is expanded to the user's home directory.
5. **Given** both `--user` and `--path` are specified, **When** the user runs the command, **Then** `--path` takes precedence and the file is installed to the custom path.

---

### Edge Cases

- **Directory doesn't exist**: The skills directory (default or custom) is created automatically with mode 0755 (rwxr-xr-x).
- **Path is an existing file**: If `--path` points to an existing file (not a directory), print error message to stderr: `Error: path is a file, not a directory: <path>` and exit with code 1.
- **Cannot create directory**: If the directory cannot be created (e.g., parent path is a file, or invalid path characters), print error message to stderr with the specific path and reason, then exit with code 1.
- **Permission denied**: If the directory cannot be created or written to, print error message to stderr with the specific path and exit with code 1.
- **Disk full / Write failure**: If file write fails mid-operation, remove any partial file and exit with code 1.
- **HOME/USERPROFILE not set (--user mode)**: If the home directory environment variable is not set when using `--user`, print error message to stderr explaining the issue and suggesting `--path` flag, then exit with code 1.
- **Unsupported OS (--user mode)**: If the operating system is not Windows, macOS, or Linux when using `--user`, print error message to stderr and exit with code 1 (custom `--path` can still be used).
- **Unknown agent**: If user runs `kessoku llm-setup unknown-agent`, print error message listing available agents and exit with code 1.
- **Symlink in path**: Symlinks in the path are followed; no special handling required.
- **Read-only filesystem**: Treated as permission denied error.

## Requirements *(mandatory)*

### Functional Requirements

#### Command Structure

- **FR-001**: System MUST provide a `llm-setup` subcommand with agent-specific subcommands (e.g., `kessoku llm-setup claude-code`).
- **FR-002**: System MUST support `claude-code` as the initial agent subcommand under `llm-setup`.
- **FR-003**: When `kessoku llm-setup` is run without an agent subcommand, the system MUST display usage information listing available agents and exit with code 0.
- **FR-004**: When `kessoku llm-setup <unknown-agent>` is run, the system MUST display an error message listing available agents and exit with code 1.

#### File Embedding and Installation

- **FR-005**: System MUST embed Skills files in the binary using Go's `go:embed` directive at compile time.
- **FR-006**: System MUST install a single Skills file named `kessoku.md` to the target directory.
- **FR-007**: System MUST perform atomic file installation: write to a temporary file in the same directory, then rename to the final filename.
- **FR-008**: If atomic rename fails or is interrupted, the system MUST NOT leave partial or corrupted files in the target directory.
- **FR-009**: System MUST overwrite existing `kessoku.md` when running llm-setup again (idempotent operation).
- **FR-010**: The installed file MUST have the same byte content as the embedded source file.

#### Path Resolution

- **FR-011**: System MUST default to project-level installation:
  - **Default (project-level)**: `./.claude/skills/` (relative to current working directory)
- **FR-012**: System MUST provide a `--user` flag for user-level installation:
  - **Linux**: `$HOME/.claude/skills/`
  - **macOS**: `$HOME/.claude/skills/`
  - **Windows**: `%USERPROFILE%\.claude\skills\`
- **FR-012a**: If `HOME` (Linux/macOS) or `USERPROFILE` (Windows) environment variable is not set when `--user` is specified, the system MUST fail with a clear error message suggesting the `--path` flag.
- **FR-013**: System MUST provide a `--path` flag to specify a custom installation directory. If provided, `--path` takes precedence over `--user`.
- **FR-014**: System MUST resolve relative paths (e.g., `./skills`) relative to the current working directory.
- **FR-015**: System MUST expand tilde (`~`) to the user's home directory.
- **FR-016**: System MUST create the target directory (and parent directories) with mode 0755 if it does not exist.
- **FR-016a**: If the target path exists but is a file (not a directory), the system MUST fail with a clear error message and exit with code 1.

#### Output and Error Handling

- **FR-017**: On success, the system MUST print a message to stdout: `Skills installed to: <path>` and exit with code 0.
- **FR-018**: On failure, the system MUST print an error message to stderr describing the failure reason and exit with a non-zero code.
- **FR-019**: Exit codes:
  - `0`: Success
  - `1`: General error (permission denied, disk full, HOME not set, unknown agent, unsupported OS)

#### Skills Content

- **FR-020**: The `kessoku.md` Skills file MUST include documentation covering:
  - Provider creation (`kessoku.Provide`)
  - Async providers (`kessoku.Async`)
  - Binding interfaces (`kessoku.Bind`)
  - Value injection (`kessoku.Value`)
  - Sets (`kessoku.Set`)
  - Struct providers (`kessoku.Struct`)
  - Wire migration (`kessoku migrate`)
  - Common patterns and best practices

#### Extensibility

- **FR-021**: Adding a new agent type MUST only require:
  1. Adding a new subcommand implementation
  2. Providing agent-specific configuration (default paths, file naming)
  3. Embedding agent-specific Skills files
- **FR-022**: The core llm-setup logic (file writing, path resolution, error handling) MUST be reusable across all agents without modification.

### Key Entities

- **Skills File**: A markdown file named `kessoku.md` containing kessoku usage instructions. Embedded in the binary at compile time using `go:embed`. Uses LF line endings (Unix-style) on all platforms.
- **Agent Type**: A supported coding agent identified by its subcommand name (e.g., `claude-code`). Each agent defines:
  - Project-level and user-level installation paths
  - Skills file name(s)
- **Skills Directory**: The file system location where skills are installed. Created with mode 0755 if it does not exist.
- **Installation Scope**: Either project-level (default) or user-level (with `--user` flag).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can install kessoku Skills for Claude Code at project-level with a single command (`kessoku llm-setup claude-code`).
- **SC-001a**: Users can install kessoku Skills for Claude Code at user-level with `kessoku llm-setup claude-code --user`.
- **SC-002**: Installed `kessoku.md` file exists at the expected path and contains the complete embedded content (byte-for-byte match).
- **SC-003**: The llm-setup command works correctly on Windows, macOS, and Linux with both project-level and user-level paths.
- **SC-004**: Running `kessoku llm-setup claude-code` twice produces identical output files (idempotent).
- **SC-005**: Adding support for a new agent type requires only adding a new subcommand and agent-specific configuration, with no changes to core file installation logic.
- **SC-006**: All error conditions produce appropriate error messages to stderr and non-zero exit codes.

## Assumptions

- Claude Code stores skills in `.claude/skills/` directory (project-level) or `~/.claude/skills/` (user-level).
- Claude Code recognizes `.md` files in its skills directory as Skills without additional configuration.
- Skills files use LF line endings on all platforms for consistency.
- The user's home directory environment variable (`HOME` or `USERPROFILE`) is set in typical usage scenarios.
- Users have Claude Code installed and configured before running the llm-setup command.
- Project-level skills take precedence over user-level skills in Claude Code.

## Out of Scope

- **Uninstall command**: Removing installed Skills files is not included in this feature.
- **Version checking**: Detecting whether installed Skills are outdated compared to binary version.
- **Multiple Skills files**: Each agent uses a single Skills file (`kessoku.md`).
- **Interactive prompts**: The command operates non-interactively; no confirmation prompts.
- **Verbose/debug output**: No `--verbose` or `--debug` flags in initial implementation.
