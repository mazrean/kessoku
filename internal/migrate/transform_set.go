package migrate

import "maps"

import "go/types"

// transformNewSet transforms wire.NewSet to kessoku.Set.
func (t *Transformer) transformNewSet(ws *WireNewSet, pkg *types.Package) (*KessokuSet, error) {
	elements, err := t.transformElements(ws.Elements, pkg)
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
func (t *Transformer) transformElements(elements []WirePattern, pkg *types.Package) ([]KessokuPattern, error) {
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
			nestedSet, err := t.transformNewSet(we, pkg)
			if err != nil {
				return nil, err
			}
			// Flatten nested set elements into parent
			result = append(result, nestedSet.Elements...)
		case *WireBind:
			transformed, err := t.transformBind(we, pkg)
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
			result = append(result, t.transformSetRef(we))
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
