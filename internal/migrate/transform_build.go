package migrate

import (
	"fmt"
	"go/types"
)

// transformBuild transforms wire.Build to kessoku.Inject.
func (t *Transformer) transformBuild(wb *WireBuild, pkg *types.Package) (*KessokuInject, error) {
	// Validate return signature - injector must have exactly 1 or 2 return values
	// (the injected type, optionally followed by error)
	numReturns := len(wb.ReturnTypes)
	if numReturns == 0 {
		return nil, &ParseError{
			Kind:    ParseErrorMissingConstructor,
			File:    wb.File,
			Pos:     wb.Pos,
			Message: fmt.Sprintf("injector function %q must have at least one return value", wb.FuncName),
		}
	}
	if numReturns > maxInjectorReturns {
		return nil, &ParseError{
			Kind:    ParseErrorMissingConstructor,
			File:    wb.File,
			Pos:     wb.Pos,
			Message: fmt.Sprintf("injector function %q has %d return values, expected 1 or %d", wb.FuncName, numReturns, maxInjectorReturns),
		}
	}

	// For 2-return functions, the second must be error
	hasError := false
	if numReturns == maxInjectorReturns {
		if !isErrorType(wb.ReturnTypes[1]) {
			return nil, &ParseError{
				Kind:    ParseErrorMissingConstructor,
				File:    wb.File,
				Pos:     wb.Pos,
				Message: fmt.Sprintf("injector function %q has 2 return values but second is not error (got %s)", wb.FuncName, wb.ReturnTypes[1]),
			}
		}
		hasError = true
	}

	// Transform elements using common logic.
	// wire.Build lives inside a function; build a local set index from the
	// build elements themselves (they cannot reference top-level sets by name
	// the same way, but we still pass a non-nil map for consistency).
	elements, err := t.transformElements(wb.Elements, pkg, nil)
	if err != nil {
		return nil, err
	}

	return &KessokuInject{
		FuncName:           wb.FuncName,
		FuncDecl:           wb.FuncDecl,
		ReturnType:         wb.ReturnTypes[0],
		NeedsErrorSentinel: hasError && !anyProviderReturnsError(wb.Elements, t.setIndex, make(map[string]bool)),
		Elements:           elements,
		SourcePos:          wb.Pos,
	}, nil
}

// anyProviderReturnsError reports whether any provider function reachable from
// elements (including through nested sets and same-package set references)
// returns an error. When it does, the generated injector already carries the
// error return and no kessoku.Value((error)(nil)) sentinel is needed.
// Unresolvable references are treated as not returning error, which at worst
// emits a redundant (but harmless) sentinel.
func anyProviderReturnsError(elements []WirePattern, setIndex map[string]*WireNewSet, visited map[string]bool) bool {
	for _, element := range elements {
		switch we := element.(type) {
		case *WireProviderFunc:
			if we.Func == nil {
				continue
			}
			sig, ok := we.Func.Type().(*types.Signature)
			if !ok {
				continue
			}
			results := sig.Results()
			if results.Len() > 0 && isErrorType(results.At(results.Len()-1).Type()) {
				return true
			}
		case *WireNewSet:
			if anyProviderReturnsError(we.Elements, setIndex, visited) {
				return true
			}
		case *WireSetRef:
			if visited[we.Name] {
				continue
			}
			visited[we.Name] = true
			if nested, ok := setIndex[we.Name]; ok && anyProviderReturnsError(nested.Elements, setIndex, visited) {
				return true
			}
		}
	}
	return false
}

// isErrorType checks if a type is the built-in error type.
func isErrorType(t types.Type) bool {
	if t == nil {
		return false
	}

	// Get the predeclared error type from Universe
	errorType := types.Universe.Lookup("error").Type()

	// Compare directly with the predeclared error type
	if types.Identical(t, errorType) {
		return true
	}

	// Handle type aliases by checking the underlying type
	underlying := t.Underlying()
	if underlying != nil && types.Identical(underlying, errorType.Underlying()) {
		return true
	}

	return false
}
