package migrate

import (
	"go/ast"
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
		{
			name: "alias type in same package",
			typeFunc: func() types.Type {
				aliasName := types.NewTypeName(token.NoPos, currentPkg, "AppAlias", nil)
				underlying := types.NewNamed(
					types.NewTypeName(token.NoPos, currentPkg, "App", nil),
					types.NewStruct(nil, nil),
					nil,
				)
				return types.NewAlias(aliasName, underlying)
			},
			currentPkg: currentPkg,
			wantText:   "AppAlias",
		},
		{
			name: "alias type in external package",
			typeFunc: func() types.Type {
				aliasName := types.NewTypeName(token.NoPos, externalPkg, "ExtAlias", nil)
				underlying := types.NewNamed(
					types.NewTypeName(token.NoPos, externalPkg, "ExtType", nil),
					types.NewStruct(nil, nil),
					nil,
				)
				return types.NewAlias(aliasName, underlying)
			},
			currentPkg: currentPkg,
			wantText:   "pkg.ExtAlias",
		},
		{
			name: "pointer to alias type in same package",
			typeFunc: func() types.Type {
				aliasName := types.NewTypeName(token.NoPos, currentPkg, "AppAlias", nil)
				underlying := types.NewNamed(
					types.NewTypeName(token.NoPos, currentPkg, "App", nil),
					types.NewStruct(nil, nil),
					nil,
				)
				return types.NewPointer(types.NewAlias(aliasName, underlying))
			},
			currentPkg: currentPkg,
			wantText:   "*AppAlias",
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

// TestTypeConverterImportsV2ModuleExplicitAlias reproduces the bug where
// Imports() drops the alias when an explicit alias happens to equal the last
// path element of a v2 module path.
//
// Scenario:
//
//	import v2 "example.com/bar/v2"   // package declares: package bar
//
// lastPathElement("example.com/bar/v2") == "v2" == alias "v2", so the old
// code suppressed the alias.  The generated import became
// `"example.com/bar/v2"` (no alias), which Go resolves to "bar", not "v2".
// Any reference to v2.Service then becomes "undefined: v2".
//
// After the fix, CollectExprImports marks source-provided aliases as explicit,
// and Imports() always emits the alias for explicit entries.
func TestTypeConverterImportsV2ModuleExplicitAlias(t *testing.T) {
	tc := NewTypeConverter(nil)

	// Simulate the source import map produced by parser.ExtractImports for:
	//   import v2 "example.com/bar/v2"
	// The key is the local alias "v2", value is the import path.
	sourceImports := map[string]string{
		"v2": "example.com/bar/v2",
	}
	// ExtractImports also returns the set of paths with explicit aliases.
	explicitAliasPaths := map[string]bool{
		"example.com/bar/v2": true,
	}

	// Simulate an AST expression that references v2.Service.
	pkgIdent := &ast.Ident{Name: "v2"}
	sel := &ast.SelectorExpr{
		X:   pkgIdent,
		Sel: &ast.Ident{Name: "Service"},
	}
	tc.CollectExprImports(sel, sourceImports, explicitAliasPaths)

	specs := tc.Imports()
	if len(specs) != 1 {
		t.Fatalf("expected 1 import spec, got %d: %v", len(specs), specs)
	}
	got := specs[0]
	if got.Path != "example.com/bar/v2" {
		t.Errorf("import path: got %q, want %q", got.Path, "example.com/bar/v2")
	}
	// The alias "v2" must be present: without it Go resolves to the package
	// declaration name "bar", leaving all "v2.X" references undefined.
	if got.Name != "v2" {
		t.Errorf("import alias: got %q, want %q (must not be suppressed for v2 module with explicit alias)", got.Name, "v2")
	}
}

// TestTypeConverterImportsNonExplicitAlias verifies that when an import is
// added via TypeToExpr (i.e. from the package's declared name, not from an
// explicit source alias), the alias is still omitted when it matches the last
// path element — preserving the existing "no redundant alias" behaviour.
func TestTypeConverterImportsNonExplicitAlias(t *testing.T) {
	// package path "github.com/example/foo", declared name "foo"
	// lastPathElement == "foo" == declared name → alias should be omitted.
	fooPkg := types.NewPackage("github.com/example/foo", "foo")
	mainPkg := types.NewPackage("example.com/app", "main")

	fooTypeName := types.NewTypeName(token.NoPos, fooPkg, "Bar", nil)
	fooType := types.NewNamed(fooTypeName, types.Typ[types.Int], nil)

	tc := NewTypeConverter(mainPkg)
	_ = tc.TypeToExpr(fooType)

	specs := tc.Imports()
	if len(specs) != 1 {
		t.Fatalf("expected 1 import spec, got %d", len(specs))
	}
	// Alias should be empty: "foo" matches lastPathElement("github.com/example/foo").
	if specs[0].Name != "" {
		t.Errorf("import alias: got %q, want %q (should be omitted for non-explicit alias matching last path element)", specs[0].Name, "")
	}
}

// TestTypeConverterWrongCurrentPkgProducesUnqualifiedName demonstrates the bug described
// in the sharedTypeConverter issue: when a TypeConverter is initialised with a dependency
// package as currentPkg (pkgs[0] in packages.Load results), TypeToExpr treats types from
// that dependency package as "same package" and returns an unqualified identifier, omitting
// the necessary package qualifier and import.
//
// After the fix, TypeConverter.currentPkg must match the package being migrated (the
// output package), not the first package returned by packages.Load.
func TestTypeConverterWrongCurrentPkgProducesUnqualifiedName(t *testing.T) {
	// Simulate the dependency package "pkg" that defines the Doer interface.
	depPkg := types.NewPackage("github.com/example/pkg", "pkg")
	doerTypeName := types.NewTypeName(token.NoPos, depPkg, "Doer", nil)
	method := types.NewFunc(token.NoPos, nil, "Do", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	doerIface := types.NewInterfaceType([]*types.Func{method}, nil)
	doerIface.Complete()
	doerType := types.NewNamed(doerTypeName, doerIface, nil)

	// Simulate the "main" package that depends on pkg.
	mainPkg := types.NewPackage("example.com/app", "main")

	// BUG scenario: TypeConverter initialised with depPkg as currentPkg.
	// This mimics the bug where pkgs[0] is the dependency, not the package being migrated.
	tcWrong := NewTypeConverter(depPkg)
	gotWrong := exprToString(tcWrong.TypeToExpr(doerType))
	if gotWrong != "Doer" {
		t.Errorf("bug scenario: TypeToExpr with wrong currentPkg = %q, want %q (unqualified)", gotWrong, "Doer")
	}
	// Also confirm no import was collected (the bug: import is silently skipped).
	importsWrong := tcWrong.Imports()
	if len(importsWrong) != 0 {
		t.Errorf("bug scenario: expected 0 imports when currentPkg == depPkg, got %d", len(importsWrong))
	}

	// CORRECT scenario: TypeConverter initialised with mainPkg as currentPkg.
	// This is what the fix ensures: each package in the migration loop gets its own TypeConverter
	// (or the shared one has its currentPkg updated) to reflect the package being migrated.
	tcCorrect := NewTypeConverter(mainPkg)
	gotCorrect := exprToString(tcCorrect.TypeToExpr(doerType))
	if gotCorrect != "pkg.Doer" {
		t.Errorf("correct scenario: TypeToExpr with correct currentPkg = %q, want %q (qualified)", gotCorrect, "pkg.Doer")
	}
	// Confirm the import is collected.
	importsCorrect := tcCorrect.Imports()
	if len(importsCorrect) != 1 || importsCorrect[0].Path != "github.com/example/pkg" {
		t.Errorf("correct scenario: expected import for github.com/example/pkg, got %v", importsCorrect)
	}
}
