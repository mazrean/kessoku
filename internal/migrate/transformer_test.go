package migrate

import (
	"go/ast"
	"go/types"
	"testing"
)

func TestIsErrorType(t *testing.T) {
	tests := []struct {
		typeFunc func() types.Type
		name     string
		want     bool
	}{
		{
			name: "nil type",
			typeFunc: func() types.Type {
				return nil
			},
			want: false,
		},
		{
			name: "built-in error type",
			typeFunc: func() types.Type {
				return types.Universe.Lookup("error").Type()
			},
			want: true,
		},
		{
			name: "int type",
			typeFunc: func() types.Type {
				return types.Typ[types.Int]
			},
			want: false,
		},
		{
			name: "string type",
			typeFunc: func() types.Type {
				return types.Typ[types.String]
			},
			want: false,
		},
		{
			name: "bool type",
			typeFunc: func() types.Type {
				return types.Typ[types.Bool]
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ := tt.typeFunc()
			got := isErrorType(typ)
			if got != tt.want {
				t.Errorf("isErrorType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTypeToExpr(t *testing.T) {
	tests := []struct {
		typeFunc func() types.Type
		name     string
		wantNil  bool
	}{
		{
			name: "nil type",
			typeFunc: func() types.Type {
				return nil
			},
			wantNil: true,
		},
		{
			name: "basic int type",
			typeFunc: func() types.Type {
				return types.Typ[types.Int]
			},
			wantNil: false,
		},
		{
			name: "basic string type",
			typeFunc: func() types.Type {
				return types.Typ[types.String]
			},
			wantNil: false,
		},
		{
			name: "pointer to int",
			typeFunc: func() types.Type {
				return types.NewPointer(types.Typ[types.Int])
			},
			wantNil: false,
		},
		{
			name: "slice of int",
			typeFunc: func() types.Type {
				return types.NewSlice(types.Typ[types.Int])
			},
			wantNil: false,
		},
		{
			name: "map of string to int",
			typeFunc: func() types.Type {
				return types.NewMap(types.Typ[types.String], types.Typ[types.Int])
			},
			wantNil: false,
		},
		{
			name: "array of 10 int",
			typeFunc: func() types.Type {
				return types.NewArray(types.Typ[types.Int], 10)
			},
			wantNil: false,
		},
		{
			name: "channel of int",
			typeFunc: func() types.Type {
				return types.NewChan(types.SendRecv, types.Typ[types.Int])
			},
			wantNil: false,
		},
		{
			name: "empty interface (any)",
			typeFunc: func() types.Type {
				return types.NewInterfaceType(nil, nil)
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ := tt.typeFunc()
			got := typeToExpr(typ)
			if tt.wantNil && got != nil {
				t.Errorf("typeToExpr() = %v, want nil", got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("typeToExpr() = nil, want non-nil")
			}
		})
	}
}

func TestToLowerCamel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "single lowercase letter",
			input: "a",
			want:  "a",
		},
		{
			name:  "single uppercase letter",
			input: "A",
			want:  "a",
		},
		{
			name:  "PascalCase",
			input: "FooBar",
			want:  "fooBar",
		},
		{
			name:  "already camelCase",
			input: "fooBar",
			want:  "fooBar",
		},
		{
			name:  "all uppercase",
			input: "FOO",
			want:  "fOO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toLowerCamel(tt.input)
			if got != tt.want {
				t.Errorf("toLowerCamel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUnwrapPointer(t *testing.T) {
	tests := []struct {
		typ      types.Type
		wantType types.Type
		name     string
	}{
		{
			name:     "non-pointer type returns same type",
			typ:      types.Typ[types.Int],
			wantType: types.Typ[types.Int],
		},
		{
			name:     "pointer to basic type returns element type",
			typ:      types.NewPointer(types.Typ[types.Int]),
			wantType: types.Typ[types.Int],
		},
		{
			name:     "pointer to string returns string",
			typ:      types.NewPointer(types.Typ[types.String]),
			wantType: types.Typ[types.String],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unwrapPointer(tt.typ)
			if got != tt.wantType {
				t.Errorf("unwrapPointer() = %v, want %v", got, tt.wantType)
			}
		})
	}
}

func TestTypeConverterAddImportCollision(t *testing.T) {
	tc := NewTypeConverter(nil)

	// First import: v1 -> path1
	name1 := tc.AddImport("example.com/api1", "v1")
	if name1 != "v1" {
		t.Errorf("AddImport() first call = %q, want %q", name1, "v1")
	}

	// Same path again should return the same name
	name1Again := tc.AddImport("example.com/api1", "v1")
	if name1Again != "v1" {
		t.Errorf("AddImport() same path = %q, want %q", name1Again, "v1")
	}

	// Different path with same name should get renamed
	name2 := tc.AddImport("example.com/api2", "v1")
	if name2 != "v1_1" {
		t.Errorf("AddImport() collision = %q, want %q", name2, "v1_1")
	}

	// Third collision should get v1_2
	name3 := tc.AddImport("example.com/api3", "v1")
	if name3 != "v1_2" {
		t.Errorf("AddImport() second collision = %q, want %q", name3, "v1_2")
	}

	// Verify imports are correct
	imports := tc.Imports()
	if len(imports) != 3 {
		t.Errorf("Imports() count = %d, want 3", len(imports))
	}

	importMap := make(map[string]string)
	for _, imp := range imports {
		name := imp.Name
		if name == "" {
			name = lastPathElement(imp.Path)
		}
		importMap[imp.Path] = name
	}

	expectedImports := map[string]string{
		"example.com/api1": "v1",
		"example.com/api2": "v1_1",
		"example.com/api3": "v1_2",
	}

	for path, wantName := range expectedImports {
		if gotName := importMap[path]; gotName != wantName {
			t.Errorf("import %q has name %q, want %q", path, gotName, wantName)
		}
	}
}

func TestTypeConverterCollectExprImports(t *testing.T) {
	tests := []struct {
		expr          ast.Expr
		sourceImports map[string]string
		wantImports   map[string]string
		name          string
	}{
		{
			name:          "nil expression",
			expr:          nil,
			sourceImports: map[string]string{"pkg": "example.com/pkg"},
			wantImports:   map[string]string{},
		},
		{
			name:          "simple identifier (no package)",
			expr:          ast.NewIdent("Foo"),
			sourceImports: map[string]string{"pkg": "example.com/pkg"},
			wantImports:   map[string]string{},
		},
		{
			name: "selector expression with package",
			expr: &ast.SelectorExpr{
				X:   ast.NewIdent("pkg"),
				Sel: ast.NewIdent("NewFoo"),
			},
			sourceImports: map[string]string{"pkg": "example.com/pkg"},
			wantImports:   map[string]string{"example.com/pkg": "pkg"},
		},
		{
			name: "selector expression with unknown package",
			expr: &ast.SelectorExpr{
				X:   ast.NewIdent("unknown"),
				Sel: ast.NewIdent("NewFoo"),
			},
			sourceImports: map[string]string{"pkg": "example.com/pkg"},
			wantImports:   map[string]string{},
		},
		{
			name: "nested call expression",
			expr: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("traq"),
					Sel: ast.NewIdent("NewOIDC"),
				},
			},
			sourceImports: map[string]string{"traq": "github.com/traPtitech/traQ"},
			wantImports:   map[string]string{"github.com/traPtitech/traQ": "traq"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTypeConverter(nil)
			tc.CollectExprImports(tt.expr, tt.sourceImports)

			got := make(map[string]string)
			for _, imp := range tc.Imports() {
				name := imp.Name
				if name == "" {
					// Extract name from path
					for i := len(imp.Path) - 1; i >= 0; i-- {
						if imp.Path[i] == '/' {
							name = imp.Path[i+1:]
							break
						}
					}
					if name == "" {
						name = imp.Path
					}
				}
				got[imp.Path] = name
			}

			if len(got) != len(tt.wantImports) {
				t.Errorf("CollectExprImports() got %d imports, want %d", len(got), len(tt.wantImports))
			}
			for path, name := range tt.wantImports {
				if gotName, exists := got[path]; !exists {
					t.Errorf("CollectExprImports() missing import path %q", path)
				} else if gotName != name {
					t.Errorf("CollectExprImports() got name %q for path %q, want %q", gotName, path, name)
				}
			}
		})
	}
}

func TestTypeConverterCollectPatternImports(t *testing.T) {
	sourceImports := map[string]string{
		"pkg":  "example.com/pkg",
		"traq": "github.com/traPtitech/traQ",
	}

	tests := []struct {
		name        string
		pattern     KessokuPattern
		wantImports []string // import paths
	}{
		{
			name:        "nil pattern",
			pattern:     nil,
			wantImports: []string{},
		},
		{
			name: "KessokuProvide with package reference",
			pattern: &KessokuProvide{
				FuncExpr: &ast.SelectorExpr{
					X:   ast.NewIdent("pkg"),
					Sel: ast.NewIdent("NewFoo"),
				},
			},
			wantImports: []string{"example.com/pkg"},
		},
		{
			name: "KessokuSet with nested providers",
			pattern: &KessokuSet{
				VarName: "TestSet",
				Elements: []KessokuPattern{
					&KessokuProvide{
						FuncExpr: &ast.SelectorExpr{
							X:   ast.NewIdent("pkg"),
							Sel: ast.NewIdent("NewFoo"),
						},
					},
					&KessokuProvide{
						FuncExpr: &ast.SelectorExpr{
							X:   ast.NewIdent("traq"),
							Sel: ast.NewIdent("NewBar"),
						},
					},
				},
			},
			wantImports: []string{"example.com/pkg", "github.com/traPtitech/traQ"},
		},
		{
			name: "KessokuValue with package reference",
			pattern: &KessokuValue{
				Expr: &ast.SelectorExpr{
					X:   ast.NewIdent("pkg"),
					Sel: ast.NewIdent("DefaultConfig"),
				},
			},
			wantImports: []string{"example.com/pkg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTypeConverter(nil)
			tc.CollectPatternImports(tt.pattern, sourceImports)

			got := tc.Imports()
			gotPaths := make(map[string]bool)
			for _, imp := range got {
				gotPaths[imp.Path] = true
			}

			if len(gotPaths) != len(tt.wantImports) {
				t.Errorf("CollectPatternImports() got %d imports, want %d", len(gotPaths), len(tt.wantImports))
			}
			for _, wantPath := range tt.wantImports {
				if !gotPaths[wantPath] {
					t.Errorf("CollectPatternImports() missing import path %q", wantPath)
				}
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		slice []string
		want  bool
	}{
		{
			name:  "empty slice",
			slice: []string{},
			s:     "foo",
			want:  false,
		},
		{
			name:  "element exists",
			slice: []string{"foo", "bar", "baz"},
			s:     "bar",
			want:  true,
		},
		{
			name:  "element does not exist",
			slice: []string{"foo", "bar", "baz"},
			s:     "qux",
			want:  false,
		},
		{
			name:  "single element slice - match",
			slice: []string{"foo"},
			s:     "foo",
			want:  true,
		},
		{
			name:  "single element slice - no match",
			slice: []string{"foo"},
			s:     "bar",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.s)
			if got != tt.want {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.s, got, tt.want)
			}
		})
	}
}
