package migrate

import (
	"go/ast"
	"go/types"
)

// transformBind transforms wire.Bind to kessoku.Bind.
// elements is the sibling element list of the current set/build; it is used to
// find the provider for the concrete type when it lives in the same element list
// (as a WireProviderFunc) rather than being looked up by the "New+TypeName" convention.
// setIndex maps WireNewSet variable names to their parsed WireNewSet; it lets us
// determine what a WireSetRef contributes so we can avoid duplicate providers.
func (t *Transformer) transformBind(wb *WireBind, pkg *types.Package, elements []WirePattern) (*KessokuBind, error) {
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
			Message: "implementation type must be a named type",
		}
	}

	implPkg := named.Obj().Pkg()

	// Step 1: look for the provider in the sibling elements of the current set.
	// This handles both "New+TypeName" and any other naming convention.
	var constructor *types.Func
	for _, elem := range elements {
		wpf, ok := elem.(*WireProviderFunc)
		if !ok || wpf.Func == nil {
			continue
		}
		sig, ok := wpf.Func.Type().(*types.Signature)
		if !ok || sig.Results().Len() == 0 {
			continue
		}
		retType := sig.Results().At(0).Type()
		// Match: the concrete type (unwrapped from pointer) of the first return value.
		retUnwrapped := retType
		for {
			if ptr, ok := retUnwrapped.(*types.Pointer); ok {
				retUnwrapped = ptr.Elem()
			} else {
				break
			}
		}
		if types.Identical(retUnwrapped, named) || types.Identical(retType, wb.Implementation) {
			constructor = wpf.Func
			break
		}
	}

	// Step 2: if not found in the sibling elements, fall back to searching the
	// package scope for any function whose first return type matches the impl type.
	if constructor == nil && implPkg != nil {
		scope := implPkg.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			fn, ok := obj.(*types.Func)
			if !ok {
				continue
			}
			sig, ok := fn.Type().(*types.Signature)
			if !ok || sig.Results().Len() == 0 {
				continue
			}
			retType := sig.Results().At(0).Type()
			retUnwrapped := retType
			for {
				if ptr, ok := retUnwrapped.(*types.Pointer); ok {
					retUnwrapped = ptr.Elem()
				} else {
					break
				}
			}
			if types.Identical(retUnwrapped, named) || types.Identical(retType, wb.Implementation) {
				constructor = fn
				break
			}
		}
	}

	if constructor == nil {
		return nil, &ParseError{
			Kind:    ParseErrorMissingConstructor,
			File:    wb.File,
			Pos:     wb.Pos,
			Message: "no constructor found for type \"" + named.Obj().Name() + "\"",
		}
	}

	// Build the constructor expression from the resolved *types.Func.
	// Always build a fresh expression so that original source positions do not
	// interfere with the writer's synthetic position system.
	constructorName := constructor.Name()
	var constructorExpr ast.Expr
	if implPkg != nil && implPkg != pkg {
		// External package — use selector with proper import handling.
		pkgName := implPkg.Name()
		if t.tc != nil {
			pkgName = t.tc.AddImport(implPkg.Path(), implPkg.Name())
		}
		constructorExpr = &ast.SelectorExpr{
			X:   ast.NewIdent(pkgName),
			Sel: ast.NewIdent(constructorName),
		}
	} else {
		constructorExpr = ast.NewIdent(constructorName)
	}

	return &KessokuBind{
		Interface: unwrapPointer(wb.Interface),
		Provider: &KessokuProvide{
			FuncExpr:  constructorExpr,
			SourcePos: wb.Pos,
		},
		SourcePos: wb.Pos,
	}, nil
}
