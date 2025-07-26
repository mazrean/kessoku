// Package kessoku provides dependency injection code generation functionality.
package kessoku

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
)

type Package struct {
	Name string
	Path string
}

type Import struct {
	Name          string
	IsDefaultName bool
	IsUsed        bool
}

func importSpec(imp *Import, path string) *ast.ImportSpec {
	// Mark import as used when generating import specification
	imp.IsUsed = true
	
	if imp.IsDefaultName {
		return &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(path),
			},
		}
	}

	return &ast.ImportSpec{
		Name: ast.NewIdent(imp.Name),
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(path),
		},
	}
}

// MarkImportUsed marks an import as used if it exists in the imports map
func MarkImportUsed(imports map[string]*Import, pkgPath string) {
	if imp, exists := imports[pkgPath]; exists {
		imp.IsUsed = true
	}
}

// GetUsedImports returns only the imports that are marked as used
func GetUsedImports(imports map[string]*Import) map[string]*Import {
	used := make(map[string]*Import)
	for path, imp := range imports {
		if imp.IsUsed {
			used[path] = imp
		}
	}
	return used
}

type MetaData struct {
	Imports map[string]*Import
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
	name        string
	channelName string
	types       []types.Type
	refCounter  int
	withChannel bool
	isArg       bool
}

func NewInjectorParam(ts []types.Type, isArg bool) *InjectorParam {
	return &InjectorParam{
		types: ts,
		isArg: isArg,
	}
}

func (p *InjectorParam) Ref(isWait bool) {
	p.refCounter++
	p.withChannel = !p.isArg && (p.withChannel || isWait)
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
