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
	IsAsync       bool
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
	Provider       *ProviderSpec
	Arguments      []*InjectorParam
	Returns        []*InjectorParam
	ParallelGroup  int // Group ID for parallel execution (0 means sequential)
}

// DependencyChain represents a sequence of providers that must be executed in order within a single goroutine
type DependencyChain struct {
	ID         int               // Unique chain ID within the parallel group
	Statements []*InjectorStmt   // Providers in execution order
	Inputs     []*ChannelInput   // Channels to receive data from other chains
	Outputs    []*ChannelOutput  // Channels to send data to other chains
}

// ChannelInput represents input from another dependency chain
type ChannelInput struct {
	FromChainID   int             // Source chain ID
	ParamName     string          // Parameter name to receive
	ParamType     *InjectorParam  // Parameter reference
	ChannelName   string          // Generated channel variable name
}

// ChannelOutput represents output to another dependency chain
type ChannelOutput struct {
	ToChainID     int             // Target chain ID
	ParamName     string          // Parameter name to send
	ParamType     *InjectorParam  // Parameter reference
	ChannelName   string          // Generated channel variable name
}

// ParallelExecutionPlan represents the optimized execution plan for a parallel group
type ParallelExecutionPlan struct {
	GroupID    int                 // Parallel group ID
	Chains     []*DependencyChain  // Dependency chains that can run in parallel
	Channels   map[string]string   // Channel name mapping (param -> channel name)
}

type Injector struct {
	Return             *InjectorReturn
	Name               string
	Params             []*InjectorParam
	Args               []*InjectorArgument
	Stmts              []*InjectorStmt
	IsReturnError      bool
	HasExistingContext bool
	ContextParamName   string
	ExecutionPlans     map[int]*ParallelExecutionPlan // Map from parallel group ID to execution plan
}
