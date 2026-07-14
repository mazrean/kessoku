package kessoku

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// update flag for regenerating golden files
var update = flag.Bool("update", false, "update golden files")

// TestGoldenGeneration runs golden file tests for code generation.
func TestGoldenGeneration(t *testing.T) {
	testdataDir := "testdata"

	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("failed to read testdata directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testName := entry.Name()
		t.Run(testName, func(t *testing.T) {
			// Note: parallel execution is disabled because we run the processor
			// directly on testdata files, which requires sequential access
			// to avoid race conditions when writing generated files.
			runGoldenTest(t, testdataDir, testName)
		})
	}
}

// runGoldenTest runs a single golden test case.
func runGoldenTest(t *testing.T, testdataDir, testName string) {
	t.Helper()

	srcDir := filepath.Join(testdataDir, testName)

	// Validate kessoku.go exists; skip empty stub directories.
	kessokuPath := filepath.Join(srcDir, "kessoku.go")
	if _, err := os.Stat(kessokuPath); os.IsNotExist(err) {
		t.Skipf("test case %s: missing kessoku.go (empty stub directory)", testName)
	}

	// If expected_error.txt exists, this is an error-case golden test: the
	// processor must fail and the error message must contain the expected text.
	expectedErrPath := filepath.Join(srcDir, "expected_error.txt")
	if _, statErr := os.Stat(expectedErrPath); statErr == nil {
		runGoldenErrorTest(t, testName, kessokuPath, expectedErrPath)
		return
	}

	// Generated file path (kessoku.go -> kessoku_band.go)
	generatedPath := filepath.Join(srcDir, "kessoku_band.go")

	// Clean up generated file after test (always, including update mode).
	// In update mode the generated file is also removed explicitly on the
	// happy path (line below), but we need a defer so that cleanup still
	// happens when os.WriteFile fails and t.Fatalf fires (which calls
	// runtime.Goexit, bypassing any code that follows t.Fatalf).
	defer func() {
		_ = os.Remove(generatedPath)
	}()

	// Run processor directly on the testdata directory
	// This works because testdata is within the main module
	processor := NewProcessor()
	if err := processor.ProcessFiles([]string{kessokuPath}); err != nil {
		t.Fatalf("test case %s: generation failed: %v", testName, err)
	}

	// Read generated output
	actual, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("test case %s: failed to read generated file: %v", testName, err)
	}

	// Handle update mode
	expectedPath := filepath.Join(srcDir, "expected.go")
	if *update {
		if writeErr := os.WriteFile(expectedPath, actual, 0644); writeErr != nil {
			t.Fatalf("failed to update golden file: %v", writeErr)
		}
		// Also remove the generated file in update mode
		_ = os.Remove(generatedPath)
		t.Logf("updated golden file: %s", expectedPath)
		return
	}

	// Compare with expected.go
	expected, readErr := os.ReadFile(expectedPath)
	if readErr != nil {
		t.Fatalf("test case %s: missing golden file: %s", testName, expectedPath)
	}

	if string(actual) != string(expected) {
		t.Errorf("test case %s: output mismatch:\n--- expected ---\n%s\n--- got ---\n%s",
			testName, string(expected), string(actual))
	}
}

// runGoldenErrorTest handles test cases where generation is expected to fail.
// The expected_error.txt file contains a substring that must appear in the error.
func runGoldenErrorTest(t *testing.T, testName, kessokuPath, expectedErrPath string) {
	t.Helper()

	expectedErrBytes, err := os.ReadFile(expectedErrPath)
	if err != nil {
		t.Fatalf("test case %s: failed to read expected_error.txt: %v", testName, err)
	}
	expectedErrSubstr := strings.TrimSpace(string(expectedErrBytes))

	processor := NewProcessor()
	genErr := processor.ProcessFiles([]string{kessokuPath})
	if genErr == nil {
		t.Fatalf("test case %s: expected generation to fail (containing %q) but it succeeded", testName, expectedErrSubstr)
	}

	if !strings.Contains(genErr.Error(), expectedErrSubstr) {
		t.Errorf("test case %s: error message mismatch:\n  want substring: %q\n  got:            %q",
			testName, expectedErrSubstr, genErr.Error())
	}
}
