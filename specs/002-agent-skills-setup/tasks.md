# Tasks: Agent Skills Setup Subcommand

**Input**: Design documents from `/specs/002-agent-skills-setup/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Unit tests ARE included as they are part of the implementation plan (TDD approach per CLAUDE.md guidelines).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Project Type**: Single Go module with CLI tool
- **New Package**: `internal/llmsetup/` for the llm-setup subcommand implementation
- **Skills Files**: `internal/llmsetup/skills/` for embedded content

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] [T001] Create directory structure `internal/llmsetup/` and `internal/llmsetup/skills/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

- [x] [T002] [P] Create Agent interface and registry in `internal/llmsetup/agent.go`
- [x] [T003] [P] Create embedded Skills content file at `internal/llmsetup/skills/kessoku.md`
- [x] [T004] [P] Create Claude Code agent implementation in `internal/llmsetup/claudecode.go`
- [x] [T005] [P] Create LLMSetupCmd structure and ClaudeCodeCmd in `internal/llmsetup/llmsetup.go`
- [x] [T006] Integrate LLMSetupCmd into CLI struct in `internal/config/config.go`

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Install Claude Code Skills (Project-Level) (Priority: P1)

**Goal**: Users can install kessoku Skills for Claude Code at project-level with `kessoku llm-setup claude-code`

**Independent Test**: Run `kessoku llm-setup claude-code` and verify `kessoku.md` appears in `./.claude/skills/` with correct content

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] [T007] [P] [US1] Unit test for project-level path resolution in `internal/llmsetup/path_test.go` (TestResolvePath_ProjectLevel)
- [x] [T008] [P] [US1] Unit test for directory creation in `internal/llmsetup/install_test.go` (TestInstallFile_CreatesDirectory)
- [x] [T009] [P] [US1] Unit test for atomic overwrite in `internal/llmsetup/install_test.go` (TestInstallFile_AtomicOverwrite)
- [x] [T010] [P] [US1] Unit test for idempotent install in `internal/llmsetup/install_test.go` (TestInstallFile_Idempotent)
- [x] [T011] [P] [US1] Unit test for partial file cleanup in `internal/llmsetup/install_test.go` (TestInstallFile_CleansUpOnError)
- [x] [T012] [P] [US1] Unit test for success output in `internal/llmsetup/llmsetup_test.go` (TestClaudeCodeCmd_SuccessOutput)
- [x] [T013] [P] [US1] Unit test for error output in `internal/llmsetup/llmsetup_test.go` (TestClaudeCodeCmd_ErrorOutput)

### Implementation for User Story 1

- [x] [T014] [P] [US1] Implement ResolvePath function for project-level in `internal/llmsetup/path.go`
- [x] [T015] [P] [US1] Implement ValidatePath function in `internal/llmsetup/path.go`
- [x] [T016] [US1] Implement InstallFile function with atomic write in `internal/llmsetup/install.go`
- [x] [T017] [US1] Implement Install function (orchestrates path resolution and file install) in `internal/llmsetup/install.go`
- [x] [T018] [US1] Implement ClaudeCodeCmd.Run() method in `internal/llmsetup/llmsetup.go`
- [x] [T019] [US1] Add success message output ("Skills installed to: <path>") in `internal/llmsetup/llmsetup.go`
- [x] [T020] [US1] Add error handling and stderr output in `internal/llmsetup/llmsetup.go`

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Install Claude Code Skills (User-Level) (Priority: P2)

**Goal**: Users can install kessoku Skills at user-level with `kessoku llm-setup claude-code --user`

**Independent Test**: Run `kessoku llm-setup claude-code --user` and verify file is installed to `~/.claude/skills/kessoku.md`

### Tests for User Story 2

- [x] [T021] [P] [US2] Unit test for user-level path resolution (Linux) in `internal/llmsetup/path_test.go` (TestResolvePath_UserLevel_Linux)
- [x] [T022] [P] [US2] Unit test for user-level path resolution (Darwin) in `internal/llmsetup/path_test.go` (TestResolvePath_UserLevel_Darwin)
- [x] [T023] [P] [US2] Unit test for user-level path resolution (Windows) in `internal/llmsetup/path_test.go` (TestResolvePath_UserLevel_Windows)
- [x] [T024] [P] [US2] Unit test for HOME not set error in `internal/llmsetup/path_test.go` (TestResolvePath_HomeNotSet)
- [x] [T025] [P] [US2] Unit test for unsupported OS error in `internal/llmsetup/path_test.go` (TestResolvePath_UnsupportedOS)

### Implementation for User Story 2

- [x] [T026] [US2] Add `--user` flag to ClaudeCodeCmd struct in `internal/llmsetup/llmsetup.go`
- [x] [T027] [US2] Implement getHomeDir helper (explicit env var check) in `internal/llmsetup/path.go`
- [x] [T028] [US2] Extend ResolvePath for user-level paths in `internal/llmsetup/path.go`
- [x] [T029] [US2] Add OS validation (linux, darwin, windows) in `internal/llmsetup/path.go`

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - List Available Agents (Priority: P2)

**Goal**: Users can see available agents with `kessoku llm-setup` or `kessoku llm-setup --help`

**Independent Test**: Run `kessoku llm-setup` and verify it prints usage with available agents and exits with code 0

### Tests for User Story 3

- [x] [T030] [P] [US3] Unit test for no subcommand exit 0 in `internal/llmsetup/llmsetup_test.go` (TestLLMSetupCmd_NoSubcommand)
- [x] [T031] [P] [US3] Unit test for unknown agent exit 1 in `internal/llmsetup/llmsetup_test.go` (TestLLMSetupCmd_UnknownAgent)

### Implementation for User Story 3

- [x] [T032] [US3] Implement LLMSetupCmd.Run() to print usage and exit 0 in `internal/llmsetup/llmsetup.go`
- [x] [T033] [US3] Configure Kong help text for agent subcommands in `internal/llmsetup/llmsetup.go`

**Checkpoint**: Users can now discover available agents

---

## Phase 6: User Story 4 - Custom Installation Path (Priority: P3)

**Goal**: Users can install Skills to a custom location with `kessoku llm-setup claude-code --path <dir>`

**Independent Test**: Run `kessoku llm-setup claude-code --path /tmp/test` and verify file is installed to `/tmp/test/kessoku.md`

### Tests for User Story 4

- [x] [T034] [P] [US4] Unit test for --path overrides --user in `internal/llmsetup/path_test.go` (TestResolvePath_PathOverridesUser)
- [x] [T035] [P] [US4] Unit test for tilde expansion in `internal/llmsetup/path_test.go` (TestResolvePath_TildeExpansion)
- [x] [T036] [P] [US4] Unit test for relative path resolution in `internal/llmsetup/path_test.go` (TestResolvePath_RelativePath)
- [x] [T037] [P] [US4] Unit test for path is file error in `internal/llmsetup/path_test.go` (TestValidatePath_PathIsFile)
- [x] [T038] [P] [US4] Unit test for permission denied in `internal/llmsetup/install_test.go` (TestInstallFile_PermissionDenied)

### Implementation for User Story 4

- [x] [T039] [US4] Add `--path` flag to ClaudeCodeCmd struct in `internal/llmsetup/llmsetup.go`
- [x] [T040] [US4] Implement tilde expansion in ResolvePath in `internal/llmsetup/path.go`
- [x] [T041] [US4] Implement relative path resolution in ResolvePath in `internal/llmsetup/path.go`
- [x] [T042] [US4] Implement --path precedence over --user in `internal/llmsetup/path.go`

**Checkpoint**: All path options (project-level, user-level, custom) now work

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and cleanup

- [x] [T043] Run quickstart.md validation scenarios in `specs/002-agent-skills-setup/quickstart.md`
- [x] [T044] [P] Run linter on `internal/llmsetup/` package
- [x] [T045] [P] Run all tests in `internal/llmsetup/` package
- [x] [T046] Verify Windows-specific rename handling in `internal/llmsetup/install.go`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - User stories can proceed sequentially in priority order (P1 -> P2 -> P3)
  - US3 (List Agents) can run in parallel with US1/US2 as it doesn't affect installation logic
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Extends path.go from US1 - shares ResolvePath function
- **User Story 3 (P2)**: Independent of installation logic - can run parallel with US1/US2
- **User Story 4 (P3)**: Extends path.go from US1/US2 - shares ResolvePath function

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Path resolution before installation logic
- Installation logic before command handling
- Core implementation before output formatting
- Story complete before moving to next priority

### Parallel Opportunities

- All Foundational tasks marked [P] can run in parallel ([T002]-[T005])
- All tests within a user story marked [P] can run in parallel
- Different user stories MAY overlap if dependencies are respected:
  - US1 tests can start while Foundational completes
  - US3 can run in parallel with US1/US2 (different functionality)

---

## Parallel Example: User Story 1 Tests

```bash
# Launch all tests for User Story 1 together:
Task: "Unit test for project-level path resolution in internal/llmsetup/path_test.go"
Task: "Unit test for directory creation in internal/llmsetup/install_test.go"
Task: "Unit test for atomic overwrite in internal/llmsetup/install_test.go"
Task: "Unit test for idempotent install in internal/llmsetup/install_test.go"
Task: "Unit test for partial file cleanup in internal/llmsetup/install_test.go"
Task: "Unit test for success output in internal/llmsetup/llmsetup_test.go"
Task: "Unit test for error output in internal/llmsetup/llmsetup_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Project-Level Install)
4. **STOP and VALIDATE**: Test `kessoku llm-setup claude-code` independently
5. Deploy/demo if ready - users can install Skills at project level

### Incremental Delivery

1. Complete Setup + Foundational -> Foundation ready
2. Add User Story 1 -> Test independently -> Users can install at project-level (MVP!)
3. Add User Story 2 -> Test independently -> Users can install at user-level
4. Add User Story 3 -> Test independently -> Users can discover agents
5. Add User Story 4 -> Test independently -> Users can use custom paths
6. Each story adds value without breaking previous stories

### TDD Approach (Per CLAUDE.md)

For each user story:
1. Write tests first (all test tasks marked [P] for the story)
2. Verify tests fail (expected - implementation not done)
3. Implement functionality (implementation tasks)
4. Verify tests pass
5. Refactor if needed
6. Move to next story

---

## Summary

| Metric | Count |
|--------|-------|
| **Total Tasks** | 46 |
| **Setup Tasks** | 1 |
| **Foundational Tasks** | 5 |
| **User Story 1 Tasks** | 14 (7 tests + 7 impl) |
| **User Story 2 Tasks** | 9 (5 tests + 4 impl) |
| **User Story 3 Tasks** | 4 (2 tests + 2 impl) |
| **User Story 4 Tasks** | 9 (5 tests + 4 impl) |
| **Polish Tasks** | 4 |
| **Parallelizable Tasks** | 33 |

### MVP Scope

**Suggested MVP**: User Story 1 only (Phases 1-3)
- 20 tasks total (Setup: 1, Foundational: 5, US1: 14)
- Delivers core functionality: `kessoku llm-setup claude-code`
- Users can immediately benefit from Claude Code Skills integration

### Format Validation

All tasks follow the required format:
- Checkbox: `- [ ]`
- Task ID: `[T001]`-`[T046]`
- [P] marker for parallelizable tasks
- [Story] label for user story phases (US1, US2, US3, US4)
- Clear description with file path
