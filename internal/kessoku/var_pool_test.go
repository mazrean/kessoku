package kessoku

import (
	"go/types"
	"testing"
)

func TestVarPool_GetBaseName(t *testing.T) {
	t.Parallel()

	pool := NewVarPool()

	tests := []struct {
		name     string
		typeExpr types.Type
		expected string
	}{
		// Basic types
		{
			name:     "int type",
			typeExpr: types.Typ[types.Int],
			expected: "num",
		},
		{
			name:     "int8 type",
			typeExpr: types.Typ[types.Int8],
			expected: "num",
		},
		{
			name:     "int16 type",
			typeExpr: types.Typ[types.Int16],
			expected: "num",
		},
		{
			name:     "int32 type",
			typeExpr: types.Typ[types.Int32],
			expected: "num",
		},
		{
			name:     "int64 type",
			typeExpr: types.Typ[types.Int64],
			expected: "num",
		},
		{
			name:     "uint type",
			typeExpr: types.Typ[types.Uint],
			expected: "num",
		},
		{
			name:     "uint8 type",
			typeExpr: types.Typ[types.Uint8],
			expected: "num",
		},
		{
			name:     "uint16 type",
			typeExpr: types.Typ[types.Uint16],
			expected: "num",
		},
		{
			name:     "uint32 type",
			typeExpr: types.Typ[types.Uint32],
			expected: "num",
		},
		{
			name:     "uint64 type",
			typeExpr: types.Typ[types.Uint64],
			expected: "num",
		},
		{
			name:     "float32 type",
			typeExpr: types.Typ[types.Float32],
			expected: "num",
		},
		{
			name:     "float64 type",
			typeExpr: types.Typ[types.Float64],
			expected: "num",
		},
		{
			name:     "string type",
			typeExpr: types.Typ[types.String],
			expected: "str",
		},
		{
			name:     "bool type",
			typeExpr: types.Typ[types.Bool],
			expected: "flag",
		},
		{
			name:     "complex64 type",
			typeExpr: types.Typ[types.Complex64],
			expected: "complex",
		},
		{
			name:     "complex128 type",
			typeExpr: types.Typ[types.Complex128],
			expected: "complex",
		},
		{
			name:     "uintptr type",
			typeExpr: types.Typ[types.Uintptr],
			expected: "ptr",
		},
		{
			name:     "unsafe pointer type",
			typeExpr: types.Typ[types.UnsafePointer],
			expected: "ptr",
		},
		{
			name:     "untyped nil",
			typeExpr: types.Typ[types.UntypedNil],
			expected: "null",
		},
		{
			name:     "invalid type",
			typeExpr: types.Typ[types.Invalid],
			expected: "invalid",
		},
		{
			name:     "untyped int",
			typeExpr: types.Typ[types.UntypedInt],
			expected: "num",
		},
		{
			name:     "untyped float",
			typeExpr: types.Typ[types.UntypedFloat],
			expected: "num",
		},
		{
			name:     "untyped string",
			typeExpr: types.Typ[types.UntypedString],
			expected: "str",
		},
		{
			name:     "untyped bool",
			typeExpr: types.Typ[types.UntypedBool],
			expected: "flag",
		},
		{
			name:     "untyped complex",
			typeExpr: types.Typ[types.UntypedComplex],
			expected: "complex",
		},
		{
			name:     "untyped rune",
			typeExpr: types.Typ[types.UntypedRune],
			expected: "num",
		},
		// Named types
		{
			name: "named type Service",
			typeExpr: func() types.Type {
				obj := types.NewTypeName(0, nil, "Service", nil)
				return types.NewNamed(obj, types.NewStruct(nil, nil), nil)
			}(),
			expected: "service",
		},
		{
			name: "named type UserRepository",
			typeExpr: func() types.Type {
				obj := types.NewTypeName(0, nil, "UserRepository", nil)
				return types.NewNamed(obj, types.NewStruct(nil, nil), nil)
			}(),
			expected: "userRepository",
		},
		{
			name: "context.Context type",
			typeExpr: func() types.Type {
				pkg := types.NewPackage("context", "context")
				obj := types.NewTypeName(0, pkg, "Context", nil)
				return types.NewNamed(obj, types.NewInterfaceType([]*types.Func{}, nil), nil)
			}(),
			expected: "ctx",
		},
		// Pointer types
		{
			name:     "pointer to int",
			typeExpr: types.NewPointer(types.Typ[types.Int]),
			expected: "num",
		},
		{
			name:     "pointer to string",
			typeExpr: types.NewPointer(types.Typ[types.String]),
			expected: "str",
		},
		{
			name: "pointer to named type",
			typeExpr: func() types.Type {
				obj := types.NewTypeName(0, nil, "DatabaseConfig", nil)
				namedType := types.NewNamed(obj, types.NewStruct(nil, nil), nil)
				return types.NewPointer(namedType)
			}(),
			expected: "databaseConfig",
		},
		{
			name: "double pointer to named type",
			typeExpr: func() types.Type {
				obj := types.NewTypeName(0, nil, "Service", nil)
				namedType := types.NewNamed(obj, types.NewStruct(nil, nil), nil)
				singlePtr := types.NewPointer(namedType)
				return types.NewPointer(singlePtr)
			}(),
			expected: "service",
		},
		{
			name: "pointer to context.Context",
			typeExpr: func() types.Type {
				pkg := types.NewPackage("context", "context")
				obj := types.NewTypeName(0, pkg, "Context", nil)
				namedType := types.NewNamed(obj, types.NewInterfaceType([]*types.Func{}, nil), nil)
				return types.NewPointer(namedType)
			}(),
			expected: "ctx",
		},
		// Non-basic, non-named types (should fall through to "val")
		{
			name:     "slice type",
			typeExpr: types.NewSlice(types.Typ[types.String]),
			expected: "val",
		},
		{
			name:     "map type",
			typeExpr: types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
			expected: "val",
		},
		{
			name:     "chan type",
			typeExpr: types.NewChan(types.SendRecv, types.Typ[types.String]),
			expected: "val",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := pool.getBaseName(tt.typeExpr)
			if result != tt.expected {
				t.Errorf("getBaseName() = %v, want %v", result, tt.expected)
			}
		})
	}
}
