// Package migrate provides migration from google/wire to kessoku format.
package migrate

import (
	"go/ast"
	"go/token"
	"go/types"
)

// WirePatternKind identifies the type of wire pattern.
type WirePatternKind int

const (
	PatternNewSet WirePatternKind = iota
	PatternBind
	PatternValue
	PatternInterfaceValue
	PatternStruct
	PatternFieldsOf
	PatternProviderFunc
	PatternUnsupported
)

// KessokuPatternKind identifies the type of kessoku pattern.
type KessokuPatternKind int

const (
	KessokuPatternSet KessokuPatternKind = iota
	KessokuPatternProvide
	KessokuPatternBind
	KessokuPatternValue
)

// WarningCode identifies warning types.
type WarningCode int

const (
	WarnNoWireImport WarningCode = iota
	WarnNoWirePatterns
	WarnUnsupportedPattern
)

// ParseErrorKind identifies parse error types.
type ParseErrorKind int

const (
	ParseErrorSyntax ParseErrorKind = iota
	ParseErrorTypeResolution
	ParseErrorMissingConstructor
)

// MergeErrorKind identifies merge error types.
type MergeErrorKind int

const (
	MergeErrorPackageMismatch MergeErrorKind = iota
	MergeErrorNameCollision
)

// WirePattern is the interface for all wire pattern representations.
type WirePattern interface {
	wirePattern()
	Position() token.Pos
}

// baseWirePattern provides common fields for wire patterns.
type baseWirePattern struct {
	File string
	Pos  token.Pos
}

func (b *baseWirePattern) Position() token.Pos { return b.Pos }

// WireNewSet represents wire.NewSet(...) pattern.
type WireNewSet struct {
	baseWirePattern
	VarName  string
	Elements []WirePattern
}

func (*WireNewSet) wirePattern() {}

// WireBind represents wire.Bind(new(Interface), new(Impl)) pattern.
type WireBind struct {
	Interface      types.Type
	Implementation types.Type
	baseWirePattern
}

func (*WireBind) wirePattern() {}

// WireValue represents wire.Value(expr) pattern.
type WireValue struct {
	Expr ast.Expr
	Type types.Type
	baseWirePattern
}

func (*WireValue) wirePattern() {}

// WireInterfaceValue represents wire.InterfaceValue(new(Interface), expr) pattern.
type WireInterfaceValue struct {
	Interface types.Type
	Expr      ast.Expr
	baseWirePattern
}

func (*WireInterfaceValue) wirePattern() {}

// WireStruct represents wire.Struct(new(Type), fields...) pattern.
type WireStruct struct {
	baseWirePattern
	StructType types.Type
	Fields     []string
	IsPointer  bool
}

func (*WireStruct) wirePattern() {}

// WireFieldsOf represents wire.FieldsOf(new(Type), fields...) pattern.
type WireFieldsOf struct {
	baseWirePattern
	StructType types.Type
	Fields     []string
}

func (*WireFieldsOf) wirePattern() {}

// WireProviderFunc represents a provider function reference within a set.
type WireProviderFunc struct {
	Expr ast.Expr
	Func *types.Func
	Name string
	baseWirePattern
}

func (*WireProviderFunc) wirePattern() {}

// WireSetRef represents a reference to another provider set variable.
type WireSetRef struct {
	Expr ast.Expr
	Name string
	baseWirePattern
}

func (*WireSetRef) wirePattern() {}

// WireBuild represents wire.Build(...) pattern in an injector function.
type WireBuild struct {
	baseWirePattern
	FuncName    string        // Name of the enclosing injector function
	FuncDecl    *ast.FuncDecl // The enclosing function declaration
	Elements    []WirePattern // Providers/sets passed to wire.Build
	ReturnTypes []types.Type  // Return types of the injector function
}

func (*WireBuild) wirePattern() {}

// KessokuPattern is the interface for all kessoku pattern representations.
type KessokuPattern interface {
	kessokuPattern()
}

// KessokuSet represents kessoku.Set(...) pattern.
type KessokuSet struct {
	VarName   string
	Elements  []KessokuPattern
	SourcePos token.Pos
}

func (*KessokuSet) kessokuPattern() {}

// KessokuProvide represents kessoku.Provide(fn) pattern.
type KessokuProvide struct {
	FuncExpr  ast.Expr
	SourcePos token.Pos
}

func (*KessokuProvide) kessokuPattern() {}

// KessokuBind represents kessoku.Bind[I](provider) pattern.
type KessokuBind struct {
	Interface types.Type
	Provider  KessokuPattern
	SourcePos token.Pos
}

func (*KessokuBind) kessokuPattern() {}

// KessokuValue represents kessoku.Value(expr) pattern.
type KessokuValue struct {
	Expr      ast.Expr
	SourcePos token.Pos
}

func (*KessokuValue) kessokuPattern() {}

// KessokuSetRef represents a reference to another provider set variable.
type KessokuSetRef struct {
	Expr      ast.Expr
	Name      string
	SourcePos token.Pos
}

func (*KessokuSetRef) kessokuPattern() {}

// KessokuInject represents kessoku.Inject[T](...) pattern for injector functions.
type KessokuInject struct {
	ReturnType types.Type
	FuncDecl   *ast.FuncDecl
	FuncName   string
	Elements   []KessokuPattern
	SourcePos  token.Pos
	HasError   bool
}

func (*KessokuInject) kessokuPattern() {}

// Warning represents a non-fatal issue during migration.
type Warning struct {
	Message string
	Pos     token.Pos
	Code    WarningCode
}

// ParseError represents an error during single-file parsing/analysis.
type ParseError struct {
	File    string
	Message string
	Kind    ParseErrorKind
	Pos     token.Pos
}

func (e *ParseError) Error() string {
	return e.Message
}

// MergeError represents an error during multi-file merging.
type MergeError struct {
	Message    string
	Identifier string
	Files      []string
	Packages   []string
	Kind       MergeErrorKind
}

func (e *MergeError) Error() string {
	return e.Message
}

// ImportSpec represents an import declaration.
type ImportSpec struct {
	Path string
	Name string
}

// MigrationResult represents the result of migrating a single file.
type MigrationResult struct {
	SourceFile    string
	Package       string
	TypesPackage  *types.Package
	SourceImports map[string]string // package name (or alias) -> import path
	Imports       []ImportSpec
	Patterns      []KessokuPattern
	Warnings      []Warning
}

// MergedOutput represents the result of merging multiple file migrations.
type MergedOutput struct {
	Package       string
	Imports       []ImportSpec
	TopLevelDecls []ast.Decl
}
