// Package kessoku provides dependency injection code generation functionality.
package kessoku

import (
	"go/ast"
	"go/types"
)

type MetaData struct {
	Package string
	Imports []*ast.ImportSpec
}

// ProviderType represents the type of provider.
type ProviderType string

const (
	ProviderTypeFunction ProviderType = "function"
	ProviderTypeArg      ProviderType = "arg"
)

// Argument represents a function argument.
type Argument struct {
	Name        string
	Type        types.Type
	ASTTypeExpr ast.Expr
}

// ProviderSpec represents a provider specification from annotations.
type ProviderSpec struct {
	Type          ProviderType
	Provides      []types.Type
	Requires      []types.Type
	IsReturnError bool
	ASTExpr       ast.Expr // The full AST expression from Inject call
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
	ID         uint64
	name       string
	refCounter int
}

var injectorParamIDCounter uint64 = 0

func NewInjectorParam(name string) *InjectorParam {
	id := injectorParamIDCounter
	injectorParamIDCounter++
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
	Name          string
	Params        []*InjectorParam
	Args          []*InjectorArgument
	Stmts         []*InjectorStmt
	Return        *InjectorReturn
	IsReturnError bool
}
