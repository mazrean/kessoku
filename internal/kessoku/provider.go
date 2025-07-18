// Package kessoku provides dependency injection code generation functionality.
package kessoku

import (
	"go/ast"
	"go/types"
)

type Package struct {
	Name string
	Path string
}

type MetaData struct {
	Imports map[string]*ast.ImportSpec // Map from package path to import spec
	Package Package
}

// ProviderType represents the type of provider.
type ProviderType string

const (
	ProviderTypeFunction ProviderType = "function"
	ProviderTypeArg      ProviderType = "arg"
)

// ProviderSpec represents a provider specification from annotations.
type ProviderSpec struct {
	ASTExpr       ast.Expr
	Type          ProviderType
	Provides      [][]types.Type
	Requires      []types.Type
	IsReturnError bool
	IsAsync       bool
}

type Return struct {
	Type        types.Type
	ASTTypeExpr ast.Expr
}

// BuildDirective represents a kessoku.Inject call.
type BuildDirective struct {
	InjectorName string
	Return       *Return
	Providers    []*ProviderSpec
}

type InjectorParam struct {
	types       []types.Type
	name        string
	channelName string
	refCounter  int
	withChannel bool
}

func NewInjectorParam(ts []types.Type) *InjectorParam {
	return &InjectorParam{
		types: ts,
	}
}

func (p *InjectorParam) Ref(isWait bool) {
	p.refCounter++
	p.withChannel = p.withChannel || isWait
}

func (p *InjectorParam) Name(varPool *VarPool) string {
	if p.name != "" {
		return p.name
	}

	if p.refCounter == 0 {
		return "_"
	}
	p.name = varPool.Get(p.types[0])

	return p.name
}

func (p *InjectorParam) ChannelName(varPool *VarPool) string {
	if p.channelName != "" {
		return p.channelName
	}

	if p.refCounter == 0 {
		return "_"
	}
	p.channelName = varPool.GetChannel(p.types[0])

	return p.channelName
}

func (p *InjectorParam) WithChannel() bool {
	return p.withChannel
}

func (p *InjectorParam) Type() types.Type {
	return p.types[0]
}

type InjectorArgument struct {
	Param       *InjectorParam
	Type        types.Type
	ASTTypeExpr ast.Expr
}

type InjectorReturn struct {
	Param  *InjectorParam
	Return *Return
}

type InjectorStmt interface {
	Stmt(varPool *VarPool, injector *Injector, returnErrStmts func(errExpr ast.Expr) []ast.Stmt) ([]ast.Stmt, []string)
	HasAsync() bool
}

type InjectorCallArgument struct {
	Param  *InjectorParam
	IsWait bool
}

type InjectorProviderCallStmt struct {
	Provider  *ProviderSpec
	Arguments []*InjectorCallArgument
	Returns   []*InjectorParam
}

func (stmt *InjectorProviderCallStmt) HasAsync() bool {
	return stmt.Provider.IsAsync
}

type InjectorChainStmt struct {
	Statements []InjectorStmt
}

func (stmt *InjectorChainStmt) HasAsync() bool {
	for _, chainStmt := range stmt.Statements {
		if chainStmt.HasAsync() {
			return true
		}
	}
	return false
}

type Injector struct {
	Return        *InjectorReturn
	Name          string
	Params        []*InjectorParam
	Args          []*InjectorArgument
	Vars          []*InjectorParam
	Stmts         []InjectorStmt
	IsReturnError bool
}
