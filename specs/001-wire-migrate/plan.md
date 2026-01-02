# Implementation Plan: Wire to Kessoku Migration Tool

**Branch**: `001-wire-migrate` | **Date**: 2026-01-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-wire-migrate/spec.md`

## Summary

Create a `kessoku migrate` subcommand that automatically converts google/wire configuration files to kessoku format. The tool parses wire patterns (NewSet, Bind, Value, InterfaceValue, Struct, FieldsOf) using Go's AST packages and generates equivalent kessoku code. Multiple input files are merged into a single output file (default: `kessoku.go`).

## Technical Context

**Language/Version**: Go 1.24+
**Primary Dependencies**: github.com/alecthomas/kong (CLI), golang.org/x/tools (AST parsing, type checking)
**Storage**: N/A (file-based input/output, no persistent storage)
**Testing**: go test with table-driven tests
**Target Platform**: Linux, macOS, Windows (cross-platform CLI tool)
**Project Type**: single
**Performance Goals**: N/A (batch processing, not real-time)
**Constraints**: None specific
**Scale/Scope**: Single file or multiple files within same Go package

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution file contains template placeholders only. No specific project constraints are defined.

**Status**: PASS (no constraints to violate)

## Project Structure

### Documentation (this feature)

```text
specs/001-wire-migrate/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── config/
│   └── config.go        # CLI configuration (add migrate subcommand)
├── migrate/             # NEW: Migration module
│   ├── migrate.go       # Migrator orchestrator
│   ├── parser.go        # Wire pattern AST parser
│   ├── transformer.go   # Wire → Kessoku transformations
│   ├── writer.go        # Output file generation
│   ├── patterns.go      # Wire/Kessoku pattern definitions
│   └── *_test.go        # Unit tests for each component
└── kessoku/             # Existing DI implementation (reference only)
```

**Structure Decision**: New `internal/migrate` package following existing module structure. Integrates with existing CLI via kong subcommand pattern.

## Implementation Milestones

### Milestone 1: CLI Foundation
**Exit Criteria**:
- `kessoku migrate --help` displays help message
- `kessoku migrate` without args shows usage error
- Existing `kessoku <files>` (generate) behavior unchanged
- Unit tests for CLI parsing pass

**Work Breakdown**:
1. Refactor `internal/config/config.go` to use kong subcommand pattern
2. Add `MigrateCmd` struct with `-o` flag
3. Add integration test for help output

### Milestone 2: Wire Pattern Parser
**Exit Criteria**:
- Parser detects all wire patterns: NewSet, Bind, Value, InterfaceValue, Struct, FieldsOf
- Parser emits warnings for unsupported patterns (wire.Build)
- Parser emits warnings for files without wire import
- Parser returns ParseError for syntax/type errors
- Unit tests with golden files pass for each pattern type

**Work Breakdown**:
1. Define `WirePattern` types in `patterns.go`
2. Implement `Parser` with AST visitor in `parser.go`
3. Add wire import detection
4. Add testdata golden files for each pattern
5. Unit tests covering all pattern types and edge cases

### Milestone 3: Pattern Transformer
**Exit Criteria**:
- Each wire pattern correctly transforms to kessoku equivalent
- Pointer types handled correctly in Bind transformation
- Struct/FieldsOf generate correct anonymous functions
- All fields (including unexported) included for Struct "*"
- Missing constructor for Bind returns error (not invalid code)
- Unit tests with transformation verification pass

**Work Breakdown**:
1. Define `KessokuPattern` types in `patterns.go`
2. Implement `Transformer` in `transformer.go`
3. Add constructor lookup for Bind with error on missing constructor
4. Add function literal generation for Struct/FieldsOf
5. Unit tests for each transformation rule including error cases

### Milestone 4: Output Writer
**Exit Criteria**:
- Generated code compiles without errors
- Generated code passes `go fmt`
- Imports correctly replaced (wire → kessoku)
- Build tags and comments stripped from output
- Output file created only when patterns found
- Unit tests pass

**Work Breakdown**:
1. Implement `Writer` in `writer.go`
2. Add import handling (remove wire, add kessoku)
3. Add go/format integration
4. Add no-output logic for empty results

### Milestone 5: Multi-file Merge
**Exit Criteria**:
- Multiple files merged into single output
- Imports deduplicated
- Package mismatch detected and reported as error
- Name collision detected and reported as error
- Integration tests pass

**Work Breakdown**:
1. Add `MergedOutput` logic to `migrate.go`
2. Add package validation
3. Add identifier collision detection
4. Add import deduplication
5. Integration tests with multi-file scenarios

### Milestone 6: End-to-End Integration
**Exit Criteria**:
- All acceptance scenarios from spec pass
- All edge cases from spec handled correctly
- `go tool tools lint ./...` passes
- `go test -v ./...` passes
- Documentation updated (README if needed)

**Work Breakdown**:
1. Create end-to-end test suite
2. Test with real-world wire examples
3. Fix any remaining issues
4. Update CLAUDE.md if commands change

## Complexity Tracking

No constitution violations - no entries needed.
