package migrate

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"unicode"
)

// maxInjectorReturns is the maximum number of return values for an injector function.
// An injector can return at most 2 values: the injected type and optionally an error.
const maxInjectorReturns = 2

// Transformer converts wire patterns to kessoku patterns.
type Transformer struct {
	tc *TypeConverter
}

// NewTransformer creates a new Transformer instance.
func NewTransformer() *Transformer {
	return &Transformer{}
}

// Transform transforms a list of wire patterns to kessoku patterns.
// If tc is non-nil, it will be used for proper package-qualified type expressions.
func (t *Transformer) Transform(patterns []WirePattern, pkg *types.Package, tc *TypeConverter) ([]KessokuPattern, error) {
	t.tc = tc
	var result []KessokuPattern

	for _, p := range patterns {
		switch wp := p.(type) {
		case *WireNewSet:
			transformed, err := t.transformNewSet(wp, pkg)
			if err != nil {
				return nil, err
			}
			result = append(result, transformed)
		case *WireBind:
			transformed, err := t.transformBind(wp, pkg)
			if err != nil {
				return nil, err
			}
			result = append(result, transformed)
		case *WireValue:
			result = append(result, t.transformValue(wp))
		case *WireInterfaceValue:
			result = append(result, t.transformInterfaceValue(wp))
		case *WireStruct:
			result = append(result, t.transformStruct(wp, pkg))
		case *WireFieldsOf:
			result = append(result, t.transformFieldsOf(wp, pkg))
		case *WireProviderFunc:
			result = append(result, t.transformProviderFunc(wp))
		case *WireSetRef:
			result = append(result, t.transformSetRef(wp))
		case *WireBuild:
			transformed, err := t.transformBuild(wp, pkg)
			if err != nil {
				return nil, err
			}
			result = append(result, transformed)
		}
	}

	return result, nil
}

// typeExpr converts a types.Type to ast.Expr using TypeConverter if available,
// otherwise falls back to the standalone typeToExpr function.
func (t *Transformer) typeExpr(typ types.Type) ast.Expr {
	if t.tc != nil {
		return t.tc.TypeToExpr(typ)
	}
	return typeToExpr(typ)
}

// Helper functions

func unwrapPointer(t types.Type) types.Type {
	if ptr, ok := t.(*types.Pointer); ok {
		return ptr.Elem()
	}
	return t
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func toLowerCamel(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func typeToExpr(t types.Type) ast.Expr {
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
		// Note: For cross-package types, this loses the package qualifier.
		// In practice, wire migrations typically use same-package types.
		return ast.NewIdent(obj.Name())
	case *types.Pointer:
		return &ast.StarExpr{X: typeToExpr(typ.Elem())}
	case *types.Slice:
		return &ast.ArrayType{Elt: typeToExpr(typ.Elem())}
	case *types.Map:
		return &ast.MapType{
			Key:   typeToExpr(typ.Key()),
			Value: typeToExpr(typ.Elem()),
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
			Elt: typeToExpr(typ.Elem()),
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
		return &ast.ChanType{Dir: dir, Value: typeToExpr(typ.Elem())}
	default:
		return ast.NewIdent(t.String())
	}
}
