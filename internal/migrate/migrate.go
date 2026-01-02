package migrate

import (
	"fmt"
	"go/ast"
	"log/slog"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Migrator orchestrates the migration of wire files to kessoku format.
type Migrator struct {
	parser      *Parser
	transformer *Transformer
	writer      *Writer
}

// NewMigrator creates a new Migrator instance.
func NewMigrator() *Migrator {
	return &Migrator{
		parser:      NewParser(),
		transformer: NewTransformer(),
		writer:      NewWriter(),
	}
}

// MigrateFiles migrates the specified wire files to kessoku format.
func (m *Migrator) MigrateFiles(files []string, outputPath string) error {
	// Load packages with type info
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo |
			packages.NeedName | packages.NeedFiles | packages.NeedImports,
	}

	pkgs, err := packages.Load(cfg, files...)
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

	for _, pkg := range pkgs {
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

			// Extract source imports for package reference resolution
			sourceImports := m.parser.ExtractImports(file)

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
			kessokuPatterns, err = m.transformer.Transform(patterns, pkg.Types)
			if err != nil {
				return err
			}

			results = append(results, MigrationResult{
				SourceFile:    filePath,
				Package:       pkg.Name,
				TypesPackage:  pkg.Types,
				SourceImports: sourceImports,
				Imports:       nil, // Imports are computed based on actual usage
				Patterns:      kessokuPatterns,
				Warnings:      warnings,
			})
		}
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

	// Merge results
	merged, err := m.mergeResults(results)
	if err != nil {
		return err
	}

	// Write output
	if err := m.writer.Write(merged, outputPath); err != nil {
		return err
	}

	slog.Info("Generated kessoku configuration", "output", outputPath)
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
func (m *Migrator) mergeResults(results []MigrationResult) (*MergedOutput, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no results to merge")
	}

	// Validate package names
	pkgName := results[0].Package
	for _, r := range results[1:] {
		if r.Package != pkgName {
			return nil, &MergeError{
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
					return nil, &MergeError{
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

	// Set up the TypeConverter for proper package-qualified type expressions
	var typeConverter *TypeConverter
	if results[0].TypesPackage != nil {
		typeConverter = NewTypeConverter(results[0].TypesPackage)
		m.writer.SetTypeConverter(typeConverter)
	}

	// Merge all source imports from all files
	mergedSourceImports := make(map[string]string)
	for _, r := range results {
		for name, path := range r.SourceImports {
			mergedSourceImports[name] = path
		}
	}

	// Collect imports from expressions in patterns (provider functions, values, etc.)
	if typeConverter != nil {
		for _, r := range results {
			for _, p := range r.Patterns {
				typeConverter.CollectPatternImports(p, mergedSourceImports)
			}
		}
	}

	// Generate declarations
	var decls []ast.Decl
	for _, r := range results {
		for _, p := range r.Patterns {
			decl := m.writer.PatternToDecl(p)
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
	}, nil
}
