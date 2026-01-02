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

	// Transform elements using common logic
	elements, err := t.transformElements(wb.Elements, pkg)
	if err != nil {
		return nil, err
	}

	return &KessokuInject{
		FuncName:   wb.FuncName,
		FuncDecl:   wb.FuncDecl,
		ReturnType: wb.ReturnTypes[0],
		HasError:   hasError,
		Elements:   elements,
		SourcePos:  wb.Pos,
	}, nil
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
