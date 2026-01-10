# Feature Specification: Golden Test for Code Generation

**Feature Branch**: `003-golden-test`
**Created**: 2026-01-10
**Status**: Draft
**Input**: User description: "コード生成の実装にGolden Testを追加して。"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Verify Code Generation Output (Priority: P1)

When a developer makes changes to the code generation logic in `internal/kessoku/`, they need confidence that their changes produce correct output. Golden Tests provide a way to compare actual generated code against known-good snapshots.

**Why this priority**: This is the core value proposition. Without the ability to verify generated code, developers risk introducing regressions that break user applications.

**Independent Test**: Can be fully tested by running `go test ./internal/kessoku/...` and verifying that generated output matches golden files.

**Acceptance Scenarios**:

1. **Given** a test case directory with input `kessoku.go` file, **When** the code generator runs, **Then** the output matches the corresponding `expected.go` file.
2. **Given** a test case with async providers, **When** the code generator runs, **Then** the output includes correct errgroup and goroutine handling.
3. **Given** a test case with error-returning providers, **When** the code generator runs, **Then** the output includes correct error propagation logic.

---

### User Story 2 - Add New Golden Test Case (Priority: P2)

When a developer adds a new code generation feature, they can easily add a new Golden Test case by creating a new directory with input and expected output files.

**Why this priority**: Extensibility ensures the test suite grows with the codebase. Without easy test addition, developers skip writing tests.

**Independent Test**: Can be tested by creating a new test case directory and verifying it runs automatically with existing tests.

**Acceptance Scenarios**:

1. **Given** a new directory in `internal/kessoku/testdata/` with valid input files, **When** tests run, **Then** the new case is automatically discovered and executed.
2. **Given** test output doesn't match expected, **When** tests run, **Then** clear diff output shows the mismatch.

---

### User Story 3 - Update Golden Files (Priority: P3)

When a developer intentionally changes code generation output, they can update golden files to reflect the new expected behavior.

**Why this priority**: Without an update mechanism, maintaining tests becomes tedious when output format legitimately changes.

**Independent Test**: Can be tested by running tests with update flag and verifying golden files are updated.

**Acceptance Scenarios**:

1. **Given** code generation output has legitimately changed, **When** tests run with `-update` flag, **Then** golden files are updated to match actual output.
2. **Given** update flag is used, **When** tests complete, **Then** developer can review changes via git diff.

---

### Edge Cases

- What happens when a golden file (`expected.go`) doesn't exist for a test case? → Test MUST fail with clear message indicating missing golden file
- How does the system handle test cases with subdirectories? → Discovery is **shallow** (single level only), matching `internal/migrate/` pattern. Cross-package test cases use separate top-level directories
- What happens when input files have syntax errors? → Test MUST report parse error clearly with file path and line number

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST automatically discover all test case directories in `internal/kessoku/testdata/` (shallow discovery, single level)
- **FR-002**: System MUST compare generated output against `expected.go` golden file in each test case directory
- **FR-003**: System MUST provide clear error messages when output differs from expected, showing the diff
- **FR-004**: System MUST support update mode via `-update` test flag (using `flag.Bool`) to regenerate golden files
- **FR-005**: System MUST handle multiple example patterns (basic, async, complex, sets, struct_expansion)
- **FR-006**: System MUST execute each test case in isolation using `t.TempDir()` for generated output
- **FR-007**: System MUST support parallel test execution via `t.Parallel()` when `-update` flag is NOT set
- **FR-008**: When `-update` flag is set, system MUST skip `t.Parallel()` and execute tests sequentially to avoid file clobbering when writing to `expected.go`

### Key Entities

- **Test Case**: A directory in `internal/kessoku/testdata/` containing:
  - Input files: `kessoku.go` and related provider definition files (e.g., `service.go`, `config.go`)
  - Expected output: `expected.go` (the golden file)
- **Golden File**: The `expected.go` file containing pre-approved expected generated code
- **Test Runner**: The `TestGoldenGeneration` function in `internal/kessoku/golden_test.go`

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All test cases derived from existing examples (`examples/basic`, `examples/complex`, etc.) pass
- **SC-002**: Test output clearly identifies which test case failed and shows the difference between expected and actual
- **SC-003**: Developers can add a new test case in under 5 minutes by copying an existing test case directory
- **SC-004**: Test suite runs to completion without flaky failures across 10 consecutive runs
- **SC-005**: `go test ./internal/kessoku/... -update` correctly regenerates all golden files in a single run

## Assumptions

- **Location**: Golden Test infrastructure is located at `internal/kessoku/testdata/`
- **Test Case Creation**: Test cases are **copied** from `examples/` (not symlinked), with generated `*_band.go` renamed to `expected.go`
- **Update Mode**: Controlled via `-update` test flag using `flag.Bool` (idiomatic Go pattern for golden tests)
- **Golden File Naming**: Uses `expected.go` (matching `internal/migrate/` convention, NOT `expected_band.go`)
- **Discovery**: Shallow discovery only (top-level directories in testdata), matching `internal/migrate/` pattern

## Differences from internal/migrate Pattern

This implementation follows the `internal/migrate/testdata/` pattern with these deliberate differences:

| Aspect | internal/migrate | internal/kessoku (this feature) |
|--------|------------------|----------------------------------|
| Input file | `input.go` | `kessoku.go` + provider files |
| Golden file | `expected.go` | `expected.go` (same) |
| Update mode | Not supported | `-update` test flag |
| Discovery | Shallow | Shallow (same) |
| Test isolation | `t.TempDir()` | `t.TempDir()` (same) |
