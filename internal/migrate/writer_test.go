package migrate

import (
	"go/ast"
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
			name: "KessokuProvide pattern",
			pattern: &KessokuProvide{
				FuncExpr: ast.NewIdent("NewFoo"),
			},
			wantNil: false,
		},
		{
			name: "KessokuBind pattern",
			pattern: &KessokuBind{
				Interface: types.Typ[types.Int],
				Provider: &KessokuProvide{
					FuncExpr: ast.NewIdent("NewFoo"),
				},
			},
			wantNil: false,
		},
		{
			name: "KessokuValue pattern",
			pattern: &KessokuValue{
				Expr: ast.NewIdent("someValue"),
			},
			wantNil: false,
		},
		{
			name: "KessokuSetRef pattern",
			pattern: &KessokuSetRef{
				Expr: ast.NewIdent("OtherSet"),
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.patternToExpr(tt.pattern)
			if tt.wantNil && got != nil {
				t.Errorf("patternToExpr() = %v, want nil", got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("patternToExpr() = nil, want non-nil")
			}
		})
	}
}

func TestPatternToExprWithPos(t *testing.T) {
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
			name: "KessokuProvide pattern",
			pattern: &KessokuProvide{
				FuncExpr: ast.NewIdent("NewFoo"),
			},
			wantNil: false,
		},
		{
			name: "KessokuBind pattern",
			pattern: &KessokuBind{
				Interface: types.Typ[types.Int],
				Provider: &KessokuProvide{
					FuncExpr: ast.NewIdent("NewFoo"),
				},
			},
			wantNil: false,
		},
		{
			name: "KessokuValue pattern",
			pattern: &KessokuValue{
				Expr: ast.NewIdent("someValue"),
			},
			wantNil: false,
		},
		{
			name: "KessokuSetRef pattern",
			pattern: &KessokuSetRef{
				Expr: ast.NewIdent("OtherSet"),
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.patternToExprWithPos(tt.pattern, 100)
			if tt.wantNil && got != nil {
				t.Errorf("patternToExprWithPos() = %v, want nil", got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("patternToExprWithPos() = nil, want non-nil")
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
