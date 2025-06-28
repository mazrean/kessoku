package kessoku

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log/slog"
	"slices"
	"strings"

	"github.com/mazrean/kessoku/internal/pkg/collection"
)

// createASTTypeExpr creates an AST type expression from a types.Type
func createASTTypeExpr(t types.Type) ast.Expr {
	switch typ := t.(type) {
	case *types.Basic:
		return ast.NewIdent(typ.Name())
	case *types.Pointer:
		return &ast.StarExpr{
			X: createASTTypeExpr(typ.Elem()),
		}
	case *types.Named:
		name := typ.Obj().Name()
		if pkg := typ.Obj().Pkg(); pkg != nil && pkg.Name() != "main" {
			// For types from other packages, create a selector expression
			// Format: package.TypeName
			return &ast.SelectorExpr{
				X:   ast.NewIdent(pkg.Name()),
				Sel: ast.NewIdent(name),
			}
		}
		return ast.NewIdent(name)
	case *types.Slice:
		return &ast.ArrayType{
			Elt: createASTTypeExpr(typ.Elem()),
		}
	case *types.Array:
		return &ast.ArrayType{
			Len: &ast.BasicLit{
				Kind:  token.INT,
				Value: fmt.Sprintf("%d", typ.Len()),
			},
			Elt: createASTTypeExpr(typ.Elem()),
		}
	case *types.Map:
		return &ast.MapType{
			Key:   createASTTypeExpr(typ.Key()),
			Value: createASTTypeExpr(typ.Elem()),
		}
	case *types.Interface:
		if typ.NumMethods() == 0 {
			return ast.NewIdent("interface{}")
		}
		// For non-empty interfaces, use interface{} as fallback
		// Named interfaces should be handled by the *types.Named case above
		return ast.NewIdent("interface{}")
	case *types.Chan:
		var dir ast.ChanDir
		switch typ.Dir() {
		case types.SendRecv:
			dir = ast.SEND | ast.RECV
		case types.SendOnly:
			dir = ast.SEND
		case types.RecvOnly:
			dir = ast.RECV
		}
		return &ast.ChanType{
			Dir:   dir,
			Value: createASTTypeExpr(typ.Elem()),
		}
	case *types.Signature:
		// For function types, use a simplified representation
		return ast.NewIdent("func")
	default:
		// Fallback: try to use the string representation
		typeStr := t.String()
		// Remove package paths and just use the type name
		if idx := strings.LastIndex(typeStr, "."); idx != -1 {
			typeStr = typeStr[idx+1:]
		}
		return ast.NewIdent(typeStr)
	}
}

func CreateInjector(metaData *MetaData, build *BuildDirective) (*Injector, error) {
	slog.Debug("CreateInjector", "build", build)
	for _, provider := range build.Providers {
		slog.Debug("provider", "provider", provider)
	}
	graph, err := NewGraph(build)
	if err != nil {
		return nil, fmt.Errorf("create graph: %w", err)
	}

	injector, err := graph.Build()
	if err != nil {
		return nil, fmt.Errorf("build injector: %w", err)
	}

	return injector, nil
}

type node struct {
	arg          *Argument
	providerSpec *ProviderSpec
	providerArgs []*InjectorParam
	requireCount int
}

type edgeNode struct {
	node          *node
	provideArgSrc int
	provideArgDst int
}

type returnVal struct {
	node        *node
	returnIndex int
}

type Graph struct {
	returnType     *Return
	returnValue    *returnVal
	waitNodes      *collection.Queue[*node]
	waitNodesAdded map[*node]bool
	edges          map[*node][]*edgeNode
	injectorName   string
}

func NewGraph(build *BuildDirective) (*Graph, error) {
	graph := &Graph{
		injectorName:   build.InjectorName,
		returnType:     build.Return,
		waitNodes:      collection.NewQueue[*node](),
		waitNodesAdded: make(map[*node]bool),
		edges:          make(map[*node][]*edgeNode),
	}

	argProviderMap := make(map[string]*Argument)
	for _, arg := range build.Arguments {
		key := arg.Type.String()
		if _, ok := argProviderMap[key]; ok {
			return nil, fmt.Errorf("multiple args provide %s", key)
		}

		argProviderMap[key] = arg
	}

	type typeProvider struct {
		provider    *ProviderSpec
		returnIndex int
	}

	typeProviderMap := make(map[string]*typeProvider)
	for _, provider := range build.Providers {
		for i, t := range provider.Provides {
			key := t.String()
			if _, ok := argProviderMap[key]; ok {
				return nil, fmt.Errorf("multiple providers provide %s", key)
			}

			if _, ok := typeProviderMap[key]; ok {
				return nil, fmt.Errorf("multiple providers provide %s", key)
			}

			typeProviderMap[key] = &typeProvider{
				provider:    provider,
				returnIndex: i,
			}
		}
	}

	returnTypeKey := build.Return.Type.String()

	if returnArg, ok := argProviderMap[returnTypeKey]; ok {
		returnNode := &node{
			requireCount: 0,
			arg:          returnArg,
		}
		graph.returnValue = &returnVal{
			node:        returnNode,
			returnIndex: 0,
		}
		graph.waitNodes.Push(returnNode)
		graph.waitNodesAdded[returnNode] = true

		return graph, nil
	}

	returnProvider, ok := typeProviderMap[returnTypeKey]
	if !ok {
		return nil, fmt.Errorf("no provider provides %s", returnTypeKey)
	}

	providerNodeMap := make(map[*ProviderSpec]*node)
	argNodeMap := make(map[*Argument]*node)
	queue := collection.NewQueue[*node]()
	visited := make(map[*node]bool)

	returnNode := &node{
		requireCount: len(returnProvider.provider.Requires),
		providerSpec: returnProvider.provider,
		providerArgs: make([]*InjectorParam, len(returnProvider.provider.Requires)),
	}
	graph.returnValue = &returnVal{
		node:        returnNode,
		returnIndex: returnProvider.returnIndex,
	}
	queue.Push(returnNode)
	if returnNode.requireCount == 0 {
		graph.waitNodes.Push(returnNode)
		graph.waitNodesAdded[returnNode] = true
	}

	for n1 := range queue.Iter {
		// Skip if node is nil or already been processed
		if n1 == nil || visited[n1] {
			continue
		}
		visited[n1] = true

		// Skip argument nodes - they don't have dependencies to process
		if n1.providerSpec == nil {
			continue
		}

		for i, t := range n1.providerSpec.Requires {
			key := t.String()
			var (
				n2       *node
				srcIndex int
			)
			if arg, ok := argProviderMap[key]; ok {
				n2, ok = argNodeMap[arg]
				if !ok {
					n2 = &node{
						requireCount: 0,
						arg:          arg,
					}
					argNodeMap[arg] = n2
					queue.Push(n2)
				}

				srcIndex = 0
			} else if provider, ok := typeProviderMap[key]; ok {
				n2, ok = providerNodeMap[provider.provider]
				if !ok {
					n2 = &node{
						requireCount: len(provider.provider.Requires),
						providerSpec: provider.provider,
						providerArgs: make([]*InjectorParam, len(provider.provider.Requires)),
					}
					providerNodeMap[provider.provider] = n2
					queue.Push(n2)
				}

				srcIndex = provider.returnIndex
			} else {
				// Auto-detect missing dependency and create an argument for it
				argName := fmt.Sprintf("arg%d", len(argProviderMap))
				arg := &Argument{
					Name:        argName,
					Type:        t,
					ASTTypeExpr: createASTTypeExpr(t),
				}
				argProviderMap[key] = arg
				
				n2 = &node{
					requireCount: 0,
					arg:          arg,
				}
				argNodeMap[arg] = n2
				queue.Push(n2)
				srcIndex = 0
			}

			graph.edges[n2] = append(graph.edges[n2], &edgeNode{
				node:          n1,
				provideArgSrc: srcIndex,
				provideArgDst: i,
			})
			if n2.requireCount == 0 && !graph.waitNodesAdded[n2] {
				graph.waitNodes.Push(n2)
				graph.waitNodesAdded[n2] = true
			}
		}
	}

	// Add auto-detected arguments to the build directive
	for _, arg := range argProviderMap {
		// Only add arguments that were auto-detected (not originally in build.Arguments)
		if !slices.Contains(build.Arguments, arg) {
			build.Arguments = append(build.Arguments, arg)
		}
	}

	return graph, nil
}

func (g *Graph) Build() (*Injector, error) {
	injector := &Injector{
		Name:          g.injectorName,
		IsReturnError: false,
	}

	variableNameCounter := 0
	buildVisited := make(map[*node]bool)
	for n := range g.waitNodes.Iter {
		// Skip if this node has already been processed
		if buildVisited[n] {
			continue
		}
		buildVisited[n] = true
		slog.Debug("waitNodes", "waitNodes", n)
		var returnValues []*InjectorParam
		switch {
		case n.arg != nil:
			param := NewInjectorParam(n.arg.Name)
			injector.Params = append(injector.Params, param)
			returnValues = append(returnValues, param)

			injector.Args = append(injector.Args, &InjectorArgument{
				Param: param,
				Arg:   n.arg,
			})
		case n.providerSpec != nil:
			returnValues = make([]*InjectorParam, 0, len(n.providerSpec.Provides))
			for range n.providerSpec.Provides {
				param := NewInjectorParam(fmt.Sprintf("v%d", variableNameCounter))
				variableNameCounter++
				injector.Params = append(injector.Params, param)
				returnValues = append(returnValues, param)
			}

			injector.Stmts = append(injector.Stmts, &InjectorStmt{
				Provider:  n.providerSpec,
				Arguments: n.providerArgs,
				Returns:   returnValues,
			})

			if n.providerSpec.IsReturnError {
				injector.IsReturnError = true
			}
		default:
			return nil, errors.New("invalid node")
		}

		for _, edge := range g.edges[n] {
			edge.node.requireCount--
			slog.Debug("edge", "edge", edge, "node", edge.node)
			edge.node.providerArgs[edge.provideArgDst] = returnValues[edge.provideArgSrc]
			returnValues[edge.provideArgSrc].Ref()
			if edge.node.requireCount == 0 {
				g.waitNodes.Push(edge.node)
			}
		}

		if n == g.returnValue.node {
			returnValues[g.returnValue.returnIndex].Ref()
			injector.Return = &InjectorReturn{
				Param:  returnValues[g.returnValue.returnIndex],
				Return: g.returnType,
			}
		}
	}

	return injector, nil
}
