package migrate

import (
	"go/types"
	"testing"
)

func TestIsErrorType(t *testing.T) {
	tests := []struct {
		typeFunc func() types.Type
		name     string
		want     bool
	}{
		{
			name: "nil type",
			typeFunc: func() types.Type {
				return nil
			},
			want: false,
		},
		{
			name: "built-in error type",
			typeFunc: func() types.Type {
				return types.Universe.Lookup("error").Type()
			},
			want: true,
		},
		{
			name: "int type",
			typeFunc: func() types.Type {
				return types.Typ[types.Int]
			},
			want: false,
		},
		{
			name: "string type",
			typeFunc: func() types.Type {
				return types.Typ[types.String]
			},
			want: false,
		},
		{
			name: "bool type",
			typeFunc: func() types.Type {
				return types.Typ[types.Bool]
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ := tt.typeFunc()
			got := isErrorType(typ)
			if got != tt.want {
				t.Errorf("isErrorType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTypeToExpr(t *testing.T) {
	tests := []struct {
		typeFunc func() types.Type
		name     string
		wantNil  bool
	}{
		{
			name: "nil type",
			typeFunc: func() types.Type {
				return nil
			},
			wantNil: true,
		},
		{
			name: "basic int type",
			typeFunc: func() types.Type {
				return types.Typ[types.Int]
			},
			wantNil: false,
		},
		{
			name: "basic string type",
			typeFunc: func() types.Type {
				return types.Typ[types.String]
			},
			wantNil: false,
		},
		{
			name: "pointer to int",
			typeFunc: func() types.Type {
				return types.NewPointer(types.Typ[types.Int])
			},
			wantNil: false,
		},
		{
			name: "slice of int",
			typeFunc: func() types.Type {
				return types.NewSlice(types.Typ[types.Int])
			},
			wantNil: false,
		},
		{
			name: "map of string to int",
			typeFunc: func() types.Type {
				return types.NewMap(types.Typ[types.String], types.Typ[types.Int])
			},
			wantNil: false,
		},
		{
			name: "array of 10 int",
			typeFunc: func() types.Type {
				return types.NewArray(types.Typ[types.Int], 10)
			},
			wantNil: false,
		},
		{
			name: "channel of int",
			typeFunc: func() types.Type {
				return types.NewChan(types.SendRecv, types.Typ[types.Int])
			},
			wantNil: false,
		},
		{
			name: "empty interface (any)",
			typeFunc: func() types.Type {
				return types.NewInterfaceType(nil, nil)
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ := tt.typeFunc()
			got := typeToExpr(typ)
			if tt.wantNil && got != nil {
				t.Errorf("typeToExpr() = %v, want nil", got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("typeToExpr() = nil, want non-nil")
			}
		})
	}
}

func TestToLowerCamel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "single lowercase letter",
			input: "a",
			want:  "a",
		},
		{
			name:  "single uppercase letter",
			input: "A",
			want:  "a",
		},
		{
			name:  "PascalCase",
			input: "FooBar",
			want:  "fooBar",
		},
		{
			name:  "already camelCase",
			input: "fooBar",
			want:  "fooBar",
		},
		{
			name:  "all uppercase",
			input: "FOO",
			want:  "fOO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toLowerCamel(tt.input)
			if got != tt.want {
				t.Errorf("toLowerCamel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUnwrapPointer(t *testing.T) {
	tests := []struct {
		typ      types.Type
		wantType types.Type
		name     string
	}{
		{
			name:     "non-pointer type returns same type",
			typ:      types.Typ[types.Int],
			wantType: types.Typ[types.Int],
		},
		{
			name:     "pointer to basic type returns element type",
			typ:      types.NewPointer(types.Typ[types.Int]),
			wantType: types.Typ[types.Int],
		},
		{
			name:     "pointer to string returns string",
			typ:      types.NewPointer(types.Typ[types.String]),
			wantType: types.Typ[types.String],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unwrapPointer(tt.typ)
			if got != tt.wantType {
				t.Errorf("unwrapPointer() = %v, want %v", got, tt.wantType)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		slice []string
		want  bool
	}{
		{
			name:  "empty slice",
			slice: []string{},
			s:     "foo",
			want:  false,
		},
		{
			name:  "element exists",
			slice: []string{"foo", "bar", "baz"},
			s:     "bar",
			want:  true,
		},
		{
			name:  "element does not exist",
			slice: []string{"foo", "bar", "baz"},
			s:     "qux",
			want:  false,
		},
		{
			name:  "single element slice - match",
			slice: []string{"foo"},
			s:     "foo",
			want:  true,
		},
		{
			name:  "single element slice - no match",
			slice: []string{"foo"},
			s:     "bar",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.s)
			if got != tt.want {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.s, got, tt.want)
			}
		})
	}
}
