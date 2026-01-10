# Research: Golden Test for Code Generation

**Date**: 2026-01-10
**Feature**: Golden Test for Code Generation

## Research Areas

### 1. Golden Test Pattern in Go

**Decision**: Follow idiomatic Go golden test pattern with `-update` flag

**Rationale**:
- Standard Go testing uses `flag.Bool` for update mode (e.g., `var update = flag.Bool("update", false, "update golden files")`)
- Existing `internal/migrate/testdata/` pattern provides a project-specific reference
- `t.TempDir()` for test isolation is standard Go 1.15+ practice

**Alternatives Considered**:
- External test frameworks (testify golden, etc.) - Rejected: adds dependency, overkill for file comparison
- Snapshot testing library - Rejected: additional dependency, non-idiomatic

### 2. Test Discovery Pattern

**Decision**: Shallow directory discovery (single level in `testdata/`)

**Rationale**:
- Matches `internal/migrate/` pattern already in codebase
- Simple to implement and maintain
- Each test case is self-contained in its own directory

**Alternatives Considered**:
- Recursive discovery - Rejected: adds complexity, cross-package test cases already handled by separate directories
- Explicit test registration - Rejected: requires maintenance when adding tests

### 3. Input File Structure

**Decision**: Use `kessoku.go` + provider files (not `input.go`)

**Rationale**:
- Matches the actual usage pattern in `examples/`
- Code generator operates on `kessoku.go` files containing `kessoku.Inject` calls
- Provider files (config.go, service.go, etc.) are natural companions

**Alternatives Considered**:
- Single `input.go` file - Rejected: doesn't match real-world usage with multiple provider files

### 4. Golden File Naming

**Decision**: Use `expected.go` (not `expected_band.go`)

**Rationale**:
- Matches `internal/migrate/` convention
- Clear semantic meaning: "expected output"
- Distinguishes from generated `*_band.go` files

**Alternatives Considered**:
- `expected_band.go` - Rejected: redundant suffix, inconsistent with migrate pattern
- `golden.go` - Rejected: less explicit about its purpose

### 5. Parallel Execution

**Decision**: Use `t.Parallel()` when `-update` flag is NOT set; skip parallel when updating

**Rationale**:
- Parallel execution speeds up test suite significantly
- Update mode must be sequential to avoid race conditions when writing `expected.go`
- FR-007 and FR-008 in spec explicitly require this behavior

**Alternatives Considered**:
- Always sequential - Rejected: unnecessarily slow for normal test runs
- Per-file locking during update - Rejected: adds complexity, sequential is simple and correct

### 6. Error Handling for Missing Golden Files

**Decision**: Fail test with clear error message

**Rationale**:
- Prevents silent failures when golden file is missing
- Follows spec edge case requirement
- Developer must explicitly create/update golden files

**Alternatives Considered**:
- Auto-create golden file - Rejected: could mask bugs, unexpected behavior
- Skip test - Rejected: hides missing coverage

### 7. Diff Output Format

**Decision**: Use simple string comparison with full expected/actual output

**Rationale**:
- Matches existing `internal/migrate/migrate_test.go` pattern
- Clear visibility into differences
- No additional dependencies required

**Alternatives Considered**:
- External diff library - Rejected: adds dependency
- Line-by-line diff - Rejected: harder to read for generated code

### 8. Error/Skip Markers from internal/migrate

**Decision**: NOT adopted for kessoku golden tests

**Rationale**:
- `internal/migrate/` uses `// SKIP:` and `// ERROR:` markers for edge cases in wire migration
- Code generation for kessoku should always succeed for valid input
- Error cases are already covered by unit tests in `generator_test.go`
- Golden tests focus purely on output correctness

**Alternatives Considered**:
- Adopt same markers - Rejected: unnecessary complexity for this use case
- Separate error test directory - Deferred: can be added later if needed

## Implementation Notes

### Code Generation Flow

The golden test invokes `Processor.ProcessFiles()` directly:

```go
// 1. Copy input files to t.TempDir()
tmpDir := t.TempDir()
copyInputFiles(srcDir, tmpDir)  // all .go except expected.go

// 2. Run processor on copied kessoku.go
processor := NewProcessor()
processor.ProcessFiles([]string{filepath.Join(tmpDir, "kessoku.go")})

// 3. Processor writes kessoku_band.go to tmpDir (fixed naming)
actual, _ := os.ReadFile(filepath.Join(tmpDir, "kessoku_band.go"))

// 4. Compare with expected.go from testdata
expected, _ := os.ReadFile(filepath.Join(srcDir, "expected.go"))
```

**Why this approach**:
- Generator has fixed output naming: `input.go` → `input_band.go`
- Using `t.TempDir()` keeps generated files out of repo
- Comparing against `expected.go` in testdata (not tmpDir)

### Output Normalization

Before comparison:
1. Run `go/format.Source()` on generated code
2. Ensure deterministic output (stable map iteration, sorted imports)
3. Compare byte-for-byte with `expected.go`

### Test Case Preparation

Test cases are **selectively copied** from `examples/`:

```bash
# Only copy .go source files (not binaries, not *_band.go)
for f in examples/$example/*.go; do
    [[ "$f" == *_band.go ]] && continue
    cp "$f" internal/kessoku/testdata/$example/
done
# Rename generated output to expected.go
cp examples/$example/*_band.go internal/kessoku/testdata/$example/expected.go
```

**Excluded files**:
- Compiled binaries (e.g., `examples/basic/basic`)
- Generated `*_band.go` (copied separately as `expected.go`)

Initial test cases:
- `basic` - Simple linear dependencies
- `async_parallel` - Async provider with parallel execution
- `complex` - Multiple providers with shared dependencies
- `complex_async` - Async providers with complex dependencies
- `sets` - Provider sets
- `struct_expansion` - Struct field injection
