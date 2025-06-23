package kessoku

import (
	"go/types"
	"testing"
)

func createTestTypes() (configType, serviceType, intType types.Type) {
	configType = types.NewPointer(types.NewNamed(
		types.NewTypeName(0, types.NewPackage("test", "test"), "Config", nil),
		types.NewStruct(nil, nil),
		nil,
	))
	
	serviceType = types.NewPointer(types.NewNamed(
		types.NewTypeName(0, types.NewPackage("test", "test"), "Service", nil),
		types.NewStruct(nil, nil),
		nil,
	))
	
	intType = types.Typ[types.Int]
	
	return configType, serviceType, intType
}

func TestNewGraph(t *testing.T) {
	t.Parallel()
	
	configType, serviceType, intType := createTestTypes()
	
	tests := []struct {
		name            string
		build           *BuildDirective
		expectError     bool
		expectedName    string
		errorContains   string
	}{
		{
			name: "basic dependency graph",
			build: &BuildDirective{
				InjectorName: "InitializeService",
				Arguments:    nil,
				Return: &Return{
					Type: serviceType,
				},
				Providers: []*ProviderSpec{
					{
						Type:          ProviderTypeFunction,
						Provides:      []types.Type{configType},
						Requires:      []types.Type{},
						IsReturnError: false,
					},
					{
						Type:          ProviderTypeFunction,
						Provides:      []types.Type{serviceType},
						Requires:      []types.Type{configType},
						IsReturnError: false,
					},
				},
			},
			expectError:  false,
			expectedName: "InitializeService",
		},
		{
			name: "with argument",
			build: &BuildDirective{
				InjectorName: "InitializeService",
				Arguments: []*Argument{
					{
						Name: "value",
						Type: intType,
					},
				},
				Return: &Return{
					Type: serviceType,
				},
				Providers: []*ProviderSpec{
					{
						Type:          ProviderTypeFunction,
						Provides:      []types.Type{serviceType},
						Requires:      []types.Type{intType},
						IsReturnError: false,
					},
				},
			},
			expectError:  false,
			expectedName: "InitializeService",
		},
		{
			name: "missing provider",
			build: &BuildDirective{
				InjectorName: "InitializeService",
				Arguments:    nil,
				Return: &Return{
					Type: serviceType,
				},
				Providers: []*ProviderSpec{
					{
						Type:          ProviderTypeFunction,
						Provides:      []types.Type{serviceType},
						Requires:      []types.Type{configType}, // Config provider is missing
						IsReturnError: false,
					},
				},
			},
			expectError:   true,
			errorContains: "no provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			graph, err := NewGraph(tt.build)
			
			if tt.expectError {
				if err == nil {
					t.Fatal("Expected NewGraph to fail")
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}
			
			if err != nil {
				t.Fatalf("NewGraph failed: %v", err)
			}
			
			if graph == nil {
				t.Fatal("Expected graph to be created")
			}
			
			if graph.injectorName != tt.expectedName {
				t.Errorf("Expected injector name %q, got %q", tt.expectedName, graph.injectorName)
			}
			
			if graph.returnValue == nil {
				t.Fatal("Expected return value to be set")
			}
		})
	}
}

func TestGraphBuild(t *testing.T) {
	t.Parallel()
	
	configType, serviceType, _ := createTestTypes()
	
	tests := []struct {
		name             string
		graph            *Graph
		expectedName     string
		expectedStmts    int
		expectError      bool
	}{
		{
			name: "basic injector build",
			expectedName: "InitializeService",
			expectedStmts: 2,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			// Create a test graph
			build := &BuildDirective{
				InjectorName: tt.expectedName,
				Arguments:    nil,
				Return: &Return{
					Type: serviceType,
				},
				Providers: []*ProviderSpec{
					{
						Type:          ProviderTypeFunction,
						Provides:      []types.Type{configType},
						Requires:      []types.Type{},
						IsReturnError: false,
					},
					{
						Type:          ProviderTypeFunction,
						Provides:      []types.Type{serviceType},
						Requires:      []types.Type{configType},
						IsReturnError: false,
					},
				},
			}
			
			graph, err := NewGraph(build)
			if err != nil {
				t.Fatalf("Failed to create graph: %v", err)
			}
			
			injector, err := graph.Build()
			
			if tt.expectError {
				if err == nil {
					t.Fatal("Expected Build to fail")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			
			if injector == nil {
				t.Fatal("Expected injector to be created")
			}
			
			if injector.Name != tt.expectedName {
				t.Errorf("Expected injector name %q, got %q", tt.expectedName, injector.Name)
			}
			
			if len(injector.Stmts) != tt.expectedStmts {
				t.Errorf("Expected %d statements, got %d", tt.expectedStmts, len(injector.Stmts))
			}
			
			if injector.Return == nil {
				t.Fatal("Expected injector to have return")
			}
		})
	}
}

func TestCreateInjector(t *testing.T) {
	t.Parallel()
	
	configType, serviceType, _ := createTestTypes()
	
	tests := []struct {
		name         string
		metadata     *MetaData
		build        *BuildDirective
		expectedName string
		expectError  bool
	}{
		{
			name: "successful creation",
			metadata: &MetaData{
				Package: "test",
				Imports: nil,
			},
			build: &BuildDirective{
				InjectorName: "InitializeService",
				Arguments:    nil,
				Return: &Return{
					Type: serviceType,
				},
				Providers: []*ProviderSpec{
					{
						Type:          ProviderTypeFunction,
						Provides:      []types.Type{configType},
						Requires:      []types.Type{},
						IsReturnError: false,
					},
					{
						Type:          ProviderTypeFunction,
						Provides:      []types.Type{serviceType},
						Requires:      []types.Type{configType},
						IsReturnError: false,
					},
				},
			},
			expectedName: "InitializeService",
			expectError:  false,
		},
		{
			name: "missing provider error",
			metadata: &MetaData{
				Package: "test",
				Imports: nil,
			},
			build: &BuildDirective{
				InjectorName: "InitializeService",
				Arguments:    nil,
				Return: &Return{
					Type: serviceType,
				},
				Providers: []*ProviderSpec{
					{
						Type:          ProviderTypeFunction,
						Provides:      []types.Type{serviceType},
						Requires:      []types.Type{configType}, // Missing provider
						IsReturnError: false,
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			injector, err := CreateInjector(tt.metadata, tt.build)
			
			if tt.expectError {
				if err == nil {
					t.Fatal("Expected CreateInjector to fail")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("CreateInjector failed: %v", err)
			}
			
			if injector == nil {
				t.Fatal("Expected injector to be created")
			}
			
			if injector.Name != tt.expectedName {
				t.Errorf("Expected injector name %q, got %q", tt.expectedName, injector.Name)
			}
		})
	}
}

func TestNewGraphWithCircularDependency(t *testing.T) {
	// Note: Current implementation doesn't have explicit cycle detection
	// This test is skipped for now as cycle detection would require 
	// more sophisticated graph analysis
	t.Skip("Cycle detection not implemented yet")
}