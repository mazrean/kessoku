package kessoku

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Processor handles the overall dependency injection code generation process.
type Processor struct {
	parser  *Parser
	varPool *VarPool
}

// NewProcessor creates a new processor instance.
func NewProcessor() *Processor {
	return &Processor{
		parser:  NewParser(),
		varPool: NewVarPool(),
	}
}

// parsedFile holds the parse result for one input file awaiting generation.
type parsedFile struct {
	metaData *MetaData
	filename string
	builds   []*BuildDirective
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

		metaData, builds, err := p.parser.ParseFile(filename, p.varPool)
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
func (p *Processor) generateFile(pf *parsedFile) error {
	outputFileName := outputFileName(pf.filename)
	slog.Debug("outputFileName", "outputFileName", outputFileName)

	injectors := make([]*Injector, 0, len(pf.builds))
	for _, build := range pf.builds {
		injector, injectorErr := CreateInjector(pf.metaData, build, p.varPool)
		if injectorErr != nil {
			return fmt.Errorf("create injector: %w", injectorErr)
		}

		injectors = append(injectors, injector)
	}

	slog.Debug("injectors", "injectors", injectors)

	f, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("create file %s: %w", outputFileName, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			slog.Error("Failed to close file", "error", closeErr)
		}
	}()

	if genErr := Generate(f, pf.filename, pf.metaData, injectors, p.varPool); genErr != nil {
		return fmt.Errorf("generate: %w", genErr)
	}

	return nil
}

func outputFileName(filename string) string {
	ext := filepath.Ext(filename)
	return strings.TrimSuffix(filename, ext) + "_band" + ext
}
