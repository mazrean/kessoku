package migrate

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

// Constants for wire pattern argument counts.
const (
	// wireBindArgCount is the expected number of arguments for wire.Bind.
	wireBindArgCount = 2

	// wireInterfaceValueArgCount is the expected number of arguments for wire.InterfaceValue.
	wireInterfaceValueArgCount = 2

	// wireFieldsOfMinArgs is the minimum number of arguments for wire.FieldsOf.
	wireFieldsOfMinArgs = 2
)

// Parser extracts wire patterns from Go source files.
type Parser struct{}

// NewParser creates a new Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// FindWireImport finds the wire import in the file and returns its alias.
// Returns empty string if no wire import found.
func (p *Parser) FindWireImport(file *ast.File) string {
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if path == "github.com/google/wire" {
			if imp.Name != nil {
				return imp.Name.Name
			}
			return "wire"
		}
	}
	return ""
}

// ExtractImports extracts all imports from a file as a map from package name/alias to import path.
func (p *Parser) ExtractImports(file *ast.File) map[string]string {
	imports := make(map[string]string)
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		var name string
		if imp.Name != nil {
			name = imp.Name.Name
		} else {
			// Use the last element of the path as the default package name
			name = lastPathElement(path)
		}
		// Skip dot imports and blank imports
		if name != "." && name != "_" {
			imports[name] = path
		}
	}
	return imports
}

// ExtractPatterns extracts wire patterns from the file.
func (p *Parser) ExtractPatterns(file *ast.File, info *types.Info, wireAlias string, filePath string) ([]WirePattern, []Warning) {
	var patterns []WirePattern
	var warnings []Warning

	// Visit all declarations
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			// Handle var declarations (wire.NewSet, etc.)
			if d.Tok != token.VAR {
				continue
			}

			for _, spec := range d.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}

				for i, value := range valueSpec.Values {
					call, ok := value.(*ast.CallExpr)
					if !ok {
						continue
					}

					varName := ""
					if i < len(valueSpec.Names) {
						varName = valueSpec.Names[i].Name
					}

					pattern, warn := p.parseCallExpr(call, info, wireAlias, filePath, varName)
					if warn != nil {
						warnings = append(warnings, *warn)
					}
					if pattern != nil {
						patterns = append(patterns, pattern)
					}
				}
			}

		case *ast.FuncDecl:
			// Handle function declarations (wire.Build injectors)
			if d.Body == nil {
				continue
			}

			// Look for wire.Build calls in the function body
			for _, stmt := range d.Body.List {
				exprStmt, ok := stmt.(*ast.ExprStmt)
				if !ok {
					continue
				}

				call, ok := exprStmt.X.(*ast.CallExpr)
				if !ok {
					continue
				}

				// Check if it's a wire.Build call
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					continue
				}

				ident, ok := sel.X.(*ast.Ident)
				if !ok || ident.Name != wireAlias || sel.Sel.Name != "Build" {
					continue
				}

				// Parse wire.Build
				buildPattern := p.parseBuild(call, d, info, wireAlias, filePath)
				if buildPattern != nil {
					patterns = append(patterns, buildPattern)
				}
			}
		}
	}

	return patterns, warnings
}

// parseCallExpr parses a call expression and returns a wire pattern if applicable.
func (p *Parser) parseCallExpr(call *ast.CallExpr, info *types.Info, wireAlias string, filePath string, varName string) (WirePattern, *Warning) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, nil
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != wireAlias {
		return nil, nil
	}

	switch sel.Sel.Name {
	case "NewSet":
		return p.parseNewSet(call, info, wireAlias, filePath, varName), nil
	case "Bind":
		return p.parseBind(call, info, filePath), nil
	case "Value":
		return p.parseValue(call, info, filePath), nil
	case "InterfaceValue":
		return p.parseInterfaceValue(call, info, filePath), nil
	case "Struct":
		return p.parseStruct(call, info, filePath), nil
	case "FieldsOf":
		return p.parseFieldsOf(call, info, filePath), nil
	case "Build":
		// wire.Build is handled separately in ExtractPatterns for function declarations
		return nil, nil
	}

	return nil, nil
}

// parseNewSet parses wire.NewSet(...) pattern.
func (p *Parser) parseNewSet(call *ast.CallExpr, info *types.Info, wireAlias string, filePath string, varName string) *WireNewSet {
	set := &WireNewSet{
		baseWirePattern: baseWirePattern{
			Pos:  call.Pos(),
			File: filePath,
		},
		VarName: varName,
	}

	for _, arg := range call.Args {
		elem := p.parseSetElement(arg, info, wireAlias, filePath)
		if elem != nil {
			set.Elements = append(set.Elements, elem)
		}
	}

	return set
}

// parseSetElement parses an element within wire.NewSet.
func (p *Parser) parseSetElement(expr ast.Expr, info *types.Info, wireAlias string, filePath string) WirePattern {
	switch e := expr.(type) {
	case *ast.CallExpr:
		// Nested wire call (Bind, Value, etc.)
		pattern, _ := p.parseCallExpr(e, info, wireAlias, filePath, "")
		return pattern
	case *ast.Ident:
		// Could be a provider function or set reference
		if obj := info.ObjectOf(e); obj != nil {
			if fn, ok := obj.(*types.Func); ok {
				return &WireProviderFunc{
					baseWirePattern: baseWirePattern{
						Pos:  e.Pos(),
						File: filePath,
					},
					Func: fn,
					Name: e.Name,
					Expr: e,
				}
			}
			// Variable reference (another set)
			return &WireSetRef{
				baseWirePattern: baseWirePattern{
					Pos:  e.Pos(),
					File: filePath,
				},
				Name: e.Name,
				Expr: e,
			}
		}
		// Fallback: treat as provider function reference
		return &WireProviderFunc{
			baseWirePattern: baseWirePattern{
				Pos:  e.Pos(),
				File: filePath,
			},
			Name: e.Name,
			Expr: e,
		}
	case *ast.SelectorExpr:
		// Package-qualified identifier (e.g., pkg.NewFoo or pkg.FooSet)
		if obj := info.ObjectOf(e.Sel); obj != nil {
			if fn, ok := obj.(*types.Func); ok {
				return &WireProviderFunc{
					baseWirePattern: baseWirePattern{
						Pos:  e.Pos(),
						File: filePath,
					},
					Func: fn,
					Name: e.Sel.Name,
					Expr: e,
				}
			}
		}
		// Variable reference (another set from different package)
		return &WireSetRef{
			baseWirePattern: baseWirePattern{
				Pos:  e.Pos(),
				File: filePath,
			},
			Name: e.Sel.Name,
			Expr: e,
		}
	}

	return nil
}

// parseBind parses wire.Bind(new(Interface), new(Impl)) pattern.
// extractStringFields extracts string literals from a slice of expressions.
func extractStringFields(args []ast.Expr) []string {
	var fields []string
	for _, arg := range args {
		if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			field := strings.Trim(lit.Value, "\"")
			fields = append(fields, field)
		}
	}
	return fields
}

func (p *Parser) parseBind(call *ast.CallExpr, info *types.Info, filePath string) *WireBind {
	if len(call.Args) != wireBindArgCount {
		return nil
	}

	ifaceType := p.extractTypeFromNew(call.Args[0], info)
	implType := p.extractTypeFromNew(call.Args[1], info)

	if ifaceType == nil || implType == nil {
		return nil
	}

	return &WireBind{
		baseWirePattern: baseWirePattern{
			Pos:  call.Pos(),
			File: filePath,
		},
		Interface:      ifaceType,
		Implementation: implType,
	}
}

// parseValue parses wire.Value(expr) pattern.
func (p *Parser) parseValue(call *ast.CallExpr, info *types.Info, filePath string) *WireValue {
	if len(call.Args) != 1 {
		return nil
	}

	var valueType types.Type
	if tv, ok := info.Types[call.Args[0]]; ok {
		valueType = tv.Type
	}

	return &WireValue{
		baseWirePattern: baseWirePattern{
			Pos:  call.Pos(),
			File: filePath,
		},
		Expr: call.Args[0],
		Type: valueType,
	}
}

// parseInterfaceValue parses wire.InterfaceValue(new(Interface), expr) pattern.
func (p *Parser) parseInterfaceValue(call *ast.CallExpr, info *types.Info, filePath string) *WireInterfaceValue {
	if len(call.Args) != wireInterfaceValueArgCount {
		return nil
	}

	ifaceType := p.extractTypeFromNew(call.Args[0], info)
	if ifaceType == nil {
		return nil
	}

	return &WireInterfaceValue{
		baseWirePattern: baseWirePattern{
			Pos:  call.Pos(),
			File: filePath,
		},
		Interface: ifaceType,
		Expr:      call.Args[1],
	}
}

// parseStruct parses wire.Struct(new(Type), fields...) pattern.
func (p *Parser) parseStruct(call *ast.CallExpr, info *types.Info, filePath string) *WireStruct {
	if len(call.Args) < 1 {
		return nil
	}

	structType := p.extractTypeFromNew(call.Args[0], info)
	if structType == nil {
		return nil
	}

	isPointer := false
	if _, ok := structType.(*types.Pointer); ok {
		isPointer = true
	}

	fields := extractStringFields(call.Args[1:])
	if len(fields) == 0 {
		fields = []string{"*"}
	}

	return &WireStruct{
		baseWirePattern: baseWirePattern{
			Pos:  call.Pos(),
			File: filePath,
		},
		StructType: structType,
		Fields:     fields,
		IsPointer:  isPointer,
	}
}

// parseFieldsOf parses wire.FieldsOf(new(Type), fields...) pattern.
func (p *Parser) parseFieldsOf(call *ast.CallExpr, info *types.Info, filePath string) *WireFieldsOf {
	if len(call.Args) < wireFieldsOfMinArgs {
		return nil
	}

	structType := p.extractTypeFromNew(call.Args[0], info)
	if structType == nil {
		return nil
	}

	fields := extractStringFields(call.Args[1:])

	return &WireFieldsOf{
		baseWirePattern: baseWirePattern{
			Pos:  call.Pos(),
			File: filePath,
		},
		StructType: structType,
		Fields:     fields,
	}
}

// parseBuild parses wire.Build(...) pattern in an injector function.
func (p *Parser) parseBuild(call *ast.CallExpr, funcDecl *ast.FuncDecl, info *types.Info, wireAlias string, filePath string) *WireBuild {
	build := &WireBuild{
		baseWirePattern: baseWirePattern{
			Pos:  call.Pos(),
			File: filePath,
		},
		FuncName: funcDecl.Name.Name,
		FuncDecl: funcDecl,
	}

	// Parse elements passed to wire.Build (same as wire.NewSet elements)
	for _, arg := range call.Args {
		elem := p.parseSetElement(arg, info, wireAlias, filePath)
		if elem != nil {
			build.Elements = append(build.Elements, elem)
		}
	}

	// Extract return types from the function signature
	// Note: Each field may have multiple names sharing the same type (e.g., "(a, b *App)")
	if funcDecl.Type.Results != nil {
		for _, field := range funcDecl.Type.Results.List {
			if tv, ok := info.Types[field.Type]; ok {
				// If field has names, add the type once for each name
				// If no names (anonymous return), add the type once
				count := len(field.Names)
				if count == 0 {
					count = 1
				}
				for i := 0; i < count; i++ {
					build.ReturnTypes = append(build.ReturnTypes, tv.Type)
				}
			}
		}
	}

	return build
}

// extractTypeFromNew extracts the type from new(T) expression.
func (p *Parser) extractTypeFromNew(expr ast.Expr, info *types.Info) types.Type {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}

	ident, ok := call.Fun.(*ast.Ident)
	if !ok || ident.Name != "new" {
		return nil
	}

	if len(call.Args) != 1 {
		return nil
	}

	if tv, ok := info.Types[call.Args[0]]; ok {
		return types.NewPointer(tv.Type)
	}

	return nil
}
