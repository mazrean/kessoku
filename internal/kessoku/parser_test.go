package kessoku

import (
	"os"
	"path/filepath"
	"testing"
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
		content              string
		expectedInjectorName string
		expectedProviderType string
		errorContains        string
		name                 string
		expectedBuilds       int
		expectedProviders    int
		expectedArgs         int
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
			name: "dependency with missing provider",
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
	kessoku.Provide(NewService),
)
`,
			expectedBuilds:       1,
			expectedProviders:    1,
			expectedArgs:         0, // Parser doesn't auto-detect yet - that happens in graph phase
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
		{
			name: "kessoku inline Set call",
			content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	Value string
}

func NewConfig() *Config {
	return &Config{Value: "test"}
}

type Database struct {
	config *Config
}

func NewDatabase(config *Config) (*Database, error) {
	return &Database{config: config}, nil
}

type Service struct {
	db *Database
}

func NewService(db *Database) *Service {
	return &Service{db: db}
}

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Set(
		kessoku.Provide(NewConfig),
		kessoku.Provide(NewDatabase),
	),
	kessoku.Provide(NewService),
)
`,
			expectedBuilds:       1,
			expectedProviders:    3,
			expectedArgs:         0,
			expectedInjectorName: "InitializeService",
			shouldError:          false,
		},
		{
			name: "kessoku Set variable",
			content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	Value string
}

func NewConfig() *Config {
	return &Config{Value: "test"}
}

type Database struct {
	config *Config
}

func NewDatabase(config *Config) (*Database, error) {
	return &Database{config: config}, nil
}

type Service struct {
	db *Database
}

func NewService(db *Database) *Service {
	return &Service{db: db}
}

var DatabaseSet = kessoku.Set(
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewDatabase),
)

var _ = kessoku.Inject[*Service](
	"InitializeService",
	DatabaseSet,
	kessoku.Provide(NewService),
)
`,
			expectedBuilds:       1,
			expectedProviders:    3,
			expectedArgs:         0,
			expectedInjectorName: "InitializeService",
			shouldError:          false,
		},
		{
			name: "kessoku nested Set variables",
			content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	Value string
}

func NewConfig() *Config {
	return &Config{Value: "test"}
}

type Database struct {
	config *Config
}

func NewDatabase(config *Config) (*Database, error) {
	return &Database{config: config}, nil
}

type UserService struct {
	db *Database
}

func NewUserService(db *Database) *UserService {
	return &UserService{db: db}
}

type App struct {
	service *UserService
}

func NewApp(service *UserService) *App {
	return &App{service: service}
}

var DatabaseSet = kessoku.Set(
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewDatabase),
)

var ServiceSet = kessoku.Set(
	DatabaseSet,
	kessoku.Provide(NewUserService),
)

var _ = kessoku.Inject[*App](
	"InitializeApp",
	ServiceSet,
	kessoku.Provide(NewApp),
)
`,
			expectedBuilds:       1,
			expectedProviders:    4,
			expectedArgs:         0,
			expectedInjectorName: "InitializeApp",
			shouldError:          false,
		},
		{
			name: "kessoku multiple injectors with Set variables",
			content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	Value string
}

func NewConfig() *Config {
	return &Config{Value: "test"}
}

type Database struct {
	config *Config
}

func NewDatabase(config *Config) (*Database, error) {
	return &Database{config: config}, nil
}

type Service struct {
	db *Database
}

func NewService(db *Database) *Service {
	return &Service{db: db}
}

type App struct {
	service *Service
}

func NewApp(service *Service) *App {
	return &App{service: service}
}

var DatabaseSet = kessoku.Set(
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewDatabase),
)

var _ = kessoku.Inject[*Service](
	"InitializeService",
	DatabaseSet,
	kessoku.Provide(NewService),
)

var _ = kessoku.Inject[*App](
	"InitializeApp",
	DatabaseSet,
	kessoku.Provide(NewService),
	kessoku.Provide(NewApp),
)
`,
			expectedBuilds:       2,
			expectedProviders:    3, // Check first injector
			expectedArgs:         0,
			expectedInjectorName: "InitializeService",
			shouldError:          false,
		},
		{
			name: "bind provider should create multiple type bindings",
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
			shouldError:          false,
		},
	}

	// Additional test for edge cases related to Set variable parsing
	setVariableEdgeCases := []struct {
		content         string
		name            string
		expectedBuilds  int
		shouldHaveError bool
	}{
		{
			name: "undefined Set variable graceful handling",
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
	UndefinedSet,
	kessoku.Provide(NewService),
)
`,
			expectedBuilds:  0,     // Should skip this injector due to parse error
			shouldHaveError: false, // Parser should not fail completely, just skip the injector
		},
	}

	// Run edge case tests
	for _, tt := range setVariableEdgeCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.go")

			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			parser := NewParser()
			metadata, builds, err := parser.ParseFile(testFile, NewVarPool())

			if tt.shouldHaveError {
				if err == nil {
					t.Fatal("Expected ParseFile to fail")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			if tt.expectedBuilds == 0 {
				if metadata != nil && len(builds) > 0 {
					t.Errorf("Expected no builds due to parse errors, got %d", len(builds))
				}
				return
			}

			if len(builds) != tt.expectedBuilds {
				t.Fatalf("Expected %d build directives, got %d", tt.expectedBuilds, len(builds))
			}
		})
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
			metadata, builds, err := parser.ParseFile(testFile, NewVarPool())

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

			// Arguments field no longer exists in BuildDirective
			_ = tt.expectedArgs // Suppress unused variable warning

			if build.Return == nil {
				t.Fatal("Expected return type to be set")
			}

			// Check specific provider type if specified
			if tt.expectedProviderType != "" {
				var foundProvider *ProviderSpec
				for _, provider := range build.Providers {
					if len(provider.Provides) > 0 {
						// Check all provided types, not just the first one
						for _, typeGroup := range provider.Provides {
							for _, providedType := range typeGroup {
								if providedType.String() == tt.expectedProviderType {
									foundProvider = provider
									break
								}
							}
							if foundProvider != nil {
								break
							}
						}
						if foundProvider != nil {
							break
						}
					}
				}

				if foundProvider == nil {
					t.Errorf("Expected to find provider for type %q", tt.expectedProviderType)
				}
			}
		})
	}
}

func TestParseBindProviderMultipleTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		content               string
		name                  string
		expectedConcreteType  string
		expectedInterfaceType string
		expectedBuilds        int
		expectedProviders     int
		shouldError           bool
	}{
		{
			name: "bind provider should provide both concrete and interface types",
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

type ConcreteService struct {
	impl *ConcreteImpl
}

func NewConcreteService(impl *ConcreteImpl) *ConcreteService {
	return &ConcreteService{impl: impl}
}

var _ = kessoku.Inject[*ConcreteService](
	"InitializeConcreteService",
	kessoku.Bind[Interface](kessoku.Provide(NewConcreteImpl)),
	kessoku.Provide(NewConcreteService),
)
`,
			expectedBuilds:        1,
			expectedProviders:     2,
			expectedConcreteType:  "*command-line-arguments.ConcreteImpl",
			expectedInterfaceType: "command-line-arguments.Interface",
			shouldError:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.go")
			varPool := NewVarPool()

			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			parser := NewParser()
			metadata, builds, err := parser.ParseFile(testFile, varPool)

			if tt.shouldError {
				if err == nil {
					t.Fatal("Expected ParseFile to fail")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			if len(builds) != tt.expectedBuilds {
				t.Fatalf("Expected %d build directives, got %d", tt.expectedBuilds, len(builds))
			}

			if metadata == nil {
				t.Fatal("Expected metadata to be returned")
			}

			build := builds[0]
			if len(build.Providers) != tt.expectedProviders {
				t.Errorf("Expected %d providers, got %d", tt.expectedProviders, len(build.Providers))
			}

			// Find the bind provider
			var bindProvider *ProviderSpec
			for _, provider := range build.Providers {
				if len(provider.Provides) > 0 {
					// Check if this provider provides the interface type
					for _, typeGroup := range provider.Provides {
						for _, providedType := range typeGroup {
							if providedType.String() == tt.expectedInterfaceType {
								bindProvider = provider
								break
							}
						}
						if bindProvider != nil {
							break
						}
					}
					if bindProvider != nil {
						break
					}
				}
			}

			if bindProvider == nil {
				t.Errorf("Expected to find bind provider for interface type %q", tt.expectedInterfaceType)
			} else {
				// TODO: This test will fail initially because the current implementation only
				// provides the interface type, not both types. After implementing the feature,
				// this test should verify that bindProvider.Provides contains both types.
				t.Logf("Current bind provider provides: %v", bindProvider.Provides)

				// This is what we want to achieve: bindProvider should provide both types
				// Count total types across all groups
				totalTypes := 0
				for _, typeGroup := range bindProvider.Provides {
					totalTypes += len(typeGroup)
				}
				if totalTypes != 2 {
					t.Errorf("Expected bind provider to provide 2 types (concrete and interface), got %d", totalTypes)
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

func TestParseProviderArgument_SetVariables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		content           string
		name              string
		expectedProviders int
		shouldError       bool
	}{
		{
			name: "Set variable resolution",
			content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	Value string
}

func NewConfig() *Config {
	return &Config{Value: "test"}
}

type Database struct {
	config *Config
}

func NewDatabase(config *Config) (*Database, error) {
	return &Database{config: config}, nil
}

var DatabaseSet = kessoku.Set(
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewDatabase),
)

type Service struct {
	db *Database
}

func NewService(db *Database) *Service {
	return &Service{db: db}
}

var _ = kessoku.Inject[*Service](
	"InitializeService",
	DatabaseSet,
	kessoku.Provide(NewService),
)
`,
			expectedProviders: 3,
			shouldError:       false,
		},
		{
			name: "Nested Set variables",
			content: `package main

import "github.com/mazrean/kessoku"

type Config struct{}
func NewConfig() *Config { return &Config{} }

type Database struct{}
func NewDatabase(config *Config) *Database { return &Database{} }

type UserService struct{}
func NewUserService(db *Database) *UserService { return &UserService{} }

type App struct{}
func NewApp(service *UserService) *App { return &App{} }

var DatabaseSet = kessoku.Set(
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewDatabase),
)

var ServiceSet = kessoku.Set(
	DatabaseSet,
	kessoku.Provide(NewUserService),
)

var _ = kessoku.Inject[*App](
	"InitializeApp",
	ServiceSet,
	kessoku.Provide(NewApp),
)
`,
			expectedProviders: 4,
			shouldError:       false,
		},
		{
			name: "Mixed inline and variable Sets",
			content: `package main

import "github.com/mazrean/kessoku"

type Config struct{}
func NewConfig() *Config { return &Config{} }

type Database struct{}
func NewDatabase(config *Config) *Database { return &Database{} }

type Cache struct{}
func NewCache() *Cache { return &Cache{} }

type Service struct{}
func NewService(db *Database, cache *Cache) *Service { return &Service{} }

var DatabaseSet = kessoku.Set(
	kessoku.Provide(NewConfig),
	kessoku.Provide(NewDatabase),
)

var _ = kessoku.Inject[*Service](
	"InitializeService",
	DatabaseSet,
	kessoku.Set(
		kessoku.Provide(NewCache),
	),
	kessoku.Provide(NewService),
)
`,
			expectedProviders: 4,
			shouldError:       false,
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
			metadata, builds, err := parser.ParseFile(testFile, NewVarPool())

			if tt.shouldError {
				if err == nil {
					t.Fatal("Expected ParseFile to fail")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			if metadata == nil {
				t.Fatal("Expected metadata to be returned")
			}

			if len(builds) != 1 {
				t.Fatalf("Expected 1 build directive, got %d", len(builds))
			}

			build := builds[0]
			if len(build.Providers) != tt.expectedProviders {
				t.Errorf("Expected %d providers, got %d", tt.expectedProviders, len(build.Providers))

				// Log provider details for debugging
				t.Logf("Found providers:")
				for i, provider := range build.Providers {
					t.Logf("  %d: Provides %v", i, provider.Provides)
				}
			}

			// Verify all providers have valid type information
			for i, provider := range build.Providers {
				if len(provider.Provides) == 0 {
					t.Errorf("Provider %d has no provides types", i)
				}
				for j, providedType := range provider.Provides {
					if providedType == nil {
						t.Errorf("Provider %d provides type %d is nil", i, j)
					}
				}
			}
		})
	}
}

// Tests for User Story 1 - Struct Provider Parsing
func TestParseStructProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		content              string
		name                 string
		expectedInjectorName string
		errorContains        string
		expectedProviders    int
		expectedBuilds       int
		shouldError          bool
	}{
		{
			name: "basic struct provider parsing",
			content: `package main

import "github.com/mazrean/kessoku"

type Config struct {
	DBHost string
	DBPort int
}

func NewConfig() *Config {
	return &Config{DBHost: "localhost", DBPort: 5432}
}

type App struct {
	host string
	port int
}

func NewApp(host string, port int) *App {
	return &App{host: host, port: port}
}

var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewConfig),
	kessoku.Struct[*Config](),
	kessoku.Provide(NewApp),
)
`,
			expectedBuilds:       1,
			expectedProviders:    3, // NewConfig, Struct[*Config], NewApp
			expectedInjectorName: "InitializeApp",
			shouldError:          false,
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
			_, builds, err := parser.ParseFile(testFile, NewVarPool())

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

			// Verify struct provider was parsed
			hasStructProvider := false
			for _, provider := range build.Providers {
				if provider.Type == ProviderTypeStruct {
					hasStructProvider = true
					if provider.StructType == nil {
						t.Error("Struct provider should have StructType set")
					}
					if len(provider.StructFields) == 0 {
						t.Error("Struct provider should have StructFields populated")
					}
				}
			}
			if !hasStructProvider {
				t.Error("Expected to find a struct provider")
			}
		})
	}
}

func TestParseStructProviderNonStructError(t *testing.T) {
	t.Parallel()

	content := `package main

import "github.com/mazrean/kessoku"

type App struct{}

func NewApp() *App {
	return &App{}
}

var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Struct[string](),
	kessoku.Provide(NewApp),
)
`

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	_, builds, err := parser.ParseFile(testFile, NewVarPool())

	// Either parse error or no builds due to error handling
	if err == nil && len(builds) > 0 {
		// If it parsed successfully, check if there was a warning logged
		t.Log("Parser handled non-struct type gracefully")
	}
}

func TestExtractExportedFields(t *testing.T) {
	t.Parallel()

	content := `package main

import "github.com/mazrean/kessoku"

type Config struct {
	DBHost   string
	DBPort   int
	password string // unexported - should be filtered
	Debug    bool
}

func NewConfig() *Config {
	return &Config{}
}

type App struct{}

func NewApp(host string, port int, debug bool) *App {
	return &App{}
}

var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewConfig),
	kessoku.Struct[*Config](),
	kessoku.Provide(NewApp),
)
`

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	_, builds, err := parser.ParseFile(testFile, NewVarPool())

	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(builds) != 1 {
		t.Fatalf("Expected 1 build directive, got %d", len(builds))
	}

	// Find struct provider
	var structProvider *ProviderSpec
	for _, provider := range builds[0].Providers {
		if provider.Type == ProviderTypeStruct {
			structProvider = provider
			break
		}
	}

	if structProvider == nil {
		t.Fatal("Expected to find struct provider")
	}

	// Should have 3 exported fields (DBHost, DBPort, Debug), not 4 (password is unexported)
	if len(structProvider.StructFields) != 3 {
		t.Errorf("Expected 3 exported fields, got %d", len(structProvider.StructFields))
		for _, field := range structProvider.StructFields {
			t.Logf("  Field: %s (%s)", field.Name, field.Type)
		}
	}
}

func TestExtractExportedFieldsUnexported(t *testing.T) {
	t.Parallel()

	content := `package main

import "github.com/mazrean/kessoku"

type Config struct {
	host     string
	port     int
	password string
}

func NewConfig() *Config {
	return &Config{}
}

type App struct{}

func NewApp() *App {
	return &App{}
}

var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewConfig),
	kessoku.Struct[*Config](),
	kessoku.Provide(NewApp),
)
`

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	_, builds, err := parser.ParseFile(testFile, NewVarPool())

	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(builds) != 1 {
		t.Fatalf("Expected 1 build directive, got %d", len(builds))
	}

	// Find struct provider
	var structProvider *ProviderSpec
	for _, provider := range builds[0].Providers {
		if provider.Type == ProviderTypeStruct {
			structProvider = provider
			break
		}
	}

	if structProvider == nil {
		t.Fatal("Expected to find struct provider")
	}

	// Should have 0 exported fields (all are unexported)
	if len(structProvider.StructFields) != 0 {
		t.Errorf("Expected 0 exported fields, got %d", len(structProvider.StructFields))
	}
}

func TestExtractExportedFieldsAlphabeticalOrder(t *testing.T) {
	t.Parallel()

	content := `package main

import "github.com/mazrean/kessoku"

type Config struct {
	Zebra int
	Apple string
	Mango bool
}

func NewConfig() *Config {
	return &Config{}
}

type App struct{}

func NewApp(a string, m bool, z int) *App {
	return &App{}
}

var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Provide(NewConfig),
	kessoku.Struct[*Config](),
	kessoku.Provide(NewApp),
)
`

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	_, builds, err := parser.ParseFile(testFile, NewVarPool())

	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Find struct provider
	var structProvider *ProviderSpec
	for _, provider := range builds[0].Providers {
		if provider.Type == ProviderTypeStruct {
			structProvider = provider
			break
		}
	}

	if structProvider == nil {
		t.Fatal("Expected to find struct provider")
	}

	// Fields should be in alphabetical order: Apple, Mango, Zebra
	expectedOrder := []string{"Apple", "Mango", "Zebra"}
	if len(structProvider.StructFields) != 3 {
		t.Fatalf("Expected 3 fields, got %d", len(structProvider.StructFields))
	}

	for i, expected := range expectedOrder {
		if structProvider.StructFields[i].Name != expected {
			t.Errorf("Expected field %d to be %q, got %q", i, expected, structProvider.StructFields[i].Name)
		}
	}
}

func TestParseBindProviderInterfaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		content                       string
		name                          string
		expectedBuilds                int
		expectedProviders             int
		expectedConcreteTypeProvided  bool
		expectedInterfaceTypeProvided bool
		shouldError                   bool
	}{
		{
			name: "bind provider should provide both concrete and interface types for dependency resolution",
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

type ServiceNeedsInterface struct {
	impl Interface
}

func NewServiceNeedsInterface(impl Interface) *ServiceNeedsInterface {
	return &ServiceNeedsInterface{impl: impl}
}

type ServiceNeedsConcrete struct {
	impl *ConcreteImpl
}

func NewServiceNeedsConcrete(impl *ConcreteImpl) *ServiceNeedsConcrete {
	return &ServiceNeedsConcrete{impl: impl}
}

type App struct {
	interfaceService *ServiceNeedsInterface
	concreteService  *ServiceNeedsConcrete
}

func NewApp(interfaceService *ServiceNeedsInterface, concreteService *ServiceNeedsConcrete) *App {
	return &App{
		interfaceService: interfaceService,
		concreteService:  concreteService,
	}
}

var _ = kessoku.Inject[*App](
	"InitializeApp",
	kessoku.Bind[Interface](kessoku.Provide(NewConcreteImpl)),
	kessoku.Provide(NewServiceNeedsInterface),
	kessoku.Provide(NewServiceNeedsConcrete),
	kessoku.Provide(NewApp),
)
`,
			expectedBuilds:                1,
			expectedProviders:             4,
			expectedConcreteTypeProvided:  true, // This is what we want to achieve
			expectedInterfaceTypeProvided: true, // This is what we want to achieve
			shouldError:                   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.go")
			varPool := NewVarPool()

			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			parser := NewParser()
			metadata, builds, err := parser.ParseFile(testFile, varPool)

			if tt.shouldError {
				if err == nil {
					t.Fatal("Expected ParseFile to fail")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			if len(builds) != tt.expectedBuilds {
				t.Fatalf("Expected %d build directives, got %d", tt.expectedBuilds, len(builds))
			}

			if metadata == nil {
				t.Fatal("Expected metadata to be returned")
			}

			build := builds[0]
			if len(build.Providers) != tt.expectedProviders {
				t.Errorf("Expected %d providers, got %d", tt.expectedProviders, len(build.Providers))
			}

			// Find the bind provider
			var bindProvider *ProviderSpec
			for _, provider := range build.Providers {
				if len(provider.Provides) > 0 {
					for _, typeGroup := range provider.Provides {
						for _, providedType := range typeGroup {
							if providedType.String() == "command-line-arguments.Interface" {
								bindProvider = provider
								break
							}
						}
						if bindProvider != nil {
							break
						}
					}
				}
			}

			if bindProvider == nil {
				t.Fatal("Expected to find bind provider")
			}

			// Check if concrete type is provided
			concreteTypeProvided := false
			interfaceTypeProvided := false
			for _, typeGroup := range bindProvider.Provides {
				for _, providedType := range typeGroup {
					switch providedType.String() {
					case "*command-line-arguments.ConcreteImpl":
						concreteTypeProvided = true
					case "command-line-arguments.Interface":
						interfaceTypeProvided = true
					}
				}
			}

			if tt.expectedConcreteTypeProvided && !concreteTypeProvided {
				t.Errorf("Expected bind provider to provide concrete type, but it doesn't")
			}

			if tt.expectedInterfaceTypeProvided && !interfaceTypeProvided {
				t.Errorf("Expected bind provider to provide interface type, but it doesn't")
			}

			// TODO: After implementing the multi-type binding, this should pass
			// For now, this test documents the expected behavior
			t.Logf("Bind provider currently provides: %v", bindProvider.Provides)
		})
	}
}
