package kessoku

import (
	"fmt"
	"go/types"

	"github.com/mazrean/kessoku/internal/pkg/strings"
)

type VarPool struct {
	vars map[string]int
}

func NewVarPool() *VarPool {
	vars := make(map[string]int, len(goPredeclaredIdentifiers)+len(goReservedKeywords))
	for _, id := range goPredeclaredIdentifiers {
		vars[id] = 1
	}
	for _, id := range goReservedKeywords {
		vars[id] = 1
	}

	return &VarPool{
		vars: vars,
	}
}

func (p *VarPool) GetName(baseName string) string {
	count := p.vars[baseName]
	p.vars[baseName] = count + 1

	if count == 0 {
		return baseName
	}

	return fmt.Sprintf("%s%d", baseName, count-1)
}

func (p *VarPool) Get(t types.Type) string {
	name := p.getBaseName(t)

	return p.GetName(name)
}

func (p *VarPool) GetChannel(t types.Type) string {
	name := p.getBaseName(t) + "Ch"

	count, ok := p.vars[name]
	if !ok {
		count = 0
	}
	p.vars[name] = count + 1

	if count == 0 {
		return name
	}

	return fmt.Sprintf("%s%d", name, count-1)
}

// getTypeBaseName extracts a base name from a type for argument naming
func (p *VarPool) getBaseName(t types.Type) string {
	// For pointers, recurse on the element type
	for ptr, ok := t.(*types.Pointer); ok; ptr, ok = t.(*types.Pointer) {
		t = ptr.Elem()
	}

	var baseName string
	switch t := t.(type) {
	case *types.Named:
		if obj := t.Obj(); obj != nil && obj.Pkg() != nil {
			if obj.Pkg().Path() == "context" && obj.Name() == "Context" {
				return "ctx"
			}
		}

		baseName = strings.ToLowerCamel(t.Obj().Name())
	case *types.Basic:
		// Check by kind for all basic types (byte and rune are handled by their underlying types)
		switch t.Kind() {
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
			types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64,
			types.Float32, types.Float64,
			types.UntypedInt, types.UntypedFloat, types.UntypedRune:
			return "num"
		case types.String, types.UntypedString:
			return "str"
		case types.Bool, types.UntypedBool:
			return "flag"
		case types.Complex64, types.Complex128, types.UntypedComplex:
			return "complex"
		case types.Uintptr, types.UnsafePointer:
			return "ptr"
		case types.UntypedNil:
			return "null"
		case types.Invalid:
			return "invalid"
		default:
			baseName = strings.ToLowerCamel(t.Name())
		}
	default:
		baseName = "val"
	}

	return baseName
}
