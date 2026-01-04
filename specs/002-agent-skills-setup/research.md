# Research: Agent Skills Setup Subcommand

**Date**: 2026-01-03
**Feature**: 002-agent-skills-setup

## 1. Go's `go:embed` Directive

### Decision
Use `//go:embed` with `string` type for embedding the single `kessoku.md` Skills file.

### Rationale
- **Simplest approach** for single file embedding
- No extra overhead of `embed.FS` filesystem abstraction
- Direct string access without file I/O at runtime
- Compile-time embedding with zero runtime overhead

### Alternatives Considered
| Approach | Use Case | Why Rejected |
|----------|----------|--------------|
| `[]byte` | Binary data | Markdown is text, not binary |
| `embed.FS` | Multiple files/dirs | Single file; extra complexity |
| External file | Runtime loading | Violates spec requirement for embedded files |

### Implementation Pattern
```go
package setup

import _ "embed"

//go:embed skills/kessoku.md
var claudeCodeSkills string

func GetClaudeCodeSkills() string {
    return claudeCodeSkills
}
```

### Gotchas
- Path must be relative to package directory
- Forward slashes only (even on Windows)
- Cannot embed files with `..` parent references
- Hidden files (`.filename`) require explicit embedding

---

## 2. Kong CLI Nested Subcommands

### Decision
Use Kong's native nested command struct pattern with `kong:"cmd"` tags.

### Rationale
- Consistent with existing `GenerateCmd` and `MigrateCmd` patterns
- Kong automatically generates help for nested commands
- Clean separation of concerns per agent type
- Easy to add new agent subcommands

### Implementation Pattern
```go
// CLI root struct (in config.go)
type CLI struct {
    LogLevel string      `kong:"short='l',..."`
    Generate GenerateCmd `kong:"cmd,default='withargs',..."`
    Migrate  MigrateCmd  `kong:"cmd,..."`
    Setup    SetupCmd    `kong:"cmd,help='Setup coding agent skills'"`
    Version  kong.VersionFlag `kong:"short='v',..."`
}

// Setup command with agent subcommands
type SetupCmd struct {
    ClaudeCode ClaudeCodeCmd `kong:"cmd,name='claude-code',help='Install Claude Code skills'"`
}

// When setup called without subcommand, show usage
func (c *SetupCmd) Run(ctx *kong.Context) error {
    // This runs when no subcommand is selected
    ctx.PrintUsage(true)
    return nil
}

// Claude Code agent implementation
type ClaudeCodeCmd struct {
    Path string `kong:"short='p',help='Custom installation path'"`
}

func (c *ClaudeCodeCmd) Run(cli *CLI) error {
    // Install skills to path
    return installSkills(c.Path)
}
```

### Handling Missing Subcommand
Kong calls the parent command's `Run()` method when no subcommand is specified. The `SetupCmd.Run()` method prints usage information (FR-003).

### Help Generation
Kong automatically generates help:
- `kessoku setup --help` → Lists available agent subcommands
- `kessoku setup claude-code --help` → Shows claude-code options

---

## 3. Atomic File Writes

### Decision
Use write-to-temp + rename pattern with platform-specific handling for Windows.

### Rationale
- **Atomicity**: `os.Rename()` is atomic within same filesystem on Unix
- **Failure safety**: No partial files on interruption (FR-008)
- **Cross-platform**: Works on Linux, macOS, Windows with minor adjustments
- **Same directory**: Temp file in target directory ensures same filesystem

### Implementation Pattern
```go
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
    dir := filepath.Dir(path)

    // Create directory if needed (FR-016)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("create directory: %w", err)
    }

    // Create temp file in same directory (same filesystem)
    tmp, err := os.CreateTemp(dir, ".tmp-*")
    if err != nil {
        return fmt.Errorf("create temp file: %w", err)
    }
    tmpName := tmp.Name()

    // Write data
    if _, err := tmp.Write(data); err != nil {
        tmp.Close()
        os.Remove(tmpName)
        return fmt.Errorf("write data: %w", err)
    }

    // CRITICAL: Close before rename (Windows file locking)
    if err := tmp.Close(); err != nil {
        os.Remove(tmpName)
        return fmt.Errorf("close temp file: %w", err)
    }

    // Set permissions
    if err := os.Chmod(tmpName, perm); err != nil {
        os.Remove(tmpName)
        return fmt.Errorf("set permissions: %w", err)
    }

    // Windows: Remove destination before rename
    if runtime.GOOS == "windows" {
        _ = os.Remove(path)
    }

    // Atomic rename
    if err := os.Rename(tmpName, path); err != nil {
        os.Remove(tmpName)
        return fmt.Errorf("rename: %w", err)
    }

    return nil
}
```

### Platform Considerations
| Platform | Behavior | Handling |
|----------|----------|----------|
| Linux/macOS | `os.Rename()` is atomic; overwrites existing | Standard pattern |
| Windows | File locking; rename fails if dest exists | Remove dest before rename; close file before rename |

---

## 4. Cross-Platform Home Directory Resolution

### Decision
Use `os.UserHomeDir()` from Go standard library.

### Rationale
- Standard library (no external dependencies)
- Cross-platform: Unix (`$HOME`), Windows (`%USERPROFILE%`)
- Available since Go 1.12
- Returns error if environment variable not set

### Implementation Pattern
```go
func getDefaultSkillsPath() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", fmt.Errorf("cannot determine home directory: %w (use --path flag)")
    }
    return filepath.Join(home, ".claude", "skills"), nil
}
```

### Tilde Expansion (FR-015)
```go
func expandPath(path string) (string, error) {
    if strings.HasPrefix(path, "~") {
        home, err := os.UserHomeDir()
        if err != nil {
            return "", err
        }
        path = filepath.Join(home, path[1:])
    }
    return path, nil
}
```

### Path Validation (FR-016a)
```go
func validatePath(path string) error {
    info, err := os.Stat(path)
    if err == nil && !info.IsDir() {
        return fmt.Errorf("path is a file, not a directory: %s", path)
    }
    return nil // Path doesn't exist or is a directory
}
```

---

## 5. Skills File Content Structure

### Decision
Single markdown file `kessoku.md` covering all kessoku features per FR-020.

### Content Outline
```markdown
# Kessoku Skills for Claude Code

## Quick Reference
- Provider creation
- Async providers
- Interface binding
- Value injection
- Sets
- Struct providers
- Wire migration

## Detailed Usage
[Each section with examples]

## Common Patterns
[Best practices and patterns]
```

---

## Summary of Technical Decisions

| Area | Decision | Key Dependencies |
|------|----------|------------------|
| File Embedding | `//go:embed` with `string` | Standard library |
| CLI Framework | Kong nested commands | github.com/alecthomas/kong |
| File Writes | Temp file + rename pattern | Standard library |
| Home Directory | `os.UserHomeDir()` | Standard library |
| Skills Format | Single markdown file | None |

All unknowns resolved. Ready for Phase 1 design.
