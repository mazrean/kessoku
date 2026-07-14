package migrate

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
)

// fieldInfo holds information about a struct field for code generation.
type fieldInfo struct {
	typ      types.Type
	name     string
	exported bool
}

// transformStruct transforms wire.Struct to kessoku.Provide with function literal.
func (t *Transformer) transformStruct(ws *WireStruct, pkg *types.Package) *KessokuProvide {
	// Unwrap all pointer layers to reach the underlying struct type.
	// wire.Struct(new(*T)) yields **T from extractTypeFromNew; we must strip
	// both layers to reach T before calling Underlying().
	// transformFieldsOf uses the same loop — mirror it here.
	structType := unwrapPointer(ws.StructType)
	for {
		if ptr, ok := structType.(*types.Pointer); ok {
			structType = ptr.Elem()
		} else {
			break
		}
	}
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

	// Collect fields to include.
	// When using "*", wire always means exported fields only, regardless of package.
	// When listing fields by name, unexported fields from external packages are skipped.
	// Fields with the struct tag wire:"-" are always skipped, matching wire's isPrevented logic.
	var fieldInfos []fieldInfo
	for i := range st.NumFields() {
		field := st.Field(i)
		// Skip fields marked wire:"-" (wire's isPrevented logic)
		if reflect.StructTag(st.Tag(i)).Get("wire") == "-" {
			continue
		}
		switch {
		case ws.Fields[0] == "*":
			// Skip unexported fields: wire's "*" always means exported fields only
			if !field.Exported() {
				continue
			}
		case contains(ws.Fields, field.Name()):
			// Skip unexported fields from external packages
			if isExternalPkg && !field.Exported() {
				continue
			}
		default:
			continue
		}
		fieldInfos = append(fieldInfos, fieldInfo{
			name:     field.Name(),
			typ:      field.Type(),
			exported: field.Exported(),
		})
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
	// wf.StructType comes from extractTypeFromNew which wraps the new(T) arg in one extra pointer:
	//   new(T)   -> *T   (paramType = T,  a value type)
	//   new(*T)  -> **T  (paramType = *T, a pointer type)
	// Strip exactly the one pointer layer that extractTypeFromNew added to get the real parameter type.
	paramType := unwrapPointer(wf.StructType)

	// Now strip any remaining pointer layers to reach the plain struct type for field lookup.
	structType := paramType
	for {
		if ptr, ok := structType.(*types.Pointer); ok {
			structType = ptr.Elem()
		} else {
			break
		}
	}
	underlying := structType.Underlying()
	if _, ok := underlying.(*types.Struct); !ok {
		return &KessokuProvide{SourcePos: wf.Pos}
	}

	// Check if struct is from external package
	isExternalPkg := false
	if named, ok := structType.(*types.Named); ok {
		if named.Obj().Pkg() != nil && named.Obj().Pkg() != pkg {
			isExternalPkg = true
		}
	}

	// Collect field types (skip unexported fields from external packages).
	// Use LookupFieldOrMethod instead of st.Fields() so that promoted
	// (embedded) fields are found in addition to direct fields.
	var fieldInfos []fieldInfo
	for _, fieldName := range wf.Fields {
		obj, _, _ := types.LookupFieldOrMethod(structType, true, pkg, fieldName)
		field, ok := obj.(*types.Var)
		if !ok {
			continue
		}
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

	// Build accessor function using the real parameter type (value or pointer).
	funcLit := t.buildFieldAccessor(paramType, fieldInfos, wf.IsPtrToStruct)

	return &KessokuProvide{
		FuncExpr:  funcLit,
		SourcePos: wf.Pos,
	}
}

// buildStructConstructor builds a function literal for struct construction.
func (t *Transformer) buildStructConstructor(structType types.Type, fields []fieldInfo, isPointer bool) *ast.FuncLit {
	// Build parameter list, resolving keyword conflicts and name collisions.
	usedNames := make(map[string]int)
	var params []*ast.Field
	var paramNames []string
	for _, f := range fields {
		paramName := uniqueParamName(sanitizeParamName(toLowerCamel(f.name)), usedNames)
		paramNames = append(paramNames, paramName)
		params = append(params, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(paramName)},
			Type:  t.typeExpr(f.typ),
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
		Type: t.typeExpr(structType),
		Elts: elts,
	}

	returnExpr := ast.Expr(structLit)
	returnType := t.typeExpr(structType)

	if isPointer {
		returnExpr = &ast.UnaryExpr{
			Op: token.AND,
			X:  structLit,
		}
		returnType = &ast.StarExpr{X: t.typeExpr(structType)}
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
// paramType is the actual parameter type for the accessor (T for new(T), *T for new(*T)).
// When isPtrToStruct is true (new(*S) form), wire provides both FieldType and *FieldType
// for each field, so the generated accessor returns both value and pointer for each field.
func (t *Transformer) buildFieldAccessor(paramType types.Type, fields []fieldInfo, isPtrToStruct bool) *ast.FuncLit {
	// Parameter: use the type as-is (value or pointer depending on wire.FieldsOf usage).
	paramName := "s"
	param := &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(paramName)},
		Type:  t.typeExpr(paramType),
	}

	// Return types and expressions.
	// When isPtrToStruct, each field produces two outputs: FieldType and *FieldType.
	var resultTypes []*ast.Field
	var returnExprs []ast.Expr
	for _, f := range fields {
		fieldExpr := &ast.SelectorExpr{
			X:   ast.NewIdent(paramName),
			Sel: ast.NewIdent(f.name),
		}
		resultTypes = append(resultTypes, &ast.Field{Type: t.typeExpr(f.typ)})
		returnExprs = append(returnExprs, fieldExpr)

		if isPtrToStruct {
			// Also provide *FieldType by taking address of the field.
			resultTypes = append(resultTypes, &ast.Field{Type: &ast.StarExpr{X: t.typeExpr(f.typ)}})
			returnExprs = append(returnExprs, &ast.UnaryExpr{
				Op: token.AND,
				X: &ast.SelectorExpr{
					X:   ast.NewIdent(paramName),
					Sel: ast.NewIdent(f.name),
				},
			})
		}
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
