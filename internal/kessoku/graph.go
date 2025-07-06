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
	allNodes       []*node // Track all nodes in the graph
}

func NewGraph(metaData *MetaData, build *BuildDirective) (*Graph, error) {
	graph := &Graph{
		injectorName:   build.InjectorName,
		returnType:     build.Return,
		waitNodes:      collection.NewQueue[*node](),
		waitNodesAdded: make(map[*node]bool),
		edges:          make(map[*node][]*edgeNode),
		allNodes:       make([]*node, 0),
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
		Name:           g.injectorName,
		IsReturnError:  false,
		ExecutionPlans: make(map[int]*ParallelExecutionPlan),
	}

	// Analyze parallel execution groups
	g.analyzeParallelGroups()

	// Check for existing context.Context and determine if async providers need context
	g.analyzeContextRequirements(injector)

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
	
	// Build statements in correct dependency order using topological sort
	orderedProviderNodes := g.topologicalSortNodes(nodeParams)
	for _, n := range orderedProviderNodes {
		if n.providerSpec != nil {
			returnValues := nodeParams[n]
			injector.Stmts = append(injector.Stmts, &InjectorStmt{
				Provider:      n.providerSpec,
				Arguments:     n.providerArgs,
				Returns:       returnValues,
				ParallelGroup: n.parallelGroup,
			})
		}
	}

	// Analyze execution plans for optimized parallel execution
	g.analyzeExecutionPlans(injector)

	return injector, nil
}

// analyzeParallelGroups analyzes the DAG to determine parallel execution groups
func (g *Graph) analyzeParallelGroups() {
	// Initialize parallel groups
	parallelGroupCounter := 1
	
	// Use all nodes in the graph, not just waitNodes
	currentNodes := g.allNodes
	
	// Simple parallel group assignment for async providers
	// Find nodes that can be executed in parallel (same level in dependency graph)
	for _, n := range currentNodes {
		if n.providerSpec != nil && n.providerSpec.IsAsync {
			// Check if there are other async nodes at the same level
			asyncPeers := make([]*node, 0)
			for _, peer := range currentNodes {
				if peer != n && peer.providerSpec != nil && peer.providerSpec.IsAsync && peer.requireCount == n.requireCount {
					asyncPeers = append(asyncPeers, peer)
				}
			}
			
			if len(asyncPeers) > 0 {
				// Assign same parallel group to async nodes at the same level
				if n.parallelGroup == 0 {
					groupID := parallelGroupCounter
					parallelGroupCounter++
					n.parallelGroup = groupID
					
					for _, peer := range asyncPeers {
						if peer.parallelGroup == 0 {
							peer.parallelGroup = groupID
						}
					}
				}
			} else {
				// Single async node can still benefit from async execution
				if n.parallelGroup == 0 {
					n.parallelGroup = parallelGroupCounter
					parallelGroupCounter++
				}
			}
		} else {
			// Non-async nodes get sequential execution
			n.parallelGroup = 0
		}
	}
}

// canExecuteSequentiallyAfter checks if node1 can be executed immediately after node2 in the same goroutine
func (g *Graph) canExecuteSequentiallyAfter(node1, node2 *node) bool {
	// Check if node1 depends directly on node2's output
	for _, arg := range node1.providerArgs {
		if arg != nil {
			// Find which node provides this argument
			for _, edge := range g.edges[node2] {
				if edge.node == node1 {
					// node1 depends on node2 - they can be in the same goroutine
					return true
				}
			}
		}
	}
	return false
}

// analyzeContextRequirements checks for existing context.Context and determines context needs
func (g *Graph) analyzeContextRequirements(injector *Injector) {
	// Check if any async providers need parallel execution
	hasAsyncProviders := false
	for _, n := range g.allNodes {
		if n.providerSpec != nil && n.providerSpec.IsAsync && n.parallelGroup > 0 {
			hasAsyncProviders = true
			break
		}
	}

	if !hasAsyncProviders {
		return
	}

	// Check if context.Context already exists in arguments
	existingContextArg := ""
	for _, n := range g.allNodes {
		if n.arg != nil && isContextType(n.arg.Type) {
			existingContextArg = n.arg.Name
			break
		}
	}

	// Check if context.Context is provided by any provider
	existingContextProvider := ""
	for _, n := range g.allNodes {
		if n.providerSpec != nil {
			for _, providedType := range n.providerSpec.Provides {
				if isContextType(providedType) {
					// Find the corresponding return parameter name
					for _, stmt := range injector.Stmts {
						if stmt.Provider == n.providerSpec {
							for i, ret := range stmt.Returns {
								if i < len(n.providerSpec.Provides) && isContextType(n.providerSpec.Provides[i]) {
									existingContextProvider = ret.Name()
									break
								}
							}
							break
						}
					}
					break
				}
			}
			if existingContextProvider != "" {
				break
			}
		}
	}

	if existingContextArg != "" {
		injector.HasExistingContext = true
		injector.ContextParamName = existingContextArg
		injector.IsReturnError = true
	} else if existingContextProvider != "" {
		injector.HasExistingContext = true
		injector.ContextParamName = existingContextProvider
		injector.IsReturnError = true
	} else {
		// No existing context found, need to add context parameter
		injector.HasExistingContext = false
		injector.ContextParamName = "ctx"
		injector.IsReturnError = true
	}
}

// analyzeExecutionPlans creates optimized execution plans for parallel groups
func (g *Graph) analyzeExecutionPlans(injector *Injector) {
	// Group statements by parallel group
	parallelGroups := make(map[int][]*InjectorStmt)
	for _, stmt := range injector.Stmts {
		if stmt.ParallelGroup > 0 {
			parallelGroups[stmt.ParallelGroup] = append(parallelGroups[stmt.ParallelGroup], stmt)
		}
	}

	// Build execution plans for each parallel group
	for groupID, statements := range parallelGroups {
		if len(statements) > 1 {
			plan := g.buildExecutionPlan(groupID, statements)
			injector.ExecutionPlans[groupID] = plan
		}
	}
}

// buildExecutionPlan creates a detailed execution plan for a parallel group
func (g *Graph) buildExecutionPlan(groupID int, statements []*InjectorStmt) *ParallelExecutionPlan {
	// Build dependency chains within the parallel group
	chains := g.buildDependencyChains(statements)
	
	// Identify channel communication needs between chains
	channels := g.identifyChannelCommunication(chains)
	
	return &ParallelExecutionPlan{
		GroupID:  groupID,
		Chains:   chains,
		Channels: channels,
	}
}

// buildDependencyChains analyzes statements to build dependency chains
func (g *Graph) buildDependencyChains(statements []*InjectorStmt) []*DependencyChain {
	// Create a map of parameter dependencies
	paramProviders := make(map[*InjectorParam]*InjectorStmt)
	paramConsumers := make(map[*InjectorParam][]*InjectorStmt)
	
	// Build dependency mapping
	for _, stmt := range statements {
		// Map each return parameter to its provider
		for _, ret := range stmt.Returns {
			paramProviders[ret] = stmt
		}
		
		// Map each argument parameter to its consumers
		for _, arg := range stmt.Arguments {
			paramConsumers[arg] = append(paramConsumers[arg], stmt)
		}
	}
	
	// Find statements with no dependencies within this parallel group
	independentStmts := make([]*InjectorStmt, 0)
	for _, stmt := range statements {
		hasInternalDependency := false
		for _, arg := range stmt.Arguments {
			if provider, exists := paramProviders[arg]; exists {
				// This statement depends on another statement in the same parallel group
				_ = provider
				hasInternalDependency = true
				break
			}
		}
		if !hasInternalDependency {
			independentStmts = append(independentStmts, stmt)
		}
	}
	
	// Build chains starting from independent statements
	chains := make([]*DependencyChain, 0)
	visited := make(map[*InjectorStmt]bool)
	chainIDCounter := 1
	
	for _, stmt := range independentStmts {
		if !visited[stmt] {
			chain := g.buildChainFromStatement(stmt, paramConsumers, visited, chainIDCounter)
			chains = append(chains, chain)
			chainIDCounter++
		}
	}
	
	// Handle any remaining unvisited statements (shouldn't happen in well-formed DAG)
	for _, stmt := range statements {
		if !visited[stmt] {
			chain := &DependencyChain{
				ID:         chainIDCounter,
				Statements: []*InjectorStmt{stmt},
				Inputs:     make([]*ChannelInput, 0),
				Outputs:    make([]*ChannelOutput, 0),
			}
			chains = append(chains, chain)
			chainIDCounter++
			visited[stmt] = true
		}
	}
	
	return chains
}

// buildChainFromStatement builds a dependency chain starting from a given statement
func (g *Graph) buildChainFromStatement(stmt *InjectorStmt, paramConsumers map[*InjectorParam][]*InjectorStmt, visited map[*InjectorStmt]bool, chainID int) *DependencyChain {
	chain := &DependencyChain{
		ID:         chainID,
		Statements: make([]*InjectorStmt, 0),
		Inputs:     make([]*ChannelInput, 0),
		Outputs:    make([]*ChannelOutput, 0),
	}
	
	// DFS to build the chain
	var buildChain func(*InjectorStmt)
	buildChain = func(current *InjectorStmt) {
		if visited[current] {
			return
		}
		visited[current] = true
		chain.Statements = append(chain.Statements, current)
		
		// Find direct consumers within the same parallel group
		for _, ret := range current.Returns {
			if consumers, exists := paramConsumers[ret]; exists {
				for _, consumer := range consumers {
					if !visited[consumer] {
						// Check if this consumer only depends on the current statement
						// within this parallel group (no other internal dependencies)
						hasOtherDependencies := false
						for _, arg := range consumer.Arguments {
							if arg != ret {
								// Check if this argument is provided by another statement in the group
								for _, otherStmt := range chain.Statements {
									for _, otherRet := range otherStmt.Returns {
										if arg == otherRet && otherStmt != current {
											hasOtherDependencies = true
											break
										}
									}
									if hasOtherDependencies {
										break
									}
								}
							}
						}
						
						if !hasOtherDependencies {
							buildChain(consumer)
						}
					}
				}
			}
		}
	}
	
	buildChain(stmt)
	return chain
}

// identifyChannelCommunication identifies necessary channel communication between chains
func (g *Graph) identifyChannelCommunication(chains []*DependencyChain) map[string]string {
	channels := make(map[string]string)
	channelCounter := 1
	
	// Build parameter to chain mapping
	paramToChain := make(map[*InjectorParam]*DependencyChain)
	for _, chain := range chains {
		for _, stmt := range chain.Statements {
			for _, ret := range stmt.Returns {
				paramToChain[ret] = chain
			}
		}
	}
	
	// Identify cross-chain dependencies
	for _, chain := range chains {
		for _, stmt := range chain.Statements {
			for _, arg := range stmt.Arguments {
				if sourceChain, exists := paramToChain[arg]; exists && sourceChain != chain {
					// This is a cross-chain dependency
					channelName := fmt.Sprintf("ch%d", channelCounter)
					channelCounter++
					
					// Add output to source chain
					sourceChain.Outputs = append(sourceChain.Outputs, &ChannelOutput{
						ToChainID:   chain.ID,
						ParamName:   arg.Name(),
						ParamType:   arg,
						ChannelName: channelName,
					})
					
					// Add input to target chain
					chain.Inputs = append(chain.Inputs, &ChannelInput{
						FromChainID: sourceChain.ID,
						ParamName:   arg.Name(),
						ParamType:   arg,
						ChannelName: channelName,
					})
					
					channels[arg.Name()] = channelName
				}
			}
		}
	}
	
	return channels
}

// topologicalSortNodes sorts nodes in dependency order for proper code generation
func (g *Graph) topologicalSortNodes(nodeParams map[*node][]*InjectorParam) []*node {
	// Build a set of all provider nodes
	providerNodes := make([]*node, 0)
	for n := range nodeParams {
		if n.providerSpec != nil {
			providerNodes = append(providerNodes, n)
		}
	}
	
	// Build dependency graph between nodes based on parameter dependencies
	nodeDeps := make(map[*node]map[*node]bool) // nodeDeps[a][b] = true means node a depends on node b
	paramToNode := make(map[*InjectorParam]*node)
	
	// Map parameters to their provider nodes
	for n, params := range nodeParams {
		if n.providerSpec != nil {
			for _, param := range params {
				paramToNode[param] = n
			}
		}
	}
	
	// Initialize dependencies
	for _, n := range providerNodes {
		nodeDeps[n] = make(map[*node]bool)
	}
	
	// Analyze dependencies between nodes
	for _, n := range providerNodes {
		for _, arg := range n.providerArgs {
			if arg != nil {
				if providerNode, exists := paramToNode[arg]; exists && providerNode != n {
					// This node depends on the provider node
					nodeDeps[n][providerNode] = true
				}
			}
		}
	}
	
	// Perform topological sort using Kahn's algorithm
	inDegree := make(map[*node]int)
	for _, n := range providerNodes {
		inDegree[n] = 0
	}
	
	// Calculate in-degrees
	for _, n := range providerNodes {
		for range nodeDeps[n] {
			inDegree[n]++
		}
	}
	
	// Find nodes with no dependencies
	queue := make([]*node, 0)
	for _, n := range providerNodes {
		if inDegree[n] == 0 {
			queue = append(queue, n)
		}
	}
	
	// Process queue
	result := make([]*node, 0)
	for len(queue) > 0 {
		// Sort queue for deterministic output
		sort.Slice(queue, func(i, j int) bool {
			// Process non-async nodes first, then by parallel group, then by parameter name
			if queue[i].providerSpec.IsAsync != queue[j].providerSpec.IsAsync {
				return !queue[i].providerSpec.IsAsync // non-async first
			}
			if queue[i].parallelGroup != queue[j].parallelGroup {
				return queue[i].parallelGroup < queue[j].parallelGroup
			}
			// Compare by first return parameter name for deterministic ordering
			params_i := nodeParams[queue[i]]
			params_j := nodeParams[queue[j]]
			if len(params_i) > 0 && len(params_j) > 0 {
				return params_i[0].Name() < params_j[0].Name()
			}
			return false
		})
		
		currentNode := queue[0]
		queue = queue[1:]
		result = append(result, currentNode)
		
		// Update in-degrees for dependent nodes
		for _, n := range providerNodes {
			if nodeDeps[n][currentNode] {
				inDegree[n]--
				if inDegree[n] == 0 {
					queue = append(queue, n)
				}
			}
		}
	}
	
	// Handle any remaining nodes (shouldn't happen in a valid DAG)
	for _, n := range providerNodes {
		found := false
		for _, resultNode := range result {
			if resultNode == n {
				found = true
				break
			}
		}
		if !found {
			result = append(result, n)
		}
	}
	
	return result
}
