package kessoku

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// writeAtomically writes content produced by fn to a temporary file in the
// same directory as dst, then renames the temp file to dst on success.
// If fn returns an error, the temporary file is removed and dst is left
// untouched.
func writeAtomically(dst string, fn func(*os.File) error) error {
	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".kessoku-tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	// cleanup removes the temp file; safe to call after tmp is closed.
	cleanup := func() {
		if removeErr := os.Remove(tmpName); removeErr != nil && !os.IsNotExist(removeErr) {
			slog.Error("Failed to remove temp file", "file", tmpName, "error", removeErr)
		}
	}

	if err := fn(tmp); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}

	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, dst); err != nil {
		cleanup()
		return fmt.Errorf("rename temp file to %s: %w", dst, err)
	}
	return nil
}

// Processor handles the overall dependency injection code generation process.
type Processor struct {
	parser *Parser
}

// NewProcessor creates a new processor instance.
func NewProcessor() *Processor {
	return &Processor{
		parser: NewParser(),
	}
}

// parsedFile holds the parse result for one input file awaiting generation.
type parsedFile struct {
	metaData *MetaData
	filename string
	builds   []*BuildDirective
	varPool  *VarPool
}

// ProcessFiles processes specified Go files for wire generation.
// All files are parsed and validated before any output is written, so a
// failure in one file does not leave partial *_band.go files behind.
func (p *Processor) ProcessFiles(files []string) error {
	parsedFiles := make([]*parsedFile, 0, len(files))
	// injector function names must be unique per package
	seenNames := make(map[string]string)

	for _, filename := range files {
		slog.Debug("Processing file", "file", filename)

		// Create a fresh VarPool per file so import aliases and package-level
		// name reservations from one file do not contaminate another file.
		fileVarPool := NewVarPool()

		metaData, builds, err := p.parser.ParseFile(filename, fileVarPool)
		if err != nil {
			return fmt.Errorf("parse file %s: %w", filename, err)
		}

		if len(builds) == 0 {
			continue
		}

		slog.Info("Found inject directives", "file", filename, "count", len(builds))

		for _, build := range builds {
			key := metaData.Package.Path + "." + build.InjectorName
			if prevFile, ok := seenNames[key]; ok {
				return fmt.Errorf("duplicate injector name %q in package %s: declared in both %s and %s",
					build.InjectorName, metaData.Package.Path, prevFile, filename)
			}
			seenNames[key] = filename
		}

		parsedFiles = append(parsedFiles, &parsedFile{
			metaData: metaData,
			filename: filename,
			builds:   builds,
			varPool:  fileVarPool,
		})
	}

	for _, pf := range parsedFiles {
		if err := p.generateFile(pf); err != nil {
			return err
		}
	}

	return nil
}

// generateFile generates the *_band.go file for a parsed input file.
// It uses pf.varPool (a fresh VarPool created per file in ProcessFiles) for
// import-alias allocation; Generate creates a per-injector snapshot for local
// variable names.
func (p *Processor) generateFile(pf *parsedFile) error {
	outputFileName := outputFileName(pf.filename)
	slog.Debug("outputFileName", "outputFileName", outputFileName)

	injectors := make([]*Injector, 0, len(pf.builds))
	for _, build := range pf.builds {
		// CreateInjector uses pf.varPool only for new import alias allocation
		// (package-level names and existing imports are already registered).
		injector, injectorErr := CreateInjector(pf.metaData, build, pf.varPool)
		if injectorErr != nil {
			return fmt.Errorf("create injector: %w", injectorErr)
		}

		injectors = append(injectors, injector)
	}

	slog.Debug("injectors", "injectors", injectors)

	// Detect duplicate injector names across sibling *_band.go files.
	// This catches the case where kessoku is invoked per-file (go:generate $GOFILE)
	// and two files in the same package declare the same injector name.
	if dupErr := checkDuplicateInjectorNames(pf.filename, outputFileName, injectors); dupErr != nil {
		return dupErr
	}

	if err := writeAtomically(outputFileName, func(f *os.File) error {
		return Generate(f, pf.filename, pf.metaData, injectors, pf.varPool)
	}); err != nil {
		return fmt.Errorf("write %s: %w", outputFileName, err)
	}

	return nil
}

func outputFileName(filename string) string {
	ext := filepath.Ext(filename)
	return strings.TrimSuffix(filename, ext) + "_band" + ext
}

// checkDuplicateInjectorNames scans sibling *_band.go files in the same
// directory for top-level function declarations that conflict with the names
// of injectors we are about to generate.  It is necessary because the
// canonical //go:generate invocation runs kessoku once per source file with a
// fresh Processor, so per-invocation deduplication is insufficient.
func checkDuplicateInjectorNames(srcFile, ownOutputFile string, injectors []*Injector) error {
	// Build a set of injector names this invocation will emit.
	wantNames := make(map[string]struct{}, len(injectors))
	for _, inj := range injectors {
		wantNames[inj.Name] = struct{}{}
	}

	dir := filepath.Dir(srcFile)

	// Collect all *_band.go files in the same directory, excluding the one we
	// are about to write so that re-running the tool is idempotent.
	absOwn, err := filepath.Abs(ownOutputFile)
	if err != nil {
		absOwn = ownOutputFile
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir %s: %w", dir, err)
	}

	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, "_band.go") {
			continue
		}

		bandPath := filepath.Join(dir, name)
		absBand, absErr := filepath.Abs(bandPath)
		if absErr != nil {
			absBand = bandPath
		}
		// Skip the output file we are about to (re)write.
		if absBand == absOwn {
			continue
		}

		astFile, parseErr := parser.ParseFile(fset, bandPath, nil, 0)
		if parseErr != nil {
			// If the sibling band file cannot be parsed, skip it rather than
			// blocking generation — a broken sibling is not our problem.
			slog.Warn("failed to parse sibling band file", "file", bandPath, "error", parseErr)
			continue
		}

		for _, decl := range astFile.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Name == nil {
				continue
			}
			if _, clash := wantNames[funcDecl.Name.Name]; clash {
				return fmt.Errorf(
					"duplicate injector name %q: already declared in %s",
					funcDecl.Name.Name, bandPath,
				)
			}
		}
	}

	return nil
}
