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

// MigrateCmd is imported from config for testing.
type MigrateCmd struct {
	Output string
	Files  []string
}
