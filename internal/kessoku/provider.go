// Package kessoku provides dependency injection code generation functionality.
package kessoku

import (
	"go/ast"
	"go/token"
	"go/types"
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
	IsAsync       bool
}

type Return struct {
	Type        types.Type
	ASTTypeExpr ast.Expr
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
	refCounter int
}

func NewInjectorParam(name string) *InjectorParam {
	return &InjectorParam{
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

type InjectorChannel struct {
	name string
}

func NewInjectorChannel(name string) *InjectorChannel {
	return &InjectorChannel{
		name: name,
	}
}

func (c *InjectorChannel) Name() string {
	return c.name
}

type InjectorArgument struct {
	Param *InjectorParam
	Arg   *Argument
}

type InjectorReturn struct {
	Param  *InjectorParam
	Return *Return
}

type InjectorStmt interface {
	Stmt(injector *Injector) []ast.Stmt
	HasAsync() bool
}

type InjectorProviderCallStmt struct {
	Provider  *ProviderSpec
	Arguments []*InjectorParam
	Returns   []*InjectorParam
	Channel   *InjectorChannel
}

func (stmt *InjectorProviderCallStmt) Stmt(injector *Injector) []ast.Stmt {
	var stmts []ast.Stmt

	lhs := make([]ast.Expr, 0, len(stmt.Returns)+1)
	for _, ret := range stmt.Returns {
		lhs = append(lhs, &ast.Ident{
			Name: ret.Name(),
		})
	}
	if stmt.Provider.IsReturnError {
		lhs = append(lhs, &ast.Ident{
			Name: "err",
		})
	}

	args := make([]ast.Expr, 0, len(stmt.Arguments))
	for _, arg := range stmt.Arguments {
		args = append(args, ast.NewIdent(arg.Name()))
	}

	// Generate call to provider.Fn()() - call the Fn method, then call the returned function
	rhs := &ast.CallExpr{
		Fun: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   stmt.Provider.ASTExpr,
				Sel: ast.NewIdent("Fn"),
			},
			Args: []ast.Expr{},
		},
		Args: args,
	}

	stmts = append(stmts, &ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: lhs,
		Rhs: []ast.Expr{rhs},
	})

	if stmt.Provider.IsReturnError {
		stmts = append(stmts, &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  ast.NewIdent("err"),
				Op: token.NEQ,
				Y:  ast.NewIdent("nil"),
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.DeclStmt{
						Decl: &ast.GenDecl{
							Tok: token.VAR,
							Specs: []ast.Spec{
								&ast.ValueSpec{
									Names: []*ast.Ident{ast.NewIdent("zero")},
									Type:  injector.Return.Return.ASTTypeExpr,
								},
							},
						},
					},
					&ast.ReturnStmt{
						Results: []ast.Expr{
							ast.NewIdent("zero"),
							ast.NewIdent("err"),
						},
					},
				},
			},
		})
	}

	return stmts
}

func (stmt *InjectorProviderCallStmt) HasAsync() bool {
	return stmt.Provider.IsAsync
}

type InjectorChainStmt struct {
	Statements []InjectorStmt
	Inputs     []*InjectorChannel
}

func (stmt *InjectorChainStmt) Stmt(injector *Injector) []ast.Stmt {
	var stmts []ast.Stmt

	// For now, generate each statement in the chain sequentially
	// TODO: Add goroutine and synchronization logic for async chains
	for _, chainStmt := range stmt.Statements {
		stmts = append(stmts, chainStmt.Stmt(injector)...)
	}

	return stmts
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
	Stmts         []InjectorStmt
	IsReturnError bool
}
