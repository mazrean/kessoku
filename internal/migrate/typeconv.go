package migrate

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

// TypeConverter handles conversion of types.Type to ast.Expr with proper package qualifiers.
// It tracks which imports are needed for external types.
type TypeConverter struct {
	currentPkg    *types.Package
	imports       map[string]string // package path -> local name
	usedNames     map[string]string // local name -> package path (for collision detection)
	nameCounters  map[string]int    // base name -> counter for generating unique names
	explicitAlias map[string]bool   // package path -> true if alias was explicitly provided by source
}

// NewTypeConverter creates a new TypeConverter for the given package.
func NewTypeConverter(currentPkg *types.Package) *TypeConverter {
	return &TypeConverter{
		currentPkg:    currentPkg,
		imports:       make(map[string]string),
		usedNames:     make(map[string]string),
		nameCounters:  make(map[string]int),
		explicitAlias: make(map[string]bool),
	}
}

// Imports returns the collected import specifications needed for the generated code.
// Each path appears exactly once (guaranteed by the map structure).
func (tc *TypeConverter) Imports() []ImportSpec {
	// Use a map to deduplicate by path (should already be unique, but ensure)
	seen := make(map[string]bool)
	var specs []ImportSpec
	for path, name := range tc.imports {
		if seen[path] {
			// Should never happen since tc.imports is a map, but log if it does
			continue
		}
		seen[path] = true
		spec := ImportSpec{Path: path}
		// Omit the alias only when:
		//   1. The alias was NOT explicitly provided by the source (i.e. it was derived
		//      from the package's declared name via types.Package.Name()), AND
		//   2. The alias equals lastPathElement(path) — meaning Go would resolve the
		//      unaliased import to the same identifier anyway.
		//
		// When the alias WAS explicitly specified in the source import (e.g.
		// `v2 "example.com/bar/v2"`), we must always emit it: Go resolves an
		// unaliased import using the package's `package` declaration ("bar"), not the
		// last path element ("v2"), so suppressing the alias would produce
		// uncompilable code that references an undefined identifier.
		pkgName := lastPathElement(path)
		if !tc.explicitAlias[path] && name == pkgName {
			// Safe to omit: Go will resolve the import to the same name.
		} else {
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
// It uses sourceImports to map package names to import paths, and explicitAliasPaths
// to identify which import paths had an explicit alias written in the source.
// It also renames package references in the expression if there are name collisions.
func (tc *TypeConverter) CollectExprImports(expr ast.Expr, sourceImports map[string]string, explicitAliasPaths map[string]bool) {
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
			// If the source declared an explicit alias for this import path, mark it so
			// that Imports() always emits the alias.  Go resolves an unaliased import using
			// the package's declared name (not lastPathElement), so for paths like
			// `v2 "example.com/bar/v2"` where the package declares `package bar`, omitting
			// the alias would produce uncompilable code referencing undefined identifier "v2".
			if explicitAliasPaths[importPath] {
				tc.explicitAlias[importPath] = true
			}
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
func (tc *TypeConverter) CollectPatternImports(p KessokuPattern, sourceImports map[string]string, explicitAliasPaths map[string]bool) {
	if p == nil {
		return
	}
	switch kp := p.(type) {
	case *KessokuSet:
		for _, elem := range kp.Elements {
			tc.CollectPatternImports(elem, sourceImports, explicitAliasPaths)
		}
	case *KessokuProvide:
		tc.CollectExprImports(kp.FuncExpr, sourceImports, explicitAliasPaths)
	case *KessokuBind:
		tc.CollectPatternImports(kp.Provider, sourceImports, explicitAliasPaths)
	case *KessokuValue:
		tc.CollectExprImports(kp.Expr, sourceImports, explicitAliasPaths)
	case *KessokuSetRef:
		tc.CollectExprImports(kp.Expr, sourceImports, explicitAliasPaths)
	case *KessokuInject:
		for _, elem := range kp.Elements {
			tc.CollectPatternImports(elem, sourceImports, explicitAliasPaths)
		}
	}
}

// TypeToExpr converts a types.Type to an ast.Expr with proper package qualifiers.
// For types from external packages, it adds the necessary import and generates
// a SelectorExpr with the package qualifier.
func (tc *TypeConverter) TypeToExpr(t types.Type) ast.Expr {
	if t == nil {
		return nil
	}
	switch typ := t.(type) {
	case *types.Named:
		obj := typ.Obj()
		if obj.Pkg() == nil {
			// Built-in type (e.g., error)
			return wrapTypeArgs(ast.NewIdent(obj.Name()), typ.TypeArgs(), tc.TypeToExpr)
		}
		// Check if this type is from an external package
		var baseExpr ast.Expr
		if tc.currentPkg != nil && obj.Pkg() != tc.currentPkg {
			// External package - add import and generate SelectorExpr
			pkgPath := obj.Pkg().Path()
			pkgName := obj.Pkg().Name()
			actualName := tc.AddImport(pkgPath, pkgName)
			baseExpr = &ast.SelectorExpr{
				X:   ast.NewIdent(actualName),
				Sel: ast.NewIdent(obj.Name()),
			}
		} else {
			// Same package - just use the type name
			baseExpr = ast.NewIdent(obj.Name())
		}
		return wrapTypeArgs(baseExpr, typ.TypeArgs(), tc.TypeToExpr)
	case *types.Pointer:
		return &ast.StarExpr{X: tc.TypeToExpr(typ.Elem())}
	case *types.Slice:
		return &ast.ArrayType{Elt: tc.TypeToExpr(typ.Elem())}
	case *types.Map:
		return &ast.MapType{
			Key:   tc.TypeToExpr(typ.Key()),
			Value: tc.TypeToExpr(typ.Elem()),
		}
	case *types.Basic:
		if typ.Kind() == types.UnsafePointer {
			tc.AddImport("unsafe", "unsafe")
			return &ast.SelectorExpr{X: ast.NewIdent("unsafe"), Sel: ast.NewIdent("Pointer")}
		}
		return ast.NewIdent(typ.Name())
	case *types.Interface:
		if typ.Empty() {
			return &ast.InterfaceType{Methods: &ast.FieldList{}}
		}
		return ast.NewIdent("any")
	case *types.Array:
		return &ast.ArrayType{
			Len: &ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", typ.Len())},
			Elt: tc.TypeToExpr(typ.Elem()),
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
		return &ast.ChanType{
			Dir:   dir,
			Value: tc.TypeToExpr(typ.Elem()),
		}
	default:
		return ast.NewIdent(t.String())
	}
}

// wrapTypeArgs wraps base with type argument indices if typeArgs is non-empty.
// For a single type argument, it returns ast.IndexExpr{X: base, Index: arg}.
// For multiple type arguments, it returns ast.IndexListExpr{X: base, Indices: args}.
// The conv function converts each types.Type argument to ast.Expr.
func wrapTypeArgs(base ast.Expr, typeArgs *types.TypeList, conv func(types.Type) ast.Expr) ast.Expr {
	if typeArgs == nil || typeArgs.Len() == 0 {
		return base
	}
	if typeArgs.Len() == 1 {
		return &ast.IndexExpr{
			X:     base,
			Index: conv(typeArgs.At(0)),
		}
	}
	indices := make([]ast.Expr, typeArgs.Len())
	for i := range typeArgs.Len() {
		indices[i] = conv(typeArgs.At(i))
	}
	return &ast.IndexListExpr{
		X:       base,
		Indices: indices,
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
