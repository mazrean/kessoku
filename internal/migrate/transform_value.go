package migrate

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
