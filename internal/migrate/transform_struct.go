package migrate

import (
	"go/ast"
	"go/token"
	"go/types"
)

// fieldInfo holds information about a struct field for code generation.
type fieldInfo struct {
	typ      types.Type
	name     string
	exported bool
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
	// Unwrap pointer(s) to get the struct type
	// wire.FieldsOf(new(T), ...) -> *T, unwrap once -> T
	// wire.FieldsOf(new(*T), ...) -> **T, unwrap once -> *T, unwrap again -> T
	structType := unwrapPointer(wf.StructType)
	// Keep unwrapping if still a pointer
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
func (t *Transformer) buildFieldAccessor(structType types.Type, fields []fieldInfo) *ast.FuncLit {
	// Parameter: pointer to struct
	paramName := "s"
	param := &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(paramName)},
		Type:  &ast.StarExpr{X: t.typeExpr(structType)},
	}

	// Return types
	var resultTypes []*ast.Field
	var returnExprs []ast.Expr
	for _, f := range fields {
		resultTypes = append(resultTypes, &ast.Field{Type: t.typeExpr(f.typ)})
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
