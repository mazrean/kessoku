# Tasks: Golden Test for Code Generation

**Input**: Design documents from `/specs/003-golden-test/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: No explicit test tasks - the feature itself IS a test infrastructure. Validation is done via running the golden tests.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Testdata Infrastructure)

**Purpose**: Create testdata directory structure with test cases from examples

- [X] T001 Create testdata directory at internal/kessoku/testdata/
- [X] T002 [P] Copy basic example to internal/kessoku/testdata/basic/ (exclude binary and *_band.go, rename kessoku_band.go to expected.go)
- [X] T003 [P] Copy async_parallel example to internal/kessoku/testdata/async_parallel/ (exclude binary, rename *_band.go to expected.go)
- [X] T004 [P] Copy complex example to internal/kessoku/testdata/complex/ (exclude binary, rename *_band.go to expected.go)
- [X] T005 [P] Copy complex_async example to internal/kessoku/testdata/complex_async/ (exclude binary, rename *_band.go to expected.go)
- [X] T006 [P] Copy sets example to internal/kessoku/testdata/sets/ (exclude binary, rename *_band.go to expected.go)
- [X] T007 [P] Copy struct_expansion example to internal/kessoku/testdata/struct_expansion/ (exclude binary, rename *_band.go to expected.go)

**Checkpoint**: Testdata directories created with input files and expected.go golden files

---

## Phase 2: User Story 1 - Verify Code Generation Output (Priority: P1)

**Goal**: Enable developers to verify that code generation produces correct output by comparing against golden files

**Independent Test**: Run `go test -v -run TestGoldenGeneration ./internal/kessoku/...` and verify all test cases pass

### Implementation for User Story 1

- [X] T008 [US1] Create golden_test.go scaffold with TestGoldenGeneration function in internal/kessoku/golden_test.go
- [X] T009 [US1] Implement test case discovery using os.ReadDir for shallow directory scanning in internal/kessoku/golden_test.go
- [X] T010 [US1] Implement copyInputFiles helper to copy all .go files except expected.go to t.TempDir() in internal/kessoku/golden_test.go (Note: Simplified to run directly on testdata)
- [X] T011 [US1] Implement runTest function that invokes Processor.ProcessFiles() on copied files in internal/kessoku/golden_test.go
- [X] T012 [US1] Implement output comparison between generated kessoku_band.go and expected.go in internal/kessoku/golden_test.go
- [X] T013 [US1] Add clear diff output when comparison fails (show expected vs got) in internal/kessoku/golden_test.go
- [X] T014 [US1] Add error handling for missing expected.go with clear error message in internal/kessoku/golden_test.go
- [X] T015 [US1] Add error handling for parse errors with file path and line number in internal/kessoku/golden_test.go
- [X] T016 [US1] Run `go test -v -run TestGoldenGeneration ./internal/kessoku/...` and verify all 6 test cases pass

**Checkpoint**: Core golden test infrastructure complete - developers can verify code generation output

---

## Phase 3: User Story 2 - Add New Golden Test Case (Priority: P2)

**Goal**: Enable developers to easily add new test cases by creating directories with input and expected files

**Independent Test**: Create a minimal new test case directory and verify it runs automatically with existing tests

### Implementation for User Story 2

- [X] T017 [US2] Verify automatic discovery works by adding a minimal test case to internal/kessoku/testdata/ (Verified: all 6 test cases discovered automatically)
- [X] T018 [US2] Enhance diff output to show test case name prominently when mismatch occurs in internal/kessoku/golden_test.go
- [X] T019 [US2] Add validation that kessoku.go exists in each test case directory in internal/kessoku/golden_test.go

**Checkpoint**: New test cases are automatically discovered and executed

---

## Phase 4: User Story 3 - Update Golden Files (Priority: P3)

**Goal**: Enable developers to update golden files when code generation output legitimately changes

**Independent Test**: Run `go test -run TestGoldenGeneration ./internal/kessoku/... -update` and verify expected.go files are updated

### Implementation for User Story 3

- [X] T020 [US3] Add -update flag using flag.Bool at package level in internal/kessoku/golden_test.go
- [X] T021 [US3] Implement update mode that overwrites expected.go with generated output in internal/kessoku/golden_test.go
- [X] T022 [US3] Skip t.Parallel() when -update flag is set to prevent race conditions in internal/kessoku/golden_test.go
- [X] T023 [US3] Add log message when golden file is updated in internal/kessoku/golden_test.go
- [X] T024 [US3] Run `go test -v -run TestGoldenGeneration ./internal/kessoku/... -update` and verify expected.go files in internal/kessoku/testdata/*/expected.go are overwritten

**Checkpoint**: Update mode fully functional - developers can regenerate golden files with -update flag

---

## Phase 5: Polish & Validation

**Purpose**: Final validation and documentation

- [X] T025 [P] Run full test suite `go test -v ./internal/kessoku/...` and verify all tests pass
- [X] T026 [P] Verify parallel execution by running `go test -v -run TestGoldenGeneration ./internal/kessoku/... -count=3` multiple times
- [X] T027 [P] Verify sequential execution by running `go test -v -run TestGoldenGeneration ./internal/kessoku/... -update` completes without race conditions
- [X] T028 Update CLAUDE.md with golden test commands if needed
- [X] T029 Run validation scenarios from specs/003-golden-test/quickstart.md (all commands in Running Golden Tests and Adding a New Test Case sections)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **User Story 1 (Phase 2)**: Depends on Setup completion (T001-T007)
- **User Story 2 (Phase 3)**: Depends on User Story 1 completion (T008-T016)
- **User Story 3 (Phase 4)**: Depends on User Story 1 completion (T008-T016), can run parallel to US2
- **Polish (Phase 5)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Setup (Phase 1) - Core functionality
- **User Story 2 (P2)**: Depends on US1 core infrastructure being complete
- **User Story 3 (P3)**: Depends on US1 core infrastructure, independent of US2

### Within Each User Story

- T008 scaffold must be complete before other US1 tasks
- T009 discovery must be complete before T016 verification
- T010-T015 can be implemented incrementally
- T020 flag must be complete before T021-T024

### Parallel Opportunities

**Phase 1 - Setup:**
```bash
# All testdata copy tasks can run in parallel:
Task: T002 "Copy basic example..."
Task: T003 "Copy async_parallel example..."
Task: T004 "Copy complex example..."
Task: T005 "Copy complex_async example..."
Task: T006 "Copy sets example..."
Task: T007 "Copy struct_expansion example..."
```

**Phase 5 - Polish:**
```bash
# Validation tasks can run in parallel:
Task: T025 "Run full test suite..."
Task: T026 "Verify parallel execution..."
Task: T027 "Verify sequential execution..."
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (create testdata with all 6 examples)
2. Complete Phase 2: User Story 1 (core golden test runner)
3. **STOP and VALIDATE**: Run `go test -v -run TestGoldenGeneration ./internal/kessoku/...`
4. All 6 test cases should pass

### Incremental Delivery

1. Complete Setup + US1 -> Core golden tests working (MVP!)
2. Add US2 -> Verify new test case addition works
3. Add US3 -> -update flag support
4. Polish -> Full validation and documentation

### Implementation Notes

- Generator writes to `*_band.go` next to input file (fixed naming in processor.go)
- Test uses t.TempDir() so generated output goes there, not in testdata
- Comparison uses expected.go from original testdata directory
- No output normalization needed if generator already produces stable output

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Test cases derived from existing examples/: basic, async_parallel, complex, complex_async, sets, struct_expansion
