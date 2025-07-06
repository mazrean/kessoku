package kessoku

import (
	"testing"
)

func TestInjectorParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		paramName    string
		expectedName string
		name         string
		refCount     int
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
