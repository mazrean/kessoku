package migrate

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"sort"
)

// Writer generates kessoku output files.
type Writer struct{}

// NewWriter creates a new Writer instance.
func NewWriter() *Writer {
	return &Writer{}
}

// Write writes the merged output to the specified file.
func (w *Writer) Write(output *MergedOutput, path string) error {
	file := w.buildFile(output)

	var buf bytes.Buffer
	fset := token.NewFileSet()

	if err := format.Node(&buf, fset, file); err != nil {
		return err
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}

// buildFile builds an AST file from the merged output.
func (w *Writer) buildFile(output *MergedOutput) *ast.File {
	file := &ast.File{
		Name:  ast.NewIdent(output.Package),
		Decls: make([]ast.Decl, 0),
	}

	// Add imports
	if len(output.Imports) > 0 {
		importDecl := w.buildImportDecl(output.Imports)
		file.Decls = append(file.Decls, importDecl)
	}

	// Add top-level declarations
	file.Decls = append(file.Decls, output.TopLevelDecls...)

	return file
}

// buildImportDecl builds an import declaration.
func (w *Writer) buildImportDecl(imports []ImportSpec) *ast.GenDecl {
	// Sort imports by path
	sorted := make([]ImportSpec, len(imports))
	copy(sorted, imports)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Path < sorted[j].Path
	})

	var specs []ast.Spec
	for _, imp := range sorted {
		spec := &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: `"` + imp.Path + `"`,
			},
		}
		if imp.Name != "" {
			spec.Name = ast.NewIdent(imp.Name)
		}
		specs = append(specs, spec)
	}

	return &ast.GenDecl{
		Tok:    token.IMPORT,
		Lparen: token.Pos(1),
		Specs:  specs,
		Rparen: token.Pos(1),
	}
}

// PatternToDecl converts a kessoku pattern to an AST declaration.
func (w *Writer) PatternToDecl(p KessokuPattern) ast.Decl {
	switch kp := p.(type) {
	case *KessokuSet:
		return w.setToDecl(kp)
	case *KessokuInject:
		return w.injectToDecl(kp)
	default:
		return nil
	}
}

// setToDecl converts a KessokuSet to a variable declaration.
func (w *Writer) setToDecl(ks *KessokuSet) *ast.GenDecl {
	// Build kessoku.Set call
	var args []ast.Expr
	for _, elem := range ks.Elements {
		args = append(args, w.patternToExpr(elem))
	}

	setCall := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("kessoku"),
			Sel: ast.NewIdent("Set"),
		},
		Args: args,
	}

	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names:  []*ast.Ident{ast.NewIdent(ks.VarName)},
				Values: []ast.Expr{setCall},
			},
		},
	}
}

// patternToExpr converts a kessoku pattern to an AST expression.
func (w *Writer) patternToExpr(p KessokuPattern) ast.Expr {
	switch kp := p.(type) {
	case *KessokuProvide:
		return w.provideToExpr(kp)
	case *KessokuBind:
		return w.bindToExpr(kp)
	case *KessokuValue:
		return w.valueToExpr(kp)
	case *KessokuSetRef:
		return kp.Expr
	default:
		return nil
	}
}

// provideToExpr converts a KessokuProvide to kessoku.Provide(...) expression.
func (w *Writer) provideToExpr(kp *KessokuProvide) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("kessoku"),
			Sel: ast.NewIdent("Provide"),
		},
		Args: []ast.Expr{kp.FuncExpr},
	}
}

// bindToExpr converts a KessokuBind to kessoku.Bind[I](...) expression.
func (w *Writer) bindToExpr(kb *KessokuBind) ast.Expr {
	// Build type parameter
	typeExpr := typeToExpr(kb.Interface)

	// Build the index expression for type parameter
	indexExpr := &ast.IndexExpr{
		X: &ast.SelectorExpr{
			X:   ast.NewIdent("kessoku"),
			Sel: ast.NewIdent("Bind"),
		},
		Index: typeExpr,
	}

	// Build the call with the provider
	return &ast.CallExpr{
		Fun:  indexExpr,
		Args: []ast.Expr{w.patternToExpr(kb.Provider)},
	}
}

// valueToExpr converts a KessokuValue to kessoku.Value(...) expression.
func (w *Writer) valueToExpr(kv *KessokuValue) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("kessoku"),
			Sel: ast.NewIdent("Value"),
		},
		Args: []ast.Expr{kv.Expr},
	}
}

// injectToDecl converts a KessokuInject to a variable declaration.
// kessoku.Inject is used as: var _ = kessoku.Inject[T]("FuncName", providers...)
func (w *Writer) injectToDecl(ki *KessokuInject) *ast.GenDecl {
	// Build arguments: first is the function name as string, then providers
	args := []ast.Expr{
		&ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf("%q", ki.FuncName),
		},
	}
	for _, elem := range ki.Elements {
		args = append(args, w.patternToExpr(elem))
	}

	// Build type parameter for Inject[T]
	typeExpr := typeToExpr(ki.ReturnType)

	injectCall := &ast.CallExpr{
		Fun: &ast.IndexExpr{
			X: &ast.SelectorExpr{
				X:   ast.NewIdent("kessoku"),
				Sel: ast.NewIdent("Inject"),
			},
			Index: typeExpr,
		},
		Args: args,
	}

	// Build var _ = kessoku.Inject[T]("FuncName", ...)
	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names:  []*ast.Ident{ast.NewIdent("_")},
				Values: []ast.Expr{injectCall},
			},
		},
	}
}
