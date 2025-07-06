package kessoku

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log/slog"
	"slices"
	"sort"
	"strings"

	"github.com/mazrean/kessoku/internal/pkg/collection"
)

// createASTTypeExpr creates an AST type expression from a types.Type and returns required imports
func createASTTypeExpr(t types.Type) (ast.Expr, []string) {
	var imports []string

	switch typ := t.(type) {
	case *types.Basic:
		return ast.NewIdent(typ.Name()), imports
	case *types.Pointer:
		expr, elemImports := createASTTypeExpr(typ.Elem())
		return &ast.StarExpr{
			X: expr,
		}, elemImports
	case *types.Named:
		name := typ.Obj().Name()
		if pkg := typ.Obj().Pkg(); pkg != nil && pkg.Name() != "main" {
			// For types from other packages, create a selector expression
			// Format: package.TypeName
			imports = append(imports, pkg.Path())
			return &ast.SelectorExpr{
				X:   ast.NewIdent(pkg.Name()),
				Sel: ast.NewIdent(name),
			}, imports
		}
		return ast.NewIdent(name), imports
	case *types.Slice:
		expr, elemImports := createASTTypeExpr(typ.Elem())
		return &ast.ArrayType{
			Elt: expr,
		}, elemImports
	case *types.Array:
		expr, elemImports := createASTTypeExpr(typ.Elem())
		return &ast.ArrayType{
			Len: &ast.BasicLit{
				Kind:  token.INT,
				Value: fmt.Sprintf("%d", typ.Len()),
			},
			Elt: expr,
		}, elemImports
	case *types.Map:
		keyExpr, keyImports := createASTTypeExpr(typ.Key())
		valueExpr, valueImports := createASTTypeExpr(typ.Elem())
		allImports := make([]string, 0, len(keyImports)+len(valueImports))
		allImports = append(allImports, keyImports...)
		allImports = append(allImports, valueImports...)
		return &ast.MapType{
			Key:   keyExpr,
			Value: valueExpr,
		}, allImports
	case *types.Interface:
		methodFields := make([]*ast.Field, 0, typ.NumMethods())
		for method := range typ.Methods() {
			expr, newImports := createASTTypeExpr(method.Signature())
			imports = append(imports, newImports...)
			methodFields = append(methodFields, &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(method.Name())},
				Type:  expr,
			})
		}
		return &ast.InterfaceType{
			Methods: &ast.FieldList{
				List: methodFields,
			},
		}, imports
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
		expr, elemImports := createASTTypeExpr(typ.Elem())
		return &ast.ChanType{
			Dir:   dir,
			Value: expr,
		}, elemImports
	case *types.Signature:
		funcFields := make([]*ast.Field, 0, typ.Params().Len())
		for i := 0; i < typ.Params().Len(); i++ {
			expr, newImports := createASTTypeExpr(typ.Params().At(i).Type())
			imports = append(imports, newImports...)
			funcFields = append(funcFields, &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(fmt.Sprintf("arg%d", i))},
				Type:  expr,
			})
		}
		resultsFields := make([]*ast.Field, 0, typ.Results().Len())
		for i := 0; i < typ.Results().Len(); i++ {
			expr, newImports := createASTTypeExpr(typ.Results().At(i).Type())
			imports = append(imports, newImports...)
			resultsFields = append(resultsFields, &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(fmt.Sprintf("result%d", i))},
				Type:  expr,
			})
		}
		return &ast.FuncType{
			Params: &ast.FieldList{
				List: funcFields,
			},
			Results: &ast.FieldList{
				List: resultsFields,
			},
		}, imports
	default:
		// Fallback: try to use the string representation
		typeStr := t.String()
		// Remove package paths and just use the type name
		if idx := strings.LastIndex(typeStr, "."); idx != -1 {
			typeStr = typeStr[idx+1:]
		}
		return ast.NewIdent(typeStr), imports
	}
}

func CreateInjector(metaData *MetaData, build *BuildDirective) (*Injector, error) {
	slog.Debug("CreateInjector", "build", build)
	for _, provider := range build.Providers {
		slog.Debug("provider", "provider", provider)
	}
	graph, err := NewGraph(metaData, build)
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
	arg           *Argument
	providerSpec  *ProviderSpec
	providerArgs  []*InjectorParam
	requireCount  int
	parallelGroup int // Parallel execution group ID
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
	allNodes       []*node
}

func NewGraph(metaData *MetaData, build *BuildDirective) (*Graph, error) {
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
		graph.allNodes = append(graph.allNodes, returnNode)

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
	graph.allNodes = append(graph.allNodes, returnNode)
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
					graph.allNodes = append(graph.allNodes, n2)
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
					graph.allNodes = append(graph.allNodes, n2)
				}

				srcIndex = provider.returnIndex
			} else {
				// Auto-detect missing dependency and create an argument for it
				// Generate argument name from type name
				argName := generateArgName(t, argProviderMap)
				expr, requiredImports := createASTTypeExpr(t)

				// Add required imports to metadata
				for _, importPath := range requiredImports {
					if _, exists := metaData.Imports[importPath]; !exists {
						// Create import spec for the required package
						metaData.Imports[importPath] = &ast.ImportSpec{
							Path: &ast.BasicLit{
								Kind:  token.STRING,
								Value: fmt.Sprintf("\"%s\"", importPath),
							},
						}
					}
				}

				arg := &Argument{
					Name:        argName,
					Type:        t,
					ASTTypeExpr: expr,
				}
				argProviderMap[key] = arg

				n2 = &node{
					requireCount: 0,
					arg:          arg,
				}
				argNodeMap[arg] = n2
				queue.Push(n2)
				graph.allNodes = append(graph.allNodes, n2)
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

	// Add auto-detected arguments to the build directive and sort them deterministically
	autoDetectedArgs := make([]*Argument, 0)
	for _, arg := range argProviderMap {
		// Only add arguments that were auto-detected (not originally in build.Arguments)
		if !slices.Contains(build.Arguments, arg) {
			autoDetectedArgs = append(autoDetectedArgs, arg)
		}
	}

	// Sort arguments deterministically: context.Context first, then by type name
	sortArguments(autoDetectedArgs)
	build.Arguments = append(build.Arguments, autoDetectedArgs...)

	return graph, nil
}

// generateArgName creates a meaningful argument name from the type
func generateArgName(t types.Type, existingArgs map[string]*Argument) string {
	baseName := getTypeBaseName(t)

	// Check for conflicts and add suffix if needed
	counter := 0
	name := baseName
	for {
		// Check if this name conflicts with any existing argument names
		conflict := false
		for _, arg := range existingArgs {
			if arg.Name == name {
				conflict = true
				break
			}
		}
		if !conflict {
			break
		}
		counter++
		name = fmt.Sprintf("%s%d", baseName, counter)
	}
	return name
}

var ()

// getTypeBaseName extracts a base name from a type for argument naming
func getTypeBaseName(t types.Type) string {
	if named, ok := t.(*types.Named); ok {
		if obj := named.Obj(); obj != nil && obj.Pkg() != nil {
			if obj.Pkg().Path() == "context" && obj.Name() == "Context" {
				return "ctx"
			}
		}
	}

	// For pointers, recurse on the element type
	if ptr, ok := t.(*types.Pointer); ok {
		return getTypeBaseName(ptr.Elem())
	}

	// Handle basic types
	if basic, ok := t.(*types.Basic); ok {
		// Check by kind for all basic types (byte and rune are handled by their underlying types)
		switch basic.Kind() {
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
			return "num"
		case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
			return "num"
		case types.Float32, types.Float64:
			return "value"
		case types.String:
			return "str"
		case types.Bool:
			return "flag"
		case types.Complex64, types.Complex128:
			return "complex"
		case types.Uintptr:
			return "ptr"
		case types.UnsafePointer:
			return "unsafe"
		case types.UntypedBool, types.UntypedInt, types.UntypedRune, types.UntypedFloat, types.UntypedComplex, types.UntypedString, types.UntypedNil:
			return "untyped"
		case types.Invalid:
			return "invalid"
		default:
			return strings.ToLower(basic.Name())
		}
	}

	// Handle named types
	if named, ok := t.(*types.Named); ok {
		return strings.ToLower(named.Obj().Name())
	}

	// For other types, use the string representation and extract the type name
	typeStr := t.String()
	if idx := strings.LastIndex(typeStr, "."); idx != -1 {
		typeStr = typeStr[idx+1:]
	}
	// Remove pointer prefix if any
	typeStr = strings.TrimPrefix(typeStr, "*")

	return strings.ToLower(typeStr)
}

// isContextType checks if a type is context.Context
func isContextType(t types.Type) bool {
	if named, ok := t.(*types.Named); ok {
		if obj := named.Obj(); obj != nil && obj.Pkg() != nil {
			return obj.Pkg().Path() == "context" && obj.Name() == "Context"
		}
	}

	return false
}

// sortArguments sorts arguments deterministically: context.Context first, then by type name
func sortArguments(args []*Argument) {
	sort.Slice(args, func(i, j int) bool {
		iType := args[i].Type
		jType := args[j].Type

		// context.Context always comes first
		iIsContext := isContextType(iType)
		jIsContext := isContextType(jType)

		if iIsContext && !jIsContext {
			return true
		}
		if !iIsContext && jIsContext {
			return false
		}

		// For non-context types, sort by type name
		return iType.String() < jType.String()
	})
}

func (g *Graph) Build() (*Injector, error) {
	injector := &Injector{
		Name:          g.injectorName,
		IsReturnError: false,
	}

	// Collect all nodes and parameters using topological traversal
	nodeParams := make(map[*node][]*InjectorParam)
	variableNameCounter := 0
	buildVisited := make(map[*node]bool)

	// Traverse nodes in dependency order to collect parameters
	for n := range g.waitNodes.Iter {
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

			if n.providerSpec.IsReturnError {
				injector.IsReturnError = true
			}
		default:
			return nil, errors.New("invalid node")
		}

		nodeParams[n] = returnValues

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

	if injector.Return == nil {
		return nil, errors.New("no return value provider found")
	}

	// Build optimized execution chains using job chaining strategy
	injector.Stmts = g.buildOptimizedExecutionChains(nodeParams)

	return injector, nil
}

// buildOptimizedExecutionChains creates optimized execution chains using job chaining strategy
func (g *Graph) buildOptimizedExecutionChains(nodeParams map[*node][]*InjectorParam) []InjectorStmt {
	// Build dependency graph between nodes
	depGraph := g.buildNodeDependencyGraph()
	
	// Find optimal execution chains
	chains := g.findOptimalJobChains(depGraph, nodeParams)
	
	// Convert chains to statements
	return g.convertChainsToStatements(chains, nodeParams)
}

// buildNodeDependencyGraph creates a dependency graph between provider nodes
func (g *Graph) buildNodeDependencyGraph() map[*node][]*node {
	graph := make(map[*node][]*node)
	
	// Build adjacency list from edges
	for parent, edges := range g.edges {
		for _, edge := range edges {
			if edge.node.providerSpec != nil {
				graph[parent] = append(graph[parent], edge.node)
			}
			}
	}
	
	return graph
}

// ExecutionChain represents a sequence of jobs that can run in the same goroutine
type ExecutionChain struct {
	nodes      []*node
	isAsync    bool
	chainID    string
	waitFor    []*ExecutionChain // chains this chain must wait for
	completeCh *InjectorChannel   // channel to signal completion
}

// findOptimalJobChains identifies optimal execution chains using job chaining rules
func (g *Graph) findOptimalJobChains(depGraph map[*node][]*node, nodeParams map[*node][]*InjectorParam) []*ExecutionChain {
	visited := make(map[*node]bool)
	chains := make([]*ExecutionChain, 0)
	chainCounter := 0
	
	// Find provider nodes sorted by dependency order
	providerNodes := g.getProviderNodesInTopologicalOrder()
	
	for _, node := range providerNodes {
		if visited[node] || node.providerSpec == nil {
			continue
		}
		
		chain := g.buildChainFromNode(node, depGraph, visited, chainCounter)
		if chain != nil {
			chains = append(chains, chain)
			chainCounter++
		}
	}
	
	// Optimize chain dependencies for multiple parent scenarios
	g.optimizeChainDependencies(chains, depGraph)
	
	return chains
}

// getProviderNodesInTopologicalOrder returns provider nodes in topological order
func (g *Graph) getProviderNodesInTopologicalOrder() []*node {
	inDegree := make(map[*node]int)
	adjacency := make(map[*node][]*node)
	
	// Initialize in-degree count for provider nodes
	for _, n := range g.allNodes {
		if n.providerSpec != nil {
			inDegree[n] = len(n.providerSpec.Requires)
		}
	}
	
	// Build adjacency list from edges
	for parent, edges := range g.edges {
		for _, edge := range edges {
			if edge.node.providerSpec != nil {
				adjacency[parent] = append(adjacency[parent], edge.node)
			}
		}
	}
	
	// Use Kahn's algorithm for topological sorting
	queue := collection.NewQueue[*node]()
	result := make([]*node, 0)
	
	// Start with nodes that have no dependencies (including argument nodes)
	for _, n := range g.allNodes {
		if n.providerSpec != nil && inDegree[n] == 0 {
			queue.Push(n)
		} else if n.arg != nil {
			// Also process argument nodes to reduce in-degree of dependent providers
			queue.Push(n)
		}
	}
	
	for n := range queue.Iter {
		if n == nil {
			break
		}
		
		// Only add provider nodes to result
		if n.providerSpec != nil {
			result = append(result, n)
		}
		
		// Reduce in-degree for all dependent nodes
		for _, dependent := range adjacency[n] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue.Push(dependent)
			}
		}
	}
	
	return result
}

// buildChainFromNode builds an execution chain starting from a given node
func (g *Graph) buildChainFromNode(startNode *node, depGraph map[*node][]*node, visited map[*node]bool, chainID int) *ExecutionChain {
	if visited[startNode] || startNode.providerSpec == nil {
		return nil
	}
	
	chain := &ExecutionChain{
		nodes:   []*node{startNode},
		isAsync: startNode.providerSpec.IsAsync,
		chainID: fmt.Sprintf("chain_%d", chainID),
	}
	visited[startNode] = true
	
	current := startNode
	
	// Extend chain forward by following linear dependencies
	for {
		nextNode := g.findBestChainableNode(current, depGraph, visited)
		if nextNode == nil {
			break
		}
		
		// Apply job chaining rules
		currentAsync := current.providerSpec.IsAsync
		nextAsync := nextNode.providerSpec.IsAsync
		
		// Job chaining rules:
		// sync → sync: ✅, async → async: ✅, async → sync: ✅, sync → async: ❌
		if !currentAsync && nextAsync {
			break // Cannot chain sync → async
		}
		
		chain.nodes = append(chain.nodes, nextNode)
		visited[nextNode] = true
		current = nextNode
		
		// Update chain async status
		if nextAsync {
			chain.isAsync = true
		}
	}
	
	return chain
}

// findBestChainableNode finds the best child node to chain with
func (g *Graph) findBestChainableNode(currentNode *node, depGraph map[*node][]*node, visited map[*node]bool) *node {
	children := depGraph[currentNode]
	var candidates []*node
	
	// Filter unvisited children with single parent (linear dependency)
	for _, child := range children {
		if visited[child] || child.providerSpec == nil {
			continue
		}
		
		// Check if child has single parent
		parentCount := g.countParentNodes(child, depGraph)
		if parentCount == 1 {
			candidates = append(candidates, child)
		}
	}
	
	if len(candidates) == 0 {
		return nil
	}
	
	// For single candidate, return it
	if len(candidates) == 1 {
		return candidates[0]
	}
	
	// For multiple candidates, choose based on priority:
	// 1. Same async type (async → async, sync → sync)
	// 2. Async → sync (allowed but lower priority)
	currentAsync := currentNode.providerSpec.IsAsync
	
	for _, candidate := range candidates {
		candidateAsync := candidate.providerSpec.IsAsync
		if currentAsync == candidateAsync {
			return candidate // Prefer same type
		}
	}
	
	// Return first async → sync candidate
	for _, candidate := range candidates {
		candidateAsync := candidate.providerSpec.IsAsync
		if currentAsync && !candidateAsync {
			return candidate
		}
	}
	
	return candidates[0] // Fallback to first candidate
}

// countParentNodes counts how many nodes depend on this node
func (g *Graph) countParentNodes(node *node, depGraph map[*node][]*node) int {
	count := 0
	for _, children := range depGraph {
		for _, child := range children {
			if child == node {
				count++
			}
		}
	}
	return count
}

// optimizeChainDependencies optimizes chain dependencies for multiple parent scenarios
func (g *Graph) optimizeChainDependencies(chains []*ExecutionChain, depGraph map[*node][]*node) {
	// Build chain dependency graph
	chainByNode := make(map[*node]*ExecutionChain)
	for _, chain := range chains {
		for _, node := range chain.nodes {
			chainByNode[node] = chain
		}
	}
	
	// For each chain, find which other chains it depends on
	for _, chain := range chains {
		dependentChains := make(map[*ExecutionChain]bool)
		
		// Check dependencies of the first node in the chain
		if len(chain.nodes) > 0 {
			firstNode := chain.nodes[0]
			
			// Find all parent nodes that this chain depends on
			for parentNode := range depGraph {
				for _, childNode := range depGraph[parentNode] {
					if childNode == firstNode {
						// This chain depends on parentNode
						if parentChain, exists := chainByNode[parentNode]; exists && parentChain != chain {
							dependentChains[parentChain] = true
						}
					}
				}
			}
		}
		
		// Convert to slice
		for depChain := range dependentChains {
			chain.waitFor = append(chain.waitFor, depChain)
		}
	}
	
	// Assign completion channels to chains that others wait for
	for _, chain := range chains {
		if len(chain.waitFor) > 0 {
			for _, waitChain := range chain.waitFor {
				if waitChain.completeCh == nil {
					waitChain.completeCh = NewInjectorChannel(fmt.Sprintf("%s_complete", waitChain.chainID))
				}
			}
		}
	}
}

// convertChainsToStatements converts execution chains to InjectorStmt objects
func (g *Graph) convertChainsToStatements(chains []*ExecutionChain, nodeParams map[*node][]*InjectorParam) []InjectorStmt {
	var result []InjectorStmt
	
	for _, chain := range chains {
		if len(chain.nodes) == 1 {
			// Single node, create individual statement
			node := chain.nodes[0]
			returnValues := nodeParams[node]
			stmt := &InjectorProviderCallStmt{
				Provider:  node.providerSpec,
				Arguments: node.providerArgs,
				Returns:   returnValues,
				Channel:   chain.completeCh,
			}
			result = append(result, stmt)
		} else {
			// Multiple nodes, create chain statement
			chainStmts := make([]InjectorStmt, 0, len(chain.nodes))
			for _, node := range chain.nodes {
				returnValues := nodeParams[node]
				stmt := &InjectorProviderCallStmt{
					Provider:  node.providerSpec,
					Arguments: node.providerArgs,
					Returns:   returnValues,
				}
				chainStmts = append(chainStmts, stmt)
			}
			
			// Create input channels for waiting on dependencies
			inputs := make([]*InjectorChannel, 0, len(chain.waitFor))
			for _, waitChain := range chain.waitFor {
				if waitChain.completeCh != nil {
					inputs = append(inputs, waitChain.completeCh)
				}
			}
			
			chainStmt := &InjectorChainStmt{
				Statements: chainStmts,
				Inputs:     inputs,
			}
			result = append(result, chainStmt)
		}
	}
	
	return result
}








