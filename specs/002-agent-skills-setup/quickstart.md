# Quickstart: Agent Skills Setup Subcommand

**Date**: 2026-01-03
**Feature**: 002-agent-skills-setup

## Overview

The `kessoku llm-setup` command installs kessoku usage documentation as Skills files for coding agents, enabling AI assistants to help with kessoku-specific tasks.

## Prerequisites

- kessoku installed and available in PATH
- Claude Code installed (for `claude-code` agent)

## Basic Usage

### Install Claude Code Skills (Project-Level - Default)

```bash
# Install to project directory (./.claude/skills/)
kessoku llm-setup claude-code

# Expected output (stdout):
# Skills installed to: /home/user/myproject/.claude/skills/kessoku.md
```

### Install Claude Code Skills (User-Level)

```bash
# Install to user home directory (~/.claude/skills/)
kessoku llm-setup claude-code --user

# Expected output (stdout):
# Skills installed to: /home/user/.claude/skills/kessoku.md
```

### List Available Agents

```bash
# Show available agent subcommands (exit 0)
kessoku llm-setup --help

# Or just run llm-setup without subcommand (also exit 0)
kessoku llm-setup

# Output:
# Usage: kessoku llm-setup <command>
#
# Setup coding agent skills
#
# Commands:
#   claude-code    Install Claude Code skills
#
# Run "kessoku llm-setup <command> --help" for more information on a command.

# Show claude-code specific options
kessoku llm-setup claude-code --help

# Output:
# Usage: kessoku llm-setup claude-code [--user] [--path <dir>]
#
# Install Claude Code skills
#
# Flags:
#   -h, --help         Show help
#   --user             Install to user-level directory
#   -p, --path <dir>   Custom installation directory
```

### Custom Installation Path

```bash
# Install to custom directory (overrides --user if both specified)
kessoku llm-setup claude-code --path /custom/path

# With tilde expansion
kessoku llm-setup claude-code --path ~/my-skills

# With relative path (resolved from cwd)
kessoku llm-setup claude-code --path ./skills
```

## Installation Scopes

| Scope | Flag | Path | Use Case |
|-------|------|------|----------|
| Project (default) | (none) | `./.claude/skills/` | Team collaboration, per-project config |
| User | `--user` | `~/.claude/skills/` | Personal, cross-project config |
| Custom | `--path <dir>` | User-specified | Testing, non-standard setups |

**Note**: `--path` takes precedence over `--user` if both are specified.

## Error Handling

### Unknown Agent

```bash
# Unknown agent exits with code 1
kessoku llm-setup unknown-agent

# Output (stderr):
# Error: unknown command "unknown-agent"
#
# Available agents:
#   claude-code    Install Claude Code skills
```

### Home Directory Not Set (--user mode)

```bash
# If HOME/USERPROFILE not set when using --user (exit 1)
# Output (stderr):
# Error: cannot determine home directory: $HOME not set (use --path flag)

# Workaround: use --path
kessoku llm-setup claude-code --path /explicit/path
```

### Unsupported Operating System (--user mode)

```bash
# On unsupported OS with --user (exit 1)
# Output (stderr):
# Error: unsupported operating system: plan9. Use --path flag

# Workaround: use --path (project-level also works)
kessoku llm-setup claude-code --path /explicit/path
# or
kessoku llm-setup claude-code  # project-level works on any OS
```

### Path Is a File

```bash
# If --path points to a file (exit 1)
# Output (stderr):
# Error: path is a file, not a directory: /path/to/file
```

### Permission Denied

```bash
# If directory cannot be created or written (exit 1)
# Output (stderr):
# Error: permission denied: /protected/path
```

## Verification

After installation, verify the Skills file exists:

```bash
# Project-level (default)
cat ./.claude/skills/kessoku.md

# User-level (--user)
cat ~/.claude/skills/kessoku.md

# Windows (PowerShell) - user-level
Get-Content $env:USERPROFILE\.claude\skills\kessoku.md
```

## What's Included

The installed `kessoku.md` Skills file covers:

- **Provider creation** (`kessoku.Provide`)
- **Async providers** (`kessoku.Async`)
- **Interface binding** (`kessoku.Bind`)
- **Value injection** (`kessoku.Value`)
- **Sets** (`kessoku.Set`)
- **Struct providers** (`kessoku.Struct`)
- **Wire migration** (`kessoku migrate`)
- **Common patterns and best practices**

## Updating Skills

Running `kessoku llm-setup claude-code` again updates the Skills file to the latest version (atomic replacement). The operation is idempotent.

```bash
# First install (project-level)
kessoku llm-setup claude-code
# Skills installed to: /home/user/myproject/.claude/skills/kessoku.md

# Update (same command)
kessoku llm-setup claude-code
# Skills installed to: /home/user/myproject/.claude/skills/kessoku.md
```

## Default Paths

### Project-Level (Default)

| OS | Path |
|----|------|
| All | `./.claude/skills/` (relative to cwd) |

### User-Level (--user)

| OS | Path |
|----|------|
| Linux | `$HOME/.claude/skills/` |
| macOS | `$HOME/.claude/skills/` |
| Windows | `%USERPROFILE%\.claude\skills\` |

## Development Testing

For local testing during development:

```bash
# Build kessoku
go build -o bin/kessoku ./cmd/kessoku

# Test project-level install
./bin/kessoku llm-setup claude-code
cat ./.claude/skills/kessoku.md

# Test user-level install
./bin/kessoku llm-setup claude-code --user
cat ~/.claude/skills/kessoku.md

# Test custom path
./bin/kessoku llm-setup claude-code --path /tmp/test-skills
cat /tmp/test-skills/kessoku.md

# Test no-subcommand behavior (should exit 0, print usage)
./bin/kessoku llm-setup
echo $?  # 0

# Test unknown agent (should exit 1)
./bin/kessoku llm-setup unknown-agent
echo $?  # 1
```
