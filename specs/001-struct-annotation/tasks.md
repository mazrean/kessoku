---
description: "Task list for Struct Annotation for Field Expansion feature"
---

# Tasks: Struct Annotation for Field Expansion

**Input**: Design documents from `/specs/001-struct-annotation/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: Included per CLAUDE.md TDD guidelines

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Project type**: Single Go module with tools submodule
- **Source code**: Repository root (`annotation.go`, `internal/kessoku/`)
- **Tests**: Embedded in `*_test.go` files alongside source
- **Examples**: `examples/` directory

---

## Phase 1: Setup

**Purpose**: Project preparation and branch verification

- [X] T001 Verify branch `001-struct-annotation` is checked out and up to date in repository root
- [X] T002 Run existing tests to ensure baseline passes with `go test -v ./...` in repository root
- [X] T003 Run linter to verify clean baseline with `go tool tools lint ./...` in repository root

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types and structures that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### 2.1 Provider Type Extensions

- [X] T004 Add `ProviderTypeStruct` and `ProviderTypeFieldAccess` constants in `internal/kessoku/provider.go`
- [X] T005 Add `StructFieldSpec` struct in `internal/kessoku/provider.go`
- [X] T006 Extend `ProviderSpec` with `StructType`, `StructFields`, `SourceField`, and `DeclOrder` fields in `internal/kessoku/provider.go`
- [X] T007 Add `InjectorFieldAccessStmt` struct implementing `InjectorStmt` interface in `internal/kessoku/provider.go`

### 2.2 Public API

- [X] T008 [P] Add `structProvider[T any]` type with `provide()` method in `annotation.go`
- [X] T009 Add `Struct[T any]() structProvider[T]` function with documentation in `annotation.go`

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Expand Config Struct Fields (Priority: P1) 🎯 MVP

**Goal**: Automatically expand all exported fields of a struct as individual dependencies

**Independent Test**: Create a struct with multiple fields, use `kessoku.Struct` to expand them, verify each field becomes an injectable dependency

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T010 [US1] Add test `TestExtractExportedFields` for basic field extraction in `internal/kessoku/parser_test.go`
- [X] T011 [US1] Add test `TestExtractExportedFieldsUnexported` for unexported field filtering in `internal/kessoku/parser_test.go`
- [X] T012 [US1] Add test `TestExtractExportedFieldsAlphabeticalOrder` for alphabetical ordering in `internal/kessoku/parser_test.go`
- [X] T013 [US1] Add test `TestParseStructProvider` for structProvider parsing in `internal/kessoku/parser_test.go`
- [X] T014 [US1] Add test `TestParseStructProviderNonStructError` for non-struct type error (SC-004) in `internal/kessoku/parser_test.go`
- [X] T015 [P] [US1] Add test `TestStructProviderExpansion` for field expansion in graph in `internal/kessoku/graph_test.go`
- [X] T016 [US1] Add test `TestStructProviderCountAndTypes` for SC-001 verification in `internal/kessoku/graph_test.go`
- [X] T017 [US1] Add test `TestStructProviderMissingProvider` for missing provider error (FR-010, SC-004) in `internal/kessoku/graph_test.go`
- [X] T018 [US1] Add test `TestStructProviderDuplicateType` for duplicate type error (FR-005) in `internal/kessoku/graph_test.go`
- [X] T019 [US1] Add test `TestStructMultipleStructOrdering` for FR-012 multiple struct ordering in `internal/kessoku/graph_test.go`
- [X] T020 [P] [US1] Add integration test `TestStructBasicFieldExpansion` in `internal/kessoku/processor_test.go`

### Implementation for User Story 1

- [X] T021 [US1] Implement `extractExportedFields()` helper function in `internal/kessoku/parser.go`
- [X] T022 [US1] Add `structProvider` case to `parseProviderType()` in `internal/kessoku/parser.go` (line ~446)
- [X] T023 [US1] Update `parseProviderArgument()` to populate `StructType` and `StructFields` for struct providers in `internal/kessoku/parser.go` (line ~347)
- [X] T024 [P] [US1] Implement first pass (skip struct providers) in `NewGraph()` in `internal/kessoku/graph.go` (line ~261)
- [X] T025 [US1] Implement second pass (expand struct providers) in `NewGraph()` with FR-010, FR-005 error handling in `internal/kessoku/graph.go`
- [X] T026 [US1] Add `DeclOrder` assignment to function providers in `NewGraph()` first pass in `internal/kessoku/graph.go`
- [X] T027 [US1] Add `DeclOrder` assignment to synthetic field accessors in `NewGraph()` second pass in `internal/kessoku/graph.go`
- [X] T028 [US1] Update `Build()` to sort by (TopoLevel, DeclOrder) for FR-012 deterministic ordering in `internal/kessoku/graph.go`
- [X] T029 [US1] Add `ProviderTypeFieldAccess` handling in `buildPoolStmts()` to emit `InjectorFieldAccessStmt` in `internal/kessoku/graph.go` (line ~1037)
- [X] T030 [P] [US1] Implement `InjectorFieldAccessStmt.Stmt()` method to generate field access AST in `internal/kessoku/provider.go`
- [X] T031 [US1] Implement `InjectorFieldAccessStmt.HasAsync()` returning false in `internal/kessoku/provider.go`
- [X] T032 [US1] Run tests and verify User Story 1 acceptance scenarios pass in repository root with `go test -v ./internal/kessoku/...`

**Checkpoint**: User Story 1 complete - basic struct field expansion works independently

---

## Phase 4: User Story 2 - Type-Safe Field Access (Priority: P1)

**Goal**: Generate type-safe field extraction code that compiles without errors

**Independent Test**: Verify generated code correctly extracts fields with proper types and compilation succeeds

### Tests for User Story 2

- [X] T033 [P] [US2] Add test `TestFieldAccessStmtGeneration` for `InjectorFieldAccessStmt.Stmt()` AST in `internal/kessoku/provider_test.go`
- [X] T034 [US2] Add test `TestStructDifferentFieldTypes` for string, int, custom type fields in `internal/kessoku/processor_test.go`
- [X] T035 [US2] Add test `TestStructPointerAndValueFields` for pointer and value field handling in `internal/kessoku/processor_test.go`
- [X] T036 [US2] Add test `TestGeneratedCodeCompiles` for SC-003 compilation verification in `internal/kessoku/processor_test.go`

### Implementation for User Story 2

- [X] T037 [US2] Verify import collection works for field types via existing `collectImportsFromType()` in `internal/kessoku/provider.go`
- [X] T038 [US2] Update field accessor parameter creation to use `NewInjectorParamWithImports()` in `internal/kessoku/graph.go`
- [X] T039 [US2] Run tests and verify User Story 2 acceptance scenarios pass in repository root with `go test -v ./internal/kessoku/...`

**Checkpoint**: User Story 2 complete - type-safe field access works with various field types

---

## Phase 5: User Story 3 - Use with Existing Annotations (Priority: P2)

**Goal**: Enable `kessoku.Struct` to work alongside Provide, Async, Bind, and Set annotations

**Independent Test**: Combine `kessoku.Struct` with existing annotations and verify correct behavior

### Tests for User Story 3

- [X] T040 [US3] Add test `TestStructWithProvide` for Provide + Struct integration in `internal/kessoku/processor_test.go`
- [X] T041 [US3] Add test `TestStructInsideSet` for Struct inside Set in `internal/kessoku/processor_test.go`
- [X] T042 [US3] Add test `TestStructMissingProvider` for error when struct provider missing in `internal/kessoku/processor_test.go`
- [X] T043 [US3] Add test `TestStructWithAsync` for Async + Struct integration in `internal/kessoku/processor_test.go`
- [X] T044 [US3] Add test `TestStructWithBind` for Bind + Struct integration in `internal/kessoku/processor_test.go`
- [X] T045 [US3] Add test `TestStructSetInPlaceOrdering` for FR-012 Set in-place expansion in `internal/kessoku/processor_test.go`
- [X] T046 [P] [US3] Add test `TestStructTypeMismatch` for pointer/value type mismatch error (FR-006, SC-004) in `internal/kessoku/graph_test.go`

### Implementation for User Story 3

- [X] T047 [US3] Implement type mismatch detection in `NewGraph()` second pass for FR-006 in `internal/kessoku/graph.go`
- [X] T048 [US3] Verify Set recursive parsing correctly handles Struct via existing `parseProviderArgument()` in `internal/kessoku/parser.go`
- [X] T049 [US3] Verify Async integration works via existing `IsWait` mechanism (no code changes expected) in `internal/kessoku/graph.go`
- [X] T050 [US3] Run tests and verify User Story 3 acceptance scenarios pass in repository root with `go test -v ./internal/kessoku/...`

**Checkpoint**: User Story 3 complete - Struct integrates correctly with all existing annotations

---

## Phase 6: User Story 4 - Handle Embedded Fields (Priority: P3)

**Goal**: Handle embedded (anonymous) struct fields correctly

**Independent Test**: Create struct with embedded fields and verify embedded type becomes a dependency

### Tests for User Story 4

- [X] T051 [US4] Add test `TestStructEmbeddedValue` for embedded value type in `internal/kessoku/processor_test.go`
- [X] T052 [US4] Add test `TestStructEmbeddedPointer` for embedded pointer type in `internal/kessoku/processor_test.go`
- [X] T053 [US4] Add test `TestStructNoRecursiveExpansion` to verify nested fields NOT expanded in `internal/kessoku/processor_test.go`
- [X] T054 [US4] Add test `TestStructUnexportedEmbedded` for unexported embedded type ignored in `internal/kessoku/processor_test.go`
- [X] T055 [P] [US4] Add test `TestExtractExportedFieldsEmbedded` for FR-009 Anonymous flag in `internal/kessoku/parser_test.go`

### Implementation for User Story 4

- [X] T056 [US4] Verify `extractExportedFields()` correctly sets `Anonymous` flag for embedded fields in `internal/kessoku/parser.go`
- [X] T057 [US4] Verify unexported embedded types are filtered via `field.Exported()` check in `internal/kessoku/parser.go`
- [X] T058 [US4] Run tests and verify User Story 4 acceptance scenarios pass in repository root with `go test -v ./internal/kessoku/...`

**Checkpoint**: User Story 4 complete - embedded fields handled correctly

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, examples, and final validation

- [X] T059 Create example directory at `examples/struct_expansion/`
- [X] T060 Add basic Struct usage example in `examples/struct_expansion/main.go`
- [X] T061 Add Inject declaration example in `examples/struct_expansion/kessoku.go`
- [X] T062 [P] Add test `TestStructWithNoExportedFields` for FR-011 silent success in `internal/kessoku/processor_test.go`
- [ ] T063 Update Struct annotation documentation in `README.md`
- [X] T064 Run full test suite with `go test -v ./...` in repository root
- [X] T065 Run linter with `go tool tools lint ./...` in repository root
- [X] T066 Verify quickstart.md examples work by running generated code in `examples/struct_expansion/`
- [ ] T067 Run API compatibility check with `go tool tools apicompat github.com/mazrean/kessoku@latest github.com/mazrean/kessoku` in repository root

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - US1 (P1): Core expansion - no dependencies on other stories
  - US2 (P1): Type safety - builds on US1 implementation but independently testable
  - US3 (P2): Integration - tests existing annotation interactions
  - US4 (P3): Embedded fields - extends field extraction logic
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

```
Phase 1 (Setup)
    │
    ▼
Phase 2 (Foundational) ──── BLOCKS ALL STORIES
    │
    ├──────────────────────────────────────┐
    │                                      │
    ▼                                      ▼
Phase 3 (US1: Core Expansion)     (US2, US3, US4 can start after Foundational)
    │
    ▼
Phase 4 (US2: Type Safety)
    │
    ▼
Phase 5 (US3: Integration)
    │
    ▼
Phase 6 (US4: Embedded Fields)
    │
    ▼
Phase 7 (Polish)
```

### Within Each User Story

1. Tests MUST be written and FAIL before implementation
2. Parser changes before graph changes
3. Graph changes before generator integration
4. Core implementation before integration verification
5. Story complete before moving to next priority

### Parallel Opportunities

Tasks marked with [P] can run in parallel because they operate on different files:

**Phase 2 (Foundational)**:
- T008 (`annotation.go`) can run in parallel with T004-T007 (`internal/kessoku/provider.go`)

**Phase 3 (US1)**:
- T015 (`graph_test.go`) can run in parallel with T010-T014 (`parser_test.go`)
- T020 (`processor_test.go`) can run in parallel with parser and graph tests
- T024 (`graph.go`) can run in parallel with T021-T023 (`parser.go`)
- T030 (`provider.go`) can run in parallel with T024-T029 (`graph.go`)

**Phase 4 (US2)**:
- T033 (`provider_test.go`) can run in parallel with T034-T036 (`processor_test.go`)

**Phase 5 (US3)**:
- T046 (`graph_test.go`) can run in parallel with T040-T045 (`processor_test.go`)

**Phase 6 (US4)**:
- T055 (`parser_test.go`) can run in parallel with T051-T054 (`processor_test.go`)

**Phase 7 (Polish)**:
- T062 (`processor_test.go`) can run in parallel with T059-T061 (`examples/`)

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Basic struct field expansion works - can demo/validate

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Core expansion works (MVP!)
3. Add User Story 2 → Test independently → Type-safe generation verified
4. Add User Story 3 → Test independently → All annotations integrate
5. Add User Story 4 → Test independently → Embedded fields supported
6. Each story adds value without breaking previous stories

### Suggested MVP Scope

**User Story 1 only** - This delivers:
- Basic `kessoku.Struct[T]()` annotation
- Field expansion for structs with exported fields
- Alphabetical field ordering for deterministic output
- Basic error messages for common issues

After MVP validation, proceed with US2-US4 for full feature completeness.

---

## Task Summary

| Phase | Task Count | Parallel Tasks |
|-------|------------|----------------|
| Phase 1: Setup | 3 | 0 |
| Phase 2: Foundational | 6 | 1 |
| Phase 3: US1 | 23 | 4 |
| Phase 4: US2 | 7 | 1 |
| Phase 5: US3 | 11 | 1 |
| Phase 6: US4 | 8 | 1 |
| Phase 7: Polish | 9 | 1 |
| **Total** | **67** | **9** |

### Tasks per User Story

- **US1 (P1)**: 23 tasks (core expansion)
- **US2 (P1)**: 7 tasks (type safety)
- **US3 (P2)**: 11 tasks (integration)
- **US4 (P3)**: 8 tasks (embedded fields)

### Independent Test Criteria

| Story | Independent Test Criteria |
|-------|---------------------------|
| US1 | Struct with multiple fields → each field becomes injectable dependency |
| US2 | Generated code compiles; field types correctly extracted |
| US3 | Struct + Provide/Async/Bind/Set combinations work correctly |
| US4 | Embedded fields become dependencies; nested fields NOT expanded |

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- Tests are written first per TDD methodology (CLAUDE.md guidelines)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Run `go tool tools lint ./...` after code changes
- Run `go test -v ./...` to verify tests pass
