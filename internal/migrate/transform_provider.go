package migrate

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
