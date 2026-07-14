package migrate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Migrator orchestrates the migration of wire files to kessoku format.
type Migrator struct {
	parser      *Parser
	transformer *Transformer
}

// NewMigrator creates a new Migrator instance.
func NewMigrator() *Migrator {
	return &Migrator{
		parser:      NewParser(),
		transformer: NewTransformer(),
	}
}

// MigrateFiles migrates the specified wire files to kessoku format.
// patterns are Go package patterns (e.g., "./", "./pkg/...", "example.com/pkg").
func (m *Migrator) MigrateFiles(patterns []string, outputPath string) error {
	// Load packages with type info
	// Use wireinject build tag to load wire configuration files
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo |
			packages.NeedName | packages.NeedFiles | packages.NeedImports,
		BuildFlags: []string{"-tags=wireinject"},
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return fmt.Errorf("failed to load packages: %w", err)
	}

	// Check for load errors
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			return m.convertPackageError(pkg.Errors[0])
		}
	}

	// Extract wire patterns from each file
	var results []MigrationResult
	var allWarnings []Warning

	// sharedTypeConverter accumulates import requirements across all packages so that
	// the single output file has a complete import block.  Its currentPkg field is
	// updated inside the loop to match the package currently being transformed; this
	// ensures that TypeToExpr emits qualified identifiers (e.g. "pkg.Doer") for types
	// that come from external packages rather than treating them as same-package types.
	// Using pkgs[0] here was the original bug: when packages.Load returns a dependency
	// package before the package being migrated, every interface from that dependency is
	// misidentified as a local type and loses its package qualifier.
	var sharedTypeConverter *TypeConverter

	for _, pkg := range pkgs {
		// Update the shared TypeConverter's view of the current package before
		// processing each package's files.  This guarantees that TypeToExpr sees the
		// correct "home" package and emits qualified names for all external types.
		if pkg.Types != nil {
			if sharedTypeConverter == nil {
				sharedTypeConverter = NewTypeConverter(pkg.Types)
			} else {
				sharedTypeConverter.currentPkg = pkg.Types
			}
		}
		// Pre-populate the transformer's set index with all WireNewSet declarations
		// from every file in this package. This allows anyProviderReturnsError (called
		// from transformBuild) to resolve WireSetRef entries that are defined in a
		// different file than the wire.Build injector, preventing spurious
		// kessoku.Value((error)(nil)) sentinels in cross-file packages.
		var allPkgPatterns []WirePattern
		for _, file := range pkg.Syntax {
			wireImport := m.parser.FindWireImport(file)
			if wireImport == "" {
				continue
			}
			filePatterns, _ := m.parser.ExtractPatterns(file, pkg.TypesInfo, wireImport, "")
			allPkgPatterns = append(allPkgPatterns, filePatterns...)
		}
		m.transformer.setIndex = buildSetIndex(allPkgPatterns)
		// Pre-populate bindVarTypes package-wide so that a top-level wire.Bind
		// variable defined in file A is visible when transforming a wire.NewSet in
		// file B that references it.  Without this, t.bindVarTypes was reset to
		// only the current file's patterns on each Transform call, causing the
		// cross-file WireSetRef lookup in collectBoundTypes to miss the bind var
		// and emit a duplicate kessoku.Provide for the same implementation type.
		m.transformer.bindVarTypes = buildBindVarTypes(allPkgPatterns)

		// Build a map from syntax position to file path
		syntaxFiles := pkg.CompiledGoFiles
		if len(syntaxFiles) == 0 {
			syntaxFiles = pkg.GoFiles
		}

		for i, file := range pkg.Syntax {
			var filePath string
			switch {
			case i < len(syntaxFiles):
				filePath = syntaxFiles[i]
			case file.Name != nil:
				filePath = file.Name.Name + ".go"
			default:
				filePath = "unknown.go"
			}
			slog.Debug("Processing file", "file", filePath)

			// Check for wire import
			wireImport := m.parser.FindWireImport(file)
			if wireImport == "" {
				allWarnings = append(allWarnings, Warning{
					Code:    WarnNoWireImport,
					Message: fmt.Sprintf("No wire import found in %s", filePath),
				})
				continue
			}

			// Build a map from import path to actual package name using the type-checker's
			// view of imported packages.  This is required so that major-version suffix paths
			// (e.g. "example.com/lib/v2" → package name "lib") and gopkg.in-style paths
			// (e.g. "gopkg.in/yaml.v3" → package name "yaml") are keyed correctly.
			pkgNameByPath := make(map[string]string, len(pkg.Imports))
			for importPath, importedPkg := range pkg.Imports {
				pkgNameByPath[importPath] = importedPkg.Name
			}

			// Extract source imports for package reference resolution
			sourceImports, explicitAliasPaths := m.parser.ExtractImports(file, pkgNameByPath)

			// Extract patterns
			patterns, warnings := m.parser.ExtractPatterns(file, pkg.TypesInfo, wireImport, filePath)
			allWarnings = append(allWarnings, warnings...)

			if len(patterns) == 0 {
				allWarnings = append(allWarnings, Warning{
					Code:    WarnNoWirePatterns,
					Message: fmt.Sprintf("No wire patterns found in %s", filePath),
				})
				continue
			}

			// Transform patterns
			var kessokuPatterns []KessokuPattern
			kessokuPatterns, err = m.transformer.Transform(patterns, pkg.Types, sharedTypeConverter)
			if err != nil {
				return err
			}

			results = append(results, MigrationResult{
				SourceFile:         filePath,
				Package:            pkg.Name,
				TypesPackage:       pkg.Types,
				SourceImports:      sourceImports,
				ExplicitAliasPaths: explicitAliasPaths,
				Imports:            nil, // Imports are computed based on actual usage
				Patterns:           kessokuPatterns,
				Warnings:           warnings,
			})
		}

		// Reset the set index and bind var types after each package so cross-package
		// contamination cannot occur when MigrateFiles is called with multi-package patterns.
		m.transformer.setIndex = nil
		m.transformer.bindVarTypes = nil
	}

	// Log warnings
	for _, w := range allWarnings {
		slog.Warn(w.Message)
	}

	// Check if we have any results
	if len(results) == 0 {
		slog.Warn("No wire patterns found in any input file, no output generated")
		return nil
	}

	// Merge results and create writer
	merged, writer, err := m.mergeResults(results, sharedTypeConverter)
	if err != nil {
		return err
	}

	// Validate that the generated package name matches the existing package in the output directory.
	// This prevents writing a file with a conflicting package declaration (e.g., 'package main'
	// into a directory that already contains 'package kessoku' files), which would break builds.
	if err := m.validateOutputPackage(outputPath, merged.Package); err != nil {
		return err
	}

	// Write output
	if err := writer.Write(merged, outputPath); err != nil {
		return err
	}

	slog.Info("Generated kessoku configuration", "output", outputPath)
	return nil
}

// validateOutputPackage checks that the generated package name matches the existing package
// declarations in the output directory (excluding the output file itself, if it exists).
// Returns an error if any existing Go file in the directory declares a different package name.
func (m *Migrator) validateOutputPackage(outputPath string, generatedPkg string) error {
	outputDir := filepath.Dir(outputPath)
	outputBase := filepath.Base(outputPath)

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Output directory doesn't exist yet; nothing to conflict with.
			return nil
		}
		return fmt.Errorf("failed to read output directory %s: %w", outputDir, err)
	}

	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip non-Go files and the output file itself (it may already exist from a previous run).
		if !strings.HasSuffix(name, ".go") || name == outputBase {
			continue
		}

		filePath := filepath.Join(outputDir, name)
		f, err := parser.ParseFile(fset, filePath, nil, parser.PackageClauseOnly)
		if err != nil {
			// Skip files that cannot be parsed (e.g., test build constraints).
			continue
		}

		existingPkg := f.Name.Name
		if existingPkg != generatedPkg {
			return fmt.Errorf(
				"package name conflict: output file would declare %q but %s already declares %q; "+
					"use -o to choose an output path in the correct directory or ensure the wire source uses the right package name",
				generatedPkg, filePath, existingPkg,
			)
		}
	}

	return nil
}

// convertPackageError converts packages.Error to ParseError.
func (m *Migrator) convertPackageError(pkgErr packages.Error) error {
	kind := ParseErrorSyntax
	if pkgErr.Kind == packages.TypeError {
		kind = ParseErrorTypeResolution
	}

	message := pkgErr.Msg
	if pkgErr.Pos != "" {
		message = pkgErr.Pos + ": " + pkgErr.Msg
	}

	file := ""
	if pkgErr.Pos != "" {
		if idx := strings.Index(pkgErr.Pos, ".go:"); idx > 0 {
			file = pkgErr.Pos[:idx+3]
		}
	}

	return &ParseError{
		Kind:    kind,
		File:    file,
		Message: message,
	}
}

// mergeResults merges multiple migration results into a single output.
// Returns the merged output and the writer configured for this output.
func (m *Migrator) mergeResults(results []MigrationResult, typeConverter *TypeConverter) (*MergedOutput, *Writer, error) {
	if len(results) == 0 {
		return nil, nil, fmt.Errorf("no results to merge")
	}

	// Validate package names
	pkgName := results[0].Package
	for _, r := range results[1:] {
		if r.Package != pkgName {
			return nil, nil, &MergeError{
				Kind:     MergeErrorPackageMismatch,
				Message:  fmt.Sprintf("package mismatch: %s vs %s", pkgName, r.Package),
				Files:    []string{results[0].SourceFile, r.SourceFile},
				Packages: []string{pkgName, r.Package},
			}
		}
	}

	// Check for identifier collisions
	identifiers := make(map[string]string) // identifier -> file
	for _, r := range results {
		for _, p := range r.Patterns {
			if set, ok := p.(*KessokuSet); ok {
				if existingFile, exists := identifiers[set.VarName]; exists {
					return nil, nil, &MergeError{
						Kind:       MergeErrorNameCollision,
						Message:    fmt.Sprintf("identifier %q defined in multiple files", set.VarName),
						Files:      []string{existingFile, r.SourceFile},
						Identifier: set.VarName,
					}
				}
				identifiers[set.VarName] = r.SourceFile
			}
		}
	}

	// Collect imports from expressions in patterns (provider functions, values, etc.)
	// Use each file's own source imports to correctly resolve same-named packages from different files.
	// Also pass the explicit alias paths so that Imports() knows which aliases must always be emitted.
	if typeConverter != nil {
		for _, r := range results {
			for _, p := range r.Patterns {
				typeConverter.CollectPatternImports(p, r.SourceImports, r.ExplicitAliasPaths)
			}
		}
	}

	// Create writer with the TypeConverter for proper package-qualified type expressions
	writer := NewWriter(typeConverter)

	// Generate declarations
	var decls []ast.Decl
	for _, r := range results {
		for _, p := range r.Patterns {
			decl := writer.PatternToDecl(p)
			if decl != nil {
				decls = append(decls, decl)
			}
		}
	}

	// Collect imports: always include kessoku, plus any imports needed for external types
	imports := []ImportSpec{{Path: "github.com/mazrean/kessoku"}}
	if typeConverter != nil {
		collectedImports := typeConverter.Imports()
		imports = append(imports, collectedImports...)
	}

	return &MergedOutput{
		Package:       pkgName,
		Imports:       imports,
		TopLevelDecls: decls,
	}, writer, nil
}
