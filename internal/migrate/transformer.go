package migrate

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"unicode"
)

// maxInjectorReturns is the maximum number of return values for an injector function.
// An injector can return at most 2 values: the injected type and optionally an error.
const maxInjectorReturns = 2

// TypeConverter handles conversion of types.Type to ast.Expr with proper package qualifiers.
// It tracks which imports are needed for external types.
type TypeConverter struct {
	currentPkg   *types.Package
	imports      map[string]string // package path -> local name
	usedNames    map[string]string // local name -> package path (for collision detection)
	nameCounters map[string]int    // base name -> counter for generating unique names
}

// NewTypeConverter creates a new TypeConverter for the given package.
func NewTypeConverter(currentPkg *types.Package) *TypeConverter {
	return &TypeConverter{
		currentPkg:   currentPkg,
		imports:      make(map[string]string),
		usedNames:    make(map[string]string),
		nameCounters: make(map[string]int),
	}
}

// Imports returns the collected import specifications needed for the generated code.
func (tc *TypeConverter) Imports() []ImportSpec {
	var specs []ImportSpec
	for path, name := range tc.imports {
		spec := ImportSpec{Path: path}
		// Only set name if it differs from the last element of the path
		pkgName := lastPathElement(path)
		if name != pkgName {
			spec.Name = name
		}
		specs = append(specs, spec)
	}
	return specs
}

// AddImport adds an import to the collected imports, handling name collisions.
// Returns the actual name to use for this import path.
func (tc *TypeConverter) AddImport(path, desiredName string) string {
	// If already imported, return the existing name
	if existingName, exists := tc.imports[path]; exists {
		return existingName
	}

	// Check if the desired name is already used by a different path
	if existingPath, exists := tc.usedNames[desiredName]; exists && existingPath != path {
		// Name collision - generate a unique name
		baseName := desiredName
		counter := tc.nameCounters[baseName]
		for {
			counter++
			newName := fmt.Sprintf("%s_%d", baseName, counter)
			if _, used := tc.usedNames[newName]; !used {
				tc.nameCounters[baseName] = counter
				tc.imports[path] = newName
				tc.usedNames[newName] = path
				return newName
			}
		}
	}

	// No collision - use desired name
	tc.imports[path] = desiredName
	tc.usedNames[desiredName] = path
	return desiredName
}

// CollectExprImports walks an AST expression and collects package references.
// It uses sourceImports to map package names to import paths.
// It also renames package references in the expression if there are name collisions.
func (tc *TypeConverter) CollectExprImports(expr ast.Expr, sourceImports map[string]string) {
	if expr == nil {
		return
	}
	ast.Inspect(expr, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		// Check if X is an identifier (package reference)
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		// Look up the package name in source imports
		pkgName := ident.Name
		if importPath, exists := sourceImports[pkgName]; exists {
			// Add import and get the actual name (may be renamed due to collision)
			actualName := tc.AddImport(importPath, pkgName)
			// Update the identifier if it was renamed
			if actualName != pkgName {
				ident.Name = actualName
			}
		}
		return true
	})
}

// CollectPatternImports collects imports from all expressions in a pattern.
func (tc *TypeConverter) CollectPatternImports(p KessokuPattern, sourceImports map[string]string) {
	if p == nil {
		return
	}
	switch kp := p.(type) {
	case *KessokuSet:
		for _, elem := range kp.Elements {
			tc.CollectPatternImports(elem, sourceImports)
		}
	case *KessokuProvide:
		tc.CollectExprImports(kp.FuncExpr, sourceImports)
	case *KessokuBind:
		tc.CollectPatternImports(kp.Provider, sourceImports)
	case *KessokuValue:
		tc.CollectExprImports(kp.Expr, sourceImports)
	case *KessokuSetRef:
		tc.CollectExprImports(kp.Expr, sourceImports)
	case *KessokuInject:
		for _, elem := range kp.Elements {
			tc.CollectPatternImports(elem, sourceImports)
		}
	}
}

// lastPathElement returns the last element of an import path.
func lastPathElement(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

// TypeToExpr converts a types.Type to an ast.Expr, using package qualifiers for external types.
func (tc *TypeConverter) TypeToExpr(t types.Type) ast.Expr {
	if t == nil {
		return nil
	}
	switch typ := t.(type) {
	case *types.Named:
		obj := typ.Obj()
		pkg := obj.Pkg()
		if pkg == nil {
			// Built-in type (e.g., error)
			return ast.NewIdent(obj.Name())
		}
		// Check if it's from the current package
		if tc.currentPkg != nil && pkg.Path() == tc.currentPkg.Path() {
			return ast.NewIdent(obj.Name())
		}
		// External type - use package qualifier
		pkgName := pkg.Name()
		tc.imports[pkg.Path()] = pkgName
		return &ast.SelectorExpr{
			X:   ast.NewIdent(pkgName),
			Sel: ast.NewIdent(obj.Name()),
		}
	case *types.Pointer:
		elem := tc.TypeToExpr(typ.Elem())
		if elem == nil {
			return nil
		}
		return &ast.StarExpr{X: elem}
	case *types.Slice:
		elem := tc.TypeToExpr(typ.Elem())
		if elem == nil {
			return nil
		}
		return &ast.ArrayType{Elt: elem}
	case *types.Map:
		key := tc.TypeToExpr(typ.Key())
		val := tc.TypeToExpr(typ.Elem())
		if key == nil || val == nil {
			return nil
		}
		return &ast.MapType{Key: key, Value: val}
	case *types.Basic:
		return ast.NewIdent(typ.Name())
	case *types.Interface:
		if typ.Empty() {
			return &ast.InterfaceType{Methods: &ast.FieldList{}}
		}
		return ast.NewIdent("any")
	case *types.Array:
		elem := tc.TypeToExpr(typ.Elem())
		if elem == nil {
			return nil
		}
		return &ast.ArrayType{
			Len: &ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", typ.Len())},
			Elt: elem,
		}
	case *types.Chan:
		elem := tc.TypeToExpr(typ.Elem())
		if elem == nil {
			return nil
		}
		dir := ast.SEND | ast.RECV
		switch typ.Dir() {
		case types.SendRecv:
			dir = ast.SEND | ast.RECV
		case types.SendOnly:
			dir = ast.SEND
		case types.RecvOnly:
			dir = ast.RECV
		}
		return &ast.ChanType{Dir: dir, Value: elem}
	default:
		return ast.NewIdent(t.String())
	}
}

// Transformer converts wire patterns to kessoku patterns.
type Transformer struct{}

// NewTransformer creates a new Transformer instance.
func NewTransformer() *Transformer {
	return &Transformer{}
}

// Transform transforms a list of wire patterns to kessoku patterns.
func (t *Transformer) Transform(patterns []WirePattern, pkg *types.Package) ([]KessokuPattern, error) {
	var result []KessokuPattern

	for _, p := range patterns {
		switch wp := p.(type) {
		case *WireNewSet:
			transformed, err := t.transformNewSet(wp, pkg)
			if err != nil {
				return nil, err
			}
			result = append(result, transformed)
		case *WireBind:
			transformed, err := t.transformBind(wp, pkg)
			if err != nil {
				return nil, err
			}
			result = append(result, transformed)
		case *WireValue:
			result = append(result, t.transformValue(wp))
		case *WireInterfaceValue:
			result = append(result, t.transformInterfaceValue(wp))
		case *WireStruct:
			result = append(result, t.transformStruct(wp, pkg))
		case *WireFieldsOf:
			result = append(result, t.transformFieldsOf(wp, pkg))
		case *WireProviderFunc:
			result = append(result, t.transformProviderFunc(wp))
		case *WireSetRef:
			result = append(result, t.transformSetRef(wp))
		case *WireBuild:
			transformed, err := t.transformBuild(wp, pkg)
			if err != nil {
				return nil, err
			}
			result = append(result, transformed)
		}
	}

	return result, nil
}

// transformNewSet transforms wire.NewSet to kessoku.Set.
func (t *Transformer) transformNewSet(ws *WireNewSet, pkg *types.Package) (*KessokuSet, error) {
	var elements []KessokuPattern

	for _, elem := range ws.Elements {
		switch we := elem.(type) {
		case *WireNewSet:
			// Handle inline nested wire.NewSet
			nestedSet, err := t.transformNewSet(we, pkg)
			if err != nil {
				return nil, err
			}
			// Flatten nested set elements into parent
			elements = append(elements, nestedSet.Elements...)
		case *WireBind:
			transformed, err := t.transformBind(we, pkg)
			if err != nil {
				return nil, err
			}
			elements = append(elements, transformed)
		case *WireValue:
			elements = append(elements, t.transformValue(we))
		case *WireInterfaceValue:
			elements = append(elements, t.transformInterfaceValue(we))
		case *WireStruct:
			elements = append(elements, t.transformStruct(we, pkg))
		case *WireFieldsOf:
			elements = append(elements, t.transformFieldsOf(we, pkg))
		case *WireProviderFunc:
			elements = append(elements, t.transformProviderFunc(we))
		case *WireSetRef:
			elements = append(elements, t.transformSetRef(we))
		}
	}

	return &KessokuSet{
		VarName:   ws.VarName,
		Elements:  elements,
		SourcePos: ws.Pos,
	}, nil
}

// transformProviderFunc transforms a provider function to kessoku.Provide.
func (t *Transformer) transformProviderFunc(wf *WireProviderFunc) *KessokuProvide {
	return &KessokuProvide{
		FuncExpr:  wf.Expr,
		SourcePos: wf.Pos,
	}
}

// transformSetRef transforms a set reference.
func (t *Transformer) transformSetRef(ws *WireSetRef) *KessokuSetRef {
	return &KessokuSetRef{
		Name:      ws.Name,
		Expr:      ws.Expr,
		SourcePos: ws.Pos,
	}
}

// transformBuild transforms wire.Build to kessoku.Inject.
func (t *Transformer) transformBuild(wb *WireBuild, pkg *types.Package) (*KessokuInject, error) {
	// Validate return signature - injector must have exactly 1 or 2 return values
	// (the injected type, optionally followed by error)
	numReturns := len(wb.ReturnTypes)
	if numReturns == 0 {
		return nil, &ParseError{
			Kind:    ParseErrorMissingConstructor,
			File:    wb.File,
			Pos:     wb.Pos,
			Message: fmt.Sprintf("injector function %q must have at least one return value", wb.FuncName),
		}
	}
	if numReturns > maxInjectorReturns {
		return nil, &ParseError{
			Kind:    ParseErrorMissingConstructor,
			File:    wb.File,
			Pos:     wb.Pos,
			Message: fmt.Sprintf("injector function %q has %d return values, expected 1 or %d", wb.FuncName, numReturns, maxInjectorReturns),
		}
	}

	// For 2-return functions, the second must be error
	hasError := false
	if numReturns == maxInjectorReturns {
		if !isErrorType(wb.ReturnTypes[1]) {
			return nil, &ParseError{
				Kind:    ParseErrorMissingConstructor,
				File:    wb.File,
				Pos:     wb.Pos,
				Message: fmt.Sprintf("injector function %q has 2 return values but second is not error (got %s)", wb.FuncName, wb.ReturnTypes[1]),
			}
		}
		hasError = true
	}

	inject := &KessokuInject{
		FuncName:   wb.FuncName,
		FuncDecl:   wb.FuncDecl,
		ReturnType: wb.ReturnTypes[0],
		HasError:   hasError,
		SourcePos:  wb.Pos,
	}

	// Transform elements (same as NewSet elements)
	for _, elem := range wb.Elements {
		switch we := elem.(type) {
		case *WireNewSet:
			// Inline nested NewSet - flatten elements
			nestedSet, err := t.transformNewSet(we, pkg)
			if err != nil {
				return nil, err
			}
			inject.Elements = append(inject.Elements, nestedSet.Elements...)
		case *WireBind:
			transformed, err := t.transformBind(we, pkg)
			if err != nil {
				return nil, err
			}
			inject.Elements = append(inject.Elements, transformed)
		case *WireValue:
			inject.Elements = append(inject.Elements, t.transformValue(we))
		case *WireInterfaceValue:
			inject.Elements = append(inject.Elements, t.transformInterfaceValue(we))
		case *WireStruct:
			inject.Elements = append(inject.Elements, t.transformStruct(we, pkg))
		case *WireFieldsOf:
			inject.Elements = append(inject.Elements, t.transformFieldsOf(we, pkg))
		case *WireProviderFunc:
			inject.Elements = append(inject.Elements, t.transformProviderFunc(we))
		case *WireSetRef:
			inject.Elements = append(inject.Elements, t.transformSetRef(we))
		}
	}

	return inject, nil
}

// isErrorType checks if a type is the built-in error type.
func isErrorType(t types.Type) bool {
	if t == nil {
		return false
	}

	// Get the predeclared error type from Universe
	errorType := types.Universe.Lookup("error").Type()

	// Compare directly with the predeclared error type
	if types.Identical(t, errorType) {
		return true
	}

	// Handle type aliases by checking the underlying type
	underlying := t.Underlying()
	if underlying != nil && types.Identical(underlying, errorType.Underlying()) {
		return true
	}

	return false
}

// transformBind transforms wire.Bind to kessoku.Bind.
func (t *Transformer) transformBind(wb *WireBind, pkg *types.Package) (*KessokuBind, error) {
	// Unwrap pointer types to get the base named type
	implType := wb.Implementation
	for {
		if ptr, ok := implType.(*types.Pointer); ok {
			implType = ptr.Elem()
		} else {
			break
		}
	}

	named, ok := implType.(*types.Named)
	if !ok {
		return nil, &ParseError{
			Kind:    ParseErrorMissingConstructor,
			File:    wb.File,
			Pos:     wb.Pos,
			Message: fmt.Sprintf("implementation type must be a named type, got %T", implType),
		}
	}

	typeName := named.Obj().Name()
	implPkg := named.Obj().Pkg()

	// Look up constructor in package scope
	constructorName := "New" + typeName

	var constructor *types.Func
	if implPkg != nil {
		obj := implPkg.Scope().Lookup(constructorName)
		if obj != nil {
			if fn, ok := obj.(*types.Func); ok {
				constructor = fn
			}
		}
	}

	if constructor == nil {
		return nil, &ParseError{
			Kind:    ParseErrorMissingConstructor,
			File:    wb.File,
			Pos:     wb.Pos,
			Message: fmt.Sprintf("no constructor %q found for type %q", constructorName, typeName),
		}
	}

	// Build the constructor expression
	var funcExpr ast.Expr
	if implPkg != nil && implPkg != pkg {
		// External package - use selector
		funcExpr = &ast.SelectorExpr{
			X:   ast.NewIdent(implPkg.Name()),
			Sel: ast.NewIdent(constructorName),
		}
	} else {
		funcExpr = ast.NewIdent(constructorName)
	}

	return &KessokuBind{
		Interface: unwrapPointer(wb.Interface),
		Provider: &KessokuProvide{
			FuncExpr:  funcExpr,
			SourcePos: wb.Pos,
		},
		SourcePos: wb.Pos,
	}, nil
}

// transformValue transforms wire.Value to kessoku.Value.
func (t *Transformer) transformValue(wv *WireValue) *KessokuValue {
	return &KessokuValue{
		Expr:      wv.Expr,
		SourcePos: wv.Pos,
	}
}

// transformInterfaceValue transforms wire.InterfaceValue to kessoku.Bind + kessoku.Value.
func (t *Transformer) transformInterfaceValue(wiv *WireInterfaceValue) *KessokuBind {
	return &KessokuBind{
		Interface: unwrapPointer(wiv.Interface),
		Provider: &KessokuValue{
			Expr:      wiv.Expr,
			SourcePos: wiv.Pos,
		},
		SourcePos: wiv.Pos,
	}
}

// transformStruct transforms wire.Struct to kessoku.Provide with function literal.
func (t *Transformer) transformStruct(ws *WireStruct, pkg *types.Package) *KessokuProvide {
	structType := unwrapPointer(ws.StructType)
	underlying := structType.Underlying()
	st, ok := underlying.(*types.Struct)
	if !ok {
		return &KessokuProvide{SourcePos: ws.Pos}
	}

	// Check if struct is from external package
	isExternalPkg := false
	if named, ok := structType.(*types.Named); ok {
		if named.Obj().Pkg() != nil && named.Obj().Pkg() != pkg {
			isExternalPkg = true
		}
	}

	// Collect fields to include (skip unexported fields from external packages)
	var fieldInfos []fieldInfo
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if ws.Fields[0] == "*" || contains(ws.Fields, field.Name()) {
			// Skip unexported fields from external packages
			if isExternalPkg && !field.Exported() {
				continue
			}
			fieldInfos = append(fieldInfos, fieldInfo{
				name:     field.Name(),
				typ:      field.Type(),
				exported: field.Exported(),
			})
		}
	}

	// Build function literal
	funcLit := t.buildStructConstructor(structType, fieldInfos, ws.IsPointer)

	return &KessokuProvide{
		FuncExpr:  funcLit,
		SourcePos: ws.Pos,
	}
}

// transformFieldsOf transforms wire.FieldsOf to kessoku.Provide with accessor function.
func (t *Transformer) transformFieldsOf(wf *WireFieldsOf, pkg *types.Package) *KessokuProvide {
	structType := unwrapPointer(wf.StructType)
	underlying := structType.Underlying()
	st, ok := underlying.(*types.Struct)
	if !ok {
		return &KessokuProvide{SourcePos: wf.Pos}
	}

	// Check if struct is from external package
	isExternalPkg := false
	if named, ok := structType.(*types.Named); ok {
		if named.Obj().Pkg() != nil && named.Obj().Pkg() != pkg {
			isExternalPkg = true
		}
	}

	// Collect field types (skip unexported fields from external packages)
	var fieldInfos []fieldInfo
	for _, fieldName := range wf.Fields {
		for i := 0; i < st.NumFields(); i++ {
			field := st.Field(i)
			if field.Name() == fieldName {
				// Skip unexported fields from external packages
				if isExternalPkg && !field.Exported() {
					break
				}
				fieldInfos = append(fieldInfos, fieldInfo{
					name:     field.Name(),
					typ:      field.Type(),
					exported: field.Exported(),
				})
				break
			}
		}
	}

	// Build accessor function
	funcLit := t.buildFieldAccessor(structType, fieldInfos)

	return &KessokuProvide{
		FuncExpr:  funcLit,
		SourcePos: wf.Pos,
	}
}

type fieldInfo struct {
	typ      types.Type
	name     string
	exported bool
}

// buildStructConstructor builds a function literal for struct construction.
func (t *Transformer) buildStructConstructor(structType types.Type, fields []fieldInfo, isPointer bool) *ast.FuncLit {
	// Build parameter list
	var params []*ast.Field
	var paramNames []string
	for _, f := range fields {
		paramName := toLowerCamel(f.name)
		paramNames = append(paramNames, paramName)
		params = append(params, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(paramName)},
			Type:  typeToExpr(f.typ),
		})
	}

	// Build struct literal
	var elts []ast.Expr
	for i, f := range fields {
		elts = append(elts, &ast.KeyValueExpr{
			Key:   ast.NewIdent(f.name),
			Value: ast.NewIdent(paramNames[i]),
		})
	}

	structLit := &ast.CompositeLit{
		Type: typeToExpr(structType),
		Elts: elts,
	}

	returnExpr := ast.Expr(structLit)
	returnType := typeToExpr(structType)

	if isPointer {
		returnExpr = &ast.UnaryExpr{
			Op: token.AND,
			X:  structLit,
		}
		returnType = &ast.StarExpr{X: typeToExpr(structType)}
	}

	return &ast.FuncLit{
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: params},
			Results: &ast.FieldList{
				List: []*ast.Field{{Type: returnType}},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{returnExpr},
				},
			},
		},
	}
}

// buildFieldAccessor builds a function literal for field extraction.
func (t *Transformer) buildFieldAccessor(structType types.Type, fields []fieldInfo) *ast.FuncLit {
	// Parameter: pointer to struct
	paramName := "s"
	param := &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(paramName)},
		Type:  &ast.StarExpr{X: typeToExpr(structType)},
	}

	// Return types
	var resultTypes []*ast.Field
	var returnExprs []ast.Expr
	for _, f := range fields {
		resultTypes = append(resultTypes, &ast.Field{Type: typeToExpr(f.typ)})
		returnExprs = append(returnExprs, &ast.SelectorExpr{
			X:   ast.NewIdent(paramName),
			Sel: ast.NewIdent(f.name),
		})
	}

	return &ast.FuncLit{
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: []*ast.Field{param}},
			Results: &ast.FieldList{List: resultTypes},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{Results: returnExprs},
			},
		},
	}
}

// Helper functions

func unwrapPointer(t types.Type) types.Type {
	if ptr, ok := t.(*types.Pointer); ok {
		return ptr.Elem()
	}
	return t
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func toLowerCamel(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func typeToExpr(t types.Type) ast.Expr {
	if t == nil {
		return nil
	}
	switch typ := t.(type) {
	case *types.Named:
		obj := typ.Obj()
		if obj.Pkg() == nil {
			// Built-in type (e.g., error)
			return ast.NewIdent(obj.Name())
		}
		// Note: For cross-package types, this loses the package qualifier.
		// In practice, wire migrations typically use same-package types.
		// TODO: Add support for cross-package types by tracking current package
		// and generating SelectorExpr for external types.
		return ast.NewIdent(obj.Name())
	case *types.Pointer:
		return &ast.StarExpr{X: typeToExpr(typ.Elem())}
	case *types.Slice:
		return &ast.ArrayType{Elt: typeToExpr(typ.Elem())}
	case *types.Map:
		return &ast.MapType{
			Key:   typeToExpr(typ.Key()),
			Value: typeToExpr(typ.Elem()),
		}
	case *types.Basic:
		return ast.NewIdent(typ.Name())
	case *types.Interface:
		// Empty interface: interface{}
		if typ.Empty() {
			return &ast.InterfaceType{Methods: &ast.FieldList{}}
		}
		// Non-empty anonymous interfaces are rare in wire patterns.
		// Named interfaces (io.Reader, etc.) are handled by *types.Named.
		// For anonymous non-empty interfaces, use any as a fallback.
		return ast.NewIdent("any")
	case *types.Array:
		return &ast.ArrayType{
			Len: &ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", typ.Len())},
			Elt: typeToExpr(typ.Elem()),
		}
	case *types.Chan:
		dir := ast.SEND | ast.RECV
		switch typ.Dir() {
		case types.SendRecv:
			dir = ast.SEND | ast.RECV
		case types.SendOnly:
			dir = ast.SEND
		case types.RecvOnly:
			dir = ast.RECV
		}
		return &ast.ChanType{Dir: dir, Value: typeToExpr(typ.Elem())}
	default:
		return ast.NewIdent(t.String())
	}
}
