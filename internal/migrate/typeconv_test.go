package migrate

import (
	"go/token"
	"go/types"
	"testing"
)

func TestTypeConverterTypeToExpr(t *testing.T) {
	// Create a test package for external type testing
	externalPkg := types.NewPackage("github.com/example/pkg", "pkg")
	currentPkg := types.NewPackage("github.com/current/pkg", "current")

	// Create a named type in the external package
	externalTypeName := types.NewTypeName(token.NoPos, externalPkg, "ExternalType", nil)
	externalType := types.NewNamed(externalTypeName, types.Typ[types.Int], nil)

	// Create a named type in the current package
	currentTypeName := types.NewTypeName(token.NoPos, currentPkg, "CurrentType", nil)
	currentType := types.NewNamed(currentTypeName, types.Typ[types.Int], nil)

	// Create built-in error type
	errorType := types.Universe.Lookup("error").Type()

	// Create a non-empty interface
	method := types.NewFunc(token.NoPos, nil, "Method", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	nonEmptyInterface := types.NewInterfaceType([]*types.Func{method}, nil)
	nonEmptyInterface.Complete()

	tests := []struct {
		typeFunc   func() types.Type
		currentPkg *types.Package
		name       string
		wantText   string
	}{
		{
			name: "nil type",
			typeFunc: func() types.Type {
				return nil
			},
			wantText: "",
		},
		{
			name: "basic int type",
			typeFunc: func() types.Type {
				return types.Typ[types.Int]
			},
			wantText: "int",
		},
		{
			name: "pointer to string",
			typeFunc: func() types.Type {
				return types.NewPointer(types.Typ[types.String])
			},
			wantText: "*string",
		},
		{
			name: "slice of int",
			typeFunc: func() types.Type {
				return types.NewSlice(types.Typ[types.Int])
			},
			wantText: "[]int",
		},
		{
			name: "map of string to int",
			typeFunc: func() types.Type {
				return types.NewMap(types.Typ[types.String], types.Typ[types.Int])
			},
			wantText: "map[string]int",
		},
		{
			name: "array of 5 int",
			typeFunc: func() types.Type {
				return types.NewArray(types.Typ[types.Int], 5)
			},
			wantText: "[5]int",
		},
		{
			name: "empty interface",
			typeFunc: func() types.Type {
				return types.NewInterfaceType(nil, nil)
			},
			wantText: "interface {\n}",
		},
		{
			name: "non-empty interface",
			typeFunc: func() types.Type {
				return nonEmptyInterface
			},
			wantText: "any",
		},
		{
			name: "channel SendRecv",
			typeFunc: func() types.Type {
				return types.NewChan(types.SendRecv, types.Typ[types.Int])
			},
			wantText: "chan int",
		},
		{
			name: "channel SendOnly",
			typeFunc: func() types.Type {
				return types.NewChan(types.SendOnly, types.Typ[types.Int])
			},
			wantText: "chan<- int",
		},
		{
			name: "channel RecvOnly",
			typeFunc: func() types.Type {
				return types.NewChan(types.RecvOnly, types.Typ[types.Int])
			},
			wantText: "<-chan int",
		},
		{
			name: "built-in error type",
			typeFunc: func() types.Type {
				return errorType
			},
			wantText: "error",
		},
		{
			name: "external package type",
			typeFunc: func() types.Type {
				return externalType
			},
			currentPkg: currentPkg,
			wantText:   "pkg.ExternalType",
		},
		{
			name: "same package type",
			typeFunc: func() types.Type {
				return currentType
			},
			currentPkg: currentPkg,
			wantText:   "CurrentType",
		},
		{
			name: "unsafe.Pointer type",
			typeFunc: func() types.Type {
				return types.Typ[types.UnsafePointer]
			},
			wantText: "unsafe.Pointer",
		},
		{
			name: "instantiated generic type same package (single type arg)",
			typeFunc: func() types.Type {
				tName := types.NewTypeName(token.NoPos, nil, "T", nil)
				tParam := types.NewTypeParam(tName, types.NewInterfaceType(nil, nil))
				boxTypeName := types.NewTypeName(token.NoPos, currentPkg, "Box", nil)
				boxNamed := types.NewNamed(boxTypeName, types.NewStruct(nil, nil), nil)
				boxNamed.SetTypeParams([]*types.TypeParam{tParam})
				ctx := types.NewContext()
				inst, err := types.Instantiate(ctx, boxNamed, []types.Type{types.Typ[types.Int]}, true)
				if err != nil {
					panic(err)
				}
				return inst
			},
			currentPkg: currentPkg,
			wantText:   "Box[int]",
		},
		{
			name: "instantiated generic type same package (two type args)",
			typeFunc: func() types.Type {
				t1Name := types.NewTypeName(token.NoPos, nil, "K", nil)
				t1Param := types.NewTypeParam(t1Name, types.NewInterfaceType(nil, nil))
				t2Name := types.NewTypeName(token.NoPos, nil, "V", nil)
				t2Param := types.NewTypeParam(t2Name, types.NewInterfaceType(nil, nil))
				pairTypeName := types.NewTypeName(token.NoPos, currentPkg, "Pair", nil)
				pairNamed := types.NewNamed(pairTypeName, types.NewStruct(nil, nil), nil)
				pairNamed.SetTypeParams([]*types.TypeParam{t1Param, t2Param})
				ctx := types.NewContext()
				inst, err := types.Instantiate(ctx, pairNamed, []types.Type{types.Typ[types.String], types.Typ[types.Int]}, true)
				if err != nil {
					panic(err)
				}
				return inst
			},
			currentPkg: currentPkg,
			wantText:   "Pair[string, int]",
		},
		{
			name: "instantiated generic type external package (single type arg)",
			typeFunc: func() types.Type {
				tName := types.NewTypeName(token.NoPos, nil, "T", nil)
				tParam := types.NewTypeParam(tName, types.NewInterfaceType(nil, nil))
				boxTypeName := types.NewTypeName(token.NoPos, externalPkg, "Box", nil)
				boxNamed := types.NewNamed(boxTypeName, types.NewStruct(nil, nil), nil)
				boxNamed.SetTypeParams([]*types.TypeParam{tParam})
				ctx := types.NewContext()
				inst, err := types.Instantiate(ctx, boxNamed, []types.Type{types.Typ[types.Int]}, true)
				if err != nil {
					panic(err)
				}
				return inst
			},
			currentPkg: currentPkg,
			wantText:   "pkg.Box[int]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTypeConverter(tt.currentPkg)
			typ := tt.typeFunc()
			got := tc.TypeToExpr(typ)
			gotText := exprToString(got)
			if gotText != tt.wantText {
				t.Errorf("TypeToExpr() = %q, want %q", gotText, tt.wantText)
			}
		})
	}
}

func TestTypeConverterTypeToExprDefault(t *testing.T) {
	// Test with a type that falls through to default case
	// Using a Tuple type which is not handled by the switch
	tc := NewTypeConverter(nil)

	// Create a tuple type (not commonly used, falls to default)
	tuple := types.NewTuple(types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int]))

	got := tc.TypeToExpr(tuple)
	gotText := exprToString(got)

	// Default case uses t.String(), which for tuple is "(x int)"
	wantText := "(x int)"
	if gotText != wantText {
		t.Errorf("TypeToExpr() = %q, want %q", gotText, wantText)
	}
}

func TestTypeConverterImportsDedup(t *testing.T) {
	tc := NewTypeConverter(nil)

	// Add the same import twice
	tc.AddImport("github.com/example/pkg", "pkg")
	tc.AddImport("github.com/example/pkg", "pkg")

	imports := tc.Imports()
	if len(imports) != 1 {
		t.Errorf("expected 1 import after dedup, got %d", len(imports))
	}
}
