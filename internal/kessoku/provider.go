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

// MarkImportsUsedFromParams marks imports as used based on InjectorParams
func MarkImportsUsedFromParams(params []*InjectorParam) {
	for _, param := range params {
		for _, imp := range param.ReferencedImports {
			imp.IsUsed = true
		}
	}
}

// MarkImportsUsedFromArguments marks imports as used based on InjectorArguments
func MarkImportsUsedFromArguments(args []*InjectorArgument) {
	for _, arg := range args {
		for _, imp := range arg.Param.ReferencedImports {
			imp.IsUsed = true
		}
	}
}

// MarkImportsUsedFromStatements marks imports as used based on statements that are actually used
func MarkImportsUsedFromStatements(stmts []InjectorStmt) {
	for _, stmt := range stmts {
		if providerStmt, ok := stmt.(*InjectorProviderCallStmt); ok {
			for _, imp := range providerStmt.Provider.ReferencedImports {
				imp.IsUsed = true
			}
			// Mark imports used by arguments
			for _, arg := range providerStmt.Arguments {
				for _, imp := range arg.Param.ReferencedImports {
					imp.IsUsed = true
				}
			}
			// Mark imports used by return parameters
			for _, returnParam := range providerStmt.Returns {
				for _, imp := range returnParam.ReferencedImports {
					imp.IsUsed = true
				}
			}
		}
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
	ASTExpr           ast.Expr
	Type              ProviderType
	Provides          [][]types.Type
	Requires          []types.Type
	IsReturnError     bool
	IsAsync           bool
	ReferencedImports map[string]*Import // Package paths to Import structs this provider references
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
	name              string
	channelName       string
	types             []types.Type
	refCounter        int
	withChannel       bool
	isArg             bool
	ReferencedImports map[string]*Import // Package paths to Import structs this parameter references
}

func NewInjectorParam(ts []types.Type, isArg bool) *InjectorParam {
	return &InjectorParam{
		types:             ts,
		isArg:             isArg,
		ReferencedImports: make(map[string]*Import),
	}
}

// NewInjectorParamWithImports creates a new InjectorParam and collects imports from the types
func NewInjectorParamWithImports(ts []types.Type, isArg bool, pkg string, imports map[string]*Import, varPool *VarPool) *InjectorParam {
	referencedImports := make(map[string]*Import)
	
	// Collect imports from all types
	for _, t := range ts {
		collectImportsFromType(t, pkg, imports, referencedImports, varPool)
	}
	
	return &InjectorParam{
		types:             ts,
		isArg:             isArg,
		ReferencedImports: referencedImports,
	}
}

// collectImportsFromType recursively collects imports needed for a type
func collectImportsFromType(t types.Type, pkg string, imports map[string]*Import, referencedImports map[string]*Import, varPool *VarPool) {
	switch typ := t.(type) {
	case *types.Named:
		if objPkg := typ.Obj().Pkg(); objPkg != nil && objPkg.Path() != pkg {
			pkgPath := objPkg.Path()
			if imp, exists := imports[pkgPath]; exists {
				referencedImports[pkgPath] = imp
			} else {
				// Create new import if it doesn't exist
				pkgName := objPkg.Name()
				newPkgName := varPool.GetName(pkgName)
				newImp := &Import{
					Name:          newPkgName,
					IsDefaultName: newPkgName == pkgName,
					IsUsed:        false,
				}
				imports[pkgPath] = newImp
				referencedImports[pkgPath] = newImp
			}
		}
	case *types.Alias:
		if objPkg := typ.Obj().Pkg(); objPkg != nil && objPkg.Path() != pkg {
			pkgPath := objPkg.Path()
			if imp, exists := imports[pkgPath]; exists {
				referencedImports[pkgPath] = imp
			} else {
				// Create new import if it doesn't exist
				pkgName := objPkg.Name()
				newPkgName := varPool.GetName(pkgName)
				newImp := &Import{
					Name:          newPkgName,
					IsDefaultName: newPkgName == pkgName,
					IsUsed:        false,
				}
				imports[pkgPath] = newImp
				referencedImports[pkgPath] = newImp
			}
		}
	case *types.Pointer:
		collectImportsFromType(typ.Elem(), pkg, imports, referencedImports, varPool)
	case *types.Slice:
		collectImportsFromType(typ.Elem(), pkg, imports, referencedImports, varPool)
	case *types.Array:
		collectImportsFromType(typ.Elem(), pkg, imports, referencedImports, varPool)
	case *types.Map:
		collectImportsFromType(typ.Key(), pkg, imports, referencedImports, varPool)
		collectImportsFromType(typ.Elem(), pkg, imports, referencedImports, varPool)
	case *types.Chan:
		collectImportsFromType(typ.Elem(), pkg, imports, referencedImports, varPool)
	case *types.Signature:
		if params := typ.Params(); params != nil {
			for i := 0; i < params.Len(); i++ {
				collectImportsFromType(params.At(i).Type(), pkg, imports, referencedImports, varPool)
			}
		}
		if results := typ.Results(); results != nil {
			for i := 0; i < results.Len(); i++ {
				collectImportsFromType(results.At(i).Type(), pkg, imports, referencedImports, varPool)
			}
		}
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			collectImportsFromType(typ.Field(i).Type(), pkg, imports, referencedImports, varPool)
		}
	case *types.Interface:
		for i := 0; i < typ.NumMethods(); i++ {
			collectImportsFromType(typ.Method(i).Type(), pkg, imports, referencedImports, varPool)
		}
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
