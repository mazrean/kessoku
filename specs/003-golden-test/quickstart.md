# Quickstart: Golden Test for Code Generation

**Date**: 2026-01-10
**Feature**: Golden Test for Code Generation

## Running Golden Tests

### Run All Tests

```bash
go test -v ./internal/kessoku/...
```

### Run Only Golden Tests

```bash
go test -v -run TestGoldenGeneration ./internal/kessoku/...
```

### Update Golden Files

When you intentionally change code generation output:

```bash
go test -v -run TestGoldenGeneration ./internal/kessoku/... -update
```

Then review the changes:

```bash
git diff internal/kessoku/testdata/
```

## Adding a New Test Case

### 1. Create Test Case Directory

```bash
mkdir internal/kessoku/testdata/my_new_case
```

### 2. Add Input Files

Create `kessoku.go` with your test scenario:

```go
//go:build ignore

package main

import "github.com/mazrean/kessoku"

//go:generate go tool kessoku

func main() {
    kessoku.Inject(NewApp)
}
```

Add provider files as needed (e.g., `service.go`, `config.go`).

### 3. Generate Initial Golden File

Run with `-update` flag to create `expected.go`:

```bash
go test -v -run TestGoldenGeneration/my_new_case ./internal/kessoku/... -update
```

### 4. Verify the Golden File

Review the generated `expected.go`:

```bash
cat internal/kessoku/testdata/my_new_case/expected.go
```

## Copying from Examples

For complex scenarios, copy an existing example:

```bash
# Copy example directory
cp -r examples/basic internal/kessoku/testdata/basic

# Remove binary and rename generated file
rm -f internal/kessoku/testdata/basic/basic
mv internal/kessoku/testdata/basic/kessoku_band.go internal/kessoku/testdata/basic/expected.go
```

## Test Case Structure

Each test case directory should contain:

```
internal/kessoku/testdata/<name>/
├── kessoku.go       # Required: kessoku.Inject call
├── *.go             # Optional: provider definitions
└── expected.go      # Required: expected generated output
```

## Troubleshooting

### Test Fails with "missing golden file"

The `expected.go` file doesn't exist. Run with `-update` to create it:

```bash
go test -run TestGoldenGeneration/<case-name> ./internal/kessoku/... -update
```

### Test Fails with Diff Output

The generated code differs from `expected.go`. Either:

1. **Bug in your change**: Fix the code generation logic
2. **Intentional change**: Run with `-update` and review the diff

### Parse Error in Test Case

Check your input files for Go syntax errors:

```bash
go build ./internal/kessoku/testdata/<case-name>/
```
