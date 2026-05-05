# Repository Guidelines — kessoku

Kessoku is a compile-time dependency injection library for Go that speeds up
application startup through parallel dependency injection. Unlike `google/wire`
which initialises services sequentially, Kessoku automatically executes
independent providers in parallel. It generates optimised code at compile time
with zero runtime overhead.

> Agent configuration is managed via [apm](https://github.com/microsoft/apm).
> Common conventions live in `mazrean/apm-plackage/common`; per-stack rules come
> from `mazrean/apm-plackage/{go,goreleaser}`. Run `apm install` locally.

## Project Structure

- Go 1.24 workspace with main module `github.com/mazrean/kessoku` and tools module `./tools` (`go.work` tracks both).
- Public API lives at the root (`annotation.go`); CLI entrypoint is `cmd/kessoku/`.
- Codegen engine sits under `internal/kessoku/` (parser, graph, generator); CLI wiring lives in `internal/config/`.
- Utilities are in `internal/pkg/`; examples live in `examples/`; specs and checklists under `specs/`; built binaries land in `bin/`.

## Common Commands

```bash
# Build
go build -o bin/kessoku ./cmd/kessoku

# Test
go test -v ./...                           # run all tests
go test -v -run TestName ./...             # run a specific test
go test -v ./internal/kessoku/...          # run tests in a specific package

# Golden tests (code generation validation)
go test -v -run TestGoldenGeneration ./internal/kessoku/...           # run golden tests
go test -v -run TestGoldenGeneration ./internal/kessoku/... -update   # update golden files

# Lint (mandatory before commit)
go tool lint ./...
go tool lint -fix ./...                    # auto-fix where possible

# Code generation
go generate ./...                          # generate DI code via go:generate
go tool kessoku [files...]                 # direct codegen for specific files

# Wire migration
go tool kessoku migrate [patterns...] -o kessoku.go

# API compatibility check
go tool apicompat github.com/mazrean/kessoku@latest github.com/mazrean/kessoku

# Release
go tool goreleaser release --snapshot --clean
```

## Architecture

### Module Structure

Go 1.24 workspace with two modules:
- **Main module** (`github.com/mazrean/kessoku`): public API and codegen engine
- **Tools module** (`./tools`): custom linter combining govet + staticcheck + stylecheck analyzers

### Code Generation Pipeline

The codegen engine in `internal/kessoku/` follows this flow:

```
parser.go → graph.go → generator.go
   ↓            ↓            ↓
Parse AST   Build DAG    Emit code
& extract   & detect     with parallel
providers   cycles       execution
```

1. **Parser**: finds `kessoku.Inject` calls, extracts provider types and dependencies.
2. **Graph**: constructs dependency DAG, detects cycles, computes parallel execution pools.
3. **Generator**: emits `*_band.go` files with optimised injector functions.

### Wire Migration Tool

`kessoku migrate` converts google/wire configuration files to kessoku format. It
uses the `wireinject` build tag to load wire configuration files (same as wire).

```bash
go tool kessoku migrate              # migrate current directory
go tool kessoku migrate ./pkg/di     # migrate a specific package
go tool kessoku migrate ./... -o providers.go
```

Supported wire patterns:
- `wire.NewSet(providers...)` → `kessoku.Set(providers...)`
- `wire.Bind(new(Interface), new(Impl))` → `kessoku.Bind[Interface]()`
- `wire.Value(v)` → `kessoku.Value(v)`
- `wire.InterfaceValue(new(I), v)` → `kessoku.Bind[I](kessoku.Value(v))`
- `wire.Struct(new(T), "Field1", "Field2")` → `kessoku.Provide(func(f1, f2) *T { ... })`
- `wire.FieldsOf(new(T), "F1", "F2")` → `kessoku.Provide(func(t *T) (T1, T2) { ... })`
- Set references (e.g. `wire.NewSet(OtherSet, ...)`) are preserved.

Migration tool location: `internal/migrate/`.

### Key Code Locations

- `annotation.go`: public API (`Inject`, `Provide`, `Async`, `Bind`, `Value`, `Set`, `Struct`)
- `internal/kessoku/provider.go`: core data structures (`ProviderSpec`, `Injector`, `InjectorStmt`)
- `internal/kessoku/golden_test.go`: golden tests for code generation validation
- `internal/kessoku/testdata/`: test cases for golden tests (input + expected.go)
- `internal/config/`: CLI configuration and orchestration
- `internal/migrate/`: Wire to Kessoku migration tool

## Coding Style & Naming

- Let `go fmt` dictate whitespace (tabs) and imports; no manual styling wars.
- Keep packages small and focused; avoid clever abstractions — simple functions beat generic magic.
- Exported identifiers follow Go's MixedCaps and include clear doc comments when part of the public API.
- Group injector declarations with their providers; leave `//go:generate go tool kessoku $GOFILE` beside injector files.
- Do not hand-edit generated files; rerun codegen instead.

## Testing

- Tests live next to code in `*_test.go`; prefer table-driven cases.
- Run `go test -v ./...` before pushing; keep tests deterministic and fast.
- Add tests with every behaviour change; verify new providers/injectors through generated output where relevant.

## Mandatory Before Commit

- `go fmt ./...` — format
- `go test -v ./...` — all tests must pass
- `go tool lint ./...` — treat failures as blockers
- `go tool tools apicompat <base> <target>` — when altering public API

## Commits & PRs

- Use Conventional Commits (`fix:`, `feat:`, `chore:`, `deps:`, `ci:`); one logical change per commit.
- Documentation updates should be part of the same commit as related code changes.
- PRs should include a concise summary of the change, linked issues, and call out
  any codegen steps or API-impactful changes (attach `apicompat` output when altering public API).

## Active Technologies

- Go 1.24+ + github.com/alecthomas/kong (CLI), golang.org/x/tools (AST parsing, type checking)
- File-based input/output (no persistent storage)

## Tooling

- Code intelligence: `serena` (multi-language LSP) + `gopls` (Go LSP) MCPs declared in `apm.yml`.
- Spec-driven development uses `mazrean/agent-skills/skills/writing-*`. `cc-sdd` /
  `github/spec-kit` are deprecated org-wide and removed from `mise.toml`.
