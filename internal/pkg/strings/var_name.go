// Package strings provides string utility functions for variable naming.
package strings

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func ToLowerCamel(s string) string {
	i := 0
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !unicode.IsUpper(r) {
			break
		}
		i += size
	}

	return strings.ToLower(s[:i]) + s[i:]
}
