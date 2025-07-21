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
			param := NewInjectorParam([]types.Type{tt.paramType}, false)

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

	_, serviceType, intType := createTestTypes()

	tests := []struct {
		name               string
		setupParam         func() *InjectorParam
		expectedResult     string
		shouldBeUnderscore bool
	}{
		{
			name: "unreferenced parameter",
			setupParam: func() *InjectorParam {
				return NewInjectorParam([]types.Type{intType}, false) // No Ref() call
			},
			expectedResult:     "_",
			shouldBeUnderscore: true,
		},
		{
			name: "referenced parameter with channel",
			setupParam: func() *InjectorParam {
				p := NewInjectorParam([]types.Type{serviceType}, false)
				p.Ref(true) // Reference with channel
				return p
			},
			expectedResult:     "serviceCh",
			shouldBeUnderscore: false,
		},
		{
			name: "referenced parameter without channel",
			setupParam: func() *InjectorParam {
				p := NewInjectorParam([]types.Type{serviceType}, false)
				p.Ref(false) // Reference without channel
				return p
			},
			expectedResult:     "serviceCh", // Still gets a channel name, but WithChannel() is false
			shouldBeUnderscore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			param := tt.setupParam()
			varPool := NewVarPool()
			channelName := param.ChannelName(varPool)

			if tt.shouldBeUnderscore {
				if channelName != "_" {
					t.Errorf("Expected channel name to be '_' for unreferenced parameter, got %s", channelName)
				}
			} else {
				if channelName == "_" {
					t.Error("Expected channel name to be generated for referenced parameter")
				}
				if channelName != tt.expectedResult {
					t.Errorf("Expected channel name %s, got %s", tt.expectedResult, channelName)
				}
			}
		})
	}

	// Test caching behavior
	t.Run("caching behavior", func(t *testing.T) {
		t.Parallel()

		param := NewInjectorParam([]types.Type{serviceType}, false)
		param.Ref(true) // Reference with channel
		varPool := NewVarPool()

		// First call
		firstCall := param.ChannelName(varPool)
		// Second call should return the same cached result
		secondCall := param.ChannelName(varPool)

		if firstCall != secondCall {
			t.Errorf("Expected cached channel name to be consistent, got %s then %s", firstCall, secondCall)
		}

		if firstCall != "serviceCh" {
			t.Errorf("Expected channel name serviceCh, got %s", firstCall)
		}
	})
}

func TestInjectorChainStmt_HasAsync(t *testing.T) {
	t.Parallel()

	configType, serviceType, _ := createTestTypes()

	tests := []struct {
		chainStmt *InjectorChainStmt
		name      string
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
							Provides:      [][]types.Type{{configType}},
							Requires:      []types.Type{},
							IsReturnError: false,
							IsAsync:       false,
						},
						Arguments: []*InjectorCallArgument{},
						Returns:   []*InjectorParam{NewInjectorParam([]types.Type{configType}, false)},
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
							Provides:      [][]types.Type{{configType}},
							Requires:      []types.Type{},
							IsReturnError: false,
							IsAsync:       true,
						},
						Arguments: []*InjectorCallArgument{},
						Returns:   []*InjectorParam{NewInjectorParam([]types.Type{configType}, false)},
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
							Provides:      [][]types.Type{{configType}},
							Requires:      []types.Type{},
							IsReturnError: false,
							IsAsync:       false, // sync
						},
						Arguments: []*InjectorCallArgument{},
						Returns:   []*InjectorParam{NewInjectorParam([]types.Type{configType}, false)},
					},
					&InjectorProviderCallStmt{
						Provider: &ProviderSpec{
							Type:          ProviderTypeFunction,
							Provides:      [][]types.Type{{serviceType}},
							Requires:      []types.Type{configType},
							IsReturnError: false,
							IsAsync:       true, // async
						},
						Arguments: []*InjectorCallArgument{
							{
								Param:  NewInjectorParam([]types.Type{configType}, false),
								IsWait: false,
							},
						},
						Returns: []*InjectorParam{NewInjectorParam([]types.Type{serviceType}, false)},
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
									Provides:      [][]types.Type{{configType}},
									Requires:      []types.Type{},
									IsReturnError: false,
									IsAsync:       true, // nested async
								},
								Arguments: []*InjectorCallArgument{},
								Returns:   []*InjectorParam{NewInjectorParam([]types.Type{configType}, false)},
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
									Provides:      [][]types.Type{{configType}},
									Requires:      []types.Type{},
									IsReturnError: false,
									IsAsync:       false, // nested sync
								},
								Arguments: []*InjectorCallArgument{},
								Returns:   []*InjectorParam{NewInjectorParam([]types.Type{configType}, false)},
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

func TestInjectorParam_Type(t *testing.T) {
	t.Parallel()

	configType, serviceType, intType := createTestTypes()

	tests := []struct {
		typeExpr types.Type
		name     string
	}{
		{
			name:     "config type",
			typeExpr: configType,
		},
		{
			name:     "service type",
			typeExpr: serviceType,
		},
		{
			name:     "int type",
			typeExpr: intType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			param := NewInjectorParam([]types.Type{tt.typeExpr}, false)
			result := param.Type()
			if result != tt.typeExpr {
				t.Errorf("Type() = %v, want %v", result, tt.typeExpr)
			}
		})
	}
}
