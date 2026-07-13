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
	ProviderTypeFunction    ProviderType = "function"
	ProviderTypeArg         ProviderType = "arg"
	ProviderTypeStruct      ProviderType = "struct"
	ProviderTypeFieldAccess ProviderType = "field_access"
)

// StructFieldSpec represents a field extracted from a struct for dependency injection.
type StructFieldSpec struct {
	Type      types.Type // Field type (e.g., string) - used for dependency matching
	Name      string     // Field name (e.g., "DBHost") - used in generated code
	Index     int        // Original field index in struct - preserved for proper access
	Anonymous bool       // True for embedded fields - affects naming
}

// ProviderSpec represents a provider specification from annotations.
type ProviderSpec struct {
	ASTExpr           ast.Expr
	StructType        types.Type
	ReferencedImports map[string]*Import
	SourceField       *StructFieldSpec
	Type              ProviderType
	Provides          [][]types.Type
	Requires          []types.Type
	StructFields      []*StructFieldSpec
	DeclOrder         int
	IsReturnError     bool
	IsAsync           bool
	// IsVariadic reports whether the provider function is variadic; the last
	// element of Requires is then the variadic slice type and the generated
	// call site must expand it with `...`.
	IsVariadic bool
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
	ReferencedImports map[string]*Import
	name              string
	channelName       string
	types             []types.Type
	refCounter        int
	withChannel       bool
	isArg             bool
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
		// Recurse into type arguments (e.g. Container[*config.Config] needs config imported)
		if typeArgs := typ.TypeArgs(); typeArgs != nil {
			for typeArg := range typeArgs.Types() {
				collectImportsFromType(typeArg, pkg, imports, referencedImports, varPool)
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
		// Recurse into type arguments for generic aliases
		if typeArgs := typ.TypeArgs(); typeArgs != nil {
			for typeArg := range typeArgs.Types() {
				collectImportsFromType(typeArg, pkg, imports, referencedImports, varPool)
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
			for v := range params.Variables() {
				collectImportsFromType(v.Type(), pkg, imports, referencedImports, varPool)
			}
		}
		if results := typ.Results(); results != nil {
			for v := range results.Variables() {
				collectImportsFromType(v.Type(), pkg, imports, referencedImports, varPool)
			}
		}
	case *types.Struct:
		for field := range typ.Fields() {
			collectImportsFromType(field.Type(), pkg, imports, referencedImports, varPool)
		}
	case *types.Interface:
		for method := range typ.Methods() {
			collectImportsFromType(method.Type(), pkg, imports, referencedImports, varPool)
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
	// Stmt generates the statements for this injector statement.
	// inChain reports whether the statements are emitted inside an errgroup
	// goroutine (an InjectorChainStmt body) rather than on the main goroutine.
	Stmt(varPool *VarPool, injector *Injector, returnErrStmts func(errExpr ast.Expr) []ast.Stmt, inChain bool) ([]ast.Stmt, []string)
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

// InjectorFieldAccessStmt represents field extraction from a struct instance.
// Implements InjectorStmt interface.
type InjectorFieldAccessStmt struct {
	StructParam *InjectorParam   // The struct instance parameter
	Field       *StructFieldSpec // The field to extract
	ReturnParam *InjectorParam   // The result parameter
}

// Stmt generates: fieldVar := structVar.FieldName (or = when predeclared in async builds)
func (stmt *InjectorFieldAccessStmt) Stmt(varPool *VarPool, injector *Injector, _ func(errExpr ast.Expr) []ast.Stmt, _ bool) ([]ast.Stmt, []string) {
	// In async builds all injector variables are predeclared in a var block,
	// so plain assignment is required to avoid shadowing them with :=.
	useAssign := stmt.ReturnParam.WithChannel()
	if injector != nil && hasChainStmts(injector) {
		useAssign = true
	}

	tokenType := token.DEFINE
	if useAssign {
		tokenType = token.ASSIGN
	}

	var stmts []ast.Stmt

	// Generate the field access assignment: fieldVar := structVar.FieldName (or = for async)
	stmts = append(stmts, &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent(stmt.ReturnParam.Name(varPool))},
		Tok: tokenType,
		Rhs: []ast.Expr{
			&ast.SelectorExpr{
				X:   ast.NewIdent(stmt.StructParam.Name(varPool)),
				Sel: ast.NewIdent(stmt.Field.Name),
			},
		},
	})

	// If this parameter has a channel (for async coordination), close it
	if stmt.ReturnParam.WithChannel() {
		channelName := stmt.ReturnParam.ChannelName(varPool)
		stmts = append(stmts, &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun:  ast.NewIdent("close"),
				Args: []ast.Expr{ast.NewIdent(channelName)},
			},
		})
	}

	return stmts, nil
}

// HasAsync returns false - field access is always synchronous
func (stmt *InjectorFieldAccessStmt) HasAsync() bool {
	return false
}

type Injector struct {
	Return *InjectorReturn
	Name   string
	// asyncEgName is the generated errgroup variable name (async builds only).
	asyncEgName string
	// asyncCancelName is the generated context.CancelFunc variable name.
	// It is set only for error-returning async builds.
	asyncCancelName string
	// asyncCtxName is the name of the cancellable context variable used for
	// channel-wait select statements. It is empty when the generated code
	// must wait on channels unconditionally (non-error-returning builds).
	asyncCtxName  string
	Params        []*InjectorParam
	Args          []*InjectorArgument
	Vars          []*InjectorParam
	Stmts         []InjectorStmt
	IsReturnError bool
}

// egName returns the errgroup variable name used in generated async code.
func (injector *Injector) egName() string {
	if injector.asyncEgName != "" {
		return injector.asyncEgName
	}
	return "eg"
}
