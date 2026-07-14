package migrate

import (
	"go/ast"
	"go/types"
	"strings"
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
	// This handles both "New+TypeName" and any other naming convention, as well
	// as wire.Struct siblings which produce anonymous function literals rather
	// than named constructor functions.
	var constructor *types.Func
	for _, elem := range elements {
		// Handle wire.Struct siblings: transformStruct synthesises an anonymous
		// *ast.FuncLit, so there is no *types.Func to look up.  Detect the match
		// directly from the struct type and short-circuit to return KessokuBind.
		if ws, ok := elem.(*WireStruct); ok {
			// ws.StructType comes from extractTypeFromNew which adds one pointer
			// layer; unwrap it to reach the plain named type.
			wsBase := unwrapPointer(ws.StructType)
			for {
				if ptr, ok := wsBase.(*types.Pointer); ok {
					wsBase = ptr.Elem()
				} else {
					break
				}
			}
			if !types.Identical(wsBase, named) {
				continue
			}
			// The bound implementation type determines which provider variant to use.
			// wire.Bind(new(I), new(*T)) -> use the pointer-returning FuncLit.
			// wire.Bind(new(I), new(T))  -> use the value-returning FuncLit.
			_, implIsPointer := wb.Implementation.(*types.Pointer)
			patterns := t.transformStruct(ws, pkg)
			// transformStruct returns [valueFuncLit, ptrFuncLit] when !IsPointer,
			// and [ptrFuncLit] when IsPointer.  We need the pointer variant when
			// the implementation type is a pointer, value variant otherwise.
			var providerExpr ast.Expr
			switch {
			case ws.IsPointer:
				// Only one pattern (pointer variant) was produced.
				if len(patterns) > 0 {
					if kp, ok := patterns[0].(*KessokuProvide); ok {
						providerExpr = kp.FuncExpr
					}
				}
			case implIsPointer:
				// Value and pointer patterns produced; pick the pointer one (index 1).
				if len(patterns) > 1 {
					if kp, ok := patterns[1].(*KessokuProvide); ok {
						providerExpr = kp.FuncExpr
					}
				}
			default:
				// Value variant (index 0).
				if len(patterns) > 0 {
					if kp, ok := patterns[0].(*KessokuProvide); ok {
						providerExpr = kp.FuncExpr
					}
				}
			}
			if providerExpr == nil {
				break
			}
			return &KessokuBind{
				Interface: unwrapPointer(wb.Interface),
				Provider: &KessokuProvide{
					FuncExpr:  providerExpr,
					SourcePos: wb.Pos,
				},
				VarName:   wb.VarName,
				SourcePos: wb.Pos,
			}, nil
		}

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
	// If multiple candidates are found we cannot pick one deterministically, so we
	// return an error instead of silently choosing the alphabetically-first name.
	if constructor == nil && implPkg != nil {
		scope := implPkg.Scope()
		var candidates []*types.Func
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
				candidates = append(candidates, fn)
			}
		}
		if len(candidates) == 1 {
			constructor = candidates[0]
		} else if len(candidates) > 1 {
			names := make([]string, len(candidates))
			for i, fn := range candidates {
				names[i] = fn.Name()
			}
			return nil, &ParseError{
				Kind:    ParseErrorMissingConstructor,
				File:    wb.File,
				Pos:     wb.Pos,
				Message: "ambiguous constructor for type \"" + named.Obj().Name() + "\": multiple candidates found in package scope (" + joinNames(names) + "); specify the constructor explicitly in wire.NewSet",
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
	// Use the constructor's own package, not the impl type's package.
	// When Step 1 resolves a provider from a sibling element (e.g. factory.NewDB),
	// the constructor may live in a completely different package than the impl type
	// (e.g. impl.DB). Using implPkg here was the root cause of the bug.
	constructorPkg := constructor.Pkg()
	var constructorExpr ast.Expr
	if constructorPkg != nil && constructorPkg != pkg {
		// External package — use selector with proper import handling.
		pkgName := constructorPkg.Name()
		if t.tc != nil {
			pkgName = t.tc.AddImport(constructorPkg.Path(), constructorPkg.Name())
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
		VarName:   wb.VarName,
		SourcePos: wb.Pos,
	}, nil
}

// joinNames joins a slice of function names with ", " for use in error messages.
func joinNames(names []string) string {
	return strings.Join(names, ", ")
}

// buildBindVarTypes builds a map from top-level WireBind variable names to their
// implementation type strings (with pointer unwrapped one level).
// This mirrors buildSetIndex and is used to pre-populate t.bindVarTypes package-wide
// before per-file processing begins, ensuring cross-file bind variable visibility.
func buildBindVarTypes(patterns []WirePattern) map[string]string {
	m := make(map[string]string)
	for _, p := range patterns {
		wb, ok := p.(*WireBind)
		if !ok || wb.VarName == "" {
			continue
		}
		implType := wb.Implementation
		if ptr, ok2 := implType.(*types.Pointer); ok2 {
			implType = ptr.Elem()
		}
		m[wb.VarName] = implType.String()
	}
	return m
}
