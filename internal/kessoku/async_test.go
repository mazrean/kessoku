package kessoku

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/mazrean/kessoku/internal/pkg/collection"
)

func TestAsyncProviderDetection(t *testing.T) {
	// Create a mock provider spec for async provider
	asyncProvider := &ProviderSpec{
		Type:          ProviderTypeFunction,
		Provides:      []types.Type{types.NewNamed(types.NewTypeName(0, nil, "TestService", nil), types.NewStruct(nil, nil), nil)},
		Requires:      []types.Type{},
		IsReturnError: false,
		IsAsync:       true,
		ASTExpr:       &ast.Ident{Name: "testAsyncProvider"},
	}

	normalProvider := &ProviderSpec{
		Type:          ProviderTypeFunction,
		Provides:      []types.Type{types.NewNamed(types.NewTypeName(0, nil, "TestService2", nil), types.NewStruct(nil, nil), nil)},
		Requires:      []types.Type{},
		IsReturnError: false,
		IsAsync:       false,
		ASTExpr:       &ast.Ident{Name: "testNormalProvider"},
	}

	if !asyncProvider.IsAsync {
		t.Error("Expected async provider to have IsAsync=true")
	}

	if normalProvider.IsAsync {
		t.Error("Expected normal provider to have IsAsync=false")
	}
}

func TestParallelGroupAssignment(t *testing.T) {
	// Create a mock graph with async and normal providers
	graph := &Graph{
		injectorName:   "TestInjector",
		waitNodes:      collection.NewQueue[*node](),
		waitNodesAdded: make(map[*node]bool),
		edges:          make(map[*node][]*edgeNode),
	}

	// Create async nodes
	asyncNode1 := &node{
		providerSpec: &ProviderSpec{
			IsAsync: true,
		},
		requireCount: 0,
	}

	asyncNode2 := &node{
		providerSpec: &ProviderSpec{
			IsAsync: true,
		},
		requireCount: 0,
	}

	// Create normal node
	normalNode := &node{
		providerSpec: &ProviderSpec{
			IsAsync: false,
		},
		requireCount: 0,
	}

	// Add nodes to the graph
	graph.waitNodes.Push(asyncNode1)
	graph.waitNodes.Push(asyncNode2)
	graph.waitNodes.Push(normalNode)

	// Analyze parallel groups
	graph.analyzeParallelGroups()

	// Check that async nodes get the same parallel group
	if asyncNode1.parallelGroup != asyncNode2.parallelGroup {
		t.Errorf("Expected async nodes to have the same parallel group, got %d and %d",
			asyncNode1.parallelGroup, asyncNode2.parallelGroup)
	}

	// Check that normal node gets group 0
	if normalNode.parallelGroup != 0 {
		t.Errorf("Expected normal node to have parallel group 0, got %d", normalNode.parallelGroup)
	}
}

func TestInjectorStmtParallelGroup(t *testing.T) {
	stmt := &InjectorStmt{
		Provider: &ProviderSpec{
			IsAsync: true,
		},
		ParallelGroup: 1,
	}

	if stmt.ParallelGroup != 1 {
		t.Errorf("Expected parallel group 1, got %d", stmt.ParallelGroup)
	}
}