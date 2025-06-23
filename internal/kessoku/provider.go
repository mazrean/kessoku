// Package kessoku provides dependency injection code generation functionality.
package kessoku

import (
	"go/ast"
	"go/types"
	"sync/atomic"
)

type MetaData struct {
	Imports map[string]*ast.ImportSpec // Map from package path to import spec
	Package string
}

// ProviderType represents the type of provider.
type ProviderType string

const (
	ProviderTypeFunction ProviderType = "function"
	ProviderTypeArg      ProviderType = "arg"
)

// Argument represents a function argument.
type Argument struct {
	Type        types.Type
	ASTTypeExpr ast.Expr
	Name        string
}

// ProviderSpec represents a provider specification from annotations.
type ProviderSpec struct {
	ASTExpr       ast.Expr
	Type          ProviderType
	Provides      []types.Type
	Requires      []types.Type
	IsReturnError bool
}

type Return struct {
	Type        types.Type
	ASTTypeExpr ast.Expr
}

// Provider represents a legacy provider function (for backward compatibility).
type Provider struct {
	Fn *ast.FuncDecl
}

// BuildDirective represents a kessoku.Inject call.
type BuildDirective struct {
	InjectorName string
	Arguments    []*Argument
	Return       *Return
	Providers    []*ProviderSpec
}

type InjectorParam struct {
	name       string
	ID         uint64
	refCounter int
}

var injectorParamIDCounter uint64

func NewInjectorParam(name string) *InjectorParam {
	id := atomic.AddUint64(&injectorParamIDCounter, 1) - 1
	return &InjectorParam{
		ID:   id,
		name: name,
	}
}

func (p *InjectorParam) Ref() {
	p.refCounter++
}

func (p *InjectorParam) Name() string {
	if p.refCounter == 0 {
		return "_"
	}

	return p.name
}

type InjectorArgument struct {
	Param *InjectorParam
	Arg   *Argument
}

type InjectorReturn struct {
	Param  *InjectorParam
	Return *Return
}

type InjectorStmt struct {
	Provider  *ProviderSpec
	Arguments []*InjectorParam
	Returns   []*InjectorParam
}

type Injector struct {
	Return        *InjectorReturn
	Name          string
	Params        []*InjectorParam
	Args          []*InjectorArgument
	Stmts         []*InjectorStmt
	IsReturnError bool
}
