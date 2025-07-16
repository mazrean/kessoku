// Package strings provides string utility functions for variable naming.
package strings

import (
	"strings"
	"unicode"
)

func ToLowerCamel(s string) string {
	i := 0
	for i < len(s) && unicode.IsUpper(rune(s[i])) {
		i++
	}

	return strings.ToLower(s[:i]) + s[i:]
}
