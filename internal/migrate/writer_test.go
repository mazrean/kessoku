package migrate

import (
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"
)

func TestPatternToDecl(t *testing.T) {
	w := NewWriter(nil)

	tests := []struct {
		pattern KessokuPattern
		name    string
		wantNil bool
	}{
		{
			name:    "nil pattern",
			pattern: nil,
			wantNil: true,
		},
		{
			name: "KessokuSet pattern",
			pattern: &KessokuSet{
				VarName:  "TestSet",
				Elements: []KessokuPattern{},
			},
			wantNil: false,
		},
		{
			name: "KessokuInject pattern",
			pattern: &KessokuInject{
				FuncName:   "InitializeApp",
				Elements:   []KessokuPattern{},
				ReturnType: types.Typ[types.Int],
			},
			wantNil: false,
		},
		{
			name: "KessokuInject pattern with nil ReturnType",
			pattern: &KessokuInject{
				FuncName:   "InitializeApp",
				Elements:   []KessokuPattern{},
				ReturnType: nil,
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.PatternToDecl(tt.pattern)
			if tt.wantNil && got != nil {
				t.Errorf("PatternToDecl() = %v, want nil", got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("PatternToDecl() = nil, want non-nil")
			}
		})
	}
}

func TestPatternToExpr(t *testing.T) {
	w := NewWriter(nil)

	tests := []struct {
		pattern  KessokuPattern
		name     string
		wantText string
	}{
		{
			name:     "nil pattern",
			pattern:  nil,
			wantText: "",
		},
		{
			name: "KessokuProvide pattern",
			pattern: &KessokuProvide{
				FuncExpr: ast.NewIdent("NewFoo"),
			},
			wantText: "kessoku.Provide(NewFoo)",
		},
		{
			name: "KessokuBind pattern",
			pattern: &KessokuBind{
				Interface: types.Typ[types.Int],
				Provider: &KessokuProvide{
					FuncExpr: ast.NewIdent("NewFoo"),
				},
			},
			wantText: "kessoku.Bind[int](kessoku.Provide(NewFoo))",
		},
		{
			name: "KessokuValue pattern",
			pattern: &KessokuValue{
				Expr: ast.NewIdent("someValue"),
			},
			wantText: "kessoku.Value(someValue)",
		},
		{
			name: "KessokuSetRef pattern",
			pattern: &KessokuSetRef{
				Expr: ast.NewIdent("OtherSet"),
			},
			wantText: "OtherSet",
		},
		{
			name: "KessokuSetRef pattern with nil Expr",
			pattern: &KessokuSetRef{
				Expr: nil,
			},
			wantText: "nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.patternToExpr(tt.pattern)
			gotText := exprToString(got)
			if gotText != tt.wantText {
				t.Errorf("patternToExpr() = %q, want %q", gotText, tt.wantText)
			}
		})
	}
}

func TestPatternToExprWithPos(t *testing.T) {
	w := NewWriter(nil)

	tests := []struct {
		pattern  KessokuPattern
		name     string
		wantText string
	}{
		{
			name:     "nil pattern",
			pattern:  nil,
			wantText: "",
		},
		{
			name: "KessokuProvide pattern",
			pattern: &KessokuProvide{
				FuncExpr: ast.NewIdent("NewFoo"),
			},
			wantText: "kessoku.Provide(NewFoo)",
		},
		{
			name: "KessokuBind pattern",
			pattern: &KessokuBind{
				Interface: types.Typ[types.Int],
				Provider: &KessokuProvide{
					FuncExpr: ast.NewIdent("NewFoo"),
				},
			},
			wantText: "kessoku.Bind[int](kessoku.Provide(NewFoo))",
		},
		{
			name: "KessokuValue pattern",
			pattern: &KessokuValue{
				Expr: ast.NewIdent("someValue"),
			},
			wantText: "kessoku.Value(someValue)",
		},
		{
			name: "KessokuSetRef pattern",
			pattern: &KessokuSetRef{
				Expr: ast.NewIdent("OtherSet"),
			},
			wantText: "OtherSet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.patternToExprWithPos(tt.pattern, 100)
			gotText := exprToString(got)
			if gotText != tt.wantText {
				t.Errorf("patternToExprWithPos() = %q, want %q", gotText, tt.wantText)
			}
		})
	}
}

func TestBuildImportDecl(t *testing.T) {
	w := NewWriter(nil)

	tests := []struct {
		name    string
		imports []ImportSpec
	}{
		{
			name:    "single import",
			imports: []ImportSpec{{Path: "github.com/mazrean/kessoku"}},
		},
		{
			name: "multiple imports",
			imports: []ImportSpec{
				{Path: "github.com/mazrean/kessoku"},
				{Path: "fmt"},
			},
		},
		{
			name: "import with alias",
			imports: []ImportSpec{
				{Path: "github.com/mazrean/kessoku", Name: "k"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.buildImportDecl(tt.imports)
			if got == nil {
				t.Error("buildImportDecl() = nil, want non-nil")
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	w := NewWriter(nil)

	// Test writing to an invalid path
	output := &MergedOutput{
		Package: "test",
		Imports: []ImportSpec{{Path: "github.com/mazrean/kessoku"}},
	}

	// Try to write to a path that doesn't exist and is not writable
	err := w.Write(output, "/nonexistent/path/file.go")
	if err == nil {
		t.Error("Write() should fail for invalid path")
	}
}

func TestWriteSuccess(t *testing.T) {
	w := NewWriter(nil)

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.go")

	output := &MergedOutput{
		Package: "test",
		Imports: []ImportSpec{{Path: "github.com/mazrean/kessoku"}},
		TopLevelDecls: []ast.Decl{
			w.PatternToDecl(&KessokuSet{
				VarName: "TestSet",
				Elements: []KessokuPattern{
					&KessokuProvide{FuncExpr: ast.NewIdent("NewFoo")},
				},
			}),
		},
	}

	err := w.Write(output, outputPath)
	if err != nil {
		t.Errorf("Write() failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("output file was not created")
	}
}

func TestExprWithPos(t *testing.T) {
	w := NewWriter(nil)
	pos := token.Pos(100)

	tests := []struct {
		expr      ast.Expr
		checkExpr func(t *testing.T, expr ast.Expr)
		name      string
		wantText  string
	}{
		{
			name:     "nil expression",
			expr:     nil,
			wantText: "",
		},
		{
			name:     "Ident",
			expr:     ast.NewIdent("foo"),
			wantText: "foo",
			checkExpr: func(t *testing.T, expr ast.Expr) {
				ident, ok := expr.(*ast.Ident)
				if !ok {
					t.Fatalf("expected *ast.Ident, got %T", expr)
				}
				if ident.NamePos != pos {
					t.Errorf("expected pos %d, got %d", pos, ident.NamePos)
				}
			},
		},
		{
			name: "SelectorExpr",
			expr: &ast.SelectorExpr{
				X:   ast.NewIdent("pkg"),
				Sel: ast.NewIdent("Func"),
			},
			wantText: "pkg.Func",
			checkExpr: func(t *testing.T, expr ast.Expr) {
				sel, ok := expr.(*ast.SelectorExpr)
				if !ok {
					t.Fatalf("expected *ast.SelectorExpr, got %T", expr)
				}
				if sel.Sel.Name != "Func" {
					t.Errorf("expected 'Func', got %q", sel.Sel.Name)
				}
			},
		},
		{
			name:     "BasicLit",
			expr:     &ast.BasicLit{Kind: token.STRING, Value: `"hello"`},
			wantText: `"hello"`,
			checkExpr: func(t *testing.T, expr ast.Expr) {
				lit, ok := expr.(*ast.BasicLit)
				if !ok {
					t.Fatalf("expected *ast.BasicLit, got %T", expr)
				}
				if lit.ValuePos != pos {
					t.Errorf("expected pos %d, got %d", pos, lit.ValuePos)
				}
			},
		},
		{
			name:     "UnaryExpr",
			expr:     &ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("x")},
			wantText: "&x",
			checkExpr: func(t *testing.T, expr ast.Expr) {
				unary, ok := expr.(*ast.UnaryExpr)
				if !ok {
					t.Fatalf("expected *ast.UnaryExpr, got %T", expr)
				}
				if unary.OpPos != pos {
					t.Errorf("expected pos %d, got %d", pos, unary.OpPos)
				}
			},
		},
		{
			name: "CompositeLit",
			expr: &ast.CompositeLit{
				Type: ast.NewIdent("Foo"),
				Elts: []ast.Expr{ast.NewIdent("bar")},
			},
			wantText: "Foo{bar}",
			checkExpr: func(t *testing.T, expr ast.Expr) {
				comp, ok := expr.(*ast.CompositeLit)
				if !ok {
					t.Fatalf("expected *ast.CompositeLit, got %T", expr)
				}
				if comp.Lbrace != pos {
					t.Errorf("expected Lbrace pos %d, got %d", pos, comp.Lbrace)
				}
			},
		},
		{
			name: "CallExpr with simple args",
			expr: &ast.CallExpr{
				Fun:  ast.NewIdent("foo"),
				Args: []ast.Expr{ast.NewIdent("arg1")},
			},
			wantText: "foo(arg1)",
			checkExpr: func(t *testing.T, expr ast.Expr) {
				call, ok := expr.(*ast.CallExpr)
				if !ok {
					t.Fatalf("expected *ast.CallExpr, got %T", expr)
				}
				if call.Lparen != pos {
					t.Errorf("expected Lparen pos %d, got %d", pos, call.Lparen)
				}
			},
		},
		{
			name: "CallExpr with nested CallExpr arg",
			expr: &ast.CallExpr{
				Fun: ast.NewIdent("outer"),
				Args: []ast.Expr{
					&ast.CallExpr{
						Fun:  ast.NewIdent("inner"),
						Args: []ast.Expr{ast.NewIdent("x")},
					},
				},
			},
			wantText: "outer(inner(x))",
			checkExpr: func(t *testing.T, expr ast.Expr) {
				call, ok := expr.(*ast.CallExpr)
				if !ok {
					t.Fatalf("expected *ast.CallExpr, got %T", expr)
				}
				// Nested CallExpr should be kept as-is
				innerCall, ok := call.Args[0].(*ast.CallExpr)
				if !ok {
					t.Fatalf("expected nested *ast.CallExpr, got %T", call.Args[0])
				}
				// Inner call should NOT have updated position
				if innerCall.Lparen == pos {
					t.Error("nested CallExpr should not have updated position")
				}
			},
		},
		{
			name: "IndexExpr",
			expr: &ast.IndexExpr{
				X:     ast.NewIdent("slice"),
				Index: ast.NewIdent("i"),
			},
			wantText: "slice[i]",
			checkExpr: func(t *testing.T, expr ast.Expr) {
				idx, ok := expr.(*ast.IndexExpr)
				if !ok {
					t.Fatalf("expected *ast.IndexExpr, got %T", expr)
				}
				if idx.Lbrack != pos {
					t.Errorf("expected Lbrack pos %d, got %d", pos, idx.Lbrack)
				}
			},
		},
		{
			name: "FuncLit",
			expr: &ast.FuncLit{
				Type: &ast.FuncType{Params: &ast.FieldList{}},
				Body: &ast.BlockStmt{},
			},
			wantText: "func() {\n}",
			checkExpr: func(t *testing.T, expr ast.Expr) {
				// FuncLit should be returned as-is
				_, ok := expr.(*ast.FuncLit)
				if !ok {
					t.Fatalf("expected *ast.FuncLit, got %T", expr)
				}
			},
		},
		{
			name:     "unknown expression type (default)",
			expr:     &ast.ParenExpr{X: ast.NewIdent("x")},
			wantText: "(x)",
			checkExpr: func(t *testing.T, expr ast.Expr) {
				// Default case returns expression as-is
				_, ok := expr.(*ast.ParenExpr)
				if !ok {
					t.Fatalf("expected *ast.ParenExpr, got %T", expr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.exprWithPos(tt.expr, pos)
			gotText := exprToString(got)
			if gotText != tt.wantText {
				t.Errorf("exprWithPos() = %q, want %q", gotText, tt.wantText)
			}

			if tt.checkExpr != nil {
				tt.checkExpr(t, got)
			}
		})
	}
}

func TestProvideToExprNil(t *testing.T) {
	w := NewWriter(nil)
	kp := &KessokuProvide{FuncExpr: nil}

	got := w.provideToExpr(kp)
	if got == nil {
		t.Error("provideToExpr() = nil, want non-nil")
	}

	call, ok := got.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected *ast.CallExpr, got %T", got)
	}

	// First arg should be "nil" ident
	if len(call.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(call.Args))
	}
	ident, ok := call.Args[0].(*ast.Ident)
	if !ok {
		t.Fatalf("expected *ast.Ident, got %T", call.Args[0])
	}
	if ident.Name != "nil" {
		t.Errorf("expected 'nil', got %q", ident.Name)
	}
}

func TestBindToExprNilInterface(t *testing.T) {
	w := NewWriter(nil)
	kb := &KessokuBind{
		Interface: nil,
		Provider:  &KessokuProvide{FuncExpr: ast.NewIdent("NewFoo")},
	}

	got := w.bindToExpr(kb)
	if got == nil {
		t.Error("bindToExpr() = nil, want non-nil")
	}

	call, ok := got.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected *ast.CallExpr, got %T", got)
	}

	// Check that the IndexExpr uses "any" as type parameter
	idx, ok := call.Fun.(*ast.IndexExpr)
	if !ok {
		t.Fatalf("expected *ast.IndexExpr, got %T", call.Fun)
	}
	ident, ok := idx.Index.(*ast.Ident)
	if !ok {
		t.Fatalf("expected *ast.Ident for Index, got %T", idx.Index)
	}
	if ident.Name != "any" {
		t.Errorf("expected 'any', got %q", ident.Name)
	}
}

func TestBindToExprNilProvider(t *testing.T) {
	w := NewWriter(nil)
	kb := &KessokuBind{
		Interface: types.Typ[types.Int],
		Provider:  nil,
	}

	got := w.bindToExpr(kb)
	if got == nil {
		t.Error("bindToExpr() = nil, want non-nil")
	}

	call, ok := got.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected *ast.CallExpr, got %T", got)
	}

	// Check that arg is "nil" ident
	if len(call.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(call.Args))
	}
	ident, ok := call.Args[0].(*ast.Ident)
	if !ok {
		t.Fatalf("expected *ast.Ident, got %T", call.Args[0])
	}
	if ident.Name != "nil" {
		t.Errorf("expected 'nil', got %q", ident.Name)
	}
}

func TestValueToExprNil(t *testing.T) {
	w := NewWriter(nil)
	kv := &KessokuValue{Expr: nil}

	got := w.valueToExpr(kv)
	if got == nil {
		t.Error("valueToExpr() = nil, want non-nil")
	}

	call, ok := got.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected *ast.CallExpr, got %T", got)
	}

	// First arg should be "nil" ident
	if len(call.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(call.Args))
	}
	ident, ok := call.Args[0].(*ast.Ident)
	if !ok {
		t.Fatalf("expected *ast.Ident, got %T", call.Args[0])
	}
	if ident.Name != "nil" {
		t.Errorf("expected 'nil', got %q", ident.Name)
	}
}

func TestPatternToExprDefault(t *testing.T) {
	w := NewWriter(nil)

	// Test with an unknown pattern type (nil elements in KessokuSet)
	set := &KessokuSet{VarName: "TestSet"}

	got := w.patternToExpr(set)
	if got == nil {
		t.Error("patternToExpr() = nil for KessokuSet (default case)")
		return
	}

	// KessokuSet falls through to default, which returns nil ident
	ident, ok := got.(*ast.Ident)
	if !ok {
		t.Fatalf("expected *ast.Ident, got %T", got)
	}
	if ident.Name != "nil" {
		t.Errorf("expected 'nil', got %q", ident.Name)
	}
}

func TestGetCollectedImports(t *testing.T) {
	// Test with nil TypeConverter
	w1 := NewWriter(nil)
	if imports := w1.GetCollectedImports(); imports != nil {
		t.Errorf("expected nil imports with nil TypeConverter, got %v", imports)
	}

	// Test with TypeConverter
	tc := NewTypeConverter(nil)
	tc.AddImport("github.com/example/pkg", "pkg")

	w2 := NewWriter(tc)
	imports := w2.GetCollectedImports()
	if len(imports) != 1 {
		t.Errorf("expected 1 import, got %d", len(imports))
	}
}

func TestBuildElementArgsWithNilPattern(t *testing.T) {
	w := NewWriter(nil)

	elements := []KessokuPattern{nil, &KessokuProvide{FuncExpr: ast.NewIdent("NewFoo")}}
	args := w.buildElementArgs(elements, 2)

	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}

	// First arg should be nil ident (from nil pattern)
	ident, ok := args[0].(*ast.Ident)
	if !ok {
		t.Fatalf("expected *ast.Ident for nil pattern, got %T", args[0])
	}
	if ident.Name != "nil" {
		t.Errorf("expected 'nil', got %q", ident.Name)
	}
}

func TestBuildImportDeclDedup(t *testing.T) {
	w := NewWriter(nil)

	// Test deduplication
	imports := []ImportSpec{
		{Path: "github.com/example/pkg"},
		{Path: "github.com/example/pkg"}, // duplicate
		{Path: "fmt"},
	}

	decl := w.buildImportDecl(imports)
	if decl == nil {
		t.Fatal("buildImportDecl() = nil")
	}

	// Should have only 2 specs after dedup
	if len(decl.Specs) != 2 {
		t.Errorf("expected 2 specs after dedup, got %d", len(decl.Specs))
	}
}
