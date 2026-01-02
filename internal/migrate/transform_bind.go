package migrate

import (
	"fmt"
	"go/ast"
	"go/types"
)

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
		// External package - use selector with proper import handling
		pkgName := implPkg.Name()
		// If we have a TypeConverter, register the import and get the actual name to use
		if t.tc != nil {
			pkgName = t.tc.AddImport(implPkg.Path(), implPkg.Name())
		}
		funcExpr = &ast.SelectorExpr{
			X:   ast.NewIdent(pkgName),
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
