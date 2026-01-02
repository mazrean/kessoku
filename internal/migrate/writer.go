package migrate

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
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
type Writer struct {
	typeConverter *TypeConverter
}

// NewWriter creates a new Writer instance with the given type converter.
func NewWriter(tc *TypeConverter) *Writer {
	return &Writer{typeConverter: tc}
}

// GetCollectedImports returns the imports collected during AST generation.
func (w *Writer) GetCollectedImports() []ImportSpec {
	if w.typeConverter == nil {
		return nil
	}
	return w.typeConverter.Imports()
}

// typeToExpr converts a types.Type to ast.Expr using the type converter if available.
func (w *Writer) typeToExpr(t types.Type) ast.Expr {
	if w.typeConverter != nil {
		return w.typeConverter.TypeToExpr(t)
	}
	// Fallback to simple type expression (for backward compatibility)
	return typeToExpr(t)
}

// exprWithPos rebuilds an expression with the given position.
// This ensures proper line formatting in the generated output.
func (w *Writer) exprWithPos(expr ast.Expr, pos token.Pos) ast.Expr {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return &ast.Ident{NamePos: pos, Name: e.Name}
	case *ast.SelectorExpr:
		return &ast.SelectorExpr{
			X:   w.exprWithPos(e.X, pos),
			Sel: &ast.Ident{NamePos: pos, Name: e.Sel.Name},
		}
	case *ast.BasicLit:
		return &ast.BasicLit{ValuePos: pos, Kind: e.Kind, Value: e.Value}
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{OpPos: pos, Op: e.Op, X: w.exprWithPos(e.X, pos)}
	case *ast.CompositeLit:
		return &ast.CompositeLit{
			Type:   e.Type,
			Lbrace: pos,
			Elts:   e.Elts,
			Rbrace: e.Rbrace,
		}
	case *ast.CallExpr:
		// Apply position to simple args, but not to nested CallExprs
		// Nested CallExprs (from patternToExpr) should keep their default position
		// so the formatter puts them on the same line
		args := make([]ast.Expr, len(e.Args))
		for i, arg := range e.Args {
			if _, isCall := arg.(*ast.CallExpr); isCall {
				// Keep nested CallExpr as-is to preserve formatting
				args[i] = arg
			} else {
				args[i] = w.exprWithPos(arg, pos)
			}
		}
		return &ast.CallExpr{
			Fun:    w.exprWithPos(e.Fun, pos),
			Lparen: pos,
			Args:   args,
		}
	case *ast.IndexExpr:
		return &ast.IndexExpr{
			X:      w.exprWithPos(e.X, pos),
			Lbrack: pos,
			Index:  e.Index, // Keep type parameter as-is
		}
	case *ast.FuncLit:
		// Function literals keep their structure, just update position
		return e
	default:
		return expr
	}
}

// goGenerateComment is the go:generate directive added to generated files.
const goGenerateComment = "//go:generate go tool kessoku $GOFILE\n\n"

// Write writes the merged output to the specified file.
func (w *Writer) Write(output *MergedOutput, path string) error {
	file := w.buildFile(output)

	var buf bytes.Buffer

	// Add go:generate comment at the top
	buf.WriteString(goGenerateComment)

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
	// Deduplicate imports by path (keep first occurrence)
	seen := make(map[string]bool)
	var deduped []ImportSpec
	for _, imp := range imports {
		if seen[imp.Path] {
			continue
		}
		seen[imp.Path] = true
		deduped = append(deduped, imp)
	}

	// Sort imports by path
	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].Path < deduped[j].Path
	})

	var specs []ast.Spec
	for _, imp := range deduped {
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

// setToDecl converts a KessokuSet to a variable declaration with proper line breaks.
// buildElementArgs converts a list of kessoku patterns to positioned AST expressions.
// startLine is the line number for the first element.
func (w *Writer) buildElementArgs(elements []KessokuPattern, startLine int) []ast.Expr {
	args := make([]ast.Expr, 0, len(elements))
	for i, elem := range elements {
		pos := token.Pos((startLine + i) * lineOffsetBytes)
		expr := w.patternToExprWithPos(elem, pos)
		if expr == nil {
			expr = &ast.Ident{NamePos: pos, Name: "nil"}
		}
		args = append(args, expr)
	}
	return args
}

// wrapInVarDecl wraps an expression in a var declaration.
func wrapInVarDecl(varName string, value ast.Expr) *ast.GenDecl {
	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names:  []*ast.Ident{ast.NewIdent(varName)},
				Values: []ast.Expr{value},
			},
		},
	}
}

func (w *Writer) setToDecl(ks *KessokuSet) *ast.GenDecl {
	args := w.buildElementArgs(ks.Elements, firstArgLine)

	lastLine := firstArgLine + len(ks.Elements) - 1
	if len(ks.Elements) == 0 {
		lastLine = firstArgLine
	}

	setCall := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("kessoku"),
			Sel: ast.NewIdent("Set"),
		},
		Lparen: token.Pos(lineOffsetBytes),
		Args:   args,
		Rparen: token.Pos((lastLine + 1) * lineOffsetBytes),
	}

	return wrapInVarDecl(ks.VarName, setCall)
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
	typeExpr := w.typeToExpr(kb.Interface)
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
	// First arg is the function name
	args := []ast.Expr{
		&ast.BasicLit{
			ValuePos: token.Pos(firstArgLine * lineOffsetBytes),
			Kind:     token.STRING,
			Value:    `"` + ki.FuncName + `"`,
		},
	}

	// Append provider elements
	args = append(args, w.buildElementArgs(ki.Elements, providerStartLine)...)

	lastLine := providerStartLine + len(ki.Elements) - 1

	// Build type parameter for Inject[T]
	typeExpr := w.typeToExpr(ki.ReturnType)
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
		Lparen: token.Pos(lineOffsetBytes),
		Args:   args,
		Rparen: token.Pos((lastLine + 1) * lineOffsetBytes),
	}

	return wrapInVarDecl("_", injectCall)
}

// patternToExprWithPos converts a kessoku pattern to an AST expression with position.
func (w *Writer) patternToExprWithPos(p KessokuPattern, pos token.Pos) ast.Expr {
	if p == nil {
		return nil
	}
	expr := w.patternToExpr(p)
	if expr == nil {
		return &ast.Ident{NamePos: pos, Name: "nil"}
	}
	return w.exprWithPos(expr, pos)
}
