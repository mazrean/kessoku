package migrate

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestFindWireImport(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name     string
		src      string
		wantName string
	}{
		{
			name: "standard wire import",
			src: `package test
import "github.com/google/wire"
`,
			wantName: "wire",
		},
		{
			name: "aliased wire import",
			src: `package test
import w "github.com/google/wire"
`,
			wantName: "w",
		},
		{
			name: "no wire import",
			src: `package test
import "fmt"
`,
			wantName: "",
		},
		{
			name: "wire import with other imports",
			src: `package test
import (
	"fmt"
	"github.com/google/wire"
	"os"
)
`,
			wantName: "wire",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.src, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			got := p.FindWireImport(file)
			if got != tt.wantName {
				t.Errorf("FindWireImport() = %q, want %q", got, tt.wantName)
			}
		})
	}
}

func TestExtractPatterns(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name         string
		src          string
		wireAlias    string
		wantPatterns int
		wantWarnings int
	}{
		{
			name: "no patterns",
			src: `package test
import "github.com/google/wire"
var x = 1
`,
			wireAlias:    "wire",
			wantPatterns: 0,
			wantWarnings: 0,
		},
		{
			name: "single NewSet",
			src: `package test
import "github.com/google/wire"
var TestSet = wire.NewSet(NewFoo)
func NewFoo() *Foo { return &Foo{} }
type Foo struct{}
`,
			wireAlias:    "wire",
			wantPatterns: 1,
			wantWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use packages.Load to get type information
			cfg := &packages.Config{
				Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
				Overlay: map[string][]byte{
					"test.go": []byte(tt.src),
				},
			}

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			// Create a minimal type info (patterns won't have full type info)
			info := &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
				Defs:  make(map[*ast.Ident]types.Object),
				Uses:  make(map[*ast.Ident]types.Object),
			}

			patterns, warnings := p.ExtractPatterns(file, info, tt.wireAlias, "test.go")
			if len(patterns) != tt.wantPatterns {
				t.Errorf("ExtractPatterns() got %d patterns, want %d", len(patterns), tt.wantPatterns)
			}
			if len(warnings) != tt.wantWarnings {
				t.Errorf("ExtractPatterns() got %d warnings, want %d", len(warnings), tt.wantWarnings)
			}
			_ = cfg // Use cfg to avoid unused variable error
		})
	}
}

func TestExtractImports(t *testing.T) {
	p := NewParser()

	tests := []struct {
		want map[string]string
		name string
		src  string
	}{
		{
			name: "no imports",
			src:  `package test`,
			want: map[string]string{},
		},
		{
			name: "single import",
			src: `package test
import "fmt"
`,
			want: map[string]string{"fmt": "fmt"},
		},
		{
			name: "multiple imports",
			src: `package test
import (
	"fmt"
	"github.com/google/wire"
	"os"
)
`,
			want: map[string]string{
				"fmt":  "fmt",
				"wire": "github.com/google/wire",
				"os":   "os",
			},
		},
		{
			name: "aliased import",
			src: `package test
import w "github.com/google/wire"
`,
			want: map[string]string{"w": "github.com/google/wire"},
		},
		{
			name: "mixed aliased and non-aliased",
			src: `package test
import (
	"fmt"
	w "github.com/google/wire"
	k "github.com/mazrean/kessoku"
)
`,
			want: map[string]string{
				"fmt": "fmt",
				"w":   "github.com/google/wire",
				"k":   "github.com/mazrean/kessoku",
			},
		},
		{
			name: "dot import is skipped",
			src: `package test
import . "fmt"
`,
			want: map[string]string{},
		},
		{
			name: "blank import is skipped",
			src: `package test
import _ "embed"
`,
			want: map[string]string{},
		},
		{
			name: "nested package path",
			src: `package test
import "github.com/traPtitech/traQ/repository/gorm2/v1"
`,
			want: map[string]string{"v1": "github.com/traPtitech/traQ/repository/gorm2/v1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.src, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			got := p.ExtractImports(file)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractImports() got %d imports, want %d", len(got), len(tt.want))
			}
			for name, path := range tt.want {
				if gotPath, exists := got[name]; !exists {
					t.Errorf("ExtractImports() missing import %q", name)
				} else if gotPath != path {
					t.Errorf("ExtractImports() got path %q for %q, want %q", gotPath, name, path)
				}
			}
		})
	}
}

func TestExtractTypeFromNew(t *testing.T) {
	p := NewParser()

	// Create a simple type info
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
	}

	tests := []struct {
		expr    ast.Expr
		name    string
		wantNil bool
	}{
		{
			name:    "nil expression",
			expr:    nil,
			wantNil: true,
		},
		{
			name:    "non-call expression",
			expr:    ast.NewIdent("foo"),
			wantNil: true,
		},
		{
			name: "call expression but not new",
			expr: &ast.CallExpr{
				Fun:  ast.NewIdent("make"),
				Args: []ast.Expr{ast.NewIdent("int")},
			},
			wantNil: true,
		},
		{
			name: "new call with wrong number of args",
			expr: &ast.CallExpr{
				Fun:  ast.NewIdent("new"),
				Args: []ast.Expr{},
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.extractTypeFromNew(tt.expr, info)
			if tt.wantNil && got != nil {
				t.Errorf("extractTypeFromNew() = %v, want nil", got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("extractTypeFromNew() = nil, want non-nil")
			}
		})
	}
}
