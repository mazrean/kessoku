package migrate

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"io/fs"
	"os"
	"sort"
)

// Constants for file generation and formatting.
const (
	// filePermissions is the permission mode for generated files.
	filePermissions fs.FileMode = 0644

	// lineOffsetBytes is the byte offset between lines in the synthetic FileSet.
	// This ensures each line has a distinct position for proper formatting.
	lineOffsetBytes = 100

	// maxLines is the maximum number of lines supported in the synthetic FileSet.
	maxLines = 1000

	// maxFileSize is the maximum size of the synthetic file for position mapping.
	maxFileSize = 100000

	// firstArgLine is the line number for the first argument in Inject calls.
	firstArgLine = 2

	// providerStartLine is the starting line number for provider arguments.
	providerStartLine = 3
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

	// Create a FileSet with proper line information for formatting
	fset := token.NewFileSet()
	// Add a file with enough lines for our positions
	f := fset.AddFile("output.go", 1, maxFileSize)
	// Set line offsets: each line is lineOffsetBytes bytes apart to ensure clear line boundaries
	lines := make([]int, maxLines)
	for i := range lines {
		lines[i] = i * lineOffsetBytes
	}
	f.SetLines(lines)

	if err := format.Node(&buf, fset, file); err != nil {
		return err
	}

	return os.WriteFile(path, buf.Bytes(), filePermissions)
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
		expr := w.patternToExpr(elem)
		if expr == nil {
			expr = ast.NewIdent("nil")
		}
		args = append(args, expr)
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
	if p == nil {
		return nil
	}
	switch kp := p.(type) {
	case *KessokuProvide:
		return w.provideToExpr(kp)
	case *KessokuBind:
		return w.bindToExpr(kp)
	case *KessokuValue:
		return w.valueToExpr(kp)
	case *KessokuSetRef:
		if kp.Expr == nil {
			return ast.NewIdent("nil")
		}
		return kp.Expr
	default:
		return ast.NewIdent("nil")
	}
}

// provideToExpr converts a KessokuProvide to kessoku.Provide(...) expression.
func (w *Writer) provideToExpr(kp *KessokuProvide) ast.Expr {
	funcExpr := kp.FuncExpr
	if funcExpr == nil {
		funcExpr = ast.NewIdent("nil")
	}
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("kessoku"),
			Sel: ast.NewIdent("Provide"),
		},
		Args: []ast.Expr{funcExpr},
	}
}

// bindToExpr converts a KessokuBind to kessoku.Bind[I](...) expression.
func (w *Writer) bindToExpr(kb *KessokuBind) ast.Expr {
	// Build type parameter
	typeExpr := typeToExpr(kb.Interface)
	if typeExpr == nil {
		typeExpr = ast.NewIdent("any")
	}

	// Build the index expression for type parameter
	indexExpr := &ast.IndexExpr{
		X: &ast.SelectorExpr{
			X:   ast.NewIdent("kessoku"),
			Sel: ast.NewIdent("Bind"),
		},
		Index: typeExpr,
	}

	// Build the call with the provider
	providerExpr := w.patternToExpr(kb.Provider)
	if providerExpr == nil {
		providerExpr = ast.NewIdent("nil")
	}
	return &ast.CallExpr{
		Fun:  indexExpr,
		Args: []ast.Expr{providerExpr},
	}
}

// valueToExpr converts a KessokuValue to kessoku.Value(...) expression.
func (w *Writer) valueToExpr(kv *KessokuValue) ast.Expr {
	expr := kv.Expr
	if expr == nil {
		expr = ast.NewIdent("nil")
	}
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("kessoku"),
			Sel: ast.NewIdent("Value"),
		},
		Args: []ast.Expr{expr},
	}
}

// injectToDecl converts a KessokuInject to a variable declaration with proper line breaks.
// kessoku.Inject is used as: var _ = kessoku.Inject[T]("FuncName", providers...)
func (w *Writer) injectToDecl(ki *KessokuInject) *ast.GenDecl {
	// Build arguments with positions on different lines for proper formatting
	// FileSet has lines at offsets 0, lineOffsetBytes, 2*lineOffsetBytes, etc.
	args := []ast.Expr{
		&ast.BasicLit{
			ValuePos: token.Pos(firstArgLine * lineOffsetBytes),
			Kind:     token.STRING,
			Value:    `"` + ki.FuncName + `"`,
		},
	}

	for i, elem := range ki.Elements {
		pos := token.Pos((providerStartLine + i) * lineOffsetBytes)
		expr := w.patternToExprWithPos(elem, pos)
		if expr == nil {
			expr = &ast.Ident{NamePos: pos, Name: "nil"}
		}
		args = append(args, expr)
	}

	lastLine := providerStartLine + len(ki.Elements) - 1

	// Build type parameter for Inject[T]
	typeExpr := typeToExpr(ki.ReturnType)
	if typeExpr == nil {
		typeExpr = ast.NewIdent("any")
	}

	injectCall := &ast.CallExpr{
		Fun: &ast.IndexExpr{
			X: &ast.SelectorExpr{
				X:   ast.NewIdent("kessoku"),
				Sel: ast.NewIdent("Inject"),
			},
			Index: typeExpr,
		},
		Lparen: token.Pos(lineOffsetBytes), // line 1
		Args:   args,
		Rparen: token.Pos((lastLine + 1) * lineOffsetBytes), // closing line
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

// patternToExprWithPos converts a kessoku pattern to an AST expression with position.
func (w *Writer) patternToExprWithPos(p KessokuPattern, pos token.Pos) ast.Expr {
	if p == nil {
		return nil
	}
	switch kp := p.(type) {
	case *KessokuProvide:
		funcExpr := kp.FuncExpr
		if funcExpr == nil {
			funcExpr = &ast.Ident{NamePos: pos, Name: "nil"}
		} else if ident, ok := funcExpr.(*ast.Ident); ok {
			// Set position on the function expression if it's an identifier
			funcExpr = &ast.Ident{NamePos: pos, Name: ident.Name}
		}
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.Ident{
					NamePos: pos,
					Name:    "kessoku",
				},
				Sel: ast.NewIdent("Provide"),
			},
			Lparen: pos,
			Args:   []ast.Expr{funcExpr},
		}
	case *KessokuBind:
		typeExpr := typeToExpr(kp.Interface)
		if typeExpr == nil {
			typeExpr = ast.NewIdent("any")
		}
		providerExpr := w.patternToExpr(kp.Provider)
		if providerExpr == nil {
			providerExpr = ast.NewIdent("nil")
		}
		return &ast.CallExpr{
			Fun: &ast.IndexExpr{
				X: &ast.SelectorExpr{
					X: &ast.Ident{
						NamePos: pos,
						Name:    "kessoku",
					},
					Sel: ast.NewIdent("Bind"),
				},
				Index: typeExpr,
			},
			Lparen: pos,
			Args:   []ast.Expr{providerExpr},
		}
	case *KessokuValue:
		expr := kp.Expr
		if expr == nil {
			expr = ast.NewIdent("nil")
		}
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.Ident{
					NamePos: pos,
					Name:    "kessoku",
				},
				Sel: ast.NewIdent("Value"),
			},
			Lparen: pos,
			Args:   []ast.Expr{expr},
		}
	case *KessokuSetRef:
		if kp.Expr == nil {
			return &ast.Ident{NamePos: pos, Name: "nil"}
		}
		return kp.Expr
	default:
		return &ast.Ident{NamePos: pos, Name: "nil"}
	}
}
