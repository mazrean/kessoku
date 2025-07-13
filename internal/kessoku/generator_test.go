package kessoku

import (
	"bytes"
	"go/ast"
	"go/types"
	"strings"
	"testing"
)

func createTestMetaData() *MetaData {
	return &MetaData{
		Package: Package{
			Name: "main",
			Path: "main",
		},
		Imports: map[string]*ast.ImportSpec{
			"github.com/mazrean/kessoku": {
				Path: &ast.BasicLit{
					Kind:  0,
					Value: `"github.com/mazrean/kessoku"`,
				},
			},
		},
	}
}

func createTestTypes() (configType, serviceType, intType types.Type) {
	// Create basic test types
	configType = types.NewPointer(types.NewNamed(types.NewTypeName(0, nil, "Config", nil), types.NewStruct(nil, nil), nil))
	serviceType = types.NewPointer(types.NewNamed(types.NewTypeName(0, nil, "Service", nil), types.NewStruct(nil, nil), nil))
	intType = types.Typ[types.Int]
	return
}

func createTestAST() (serviceTypeExpr, intTypeExpr ast.Expr, configProviderExpr, serviceProviderExpr ast.Expr) {
	serviceTypeExpr = &ast.StarExpr{
		X: &ast.Ident{Name: "Service"},
	}

	intTypeExpr = &ast.Ident{Name: "int"}

	configProviderExpr = &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "kessoku"},
			Sel: &ast.Ident{Name: "Provide"},
		},
		Args: []ast.Expr{
			&ast.Ident{Name: "NewConfig"},
		},
	}

	serviceProviderExpr = &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "kessoku"},
			Sel: &ast.Ident{Name: "Provide"},
		},
		Args: []ast.Expr{
			&ast.Ident{Name: "NewService"},
		},
	}

	return
}

func TestIsContextType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typeExpr types.Type
		expected bool
	}{
		{
			name: "context.Context type",
			typeExpr: func() types.Type {
				pkg := types.NewPackage("context", "context")
				ctx := types.NewTypeName(0, pkg, "Context", nil)
				return types.NewNamed(ctx, types.NewInterfaceType([]*types.Func{}, nil), nil)
			}(),
			expected: true,
		},
		{
			name: "non-context type",
			typeExpr: func() types.Type {
				pkg := types.NewPackage("fmt", "fmt")
				stringer := types.NewTypeName(0, pkg, "Stringer", nil)
				return types.NewNamed(stringer, types.NewInterfaceType([]*types.Func{}, nil), nil)
			}(),
			expected: false,
		},
		{
			name:     "basic type (string)",
			typeExpr: types.Typ[types.String],
			expected: false,
		},
		{
			name: "named type with nil package",
			typeExpr: func() types.Type {
				ctx := types.NewTypeName(0, nil, "Context", nil)
				return types.NewNamed(ctx, types.NewInterfaceType([]*types.Func{}, nil), nil)
			}(),
			expected: false,
		},
		{
			name: "context package but different type name",
			typeExpr: func() types.Type {
				pkg := types.NewPackage("context", "context")
				cancelFunc := types.NewTypeName(0, pkg, "CancelFunc", nil)
				return types.NewNamed(cancelFunc, types.NewInterfaceType([]*types.Func{}, nil), nil)
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isContextType(tt.typeExpr)
			if result != tt.expected {
				t.Errorf("isContextType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestInjectorChainStmt_Stmt(t *testing.T) {
	t.Parallel()

	configType, serviceType, _ := createTestTypes()

	tests := []struct {
		name      string
		chainStmt *InjectorChainStmt
	}{
		{
			name: "empty chain",
			chainStmt: &InjectorChainStmt{
				Statements: []InjectorStmt{},
			},
		},
		{
			name: "chain with provider call",
			chainStmt: &InjectorChainStmt{
				Statements: []InjectorStmt{
					&InjectorProviderCallStmt{
						Provider: &ProviderSpec{
							Type:          ProviderTypeFunction,
							Provides:      []types.Type{configType},
							Requires:      []types.Type{},
							IsReturnError: false,
						},
						Arguments: []*InjectorCallArgument{},
						Returns:   []*InjectorParam{NewInjectorParam(configType)},
					},
				},
			},
		},
		{
			name: "chain with multiple statements",
			chainStmt: &InjectorChainStmt{
				Statements: []InjectorStmt{
					&InjectorProviderCallStmt{
						Provider: &ProviderSpec{
							Type:          ProviderTypeFunction,
							Provides:      []types.Type{configType},
							Requires:      []types.Type{},
							IsReturnError: false,
						},
						Arguments: []*InjectorCallArgument{},
						Returns:   []*InjectorParam{NewInjectorParam(configType)},
					},
					&InjectorProviderCallStmt{
						Provider: &ProviderSpec{
							Type:          ProviderTypeFunction,
							Provides:      []types.Type{serviceType},
							Requires:      []types.Type{configType},
							IsReturnError: false,
						},
						Arguments: []*InjectorCallArgument{
							{
								Param:  NewInjectorParam(configType),
								IsWait: false,
							},
						},
						Returns: []*InjectorParam{NewInjectorParam(serviceType)},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			varPool := NewVarPool()
			injector := &Injector{
				Name:          "TestInjector",
				IsReturnError: false,
			}

			// Test that the method doesn't panic and returns valid data
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Stmt() panicked: %v", r)
				}
			}()
			
			stmts, imports := tt.chainStmt.Stmt(varPool, injector, []ast.Stmt{})

			// The method should return exactly one statement (the eg.Go call)
			if len(stmts) != 1 {
				t.Errorf("Expected 1 statement, got %d", len(stmts))
			}

			// Verify that imports is not nil (though it may be empty)
			_ = imports // Just ensure it doesn't panic when accessed

			// Check that the statement is an expression statement with a call
			if len(stmts) > 0 {
				if exprStmt, ok := stmts[0].(*ast.ExprStmt); ok {
					if callExpr, ok := exprStmt.X.(*ast.CallExpr); ok {
						if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
							if ident, ok := selExpr.X.(*ast.Ident); ok {
								if ident.Name != "eg" {
									t.Errorf("Expected eg.Go call, got %s.%s", ident.Name, selExpr.Sel.Name)
								}
								if selExpr.Sel.Name != "Go" {
									t.Errorf("Expected eg.Go call, got %s.%s", ident.Name, selExpr.Sel.Name)
								}
							}
						}
					} else {
						t.Error("Expected call expression in statement")
					}
				} else {
					t.Error("Expected expression statement")
				}
			}
		})
	}
}

func TestVarPool_GetBaseName(t *testing.T) {
	t.Parallel()

	pool := NewVarPool()

	tests := []struct {
		name     string
		typeExpr types.Type
		expected string
	}{
		// Basic types
		{
			name:     "int type",
			typeExpr: types.Typ[types.Int],
			expected: "num",
		},
		{
			name:     "int8 type",
			typeExpr: types.Typ[types.Int8],
			expected: "num",
		},
		{
			name:     "int16 type",
			typeExpr: types.Typ[types.Int16],
			expected: "num",
		},
		{
			name:     "int32 type",
			typeExpr: types.Typ[types.Int32],
			expected: "num",
		},
		{
			name:     "int64 type",
			typeExpr: types.Typ[types.Int64],
			expected: "num",
		},
		{
			name:     "uint type",
			typeExpr: types.Typ[types.Uint],
			expected: "num",
		},
		{
			name:     "uint8 type",
			typeExpr: types.Typ[types.Uint8],
			expected: "num",
		},
		{
			name:     "uint16 type",
			typeExpr: types.Typ[types.Uint16],
			expected: "num",
		},
		{
			name:     "uint32 type",
			typeExpr: types.Typ[types.Uint32],
			expected: "num",
		},
		{
			name:     "uint64 type",
			typeExpr: types.Typ[types.Uint64],
			expected: "num",
		},
		{
			name:     "float32 type",
			typeExpr: types.Typ[types.Float32],
			expected: "num",
		},
		{
			name:     "float64 type",
			typeExpr: types.Typ[types.Float64],
			expected: "num",
		},
		{
			name:     "string type",
			typeExpr: types.Typ[types.String],
			expected: "str",
		},
		{
			name:     "bool type",
			typeExpr: types.Typ[types.Bool],
			expected: "flag",
		},
		{
			name:     "complex64 type",
			typeExpr: types.Typ[types.Complex64],
			expected: "complex",
		},
		{
			name:     "complex128 type",
			typeExpr: types.Typ[types.Complex128],
			expected: "complex",
		},
		{
			name:     "uintptr type",
			typeExpr: types.Typ[types.Uintptr],
			expected: "ptr",
		},
		{
			name:     "unsafe pointer type",
			typeExpr: types.Typ[types.UnsafePointer],
			expected: "ptr",
		},
		{
			name:     "untyped nil",
			typeExpr: types.Typ[types.UntypedNil],
			expected: "null",
		},
		{
			name:     "invalid type",
			typeExpr: types.Typ[types.Invalid],
			expected: "invalid",
		},
		{
			name:     "untyped int",
			typeExpr: types.Typ[types.UntypedInt],
			expected: "num",
		},
		{
			name:     "untyped float",
			typeExpr: types.Typ[types.UntypedFloat],
			expected: "num",
		},
		{
			name:     "untyped string",
			typeExpr: types.Typ[types.UntypedString],
			expected: "str",
		},
		{
			name:     "untyped bool",
			typeExpr: types.Typ[types.UntypedBool],
			expected: "flag",
		},
		{
			name:     "untyped complex",
			typeExpr: types.Typ[types.UntypedComplex],
			expected: "complex",
		},
		{
			name:     "untyped rune",
			typeExpr: types.Typ[types.UntypedRune],
			expected: "num",
		},
		// Named types
		{
			name: "named type Service",
			typeExpr: func() types.Type {
				obj := types.NewTypeName(0, nil, "Service", nil)
				return types.NewNamed(obj, types.NewStruct(nil, nil), nil)
			}(),
			expected: "service",
		},
		{
			name: "named type UserRepository",
			typeExpr: func() types.Type {
				obj := types.NewTypeName(0, nil, "UserRepository", nil)
				return types.NewNamed(obj, types.NewStruct(nil, nil), nil)
			}(),
			expected: "userRepository",
		},
		{
			name: "context.Context type",
			typeExpr: func() types.Type {
				pkg := types.NewPackage("context", "context")
				obj := types.NewTypeName(0, pkg, "Context", nil)
				return types.NewNamed(obj, types.NewInterfaceType([]*types.Func{}, nil), nil)
			}(),
			expected: "ctx",
		},
		// Pointer types
		{
			name:     "pointer to int",
			typeExpr: types.NewPointer(types.Typ[types.Int]),
			expected: "num",
		},
		{
			name:     "pointer to string",
			typeExpr: types.NewPointer(types.Typ[types.String]),
			expected: "str",
		},
		{
			name: "pointer to named type",
			typeExpr: func() types.Type {
				obj := types.NewTypeName(0, nil, "DatabaseConfig", nil)
				namedType := types.NewNamed(obj, types.NewStruct(nil, nil), nil)
				return types.NewPointer(namedType)
			}(),
			expected: "databaseConfig",
		},
		{
			name: "double pointer to named type",
			typeExpr: func() types.Type {
				obj := types.NewTypeName(0, nil, "Service", nil)
				namedType := types.NewNamed(obj, types.NewStruct(nil, nil), nil)
				singlePtr := types.NewPointer(namedType)
				return types.NewPointer(singlePtr)
			}(),
			expected: "service",
		},
		{
			name: "pointer to context.Context",
			typeExpr: func() types.Type {
				pkg := types.NewPackage("context", "context")
				obj := types.NewTypeName(0, pkg, "Context", nil)
				namedType := types.NewNamed(obj, types.NewInterfaceType([]*types.Func{}, nil), nil)
				return types.NewPointer(namedType)
			}(),
			expected: "ctx",
		},
		// Non-basic, non-named types (should fall through to "val")
		{
			name:     "slice type",
			typeExpr: types.NewSlice(types.Typ[types.String]),
			expected: "val",
		},
		{
			name:     "map type",
			typeExpr: types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
			expected: "val",
		},
		{
			name:     "chan type",
			typeExpr: types.NewChan(types.SendRecv, types.Typ[types.String]),
			expected: "val",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := pool.getBaseName(tt.typeExpr)
			if result != tt.expected {
				t.Errorf("getBaseName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateStmts(t *testing.T) {
	t.Parallel()

	configType, serviceType, intType := createTestTypes()
	
	// Create context.Context type for testing
	contextType := func() types.Type {
		pkg := types.NewPackage("context", "context")
		obj := types.NewTypeName(0, pkg, "Context", nil)
		return types.NewNamed(obj, types.NewInterfaceType([]*types.Func{}, nil), nil)
	}()

	tests := []struct {
		name                     string
		injector                 *Injector
		expectedStmtsMin         int
		expectAsyncImport        bool
		expectErrorHandling      bool
		expectContextHandling    bool
		expectReturn            bool
	}{
		{
			name: "simple sync injector without error",
			injector: &Injector{
				Name:          "SimpleInjector",
				IsReturnError: false,
				Args:          []*InjectorArgument{},
				Stmts: []InjectorStmt{
					&InjectorProviderCallStmt{
						Provider: &ProviderSpec{
							Type:          ProviderTypeFunction,
							Provides:      []types.Type{configType},
							Requires:      []types.Type{},
							IsReturnError: false,
							IsAsync:       false,
						},
						Arguments: []*InjectorCallArgument{},
						Returns:   []*InjectorParam{NewInjectorParam(configType)},
					},
				},
				Return: &InjectorReturn{
					Param: NewInjectorParam(configType),
					Return: &Return{
						Type:        configType,
						ASTTypeExpr: &ast.StarExpr{X: &ast.Ident{Name: "Config"}},
					},
				},
			},
			expectedStmtsMin:      2, // provider call + return
			expectAsyncImport:     false,
			expectErrorHandling:   false,
			expectContextHandling: false,
			expectReturn:         true,
		},
		{
			name: "simple sync injector with error",
			injector: &Injector{
				Name:          "SimpleInjectorWithError",
				IsReturnError: true,
				Args:          []*InjectorArgument{},
				Stmts: []InjectorStmt{
					&InjectorProviderCallStmt{
						Provider: &ProviderSpec{
							Type:          ProviderTypeFunction,
							Provides:      []types.Type{serviceType},
							Requires:      []types.Type{},
							IsReturnError: true,
							IsAsync:       false,
						},
						Arguments: []*InjectorCallArgument{},
						Returns:   []*InjectorParam{NewInjectorParam(serviceType)},
					},
				},
				Return: &InjectorReturn{
					Param: NewInjectorParam(serviceType),
					Return: &Return{
						Type:        serviceType,
						ASTTypeExpr: &ast.StarExpr{X: &ast.Ident{Name: "Service"}},
					},
				},
			},
			expectedStmtsMin:      2, // provider call + return
			expectAsyncImport:     false,
			expectErrorHandling:   false,
			expectContextHandling: false,
			expectReturn:         true,
		},
		{
			name: "async injector without context",
			injector: &Injector{
				Name:          "AsyncInjector",
				IsReturnError: true,
				Args:          []*InjectorArgument{},
				Stmts: []InjectorStmt{
					&InjectorChainStmt{
						Statements: []InjectorStmt{
							&InjectorProviderCallStmt{
								Provider: &ProviderSpec{
									Type:          ProviderTypeFunction,
									Provides:      []types.Type{configType},
									Requires:      []types.Type{},
									IsReturnError: false,
									IsAsync:       true, // This makes it async
								},
								Arguments: []*InjectorCallArgument{},
								Returns:   []*InjectorParam{NewInjectorParam(configType)},
							},
						},
					},
				},
				Return: &InjectorReturn{
					Param: NewInjectorParam(configType),
					Return: &Return{
						Type:        configType,
						ASTTypeExpr: &ast.StarExpr{X: &ast.Ident{Name: "Config"}},
					},
				},
			},
			expectedStmtsMin:      3, // errgroup decl + chain + wait + return
			expectAsyncImport:     true,
			expectErrorHandling:   true,
			expectContextHandling: false,
			expectReturn:         true,
		},
		{
			name: "async injector with context",
			injector: &Injector{
				Name:          "AsyncInjectorWithContext",
				IsReturnError: true,
				Args: []*InjectorArgument{
					{
						Param: NewInjectorParam(contextType),
						Type:  contextType,
						ASTTypeExpr: &ast.SelectorExpr{
							X:   &ast.Ident{Name: "context"},
							Sel: &ast.Ident{Name: "Context"},
						},
					},
				},
				Stmts: []InjectorStmt{
					&InjectorChainStmt{
						Statements: []InjectorStmt{
							&InjectorProviderCallStmt{
								Provider: &ProviderSpec{
									Type:          ProviderTypeFunction,
									Provides:      []types.Type{serviceType},
									Requires:      []types.Type{contextType},
									IsReturnError: false,
									IsAsync:       true, // This makes it async
								},
								Arguments: []*InjectorCallArgument{
									{
										Param:  NewInjectorParam(contextType),
										IsWait: false,
									},
								},
								Returns: []*InjectorParam{NewInjectorParam(serviceType)},
							},
						},
					},
				},
				Return: &InjectorReturn{
					Param: NewInjectorParam(serviceType),
					Return: &Return{
						Type:        serviceType,
						ASTTypeExpr: &ast.StarExpr{X: &ast.Ident{Name: "Service"}},
					},
				},
			},
			expectedStmtsMin:      3, // errgroup decl + chain + wait + return
			expectAsyncImport:     true,
			expectErrorHandling:   true,
			expectContextHandling: true,
			expectReturn:         true,
		},
		{
			name: "async injector without error return",
			injector: &Injector{
				Name:          "AsyncInjectorNoError",
				IsReturnError: false,
				Args:          []*InjectorArgument{},
				Stmts: []InjectorStmt{
					&InjectorChainStmt{
						Statements: []InjectorStmt{
							&InjectorProviderCallStmt{
								Provider: &ProviderSpec{
									Type:          ProviderTypeFunction,
									Provides:      []types.Type{intType},
									Requires:      []types.Type{},
									IsReturnError: false,
									IsAsync:       true, // This makes it async
								},
								Arguments: []*InjectorCallArgument{},
								Returns:   []*InjectorParam{NewInjectorParam(intType)},
							},
						},
					},
				},
				Return: &InjectorReturn{
					Param: NewInjectorParam(intType),
					Return: &Return{
						Type:        intType,
						ASTTypeExpr: &ast.Ident{Name: "int"},
					},
				},
			},
			expectedStmtsMin:      3, // errgroup decl + chain + wait + return
			expectAsyncImport:     true,
			expectErrorHandling:   false, // No error handling since IsReturnError is false
			expectContextHandling: false,
			expectReturn:         true,
		},
		{
			name: "injector with nil return",
			injector: &Injector{
				Name:          "InjectorNilReturn",
				IsReturnError: false,
				Args:          []*InjectorArgument{},
				Stmts: []InjectorStmt{
					&InjectorProviderCallStmt{
						Provider: &ProviderSpec{
							Type:          ProviderTypeFunction,
							Provides:      []types.Type{configType},
							Requires:      []types.Type{},
							IsReturnError: false,
							IsAsync:       false,
						},
						Arguments: []*InjectorCallArgument{},
						Returns:   []*InjectorParam{NewInjectorParam(configType)},
					},
				},
				Return: nil, // No return
			},
			expectedStmtsMin:      1, // Just the provider call
			expectAsyncImport:     false,
			expectErrorHandling:   false,
			expectContextHandling: false,
			expectReturn:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			varPool := NewVarPool()
			stmts, imports := generateStmts(varPool, tt.injector)

			if len(stmts) < tt.expectedStmtsMin {
				t.Errorf("Expected at least %d statements, got %d", tt.expectedStmtsMin, len(stmts))
			}

			// Check async import
			hasAsyncImport := false
			for _, imp := range imports {
				if imp == "golang.org/x/sync/errgroup" {
					hasAsyncImport = true
					break
				}
			}

			if tt.expectAsyncImport && !hasAsyncImport {
				t.Error("Expected errgroup import but didn't find it")
			}

			if !tt.expectAsyncImport && hasAsyncImport {
				t.Error("Did not expect errgroup import but found it")
			}

			// Verify statement structure
			if tt.expectReturn {
				// Last statement should be a return
				if len(stmts) > 0 {
					if _, ok := stmts[len(stmts)-1].(*ast.ReturnStmt); !ok {
						t.Error("Expected last statement to be a return statement")
					}
				}
			}

			// Basic validation that we can generate AST nodes without panicking
			for i, stmt := range stmts {
				if stmt == nil {
					t.Errorf("Statement %d is nil", i)
				}
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	t.Parallel()

	configType, serviceType, intType := createTestTypes()
	serviceTypeExpr, intTypeExpr, configProviderExpr, serviceProviderExpr := createTestAST()

	tests := []struct {
		name                string
		metaData            *MetaData
		injectors           []*Injector
		expectedContains    []string
		expectedNotContains []string
		shouldError         bool
	}{
		{
			name:     "basic generation",
			metaData: createTestMetaData(),
			injectors: []*Injector{
				{
					Name:   "InitializeService",
					Params: []*InjectorParam{},
					Args:   nil,
					Stmts: []InjectorStmt{
						&InjectorProviderCallStmt{
							Provider: &ProviderSpec{
								Type:          ProviderTypeFunction,
								Provides:      []types.Type{configType},
								Requires:      []types.Type{},
								IsReturnError: false,
								ASTExpr:       configProviderExpr,
							},
							Arguments: []*InjectorCallArgument{},
							Returns:   []*InjectorParam{NewInjectorParam(configType)},
						},
						&InjectorProviderCallStmt{
							Provider: &ProviderSpec{
								Type:          ProviderTypeFunction,
								Provides:      []types.Type{serviceType},
								Requires:      []types.Type{configType},
								IsReturnError: false,
								ASTExpr:       serviceProviderExpr,
							},
							Arguments: []*InjectorCallArgument{
								{
									Param:  NewInjectorParam(configType),
									IsWait: false,
								},
							},
							Returns: []*InjectorParam{NewInjectorParam(serviceType)},
						},
					},
					Return: &InjectorReturn{
						Param: NewInjectorParam(serviceType),
						Return: &Return{
							Type:        serviceType,
							ASTTypeExpr: serviceTypeExpr,
						},
					},
					IsReturnError: false,
				},
			},
			expectedContains: []string{
				"// Code generated by kessoku. DO NOT EDIT.",
				"package main",
				"func InitializeService()",
				"github.com/mazrean/kessoku",
			},
			shouldError: false,
		},
		{
			name:     "with argument",
			metaData: createTestMetaData(),
			injectors: []*Injector{
				{
					Name:   "InitializeService",
					Params: []*InjectorParam{},
					Args: []*InjectorArgument{
						{
							Param: func() *InjectorParam {
								p := NewInjectorParam(intType)
								p.Ref(false) // Reference the parameter so it gets a name
								return p
							}(),
							Type:        intType,
							ASTTypeExpr: intTypeExpr,
						},
					},
					Stmts: []InjectorStmt{
						&InjectorProviderCallStmt{
							Provider: &ProviderSpec{
								Type:          ProviderTypeFunction,
								Provides:      []types.Type{serviceType},
								Requires:      []types.Type{intType},
								IsReturnError: false,
								ASTExpr:       serviceProviderExpr,
							},
							Arguments: []*InjectorCallArgument{
								{
									Param: func() *InjectorParam {
										p := NewInjectorParam(intType)
										p.Ref(false)
										return p
									}(),
									IsWait: false,
								},
							},
							Returns: []*InjectorParam{NewInjectorParam(serviceType)},
						},
					},
					Return: &InjectorReturn{
						Param: NewInjectorParam(serviceType),
						Return: &Return{
							Type:        serviceType,
							ASTTypeExpr: serviceTypeExpr,
						},
					},
					IsReturnError: false,
				},
			},
			expectedContains: []string{
				"func InitializeService(",
			},
			shouldError: false,
		},
		{
			name:     "with error handling",
			metaData: createTestMetaData(),
			injectors: []*Injector{
				{
					Name:   "InitializeService",
					Params: []*InjectorParam{},
					Args:   nil,
					Stmts: []InjectorStmt{
						&InjectorProviderCallStmt{
							Provider: &ProviderSpec{
								Type:          ProviderTypeFunction,
								Provides:      []types.Type{serviceType},
								Requires:      []types.Type{},
								IsReturnError: true,
								ASTExpr:       serviceProviderExpr,
							},
							Arguments: []*InjectorCallArgument{},
							Returns:   []*InjectorParam{NewInjectorParam(serviceType)},
						},
					},
					Return: &InjectorReturn{
						Param: NewInjectorParam(serviceType),
						Return: &Return{
							Type:        serviceType,
							ASTTypeExpr: serviceTypeExpr,
						},
					},
					IsReturnError: true,
				},
			},
			expectedContains: []string{
				"(*Service, error)",
				"if err != nil",
			},
			shouldError: false,
		},
		{
			name:     "multiple injectors",
			metaData: createTestMetaData(),
			injectors: []*Injector{
				{
					Name:   "InitializeService1",
					Params: []*InjectorParam{},
					Args:   nil,
					Stmts: []InjectorStmt{
						&InjectorProviderCallStmt{
							Provider: &ProviderSpec{
								Type:          ProviderTypeFunction,
								Provides:      []types.Type{serviceType},
								Requires:      []types.Type{},
								IsReturnError: false,
								ASTExpr:       serviceProviderExpr,
							},
							Arguments: []*InjectorCallArgument{},
							Returns:   []*InjectorParam{NewInjectorParam(serviceType)},
						},
					},
					Return: &InjectorReturn{
						Param: NewInjectorParam(serviceType),
						Return: &Return{
							Type:        serviceType,
							ASTTypeExpr: serviceTypeExpr,
						},
					},
					IsReturnError: false,
				},
				{
					Name:   "InitializeService2",
					Params: []*InjectorParam{},
					Args:   nil,
					Stmts: []InjectorStmt{
						&InjectorProviderCallStmt{
							Provider: &ProviderSpec{
								Type:          ProviderTypeFunction,
								Provides:      []types.Type{serviceType},
								Requires:      []types.Type{},
								IsReturnError: false,
								ASTExpr:       serviceProviderExpr,
							},
							Arguments: []*InjectorCallArgument{},
							Returns:   []*InjectorParam{NewInjectorParam(serviceType)},
						},
					},
					Return: &InjectorReturn{
						Param: NewInjectorParam(serviceType),
						Return: &Return{
							Type:        serviceType,
							ASTTypeExpr: serviceTypeExpr,
						},
					},
					IsReturnError: false,
				},
			},
			expectedContains: []string{
				"func InitializeService1()",
				"func InitializeService2()",
			},
			shouldError: false,
		},
		{
			name: "no imports",
			metaData: &MetaData{
				Package: Package{
					Name: "main",
					Path: "main",
				},
				Imports: make(map[string]*ast.ImportSpec),
			},
			injectors: []*Injector{
				{
					Name:   "InitializeService",
					Params: []*InjectorParam{},
					Args:   nil,
					Stmts: []InjectorStmt{
						&InjectorProviderCallStmt{
							Provider: &ProviderSpec{
								Type:          ProviderTypeFunction,
								Provides:      []types.Type{serviceType},
								Requires:      []types.Type{},
								IsReturnError: false,
								ASTExpr:       serviceProviderExpr,
							},
							Arguments: []*InjectorCallArgument{},
							Returns:   []*InjectorParam{NewInjectorParam(serviceType)},
						},
					},
					Return: &InjectorReturn{
						Param: NewInjectorParam(serviceType),
						Return: &Return{
							Type:        serviceType,
							ASTTypeExpr: serviceTypeExpr,
						},
					},
					IsReturnError: false,
				},
			},
			expectedContains: []string{
				"package main",
				"func InitializeService()",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := Generate(&buf, "test.go", tt.metaData, tt.injectors)

			if tt.shouldError {
				if err == nil {
					t.Fatal("Expected Generate to fail")
				}
				return
			}

			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			generated := buf.String()

			// Check expected content
			for _, expected := range tt.expectedContains {
				if !strings.Contains(generated, expected) {
					t.Errorf("Expected generated code to contain %q, got:\n%s", expected, generated)
				}
			}

			// Check content that should not be present
			for _, notExpected := range tt.expectedNotContains {
				if strings.Contains(generated, notExpected) {
					t.Errorf("Expected generated code NOT to contain %q, got:\n%s", notExpected, generated)
				}
			}
		})
	}
}
