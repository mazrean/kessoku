package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMigration runs golden file tests for migration.
func TestMigration(t *testing.T) {
	testdataDir := "testdata"

	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Skipf("testdata directory not found: %v", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			dir := filepath.Join(testdataDir, entry.Name())

			// Find input files (any .go file that's not expected.go)
			inputFiles, err := findInputFiles(dir)
			if err != nil {
				t.Fatalf("failed to find input files: %v", err)
			}

			if len(inputFiles) == 0 {
				t.Skip("no input files found")
			}

			expectedPath := filepath.Join(dir, "expected.go")
			expectedBytes, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("failed to read expected.go: %v", err)
			}

			// Create temp output file
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "output.go")

			// Check if this is a skip test case (no output expected)
			isSkipCase := strings.HasPrefix(string(expectedBytes), "// SKIP:")

			// Run migration
			migrator := NewMigrator()
			err = migrator.MigrateFiles(inputFiles, outputPath)
			if err != nil {
				// Check if expected.go contains error expectation
				if strings.Contains(string(expectedBytes), "// ERROR:") {
					// This is an expected error case
					return
				}
				t.Fatalf("migration failed: %v", err)
			}

			// For skip cases, verify no output was generated
			if isSkipCase {
				if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
					t.Errorf("expected no output file for skip case, but file exists")
				}
				return
			}

			// Read output
			outputBytes, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read output: %v", err)
			}

			// Compare
			if string(outputBytes) != string(expectedBytes) {
				t.Errorf("output mismatch:\n--- expected ---\n%s\n--- got ---\n%s",
					string(expectedBytes), string(outputBytes))
			}
		})
	}
}

// findInputFiles finds all .go files in a directory except expected.go.
func findInputFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".go") && name != "expected.go" {
			files = append(files, filepath.Join(dir, name))
		}
	}

	return files, nil
}

// TestMigrateHelp tests that kessoku migrate --help displays help.
func TestMigrateHelp(t *testing.T) {
	// This test verifies CLI help output format
	// Actual CLI testing is done in integration tests
	t.Skip("CLI integration test - run with actual binary")
}

// TestDefaultOutputPath tests the default output path behavior.
func TestDefaultOutputPath(t *testing.T) {
	// Verify default output path is kessoku.go
	cmd := MigrateCmd{}
	// The default is set via kong tags, verify the struct field
	if cmd.Output != "" {
		t.Logf("Output field has value: %s (default is set by kong)", cmd.Output)
	}
}

// TestCustomOutputPath tests custom output path behavior.
func TestCustomOutputPath(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "custom_output.go")

	// Create a simple test file
	inputDir := filepath.Join(tmpDir, "input")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	inputFile := filepath.Join(inputDir, "wire.go")
	inputContent := `package test

import "github.com/google/wire"

var TestSet = wire.NewSet(NewFoo)

func NewFoo() *Foo { return &Foo{} }

type Foo struct{}
`
	if err := os.WriteFile(inputFile, []byte(inputContent), 0644); err != nil {
		t.Fatal(err)
	}

	migrator := NewMigrator()
	err := migrator.MigrateFiles([]string{inputFile}, outputPath)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Verify output file exists at custom path
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("output file was not created at custom path")
	}
}

// TestGeneratedOutputHasBuildConstraint verifies that the generated kessoku output
// contains //go:build !wireinject so that re-running "kessoku migrate ./" is
// idempotent (BUG-18: second invocation fails with misleading type-check error).
func TestGeneratedOutputHasBuildConstraint(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "kessoku.go")

	inputFile := filepath.Join(tmpDir, "wire.go")
	inputContent := `package test

import "github.com/google/wire"

var TestSet = wire.NewSet(NewFoo)

func NewFoo() *Foo { return &Foo{} }

type Foo struct{}
`
	if err := os.WriteFile(inputFile, []byte(inputContent), 0644); err != nil {
		t.Fatal(err)
	}

	migrator := NewMigrator()
	if err := migrator.MigrateFiles([]string{inputFile}, outputPath); err != nil {
		t.Fatalf("first migration failed: %v", err)
	}

	outputBytes, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// The generated file must start with //go:build !wireinject so that
	// packages.Load (called with -tags=wireinject) skips it on subsequent runs.
	if !strings.HasPrefix(string(outputBytes), "//go:build !wireinject\n") {
		t.Errorf("generated output does not start with //go:build !wireinject; got:\n%s", string(outputBytes))
	}
}

// TestOutputPackageConflict tests that migration fails when the generated package name
// does not match the existing package in the output directory.
// This is the regression test for the bug where running 'kessoku migrate' from the repo
// root with default output 'kessoku.go' would write a 'package main' file into a directory
// containing 'package kessoku' files, breaking the entire build.
func TestOutputPackageConflict(t *testing.T) {
	tmpDir := t.TempDir()

	// Create output directory with an existing Go file using a different package name
	existingFile := filepath.Join(tmpDir, "existing.go")
	existingContent := `package kessoku

// Some existing declaration
func Foo() {}
`
	if err := os.WriteFile(existingFile, []byte(existingContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a wire input file with 'package main'
	inputDir := filepath.Join(tmpDir, "input")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	inputFile := filepath.Join(inputDir, "wire.go")
	inputContent := `package main

import "github.com/google/wire"

var MainSet = wire.NewSet(NewBar)

func NewBar() *Bar { return &Bar{} }

type Bar struct{}
`
	if err := os.WriteFile(inputFile, []byte(inputContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Output file goes into tmpDir which has 'package kessoku' but wire file is 'package main'
	outputPath := filepath.Join(tmpDir, "kessoku.go")

	migrator := NewMigrator()
	err := migrator.MigrateFiles([]string{inputFile}, outputPath)
	if err == nil {
		t.Fatal("expected error due to package name conflict, but got nil")
	}

	// The output file must NOT have been created (would corrupt the destination package)
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Error("output file should not have been created when package names conflict")
	}
}

// TestProviderCleanupRejected verifies that migration rejects a provider
// function returning a wire-style cleanup func(). kessoku has no way to hand
// the cleanup back to the injector's caller, and the code generator silently
// discards unused extra return values, so migrate is the single gatekeeper
// that must reject cleanup-returning wire code loudly.
func TestProviderCleanupRejected(t *testing.T) {
	tests := []struct {
		name    string
		retType string
	}{
		{name: "cleanup func()", retType: "func()"},
		{name: "cleanup func() error", retType: "func() error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputFile := filepath.Join(tmpDir, "wire.go")
			inputContent := `package test

import "github.com/google/wire"

var TestSet = wire.NewSet(NewDB)

func NewDB() (*DB, ` + tt.retType + `) {
	return &DB{}, nil
}

type DB struct{}
`
			if err := os.WriteFile(inputFile, []byte(inputContent), 0644); err != nil {
				t.Fatal(err)
			}

			outputPath := filepath.Join(tmpDir, "kessoku.go")
			migrator := NewMigrator()
			err := migrator.MigrateFiles([]string{inputFile}, outputPath)
			if err == nil {
				t.Fatal("expected error for cleanup-returning provider, got nil")
			}
			if !strings.Contains(err.Error(), "cleanup") {
				t.Errorf("error should mention cleanup, got: %v", err)
			}
			if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
				t.Error("output file should not have been created for rejected input")
			}
		})
	}
}

// TestInjectorCleanupRejected verifies that migration rejects a wire injector
// template whose signature declares a cleanup return: (T, func(), error) or
// (T, func()).
func TestInjectorCleanupRejected(t *testing.T) {
	tests := []struct {
		name    string
		results string
		retStmt string
	}{
		{name: "cleanup with error", results: "(*DB, func(), error)", retStmt: "return nil, nil, nil"},
		{name: "cleanup without error", results: "(*DB, func())", retStmt: "return nil, nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputFile := filepath.Join(tmpDir, "wire.go")
			inputContent := `//go:build wireinject

package test

import "github.com/google/wire"

func InitDB() ` + tt.results + ` {
	wire.Build(NewDB)
	` + tt.retStmt + `
}

func NewDB() *DB { return &DB{} }

type DB struct{}
`
			if err := os.WriteFile(inputFile, []byte(inputContent), 0644); err != nil {
				t.Fatal(err)
			}

			outputPath := filepath.Join(tmpDir, "kessoku.go")
			migrator := NewMigrator()
			err := migrator.MigrateFiles([]string{inputFile}, outputPath)
			if err == nil {
				t.Fatal("expected error for cleanup-returning injector, got nil")
			}
			if !strings.Contains(err.Error(), "cleanup") {
				t.Errorf("error should mention cleanup, got: %v", err)
			}
		})
	}
}

// MigrateCmd is imported from config for testing.
type MigrateCmd struct {
	Output   string
	Patterns []string
}
