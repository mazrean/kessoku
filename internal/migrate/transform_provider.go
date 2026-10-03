package migrate

import (
	"fmt"
	"go/ast"
	"go/types"
)

// transformProviderFunc transforms a provider function to kessoku.Provide.
// It rejects providers that return a wire-style cleanup function: kessoku's
// code generator treats every non-error return value as an ordinary provided
// value and silently discards unused ones, so migrate is the single gatekeeper
// that must surface cleanup usage as a loud migration error.
func (t *Transformer) transformProviderFunc(wf *WireProviderFunc) (*KessokuProvide, error) {
	if cleanupType := providerCleanupReturn(wf.Func); cleanupType != nil {
		return nil, &ParseError{
			Kind:    ParseErrorTypeResolution,
			File:    wf.File,
			Pos:     wf.Pos,
			Message: fmt.Sprintf("provider %q returns a wire-style cleanup function (%s); kessoku does not support cleanup functions — release the resource explicitly (e.g. expose a Close method on the provided type) before migrating", wf.Name, cleanupType),
		}
	}
	return &KessokuProvide{
		FuncExpr:  wf.Expr,
		SourcePos: wf.Pos,
	}, nil
}

// providerCleanupReturn returns the wire-style cleanup return type of fn, or
// nil when fn has none (or when no type information is available). Wire's
// cleanup pattern is a func() (or func() error) returned alongside at least
// one provided value, e.g. func NewDB() (*DB, func(), error). A func() that is
// the provider's ONLY non-error return value is the provided value itself, not
// a cleanup.
func providerCleanupReturn(fn *types.Func) types.Type {
	if fn == nil {
		return nil
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return nil
	}
	nonErrorSeen := false
	for v := range sig.Results().Variables() {
		rt := v.Type()
		if isErrorType(rt) {
			continue
		}
		if nonErrorSeen && isCleanupFuncType(rt) {
			return rt
		}
		nonErrorSeen = true
	}
	return nil
}

// isCleanupFuncType reports whether t is a wire-style cleanup function type:
// an anonymous func() or func() error with no parameters. Named func types and
// aliases (e.g. type ShutdownFunc func()) are user-defined business types, not
// cleanups, and must pass through unchanged.
func isCleanupFuncType(t types.Type) bool {
	switch t.(type) {
	case *types.Named, *types.Alias:
		return false
	}
	sig, ok := t.(*types.Signature)
	if !ok || sig.Params().Len() != 0 {
		return false
	}
	switch sig.Results().Len() {
	case 0:
		// bare func()
		return true
	case 1:
		// func() error
		return isErrorType(sig.Results().At(0).Type())
	default:
		return false
	}
}

// transformSetRef transforms a set reference.
// A package-qualified reference (pkg.FooSet) is emitted as-is: kessoku can use
// a kessoku.Set declared in another package, but only once that package has
// itself been migrated from wire.NewSet to kessoku.Set, so a warning is
// recorded (once per referenced set per file) to remind the user.
func (t *Transformer) transformSetRef(ws *WireSetRef) *KessokuSetRef {
	if _, isCrossPkg := ws.Expr.(*ast.SelectorExpr); isCrossPkg {
		t.warnCrossPackageSetRef(ws)
	}
	return &KessokuSetRef{
		Name:      ws.Name,
		Expr:      ws.Expr,
		SourcePos: ws.Pos,
	}
}

// warnCrossPackageSetRef records a WarnCrossPackageSetRef warning for ws unless
// one was already recorded for the same set during the current Transform call.
func (t *Transformer) warnCrossPackageSetRef(ws *WireSetRef) {
	key := ws.Name
	if ws.PkgPath != "" {
		key = ws.PkgPath + "." + ws.Name
	}
	if t.warnedSetRefs == nil {
		t.warnedSetRefs = make(map[string]bool)
	}
	if t.warnedSetRefs[key] {
		return
	}
	t.warnedSetRefs[key] = true

	pkgDesc := "its package"
	if ws.PkgPath != "" {
		pkgDesc = fmt.Sprintf("package %q", ws.PkgPath)
	}
	loc := ""
	if ws.File != "" {
		loc = " in " + ws.File
	}
	t.warnings = append(t.warnings, Warning{
		Code:    WarnCrossPackageSetRef,
		Pos:     ws.Pos,
		Message: fmt.Sprintf("cross-package set reference %s%s is kept as-is: %s must also be migrated to kessoku (its wire.NewSet replaced by kessoku.Set) for the generated code to work", ws.Name, loc, pkgDesc),
	})
}
