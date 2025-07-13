package kessoku

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"
)

// createTestTypes is already defined in generator_test.go

func TestNewGraph(t *testing.T) {
	t.Parallel()

	configType, serviceType, _ := createTestTypes()

	tests := []struct {
		build         *BuildDirective
		name          string
		expectedName  string
		errorContains string
		expectError   bool
	}{
		{
			name: "basic dependency graph",
			build: &BuildDirective{
				InjectorName: "InitializeService",
				Return: &Return{
					Type: serviceType,
				},
				Providers: []*ProviderSpec{
					{
						Type:          ProviderTypeFunction,
						Provides:      []types.Type{configType},
						Requires:      []types.Type{},
						IsReturnError: false,
					},
					{
						Type:          ProviderTypeFunction,
						Provides:      []types.Type{serviceType},
						Requires:      []types.Type{configType},
						IsReturnError: false,
					},
				},
			},
			expectError:  false,
			expectedName: "InitializeService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			metaData := &MetaData{
				Package: Package{
					Name: "test",
					Path: "test",
				},
				Imports: make(map[string]*ast.ImportSpec),
			}
			
			graph, err := NewGraph(metaData, tt.build)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				if tt.errorContains != "" && !containsError(err.Error(), tt.errorContains) {
					t.Fatalf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if graph == nil {
				t.Fatal("Expected graph to be non-nil")
			}

			if graph.injectorName != tt.expectedName {
				t.Errorf("Expected name %q, got %q", tt.expectedName, graph.injectorName)
			}
		})
	}
}

func containsError(err, substring string) bool {
	return len(err) >= len(substring) && err[:len(substring)] == substring
}

func TestCreateASTTypeExpr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		pkg             string
		typeExpr        types.Type
		expectedImports []string
		shouldError     bool
		expectNil       bool
	}{
		{
			name:            "basic int type",
			pkg:             "main",
			typeExpr:        types.Typ[types.Int],
			expectedImports: []string{},
			shouldError:     false,
		},
		{
			name:            "basic string type",
			pkg:             "main",
			typeExpr:        types.Typ[types.String],
			expectedImports: []string{},
			shouldError:     false,
		},
		{
			name: "pointer to basic type",
			pkg:  "main",
			typeExpr: types.NewPointer(types.Typ[types.Int]),
			expectedImports: []string{},
			shouldError:     false,
		},
		{
			name: "named type in same package",
			pkg:  "main",
			typeExpr: func() types.Type {
				obj := types.NewTypeName(0, types.NewPackage("main", "main"), "Service", nil)
				return types.NewNamed(obj, types.NewStruct(nil, nil), nil)
			}(),
			expectedImports: []string{},
			shouldError:     false,
		},
		{
			name: "named type in different package",
			pkg:  "main",
			typeExpr: func() types.Type {
				pkg := types.NewPackage("fmt", "fmt")
				obj := types.NewTypeName(0, pkg, "Stringer", nil)
				return types.NewNamed(obj, types.NewInterfaceType([]*types.Func{}, nil), nil)
			}(),
			expectedImports: []string{"fmt"},
			shouldError:     false,
		},
		{
			name: "alias type in different package",
			pkg:  "main",
			typeExpr: func() types.Type {
				pkg := types.NewPackage("context", "context")
				obj := types.NewTypeName(0, pkg, "Context", nil)
				return types.NewAlias(obj, types.NewInterfaceType([]*types.Func{}, nil))
			}(),
			expectedImports: []string{"context"},
			shouldError:     false,
		},
		{
			name: "slice type",
			pkg:  "main",
			typeExpr: types.NewSlice(types.Typ[types.String]),
			expectedImports: []string{},
			shouldError:     false,
		},
		{
			name: "array type",
			pkg:  "main",
			typeExpr: types.NewArray(types.Typ[types.Int], 10),
			expectedImports: []string{},
			shouldError:     false,
		},
		{
			name: "map type",
			pkg:  "main",
			typeExpr: types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
			expectedImports: []string{},
			shouldError:     false,
		},
		{
			name: "interface type with methods",
			pkg:  "main",
			typeExpr: func() types.Type {
				// Create a method signature
				params := types.NewTuple(types.NewVar(0, nil, "s", types.Typ[types.String]))
				results := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int]))
				sig := types.NewSignatureType(nil, nil, nil, params, results, false)
				method := types.NewFunc(0, nil, "Method", sig)
				return types.NewInterfaceType([]*types.Func{method}, nil)
			}(),
			expectedImports: []string{},
			shouldError:     false,
		},
		{
			name: "channel type",
			pkg:  "main",
			typeExpr: types.NewChan(types.SendRecv, types.Typ[types.String]),
			expectedImports: []string{},
			shouldError:     false,
		},
		{
			name: "function type",
			pkg:  "main",
			typeExpr: func() types.Type {
				params := types.NewTuple(types.NewVar(0, nil, "x", types.Typ[types.Int]))
				results := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.String]))
				return types.NewSignatureType(nil, nil, nil, params, results, false)
			}(),
			expectedImports: []string{},
			shouldError:     false,
		},
		{
			name: "struct type",
			pkg:  "main",
			typeExpr: func() types.Type {
				field := types.NewVar(0, nil, "Name", types.Typ[types.String])
				return types.NewStruct([]*types.Var{field}, []string{"json:\"name\""})
			}(),
			expectedImports: []string{},
			shouldError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			expr, imports, err := createASTTypeExpr(tt.pkg, tt.typeExpr)
			
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if tt.expectNil && expr != nil {
				t.Error("Expected nil expression but got non-nil")
				return
			}
			
			if !tt.expectNil && expr == nil {
				t.Error("Expected non-nil expression but got nil")
				return
			}
			
			if len(imports) != len(tt.expectedImports) {
				t.Errorf("Expected %d imports, got %d: %v", len(tt.expectedImports), len(imports), imports)
				return
			}
			
			for i, expectedImport := range tt.expectedImports {
				if imports[i] != expectedImport {
					t.Errorf("Expected import %q at index %d, got %q", expectedImport, i, imports[i])
				}
			}
		})
	}
}

func TestAutoAddMissingDependencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		metaData         *MetaData
		dependencyType   types.Type
		expectError      bool
		expectedImports  []string
	}{
		{
			name: "basic type dependency",
			metaData: &MetaData{
				Package: Package{
					Name: "main",
					Path: "main",
				},
				Imports: make(map[string]*ast.ImportSpec),
			},
			dependencyType: types.Typ[types.String],
			expectError:    false,
			expectedImports: []string{},
		},
		{
			name: "external package dependency",
			metaData: &MetaData{
				Package: Package{
					Name: "main",
					Path: "main",
				},
				Imports: make(map[string]*ast.ImportSpec),
			},
			dependencyType: func() types.Type {
				pkg := types.NewPackage("fmt", "fmt")
				obj := types.NewTypeName(0, pkg, "Stringer", nil)
				return types.NewNamed(obj, types.NewInterfaceType([]*types.Func{}, nil), nil)
			}(),
			expectError:    false,
			expectedImports: []string{"fmt"},
		},
		{
			name: "context package dependency",
			metaData: &MetaData{
				Package: Package{
					Name: "main",
					Path: "main",
				},
				Imports: make(map[string]*ast.ImportSpec),
			},
			dependencyType: func() types.Type {
				pkg := types.NewPackage("context", "context")
				obj := types.NewTypeName(0, pkg, "Context", nil)
				return types.NewNamed(obj, types.NewInterfaceType([]*types.Func{}, nil), nil)
			}(),
			expectError:    false,
			expectedImports: []string{"context"},
		},
		{
			name: "dependency with existing import",
			metaData: &MetaData{
				Package: Package{
					Name: "main",
					Path: "main",
				},
				Imports: map[string]*ast.ImportSpec{
					"fmt": {
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"fmt"`,
						},
					},
				},
			},
			dependencyType: func() types.Type {
				pkg := types.NewPackage("fmt", "fmt")
				obj := types.NewTypeName(0, pkg, "Stringer", nil)
				return types.NewNamed(obj, types.NewInterfaceType([]*types.Func{}, nil), nil)
			}(),
			expectError:    false,
			expectedImports: []string{"fmt"},
		},
		{
			name: "pointer type dependency",
			metaData: &MetaData{
				Package: Package{
					Name: "main",
					Path: "main",
				},
				Imports: make(map[string]*ast.ImportSpec),
			},
			dependencyType: types.NewPointer(types.Typ[types.Int]),
			expectError:    false,
			expectedImports: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			graph := &Graph{
				edges:        make(map[*node][]*edgeNode),
				reverseEdges: make(map[*node][]*node),
			}
			
			node, err := graph.autoAddMissingDependencies(tt.metaData, tt.dependencyType)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if node == nil {
				t.Error("Expected node to be non-nil")
				return
			}
			
			if node.arg == nil {
				t.Error("Expected node.arg to be non-nil")
				return
			}
			
			if node.arg.Type != tt.dependencyType {
				t.Error("Expected node.arg.Type to match dependency type")
			}
			
			if node.arg.ASTTypeExpr == nil {
				t.Error("Expected node.arg.ASTTypeExpr to be non-nil")
			}
			
			// Check that required imports were added
			for _, expectedImport := range tt.expectedImports {
				if _, exists := tt.metaData.Imports[expectedImport]; !exists {
					t.Errorf("Expected import %q to be added to metadata", expectedImport)
				}
			}
		})
	}
}

func TestGraph_BuildPoolStmts(t *testing.T) {
	t.Parallel()

	configType, _, intType := createTestTypes()

	tests := []struct {
		name                     string
		setupGraph              func() *Graph
		pool                    []*node
		pools                   [][]*node
		visited                 []bool
		poolDependencyMap       map[*node][]int
		nodeProvidedNodes       map[*node]map[*node]struct{}
		expectedStmtsMin        int
		expectError             bool
		expectProviderCallStmt  bool
		expectChainStmt         bool
	}{
		{
			name: "empty pool",
			setupGraph: func() *Graph {
				return &Graph{
					edges:        make(map[*node][]*edgeNode),
					reverseEdges: make(map[*node][]*node),
				}
			},
			pool:                []*node{},
			pools:               [][]*node{},
			visited:             []bool{},
			poolDependencyMap:   make(map[*node][]int),
			nodeProvidedNodes:   make(map[*node]map[*node]struct{}),
			expectedStmtsMin:    0,
			expectError:         false,
			expectProviderCallStmt: false,
			expectChainStmt:     false,
		},
		{
			name: "pool with argument node only",
			setupGraph: func() *Graph {
				return &Graph{
					edges:        make(map[*node][]*edgeNode),
					reverseEdges: make(map[*node][]*node),
				}
			},
			pool: []*node{
				{
					// argument node (providerSpec == nil)
					arg: &argument{
						Type:        intType,
						ASTTypeExpr: &ast.Ident{Name: "int"},
					},
					providerSpec: nil, // This is the key - argument nodes have nil providerSpec
				},
			},
			pools:               [][]*node{},
			visited:             []bool{},
			poolDependencyMap:   make(map[*node][]int),
			nodeProvidedNodes:   make(map[*node]map[*node]struct{}),
			expectedStmtsMin:    0, // argument nodes are skipped
			expectError:         false,
			expectProviderCallStmt: false,
			expectChainStmt:     false,
		},
		{
			name: "pool with provider node",
			setupGraph: func() *Graph {
				return &Graph{
					edges:        make(map[*node][]*edgeNode),
					reverseEdges: make(map[*node][]*node),
				}
			},
			pool: []*node{
				{
					providerSpec: &ProviderSpec{
						Type:          ProviderTypeFunction,
						Provides:      []types.Type{configType},
						Requires:      []types.Type{},
						IsReturnError: false,
					},
					providerArgs: []*InjectorCallArgument{},
					returnValues: []*InjectorParam{NewInjectorParam(configType)},
				},
			},
			pools:               [][]*node{},
			visited:             []bool{},
			poolDependencyMap:   make(map[*node][]int),
			nodeProvidedNodes:   make(map[*node]map[*node]struct{}),
			expectedStmtsMin:    1,
			expectError:         false,
			expectProviderCallStmt: true,
			expectChainStmt:     false,
		},
		{
			name: "mixed pool with argument and provider nodes",
			setupGraph: func() *Graph {
				return &Graph{
					edges:        make(map[*node][]*edgeNode),
					reverseEdges: make(map[*node][]*node),
				}
			},
			pool: []*node{
				{
					// argument node - should be skipped
					arg: &argument{
						Type:        intType,
						ASTTypeExpr: &ast.Ident{Name: "int"},
					},
					providerSpec: nil,
				},
				{
					// provider node - should generate statement
					providerSpec: &ProviderSpec{
						Type:          ProviderTypeFunction,
						Provides:      []types.Type{configType},
						Requires:      []types.Type{intType},
						IsReturnError: false,
					},
					providerArgs: []*InjectorCallArgument{
						{
							Param:  NewInjectorParam(intType),
							IsWait: false,
						},
					},
					returnValues: []*InjectorParam{NewInjectorParam(configType)},
				},
			},
			pools:               [][]*node{},
			visited:             []bool{},
			poolDependencyMap:   make(map[*node][]int),
			nodeProvidedNodes:   make(map[*node]map[*node]struct{}),
			expectedStmtsMin:    1, // Only the provider node generates a statement
			expectError:         false,
			expectProviderCallStmt: true,
			expectChainStmt:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			graph := tt.setupGraph()

			stmts, err := graph.buildPoolStmts(
				tt.pool,
				tt.pools,
				tt.visited,
				tt.poolDependencyMap,
				tt.nodeProvidedNodes,
			)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(stmts) < tt.expectedStmtsMin {
				t.Errorf("Expected at least %d statements, got %d", tt.expectedStmtsMin, len(stmts))
			}

			// Check statement types
			hasProviderCall := false
			hasChain := false

			for _, stmt := range stmts {
				switch stmt.(type) {
				case *InjectorProviderCallStmt:
					hasProviderCall = true
				case *InjectorChainStmt:
					hasChain = true
				}
			}

			if tt.expectProviderCallStmt && !hasProviderCall {
				t.Error("Expected at least one InjectorProviderCallStmt")
			}

			if tt.expectChainStmt && !hasChain {
				t.Error("Expected at least one InjectorChainStmt")
			}

			if !tt.expectProviderCallStmt && hasProviderCall {
				t.Error("Did not expect InjectorProviderCallStmt but found one")
			}

			if !tt.expectChainStmt && hasChain {
				t.Error("Did not expect InjectorChainStmt but found one")
			}
		})
	}
}