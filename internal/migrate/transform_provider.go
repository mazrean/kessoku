package migrate

import (
	"fmt"
	"go/ast"
)

// transformProviderFunc transforms a provider function to kessoku.Provide.
func (t *Transformer) transformProviderFunc(wf *WireProviderFunc) *KessokuProvide {
	return &KessokuProvide{
		FuncExpr:  wf.Expr,
		SourcePos: wf.Pos,
	}
}

// transformSetRef transforms a set reference.
// Returns an error if the reference is to a set defined in another package,
// because kessoku's parser does not support cross-package set references.
func (t *Transformer) transformSetRef(ws *WireSetRef) (*KessokuSetRef, error) {
	if _, isCrossPkg := ws.Expr.(*ast.SelectorExpr); isCrossPkg {
		return nil, &ParseError{
			Kind:    ParseErrorTypeResolution,
			File:    ws.File,
			Pos:     ws.Pos,
			Message: fmt.Sprintf("cross-package set reference %q is not supported: kessoku cannot use wire sets defined in other packages; copy the providers into the current package or inline them directly in wire.Build", ws.Name),
		}
	}
	return &KessokuSetRef{
		Name:      ws.Name,
		Expr:      ws.Expr,
		SourcePos: ws.Pos,
	}, nil
}
