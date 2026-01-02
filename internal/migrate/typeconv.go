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
		// Only set name (alias) if it differs from the last element of the path.
		// This avoids redundant aliases like: v1 "github.com/.../v1"
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
			return ast.NewIdent(obj.Name())
		}
		// Check if this type is from an external package
		if tc.currentPkg != nil && obj.Pkg() != tc.currentPkg {
			// External package - add import and generate SelectorExpr
			pkgPath := obj.Pkg().Path()
			pkgName := obj.Pkg().Name()
			actualName := tc.AddImport(pkgPath, pkgName)
			return &ast.SelectorExpr{
				X:   ast.NewIdent(actualName),
				Sel: ast.NewIdent(obj.Name()),
			}
		}
		// Same package - just use the type name
		return ast.NewIdent(obj.Name())
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

// lastPathElement returns the last element of an import path.
func lastPathElement(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}
