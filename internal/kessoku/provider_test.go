package kessoku

import (
	"go/types"
	"testing"
)

func TestInjectorParam(t *testing.T) {
	t.Parallel()

	// Create test types
	_, serviceType, intType := createTestTypes()

	tests := []struct {
		paramType    types.Type
		expectedName string
		name         string
		refCount     int
	}{
		{
			name:         "unreferenced parameter",
			paramType:    intType,
			refCount:     0,
			expectedName: "_",
		},
		{
			name:         "referenced parameter",
			paramType:    serviceType,
			refCount:     1,
			expectedName: "service", // Should get a name based on type
		},
		{
			name:         "multiple references",
			paramType:    intType,
			refCount:     3,
			expectedName: "num", // Should get a name based on type
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			param := NewInjectorParam(tt.paramType)

			// Add references
			for i := 0; i < tt.refCount; i++ {
				param.Ref(false) // Pass false for isWait parameter
			}

			varPool := NewVarPool()
			if got := param.Name(varPool); (tt.refCount == 0 && got != "_") || (tt.refCount > 0 && got == "_") {
				t.Errorf("Name() = %v, want non-underscore for refCount > 0", got)
			}
		})
	}
}

func TestInjectorParamChannelName(t *testing.T) {
	t.Parallel()

	_, serviceType, _ := createTestTypes()

	param := NewInjectorParam(serviceType)
	param.Ref(true) // Reference with channel

	varPool := NewVarPool()
	channelName := param.ChannelName(varPool)

	if channelName == "_" {
		t.Error("Expected channel name to be generated for referenced parameter")
	}

	if !param.WithChannel() {
		t.Error("Expected WithChannel() to return true")
	}
}

func TestInjectorChainStmt_HasAsync(t *testing.T) {
	t.Parallel()

	configType, serviceType, _ := createTestTypes()

	tests := []struct {
		name      string
		chainStmt *InjectorChainStmt
		expected  bool
	}{
		{
			name: "empty chain",
			chainStmt: &InjectorChainStmt{
				Statements: []InjectorStmt{},
			},
			expected: false,
		},
		{
			name: "chain with only sync providers",
			chainStmt: &InjectorChainStmt{
				Statements: []InjectorStmt{
					&InjectorProviderCallStmt{
						Provider: &ProviderSpec{
							Type:          ProviderTypeFunction,
							Provides:      []types.Type{configType},
							Requires:      []types.Type{},
							IsReturnError: false,
							IsAsync:       false,
						},
						Arguments: []*InjectorCallArgument{},
						Returns:   []*InjectorParam{NewInjectorParam(configType)},
					},
				},
			},
			expected: false,
		},
		{
			name: "chain with async provider",
			chainStmt: &InjectorChainStmt{
				Statements: []InjectorStmt{
					&InjectorProviderCallStmt{
						Provider: &ProviderSpec{
							Type:          ProviderTypeFunction,
							Provides:      []types.Type{configType},
							Requires:      []types.Type{},
							IsReturnError: false,
							IsAsync:       true,
						},
						Arguments: []*InjectorCallArgument{},
						Returns:   []*InjectorParam{NewInjectorParam(configType)},
					},
				},
			},
			expected: true,
		},
		{
			name: "chain with mixed sync and async providers",
			chainStmt: &InjectorChainStmt{
				Statements: []InjectorStmt{
					&InjectorProviderCallStmt{
						Provider: &ProviderSpec{
							Type:          ProviderTypeFunction,
							Provides:      []types.Type{configType},
							Requires:      []types.Type{},
							IsReturnError: false,
							IsAsync:       false, // sync
						},
						Arguments: []*InjectorCallArgument{},
						Returns:   []*InjectorParam{NewInjectorParam(configType)},
					},
					&InjectorProviderCallStmt{
						Provider: &ProviderSpec{
							Type:          ProviderTypeFunction,
							Provides:      []types.Type{serviceType},
							Requires:      []types.Type{configType},
							IsReturnError: false,
							IsAsync:       true, // async
						},
						Arguments: []*InjectorCallArgument{
							{
								Param:  NewInjectorParam(configType),
								IsWait: false,
							},
						},
						Returns: []*InjectorParam{NewInjectorParam(serviceType)},
					},
				},
			},
			expected: true,
		},
		{
			name: "nested chain statements",
			chainStmt: &InjectorChainStmt{
				Statements: []InjectorStmt{
					&InjectorChainStmt{
						Statements: []InjectorStmt{
							&InjectorProviderCallStmt{
								Provider: &ProviderSpec{
									Type:          ProviderTypeFunction,
									Provides:      []types.Type{configType},
									Requires:      []types.Type{},
									IsReturnError: false,
									IsAsync:       true, // nested async
								},
								Arguments: []*InjectorCallArgument{},
								Returns:   []*InjectorParam{NewInjectorParam(configType)},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "nested chain with only sync",
			chainStmt: &InjectorChainStmt{
				Statements: []InjectorStmt{
					&InjectorChainStmt{
						Statements: []InjectorStmt{
							&InjectorProviderCallStmt{
								Provider: &ProviderSpec{
									Type:          ProviderTypeFunction,
									Provides:      []types.Type{configType},
									Requires:      []types.Type{},
									IsReturnError: false,
									IsAsync:       false, // nested sync
								},
								Arguments: []*InjectorCallArgument{},
								Returns:   []*InjectorParam{NewInjectorParam(configType)},
							},
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.chainStmt.HasAsync()
			if result != tt.expected {
				t.Errorf("HasAsync() = %v, want %v", result, tt.expected)
			}
		})
	}
}