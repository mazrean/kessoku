# Repository Guidelines

## Project Structure & Module Organization
- Go 1.24 workspace with main module `github.com/mazrean/kessoku` and tools module `./tools` (`go.work` tracks both).
- Public API lives at the root (`annotation.go`); CLI entrypoint is `cmd/kessoku/`.
- Codegen engine sits under `internal/kessoku/` (parser, graph, generator); CLI wiring lives in `internal/config/`.
- Utilities are in `internal/pkg/`; examples live in `examples/`; specs and checklists under `specs/`; built binaries land in `bin/`.

## Build, Test, and Development Commands
- `go build -o bin/kessoku ./cmd/kessoku` — build the CLI.
- `go generate ./...` — run injection codegen (driven by `//go:generate go tool kessoku $GOFILE` comments).
- `go tool kessoku [files...]` — direct codegen for specific Go files.
- `go test -v ./...` — run the test suite.
- `go fmt ./...` — format; mandatory before committing.
- `go tool tools lint ./...` — run the multi-checker (vet + staticcheck + extras); treat failures as blockers.
- `go tool tools apicompat <base> <target>` — check API compatibility (e.g., `...@latest` vs current).

## Coding Style & Naming Conventions
- Let `go fmt` dictate whitespace (tabs) and imports; no manual styling wars.
- Keep packages small and focused; avoid clever abstractions—simple functions beat generic magic.
- Exported identifiers follow Go’s MixedCaps and include clear doc comments when part of the public API.
- Group injector declarations with their providers; leave `//go:generate go tool kessoku $GOFILE` beside injector files.
- Do not hand-edit generated files; rerun codegen instead.

## Testing Guidelines
- Tests live next to code in `*_test.go`; prefer table-driven cases.
- Run `go test -v ./...` before pushing; keep tests deterministic and fast.
- Add tests with every behavior change; verify new providers/injectors through generated output where relevant.

## Commit & Pull Request Guidelines
- Commit style matches history: short, imperative prefixes like `fix: ...`, `deps: ...`, `ci: ...`; one logical change per commit.
- Before opening a PR: run `go fmt ./...`, `go test -v ./...`, and `go tool tools lint ./...`; note results in the description.
- PRs should include a concise summary of the change, linked issues (if any), and call out any codegen steps or API-impactful changes (attach `apicompat` output when altering public API).

## Active Technologies
- Go 1.24+ + github.com/alecthomas/kong (CLI), golang.org/x/tools/go/packages (AST analysis) (001-struct-annotation)
- N/A (code generation tool) (001-struct-annotation)

## Recent Changes
- 001-struct-annotation: Added Go 1.24+ + github.com/alecthomas/kong (CLI), golang.org/x/tools/go/packages (AST analysis)
