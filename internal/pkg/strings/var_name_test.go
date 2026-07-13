package strings

import (
	"testing"
	"unicode/utf8"
)

func TestToLowerCamel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  string
		validUTF8 bool
	}{
		{
			name:      "ascii single uppercase",
			input:     "Service",
			expected:  "service",
			validUTF8: true,
		},
		{
			name:      "ascii multiple uppercase prefix",
			input:     "UserRepository",
			expected:  "userRepository",
			validUTF8: true,
		},
		{
			name:      "already lowercase",
			input:     "service",
			expected:  "service",
			validUTF8: true,
		},
		{
			name:      "all uppercase",
			input:     "HTTP",
			expected:  "http",
			validUTF8: true,
		},
		{
			name:      "non-ascii uppercase rune (U+00D1 Ñ)",
			input:     "Ñombre",
			expected:  "ñombre",
			validUTF8: true,
		},
		{
			name:      "non-ascii uppercase rune at start (U+00C9 É)",
			input:     "Étoile",
			expected:  "étoile",
			validUTF8: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ToLowerCamel(tt.input)
			if !utf8.ValidString(result) {
				t.Errorf("ToLowerCamel(%q) = %q, result is invalid UTF-8", tt.input, result)
			}
			if result != tt.expected {
				t.Errorf("ToLowerCamel(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
