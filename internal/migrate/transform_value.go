package migrate

import "go/ast"

// transformValue transforms wire.Value to kessoku.Value.
func (t *Transformer) transformValue(wv *WireValue) *KessokuValue {
	return &KessokuValue{
		Expr:      wv.Expr,
		SourcePos: wv.Pos,
	}
}

// transformInterfaceValue transforms wire.InterfaceValue to kessoku.Bind + kessoku.Value.
// When the value expression is the untyped nil literal, Go cannot infer the type
// parameter of kessoku.Value[T], so we emit an explicit type parameter derived from
// the interface type (e.g. kessoku.Value[Logger](nil)).
func (t *Transformer) transformInterfaceValue(wiv *WireInterfaceValue) *KessokuBind {
	ifaceType := unwrapPointer(wiv.Interface)

	value := &KessokuValue{
		Expr:      wiv.Expr,
		SourcePos: wiv.Pos,
	}

	// If the expression is an untyped nil literal, the compiler cannot infer T
	// from kessoku.Value(nil). Attach an explicit type expression so the writer
	// emits kessoku.Value[I](nil).
	if isNilIdent(wiv.Expr) {
		value.TypeExpr = t.typeExpr(ifaceType)
	}

	return &KessokuBind{
		Interface: ifaceType,
		Provider:  value,
		SourcePos: wiv.Pos,
	}
}

// isNilIdent reports whether expr is the bare identifier "nil".
func isNilIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}
