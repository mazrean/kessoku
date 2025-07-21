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
	return &VarPool{
		vars: make(map[string]int),
	}
}

// Register registers an existing name to prevent variable shadowing
func (p *VarPool) Register(name string) {
	if name == "" || name == "_" {
		return
	}
	// Set the count to at least 1 so the name won't be used without a suffix
	if count, ok := p.vars[name]; !ok || count == 0 {
		p.vars[name] = 1
	}
}

func (p *VarPool) Get(t types.Type) string {
	name := p.getBaseName(t)

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

// goReservedKeywords contains Go reserved keywords that cannot be used as variable names
var goReservedKeywords = map[string]bool{
	"break": true, "default": true, "func": true, "interface": true, "select": true,
	"case": true, "defer": true, "go": true, "map": true, "struct": true,
	"chan": true, "else": true, "goto": true, "package": true, "switch": true,
	"const": true, "fallthrough": true, "if": true, "range": true, "type": true,
	"continue": true, "for": true, "import": true, "return": true, "var": true,
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

	// Check if the base name is a Go reserved keyword
	if goReservedKeywords[baseName] {
		return baseName + "Value"
	}

	return baseName
}
