package kessoku

import (
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"strings"
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
		build         *BuildDirective
		name          string
		expectedName  string
		errorContains string
		expectError   bool
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
			name: "auto-detected argument for missing provider",
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
						Requires:      []types.Type{configType}, // Config provider is missing - should be auto-detected
						IsReturnError: false,
					},
				},
			},
			expectError:  false, // Should succeed and auto-detect the missing Config argument
			expectedName: "InitializeService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create test metadata
			metaData := &MetaData{
				Package: "test",
				Imports: make(map[string]*ast.ImportSpec),
			}

			graph, err := NewGraph(metaData, tt.build)

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
		name          string
		graph         *Graph
		expectedName  string
		expectedStmts int
		expectError   bool
	}{
		{
			name:          "basic injector build",
			expectedName:  "InitializeService",
			expectedStmts: 2,
			expectError:   false,
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

			// Create test metadata
			metaData := &MetaData{
				Package: "test",
				Imports: make(map[string]*ast.ImportSpec),
			}

			graph, err := NewGraph(metaData, build)
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
				Imports: make(map[string]*ast.ImportSpec),
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
			name: "auto-detected argument in CreateInjector",
			metadata: &MetaData{
				Package: "test",
				Imports: make(map[string]*ast.ImportSpec),
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
						Requires:      []types.Type{configType}, // Missing provider - should be auto-detected
						IsReturnError: false,
					},
				},
			},
			expectedName: "InitializeService", // Should succeed with auto-detected argument
			expectError:  false,
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

func TestCreateASTTypeExpr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		createType  func() types.Type
		name        string
		expectedAST string
		description string
	}{
		{
			name: "basic int type",
			createType: func() types.Type {
				return types.Typ[types.Int]
			},
			expectedAST: "int",
			description: "Basic integer type should be represented as 'int'",
		},
		{
			name: "basic string type",
			createType: func() types.Type {
				return types.Typ[types.String]
			},
			expectedAST: "string",
			description: "Basic string type should be represented as 'string'",
		},
		{
			name: "pointer to int",
			createType: func() types.Type {
				return types.NewPointer(types.Typ[types.Int])
			},
			expectedAST: "*int",
			description: "Pointer types should be represented with *",
		},
		{
			name: "slice of strings",
			createType: func() types.Type {
				return types.NewSlice(types.Typ[types.String])
			},
			expectedAST: "[]string",
			description: "Slice types should be represented as []T",
		},
		{
			name: "array of 10 ints",
			createType: func() types.Type {
				return types.NewArray(types.Typ[types.Int], 10)
			},
			expectedAST: "[10]int",
			description: "Array types should be represented as [N]T",
		},
		{
			name: "map from string to int",
			createType: func() types.Type {
				return types.NewMap(types.Typ[types.String], types.Typ[types.Int])
			},
			expectedAST: "map[string]int",
			description: "Map types should be represented as map[K]V",
		},
		{
			name: "empty interface",
			createType: func() types.Type {
				return types.NewInterfaceType(nil, nil)
			},
			expectedAST: "interface {\n}",
			description: "Empty interface should be represented as proper AST",
		},
		{
			name: "non-empty interface",
			createType: func() types.Type {
				// Create an interface with a method
				method := types.NewFunc(0, nil, "String", types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.String])), false))
				return types.NewInterfaceType([]*types.Func{method}, nil)
			},
			expectedAST: "interface {\n\tString() (result0 string)\n}",
			description: "Non-empty interface should generate proper AST with methods",
		},
		{
			name: "named type in main package",
			createType: func() types.Type {
				pkg := types.NewPackage("main", "main")
				obj := types.NewTypeName(0, pkg, "MyType", nil)
				return types.NewNamed(obj, types.Typ[types.String], nil)
			},
			expectedAST: "MyType",
			description: "Named types in main package should use simple name",
		},
		{
			name: "named type from other package",
			createType: func() types.Type {
				pkg := types.NewPackage("fmt", "fmt")
				obj := types.NewTypeName(0, pkg, "Stringer", nil)
				return types.NewNamed(obj, types.Typ[types.String], nil)
			},
			expectedAST: "fmt.Stringer",
			description: "Named types from other packages should use package.Name format",
		},
		{
			name: "pointer to named type from other package",
			createType: func() types.Type {
				pkg := types.NewPackage("context", "context")
				obj := types.NewTypeName(0, pkg, "Context", nil)
				namedType := types.NewNamed(obj, types.Typ[types.String], nil)
				return types.NewPointer(namedType)
			},
			expectedAST: "*context.Context",
			description: "Pointer to named type from other package",
		},
		{
			name: "slice of named types from other package",
			createType: func() types.Type {
				pkg := types.NewPackage("errors", "errors")
				obj := types.NewTypeName(0, pkg, "Error", nil)
				namedType := types.NewNamed(obj, types.Typ[types.String], nil)
				return types.NewSlice(namedType)
			},
			expectedAST: "[]errors.Error",
			description: "Slice of named types from other package",
		},
		{
			name: "map with named key and value types",
			createType: func() types.Type {
				stringPkg := types.NewPackage("strings", "strings")
				stringObj := types.NewTypeName(0, stringPkg, "Builder", nil)
				stringType := types.NewNamed(stringObj, types.Typ[types.String], nil)

				contextPkg := types.NewPackage("context", "context")
				contextObj := types.NewTypeName(0, contextPkg, "Context", nil)
				contextType := types.NewNamed(contextObj, types.Typ[types.String], nil)

				return types.NewMap(stringType, contextType)
			},
			expectedAST: "map[strings.Builder]context.Context",
			description: "Map with named types from different packages",
		},
		{
			name: "bidirectional channel",
			createType: func() types.Type {
				return types.NewChan(types.SendRecv, types.Typ[types.Int])
			},
			expectedAST: "chan int",
			description: "Bidirectional channel should be represented as chan T",
		},
		{
			name: "send-only channel",
			createType: func() types.Type {
				return types.NewChan(types.SendOnly, types.Typ[types.String])
			},
			expectedAST: "chan<- string",
			description: "Send-only channel should be represented as chan<- T",
		},
		{
			name: "receive-only channel",
			createType: func() types.Type {
				return types.NewChan(types.RecvOnly, types.Typ[types.Bool])
			},
			expectedAST: "<-chan bool",
			description: "Receive-only channel should be represented as <-chan T",
		},
		{
			name: "channel of named type",
			createType: func() types.Type {
				pkg := types.NewPackage("sync", "sync")
				obj := types.NewTypeName(0, pkg, "Mutex", nil)
				namedType := types.NewNamed(obj, types.Typ[types.String], nil)
				return types.NewChan(types.SendRecv, namedType)
			},
			expectedAST: "chan sync.Mutex",
			description: "Channel of named type from other package",
		},
		{
			name: "function signature",
			createType: func() types.Type {
				// func(int, string) bool
				params := types.NewTuple(
					types.NewVar(0, nil, "", types.Typ[types.Int]),
					types.NewVar(0, nil, "", types.Typ[types.String]),
				)
				results := types.NewTuple(
					types.NewVar(0, nil, "", types.Typ[types.Bool]),
				)
				return types.NewSignatureType(nil, nil, nil, params, results, false)
			},
			expectedAST: "func(arg0 int, arg1 string) (result0 bool)",
			description: "Function signatures should generate proper AST with parameters and results",
		},
		{
			name: "complex nested type",
			createType: func() types.Type {
				// *[]map[string]*context.Context
				contextPkg := types.NewPackage("context", "context")
				contextObj := types.NewTypeName(0, contextPkg, "Context", nil)
				contextType := types.NewNamed(contextObj, types.Typ[types.String], nil)
				pointerToContext := types.NewPointer(contextType)
				mapType := types.NewMap(types.Typ[types.String], pointerToContext)
				sliceType := types.NewSlice(mapType)
				return types.NewPointer(sliceType)
			},
			expectedAST: "*[]map[string]*context.Context",
			description: "Complex nested types should be properly handled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create the type
			typ := tt.createType()

			// Generate AST expression
			expr, _ := createASTTypeExpr(typ)

			// Convert AST expression back to string for comparison
			actualAST := exprToString(expr)

			if actualAST != tt.expectedAST {
				t.Errorf("createASTTypeExprWithImports() for %s:\n  Expected: %q\n  Actual:   %q\n  Description: %s",
					tt.name, tt.expectedAST, actualAST, tt.description)
			}

			// Additional validation: make sure the expression is valid AST
			if expr == nil {
				t.Errorf("createASTTypeExprWithImports() returned nil for %s", tt.name)
			}
		})
	}
}

func TestCreateASTTypeExprEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		createType  func() types.Type
		name        string
		description string
		expectValid bool
	}{
		{
			name: "nil package in named type",
			createType: func() types.Type {
				// Create a named type with nil package (should not happen in practice)
				obj := types.NewTypeName(0, nil, "LocalType", nil)
				return types.NewNamed(obj, types.Typ[types.String], nil)
			},
			expectValid: true,
			description: "Named type with nil package should be handled gracefully",
		},
		{
			name: "deeply nested pointers",
			createType: func() types.Type {
				// ***int
				t1 := types.NewPointer(types.Typ[types.Int])
				t2 := types.NewPointer(t1)
				return types.NewPointer(t2)
			},
			expectValid: true,
			description: "Deeply nested pointers should be handled",
		},
		{
			name: "slice of arrays",
			createType: func() types.Type {
				// [][5]string
				arrayType := types.NewArray(types.Typ[types.String], 5)
				return types.NewSlice(arrayType)
			},
			expectValid: true,
			description: "Slice of arrays should be handled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			typ := tt.createType()
			expr, _ := createASTTypeExpr(typ)

			if tt.expectValid {
				if expr == nil {
					t.Errorf("Expected valid AST expression for %s, got nil", tt.name)
				}
			} else {
				if expr != nil {
					t.Errorf("Expected nil AST expression for %s, got %v", tt.name, exprToString(expr))
				}
			}
		})
	}
}

// exprToString converts an ast.Expr to its string representation
func exprToString(expr ast.Expr) string {
	if expr == nil {
		return "<nil>"
	}

	var buf strings.Builder
	fset := token.NewFileSet()
	if err := format.Node(&buf, fset, expr); err != nil {
		return "<error formatting AST>"
	}
	return buf.String()
}

func TestCreateASTTypeExprWithRealPackages(t *testing.T) {
	t.Parallel()

	// This test uses actual Go packages to test more realistic scenarios
	tests := []struct {
		name        string
		packagePath string
		typeName    string
		expectedAST string
	}{
		{
			name:        "context.Context",
			packagePath: "context",
			typeName:    "Context",
			expectedAST: "context.Context",
		},
		{
			name:        "time.Time",
			packagePath: "time",
			typeName:    "Time",
			expectedAST: "time.Time",
		},
		{
			name:        "io.Reader",
			packagePath: "io",
			typeName:    "Reader",
			expectedAST: "io.Reader",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a mock package and named type
			pkg := types.NewPackage(tt.packagePath, tt.packagePath)
			obj := types.NewTypeName(0, pkg, tt.typeName, nil)
			namedType := types.NewNamed(obj, types.NewInterfaceType(nil, nil), nil)

			expr, _ := createASTTypeExpr(namedType)
			actualAST := exprToString(expr)

			if actualAST != tt.expectedAST {
				t.Errorf("createASTTypeExprWithImports() for %s:\n  Expected: %q\n  Actual:   %q",
					tt.name, tt.expectedAST, actualAST)
			}
		})
	}
}

func TestGetTypeBaseName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		createType   func() types.Type
		expectedName string
		description  string
	}{
		{
			name: "context.Context type",
			createType: func() types.Type {
				pkg := types.NewPackage("context", "context")
				obj := types.NewTypeName(0, pkg, "Context", nil)
				return types.NewNamed(obj, types.NewInterfaceType(nil, nil), nil)
			},
			expectedName: "ctx",
			description:  "context.Context should be named 'ctx'",
		},
		{
			name: "pointer to context.Context",
			createType: func() types.Type {
				pkg := types.NewPackage("context", "context")
				obj := types.NewTypeName(0, pkg, "Context", nil)
				namedType := types.NewNamed(obj, types.NewInterfaceType(nil, nil), nil)
				return types.NewPointer(namedType)
			},
			expectedName: "ctx",
			description:  "Pointer to context.Context should also be named 'ctx'",
		},
		{
			name: "basic int type",
			createType: func() types.Type {
				return types.Typ[types.Int]
			},
			expectedName: "num",
			description:  "Basic int should be named 'num'",
		},
		{
			name: "basic string type",
			createType: func() types.Type {
				return types.Typ[types.String]
			},
			expectedName: "str",
			description:  "Basic string should be named 'str'",
		},
		{
			name: "basic bool type",
			createType: func() types.Type {
				return types.Typ[types.Bool]
			},
			expectedName: "flag",
			description:  "Basic bool should be named 'flag'",
		},
		{
			name: "basic float64 type",
			createType: func() types.Type {
				return types.Typ[types.Float64]
			},
			expectedName: "value",
			description:  "Basic float64 should be named 'value'",
		},
		{
			name: "byte type",
			createType: func() types.Type {
				return types.Typ[types.Byte]
			},
			expectedName: "num",
			description:  "Byte type is treated as uint8, so should be named 'num'",
		},
		{
			name: "rune type",
			createType: func() types.Type {
				return types.Typ[types.Rune]
			},
			expectedName: "num",
			description:  "Rune type is treated as int32, so should be named 'num'",
		},
		{
			name: "complex128 type",
			createType: func() types.Type {
				return types.Typ[types.Complex128]
			},
			expectedName: "complex",
			description:  "Complex128 type should be named 'complex'",
		},
		{
			name: "uintptr type",
			createType: func() types.Type {
				return types.Typ[types.Uintptr]
			},
			expectedName: "ptr",
			description:  "Uintptr type should be named 'ptr'",
		},
		{
			name: "named type from other package",
			createType: func() types.Type {
				pkg := types.NewPackage("fmt", "fmt")
				obj := types.NewTypeName(0, pkg, "Stringer", nil)
				return types.NewNamed(obj, types.NewInterfaceType(nil, nil), nil)
			},
			expectedName: "stringer",
			description:  "Named type should use lowercase type name",
		},
		{
			name: "pointer to named type",
			createType: func() types.Type {
				pkg := types.NewPackage("sync", "sync")
				obj := types.NewTypeName(0, pkg, "Mutex", nil)
				namedType := types.NewNamed(obj, types.NewStruct(nil, nil), nil)
				return types.NewPointer(namedType)
			},
			expectedName: "mutex",
			description:  "Pointer to named type should recurse to element type",
		},
		{
			name: "slice type",
			createType: func() types.Type {
				return types.NewSlice(types.Typ[types.String])
			},
			expectedName: "[]string",
			description:  "Slice type should use full string representation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			typ := tt.createType()
			actualName := getTypeBaseName(typ)

			if actualName != tt.expectedName {
				t.Errorf("getTypeBaseName() for %s:\n  Expected: %q\n  Actual:   %q\n  Description: %s",
					tt.name, tt.expectedName, actualName, tt.description)
			}
		})
	}
}

func TestSortArguments(t *testing.T) {
	t.Parallel()

	// Create test types
	contextPkg := types.NewPackage("context", "context")
	contextObj := types.NewTypeName(0, contextPkg, "Context", nil)
	contextType := types.NewNamed(contextObj, types.NewInterfaceType(nil, nil), nil)

	stringType := types.Typ[types.String]
	intType := types.Typ[types.Int]

	syncPkg := types.NewPackage("sync", "sync")
	mutexObj := types.NewTypeName(0, syncPkg, "Mutex", nil)
	mutexType := types.NewNamed(mutexObj, types.NewStruct(nil, nil), nil)

	tests := []struct {
		name          string
		description   string
		args          []*Argument
		expectedOrder []string
	}{
		{
			name: "context.Context should come first",
			args: []*Argument{
				{Name: "str", Type: stringType},
				{Name: "ctx", Type: contextType},
				{Name: "num", Type: intType},
			},
			expectedOrder: []string{"ctx", "num", "str"},
			description:   "context.Context arguments should be ordered first, then by type string (int < string)",
		},
		{
			name: "multiple context.Context arguments maintain relative order",
			args: []*Argument{
				{Name: "str", Type: stringType},
				{Name: "ctx1", Type: contextType},
				{Name: "ctx2", Type: contextType},
				{Name: "num", Type: intType},
			},
			expectedOrder: []string{"ctx1", "ctx2", "num", "str"},
			description:   "Multiple context.Context arguments should come first, then by type string (int < string)",
		},
		{
			name: "non-context types sorted by type string",
			args: []*Argument{
				{Name: "mutex", Type: mutexType},
				{Name: "num", Type: intType},
				{Name: "str", Type: stringType},
			},
			expectedOrder: []string{"num", "str", "mutex"},
			description:   "Non-context types should be sorted by type string representation",
		},
		{
			name: "mixed context and non-context",
			args: []*Argument{
				{Name: "mutex", Type: mutexType},
				{Name: "str", Type: stringType},
				{Name: "ctx", Type: contextType},
				{Name: "num", Type: intType},
			},
			expectedOrder: []string{"ctx", "num", "str", "mutex"},
			description:   "context.Context first, then others sorted by type string",
		},
		{
			name: "empty slice",
			args: []*Argument{},
			expectedOrder: []string{},
			description:   "Empty slice should remain empty",
		},
		{
			name: "single argument",
			args: []*Argument{
				{Name: "str", Type: stringType},
			},
			expectedOrder: []string{"str"},
			description:   "Single argument should remain unchanged",
		},
		{
			name: "only context arguments",
			args: []*Argument{
				{Name: "ctx2", Type: contextType},
				{Name: "ctx1", Type: contextType},
			},
			expectedOrder: []string{"ctx2", "ctx1"},
			description:   "Only context arguments should maintain original order",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Make a copy of the input slice to avoid modifying the original
			argsCopy := make([]*Argument, len(tt.args))
			copy(argsCopy, tt.args)

			sortArguments(argsCopy)

			// Check that the length is correct
			if len(argsCopy) != len(tt.expectedOrder) {
				t.Fatalf("sortArguments() length mismatch for %s:\n  Expected length: %d\n  Actual length:   %d",
					tt.name, len(tt.expectedOrder), len(argsCopy))
			}

			// Check the order
			for i, expectedName := range tt.expectedOrder {
				if i >= len(argsCopy) {
					t.Fatalf("sortArguments() result too short for %s: missing element at index %d", tt.name, i)
				}
				if argsCopy[i].Name != expectedName {
					actualOrder := make([]string, len(argsCopy))
					for j, arg := range argsCopy {
						actualOrder[j] = arg.Name
					}
					t.Errorf("sortArguments() for %s:\n  Expected order: %v\n  Actual order:   %v\n  Description: %s",
						tt.name, tt.expectedOrder, actualOrder, tt.description)
					break
				}
			}
		})
	}
}
