# Contracts: Agent Skills Setup Subcommand

**Status**: CLI-only feature (no HTTP/RPC API)

This feature is a CLI subcommand with no HTTP/RPC API contracts. All contracts are CLI-based.

---

## CLI Interface Contracts

### Command: `kessoku llm-setup claude-code [--user] [--path <dir>]`

**Purpose**: Install Claude Code Skills file

**Input**:
| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--user` | bool | No | Install to user-level directory instead of project-level |
| `--path`, `-p` | string | No | Custom installation directory (overrides `--user`) |

**Output (Success)**:
- Stream: stdout
- Format: `Skills installed to: <absolute-path>`
- Exit code: `0`

**Output (Error)**:
- Stream: stderr
- Format: `Error: <message>`
- Exit code: `1`

---

### Command: `kessoku llm-setup` (no subcommand)

**Purpose**: Display available agents

**Input**: None

**Output**:
- Stream: stdout
- Format: Usage help listing available agents
- Exit code: `0`

**Example Output**:
```
Usage: kessoku llm-setup <command>

Setup coding agent skills

Commands:
  claude-code    Install Claude Code skills

Run "kessoku llm-setup <command> --help" for more information on a command.
```

**Note**: The `--user` and `--path` flags are defined on each agent subcommand (e.g., `claude-code`), not on the parent `llm-setup` command.

---

### Command: `kessoku llm-setup <unknown-agent>`

**Purpose**: Handle unknown agent subcommand

**Input**: Unknown agent name

**Output**:
- Stream: stderr
- Format: Error message with available agents
- Exit code: `1`

**Example Output**:
```
Error: unknown command "unknown-agent"

Available agents:
  claude-code    Install Claude Code skills
```

---

## Error Message Contracts

| Error Condition | Message Format | Exit Code |
|----------------|----------------|-----------|
| HOME not set (--user) | `cannot determine home directory: $HOME not set (use --path flag)` | 1 |
| Unsupported OS (--user) | `unsupported operating system: <os>. Use --path flag` | 1 |
| Path is file | `path is a file, not a directory: <path>` | 1 |
| Permission denied | `permission denied: <path>` | 1 |
| Cannot create dir | `cannot create directory: <path>: <err>` | 1 |
| Cannot write file | `cannot write file: <err>` | 1 |
| Cannot install | `cannot install file: <err>` | 1 |
| Tilde expansion fail | `cannot determine home directory for tilde expansion` | 1 |
| Cannot get cwd | `cannot determine current directory: <err>` | 1 |
| Cannot access path | `cannot access path: <path>: <err>` | 1 |

---

## File Contract

### Installed File

**Path (Project-Level - Default)**: `./.claude/skills/kessoku.md` (relative to cwd)
**Path (User-Level)**: `~/.claude/skills/kessoku.md` (or custom via `--path`)

**Format**: Markdown (UTF-8, LF line endings)

**Content**: Kessoku usage documentation (embedded at compile time)

**Permissions**: `0644` (rw-r--r--)

**Directory Permissions**: `0755` (rwxr-xr-x) if created

---

## Path Resolution Contract

### Input Processing

| Input Type | Processing |
|------------|------------|
| No flags (default) | Project-level: `./<subpath>` (relative to cwd) |
| `--user` | User-level: `~/<subpath>` |
| `--path <dir>` | Use specified directory (overrides `--user`) |
| Absolute path | Used as-is |
| Relative path | Resolved relative to cwd |
| Tilde path (`~/...`) | Tilde expanded to home directory |

### Default Paths

**Project-Level (Default)**:
| OS | Path |
|----|------|
| All | `./.claude/skills/` |

**User-Level (`--user`)**:
| OS | Path |
|----|------|
| `linux` | `$HOME/.claude/skills/` |
| `darwin` | `$HOME/.claude/skills/` |
| `windows` | `%USERPROFILE%\.claude\skills\` |

### Unsupported OS

| OS | Project-Level | User-Level (--user) |
|----|---------------|---------------------|
| `linux` | Works | Works |
| `darwin` | Works | Works |
| `windows` | Works | Works |
| Other (plan9, etc.) | Works | Error (use --path) |

---

## Flag Precedence Contract

When multiple flags are specified:

```
--path > --user > (default project-level)
```

| Flags | Result |
|-------|--------|
| (none) | Project-level |
| `--user` | User-level |
| `--path /foo` | `/foo` |
| `--user --path /foo` | `/foo` (--path wins) |

---

## Atomicity Contract

### Guarantees

1. **Atomic install**: File is written to temp file, then renamed
2. **No partial files**: On failure, no partial/corrupted files remain
3. **Idempotent**: Running twice produces identical result
4. **Overwrite**: Existing file is replaced atomically

### Implementation

```
1. Create temp file in target directory (.tmp-*)
2. Write content to temp file
3. Close temp file (critical for Windows)
4. Set permissions (0644)
5. [Windows only] Remove destination if exists
6. Rename temp file to final name
7. On any error: remove temp file
```
