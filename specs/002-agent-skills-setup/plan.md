# Implementation Plan: Agent Skills Setup Subcommand

**Branch**: `002-agent-skills-setup` | **Date**: 2026-01-03 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-agent-skills-setup/spec.md`

## Summary

Add a `kessoku llm-setup <agent>` subcommand that installs kessoku usage documentation as Skills files for coding agents. The initial implementation supports Claude Code by embedding a `kessoku.md` Skills file via `go:embed`. By default, Skills are installed at the **project level** (`./.claude/skills/`), with an optional `--user` flag for user-level installation (`~/.claude/skills/`). The architecture is designed for extensibility to support additional coding agents in the future.

## Technical Context

**Language/Version**: Go 1.24+
**Primary Dependencies**: github.com/alecthomas/kong (CLI framework, already in use)
**Storage**: File-based output (Skills files installed to filesystem)
**Testing**: go test (standard Go testing)
**Target Platform**: Linux, macOS, Windows (cross-platform CLI tool)
**Project Type**: Single Go module with CLI tool
**Performance Goals**: N/A (one-shot file installation, not performance-critical)
**Constraints**: Must use `go:embed` for Skills files; atomic file operations for installation
**Scale/Scope**: Single agent (Claude Code) in initial implementation; designed for multi-agent extensibility

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Status**: PASSED (No specific gates defined)

The constitution file is currently a template without project-specific principles. No gates to enforce. The implementation follows existing patterns:
- Kong CLI framework (consistent with existing `generate` and `migrate` commands)
- Standard Go project structure (internal/ packages)
- Test-driven development approach (per CLAUDE.md guidelines)

## Project Structure

### Documentation (this feature)

```text
specs/002-agent-skills-setup/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (N/A for CLI-only feature)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── config/
│   └── config.go        # Add LLMSetupCmd to CLI struct
├── llmsetup/            # NEW: LLM setup subcommand implementation
│   ├── llmsetup.go      # LLMSetupCmd with agent subcommands
│   ├── agent.go         # Agent interface and registry
│   ├── install.go       # Atomic file installation logic
│   ├── path.go          # Path resolution logic
│   ├── claudecode.go    # Claude Code agent implementation
│   ├── skills/          # Embedded Skills files
│   │   └── kessoku.md   # Claude Code Skills content
│   ├── llmsetup_test.go # Unit tests for llm-setup command
│   ├── install_test.go  # Unit tests for atomic installation
│   └── path_test.go     # Unit tests for path resolution
├── kessoku/             # Existing: codegen engine
└── migrate/             # Existing: wire migration
```

**Structure Decision**: Follows existing pattern in internal/ with a new `llmsetup` package. Skills files embedded in `internal/llmsetup/skills/` subdirectory.

---

## Path Resolution Algorithm (FR-011–FR-016a)

### Algorithm

```
ResolvePath(customPath string, userFlag bool, agent Agent) -> (string, error):

1. IF customPath is provided (--path flag):
   a. IF customPath starts with "~":
      - home := getHomeDir()  // see helper below
      - IF error: return error "cannot determine home directory for tilde expansion"
      - path := home + customPath[1:]
   b. ELSE IF customPath is relative (not absolute):
      - path := filepath.Abs(customPath)  // resolve relative to cwd
   c. ELSE:
      - path := customPath
   d. GOTO step 4

2. ELSE IF userFlag is true (--user flag):
   a. goos := runtime.GOOS
   b. IF goos NOT IN ["linux", "darwin", "windows"]:
      - return error "unsupported operating system: <goos>. Use --path flag"
   c. home := getHomeDir()  // explicitly checks env var
   d. IF error:
      - return error "cannot determine home directory: <err> (use --path flag)"
   e. path := filepath.Join(home, agent.UserSubPath())
      // e.g., home="/home/user", subpath=".claude/skills" -> "/home/user/.claude/skills"
   f. GOTO step 4

3. ELSE (default: project-level):
   a. cwd := os.Getwd()
   b. IF error: return error "cannot determine current directory: <err>"
   c. path := filepath.Join(cwd, agent.ProjectSubPath())
      // e.g., cwd="/home/user/myproject", subpath=".claude/skills" -> "/home/user/myproject/.claude/skills"

4. Validate path:
   a. info, err := os.Stat(path)
   b. IF err == nil AND info.IsDir() == false:
      - return error "path is a file, not a directory: <path>"
   c. IF os.IsNotExist(err):
      - // OK: directory will be created during installation
   d. ELSE IF err != nil:
      - return error "cannot access path: <path>: <err>"
   e. RETURN path, nil


// Helper: getHomeDir() - explicitly checks environment variable per FR-012a
getHomeDir() -> (string, error):
  1. goos := runtime.GOOS
  2. IF goos == "windows":
     a. home := os.Getenv("USERPROFILE")
     b. IF home == "": return error "$USERPROFILE not set"
     c. RETURN home, nil
  3. ELSE:
     a. home := os.Getenv("HOME")
     b. IF home == "": return error "$HOME not set"
     c. RETURN home, nil
```

**Note**: We use explicit environment variable checks (`os.Getenv`) rather than `os.UserHomeDir()` to ensure the mandated failure behavior when HOME/USERPROFILE is not set (FR-012a). `os.UserHomeDir()` has fallback behavior that could succeed even without the environment variable.

### Installation Scope

| Scope | Flag | Default Path | Example |
|-------|------|--------------|---------|
| Project (default) | (none) | `./.claude/skills/` | `/home/user/myproject/.claude/skills/` |
| User | `--user` | `~/.claude/skills/` | `/home/user/.claude/skills/` |
| Custom | `--path <dir>` | User-specified | `/custom/path/` |

**Note**: `--path` takes precedence over `--user`.

### User-Level Paths by OS

| OS | Environment Variable | Subpath | Resolved Example |
|----|---------------------|---------|------------------|
| linux | `$HOME` | `.claude/skills` | `/home/user/.claude/skills` |
| darwin | `$HOME` | `.claude/skills` | `/Users/user/.claude/skills` |
| windows | `%USERPROFILE%` | `.claude\skills` | `C:\Users\user\.claude\skills` |

### Error Cases

| Condition | Error Message | Exit Code |
|-----------|---------------|-----------|
| Unsupported OS (--user) | `unsupported operating system: <os>. Use --path flag` | 1 |
| HOME/USERPROFILE not set (--user) | `cannot determine home directory: <err> (use --path flag)` | 1 |
| Tilde expansion fails | `cannot determine home directory for tilde expansion` | 1 |
| Path is a file | `path is a file, not a directory: <path>` | 1 |

---

## Atomic File Installation (FR-007/FR-008)

### Algorithm

```
InstallFile(targetDir string, fileName string, content []byte) -> error:

1. Create directory:
   a. err := os.MkdirAll(targetDir, 0755)
   b. IF error: return error "cannot create directory: <path>: <err>"

2. Create temp file in same directory (ensures same filesystem):
   a. tmp := os.CreateTemp(targetDir, ".tmp-*")
   b. IF error: return error "cannot create temp file: <err>"
   c. tmpName := tmp.Name()

3. Write content:
   a. _, err := tmp.Write(content)
   b. IF error:
      - tmp.Close()
      - os.Remove(tmpName)  // cleanup partial file
      - return error "cannot write file: <err>"

4. Close file (CRITICAL for Windows):
   a. err := tmp.Close()
   b. IF error:
      - os.Remove(tmpName)
      - return error "cannot close temp file: <err>"

5. Set permissions:
   a. err := os.Chmod(tmpName, 0644)
   b. IF error:
      - os.Remove(tmpName)
      - return error "cannot set file permissions: <err>"

6. Atomic rename:
   a. finalPath := filepath.Join(targetDir, fileName)
   b. IF runtime.GOOS == "windows":
      - os.Remove(finalPath)  // Windows requires dest not exist
   c. err := os.Rename(tmpName, finalPath)
   d. IF error:
      - os.Remove(tmpName)  // cleanup temp file
      - return error "cannot install file: <err>"

7. RETURN nil (success)
```

### File Permissions

| Type | Mode | Description |
|------|------|-------------|
| Directory | `0755` | rwxr-xr-x (owner full, others read/execute) |
| File | `0644` | rw-r--r-- (owner read/write, others read) |

### Failure Guarantees

- **No partial files**: Temp file is removed on any error before rename
- **Atomic overwrite**: Rename replaces existing file atomically (except Windows quirk)
- **Windows handling**: Remove destination before rename to avoid "file exists" error

---

## Command Handling (FR-003/FR-004)

### `kessoku llm-setup` (no subcommand)

**Behavior**: Print usage with available agents, exit 0

```go
func (c *LLMSetupCmd) Run(ctx *kong.Context) error {
    // Kong calls this when no subcommand specified
    ctx.PrintUsage(true)
    return nil  // exit 0 per FR-003
}
```

**Output** (stdout):
```
Usage: kessoku llm-setup <command>

Setup coding agent skills

Commands:
  claude-code    Install Claude Code skills

Run "kessoku llm-setup <command> --help" for more information on a command.
```

**Note**: The `--user` and `--path` flags are defined on each agent subcommand (e.g., `claude-code`), not on the parent `llm-setup` command. This allows different agents to potentially have different flag options in the future.

### `kessoku llm-setup <unknown-agent>`

**Behavior**: Print error with available agents, exit 1

Kong handles unknown subcommands automatically with exit 1. Custom error message via `kong.ConfigureHelp`:

**Output** (stderr):
```
Error: unknown command "unknown-agent"

Available agents:
  claude-code    Install Claude Code skills
```

---

## Agent Registry & Extensibility (FR-021/FR-022)

### Agent Interface

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
```

**Note**: `ProjectSubPath()` and `UserSubPath()` return relative paths. The path resolution algorithm joins these with the appropriate base directory (cwd or home).

### Agent Registry

```go
// registry.go
var agents = map[string]Agent{
    "claude-code": &ClaudeCodeAgent{},
}

func GetAgent(name string) (Agent, bool) {
    a, ok := agents[name]
    return a, ok
}

func ListAgents() []Agent {
    // Return sorted list for consistent help output
}
```

### Shared Installation Logic

```go
// install.go - reused by all agents (FR-022)
func Install(agent Agent, customPath string, userFlag bool) error {
    // 1. Resolve path
    path, err := ResolvePath(customPath, userFlag, agent)
    if err != nil {
        return err
    }

    // 2. Validate path
    if err := ValidatePath(path); err != nil {
        return err
    }

    // 3. Atomic install
    content := []byte(agent.SkillsContent())
    return InstallFile(path, agent.SkillsFileName(), content)
}
```

### Adding a New Agent (FR-021)

1. Create `internal/llmsetup/newagent.go`:
   ```go
   //go:embed skills/kessoku-newagent.md
   var newAgentSkills string

   type NewAgent struct{}
   func (a *NewAgent) Name() string { return "new-agent" }
   func (a *NewAgent) SkillsContent() string { return newAgentSkills }
   func (a *NewAgent) ProjectSubPath() string { return ".newagent/skills" }
   func (a *NewAgent) UserSubPath() string { return ".newagent/skills" }
   // ... implement other methods
   ```

2. Register in `registry.go`:
   ```go
   var agents = map[string]Agent{
       "claude-code": &ClaudeCodeAgent{},
       "new-agent":   &NewAgent{},  // ADD
   }
   ```

3. Add subcommand in `llmsetup.go`:
   ```go
   type LLMSetupCmd struct {
       ClaudeCode ClaudeCodeCmd `kong:"cmd,..."`
       NewAgent   NewAgentCmd   `kong:"cmd,..."`  // ADD
   }
   ```

**No changes to core logic** (ResolvePath, InstallFile, ValidatePath).

---

## Testing Strategy

### Test File Mapping

| Acceptance Scenario | Test File | Test Function |
|---------------------|-----------|---------------|
| Project-level default path | `path_test.go` | `TestResolvePath_ProjectLevel` |
| User-level path (--user) | `path_test.go` | `TestResolvePath_UserLevel` |
| User-level per OS | `path_test.go` | `TestResolvePath_UserLevel_Linux`, `_Darwin`, `_Windows` |
| --path overrides --user | `path_test.go` | `TestResolvePath_PathOverridesUser` |
| Tilde expansion | `path_test.go` | `TestResolvePath_TildeExpansion` |
| Relative path resolution | `path_test.go` | `TestResolvePath_RelativePath` |
| HOME not set (--user) | `path_test.go` | `TestResolvePath_HomeNotSet` |
| Unsupported OS (--user) | `path_test.go` | `TestResolvePath_UnsupportedOS` |
| Path is file error | `path_test.go` | `TestValidatePath_PathIsFile` |
| Directory creation 0755 | `install_test.go` | `TestInstallFile_CreatesDirectory` |
| Atomic overwrite | `install_test.go` | `TestInstallFile_AtomicOverwrite` |
| Idempotent install | `install_test.go` | `TestInstallFile_Idempotent` |
| Partial file cleanup | `install_test.go` | `TestInstallFile_CleansUpOnError` |
| Permission denied | `install_test.go` | `TestInstallFile_PermissionDenied` |
| Unknown agent exit 1 | `llmsetup_test.go` | `TestLLMSetupCmd_UnknownAgent` |
| No subcommand exit 0 | `llmsetup_test.go` | `TestLLMSetupCmd_NoSubcommand` |
| Success message stdout | `llmsetup_test.go` | `TestClaudeCodeCmd_SuccessOutput` |
| Error message stderr | `llmsetup_test.go` | `TestClaudeCodeCmd_ErrorOutput` |

### Table-Driven Test Example

```go
func TestResolvePath(t *testing.T) {
    tests := []struct {
        name       string
        customPath string
        userFlag   bool
        goos       string
        homeDir    string
        cwd        string
        wantPath   string
        wantErr    string
    }{
        {
            name:     "project-level default",
            cwd:      "/home/user/myproject",
            wantPath: "/home/user/myproject/.claude/skills",
        },
        {
            name:     "user-level with --user",
            userFlag: true,
            goos:     "linux",
            homeDir:  "/home/user",
            wantPath: "/home/user/.claude/skills",
        },
        {
            name:       "--path overrides --user",
            customPath: "/custom/path",
            userFlag:   true,
            wantPath:   "/custom/path",
        },
        {
            name:       "tilde expansion",
            customPath: "~/custom",
            homeDir:    "/home/user",
            wantPath:   "/home/user/custom",
        },
        {
            name:     "home not set with --user",
            userFlag: true,
            goos:     "linux",
            wantErr:  "cannot determine home directory",
        },
        {
            name:     "unsupported OS with --user",
            userFlag: true,
            goos:     "plan9",
            wantErr:  "unsupported operating system: plan9",
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ... test implementation
        })
    }
}
```

---

## Complexity Tracking

No violations to justify. Implementation follows existing patterns:
- Single new package (internal/llmsetup)
- Reuses existing Kong CLI patterns
- Simple file embedding and installation logic
