package kessoku

import (
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

func TestProcessFiles(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name                    string
		files                   []fileContent
		expectedGeneratedFiles  []string
		shouldError             bool
		errorContains           string
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
				if _, err := os.Stat(generatedPath); os.IsNotExist(err) {
					t.Errorf("Expected generated file %s to exist", expectedFile)
					continue
				}
				
				// Verify generated content
				generated, err := os.ReadFile(generatedPath)
				if err != nil {
					t.Fatalf("Failed to read generated file %s: %v", expectedFile, err)
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

type fileContent struct {
	name        string
	content     string
	shouldWrite *bool // If nil, defaults to true
}

func TestProcessFile(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name                   string
		content                string
		expectedGenerated      bool
		expectedFunctionName   string
		shouldError            bool
		errorContains          string
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