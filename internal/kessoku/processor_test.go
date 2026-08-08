package kessoku

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewProcessor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "create new processor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			processor := NewProcessor()

			if processor == nil {
				t.Fatal("Expected processor to be created")
			}

			if processor.parser == nil {
				t.Error("Expected parser to be initialized")
			}
		})
	}
}

// TestWriteAtomically verifies the atomic write semantics required by BUG-09.
// With the old os.Create approach the destination file was truncated before
// generation ran, so a failure left a 0-byte file behind.  With writeAtomically
// the destination is only replaced on success and the original is preserved on
// failure.
func TestWriteAtomically(t *testing.T) {
	t.Parallel()

	const sentinel = "ORIGINAL_CONTENT"

	t.Run("success: creates destination file with written content", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		dst := filepath.Join(dir, "out.go")

		err := writeAtomically(dst, func(f *os.File) error {
			_, werr := fmt.Fprint(f, "hello")
			return werr
		})
		if err != nil {
			t.Fatalf("writeAtomically returned unexpected error: %v", err)
		}

		got, readErr := os.ReadFile(dst)
		if readErr != nil {
			t.Fatalf("failed to read destination file: %v", readErr)
		}
		if string(got) != "hello" {
			t.Errorf("destination content = %q; want %q", string(got), "hello")
		}
	})

	t.Run("failure: preserves pre-existing destination file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		dst := filepath.Join(dir, "out.go")

		// Pre-create the destination with known content (simulates a valid
		// band file that was previously generated).
		if err := os.WriteFile(dst, []byte(sentinel), 0o644); err != nil {
			t.Fatalf("failed to create pre-existing file: %v", err)
		}

		writeErr := errors.New("generate failed")
		err := writeAtomically(dst, func(f *os.File) error {
			// Write some bytes then fail, mimicking a partial Generate call.
			_, _ = fmt.Fprint(f, "partial")
			return writeErr
		})
		if err == nil {
			t.Fatal("writeAtomically should have returned an error")
		}

		// The original file must be untouched.
		got, readErr := os.ReadFile(dst)
		if readErr != nil {
			t.Fatalf("failed to read destination file after failure: %v", readErr)
		}
		if string(got) != sentinel {
			t.Errorf("destination content after failure = %q; want original %q", string(got), sentinel)
		}
	})

	t.Run("success: new file gets 0644 permissions, not CreateTemp's 0600", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		dst := filepath.Join(dir, "out.go")

		err := writeAtomically(dst, func(f *os.File) error {
			_, werr := fmt.Fprint(f, "hello")
			return werr
		})
		if err != nil {
			t.Fatalf("writeAtomically returned unexpected error: %v", err)
		}

		fi, statErr := os.Stat(dst)
		if statErr != nil {
			t.Fatalf("failed to stat destination file: %v", statErr)
		}
		if fi.Mode().Perm() != 0o644 {
			t.Errorf("destination permissions = %v; want %v", fi.Mode().Perm(), os.FileMode(0o644))
		}
	})

	t.Run("success: preserves pre-existing destination permissions", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		dst := filepath.Join(dir, "out.go")

		if err := os.WriteFile(dst, []byte(sentinel), 0o664); err != nil {
			t.Fatalf("failed to create pre-existing file: %v", err)
		}
		// WriteFile's mode is subject to umask; force the exact mode.
		if err := os.Chmod(dst, 0o664); err != nil {
			t.Fatalf("failed to chmod pre-existing file: %v", err)
		}

		err := writeAtomically(dst, func(f *os.File) error {
			_, werr := fmt.Fprint(f, "hello")
			return werr
		})
		if err != nil {
			t.Fatalf("writeAtomically returned unexpected error: %v", err)
		}

		fi, statErr := os.Stat(dst)
		if statErr != nil {
			t.Fatalf("failed to stat destination file: %v", statErr)
		}
		if fi.Mode().Perm() != 0o664 {
			t.Errorf("destination permissions = %v; want preserved %v", fi.Mode().Perm(), os.FileMode(0o664))
		}
	})

	t.Run("failure: no temp files left behind", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		dst := filepath.Join(dir, "out.go")

		_ = writeAtomically(dst, func(_ *os.File) error {
			return errors.New("generate failed")
		})

		// After a failure the directory must contain no .kessoku-tmp-* files.
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to read temp dir: %v", err)
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".kessoku-tmp-") {
				t.Errorf("temp file not cleaned up: %s", e.Name())
			}
		}
	})
}

func TestProcessFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		errorContains          string
		files                  []fileContent
		expectedGeneratedFiles []string
		shouldError            bool
	}{
		{
			name: "mixed files with and without kessoku",
			files: []fileContent{
				{
					name: "test1.go",
					content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	Value string
}

func NewConfig() *Config {
	return &Config{Value: "test"}
}

type Service struct {
	config *Config
}

func NewService(config *Config) *Service {
	return &Service{config: config}
}

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewService),
)
`,
				},
				{
					name: "test2.go",
					content: `package main

type SimpleStruct struct {
	Value string
}

func main() {
	// no kessoku
}
`,
				},
			},
			expectedGeneratedFiles: []string{"test1_band.go"},
			shouldError:            false,
		},
		{
			name: "single valid file",
			files: []fileContent{
				{
					name: "test.go",
					content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	Value string
}

func NewConfig() *Config {
	return &Config{Value: "test"}
}

type Service struct {
	config *Config
}

func NewService(config *Config) *Service {
	return &Service{config: config}
}

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewService),
)
`,
				},
			},
			expectedGeneratedFiles: []string{"test_band.go"},
			shouldError:            false,
		},
		{
			name: "file without kessoku",
			files: []fileContent{
				{
					name: "test.go",
					content: `package main

type Service struct {
	Value string
}

func main() {
	// no kessoku
}
`,
				},
			},
			expectedGeneratedFiles: []string{},
			shouldError:            false,
		},
		{
			name: "multiple injectors",
			files: []fileContent{
				{
					name: "test.go",
					content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	Value string
}

func NewConfig() *Config {
	return &Config{Value: "test"}
}

type Service1 struct {
	config *Config
}

func NewService1(config *Config) *Service1 {
	return &Service1{config: config}
}

type Service2 struct {
	config *Config
}

func NewService2(config *Config) *Service2 {
	return &Service2{config: config}
}

var _ = kessoku.Inject[*Service1](
	"InitializeService1",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewService1),
)

var _ = kessoku.Inject[*Service2](
	"InitializeService2",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewService2),
)
`,
				},
			},
			expectedGeneratedFiles: []string{"test_band.go"},
			shouldError:            false,
		},
		{
			name: "invalid syntax",
			files: []fileContent{
				{
					name: "test.go",
					content: `package main

import "github.com/mazrean/kessoku"

func invalid syntax here {
`,
				},
			},
			expectedGeneratedFiles: []string{},
			shouldError:            true,
			errorContains:          "parse file",
		},
		{
			name: "nonexistent file",
			files: []fileContent{
				{
					name:        "nonexistent.go",
					content:     "",
					shouldWrite: &[]bool{false}[0],
				},
			},
			expectedGeneratedFiles: []string{},
			shouldError:            true,
		},
		{
			name: "struct provider field expansion",
			files: []fileContent{
				{
					name: "test.go",
					content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	DBHost string
	DBPort int
}

func NewConfig() *Config {
	return &Config{DBHost: "localhost", DBPort: 5432}
}

type Database struct {
	host string
	port int
}

func NewDatabase(host string, port int) *Database {
	return &Database{host: host, port: port}
}

var _ = kessoku.Inject[*Database](
	"InitializeDatabase",
	kessoku.Provide(NewConfig),
	kessoku.Struct[*Config](),
	kessoku.Provide(NewDatabase),
)
`,
				},
			},
			expectedGeneratedFiles: []string{"test_band.go"},
			shouldError:            false,
		},
		{
			name: "struct different field types",
			files: []fileContent{
				{
					name: "test.go",
					content: `package main

import "github.com/mazrean/kessoku"

type CustomType struct {
	Value string
}

type Config struct {
	Host   string
	Port   int
	Custom *CustomType
}

func NewConfig() *Config {
	return &Config{Host: "localhost", Port: 5432, Custom: &CustomType{Value: "test"}}
}

type Service struct {
	host   string
	port   int
	custom *CustomType
}

func NewService(host string, port int, custom *CustomType) *Service {
	return &Service{host: host, port: port, custom: custom}
}

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Struct[*Config](),
	kessoku.Provide(NewService),
)
`,
				},
			},
			expectedGeneratedFiles: []string{"test_band.go"},
			shouldError:            false,
		},
		{
			name: "struct pointer and value fields",
			files: []fileContent{
				{
					name: "test.go",
					content: `package main

import "github.com/mazrean/kessoku"

type Logger struct {
	Level string
}

type Config struct {
	Host   string
	Logger *Logger
}

func NewConfig() *Config {
	return &Config{Host: "localhost", Logger: &Logger{Level: "info"}}
}

type Service struct {
	host   string
	logger *Logger
}

func NewService(host string, logger *Logger) *Service {
	return &Service{host: host, logger: logger}
}

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Struct[*Config](),
	kessoku.Provide(NewService),
)
`,
				},
			},
			expectedGeneratedFiles: []string{"test_band.go"},
			shouldError:            false,
		},
		{
			name: "struct with async provider",
			files: []fileContent{
				{
					name: "test.go",
					content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	DBHost string
	DBPort int
}

func NewConfig() *Config {
	return &Config{DBHost: "localhost", DBPort: 5432}
}

type Database struct {
	host string
	port int
}

func NewDatabase(host string, port int) *Database {
	return &Database{host: host, port: port}
}

var _ = kessoku.Inject[*Database](
	"InitializeDatabase",
	kessoku.Async(kessoku.Provide(NewConfig)),
	kessoku.Struct[*Config](),
	kessoku.Provide(NewDatabase),
)
`,
				},
			},
			expectedGeneratedFiles: []string{"test_band.go"},
			shouldError:            false,
		},
		{
			name: "struct inside set",
			files: []fileContent{
				{
					name: "test.go",
					content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	DBHost string
	DBPort int
}

func NewConfig() *Config {
	return &Config{DBHost: "localhost", DBPort: 5432}
}

type Database struct {
	host string
	port int
}

func NewDatabase(host string, port int) *Database {
	return &Database{host: host, port: port}
}

var ConfigSet = kessoku.Set(
	kessoku.Provide(NewConfig),
	kessoku.Struct[*Config](),
)

var _ = kessoku.Inject[*Database](
	"InitializeDatabase",
	ConfigSet,
	kessoku.Provide(NewDatabase),
)
`,
				},
			},
			expectedGeneratedFiles: []string{"test_band.go"},
			shouldError:            false,
		},
		{
			name: "struct embedded value type",
			files: []fileContent{
				{
					name: "test.go",
					content: `package main

import "github.com/mazrean/kessoku"

type BaseConfig struct {
	Debug bool
}

type Config struct {
	BaseConfig
	Name string
}

func NewConfig() *Config {
	return &Config{BaseConfig: BaseConfig{Debug: true}, Name: "test"}
}

type Service struct {
	base BaseConfig
	name string
}

func NewService(base BaseConfig, name string) *Service {
	return &Service{base: base, name: name}
}

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Struct[*Config](),
	kessoku.Provide(NewService),
)
`,
				},
			},
			expectedGeneratedFiles: []string{"test_band.go"},
			shouldError:            false,
		},
		{
			name: "struct embedded pointer type",
			files: []fileContent{
				{
					name: "test.go",
					content: `package main

import "github.com/mazrean/kessoku"

type Logger struct {
	Level string
}

type Config struct {
	*Logger
	Name string
}

func NewConfig() *Config {
	return &Config{Logger: &Logger{Level: "info"}, Name: "test"}
}

type Service struct {
	logger *Logger
	name   string
}

func NewService(logger *Logger, name string) *Service {
	return &Service{logger: logger, name: name}
}

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Struct[*Config](),
	kessoku.Provide(NewService),
)
`,
				},
			},
			expectedGeneratedFiles: []string{"test_band.go"},
			shouldError:            false,
		},
		{
			name: "async wrapped struct provider",
			files: []fileContent{
				{
					name: "test.go",
					content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	Host string
	Port int
}

func NewConfig() *Config {
	return &Config{Host: "localhost", Port: 5432}
}

type Database struct {
	host string
	port int
}

func NewDatabase(host string, port int) *Database {
	return &Database{host: host, port: port}
}

var _ = kessoku.Inject[*Database](
	"InitializeDatabase",
	kessoku.Provide(NewConfig),
	kessoku.Async(kessoku.Struct[*Config]()),
	kessoku.Provide(NewDatabase),
)
`,
				},
			},
			expectedGeneratedFiles: []string{"test_band.go"},
			shouldError:            false,
		},
		{
			name: "async struct provider with dependent async provider",
			files: []fileContent{
				{
					name: "test.go",
					content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	Host string
	Port int
}

func NewConfig() *Config {
	return &Config{Host: "localhost", Port: 5432}
}

type Database struct {
	host string
	port int
}

func NewDatabase(host string, port int) *Database {
	return &Database{host: host, port: port}
}

type Service struct {
	db *Database
}

func NewService(db *Database) *Service {
	return &Service{db: db}
}

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Async(kessoku.Provide(NewConfig)),
	kessoku.Async(kessoku.Struct[*Config]()),
	kessoku.Async(kessoku.Provide(NewDatabase)),
	kessoku.Provide(NewService),
)
`,
				},
			},
			expectedGeneratedFiles: []string{"test_band.go"},
			shouldError:            false,
		},
		{
			name: "bind wrapped struct provider",
			files: []fileContent{
				{
					name: "test.go",
					content: `package main

import "github.com/mazrean/kessoku"

type ConfigProvider interface {
	GetHost() string
}

type Config struct {
	Host string
	Port int
}

func (c *Config) GetHost() string {
	return c.Host
}

func NewConfig() *Config {
	return &Config{Host: "localhost", Port: 5432}
}

type Database struct {
	host string
	port int
}

func NewDatabase(host string, port int) *Database {
	return &Database{host: host, port: port}
}

var _ = kessoku.Inject[*Database](
	"InitializeDatabase",
	kessoku.Provide(NewConfig),
	kessoku.Bind[ConfigProvider](kessoku.Struct[*Config]()),
	kessoku.Provide(NewDatabase),
)
`,
				},
			},
			expectedGeneratedFiles: []string{"test_band.go"},
			shouldError:            false,
		},
		{
			name: "struct with multiple fields of same type returns clear error",
			files: []fileContent{
				{
					name: "test.go",
					content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	Host     string
	Username string
}

func NewConfig() *Config {
	return &Config{Host: "localhost", Username: "admin"}
}

type DB struct {
	host     string
	username string
}

func NewDB(host, username string) *DB {
	return &DB{host: host, username: username}
}

var _ = kessoku.Inject[*DB](
	"Init",
	kessoku.Provide(NewConfig),
	kessoku.Struct[*Config](),
	kessoku.Provide(NewDB),
)
`,
				},
			},
			shouldError:   true,
			errorContains: "same type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			var filePaths []string

			// Create test files
			for _, file := range tt.files {
				filePath := filepath.Join(tempDir, file.name)
				if file.shouldWrite == nil || *file.shouldWrite {
					if err := os.WriteFile(filePath, []byte(file.content), 0644); err != nil {
						t.Fatalf("Failed to write test file %s: %v", file.name, err)
					}
				}
				filePaths = append(filePaths, filePath)
			}

			processor := NewProcessor()
			err := processor.ProcessFiles(filePaths)

			if tt.shouldError {
				if err == nil {
					t.Fatal("Expected ProcessFiles to fail")
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("ProcessFiles failed: %v", err)
			}

			// Check generated files
			for _, expectedFile := range tt.expectedGeneratedFiles {
				generatedPath := filepath.Join(tempDir, expectedFile)
				if _, statErr := os.Stat(generatedPath); os.IsNotExist(statErr) {
					t.Errorf("Expected generated file %s to exist", expectedFile)
					continue
				}

				// Verify generated content
				generated, readErr := os.ReadFile(generatedPath)
				if readErr != nil {
					t.Fatalf("Failed to read generated file %s: %v", expectedFile, readErr)
				}

				generatedStr := string(generated)
				expectedContent := []string{
					"// Code generated by kessoku. DO NOT EDIT.",
					"package main",
				}

				for _, expected := range expectedContent {
					if !strings.Contains(generatedStr, expected) {
						t.Errorf("Expected generated file %s to contain %q", expectedFile, expected)
					}
				}
			}

			// Check that no unexpected files were generated
			entries, err := os.ReadDir(tempDir)
			if err != nil {
				t.Fatalf("Failed to read temp directory: %v", err)
			}

			var generatedFiles []string
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), "_band.go") {
					generatedFiles = append(generatedFiles, entry.Name())
				}
			}

			if len(generatedFiles) != len(tt.expectedGeneratedFiles) {
				t.Errorf("Expected %d generated files, got %d: %v",
					len(tt.expectedGeneratedFiles), len(generatedFiles), generatedFiles)
			}
		})
	}
}

// TestDuplicateInjectorNamesAcrossFiles verifies that kessoku detects duplicate
// injector names when the tool is invoked per-file (the go:generate $GOFILE
// pattern), where each invocation creates a fresh Processor.  BUG-24.
func TestDuplicateInjectorNamesAcrossFiles(t *testing.T) {
	t.Parallel()

	const sharedPkg = `package main

import "github.com/mazrean/kessoku"

type Config struct {
	Value string
}

func NewConfig() *Config {
	return &Config{Value: "test"}
}

type Service struct {
	config *Config
}

func NewService(config *Config) *Service {
	return &Service{config: config}
}
`

	// file1.go and file2.go both declare an injector named "InitializeApp".
	file1Content := sharedPkg + `
var _ = kessoku.Inject[*Service](
	"InitializeApp",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewService),
)
`
	file2Content := sharedPkg + `
var _ = kessoku.Inject[*Service](
	"InitializeApp",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewService),
)
`

	tempDir := t.TempDir()
	file1 := filepath.Join(tempDir, "file1.go")
	file2 := filepath.Join(tempDir, "file2.go")

	if err := os.WriteFile(file1, []byte(file1Content), 0644); err != nil {
		t.Fatalf("write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(file2Content), 0644); err != nil {
		t.Fatalf("write file2: %v", err)
	}

	// First invocation: process file1, should succeed.
	proc1 := NewProcessor()
	if err := proc1.ProcessFiles([]string{file1}); err != nil {
		t.Fatalf("first ProcessFiles (file1) failed unexpectedly: %v", err)
	}

	// Verify file1_band.go was generated.
	if _, err := os.Stat(filepath.Join(tempDir, "file1_band.go")); os.IsNotExist(err) {
		t.Fatal("expected file1_band.go to be created")
	}

	// Second invocation: process file2 with a fresh Processor (simulating
	// go:generate $GOFILE on the second file).  It must fail because
	// file1_band.go already declares "InitializeApp".
	proc2 := NewProcessor()
	err := proc2.ProcessFiles([]string{file2})
	if err == nil {
		t.Fatal("expected ProcessFiles (file2) to fail with duplicate name error, but it succeeded")
	}
	if !containsString(err.Error(), "duplicate injector name") {
		t.Errorf("expected error to mention duplicate injector name, got: %v", err)
	}
}

// TestSourceFileNamedBandGoNoFalsePositive verifies that when the kessoku
// source file itself ends in _band.go (e.g. inject_band.go), processing it
// does not produce a false-positive "duplicate injector name" error.
//
// Regression test for the bug where checkDuplicateInjectorNames scanned
// sibling *_band.go files but only excluded ownOutputFile from the scan.
// When the source file itself matched the *_band.go suffix it was scanned,
// and any function it declared with the same name as an injector was
// incorrectly flagged as a duplicate.
func TestSourceFileNamedBandGoNoFalsePositive(t *testing.T) {
	t.Parallel()

	// inject_band.go: the source file itself ends in _band.go.
	// It contains both a helper function InitFoo and a kessoku.Inject call
	// that will generate a function also named InitFoo.
	srcContent := `package main

import "github.com/mazrean/kessoku"

type Foo struct{}

func NewFoo() *Foo { return &Foo{} }

// InitFoo is a helper declared in the source file itself.
func InitFoo() *Foo { return NewFoo() }

var _ = kessoku.Inject[*Foo](
	"InitFoo",
	kessoku.Provide(NewFoo),
)
`

	tempDir := t.TempDir()
	// Source file is named inject_band.go — matches the *_band.go pattern.
	srcFile := filepath.Join(tempDir, "inject_band.go")
	if err := os.WriteFile(srcFile, []byte(srcContent), 0644); err != nil {
		t.Fatalf("write inject_band.go: %v", err)
	}

	proc := NewProcessor()
	// Must succeed — the source file is not a previously-generated output.
	if err := proc.ProcessFiles([]string{srcFile}); err != nil {
		t.Fatalf("ProcessFiles returned unexpected error: %v", err)
	}

	// The output file should be inject_band_band.go.
	outFile := filepath.Join(tempDir, "inject_band_band.go")
	if _, err := os.Stat(outFile); os.IsNotExist(err) {
		t.Fatal("expected inject_band_band.go to be created")
	}
}

// TestProcessFilesGraphValidationBeforeWrite verifies that a graph-level error
// (e.g. struct with duplicate field types, which is caught by CreateInjector
// rather than ParseFile) in a later file does not leave the earlier file's
// *_band.go on disk.  This is the guarantee expressed by the "all files are
// parsed and graph-validated before any output is written" comment.
func TestProcessFilesGraphValidationBeforeWrite(t *testing.T) {
	t.Parallel()

	// file1.go: valid — should produce file1_band.go on success.
	file1Content := `package main

import "github.com/mazrean/kessoku"

type Config struct {
	Value string
}

func NewConfig() *Config {
	return &Config{Value: "test"}
}

type Service struct {
	config *Config
}

func NewService(config *Config) *Service {
	return &Service{config: config}
}

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewService),
)
`

	// file2.go: graph-level error — Struct[*Config2] has two fields of type
	// string, which CreateInjector (not ParseFile) detects.
	file2Content := `package main

import "github.com/mazrean/kessoku"

type Config2 struct {
	Host     string
	Username string
}

func NewConfig2() *Config2 {
	return &Config2{Host: "localhost", Username: "admin"}
}

type DB struct {
	host     string
	username string
}

func NewDB(host, username string) *DB {
	return &DB{host: host, username: username}
}

var _ = kessoku.Inject[*DB](
	"InitDB",
	kessoku.Provide(NewConfig2),
	kessoku.Struct[*Config2](),
	kessoku.Provide(NewDB),
)
`

	tempDir := t.TempDir()
	file1 := filepath.Join(tempDir, "file1.go")
	file2 := filepath.Join(tempDir, "file2.go")

	if err := os.WriteFile(file1, []byte(file1Content), 0644); err != nil {
		t.Fatalf("write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(file2Content), 0644); err != nil {
		t.Fatalf("write file2: %v", err)
	}

	proc := NewProcessor()
	err := proc.ProcessFiles([]string{file1, file2})
	if err == nil {
		t.Fatal("expected ProcessFiles to fail due to graph error in file2")
	}

	// After the error, no *_band.go files must exist.  The guarantee is that
	// graph validation for all files completes before writing any output.
	file1Band := filepath.Join(tempDir, "file1_band.go")
	if _, statErr := os.Stat(file1Band); statErr == nil {
		t.Errorf("file1_band.go must not exist after ProcessFiles returned an error; " +
			"graph validation of file2 must happen before file1 is written")
	}
}

type fileContent struct {
	content     string
	shouldWrite *bool // If nil, defaults to true
	name        string
}

func TestProcessFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		content              string
		expectedFunctionName string
		name                 string
		errorContains        string
		errorNotContains     string
		expectedGenerated    bool
		shouldError          bool
	}{
		{
			name: "valid kessoku code",
			content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	Value string
}

func NewConfig() *Config {
	return &Config{Value: "test"}
}

type Service struct {
	config *Config
}

func NewService(config *Config) *Service {
	return &Service{config: config}
}

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewService),
)
`,
			expectedGenerated:    true,
			expectedFunctionName: "InitializeService",
			shouldError:          false,
		},
		{
			name: "no kessoku code",
			content: `package main

type Service struct {
	Value string
}

func main() {
	// no kessoku
}
`,
			expectedGenerated: false,
			shouldError:       false,
		},
		{
			name: "invalid syntax",
			content: `package main

import "github.com/mazrean/kessoku"

func invalid syntax here {
`,
			expectedGenerated: false,
			shouldError:       true,
			errorContains:     "parse file",
			errorNotContains:  "parse file test.go: parse file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.go")

			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			processor := NewProcessor()
			err := processor.ProcessFiles([]string{testFile})

			if tt.shouldError {
				if err == nil {
					t.Fatal("Expected ProcessFiles to fail")
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				if tt.errorNotContains != "" && containsString(err.Error(), tt.errorNotContains) {
					t.Errorf("Expected error to not contain %q, got %q", tt.errorNotContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("ProcessFiles failed: %v", err)
			}

			generatedFile := filepath.Join(tempDir, "test_band.go")

			if tt.expectedGenerated {
				if _, err := os.Stat(generatedFile); os.IsNotExist(err) {
					t.Fatal("Expected generated file to be created")
				}

				// Read and verify generated content
				generated, err := os.ReadFile(generatedFile)
				if err != nil {
					t.Fatalf("Failed to read generated file: %v", err)
				}

				generatedStr := string(generated)

				expectedContent := []string{
					"// Code generated by kessoku. DO NOT EDIT.",
					"package main",
				}

				if tt.expectedFunctionName != "" {
					expectedContent = append(expectedContent, "func "+tt.expectedFunctionName+"(")
				}

				for _, expected := range expectedContent {
					if !strings.Contains(generatedStr, expected) {
						t.Errorf("Expected generated file to contain %q, got:\n%s", expected, generatedStr)
					}
				}
			} else {
				if _, err := os.Stat(generatedFile); err == nil {
					t.Error("Expected no generated file to be created")
				}
			}
		})
	}
}
