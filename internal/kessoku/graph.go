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
		// Check by name first for byte and rune aliases
		switch basic.Name() {
		case "byte":
			return "b"
		case "rune":
			return "r"
		}

		// Then check by kind
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
