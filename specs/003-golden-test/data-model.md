# Data Model: Golden Test for Code Generation

**Date**: 2026-01-10
**Feature**: Golden Test for Code Generation

## Overview

This feature introduces a file-based testing infrastructure. The "data model" consists of filesystem structures and Go test constructs rather than database entities.

## Entities

### 1. Test Case Directory

**Description**: A directory containing input files and expected output for a single golden test case.

**Location**: `internal/kessoku/testdata/<test-case-name>/`

**Structure**:
```
<test-case-name>/
├── kessoku.go       # Required: Contains kessoku.Inject call
├── *.go             # Optional: Provider definition files
└── expected.go      # Required: Golden file (expected generated output)
```

**Validation Rules**:
- Directory name becomes the test case name (used in `t.Run()`)
- Must contain at least `kessoku.go` with valid `kessoku.Inject` call
- Must contain `expected.go` (golden file)
- All `.go` files except `expected.go` are treated as input

**States**: N/A (static filesystem structure)

### 2. Golden File

**Description**: The pre-approved expected output of code generation.

**Filename**: `expected.go`

**Content**: Generated Go source code that would normally be written to `*_band.go`

**Validation Rules**:
- Must be valid Go syntax
- Must compile with input files in same directory
- Package name must match input files

**States**:
- Current: Matches current code generator output
- Outdated: Differs from code generator output (test fails)
- Updated: Regenerated via `-update` flag

### 3. Test Runner

**Description**: The `TestGoldenGeneration` function in `golden_test.go`

**Responsibilities**:
1. Discover test case directories
2. For each test case:
   - Copy input files to `t.TempDir()`
   - Run code generator
   - Compare output with `expected.go`
   - Report differences or pass

**Configuration**:
- `-update` flag: When true, overwrites `expected.go` with actual output
- Parallel execution: Enabled when `-update` is false

## Relationships

```
testdata/
    │
    └──> TestCase (1:N)
              │
              ├──> InputFiles (1:N) ──> kessoku.go + provider files
              │
              └──> GoldenFile (1:1) ──> expected.go
```

## File Discovery Algorithm

```go
// Pseudocode for test case discovery
entries := os.ReadDir("testdata")
for entry in entries:
    if entry.IsDir():
        testCase := loadTestCase(entry.Name())
        if hasKessokuGo(testCase) && hasExpectedGo(testCase):
            runTest(testCase)
```

## Update Mode Flow

```
Normal Mode:
  Input Files → Generate → TempDir/output.go → Compare with expected.go → Pass/Fail

Update Mode:
  Input Files → Generate → Overwrite expected.go → Always Pass
```
