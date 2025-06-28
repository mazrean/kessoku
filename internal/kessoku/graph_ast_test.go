package kessoku

import (
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

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
			expectedAST: "interface{}",
			description: "Empty interface should be represented as interface{}",
		},
		{
			name: "non-empty interface",
			createType: func() types.Type {
				// Create an interface with a method
				method := types.NewFunc(0, nil, "String", types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.String])), false))
				return types.NewInterfaceType([]*types.Func{method}, nil)
			},
			expectedAST: "interface{}",
			description: "Non-empty anonymous interface should fallback to interface{}",
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
			expectedAST: "func",
			description: "Function signatures should be simplified to 'func'",
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
			expr := createASTTypeExpr(typ)

			// Convert AST expression back to string for comparison
			actualAST := exprToString(expr)

			if actualAST != tt.expectedAST {
				t.Errorf("createASTTypeExpr() for %s:\n  Expected: %q\n  Actual:   %q\n  Description: %s",
					tt.name, tt.expectedAST, actualAST, tt.description)
			}

			// Additional validation: make sure the expression is valid AST
			if expr == nil {
				t.Errorf("createASTTypeExpr() returned nil for %s", tt.name)
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
			expr := createASTTypeExpr(typ)

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
		name         string
		packagePath  string
		typeName     string
		expectedAST  string
	}{
		{
			name:         "context.Context",
			packagePath:  "context",
			typeName:     "Context",
			expectedAST:  "context.Context",
		},
		{
			name:         "time.Time",
			packagePath:  "time",
			typeName:     "Time",
			expectedAST:  "time.Time",
		},
		{
			name:         "io.Reader",
			packagePath:  "io",
			typeName:     "Reader",
			expectedAST:  "io.Reader",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a mock package and named type
			pkg := types.NewPackage(tt.packagePath, tt.packagePath)
			obj := types.NewTypeName(0, pkg, tt.typeName, nil)
			namedType := types.NewNamed(obj, types.NewInterfaceType(nil, nil), nil)

			expr := createASTTypeExpr(namedType)
			actualAST := exprToString(expr)

			if actualAST != tt.expectedAST {
				t.Errorf("createASTTypeExpr() for %s:\n  Expected: %q\n  Actual:   %q",
					tt.name, tt.expectedAST, actualAST)
			}
		})
	}
}