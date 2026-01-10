# Implementation Plan: Golden Test for Code Generation

**Branch**: `003-golden-test` | **Date**: 2026-01-10 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-golden-test/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Add Golden Test infrastructure to `internal/kessoku/` that validates code generation output against known-good snapshots. The implementation follows the existing `internal/migrate/testdata/` pattern with shallow directory discovery, `t.TempDir()` isolation, and adds `-update` flag support for regenerating golden files.

## Technical Context

**Language/Version**: Go 1.24+
**Primary Dependencies**: golang.org/x/tools (AST/type checking), standard library (testing, flag, os, path/filepath)
**Storage**: File-based (testdata directory with input/expected files)
**Testing**: go test with table-driven tests and golden file comparison
**Target Platform**: Linux/macOS/Windows (cross-platform Go development)
**Project Type**: Single module library
**Performance Goals**: Test suite should complete in <10 seconds for all golden tests
**Constraints**: Tests must support parallel execution (when not updating); no external test dependencies
**Scale/Scope**: ~5-10 initial test cases covering basic, async, complex, sets, struct_expansion patterns

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Note**: The project constitution (`/.specify/memory/constitution.md`) is currently a template placeholder without specific project rules. No gates defined - proceeding with standard Go testing best practices.

**Applied Principles**:
- Follow existing `internal/migrate/testdata/` pattern for consistency
- TDD approach: test infrastructure enables test-driven development workflow
- Simplicity: shallow discovery, minimal dependencies, standard Go testing conventions

## Project Structure

### Documentation (this feature)

```text
specs/003-golden-test/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/kessoku/
├── golden_test.go           # New: Golden test runner
└── testdata/                # New: Test case directory
    ├── basic/               # Selectively copied from examples/basic
    │   ├── kessoku.go       # Input: kessoku.Inject call
    │   ├── config.go        # Provider definitions
    │   ├── service.go       # Provider definitions
    │   ├── database.go      # Provider definitions
    │   ├── app.go           # Provider definitions
    │   └── expected.go      # Golden file (renamed from kessoku_band.go)
    │   # EXCLUDED: basic (binary), kessoku_band.go (source for expected.go)
    ├── async_parallel/      # Input .go files + expected.go only
    ├── complex/             # Input .go files + expected.go only
    ├── complex_async/       # Input .go files + expected.go only
    ├── sets/                # Input .go files + expected.go only
    └── struct_expansion/    # Input .go files + expected.go only
```

**Structure Decision**: Follow existing Go conventions with `testdata/` subdirectory. Test cases are **selectively copied** from `examples/` - only `.go` source files (excluding `*_band.go`) plus renamed golden file. Binaries and generated files are excluded to keep the repo clean.

## Technical Implementation Details

### Generator Invocation

The test invokes the code generator via `Processor.ProcessFiles()`:

```go
// In golden_test.go
func runTest(t *testing.T, testCaseName string) {
    // 1. Copy input files to temp directory
    tmpDir := t.TempDir()
    srcDir := filepath.Join("testdata", testCaseName)
    copyInputFiles(t, srcDir, tmpDir)  // copies all .go except expected.go

    // 2. Run processor on the copied kessoku.go
    processor := NewProcessor()
    kessokuPath := filepath.Join(tmpDir, "kessoku.go")
    if err := processor.ProcessFiles([]string{kessokuPath}); err != nil {
        t.Fatalf("generation failed: %v", err)
    }

    // 3. Processor writes to kessoku_band.go (fixed naming convention)
    generatedPath := filepath.Join(tmpDir, "kessoku_band.go")
    actual, err := os.ReadFile(generatedPath)
    if err != nil {
        t.Fatalf("failed to read generated file: %v", err)
    }

    // 4. Compare with expected.go from testdata
    expectedPath := filepath.Join(srcDir, "expected.go")
    expected, err := os.ReadFile(expectedPath)
    if err != nil {
        t.Fatalf("missing golden file: %s", expectedPath)
    }

    if !bytes.Equal(actual, expected) {
        t.Errorf("output mismatch:\n--- expected ---\n%s\n--- got ---\n%s",
            string(expected), string(actual))
    }
}
```

**Key points**:
- Generator writes to `*_band.go` next to input file (fixed naming in `processor.go:82-84`)
- Test copies inputs to `t.TempDir()` so generated output goes there
- Comparison uses `expected.go` from original testdata directory

### Output Normalization

Before comparison, generated output is normalized:

1. **gofmt**: Run `format.Source()` on generated code to ensure consistent formatting
2. **Deterministic output**: Generator must produce stable output (no map iteration order issues)
3. **Import sorting**: Use `golang.org/x/tools/imports` or rely on gofmt's import grouping

### Diff Formatting

When output differs, show a clear diff:

```go
if string(actual) != string(expected) {
    t.Errorf("output mismatch for %s:\n--- expected ---\n%s\n--- got ---\n%s",
        testName, string(expected), string(actual))
}
```

For large diffs, consider line-by-line comparison showing first N differences.

### Update Mode (`-update` flag)

When `-update` is set, the test overwrites `expected.go` instead of comparing:

```go
var update = flag.Bool("update", false, "update golden files")

func runTest(t *testing.T, testCaseName string) {
    // ... (same setup: copy to tmpDir, run processor)

    generatedPath := filepath.Join(tmpDir, "kessoku_band.go")
    actual, _ := os.ReadFile(generatedPath)

    expectedPath := filepath.Join("testdata", testCaseName, "expected.go")

    if *update {
        // Overwrite expected.go with generated output
        if err := os.WriteFile(expectedPath, actual, 0644); err != nil {
            t.Fatalf("failed to update golden file: %v", err)
        }
        t.Logf("updated golden file: %s", expectedPath)
        return  // Don't compare, just update
    }

    // Normal mode: compare
    expected, _ := os.ReadFile(expectedPath)
    if !bytes.Equal(actual, expected) {
        t.Errorf("output mismatch...")
    }
}
```

### Parallelism Rules

- **Normal mode**: `t.Parallel()` enabled for each test case (fast execution)
- **Update mode (`-update`)**: `t.Parallel()` SKIPPED to prevent race conditions when writing to `expected.go`
- Implementation:
  ```go
  func TestGoldenGeneration(t *testing.T) {
      for _, tc := range testCases {
          t.Run(tc.Name, func(t *testing.T) {
              if !*update {
                  t.Parallel()
              }
              runTest(t, tc.Name)
          })
      }
  }
  ```

### Error Handling

- **Missing `expected.go`**: Hard failure with clear message: `t.Fatalf("missing golden file: %s", expectedPath)`
- **Parse errors in input**: Report with file path and line number
- **Generator errors**: Propagate error message to test output

### Error/Skip Markers (NOT adopted)

Unlike `internal/migrate/`, this implementation does NOT use `// SKIP:` or `// ERROR:` markers because:

1. Code generation should always succeed for valid input (unlike migration which may have edge cases)
2. Error cases are tested separately in existing `generator_test.go` unit tests
3. Golden tests focus on output correctness, not error handling

If error case golden tests are needed later, they can be added in a separate `testdata/errors/` subdirectory.

## Testdata Maintenance

### Initial Setup

Test cases are **selectively copied** from `examples/` (not wholesale directory copy):

```bash
# Copy only .go source files (exclude binaries, generated files)
for example in basic async_parallel complex complex_async sets struct_expansion; do
    mkdir -p internal/kessoku/testdata/$example
    for f in examples/$example/*.go; do
        # Skip generated *_band.go files
        case "$f" in
            *_band.go) continue ;;
        esac
        cp "$f" internal/kessoku/testdata/$example/
    done
    # Rename generated output to expected.go
    cp examples/$example/*_band.go internal/kessoku/testdata/$example/expected.go
done
```

### Keeping in Sync

Testdata is **independent** of examples after initial copy:
- Changes to examples do NOT auto-update testdata
- This is intentional: testdata captures "known good" output at a point in time
- To update: re-run generator with `-update` flag and review changes

### Files to Exclude

When copying from examples, exclude:
- Compiled binaries (e.g., `examples/basic/basic`)
- Generated `*_band.go` files (copied separately as `expected.go`)
- Any `go.mod`/`go.sum` in example subdirectories

## Complexity Tracking

> No violations identified. Implementation follows existing patterns with minimal complexity.
