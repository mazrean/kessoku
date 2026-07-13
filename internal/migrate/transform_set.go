package migrate

import (
	"maps"

	"go/types"
)

// buildSetIndex builds a map from set variable name to *WireNewSet for all top-level sets.
// This is used by transformElements to look up the contents of WireSetRef elements so that
// providers that are already covered by a wire.Bind can be deduplicated.
func buildSetIndex(patterns []WirePattern) map[string]*WireNewSet {
	idx := make(map[string]*WireNewSet)
	for _, p := range patterns {
		if ws, ok := p.(*WireNewSet); ok && ws.VarName != "" {
			idx[ws.VarName] = ws
		}
	}
	return idx
}

// allProvidersAreBound returns true if every WireProviderFunc reachable from the
// given WireNewSet is superseded by one of the bound implementation types in boundTypes.
// The setIndex is used to resolve nested WireSetRef elements.
func allProvidersAreBound(ws *WireNewSet, boundTypes map[string]bool, setIndex map[string]*WireNewSet) bool {
	return setProvidersAllBound(ws.Elements, boundTypes, setIndex)
}

func setProvidersAllBound(elements []WirePattern, boundTypes map[string]bool, setIndex map[string]*WireNewSet) bool {
	hasProvider := false
	for _, elem := range elements {
		switch we := elem.(type) {
		case *WireProviderFunc:
			if we.Func == nil {
				// Cannot determine — treat as not bound to be safe.
				return false
			}
			sig, ok := we.Func.Type().(*types.Signature)
			if !ok {
				return false
			}
			results := sig.Results()
			if results.Len() == 0 {
				return false
			}
			returnType := results.At(0).Type()
			if !boundTypes[returnType.String()] {
				return false
			}
			hasProvider = true
		case *WireNewSet:
			if !setProvidersAllBound(we.Elements, boundTypes, setIndex) {
				return false
			}
		case *WireSetRef:
			nested, ok := setIndex[we.Name]
			if !ok {
				// Unknown set reference — treat as not bound to be safe.
				return false
			}
			if !allProvidersAreBound(nested, boundTypes, setIndex) {
				return false
			}
			hasProvider = true
		}
	}
	return hasProvider
}

// transformNewSet transforms wire.NewSet to kessoku.Set.
func (t *Transformer) transformNewSet(ws *WireNewSet, pkg *types.Package, setIndex map[string]*WireNewSet) (*KessokuSet, error) {
	elements, err := t.transformElements(ws.Elements, pkg, setIndex)
	if err != nil {
		return nil, err
	}

	return &KessokuSet{
		VarName:   ws.VarName,
		Elements:  elements,
		SourcePos: ws.Pos,
	}, nil
}

// mergeFieldsOf groups WireFieldsOf patterns by struct type and merges their fields.
func (t *Transformer) mergeFieldsOf(elements []WirePattern) map[string]*WireFieldsOf {
	result := make(map[string]*WireFieldsOf)

	for _, elem := range elements {
		wf, ok := elem.(*WireFieldsOf)
		if !ok {
			continue
		}

		typeKey := wf.StructType.String()
		if existing, exists := result[typeKey]; exists {
			// Merge fields (avoid duplicates)
			for _, field := range wf.Fields {
				if !contains(existing.Fields, field) {
					existing.Fields = append(existing.Fields, field)
				}
			}
		} else {
			// Create a copy to avoid modifying the original
			result[typeKey] = &WireFieldsOf{
				baseWirePattern: wf.baseWirePattern,
				StructType:      wf.StructType,
				Fields:          append([]string{}, wf.Fields...),
			}
		}
	}

	return result
}

// collectBoundTypes collects all implementation types from WireBind elements.
// transformElements transforms a list of wire patterns to kessoku patterns.
// This is the common logic shared between transformNewSet and transformBuild.
// setIndex maps WireNewSet variable names to their parsed WireNewSet, used to
// determine what providers a WireSetRef contributes (for duplicate-provider detection).
func (t *Transformer) transformElements(elements []WirePattern, pkg *types.Package, setIndex map[string]*WireNewSet) ([]KessokuPattern, error) {
	// First pass: collect all bound implementation types
	// These are the types for which wire.Bind creates an implicit provider
	boundTypes := t.collectBoundTypes(elements)

	// Merge FieldsOf patterns with the same struct type
	mergedFieldsOf := t.mergeFieldsOf(elements)

	var result []KessokuPattern

	for _, elem := range elements {
		switch we := elem.(type) {
		case *WireNewSet:
			// Handle inline nested wire.NewSet
			nestedSet, err := t.transformNewSet(we, pkg, setIndex)
			if err != nil {
				return nil, err
			}
			// Flatten nested set elements into parent
			result = append(result, nestedSet.Elements...)
		case *WireBind:
			transformed, err := t.transformBind(we, pkg, elements)
			if err != nil {
				return nil, err
			}
			result = append(result, transformed)
		case *WireValue:
			result = append(result, t.transformValue(we))
		case *WireInterfaceValue:
			result = append(result, t.transformInterfaceValue(we))
		case *WireStruct:
			result = append(result, t.transformStruct(we, pkg))
		case *WireFieldsOf:
			// Check if this is the first occurrence of this struct type
			typeKey := we.StructType.String()
			merged, exists := mergedFieldsOf[typeKey]
			if !exists {
				continue // Already processed or no merge needed
			}
			// Remove from map so subsequent occurrences are skipped
			delete(mergedFieldsOf, typeKey)
			result = append(result, t.transformFieldsOf(merged, pkg))
		case *WireProviderFunc:
			// Skip provider if its output type is already bound via wire.Bind
			// (wire.Bind creates an implicit provider for the implementation type)
			if t.isProviderBound(we, boundTypes) {
				continue
			}
			result = append(result, t.transformProviderFunc(we))
		case *WireSetRef:
			// Suppress a set reference whose every provider is already covered by a
			// wire.Bind in the same scope. Without suppression, the Bind's embedded
			// kessoku.Provide(Ctor) would duplicate the provider already emitted by
			// the set, causing a "multiple providers provide *T" error (BUG-10).
			// allProvidersAreBound recurses into nested WireSetRef entries via setIndex
			// to handle Case 2 (Bind inside a sibling set ref).
			if t.setIndex != nil {
				if ws, ok := t.setIndex[we.Name]; ok {
					if allProvidersAreBound(ws, boundTypes, t.setIndex) {
						continue
					}
				}
			}
			transformed, err := t.transformSetRef(we)
			if err != nil {
				return nil, err
			}
			result = append(result, transformed)
		}
	}

	return result, nil
}

func (t *Transformer) collectBoundTypes(elements []WirePattern) map[string]bool {
	boundTypes := make(map[string]bool)
	for _, elem := range elements {
		switch we := elem.(type) {
		case *WireBind:
			// wire.Bind(new(Interface), new(*Impl)) -> Implementation is **Impl
			// We need to unwrap one level to get *Impl which matches the provider return type
			implType := we.Implementation
			if ptr, ok := implType.(*types.Pointer); ok {
				implType = ptr.Elem()
			}
			boundTypes[implType.String()] = true
		case *WireNewSet:
			// Recursively collect from nested sets
			maps.Copy(boundTypes, t.collectBoundTypes(we.Elements))
		case *WireSetRef:
			// Look up the set in the index and collect bound types from it.
			// This ensures that a Bind inside a referenced set suppresses the matching
			// provider from other set references in the same scope (BUG-10 Case 2).
			if t.setIndex != nil {
				if ws, ok := t.setIndex[we.Name]; ok {
					maps.Copy(boundTypes, t.collectBoundTypes(ws.Elements))
				}
			}
			// If this ref points to a top-level WireBind variable, the bind already
			// wraps the constructor, so mark the implementation type as bound to
			// prevent the constructor from being emitted separately in the same set (BUG-14).
			if implTypeStr, ok := t.bindVarTypes[we.Name]; ok {
				boundTypes[implTypeStr] = true
			}
		}
	}
	return boundTypes
}

// isProviderBound checks if a provider function's output type is bound via wire.Bind.
func (t *Transformer) isProviderBound(wf *WireProviderFunc, boundTypes map[string]bool) bool {
	if wf.Func == nil {
		return false
	}

	sig, ok := wf.Func.Type().(*types.Signature)
	if !ok {
		return false
	}

	results := sig.Results()
	if results.Len() == 0 {
		return false
	}

	// Check if the first return type is in the bound types
	returnType := results.At(0).Type()
	return boundTypes[returnType.String()]
}

// isSetRefSupersededByBind returns true when every provider contributed by the
// referenced set is already bound (i.e. will be embedded inside a kessoku.Bind
// by transformBind).  In that situation, including the set reference would
// produce a duplicate provider that kessoku's code generator would reject.
//
// We can only make this determination when the referenced WireNewSet is available
// in setIndex; if it is not (e.g. it lives in another file/package), we conservatively
// return false so the set ref is kept.
func (t *Transformer) isSetRefSupersededByBind(ref *WireSetRef, boundTypes map[string]bool, setIndex map[string]*WireNewSet) bool {
	if setIndex == nil {
		return false
	}
	ws, ok := setIndex[ref.Name]
	if !ok {
		return false
	}
	if len(ws.Elements) == 0 {
		return false
	}
	// Every element of the referenced set must be a provider whose return type is bound.
	for _, elem := range ws.Elements {
		wpf, ok := elem.(*WireProviderFunc)
		if !ok {
			// Non-provider element (e.g. nested set, Bind, Value) — conservative: keep ref.
			return false
		}
		if !t.isProviderBound(wpf, boundTypes) {
			return false
		}
	}
	return true
}
