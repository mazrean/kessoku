package kessoku

import (
	"os"
	"path/filepath"
	"testing"
	"go/types"
)

func TestNewParser(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name string
	}{
		{
			name: "create new parser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parser := NewParser()
			
			if parser == nil {
				t.Fatal("Expected parser to be created")
			}
			
			if parser.fset == nil {
				t.Error("Expected file set to be initialized")
			}
			
			if parser.packages == nil {
				t.Error("Expected packages map to be initialized")
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name                string
		content             string
		expectedBuilds      int
		expectedProviders   int
		expectedArgs        int
		expectedInjectorName string
		expectedProviderType string
		shouldError         bool
		errorContains       string
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
			expectedBuilds:       1,
			expectedProviders:    2,
			expectedArgs:         0,
			expectedInjectorName: "InitializeService",
			shouldError:          false,
		},
		{
			name: "kessoku bind",
			content: `package main

import "github.com/mazrean/kessoku"

type Interface interface {
	DoSomething() string
}

type ConcreteImpl struct{}

func (c *ConcreteImpl) DoSomething() string {
	return "implementation"
}

func NewConcreteImpl() *ConcreteImpl {
	return &ConcreteImpl{}
}

type Service struct {
	impl Interface
}

func NewService(impl Interface) *Service {
	return &Service{impl: impl}
}

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Bind[Interface](kessoku.Provide(NewConcreteImpl)),
	kessoku.Provide(NewService),
)
`,
			expectedBuilds:       1,
			expectedProviders:    2,
			expectedArgs:         0,
			expectedInjectorName: "InitializeService",
			expectedProviderType: "command-line-arguments.Interface",
			shouldError:          false,
		},
		{
			name: "kessoku arg",
			content: `package main

import "github.com/mazrean/kessoku"

type Service struct {
	value int
}

func NewService(value int) *Service {
	return &Service{value: value}
}

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Arg[int]("value"),
	kessoku.Provide(NewService),
)
`,
			expectedBuilds:       1,
			expectedProviders:    1,
			expectedArgs:         1,
			expectedInjectorName: "InitializeService",
			shouldError:          false,
		},
		{
			name: "kessoku value",
			content: `package main

import "github.com/mazrean/kessoku"

type Service struct {
	value string
}

func NewService(value string) *Service {
	return &Service{value: value}
}

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Value("example value"),
	kessoku.Provide(NewService),
)
`,
			expectedBuilds:       1,
			expectedProviders:    2,
			expectedArgs:         0,
			expectedInjectorName: "InitializeService",
			shouldError:          false,
		},
		{
			name: "no kessoku import",
			content: `package main

type Service struct {
	value string
}

func main() {
	// no kessoku import
}
`,
			expectedBuilds: 0,
			shouldError:    false,
		},
		{
			name: "invalid syntax",
			content: `package main

import "github.com/mazrean/kessoku"

func invalid syntax here {
`,
			shouldError:   true,
			errorContains: "parse file",
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
			
			parser := NewParser()
			metadata, builds, err := parser.ParseFile(testFile)
			
			if tt.shouldError {
				if err == nil {
					t.Fatal("Expected ParseFile to fail")
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}
			
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}
			
			if tt.expectedBuilds == 0 {
				if metadata != nil {
					t.Error("Expected metadata to be nil")
				}
				if builds != nil {
					t.Error("Expected builds to be nil")
				}
				return
			}
			
			if metadata == nil {
				t.Fatal("Expected metadata to be returned")
			}
			
			if len(builds) != tt.expectedBuilds {
				t.Fatalf("Expected %d build directives, got %d", tt.expectedBuilds, len(builds))
			}
			
			build := builds[0]
			if build.InjectorName != tt.expectedInjectorName {
				t.Errorf("Expected injector name %q, got %q", tt.expectedInjectorName, build.InjectorName)
			}
			
			if len(build.Providers) != tt.expectedProviders {
				t.Errorf("Expected %d providers, got %d", tt.expectedProviders, len(build.Providers))
			}
			
			if len(build.Arguments) != tt.expectedArgs {
				t.Errorf("Expected %d arguments, got %d", tt.expectedArgs, len(build.Arguments))
			}
			
			if build.Return == nil {
				t.Fatal("Expected return type to be set")
			}
			
			// Check specific provider type if specified
			if tt.expectedProviderType != "" {
				var foundProvider *ProviderSpec
				for _, provider := range build.Providers {
					if len(provider.Provides) > 0 {
						typeName := provider.Provides[0].String()
						if typeName == tt.expectedProviderType {
							foundProvider = provider
							break
						}
					}
				}
				
				if foundProvider == nil {
					t.Errorf("Expected to find provider for type %q", tt.expectedProviderType)
				}
			}
			
			// Check argument type for arg test
			if tt.expectedArgs > 0 {
				arg := build.Arguments[0]
				if arg.Name != "value" {
					t.Errorf("Expected argument name 'value', got %q", arg.Name)
				}
				
				if arg.Type == nil {
					t.Fatal("Expected argument type to be set")
				}
				
				if !types.Identical(arg.Type, types.Typ[types.Int]) {
					t.Errorf("Expected argument type to be int, got %v", arg.Type)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}())
}