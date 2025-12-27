package kessoku

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log/slog"
	"maps"
	"math"
	"slices"

	"github.com/mazrean/kessoku/internal/pkg/collection"
)

// createASTTypeExpr creates an AST type expression from a types.Type and updates existingImports
func createASTTypeExpr(pkg string, t types.Type, varPool *VarPool, imports map[string]*Import) (ast.Expr, error) {
	switch typ := t.(type) {
	case *types.Basic:
		return ast.NewIdent(typ.Name()), nil
	case *types.Pointer:
		expr, err := createASTTypeExpr(pkg, typ.Elem(), varPool, imports)
		if err != nil {
			return nil, fmt.Errorf("pointer element: %w", err)
		}

		return &ast.StarExpr{
			X: expr,
		}, nil
	case *types.Named:
		name := typ.Obj().Name()
		if objPkg := typ.Obj().Pkg(); objPkg != nil && objPkg.Path() != pkg {
			// For types from other packages, create a selector expression
			// Format: package.TypeName
			pkgPath := objPkg.Path()
			pkgName := objPkg.Name()

			// Check if package is already imported
			if imp, exists := imports[pkgPath]; exists {
				pkgName = imp.Name
			} else {
				newPkgName := varPool.GetName(pkgName)
				imports[pkgPath] = &Import{
					Name:          newPkgName,
					IsDefaultName: newPkgName == pkgName,
					IsUsed:        false, // Will be marked during code generation
				}
			}

			return &ast.SelectorExpr{
				X:   ast.NewIdent(pkgName),
				Sel: ast.NewIdent(name),
			}, nil
		}

		return ast.NewIdent(name), nil
	case *types.Alias:
		name := typ.Obj().Name()
		if objPkg := typ.Obj().Pkg(); objPkg != nil && objPkg.Path() != pkg {
			// For types from other packages, create a selector expression
			// Format: package.TypeName
			pkgPath := objPkg.Path()
			pkgName := objPkg.Name()

			// Check if package is already imported
			if imp, exists := imports[pkgPath]; exists {
				pkgName = imp.Name
			} else {
				newPkgName := varPool.GetName(pkgName)
				imports[pkgPath] = &Import{
					Name:          newPkgName,
					IsDefaultName: newPkgName == pkgName,
					IsUsed:        false, // Will be marked during code generation
				}
			}

			return &ast.SelectorExpr{
				X:   ast.NewIdent(pkgName),
				Sel: ast.NewIdent(name),
			}, nil
		}

		return ast.NewIdent(name), nil
	case *types.Slice:
		expr, err := createASTTypeExpr(pkg, typ.Elem(), varPool, imports)
		if err != nil {
			return nil, fmt.Errorf("slice element: %w", err)
		}

		return &ast.ArrayType{
			Elt: expr,
		}, nil
	case *types.Array:
		expr, err := createASTTypeExpr(pkg, typ.Elem(), varPool, imports)
		if err != nil {
			return nil, fmt.Errorf("array element: %w", err)
		}

		return &ast.ArrayType{
			Len: &ast.BasicLit{
				Kind:  token.INT,
				Value: fmt.Sprintf("%d", typ.Len()),
			},
			Elt: expr,
		}, nil
	case *types.Map:
		keyExpr, err := createASTTypeExpr(pkg, typ.Key(), varPool, imports)
		if err != nil {
			return nil, fmt.Errorf("map key: %w", err)
		}
		valueExpr, err := createASTTypeExpr(pkg, typ.Elem(), varPool, imports)
		if err != nil {
			return nil, fmt.Errorf("map value: %w", err)
		}

		return &ast.MapType{
			Key:   keyExpr,
			Value: valueExpr,
		}, nil
	case *types.Interface:
		methodFields := make([]*ast.Field, 0, typ.NumMethods())
		for method := range typ.Methods() {
			expr, err := createASTTypeExpr(pkg, method.Signature(), varPool, imports)
			if err != nil {
				return nil, fmt.Errorf("method signature: %w", err)
			}

			methodFields = append(methodFields, &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(method.Name())},
				Type:  expr,
			})
		}
		return &ast.InterfaceType{
			Methods: &ast.FieldList{
				List: methodFields,
			},
		}, nil
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
		expr, err := createASTTypeExpr(pkg, typ.Elem(), varPool, imports)
		if err != nil {
			return nil, fmt.Errorf("chan element: %w", err)
		}

		return &ast.ChanType{
			Dir:   dir,
			Value: expr,
		}, nil
	case *types.Signature:
		funcFields := make([]*ast.Field, 0, typ.Params().Len())
		for i := 0; i < typ.Params().Len(); i++ {
			expr, err := createASTTypeExpr(pkg, typ.Params().At(i).Type(), varPool, imports)
			if err != nil {
				return nil, fmt.Errorf("param %d: %w", i, err)
			}
			funcFields = append(funcFields, &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(fmt.Sprintf("arg%d", i))},
				Type:  expr,
			})
		}
		resultsFields := make([]*ast.Field, 0, typ.Results().Len())
		for i := 0; i < typ.Results().Len(); i++ {
			expr, err := createASTTypeExpr(pkg, typ.Results().At(i).Type(), varPool, imports)
			if err != nil {
				return nil, fmt.Errorf("result %d: %w", i, err)
			}
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
		}, nil
	case *types.Struct:
		fields := make([]*ast.Field, 0, typ.NumFields())
		for i := 0; i < typ.NumFields(); i++ {
			expr, err := createASTTypeExpr(pkg, typ.Field(i).Type(), varPool, imports)
			if err != nil {
				return nil, fmt.Errorf("field %d: %w", i, err)
			}
			fields = append(fields, &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(typ.Field(i).Name())},
				Type:  expr,
			})
		}
		return &ast.StructType{
			Fields: &ast.FieldList{
				List: fields,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", t.String())
	}
}

func CreateInjector(metaData *MetaData, build *BuildDirective, varPool *VarPool) (*Injector, error) {
	slog.Debug("CreateInjector", "build", build)
	for _, provider := range build.Providers {
		slog.Debug("provider", "provider", provider)
	}
	graph, err := NewGraph(metaData, build, varPool)
	if err != nil {
		return nil, fmt.Errorf("create graph: %w", err)
	}

	injector, err := graph.Build(metaData, varPool)
	if err != nil {
		return nil, fmt.Errorf("build injector: %w", err)
	}

	return injector, nil
}

type argument struct {
	Type        types.Type
	ASTTypeExpr ast.Expr
}

type node struct {
	arg          *argument
	providerSpec *ProviderSpec
	providerArgs []*InjectorCallArgument
	returnValues []*InjectorParam
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
	edges        map[*node][]*edgeNode
	reverseEdges map[*node][]*node
	returnType   *Return
	returnValue  *returnVal
	injectorName string
	nodes        []*node
}

func NewGraph(metaData *MetaData, build *BuildDirective, varPool *VarPool) (*Graph, error) {
	graph := &Graph{
		injectorName: build.InjectorName,
		returnType:   build.Return,
		edges:        make(map[*node][]*edgeNode),
		reverseEdges: make(map[*node][]*node),
	}

	type fnProvider struct {
		provider    *ProviderSpec
		returnIndex int
	}

	fnProviderMap := make(map[string]*fnProvider)
	for _, provider := range build.Providers {
		for groupIndex, typeGroup := range provider.Provides {
			for typeIndex, t := range typeGroup {
				if t == nil {
					return nil, fmt.Errorf("provider has nil type at group %d, index %d", groupIndex, typeIndex)
				}
				key := t.String()

				if existing, ok := fnProviderMap[key]; ok {
					// Allow the same provider to provide multiple types (e.g., concrete and interface)
					// but still error if different providers try to provide the same type
					if existing.provider != provider {
						return nil, fmt.Errorf("multiple providers provide %s", key)
					}
					// If it's the same provider, just update the return index to the first occurrence
					// This handles the case where bindProvider adds both concrete and interface types
					continue
				}

				fnProviderMap[key] = &fnProvider{
					provider:    provider,
					returnIndex: groupIndex,
				}
			}
		}
	}

	if build.Return.Type == nil {
		return nil, fmt.Errorf("return type is nil")
	}
	returnTypeKey := build.Return.Type.String()

	returnProvider, ok := fnProviderMap[returnTypeKey]
	if !ok {
		n, err := graph.autoAddMissingDependencies(metaData, build.Return.Type, varPool)
		if err != nil {
			return nil, fmt.Errorf("auto add missing return dependency: %w", err)
		}
		graph.returnValue = &returnVal{
			node:        n,
			returnIndex: 0,
		}
		graph.nodes = append(graph.nodes, n)
		return graph, nil
	}

	providerNodeMap := make(map[*ProviderSpec]*node)
	argNodeMap := make(map[string]*node)
	queue := collection.NewQueue[*node]()
	visited := make(map[*node]bool)

	returnNode := &node{
		providerSpec: returnProvider.provider,
		providerArgs: make([]*InjectorCallArgument, len(returnProvider.provider.Requires)),
	}
	graph.returnValue = &returnVal{
		node:        returnNode,
		returnIndex: returnProvider.returnIndex,
	}
	queue.Push(returnNode)
	graph.nodes = append(graph.nodes, returnNode)

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
			if t == nil {
				return nil, fmt.Errorf("provider has nil required type at index %d", i)
			}
			key := t.String()
			var (
				n2       *node
				srcIndex int
			)
			if provider, ok := fnProviderMap[key]; ok {
				n2, ok = providerNodeMap[provider.provider]
				if !ok {
					n2 = &node{
						providerSpec: provider.provider,
						providerArgs: make([]*InjectorCallArgument, len(provider.provider.Requires)),
					}
					providerNodeMap[provider.provider] = n2
					queue.Push(n2)
					graph.nodes = append(graph.nodes, n2)
				}

				srcIndex = provider.returnIndex
			} else if n2, ok = argNodeMap[key]; ok {
				srcIndex = 0
			} else {
				// Auto-detect missing dependency and create an argument for it
				var err error
				n2, err = graph.autoAddMissingDependencies(metaData, t, varPool)
				if err != nil {
					return nil, fmt.Errorf("auto add missing dependency as argument: %w", err)
				}

				argNodeMap[key] = n2
				queue.Push(n2)
				graph.nodes = append(graph.nodes, n2)
				srcIndex = 0
			}

			graph.edges[n2] = append(graph.edges[n2], &edgeNode{
				node:          n1,
				provideArgSrc: srcIndex,
				provideArgDst: i,
			})
			graph.reverseEdges[n1] = append(graph.reverseEdges[n1], n2)
		}
	}

	// Check for cycles in the dependency graph
	if err := graph.detectCycles(); err != nil {
		return nil, fmt.Errorf("dependency cycle detected: %w", err)
	}

	return graph, nil
}

// nodeColor represents the color of a node during DFS for cycle detection
type nodeColor int

const (
	white nodeColor = iota // unvisited
	gray                   // currently being processed
	black                  // completely processed
)

// CycleError represents an error when a dependency cycle is detected
type CycleError struct {
	Cycle []*node
}

func (e *CycleError) Error() string {
	if len(e.Cycle) == 0 {
		return "circular dependency detected"
	}

	var providerTypes []string
	for _, n := range e.Cycle {
		if n.providerSpec != nil && len(n.providerSpec.Provides) > 0 && len(n.providerSpec.Provides[0]) > 0 {
			// Use the first provided type as identifier
			providerTypes = append(providerTypes, n.providerSpec.Provides[0][0].String())
		} else if n.arg != nil {
			providerTypes = append(providerTypes, fmt.Sprintf("arg(%s)", n.arg.Type.String()))
		}
	}

	if len(providerTypes) == 0 {
		return "circular dependency detected"
	}

	// Build the cycle path: TypeA -> TypeB -> TypeC -> TypeA
	cyclePath := providerTypes[0]
	for i := 1; i < len(providerTypes); i++ {
		cyclePath += fmt.Sprintf(" -> %s", providerTypes[i])
	}
	cyclePath += fmt.Sprintf(" -> %s", providerTypes[0])

	return fmt.Sprintf("circular dependency detected: %s", cyclePath)
}

// detectCycles detects cycles in the dependency graph using DFS
func (g *Graph) detectCycles() error {
	colors := make(map[*node]nodeColor)
	parent := make(map[*node]*node)

	// Initialize all nodes as white (unvisited)
	for _, n := range g.nodes {
		colors[n] = white
	}

	// Run DFS from each unvisited node
	for _, n := range g.nodes {
		if colors[n] == white {
			if cycle := g.dfsCycleDetection(n, colors, parent); cycle != nil {
				return &CycleError{Cycle: cycle}
			}
		}
	}

	return nil
}

// dfsCycleDetection performs DFS and returns the cycle if found
func (g *Graph) dfsCycleDetection(node *node, colors map[*node]nodeColor, parent map[*node]*node) []*node {
	colors[node] = gray

	// Visit all adjacent nodes (dependencies)
	for _, edge := range g.edges[node] {
		neighbor := edge.node
		parent[neighbor] = node

		if colors[neighbor] == gray {
			// Back edge found - cycle detected
			return g.buildCyclePath(neighbor, node, parent)
		}

		if colors[neighbor] == white {
			if cycle := g.dfsCycleDetection(neighbor, colors, parent); cycle != nil {
				return cycle
			}
		}
	}

	colors[node] = black
	return nil
}

// buildCyclePath builds the cycle path from the detected back edge
func (g *Graph) buildCyclePath(cycleStart, cycleEnd *node, parent map[*node]*node) []*node {
	var cycle []*node

	// Start from the cycle end and work backwards to find the cycle
	current := cycleEnd
	for current != cycleStart {
		cycle = append([]*node{current}, cycle...)
		current = parent[current]
		if current == nil {
			// This shouldn't happen in a proper cycle, but just in case
			break
		}
	}

	// Add the cycle start to complete the cycle
	cycle = append([]*node{cycleStart}, cycle...)

	return cycle
}

// hasAsyncProviders checks if any providers in the graph are async
func (g *Graph) hasAsyncProviders() bool {
	for _, n := range g.nodes {
		if n.providerSpec != nil && n.providerSpec.IsAsync {
			return true
		}
	}
	return false
}

// injectContextArg injects context.Context as the first argument when async providers exist
func (g *Graph) injectContextArg(injector *Injector, metaData *MetaData, varPool *VarPool) error {
	if !g.hasAsyncProviders() {
		return nil
	}

	// Check if context.Context already exists in arguments
	var existingContextArg *InjectorArgument
	existingContextIdx := -1
	for i, arg := range injector.Args {
		if isContextType(arg.Type) {
			existingContextArg = arg
			existingContextIdx = i
			break
		}
	}

	// If context.Context already exists, move it to the first position
	if existingContextArg != nil {
		if existingContextIdx > 0 {
			// Move existing context argument to the first position in Args only
			// Note: We don't modify Params order - we use arg.Param directly
			injector.Args = append(injector.Args[:existingContextIdx], injector.Args[existingContextIdx+1:]...)
			injector.Args = append([]*InjectorArgument{existingContextArg}, injector.Args...)
		}
		// errgroup.WithContext(ctx) requires context.Context as the first argument
		// Use the Param from the argument directly, not from Params slice
		existingContextArg.Param.Ref(false)

		// Mark context import as used
		if imp, exists := metaData.Imports[contextPkgPath]; exists {
			imp.IsUsed = true
		}

		return nil
	}

	// Create context.Context type
	contextPkg := types.NewPackage(contextPkgPath, contextPkgName)
	contextObj := types.NewTypeName(0, contextPkg, contextTypeName, nil)
	contextType := types.NewNamed(contextObj, types.NewInterfaceType([]*types.Func{}, nil), nil)

	ctxPkgName := contextPkgName
	if imp, exists := metaData.Imports[contextPkgPath]; exists {
		ctxPkgName = imp.Name
	} else {
		newPkgName := varPool.GetName(contextPkgName)
		metaData.Imports[contextPkgPath] = &Import{
			Name:          newPkgName,
			IsDefaultName: newPkgName == contextPkgName,
			IsUsed:        false, // Will be marked during code generation
		}
		ctxPkgName = newPkgName
	}
	// Mark context import as used since we're injecting context.Context
	if imp, exists := metaData.Imports[contextPkgPath]; exists {
		imp.IsUsed = true
	}

	// Create AST expression for context.Context
	contextExpr := &ast.SelectorExpr{
		X:   ast.NewIdent(ctxPkgName),
		Sel: ast.NewIdent(contextTypeName),
	}

	// Create context parameter
	contextParam := NewInjectorParamWithImports([]types.Type{contextType}, true, metaData.Package.Path, metaData.Imports, varPool)
	// errgroup.WithContext(ctx) requires context.Context as the first argument
	contextParam.Ref(false)

	// Create context argument
	contextArg := &InjectorArgument{
		Param:       contextParam,
		Type:        contextType,
		ASTTypeExpr: contextExpr,
	}

	// Insert context as the first argument
	injector.Args = append([]*InjectorArgument{contextArg}, injector.Args...)
	injector.Params = append([]*InjectorParam{contextParam}, injector.Params...)

	return nil
}

func (g *Graph) autoAddMissingDependencies(metaData *MetaData, t types.Type, varPool *VarPool) (*node, error) {
	// Auto-detect missing dependency and create an argument for it
	expr, err := createASTTypeExpr(metaData.Package.Path, t, varPool, metaData.Imports)
	if err != nil {
		return nil, fmt.Errorf("create AST type expr: %w", err)
	}

	return &node{
		arg: &argument{
			Type:        t,
			ASTTypeExpr: expr,
		},
	}, nil
}

func (g *Graph) Build(metaData *MetaData, varPool *VarPool) (*Injector, error) {
	injector := &Injector{
		Name:          g.injectorName,
		IsReturnError: g.isReturnError(),
	}

	maxAnchainSize := g.findMaximumAntichainSize()
	pools := make([][]*node, maxAnchainSize)

	initialProvidedNodes := make(map[*node]struct{})
	for _, n := range g.nodes {
		if n.providerSpec == nil {
			initialProvidedNodes[n] = struct{}{}
		}
	}

	poolProvidedNodes := make([]map[*node]struct{}, maxAnchainSize)
	for i := range poolProvidedNodes {
		poolProvidedNodes[i] = maps.Clone(initialProvidedNodes)
	}

	nodeProvidedNodes := make(map[*node]map[*node]struct{}, len(g.nodes))
	nodeToPoolIdx := make(map[*node]int, len(g.nodes))

	// First pass: assign nodes to pools and collect return values
	for n := range g.topologicalSortIter() {
		slog.Debug("Processing node", "node", n)

		var (
			returnValues  []*InjectorParam
			providedNodes map[*node]struct{}
			poolIdx       int
		)
		switch {
		case n.arg != nil:
			providedNodes = initialProvidedNodes
			poolIdx = -1 // Arguments are not in any pool

			param := NewInjectorParamWithImports([]types.Type{n.arg.Type}, true, metaData.Package.Path, metaData.Imports, varPool)
			injector.Params = append(injector.Params, param)
			returnValues = append(returnValues, param)

			injector.Args = append(injector.Args, &InjectorArgument{
				Param:       param,
				Type:        n.arg.Type,
				ASTTypeExpr: n.arg.ASTTypeExpr,
			})
		case n.providerSpec != nil:
			poolIdx = g.findOptimalPool(n, pools, poolProvidedNodes)
			pools[poolIdx] = append(pools[poolIdx], n)

			providedNodes = poolProvidedNodes[poolIdx]
			poolProvidedNodes[poolIdx][n] = struct{}{}

			returnValues = make([]*InjectorParam, 0, len(n.providerSpec.Provides))
			for _, types := range n.providerSpec.Provides {
				param := NewInjectorParamWithImports(types, false, metaData.Package.Path, metaData.Imports, varPool)
				injector.Params = append(injector.Params, param)
				injector.Vars = append(injector.Vars, param)
				returnValues = append(returnValues, param)
			}

			if n.providerSpec.IsReturnError {
				injector.IsReturnError = true
			}
		default:
			return nil, errors.New("invalid node")
		}

		n.returnValues = returnValues
		nodeToPoolIdx[n] = poolIdx

		// Mark current node as provided before processing edges
		providedNodes[n] = struct{}{}
		nodeProvidedNodes[n] = maps.Clone(providedNodes)

		if n == g.returnValue.node {
			returnValues[g.returnValue.returnIndex].Ref(false)
			injector.Return = &InjectorReturn{
				Param:  returnValues[g.returnValue.returnIndex],
				Return: g.returnType,
			}
		}
	}

	// Second pass: set up dependencies with correct IsWait flags
	for n := range g.topologicalSortIter() {
		providedNodes := nodeProvidedNodes[n]

		for _, edge := range g.edges[n] {
			_, isProvided := providedNodes[edge.node]

			param := n.returnValues[edge.provideArgSrc]

			// Check if the dependency and dependent are in the same pool
			// If they are in the same pool, no need to wait for channels
			dependencyPoolIdx := nodeToPoolIdx[n]
			dependentPoolIdx := nodeToPoolIdx[edge.node]
			inSamePool := dependencyPoolIdx != -1 && dependentPoolIdx != -1 && dependencyPoolIdx == dependentPoolIdx

			// Only wait if not provided and not in the same pool
			shouldWait := !isProvided && !inSamePool

			edge.node.providerArgs[edge.provideArgDst] = &InjectorCallArgument{
				Param:  param,
				IsWait: shouldWait,
			}
			param.Ref(shouldWait)
		}
	}

	if injector.Return == nil {
		return nil, errors.New("no return value provider found")
	}

	var err error
	injector.Stmts, err = g.buildStmts(pools, nodeProvidedNodes, initialProvidedNodes)
	if err != nil {
		return nil, fmt.Errorf("build statements: %w", err)
	}

	// Inject context.Context argument if async providers exist
	err = g.injectContextArg(injector, metaData, varPool)
	if err != nil {
		return nil, fmt.Errorf("inject context argument: %w", err)
	}

	return injector, nil
}

func (g *Graph) isReturnError() bool {
	for _, node := range g.nodes {
		if node.providerSpec != nil && node.providerSpec.IsReturnError {
			return true
		}
	}

	return false
}

// findMaximumAntichainSize finds the maximum antichain using level-based approach
func (g *Graph) findMaximumAntichainSize() uint64 {
	node2Idx := make(map[*node]int, len(g.nodes))
	for i, n := range g.nodes {
		node2Idx[n] = i
	}

	adj := make([][]int, len(g.nodes))
	for n, edges := range g.edges {
		for _, edge := range edges {
			adj[node2Idx[n]] = append(adj[node2Idx[n]], node2Idx[edge.node])
		}
	}

	matchR := make([]int, len(g.nodes))
	for i := range matchR {
		matchR[i] = -1
	}

	maxAntichainSize := len(g.nodes)
	for u := range g.nodes {
		used := make([]bool, len(g.nodes))
		if g.findAugmentingPath(u, used, matchR, adj) {
			maxAntichainSize--
		}
	}

	return uint64(maxAntichainSize)
}

func (g *Graph) findAugmentingPath(u int, used []bool, matchR []int, adj [][]int) bool {
	for _, v := range adj[u] {
		if used[v] {
			continue
		}

		used[v] = true
		if matchR[v] == -1 || g.findAugmentingPath(matchR[v], used, matchR, adj) {
			matchR[v] = u
			return true
		}
	}

	return false
}

func (g *Graph) topologicalSortIter() func(yield func(*node) bool) {
	type requireCounter struct {
		probidedArg []bool
		count       int
	}
	waitNodes := collection.NewQueue[*node]()
	requireCounts := make(map[*node]*requireCounter)
	visited := make(map[*node]struct{})

	for _, n := range g.nodes {
		requireCount := len(g.reverseEdges[n])
		requireCounts[n] = &requireCounter{
			count:       requireCount,
			probidedArg: make([]bool, len(n.providerArgs)),
		}

		if requireCount == 0 {
			waitNodes.Push(n)
		}
	}

	return func(yield func(*node) bool) {
		for n := range waitNodes.Iter {
			if _, ok := visited[n]; ok {
				continue
			}
			visited[n] = struct{}{}

			for _, edge := range g.edges[n] {
				counter := requireCounts[edge.node]
				if counter == nil || counter.probidedArg[edge.provideArgDst] {
					continue
				}

				counter.count--
				counter.probidedArg[edge.provideArgDst] = true
				if counter.count == 0 {
					waitNodes.Push(edge.node)
				}
			}

			if !yield(n) {
				return
			}
		}
	}
}

// findOptimalPool finds the optimal pool for a job considering async/sync constraints
func (g *Graph) findOptimalPool(n *node, pools [][]*node, poolProvidedNodes []map[*node]struct{}) int {
	// Skip pool assignment for argument nodes - they don't need pool scheduling
	if n.providerSpec == nil {
		return 0
	}

	dependencies := g.reverseEdges[n]

	maxProvidedCount := 0
	maxProvidedPools := make([]int, 0)
	for i, providedNodeMap := range poolProvidedNodes {
		providedCount := 0
		for _, dependency := range dependencies {
			if _, ok := providedNodeMap[dependency]; ok {
				providedCount++
			}
		}

		if !n.providerSpec.IsAsync && len(pools[i]) == 0 {
			continue
		}

		switch {
		case providedCount > maxProvidedCount:
			maxProvidedCount = providedCount
			maxProvidedPools = []int{i}
		case providedCount == maxProvidedCount:
			maxProvidedPools = append(maxProvidedPools, i)
		}
	}

	if len(maxProvidedPools) == 0 {
		return 0
	}

	if maxProvidedCount == len(dependencies) {
	POOL_LOOP:
		for _, poolIdx := range maxProvidedPools {
			if !n.providerSpec.IsAsync {
				return poolIdx
			}

			for i := range pools[poolIdx] {
				nd := pools[poolIdx][len(pools[poolIdx])-1-i]
				if slices.Contains(dependencies, nd) {
					return poolIdx
				}
				if nd.providerSpec.IsAsync {
					continue POOL_LOOP
				}
			}
			if poolIdx == 0 {
				return 0
			}
		}
	}

	if n.providerSpec.IsAsync {
		for i, pool := range pools {
			if len(pool) == 0 {
				return i
			}
		}
	}

	minSize := math.MaxInt
	minSizePool := 0
	for _, poolIdx := range maxProvidedPools {
		if !n.providerSpec.IsAsync && len(pools[poolIdx]) == 0 {
			continue
		}
		if len(pools[poolIdx]) < minSize {
			minSize = len(pools[poolIdx])
			minSizePool = poolIdx
		}
	}

	return minSizePool
}

func (g *Graph) buildStmts(pools [][]*node, nodeProvidedNodes map[*node]map[*node]struct{}, initialProvidedNodes map[*node]struct{}) ([]InjectorStmt, error) {
	visited := make([]bool, len(pools))
	for i, pool := range pools {
		visited[i] = len(pool) == 0
	}

	poolDependencyMap := make(map[*node][]int, len(pools))
	for i, pool := range pools {
		if visited[i] {
			continue
		}

		firstNode := pool[0]
		for _, dependency := range g.reverseEdges[firstNode] {
			poolDependencyMap[dependency] = append(poolDependencyMap[dependency], i)
		}
	}

	// Find all pools that can start immediately
	initialPoolIdxs := make([]int, 0, len(pools))
	for i, pool := range pools {
		if visited[i] {
			continue
		}

		firstNode := pool[0]
		allDependenciesSatisfied := true
		for _, dependency := range g.reverseEdges[firstNode] {
			if _, ok := initialProvidedNodes[dependency]; !ok {
				allDependenciesSatisfied = false
				break
			}
		}

		if allDependenciesSatisfied {
			initialPoolIdxs = append(initialPoolIdxs, i)
		}
	}

	if len(initialPoolIdxs) == 0 {
		return nil, errors.New("no initial pools found")
	}

	// Find sync pool to execute first
	syncPoolIdx := -1
	for _, poolIdx := range initialPoolIdxs {
		if !pools[poolIdx][0].providerSpec.IsAsync {
			syncPoolIdx = poolIdx
			break
		}
	}

	stmts := make([]InjectorStmt, 0)

	// If there's a sync pool, execute it first
	var parentStmts []InjectorStmt
	if syncPoolIdx != -1 {
		visited[syncPoolIdx] = true

		var err error
		parentStmts, err = g.buildPoolStmts(pools[syncPoolIdx], pools, visited, poolDependencyMap, nodeProvidedNodes)
		if err != nil {
			return nil, fmt.Errorf("build sync pool statements: %w", err)
		}
	} else {
		// If no sync pool, just start with the first available async pool
		parentPoolIdx := initialPoolIdxs[0]
		visited[parentPoolIdx] = true

		var err error
		parentStmts, err = g.buildPoolStmts(pools[parentPoolIdx], pools, visited, poolDependencyMap, nodeProvidedNodes)
		if err != nil {
			return nil, fmt.Errorf("build first async pool statements: %w", err)
		}
	}

	// Then add all remaining ready async pools as chain statements
	for _, poolIdx := range initialPoolIdxs {
		if visited[poolIdx] {
			continue // Already processed
		}

		visited[poolIdx] = true
		subStmts, err := g.buildPoolStmts(pools[poolIdx], pools, visited, poolDependencyMap, nodeProvidedNodes)
		if err != nil {
			return nil, fmt.Errorf("build async pool statements: %w", err)
		}

		stmts = append(stmts, &InjectorChainStmt{
			Statements: subStmts,
		})
	}

	stmts = append(stmts, parentStmts...)

	return stmts, nil
}

func (g *Graph) buildPoolStmts(pool []*node, pools [][]*node, visited []bool, poolDependencyMap map[*node][]int, nodeProvidedNodes map[*node]map[*node]struct{}) ([]InjectorStmt, error) {
	stmts := make([]InjectorStmt, 0, len(pool))

	for _, n := range pool {
		if n.providerSpec == nil {
			// This is an argument node, skip it
			continue
		}

		stmts = append(stmts, &InjectorProviderCallStmt{
			Provider:  n.providerSpec,
			Arguments: n.providerArgs,
			Returns:   n.returnValues,
		})

		// Check if this node's execution enables any dependency pools to start
		for _, poolIdx := range poolDependencyMap[n] {
			if visited[poolIdx] {
				continue
			}

			firstNode := pools[poolIdx][0]
			// Check if all dependencies of the pool's first node are now satisfied
			// After executing node n, check if all dependencies are available
			currentProvidedNodes := maps.Clone(nodeProvidedNodes[n])
			currentProvidedNodes[n] = struct{}{} // Add the just-executed node

			allDependenciesSatisfied := true
			for _, dependency := range g.reverseEdges[firstNode] {
				if _, ok := currentProvidedNodes[dependency]; !ok {
					allDependenciesSatisfied = false
					break
				}
			}

			if allDependenciesSatisfied {
				visited[poolIdx] = true
				subStmts, err := g.buildPoolStmts(pools[poolIdx], pools, visited, poolDependencyMap, nodeProvidedNodes)
				if err != nil {
					return nil, fmt.Errorf("build pool statements: %w", err)
				}
				stmts = append(stmts, &InjectorChainStmt{
					Statements: subStmts,
				})
			}
		}
	}

	return stmts, nil
}
