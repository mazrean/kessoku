# Tasks: Expand Agent Support

**Input**: Design documents from `/specs/001-expand-agent-support/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: Included (existing test infrastructure to be extended)

**Organization**: Due to the nature of this feature (adding 6 parallel agent implementations), tasks are organized by implementation phase rather than strictly by user story. Each agent file enables all user stories for that agent.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task contributes to (US1-US5)
- Include exact file paths in descriptions

## User Story Reference

| Story | Priority | Description |
|-------|----------|-------------|
| US1 | P1 | Install Skills for Any Supported Agent (project-level) |
| US2 | P2 | Install Skills at User Level for New Agents |
| US3 | P3 | Discover Available Agents (CLI help) |
| US4 | P1 | Default Installation Behavior |
| US5 | P2 | Shared Directory Behavior (Amp/Goose) |

---

## Phase 1: Setup

**Purpose**: No setup required - existing Go project with established patterns

*All dependencies already in place. Proceeding to implementation.*

---

## Phase 2: Foundational - Agent Implementations

**Purpose**: Create all 6 new agent struct files following existing pattern

**Story Coverage**: Each agent file enables US1, US2, US3, US4 for that agent. T005 and T006 additionally enable US5.

- [x] T001 [P] Create GeminiCLIAgent in internal/llmsetup/gemini.go with project path `.gemini/skills` and user path `.gemini/skills`
- [x] T002 [P] Create OpenCodeAgent in internal/llmsetup/opencode.go with project path `.opencode/skill` and user path `.config/opencode/skill`
- [x] T003 [P] Create CodexAgent in internal/llmsetup/codex.go with project path `.codex/skills` and user path `.codex/skills`
- [x] T004 [P] Create AmpAgent in internal/llmsetup/amp.go with project path `.agents/skills` and user path `.config/agents/skills`
- [x] T005 [P] Create GooseAgent in internal/llmsetup/goose.go with project path `.agents/skills` and user path `.config/goose/skills`
- [x] T006 [P] Create FactoryAgent in internal/llmsetup/factory.go with project path `.factory/skills` and user path `.factory/skills`

**Checkpoint**: All 6 agent structs exist. Ready for registration.

---

## Phase 3: US1 + US4 - Core Installation (Priority: P1)

**Goal**: Enable project-level skill installation for all new agents with correct default behavior

**Independent Test**: Run `go tool kessoku <agent> --project` for each new agent and verify skills are installed to correct directory

### Implementation

- [x] T007 [US1] [US4] Register 6 new agents in agents slice in internal/llmsetup/agent.go (alphabetical: Amp, Codex, Factory, GeminiCLI, Goose, OpenCode)
- [x] T008 [P] [US1] [US4] Add 6 type aliases (AmpCmd, CodexCmd, FactoryCmd, GeminiCLICmd, GooseCmd, OpenCodeCmd) in internal/llmsetup/llmsetup.go
- [x] T009 [US1] [US4] Add 6 new fields to LLMSetupCmd struct in internal/llmsetup/llmsetup.go with kong tags

**Checkpoint**: `go tool kessoku <agent-name>` works for all 6 new agents. Project-level installation enabled.

---

## Phase 4: US2 + US5 - User-Level & Shared Directory (Priority: P2)

**Goal**: Verify user-level installation and shared directory behavior work correctly

**Independent Test**: Run `go tool kessoku <agent> --user` for each new agent. Run both `amp --project` and `goose --project` in same directory.

**Note**: User-level paths are already configured in agent structs (Phase 2). Shared directory behavior is inherent in Amp/Goose configuration. This phase is primarily verification.

### Verification Tasks

- [x] T010 [US2] Verify user-level installation works by testing `go tool kessoku gemini-cli --user` writes to correct path
- [x] T011 [US5] Verify shared directory behavior: `amp --project` followed by `goose --project` produces identical files in `.agents/skills/kessoku-di/`

**Checkpoint**: User-level installation and shared directory idempotence verified.

---

## Phase 5: US3 - Discovery (Priority: P3)

**Goal**: All 9 agents visible and documented in CLI help

**Independent Test**: Run `go tool kessoku --help` and verify all agents appear

### Verification Tasks

- [x] T012 [US3] Verify `go tool kessoku --help` displays all 9 agents with descriptions

**Checkpoint**: CLI help shows all agents. Discovery complete.

---

## Phase 6: Testing

**Purpose**: Extend existing test suite to cover new agents

### Agent Struct Tests

- [x] T013 [P] Add TestAgents table entries for GeminiCLIAgent in internal/llmsetup/llmsetup_test.go
- [x] T014 [P] Add TestAgents table entries for OpenCodeAgent in internal/llmsetup/llmsetup_test.go
- [x] T015 [P] Add TestAgents table entries for CodexAgent in internal/llmsetup/llmsetup_test.go
- [x] T016 [P] Add TestAgents table entries for AmpAgent in internal/llmsetup/llmsetup_test.go
- [x] T017 [P] Add TestAgents table entries for GooseAgent in internal/llmsetup/llmsetup_test.go
- [x] T018 [P] Add TestAgents table entries for FactoryAgent in internal/llmsetup/llmsetup_test.go

### CLI Command Tests

- [x] T019 [P] Add TestGeminiCLICmd using testAgentCmd helper in internal/llmsetup/llmsetup_test.go
- [x] T020 [P] Add TestOpenCodeCmd using testAgentCmd helper in internal/llmsetup/llmsetup_test.go
- [x] T021 [P] Add TestCodexCmd using testAgentCmd helper in internal/llmsetup/llmsetup_test.go
- [x] T022 [P] Add TestAmpCmd using testAgentCmd helper in internal/llmsetup/llmsetup_test.go
- [x] T023 [P] Add TestGooseCmd using testAgentCmd helper in internal/llmsetup/llmsetup_test.go
- [x] T024 [P] Add TestFactoryCmd using testAgentCmd helper in internal/llmsetup/llmsetup_test.go

### Registry Tests

- [x] T025 Update TestGetAgent to include all 6 new agent names in internal/llmsetup/llmsetup_test.go
- [x] T026 Update TestListAgents expected count to 9 in internal/llmsetup/llmsetup_test.go
- [x] T027 Update TestLLMSetupCmd subcommands check to include 6 new commands in internal/llmsetup/llmsetup_test.go

**Checkpoint**: All tests written. Ready for validation.

---

## Phase 7: Polish & Validation

**Purpose**: Final validation and cleanup

- [x] T028 Run `go test -v ./internal/llmsetup/...` and fix any failures
- [x] T029 Run `go tool lint ./...` and fix any issues
- [x] T030 Run manual verification per quickstart.md: test all 6 agents with `--project` flag
- [x] T031 Run manual verification per quickstart.md: test all 6 agents with `--user` flag
- [x] T032 Verify `go tool kessoku --help` shows all 9 agents with correct descriptions

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: N/A - skipped
- **Phase 2 (Foundational)**: No dependencies - can start immediately
- **Phase 3 (US1+US4)**: Depends on Phase 2 completion
- **Phase 4 (US2+US5)**: Depends on Phase 3 completion
- **Phase 5 (US3)**: Depends on Phase 3 completion
- **Phase 6 (Testing)**: Depends on Phase 3 completion
- **Phase 7 (Polish)**: Depends on Phase 6 completion

### User Story Dependencies

- **US1 (P1)**: Requires Phase 2 + Phase 3
- **US4 (P1)**: Requires Phase 2 + Phase 3 (same implementation as US1)
- **US2 (P2)**: Requires Phase 2 + Phase 3 (paths already in agent structs)
- **US5 (P2)**: Requires Phase 2 + Phase 3 (shared path already configured)
- **US3 (P3)**: Requires Phase 3 (CLI commands provide discovery)

### Parallel Opportunities

**Phase 2** - All 6 agent files can be created in parallel:
```
T001, T002, T003, T004, T005, T006 → all parallel
```

**Phase 6** - Most test tasks can run in parallel:
```
Agent tests: T013, T014, T015, T016, T017, T018 → all parallel
CLI tests: T019, T020, T021, T022, T023, T024 → all parallel
```

---

## Parallel Example: Phase 2

```bash
# Launch all agent file creation tasks together:
Task: "Create GeminiCLIAgent in internal/llmsetup/gemini.go"
Task: "Create OpenCodeAgent in internal/llmsetup/opencode.go"
Task: "Create CodexAgent in internal/llmsetup/codex.go"
Task: "Create AmpAgent in internal/llmsetup/amp.go"
Task: "Create GooseAgent in internal/llmsetup/goose.go"
Task: "Create FactoryAgent in internal/llmsetup/factory.go"
```

---

## Implementation Strategy

### MVP First (US1 + US4 Only)

1. Complete Phase 2: Create all 6 agent files
2. Complete Phase 3: Register and add CLI commands
3. **STOP and VALIDATE**: Test project-level installation for all agents
4. Deploy/demo if ready - core functionality complete

### Incremental Delivery

1. Phase 2 → Agent structs ready
2. Phase 3 → Project-level installation works (US1 + US4 complete)
3. Phase 4 → User-level verified (US2 + US5 complete)
4. Phase 5 → Discovery verified (US3 complete)
5. Phase 6 → Tests added
6. Phase 7 → Final validation

### Single Developer Strategy

Execute phases sequentially:
1. T001-T006 (parallel agent files)
2. T007-T009 (integration)
3. T010-T012 (verification)
4. T013-T027 (tests - batch by similarity)
5. T028-T032 (polish)

---

## Notes

- All 6 new agents follow the identical pattern established by existing agents
- Agent structs share `claudeCodeSkillsFS` - no new embed.FS needed
- Amp and Goose intentionally share `.agents/skills/` at project level (per research.md)
- OpenCode uses singular `skill` (not `skills`) per official documentation
- Task IDs are sequential but [P] tasks within a phase can run in parallel
- Each agent enables all 5 user stories once integrated
