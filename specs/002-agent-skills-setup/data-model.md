# Data Model: Agent Skills Setup Subcommand

**Date**: 2026-01-03
**Feature**: 002-agent-skills-setup

## Entities

### 1. Agent

Represents a supported coding agent with its configuration.

| Field | Type | Description | Validation |
|-------|------|-------------|------------|
| `Name` | string | Subcommand identifier (e.g., "claude-code") | Alphanumeric, hyphens allowed |
| `Description` | string | Help text for the subcommand | Non-empty |
| `SkillsFileName` | string | Output filename (e.g., "kessoku.md") | Valid filename |
| `ProjectSubPath` | string | Relative path for project-level install | No leading slash, forward slashes |
| `UserSubPath` | string | Relative path for user-level install | No leading slash, forward slashes |

**Supported Agents (Initial)**:
- `claude-code`: Claude Code coding agent
  - `ProjectSubPath`: `.claude/skills`
  - `UserSubPath`: `.claude/skills`
  - `SkillsFileName`: `kessoku.md`

**Path Resolution**:

| Scope | Base Directory | Subpath | Example Result |
|-------|----------------|---------|----------------|
| Project (default) | `os.Getwd()` | `agent.ProjectSubPath()` | `./myproject/.claude/skills` |
| User (`--user`) | `$HOME` or `%USERPROFILE%` (env var, fails if unset) | `agent.UserSubPath()` | `~/.claude/skills` |
| Custom (`--path`) | User-specified | N/A | `/custom/path/` |

**Note**: User-level resolution explicitly checks the environment variable (`os.Getenv("HOME")` or `os.Getenv("USERPROFILE")`) and fails with an error if the variable is not set. This ensures the mandated failure behavior per FR-012a.

### 2. SkillsContent

Embedded Skills file content.

| Field | Type | Description |
|-------|------|-------------|
| `Content` | string | Markdown content (embedded via `go:embed`) |
| `FileName` | string | Output file name (`kessoku.md`) |

**Invariants**:
- LF line endings on all platforms
- UTF-8 encoded

### 3. InstallationConfig

Configuration for a Skills installation operation.

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `TargetDir` | string | Directory to install Skills | Computed from scope |
| `FileName` | string | Output file name | `kessoku.md` |
| `FileMode` | os.FileMode | File permissions | `0644` |
| `DirMode` | os.FileMode | Directory permissions | `0755` |
| `Scope` | InstallScope | Project, User, or Custom | Project |

### 4. InstallScope

Enum for installation scope.

| Value | Flag | Description |
|-------|------|-------------|
| `ScopeProject` | (default) | Install to `./<subpath>` |
| `ScopeUser` | `--user` | Install to `~/<subpath>` |
| `ScopeCustom` | `--path` | Install to user-specified path |

### 5. InstallationResult

Result of a Skills installation operation.

| Field | Type | Description |
|-------|------|-------------|
| `Success` | bool | Whether installation succeeded |
| `Path` | string | Full path to installed file |
| `Error` | error | Error if installation failed |

## State Transitions

```
                    ┌─────────────────┐
                    │  Initial State  │
                    │  (No Skills)    │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
        ┌───────────│   Installing    │───────────┐
        │           └────────┬────────┘           │
        ▼                    │                    ▼
┌───────────────┐           │           ┌─────────────────┐
│    Error      │           ▼           │    Success      │
│ (No changes)  │   ┌───────────────┐   │(Skills installed)│
└───────────────┘   │   Installed   │   └─────────────────┘
                    │ (Idempotent)  │
                    └───────────────┘
                             │
                             ▼ (Run again)
                    ┌─────────────────┐
                    │    Updated      │
                    │ (Atomic replace)│
                    └─────────────────┘
```

## Error States

| Error Type | Condition | Exit Code | Message Pattern |
|------------|-----------|-----------|-----------------|
| `ErrHomeNotSet` | `HOME`/`USERPROFILE` env var not set (--user or tilde) | 1 | "cannot determine home directory: $HOME not set (use --path flag)" |
| `ErrPathIsFile` | Target path is a file, not directory | 1 | "path is a file, not a directory: <path>" |
| `ErrPermissionDenied` | Cannot create dir or write file | 1 | "permission denied: <path>" |
| `ErrWriteFailed` | File write fails mid-operation | 1 | "cannot write file: <reason>" |
| `ErrUnknownAgent` | Invalid agent subcommand | 1 | "unknown command \"<name>\"" (Kong default) |
| `ErrUnsupportedOS` | OS not Windows/macOS/Linux (--user) | 1 | "unsupported operating system: <os>. Use --path flag" |
| `ErrTildeExpansion` | Tilde path but HOME not set | 1 | "cannot determine home directory for tilde expansion" |
| `ErrDirCreate` | Cannot create target directory | 1 | "cannot create directory: <path>: <err>" |
| `ErrTempFile` | Cannot create temp file | 1 | "cannot create temp file: <err>" |
| `ErrRename` | Atomic rename failed | 1 | "cannot install file: <err>" |
| `ErrGetCwd` | Cannot determine current directory | 1 | "cannot determine current directory: <err>" |
| `ErrPathAccess` | Cannot access/stat path | 1 | "cannot access path: <path>: <err>" |

## Validation Rules

| Entity | Field | Rule |
|--------|-------|------|
| Agent | Name | Alphanumeric + hyphens, lowercase |
| Agent | ProjectSubPath | No leading slash, forward slashes only |
| Agent | UserSubPath | No leading slash, forward slashes only |
| InstallConfig | TargetDir | Must be absolute after resolution |
| InstallConfig | FileMode | Must be 0644 |
| InstallConfig | DirMode | Must be 0755 |

## Supported Operating Systems

| OS (`runtime.GOOS`) | Project-Level | User-Level (--user) | Home Variable |
|---------------------|---------------|---------------------|---------------|
| `linux` | Yes | Yes | `$HOME` |
| `darwin` | Yes | Yes | `$HOME` |
| `windows` | Yes | Yes | `%USERPROFILE%` |
| Other (plan9, etc.) | Yes | No (requires --path) | N/A |

**Note**: Project-level installation works on any OS. User-level requires a supported OS or explicit `--path`.

## Type Definitions (Go)

```go
// Agent defines configuration for a coding agent.
type Agent interface {
    Name() string            // Subcommand name (e.g., "claude-code")
    Description() string     // Help text
    SkillsContent() string   // Embedded file content
    SkillsFileName() string  // Output filename (e.g., "kessoku.md")
    ProjectSubPath() string  // Subpath for project-level (e.g., ".claude/skills")
    UserSubPath() string     // Subpath for user-level (e.g., ".claude/skills")
}

// SkillsContent holds embedded Skills file content.
type SkillsContent struct {
    Content  string
    FileName string
}

// InstallConfig configures a Skills installation.
type InstallConfig struct {
    TargetDir string
    FileName  string
    FileMode  os.FileMode
    DirMode   os.FileMode
}

// InstallResult represents the outcome of installation.
type InstallResult struct {
    Success bool
    Path    string
    Error   error
}
```

## Relationships

```
┌─────────────────┐       1:1        ┌─────────────────────┐
│      Agent      │─────────────────▶│   SkillsContent     │
│  (claude-code)  │                  │   (kessoku.md)      │
└────────┬────────┘                  └─────────────────────┘
         │
         │ configures
         ▼
┌─────────────────┐       1:1        ┌─────────────────────┐
│  InstallConfig  │─────────────────▶│   InstallResult     │
│  (target path)  │                  │   (success/error)   │
└─────────────────┘                  └─────────────────────┘
```

## Extensibility Model

Adding a new agent type:

1. **Define Agent Config** (new struct or registration):
   ```go
   type CursorAgent struct{}

   func (a *CursorAgent) Name() string { return "cursor" }
   func (a *CursorAgent) Description() string { return "Install Cursor IDE skills" }
   func (a *CursorAgent) ProjectSubPath() string { return ".cursor/skills" }
   func (a *CursorAgent) UserSubPath() string { return ".cursor/skills" }
   // ...
   ```

2. **Add Embedded File**:
   ```go
   //go:embed skills/kessoku-cursor.md
   var cursorSkills string
   ```

3. **Add Subcommand** (Kong struct):
   ```go
   type LLMSetupCmd struct {
       ClaudeCode ClaudeCodeCmd `kong:"cmd,name='claude-code',..."`
       Cursor     CursorCmd     `kong:"cmd,name='cursor',..."`
   }
   ```

Core installation logic (FR-022) remains unchanged.
