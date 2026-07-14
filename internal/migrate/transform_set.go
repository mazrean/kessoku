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
			// The inline nested set contributed providers (setProvidersAllBound returns
			// true only when it saw at least one, all bound), so this scope has providers
			// too. Without this, a set containing only nested sets is never suppressed.
			hasProvider = true
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
	elements, err := t.transformElementsWithBoundTypes(ws.Elements, pkg, setIndex, nil)
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
				IsPtrToStruct:   wf.IsPtrToStruct,
			}
		}
	}

	return result
}

// transformElements transforms a list of wire patterns to kessoku patterns.
// This is the common logic shared between transformNewSet and transformBuild.
// setIndex maps WireNewSet variable names to their parsed WireNewSet, used to
// determine what providers a WireSetRef contributes (for duplicate-provider detection).
func (t *Transformer) transformElements(elements []WirePattern, pkg *types.Package, setIndex map[string]*WireNewSet) ([]KessokuPattern, error) {
	return t.transformElementsWithBoundTypes(elements, pkg, setIndex, nil)
}

// collectBindsFromElements collects all WireBind elements reachable from the given
// elements, recursively resolving WireNewSet and WireSetRef (via t.setIndex).
// The returned slice preserves order but does not deduplicate.
func (t *Transformer) collectBindsFromElements(elements []WirePattern) []*WireBind {
	var binds []*WireBind
	for _, elem := range elements {
		switch we := elem.(type) {
		case *WireBind:
			binds = append(binds, we)
		case *WireNewSet:
			binds = append(binds, t.collectBindsFromElements(we.Elements)...)
		case *WireSetRef:
			if t.setIndex != nil {
				if ws, ok := t.setIndex[we.Name]; ok {
					binds = append(binds, t.collectBindsFromElements(ws.Elements)...)
				}
			}
		}
	}
	return binds
}

// expandConflictingSetRefs detects WireSetRef siblings that share a concrete
// implementation type in their resolved sets' binds, and expands those conflicting
// refs inline so the normal chainedBinds logic can merge them into a single
// kessoku.Bind chain.
//
// Background: when two named sets each bind a different interface to the same
// concrete type (e.g. SetA binds IfaceA→*Impl, SetB binds IfaceB→*Impl) and a
// parent set/build includes both via set references, the migration tool would
// otherwise emit kessoku.Provide(NewImpl) inside both SetA and SetB independently.
// kessoku's code generator then sees two distinct ProviderSpec objects providing
// *Impl and fails with "multiple providers provide *Impl" (BUG-15).
//
// Fix: if any concrete implementation type appears in the binds of more than one
// sibling WireSetRef, expand all such refs into their flat element lists (exactly
// as an inline wire.NewSet would be treated), deduplicating provider functions by
// identity. The existing chainedBinds pre-pass then produces a single chained
// kessoku.Bind for that impl type without any duplicate kessoku.Provide.
func (t *Transformer) expandConflictingSetRefs(elements []WirePattern) []WirePattern {
	if t.setIndex == nil {
		return elements
	}

	type refImplKeys struct {
		ref      *WireSetRef
		implKeys map[string]bool
	}

	// Collect the set of bound impl type keys for every WireSetRef in elements.
	var refInfos []refImplKeys
	for _, elem := range elements {
		we, ok := elem.(*WireSetRef)
		if !ok {
			continue
		}
		ws, ok := t.setIndex[we.Name]
		if !ok {
			continue
		}
		keys := make(map[string]bool)
		for _, bind := range t.collectBindsFromElements(ws.Elements) {
			implType := bind.Implementation
			if ptr, ok2 := implType.(*types.Pointer); ok2 {
				implType = ptr.Elem()
			}
			keys[implType.String()] = true
		}
		if len(keys) > 0 {
			refInfos = append(refInfos, refImplKeys{ref: we, implKeys: keys})
		}
	}

	// Count how many set refs each impl type key appears in.
	implKeyRefCount := make(map[string]int)
	for _, ri := range refInfos {
		for k := range ri.implKeys {
			implKeyRefCount[k]++
		}
	}

	// Determine which set refs touch a shared (count > 1) impl key.
	expandRef := make(map[*WireSetRef]bool)
	for _, ri := range refInfos {
		for k := range ri.implKeys {
			if implKeyRefCount[k] > 1 {
				expandRef[ri.ref] = true
				break
			}
		}
	}

	if len(expandRef) == 0 {
		return elements // nothing to expand
	}

	// Rebuild the element list, replacing conflicting WireSetRefs with their
	// flat contents. Deduplicate WireProviderFunc by function full name to avoid
	// emitting NewImpl twice when both SetA and SetB declare it.
	seenProviders := make(map[string]bool)
	result := make([]WirePattern, 0, len(elements))
	for _, elem := range elements {
		we, ok := elem.(*WireSetRef)
		if !ok || !expandRef[we] {
			result = append(result, elem)
			continue
		}
		ws, ok := t.setIndex[we.Name]
		if !ok {
			result = append(result, elem) // can't resolve: keep as-is
			continue
		}
		for _, inner := range ws.Elements {
			if wpf, ok2 := inner.(*WireProviderFunc); ok2 {
				key := wpf.Name
				if wpf.Func != nil {
					key = wpf.Func.FullName()
				}
				if seenProviders[key] {
					continue // deduplicate identical provider functions
				}
				seenProviders[key] = true
			}
			result = append(result, inner)
		}
	}
	return result
}

// transformElementsWithBoundTypes transforms a list of wire patterns to kessoku patterns,
// merging any extra bound types from an outer scope into the collected bound types.
// extraBoundTypes allows callers (e.g. when flattening an inline nested WireNewSet) to
// pass the outer scope's bound types so that providers that are bound in the outer scope
// are correctly suppressed inside nested sets.
func (t *Transformer) transformElementsWithBoundTypes(elements []WirePattern, pkg *types.Package, setIndex map[string]*WireNewSet, extraBoundTypes map[string]bool) ([]KessokuPattern, error) {
	// Expand WireSetRef siblings that share a concrete implementation type so that
	// the chainedBinds pre-pass below can merge them into one kessoku.Bind chain
	// instead of emitting duplicate kessoku.Provide(Ctor) calls (BUG-15).
	elements = t.expandConflictingSetRefs(elements)

	// First pass: collect all bound implementation types
	// These are the types for which wire.Bind creates an implicit provider
	boundTypes := t.collectBoundTypes(elements)

	// Merge extra bound types from outer scope (e.g. sibling wire.Bind)
	for k := range extraBoundTypes {
		boundTypes[k] = true
	}

	// Merge FieldsOf patterns with the same struct type
	mergedFieldsOf := t.mergeFieldsOf(elements)

	// Pre-pass: group WireBind elements by their concrete implementation type.
	// When multiple WireBind entries share the same implementation type (e.g.
	// Bind(Reader, *RWImpl) + Bind(Writer, *RWImpl)), emitting independent
	// kessoku.Bind[I](kessoku.Provide(Ctor)) calls for each creates duplicate
	// ProviderSpec objects in the codegen graph, which then fails with
	// "multiple providers provide *RWImpl". The fix is to chain the binds:
	//   kessoku.Bind[Writer](kessoku.Bind[Reader](kessoku.Provide(Ctor)))
	// so that all interfaces for the same concrete type share one ProviderSpec.
	//
	// chainedBinds maps the canonicalized implementation type string to the
	// outermost chained KessokuBind (accumulating all interfaces for that type).
	// emittedBindKeys tracks which impl type strings have already been emitted in
	// the main loop so that only the first WireBind occurrence triggers output.
	chainedBinds := make(map[string]*KessokuBind)
	emittedBindKeys := make(map[string]bool)

	for _, elem := range elements {
		wb, ok := elem.(*WireBind)
		if !ok {
			continue
		}

		// Determine the canonical implementation type string (unwrap one pointer level).
		implType := wb.Implementation
		if ptr, ok2 := implType.(*types.Pointer); ok2 {
			implType = ptr.Elem()
		}
		key := implType.String()

		existing, seen := chainedBinds[key]
		if !seen {
			// First bind for this concrete type: create the base KessokuBind.
			base, err := t.transformBind(wb, pkg, elements)
			if err != nil {
				return nil, err
			}
			chainedBinds[key] = base
		} else {
			// Subsequent bind for the same concrete type: chain on top of existing.
			// Wrapping existing in a new KessokuBind means the outermost bind contains
			// the full provider chain; the inner bind already carries the Provide call.
			chained := &KessokuBind{
				Interface: unwrapPointer(wb.Interface),
				Provider:  existing,
				VarName:   wb.VarName,
				SourcePos: wb.Pos,
			}
			chainedBinds[key] = chained
		}
	}

	var result []KessokuPattern

	for _, elem := range elements {
		switch we := elem.(type) {
		case *WireNewSet:
			// Handle inline nested wire.NewSet.
			// Pass the current scope's boundTypes so that a wire.Bind that is a sibling
			// of the nested set in the outer scope is visible when filtering inner providers.
			nestedElements, err := t.transformElementsWithBoundTypes(we.Elements, pkg, setIndex, boundTypes)
			if err != nil {
				return nil, err
			}
			// Flatten nested set elements into parent
			result = append(result, nestedElements...)
		case *WireBind:
			// Emit the chained bind only on the first WireBind occurrence for each
			// implementation type; subsequent occurrences are already folded into the
			// chain built during the pre-pass above.
			implType := we.Implementation
			if ptr, ok2 := implType.(*types.Pointer); ok2 {
				implType = ptr.Elem()
			}
			key := implType.String()

			if !emittedBindKeys[key] {
				emittedBindKeys[key] = true
				result = append(result, chainedBinds[key])
			}
			// If already emitted, this bind's interface is folded into the chain — skip.
		case *WireValue:
			result = append(result, t.transformValue(we))
		case *WireInterfaceValue:
			result = append(result, t.transformInterfaceValue(we))
		case *WireStruct:
			structPatterns, err := t.transformStruct(we, pkg)
			if err != nil {
				return nil, err
			}
			result = append(result, structPatterns...)
		case *WireFieldsOf:
			// Check if this is the first occurrence of this struct type
			typeKey := we.StructType.String()
			merged, exists := mergedFieldsOf[typeKey]
			if !exists {
				continue // Already processed or no merge needed
			}
			// Remove from map so subsequent occurrences are skipped
			delete(mergedFieldsOf, typeKey)
			transformed, err := t.transformFieldsOf(merged, pkg)
			if err != nil {
				return nil, err
			}
			result = append(result, transformed)
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
			//
			// IMPORTANT: exclude this set's own internal Binds from boundTypes before
			// calling allProvidersAreBound. A set that contains both a constructor and
			// a wire.Bind must not suppress itself — only external (sibling) Binds should
			// cause suppression (QA-18).
			if t.setIndex != nil {
				if ws, ok := t.setIndex[we.Name]; ok {
					ownBoundTypes := t.collectBoundTypes(ws.Elements)
					externalBoundTypes := make(map[string]bool, len(boundTypes))
					for k, v := range boundTypes {
						if !ownBoundTypes[k] {
							externalBoundTypes[k] = v
						}
					}
					if allProvidersAreBound(ws, externalBoundTypes, t.setIndex) {
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

// collectBoundTypes collects all implementation types from WireBind elements.
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
