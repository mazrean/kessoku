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
	parser *Parser
}

// NewProcessor creates a new processor instance.
func NewProcessor() *Processor {
	return &Processor{
		parser: NewParser(),
	}
}

// ProcessFiles processes specified Go files for wire generation.
func (p *Processor) ProcessFiles(files []string) error {
	for _, filename := range files {
		if err := p.processFile(filename); err != nil {
			return err
		}
	}
	return nil
}

// processFile processes a single Go file for wire generation.
func (p *Processor) processFile(filename string) error {
	slog.Debug("Processing file", "file", filename)

	metaData, builds, err := p.parser.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("parse file %s: %w", filename, err)
	}

	if len(builds) == 0 {
		return nil
	}

	slog.Info("Found inject directives", "file", filename, "count", len(builds))

	outputFileName := outputFileName(filename)
	slog.Debug("outputFileName", "outputFileName", outputFileName)

	injectors := make([]*Injector, 0, len(builds))
	for _, build := range builds {
		injector, injectorErr := CreateInjector(metaData, build)
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

	if genErr := Generate(f, filename, metaData, injectors); genErr != nil {
		return fmt.Errorf("generate: %w", genErr)
	}

	return nil
}

func outputFileName(filename string) string {
	ext := filepath.Ext(filename)
	return strings.TrimSuffix(filename, ext) + "_band" + ext
}
