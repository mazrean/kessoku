# Tasks: Wire to Kessoku Migration Tool

**Input**: Design documents from `/specs/001-wire-migrate/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli.md, quickstart.md

**Tests**: Golden file tests included per plan.md milestones.

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1-US5 for user stories, Setup/Found/Polish for infrastructure)
- Include exact file paths in descriptions

## Path Conventions

- **Source**: `internal/migrate/` for migration module
- **Tests**: `internal/migrate/*_test.go` with `testdata/` golden files
- **CLI**: `internal/config/config.go` for CLI configuration

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 [Setup] Create `internal/migrate/` directory with `mkdir -p internal/migrate`
- [X] T002 [Setup] Define `WirePatternKind`, `KessokuPatternKind`, `WarningCode`, `ParseErrorKind`, `MergeErrorKind` enums in `internal/migrate/patterns.go`
- [X] T003 [Setup] Define `WirePattern` interface, `WireNewSet`, `WireBind`, `WireValue`, `WireInterfaceValue`, `WireStruct`, `WireFieldsOf`, `WireProviderFunc` structs in `internal/migrate/patterns.go`
- [X] T004 [Setup] Define `KessokuPattern` interface, `KessokuSet`, `KessokuProvide`, `KessokuBind`, `KessokuValue` structs in `internal/migrate/patterns.go`
- [X] T005 [Setup] Define `Warning`, `ParseError`, `MergeError`, `MigrationResult`, `MergedOutput`, `ImportSpec` types in `internal/migrate/patterns.go`
- [X] T006 [P] [Setup] Create `internal/migrate/testdata/` directory structure with `mkdir -p internal/migrate/testdata`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

- [X] T007 [Found] Refactor CLI struct to use Kong subcommand pattern with `Generate` (default) and `Migrate` commands in `internal/config/config.go`
- [X] T008 [Found] Define `MigrateCmd` struct with `Output` (`-o`, default `kessoku.go`) and `Files` (`arg`) in `internal/config/config.go`
- [X] T009 [Found] Implement `Run()` method for `MigrateCmd` that invokes migrator in `internal/config/config.go`
- [X] T010 [Found] Create `Migrator` struct with `Parser`, `Transformer`, `Writer` fields and constructor in `internal/migrate/migrate.go`
- [X] T011 [Found] Implement `packages.Load` configuration with `NeedTypes`, `NeedSyntax`, `NeedTypesInfo` in `internal/migrate/migrate.go`
- [X] T012 [Found] Implement `convertPackageError()` to convert `packages.Error` to `ParseError` in `internal/migrate/migrate.go`
- [X] T013 [Found] Create `Parser` struct and `NewParser()` constructor in `internal/migrate/parser.go`
- [X] T014 [Found] Implement wire import detection (`github.com/google/wire`) with alias support in `internal/migrate/parser.go`
- [X] T015 [Found] Create `Transformer` struct and `NewTransformer()` constructor in `internal/migrate/transformer.go`
- [X] T016 [Found] Create `Writer` struct and `NewWriter()` constructor in `internal/migrate/writer.go`
- [X] T017 [Found] Implement golden file test runner framework in `internal/migrate/migrate_test.go`

**Checkpoint**: Foundation ready - user story implementation can begin

**Note**: T018 (help verification) moved to US1 as it depends on CLI implementation.

---

## Phase 3: User Story 1 - Basic Wire File Migration (Priority: P1)

**Goal**: Convert basic wire patterns (NewSet, Bind, Value) to kessoku format with CLI support

**Independent Test**: Run `kessoku migrate` on a file with wire.NewSet, wire.Bind, wire.Value and verify correct kessoku output

### Tests for User Story 1

- [X] T018 [US1] Verify `kessoku migrate --help` displays help message (scenario 1.1) in `internal/migrate/migrate_test.go`
- [X] T019 [P] [US1] Create golden test input `internal/migrate/testdata/basic/input.go` with wire.NewSet containing provider function
- [X] T020 [P] [US1] Create golden test expected `internal/migrate/testdata/basic/expected.go` with kessoku.Set + kessoku.Provide
- [X] T021 [P] [US1] Create golden test input `internal/migrate/testdata/bind/input.go` with wire.Bind[Interface, *Impl]()
- [X] T022 [P] [US1] Create golden test expected `internal/migrate/testdata/bind/expected.go` with kessoku.Bind[Interface](kessoku.Provide(NewImpl))
- [X] T023 [P] [US1] Create golden test input `internal/migrate/testdata/value/input.go` with wire.Value(v)
- [X] T024 [P] [US1] Create golden test expected `internal/migrate/testdata/value/expected.go` with kessoku.Value(v)
- [X] T025 [P] [US1] Create golden test input `internal/migrate/testdata/import_replace/input.go` with wire import
- [X] T026 [P] [US1] Create golden test expected `internal/migrate/testdata/import_replace/expected.go` with kessoku import (scenario 1.7)
- [X] T027 [US1] Add test for default output path `kessoku.go` (scenario 1.5) in `internal/migrate/migrate_test.go`
- [X] T028 [US1] Add test for custom output path `-o output.go` (scenario 1.6) in `internal/migrate/migrate_test.go`

### Implementation for User Story 1

#### Parser Implementation

- [X] T029 [US1] Implement AST visitor to detect `wire.NewSet` calls in `internal/migrate/parser.go`
- [X] T030 [US1] Implement `parseNewSet()` to extract variable name and elements in `internal/migrate/parser.go`
- [X] T031 [US1] Implement detection of provider function references within NewSet in `internal/migrate/parser.go`
- [X] T032 [US1] Implement `parseBind()` to detect `wire.Bind(new(Interface), new(*Impl))` in `internal/migrate/parser.go`
- [X] T033 [US1] Implement pointer type unwrapping for Bind implementation type in `internal/migrate/parser.go`
- [X] T034 [US1] Implement `parseValue()` to detect `wire.Value(expr)` in `internal/migrate/parser.go`
- [X] T035 [US1] Implement `ExtractPatterns()` orchestrator method in `internal/migrate/parser.go`

#### Transformer Implementation

- [X] T036 [US1] Implement `transformNewSet()` to convert `WireNewSet` to `KessokuSet` in `internal/migrate/transformer.go`
- [X] T037 [US1] Implement `transformProviderFunc()` to wrap functions in `kessoku.Provide()` in `internal/migrate/transformer.go`
- [X] T038 [US1] Implement constructor lookup for Bind (find `New{TypeName}` in package scope) in `internal/migrate/transformer.go`
- [X] T039 [US1] Implement `transformBind()` to convert `WireBind` to `KessokuBind` + `KessokuProvide` in `internal/migrate/transformer.go`
- [X] T040 [US1] Return `ParseError` with `ParseErrorMissingConstructor` when constructor not found in `internal/migrate/transformer.go`
- [X] T041 [US1] Implement `transformValue()` to convert `WireValue` to `KessokuValue` in `internal/migrate/transformer.go`
- [X] T042 [US1] Implement `Transform()` orchestrator method in `internal/migrate/transformer.go`

#### Writer Implementation

- [X] T043 [US1] Implement AST generation for `kessoku.Set(...)` in `internal/migrate/writer.go`
- [X] T044 [US1] Implement AST generation for `kessoku.Provide(fn)` in `internal/migrate/writer.go`
- [X] T045 [US1] Implement AST generation for `kessoku.Bind[I](provider)` with type parameters in `internal/migrate/writer.go`
- [X] T046 [US1] Implement AST generation for `kessoku.Value(expr)` in `internal/migrate/writer.go`
- [X] T047 [US1] Implement import replacement (`github.com/google/wire` → `github.com/mazrean/kessoku`) in `internal/migrate/writer.go`
- [X] T048 [US1] Implement `go/format` integration for output formatting in `internal/migrate/writer.go`
- [X] T049 [US1] Implement `Write()` method to generate output file in `internal/migrate/writer.go`

#### Orchestration

- [X] T050 [US1] Implement `MigrateFiles()` orchestrator method in `internal/migrate/migrate.go`
- [X] T051 [US1] Implement default output path (`kessoku.go`) handling in `internal/migrate/migrate.go`
- [X] T052 [US1] Implement custom output path (`-o` flag) handling in `internal/migrate/migrate.go`

**Checkpoint**: User Story 1 complete - basic wire migration works (acceptance scenarios 1.1-1.7)

---

## Phase 4: User Story 2 - InterfaceValue Migration (Priority: P2)

**Goal**: Convert wire.InterfaceValue to kessoku.Bind + kessoku.Value combination

**Independent Test**: Run migrate on file with wire.InterfaceValue and verify kessoku.Bind[I](kessoku.Value(v)) output

### Tests for User Story 2

- [X] T053 [P] [US2] Create golden test input `internal/migrate/testdata/interface_value/input.go` with wire.InterfaceValue(new(Interface), value)
- [X] T054 [P] [US2] Create golden test expected `internal/migrate/testdata/interface_value/expected.go` with kessoku.Bind[Interface](kessoku.Value(value))

### Implementation for User Story 2

- [X] T055 [US2] Implement `parseInterfaceValue()` to detect `wire.InterfaceValue(new(Interface), expr)` in `internal/migrate/parser.go`
- [X] T056 [US2] Add InterfaceValue case to `ExtractPatterns()` in `internal/migrate/parser.go`
- [X] T057 [US2] Implement `transformInterfaceValue()` to convert to `KessokuBind` + `KessokuValue` in `internal/migrate/transformer.go`
- [X] T058 [US2] Add InterfaceValue case to `Transform()` in `internal/migrate/transformer.go`

**Checkpoint**: User Story 2 complete - InterfaceValue migration works (acceptance scenario 2.1)

---

## Phase 5: User Story 3 - Struct Injection Migration (Priority: P2)

**Goal**: Convert wire.Struct to kessoku.Provide with generated constructor function literal

**Independent Test**: Run migrate on file with wire.Struct and verify kessoku.Provide(func(...) *T {...}) output

### Tests for User Story 3

- [X] T059 [P] [US3] Create golden test input `internal/migrate/testdata/struct_all/input.go` with wire.Struct(new(Config), "*")
- [X] T060 [P] [US3] Create golden test expected `internal/migrate/testdata/struct_all/expected.go` with kessoku.Provide func literal for all fields
- [X] T061 [P] [US3] Create golden test input `internal/migrate/testdata/struct_fields/input.go` with wire.Struct(new(Config), "Field1", "Field2")
- [X] T062 [P] [US3] Create golden test expected `internal/migrate/testdata/struct_fields/expected.go` with kessoku.Provide func literal for selected fields

### Implementation for User Story 3

- [X] T063 [US3] Implement `parseStruct()` to detect `wire.Struct(new(Type), fields...)` in `internal/migrate/parser.go`
- [X] T064 [US3] Implement field extraction from `types.Struct` (including unexported fields) in `internal/migrate/parser.go`
- [X] T065 [US3] Add Struct case to `ExtractPatterns()` in `internal/migrate/parser.go`
- [X] T066 [US3] Implement `transformStruct()` to generate func literal AST in `internal/migrate/transformer.go`
- [X] T067 [US3] Handle "*" wildcard to include all struct fields in `internal/migrate/transformer.go`
- [X] T068 [US3] Handle explicit field list to include only specified fields in `internal/migrate/transformer.go`
- [X] T069 [US3] Generate function parameters with field types in `internal/migrate/transformer.go`
- [X] T070 [US3] Generate struct literal with field assignments in `internal/migrate/transformer.go`
- [X] T071 [US3] Add Struct case to `Transform()` in `internal/migrate/transformer.go`

**Checkpoint**: User Story 3 complete - Struct migration works (acceptance scenarios 3.1-3.2)

---

## Phase 6: User Story 4 - FieldsOf Migration (Priority: P3)

**Goal**: Convert wire.FieldsOf to kessoku.Provide with field accessor function

**Independent Test**: Run migrate on file with wire.FieldsOf and verify kessoku.Provide(func(c *Config) FieldType {...}) output

### Tests for User Story 4

- [X] T072 [P] [US4] Create golden test input `internal/migrate/testdata/fieldsof/input.go` with wire.FieldsOf(new(Config), "DB", "Cache")
- [X] T073 [P] [US4] Create golden test expected `internal/migrate/testdata/fieldsof/expected.go` with kessoku.Provide func returning multiple values

### Implementation for User Story 4

- [X] T074 [US4] Implement `parseFieldsOf()` to detect `wire.FieldsOf(new(Type), fields...)` in `internal/migrate/parser.go`
- [X] T075 [US4] Validate field names exist in struct type in `internal/migrate/parser.go`
- [X] T076 [US4] Add FieldsOf case to `ExtractPatterns()` in `internal/migrate/parser.go`
- [X] T077 [US4] Implement `transformFieldsOf()` to generate accessor func literal in `internal/migrate/transformer.go`
- [X] T078 [US4] Generate function with struct pointer parameter in `internal/migrate/transformer.go`
- [X] T079 [US4] Generate multiple return values (one per field) in `internal/migrate/transformer.go`
- [X] T080 [US4] Generate return statement with field accesses in `internal/migrate/transformer.go`
- [X] T081 [US4] Add FieldsOf case to `Transform()` in `internal/migrate/transformer.go`

**Checkpoint**: User Story 4 complete - FieldsOf migration works (acceptance scenario 4.1)

---

## Phase 7: User Story 5 - Multiple File Migration (Priority: P3)

**Goal**: Merge multiple wire files into single kessoku output with validation

**Independent Test**: Run migrate on multiple wire files and verify single merged kessoku.go with deduplicated imports

### Tests for User Story 5

- [X] T082 [P] [US5] Create golden test input `internal/migrate/testdata/merge/a.go` with wire patterns
- [X] T083 [P] [US5] Create golden test input `internal/migrate/testdata/merge/b.go` with different wire patterns
- [X] T084 [P] [US5] Create golden test expected `internal/migrate/testdata/merge/expected.go` with merged output and deduplicated imports (EC-007)
- [X] T085 [US5] Add test for package mismatch error detection (scenario 5.3) in `internal/migrate/migrate_test.go`
- [X] T086 [US5] Add test for name collision error detection (scenario 5.2) in `internal/migrate/migrate_test.go`
- [X] T087 [US5] Add test for directory path input with multiple files (scenario 5.1) in `internal/migrate/migrate_test.go`
- [X] T088 [US5] Add test for import deduplication (scenario 5.4) in `internal/migrate/migrate_test.go`

### Implementation for User Story 5

- [X] T089 [US5] Implement `mergeResults()` method in `internal/migrate/migrate.go`
- [X] T090 [US5] Implement package name validation (all files must have same package) in `internal/migrate/migrate.go`
- [X] T091 [US5] Return `MergeError` with `MergeErrorPackageMismatch` when packages differ in `internal/migrate/migrate.go`
- [X] T092 [US5] Implement identifier collision detection across files in `internal/migrate/migrate.go`
- [X] T093 [US5] Return `MergeError` with `MergeErrorNameCollision` when identifiers conflict in `internal/migrate/migrate.go`
- [X] T094 [US5] Implement import deduplication in merge result in `internal/migrate/migrate.go`
- [X] T095 [US5] Implement `MergedOutput` assembly with combined declarations in `internal/migrate/migrate.go`
- [X] T096 [US5] Update `Writer.Write()` to handle `MergedOutput` in `internal/migrate/writer.go`

**Checkpoint**: User Story 5 complete - multi-file migration works (acceptance scenarios 5.1-5.4)

---

## Phase 8: Edge Cases & Polish

**Purpose**: Edge case handling, warnings, logging, and code quality

### Edge Case Tests

- [X] T097 [P] [Polish] Create test for EC-001 (no wire patterns) in `internal/migrate/testdata/ec001_no_patterns/`
- [X] T098 [P] [Polish] Create test for EC-002 (no wire import) in `internal/migrate/testdata/ec002_no_import/`
- [ ] T099 [P] [Polish] Create test for EC-003 (syntax error) in `internal/migrate/testdata/ec003_syntax_error/`
- [X] T100 [P] [Polish] Create test for EC-004 (multiple NewSet) in `internal/migrate/testdata/ec004_multiple_newset/`
- [X] T101 [P] [Polish] Create test for EC-005 (nested NewSet) in `internal/migrate/testdata/ec005_nested_newset/`
- [ ] T102 [Polish] Create test for EC-006 (overwrite existing file) in `internal/migrate/migrate_test.go`
- [ ] T103 [P] [Polish] Create test for EC-008 (build tags stripped) in `internal/migrate/testdata/ec008_build_tags/`
- [ ] T104 [P] [Polish] Create test for EC-009 (comments stripped) in `internal/migrate/testdata/ec009_comments/`
- [ ] T105 [P] [Polish] Create test for EC-010 (wire.Build warning) in `internal/migrate/testdata/ec010_wire_build/`

### Edge Case Implementation

- [X] T106 [Polish] Implement EC-001: Warn and skip output when no wire patterns found in `internal/migrate/migrate.go`
- [X] T107 [Polish] Implement EC-002: Warn and skip file when no wire import found in `internal/migrate/parser.go`
- [ ] T108 [Polish] Implement EC-003: Return ParseError with syntax error details in `internal/migrate/migrate.go`
- [X] T109 [Polish] Implement EC-004: Handle multiple NewSet in single file in `internal/migrate/parser.go`
- [X] T110 [Polish] Implement EC-005: Handle nested NewSet (Set containing Set reference) in `internal/migrate/parser.go`
- [ ] T111 [Polish] Implement EC-006: Overwrite existing output file in `internal/migrate/writer.go`
- [ ] T112 [Polish] Implement EC-008: Strip build tags from output in `internal/migrate/writer.go`
- [ ] T113 [Polish] Implement EC-009: Strip comments from wire calls in `internal/migrate/writer.go`
- [ ] T114 [Polish] Implement EC-010: Detect wire.Build and emit warning in `internal/migrate/parser.go`

### Logging Integration

- [ ] T115 [Polish] Implement structured logging for INFO messages (migration start/complete) in `internal/migrate/migrate.go`
- [ ] T116 [Polish] Implement structured logging for WARN messages (no patterns, unsupported) in `internal/migrate/migrate.go`
- [ ] T117 [Polish] Implement structured logging for ERROR messages (syntax, type, merge errors) in `internal/migrate/migrate.go`

### Code Quality

- [X] T118 [Polish] Run `go tool tools lint ./...` and fix any issues
- [X] T119 [Polish] Run `go test -v ./internal/migrate/...` and ensure all tests pass
- [X] T120 [Polish] Run `go fmt ./...` to format code
- [X] T121 [Polish] Update CLAUDE.md with `kessoku migrate` command documentation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup (Phase 1) - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational (Phase 2)
- **User Story 2 (Phase 4)**: Depends on Foundational (Phase 2), can run parallel to US1
- **User Story 3 (Phase 5)**: Depends on Foundational (Phase 2), can run parallel to US1/US2
- **User Story 4 (Phase 6)**: Depends on Foundational (Phase 2), can run parallel to US1/US2/US3
- **User Story 5 (Phase 7)**: Depends on US1 (needs basic migration working)
- **Polish (Phase 8)**: Can start after US1 is complete

### User Story Dependencies

- **User Story 1 (P1)**: Independent after Foundational
- **User Story 2 (P2)**: Independent after Foundational, adds to parser/transformer
- **User Story 3 (P2)**: Independent after Foundational, adds to parser/transformer
- **User Story 4 (P3)**: Independent after Foundational, adds to parser/transformer
- **User Story 5 (P3)**: Requires US1 basic migration to work (builds on merge logic)

### Within Each User Story

1. Create golden file tests (can be parallel for different test directories)
2. Implement parser detection (depends on tests existing)
3. Implement transformer conversion (depends on parser)
4. Update orchestrator if needed (depends on transformer)
5. Verify golden file tests pass

### Parallel Opportunities

**User Story 1 Tests (T019-T026)**: Can run in parallel (different test directories)
```
T019+T020 (basic/), T021+T022 (bind/), T023+T024 (value/), T025+T026 (import_replace/)
```

**User Story 2-4 Tests**: Can run in parallel within each story (different directories)

**Edge Case Tests (T097-T101, T103-T105)**: Can run in parallel (different test directories)
```
T097 (ec001/), T098 (ec002/), T099 (ec003/), T100 (ec004/), T101 (ec005/), T103 (ec008/), T104 (ec009/), T105 (ec010/)
```
Note: T102 (EC-006) edits `migrate_test.go` - run sequentially with T085-T088.

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T006)
2. Complete Phase 2: Foundational (T007-T017)
3. Complete Phase 3: User Story 1 (T018-T052)
4. **STOP and VALIDATE**: Run `kessoku migrate` on basic wire file
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational (Phases 1-2) - Foundation ready
2. Add User Story 1 (Phase 3) - Basic migration MVP
3. Add User Story 2 (Phase 4) - InterfaceValue support
4. Add User Story 3 (Phase 5) - Struct support
5. Add User Story 4 (Phase 6) - FieldsOf support
6. Add User Story 5 (Phase 7) - Multi-file support
7. Complete Polish (Phase 8) - Production ready

### Parallel Team Strategy

With multiple developers after Foundational is complete:
- Developer A: User Story 1 (basic patterns)
- Developer B: User Story 3 (Struct - more complex transformation)
- Developer C: User Story 2 + 4 (InterfaceValue + FieldsOf - simpler patterns)

User Story 5 (multi-file) should be done after US1 is stable.

---

## Notes

- `[P]` tasks = different files/directories, no dependencies
- `[Story]` label: US1-US5 for user stories, Setup/Found/Polish for infrastructure
- Tests use golden file approach per research.md
- Error handling follows Continue on Warning, Stop on Error per research.md section 7
- All acceptance scenarios from spec.md must pass before story is complete
- Commit after each task or logical group
