package migrate

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"unicode"
)

// maxInjectorReturns is the maximum number of return values for an injector function.
// An injector can return at most 2 values: the injected type and optionally an error.
const maxInjectorReturns = 2

// Transformer converts wire patterns to kessoku patterns.
type Transformer struct {
	tc           *TypeConverter
	setIndex     map[string]*WireNewSet // var name → WireNewSet for dedup (BUG-10)
	bindVarTypes map[string]string      // VarName -> implementation type string for top-level WireBind vars (BUG-14)
}

// NewTransformer creates a new Transformer instance.
func NewTransformer() *Transformer {
	return &Transformer{}
}

// Transform transforms a list of wire patterns to kessoku patterns.
// If tc is non-nil, it will be used for proper package-qualified type expressions.
func (t *Transformer) Transform(patterns []WirePattern, pkg *types.Package, tc *TypeConverter) ([]KessokuPattern, error) {
	t.tc = tc

	// Build a set index so that transformElements can look up set contents by name.
	// This is used to deduplicate providers when wire.Build references both a set
	// and a wire.Bind that covers the same implementation type (BUG-10).
	//
	// t.setIndex may have been pre-populated by the caller with a package-wide
	// index (so that set refs in wire.Build can resolve sets defined in other
	// files of the same package). Merge file-local sets into the existing index
	// without overwriting entries already present from other files.
	fileSetIndex := buildSetIndex(patterns)
	if t.setIndex == nil {
		t.setIndex = fileSetIndex
	} else {
		for k, v := range fileSetIndex {
			if _, exists := t.setIndex[k]; !exists {
				t.setIndex[k] = v
			}
		}
	}
	setIndex := t.setIndex

	// First pass: build a map of top-level WireBind variable names to their
	// implementation type strings. This lets transformElements know when a
	// WireSetRef points to a top-level bind variable (so the explicit constructor
	// call can be suppressed – the bind already wraps the constructor) (BUG-14).
	//
	// t.bindVarTypes may have been pre-populated by the caller with a package-wide
	// index (so that bind vars defined in a different file of the same package are
	// visible when processing a file that only contains a wire.NewSet referencing
	// them). Merge file-local bind vars into the existing index without overwriting
	// entries already present from other files.
	fileBVT := buildBindVarTypes(patterns)
	if t.bindVarTypes == nil {
		t.bindVarTypes = fileBVT
	} else {
		for k, v := range fileBVT {
			if _, exists := t.bindVarTypes[k]; !exists {
				t.bindVarTypes[k] = v
			}
		}
	}

	var result []KessokuPattern

	for _, p := range patterns {
		switch wp := p.(type) {
		case *WireNewSet:
			transformed, err := t.transformNewSet(wp, pkg, setIndex)
			if err != nil {
				return nil, err
			}
			result = append(result, transformed)
		case *WireBind:
			if wp.VarName != "" {
				// Top-level bind variable: wrap in a kessoku.Set so it can be
				// declared as a variable and referenced by other sets/injectors (BUG-14).
				bind, err := t.transformBind(wp, pkg, nil)
				if err != nil {
					return nil, err
				}
				result = append(result, &KessokuSet{
					VarName:   wp.VarName,
					Elements:  []KessokuPattern{bind},
					SourcePos: wp.Pos,
				})
			} else {
				transformed, err := t.transformBind(wp, pkg, nil)
				if err != nil {
					return nil, err
				}
				result = append(result, transformed)
			}
		case *WireValue:
			result = append(result, t.transformValue(wp))
		case *WireInterfaceValue:
			result = append(result, t.transformInterfaceValue(wp))
		case *WireStruct:
			result = append(result, t.transformStruct(wp, pkg)...)
		case *WireFieldsOf:
			transformed, err := t.transformFieldsOf(wp, pkg)
			if err != nil {
				return nil, err
			}
			result = append(result, transformed)
		case *WireProviderFunc:
			result = append(result, t.transformProviderFunc(wp))
		case *WireSetRef:
			transformed, err := t.transformSetRef(wp)
			if err != nil {
				return nil, err
			}
			result = append(result, transformed)
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
	return slices.Contains(slice, s)
}

func toLowerCamel(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// sanitizeParamName ensures the name is a valid Go identifier for use as a
// function parameter. If the name is a Go keyword, it appends "_" to avoid a
// compile error in the generated code.
func sanitizeParamName(s string) string {
	if token.Lookup(s).IsKeyword() {
		return s + "_"
	}
	return s
}

// uniqueParamName returns a collision-free parameter name by appending a
// numeric suffix (_2, _3, …) when the candidate name is already in usedNames.
// The chosen name is recorded in usedNames before returning.
func uniqueParamName(candidate string, usedNames map[string]int) string {
	if _, taken := usedNames[candidate]; !taken {
		usedNames[candidate] = 1
		return candidate
	}
	usedNames[candidate]++
	next := fmt.Sprintf("%s_%d", candidate, usedNames[candidate])
	// Recurse in case the suffixed name is itself already taken.
	return uniqueParamName(next, usedNames)
}

func typeToExpr(t types.Type) ast.Expr {
	if t == nil {
		return nil
	}
	switch typ := t.(type) {
	case *types.Named:
		obj := typ.Obj()
		base := ast.NewIdent(obj.Name())
		return wrapTypeArgs(base, typ.TypeArgs(), typeToExpr)
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
		if typ.Kind() == types.UnsafePointer {
			return &ast.SelectorExpr{X: ast.NewIdent("unsafe"), Sel: ast.NewIdent("Pointer")}
		}
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
