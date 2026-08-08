package kessoku_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestRootPackageConsistency ensures all .go files at the module root belong to
// package "kessoku" or "kessoku_test". A stray file with a different package
// declaration (e.g. package tc6) causes go/packages.Load to fail with
// "found packages X and Y" for every downstream test and go build ./...
func TestRootPackageConsistency(t *testing.T) {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine source file path")
	}
	rootDir := filepath.Dir(thisFile)

	entries, err := os.ReadDir(rootDir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", rootDir, err)
	}

	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(rootDir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
		if err != nil {
			t.Errorf("parse %s: %v", e.Name(), err)
			continue
		}
		pkg := f.Name.Name
		if pkg != "kessoku" && pkg != "kessoku_test" {
			t.Errorf("file %s declares package %q; root directory must only contain package kessoku or kessoku_test", e.Name(), pkg)
		}
	}
}
