package kessoku

import (
	"testing"
)

func TestInjectorParam(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name         string
		paramName    string
		refCount     int
		expectedName string
	}{
		{
			name:         "unreferenced parameter",
			paramName:    "config",
			refCount:     0,
			expectedName: "_",
		},
		{
			name:         "referenced parameter",
			paramName:    "config",
			refCount:     1,
			expectedName: "config",
		},
		{
			name:         "multiple references",
			paramName:    "service",
			refCount:     3,
			expectedName: "service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			param := NewInjectorParam(tt.paramName)
			
			// Verify ID is unique
			if param.ID == 0 && injectorParamIDCounter > 1 {
				t.Error("Expected unique ID to be assigned")
			}
			
			// Add references
			for i := 0; i < tt.refCount; i++ {
				param.Ref()
			}
			
			if got := param.Name(); got != tt.expectedName {
				t.Errorf("Name() = %v, want %v", got, tt.expectedName)
			}
		})
	}
}

func TestInjectorParamIDCounter(t *testing.T) {
	// Note: Cannot be parallel because it tests global counter state
	
	tests := []struct {
		name               string
		numParams          int
		expectUnique       bool
		expectSequential   bool
	}{
		{
			name:             "create two parameters",
			numParams:        2,
			expectUnique:     true,
			expectSequential: true,
		},
		{
			name:             "create multiple parameters",
			numParams:        5,
			expectUnique:     true,
			expectSequential: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cannot be parallel because it modifies global counter
			initialCounter := injectorParamIDCounter
			var params []*InjectorParam
			
			for i := 0; i < tt.numParams; i++ {
				param := NewInjectorParam("test")
				params = append(params, param)
			}
			
			if tt.expectUnique {
				// Check all IDs are unique
				seen := make(map[uint64]bool)
				for _, param := range params {
					if seen[param.ID] {
						t.Errorf("Duplicate ID found: %d", param.ID)
					}
					seen[param.ID] = true
				}
			}
			
			if tt.expectSequential && len(params) > 1 {
				// Check IDs are sequential
				for i := 1; i < len(params); i++ {
					if params[i].ID != params[i-1].ID+1 {
						t.Errorf("Expected sequential ID assignment, got %d after %d", params[i].ID, params[i-1].ID)
					}
				}
			}
			
			expectedCounterIncrease := uint64(tt.numParams)
			if injectorParamIDCounter != initialCounter+expectedCounterIncrease {
				t.Errorf("Expected counter to increment by %d, got %d", expectedCounterIncrease, injectorParamIDCounter-initialCounter)
			}
		})
	}
}

func TestProviderType(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name         string
		providerType ProviderType
		expected     string
	}{
		{
			name:         "function provider",
			providerType: ProviderTypeFunction,
			expected:     "function",
		},
		{
			name:         "arg provider",
			providerType: ProviderTypeArg,
			expected:     "arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if string(tt.providerType) != tt.expected {
				t.Errorf("ProviderType = %v, want %v", tt.providerType, tt.expected)
			}
		})
	}
}