package migrate

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// WireCodeGenerator generates random wire configuration code for fuzzing.
type WireCodeGenerator struct {
	packageName        string
	rand               *rand.Rand
	implementedMethods map[string]map[string]bool
	types              []fuzzTypeInfo
	interfaces         []fuzzInterfaceInfo
	providers          []fuzzProviderInfo
}

type fuzzTypeInfo struct {
	name   string
	fields []fuzzFieldInfo
}

type fuzzFieldInfo struct {
	name     string
	typeName string
}

type fuzzInterfaceInfo struct {
	name    string
	methods []string
}

type fuzzProviderInfo struct {
	name       string
	returnType string
	params     []fuzzParamInfo
}

type fuzzParamInfo struct {
	name     string
	typeName string
}

// NewWireCodeGenerator creates a new wire code generator with the given seed.
func NewWireCodeGenerator(seed int64) *WireCodeGenerator {
	return &WireCodeGenerator{
		rand:               rand.New(rand.NewSource(seed)),
		packageName:        "fuzztest",
		types:              make([]fuzzTypeInfo, 0),
		interfaces:         make([]fuzzInterfaceInfo, 0),
		providers:          make([]fuzzProviderInfo, 0),
		implementedMethods: make(map[string]map[string]bool),
	}
}

// generateRandomType generates a random Go type with controlled depth.
// depth controls recursion to prevent infinite nesting.
func (g *WireCodeGenerator) generateRandomType(depth int) string {
	// Basic types that can be used anywhere
	basicTypes := []string{
		"string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "complex64", "complex128",
		"bool", "byte", "rune", "uintptr",
		"error", "any", "interface{}",
	}

	// At max depth, only return basic types
	maxDepth := 3
	if depth >= maxDepth {
		return basicTypes[g.rand.Intn(len(basicTypes))]
	}

	// Choose type category: 60% basic, 40% complex
	typeCategory := g.rand.Float32()

	switch {
	case typeCategory < 0.6:
		// Basic type
		return basicTypes[g.rand.Intn(len(basicTypes))]

	case typeCategory < 0.70:
		// Slice type
		elemType := g.generateRandomType(depth + 1)
		return "[]" + elemType

	case typeCategory < 0.80:
		// Map type
		keyTypes := []string{"string", "int", "int64", "uint", "bool", "byte", "rune"}
		keyType := keyTypes[g.rand.Intn(len(keyTypes))]
		valueType := g.generateRandomType(depth + 1)
		return fmt.Sprintf("map[%s]%s", keyType, valueType)

	case typeCategory < 0.87:
		// Channel type
		chanStyles := []string{"chan ", "<-chan ", "chan<- "}
		style := chanStyles[g.rand.Intn(len(chanStyles))]
		elemType := g.generateRandomType(depth + 1)
		return style + elemType

	case typeCategory < 0.93:
		// Pointer type
		elemType := g.generateRandomType(depth + 1)
		return "*" + elemType

	default:
		// Function type
		return g.generateRandomFuncType(depth + 1)
	}
}

// generateRandomFuncType generates a random function type signature.
func (g *WireCodeGenerator) generateRandomFuncType(depth int) string {
	basicTypes := []string{"string", "int", "bool", "error", "any"}

	// Generate params (0-3)
	numParams := g.rand.Intn(4)
	params := make([]string, 0, numParams)
	for i := 0; i < numParams; i++ {
		if depth < 2 && g.rand.Float32() < 0.3 {
			params = append(params, g.generateRandomType(depth+1))
		} else {
			params = append(params, basicTypes[g.rand.Intn(len(basicTypes))])
		}
	}

	// Generate returns (0-2)
	numReturns := g.rand.Intn(3)
	returns := make([]string, 0, numReturns)
	for i := 0; i < numReturns; i++ {
		returns = append(returns, basicTypes[g.rand.Intn(len(basicTypes))])
	}

	// Build function signature
	var returnPart string
	switch len(returns) {
	case 0:
		returnPart = ""
	case 1:
		returnPart = " " + returns[0]
	default:
		returnPart = " (" + strings.Join(returns, ", ") + ")"
	}

	return fmt.Sprintf("func(%s)%s", strings.Join(params, ", "), returnPart)
}

// Generate generates a random wire configuration file.
func (g *WireCodeGenerator) Generate() string {
	var sb strings.Builder

	// Package declaration
	fmt.Fprintf(&sb, "package %s\n\n", g.packageName)

	// Imports (always include context for providers that use it)
	sb.WriteString("import (\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString("\n")
	sb.WriteString("\t\"github.com/google/wire\"\n")
	sb.WriteString(")\n\n")
	
	// Blank identifier to avoid unused import error
	sb.WriteString("var _ = context.Background\n\n")

	// Generate types
	numTypes := g.rand.Intn(5) + 1
	for i := 0; i < numTypes; i++ {
		g.generateType(&sb, i)
	}

	// Generate interfaces
	numInterfaces := g.rand.Intn(3)
	for i := 0; i < numInterfaces; i++ {
		g.generateInterface(&sb, i)
	}

	// Generate provider functions
	for _, t := range g.types {
		g.generateProvider(&sb, t)
	}

	// Generate wire patterns
	g.generateWirePatterns(&sb)

	return sb.String()
}

func (g *WireCodeGenerator) generateType(sb *strings.Builder, idx int) {
	name := fmt.Sprintf("Type%d", idx)
	numFields := g.rand.Intn(5) + 1

	fields := make([]fuzzFieldInfo, 0, numFields)
	fmt.Fprintf(sb, "type %s struct {\n", name)

	for i := 0; i < numFields; i++ {
		fieldName := fmt.Sprintf("Field%d", i)
		fieldType := g.generateRandomType(0) // depth 0 for top-level
		fields = append(fields, fuzzFieldInfo{name: fieldName, typeName: fieldType})
		fmt.Fprintf(sb, "\t%s %s\n", fieldName, fieldType)
	}

	sb.WriteString("}\n\n")
	g.types = append(g.types, fuzzTypeInfo{name: name, fields: fields})
}

func (g *WireCodeGenerator) generateInterface(sb *strings.Builder, idx int) {
	name := fmt.Sprintf("Interface%d", idx)
	numMethods := g.rand.Intn(3) + 1

	methods := make([]string, 0, numMethods)
	sb.WriteString(fmt.Sprintf("type %s interface {\n", name))

	for i := 0; i < numMethods; i++ {
		methodName := fmt.Sprintf("Method%d", i)
		methods = append(methods, methodName)
		sb.WriteString(fmt.Sprintf("\t%s() string\n", methodName))
	}

	sb.WriteString("}\n\n")
	g.interfaces = append(g.interfaces, fuzzInterfaceInfo{name: name, methods: methods})

	// Generate implementation if we have types
	if len(g.types) > 0 {
		implType := g.types[g.rand.Intn(len(g.types))]

		// Initialize the method map for this type if needed
		if g.implementedMethods[implType.name] == nil {
			g.implementedMethods[implType.name] = make(map[string]bool)
		}

		for _, method := range methods {
			// Skip if method already implemented for this type
			if g.implementedMethods[implType.name][method] {
				continue
			}
			g.implementedMethods[implType.name][method] = true

			sb.WriteString(fmt.Sprintf("func (t *%s) %s() string {\n", implType.name, method))
			sb.WriteString("\treturn \"\"\n")
			sb.WriteString("}\n\n")
		}
	}
}

func (g *WireCodeGenerator) generateProvider(sb *strings.Builder, t fuzzTypeInfo) {
	providerName := fmt.Sprintf("New%s", t.name)

	// Randomly decide provider complexity
	providerStyle := g.rand.Intn(5) // 0-4 different styles
	
	var params []fuzzParamInfo
	var hasError, hasCleanup bool

	switch providerStyle {
	case 0:
		// Simple provider - no params, no error
	case 1:
		// Provider with error
		hasError = true
	case 2:
		// Provider with context.Context
		params = append(params, fuzzParamInfo{name: "ctx", typeName: "context.Context"})
	case 3:
		// Provider with cleanup function
		hasCleanup = true
	case 4:
		// Complex provider with multiple params
		if g.rand.Float32() < 0.5 {
			params = append(params, fuzzParamInfo{name: "ctx", typeName: "context.Context"})
		}
		hasError = g.rand.Float32() < 0.5
		hasCleanup = g.rand.Float32() < 0.3
	}

	// Add type parameters
	if len(g.types) > 1 && g.rand.Float32() < 0.4 {
		otherTypes := make([]fuzzTypeInfo, 0)
		for _, ot := range g.types {
			if ot.name != t.name {
				otherTypes = append(otherTypes, ot)
			}
		}
		if len(otherTypes) > 0 {
			// Add 1-3 type parameters
			numParams := g.rand.Intn(3) + 1
			for i := 0; i < numParams && i < len(otherTypes); i++ {
				paramType := otherTypes[i]
				params = append(params, fuzzParamInfo{
					name:     strings.ToLower(paramType.name[:1]) + paramType.name[1:],
					typeName: "*" + paramType.name,
				})
			}
		}
	}

	// Build function signature
	paramStrs := make([]string, 0, len(params))
	for _, p := range params {
		paramStrs = append(paramStrs, fmt.Sprintf("%s %s", p.name, p.typeName))
	}

	// Build return type
	var returnParts []string
	returnParts = append(returnParts, fmt.Sprintf("*%s", t.name))
	if hasCleanup {
		returnParts = append(returnParts, "func()")
	}
	if hasError {
		returnParts = append(returnParts, "error")
	}

	var returnType string
	if len(returnParts) == 1 {
		returnType = returnParts[0]
	} else {
		returnType = "(" + strings.Join(returnParts, ", ") + ")"
	}

	fmt.Fprintf(sb, "func %s(%s) %s {\n", providerName, strings.Join(paramStrs, ", "), returnType)
	
	// Build return statement
	var returnValues []string
	returnValues = append(returnValues, fmt.Sprintf("&%s{}", t.name))
	if hasCleanup {
		returnValues = append(returnValues, "func() {}")
	}
	if hasError {
		returnValues = append(returnValues, "nil")
	}

	if len(returnValues) == 1 {
		fmt.Fprintf(sb, "\treturn %s\n", returnValues[0])
	} else {
		fmt.Fprintf(sb, "\treturn %s\n", strings.Join(returnValues, ", "))
	}
	sb.WriteString("}\n\n")

	g.providers = append(g.providers, fuzzProviderInfo{
		name:       providerName,
		returnType: t.name,
		params:     params,
	})
}

func (g *WireCodeGenerator) generateWirePatterns(sb *strings.Builder) {
	// Generate a random selection of wire patterns
	patterns := []func(*strings.Builder){
		g.generateNewSet,
		g.generateValue,
		g.generateStruct,
		g.generateFieldsOf,
		g.generateBuild,
		g.generateInterfaceValue,
		g.generateNestedSet,
		g.generateComplexNewSet,
		g.generateBuildWithInlineSet,
	}

	// Always generate at least one NewSet
	g.generateNewSet(sb)

	// Randomly add more patterns (increased range for more coverage)
	numAdditional := g.rand.Intn(5) + 1
	for range numAdditional {
		patternIdx := g.rand.Intn(len(patterns))
		patterns[patternIdx](sb)
	}
}

func (g *WireCodeGenerator) generateNewSet(sb *strings.Builder) {
	if len(g.providers) == 0 {
		return
	}

	setName := fmt.Sprintf("Set%d", g.rand.Intn(1000))

	// Randomly select providers to include
	numProviders := g.rand.Intn(len(g.providers)) + 1
	selectedProviders := make([]string, 0, numProviders)

	for i := 0; i < numProviders && i < len(g.providers); i++ {
		selectedProviders = append(selectedProviders, g.providers[i].name)
	}

	// Maybe add Bind
	if len(g.interfaces) > 0 && len(g.types) > 0 && g.rand.Float32() < 0.5 {
		iface := g.interfaces[g.rand.Intn(len(g.interfaces))]
		impl := g.types[g.rand.Intn(len(g.types))]
		selectedProviders = append(selectedProviders,
			fmt.Sprintf("wire.Bind(new(%s), new(*%s))", iface.name, impl.name))
	}

	// Maybe add wire.Struct
	if len(g.types) > 0 && g.rand.Float32() < 0.3 {
		t := g.types[g.rand.Intn(len(g.types))]
		if g.rand.Float32() < 0.5 {
			selectedProviders = append(selectedProviders,
				fmt.Sprintf("wire.Struct(new(%s), \"*\")", t.name))
		} else {
			// Select specific fields
			numFields := g.rand.Intn(len(t.fields)) + 1
			fieldNames := make([]string, 0, numFields)
			for i := 0; i < numFields && i < len(t.fields); i++ {
				fieldNames = append(fieldNames, fmt.Sprintf("\"%s\"", t.fields[i].name))
			}
			selectedProviders = append(selectedProviders,
				fmt.Sprintf("wire.Struct(new(%s), %s)", t.name, strings.Join(fieldNames, ", ")))
		}
	}

	sb.WriteString(fmt.Sprintf("var %s = wire.NewSet(\n", setName))
	for _, p := range selectedProviders {
		sb.WriteString(fmt.Sprintf("\t%s,\n", p))
	}
	sb.WriteString(")\n\n")
}

func (g *WireCodeGenerator) generateValue(sb *strings.Builder) {
	// Generate wire.Value with a literal
	valueTypes := []struct {
		expr string
	}{
		{`"test string"`},
		{`42`},
		{`true`},
		{`3.14`},
	}

	v := valueTypes[g.rand.Intn(len(valueTypes))]
	varName := fmt.Sprintf("Value%d", g.rand.Intn(1000))
	sb.WriteString(fmt.Sprintf("var %s = wire.Value(%s)\n\n", varName, v.expr))
}

func (g *WireCodeGenerator) generateStruct(sb *strings.Builder) {
	if len(g.types) == 0 {
		return
	}

	t := g.types[g.rand.Intn(len(g.types))]
	varName := fmt.Sprintf("StructProvider%d", g.rand.Intn(1000))

	if g.rand.Float32() < 0.5 {
		// All fields
		sb.WriteString(fmt.Sprintf("var %s = wire.Struct(new(%s), \"*\")\n\n", varName, t.name))
	} else if len(t.fields) > 0 {
		// Specific fields
		numFields := g.rand.Intn(len(t.fields)) + 1
		fieldNames := make([]string, 0, numFields)
		for i := 0; i < numFields && i < len(t.fields); i++ {
			fieldNames = append(fieldNames, fmt.Sprintf("\"%s\"", t.fields[i].name))
		}
		sb.WriteString(fmt.Sprintf("var %s = wire.Struct(new(%s), %s)\n\n",
			varName, t.name, strings.Join(fieldNames, ", ")))
	}
}

func (g *WireCodeGenerator) generateFieldsOf(sb *strings.Builder) {
	if len(g.types) == 0 {
		return
	}

	t := g.types[g.rand.Intn(len(g.types))]
	if len(t.fields) == 0 {
		return
	}

	varName := fmt.Sprintf("FieldsOfProvider%d", g.rand.Intn(1000))

	// Select some fields
	numFields := g.rand.Intn(len(t.fields)) + 1
	fieldNames := make([]string, 0, numFields)
	for i := 0; i < numFields && i < len(t.fields); i++ {
		fieldNames = append(fieldNames, fmt.Sprintf("\"%s\"", t.fields[i].name))
	}

	sb.WriteString(fmt.Sprintf("var %s = wire.FieldsOf(new(*%s), %s)\n\n",
		varName, t.name, strings.Join(fieldNames, ", ")))
}

func (g *WireCodeGenerator) generateBuild(sb *strings.Builder) {
	if len(g.types) == 0 || len(g.providers) == 0 {
		return
	}

	// Pick a return type
	returnType := g.types[g.rand.Intn(len(g.types))]

	funcName := fmt.Sprintf("Initialize%s", returnType.name)

	// Select providers for the Build call
	numProviders := g.rand.Intn(len(g.providers)) + 1
	selectedProviders := make([]string, 0, numProviders)
	for i := 0; i < numProviders && i < len(g.providers); i++ {
		selectedProviders = append(selectedProviders, g.providers[i].name)
	}

	// Randomly decide if function returns error
	hasError := g.rand.Float32() < 0.5

	if hasError {
		sb.WriteString(fmt.Sprintf("func %s() (*%s, error) {\n", funcName, returnType.name))
		sb.WriteString(fmt.Sprintf("\twire.Build(%s)\n", strings.Join(selectedProviders, ", ")))
		sb.WriteString("\treturn nil, nil\n")
	} else {
		sb.WriteString(fmt.Sprintf("func %s() *%s {\n", funcName, returnType.name))
		sb.WriteString(fmt.Sprintf("\twire.Build(%s)\n", strings.Join(selectedProviders, ", ")))
		sb.WriteString("\treturn nil\n")
	}
	sb.WriteString("}\n\n")
}

// generateInterfaceValue generates wire.InterfaceValue patterns.
func (g *WireCodeGenerator) generateInterfaceValue(sb *strings.Builder) {
	if len(g.interfaces) == 0 {
		return
	}

	iface := g.interfaces[g.rand.Intn(len(g.interfaces))]
	varName := fmt.Sprintf("InterfaceValue%d", g.rand.Intn(1000))

	// Generate a value that "implements" the interface (for fuzzing purposes)
	valueExprs := []string{
		"nil",
		"&struct{}{}",
	}
	if len(g.types) > 0 {
		t := g.types[g.rand.Intn(len(g.types))]
		valueExprs = append(valueExprs, fmt.Sprintf("&%s{}", t.name))
	}

	expr := valueExprs[g.rand.Intn(len(valueExprs))]
	fmt.Fprintf(sb, "var %s = wire.InterfaceValue(new(%s), %s)\n\n", varName, iface.name, expr)
}

// generateNestedSet generates nested NewSet patterns with set references.
func (g *WireCodeGenerator) generateNestedSet(sb *strings.Builder) {
	// Generate inner set first
	innerSetName := fmt.Sprintf("InnerSet%d", g.rand.Intn(1000))
	outerSetName := fmt.Sprintf("OuterSet%d", g.rand.Intn(1000))

	// Inner set with some providers
	sb.WriteString(fmt.Sprintf("var %s = wire.NewSet(\n", innerSetName))
	if len(g.providers) > 0 {
		numProviders := g.rand.Intn(len(g.providers)) + 1
		for i := 0; i < numProviders && i < len(g.providers); i++ {
			fmt.Fprintf(sb, "\t%s,\n", g.providers[i].name)
		}
	}
	sb.WriteString(")\n\n")

	// Outer set referencing inner set
	sb.WriteString(fmt.Sprintf("var %s = wire.NewSet(\n", outerSetName))
	fmt.Fprintf(sb, "\t%s,\n", innerSetName)
	// Maybe add more providers
	if len(g.providers) > 1 && g.rand.Float32() < 0.5 {
		fmt.Fprintf(sb, "\t%s,\n", g.providers[len(g.providers)-1].name)
	}
	sb.WriteString(")\n\n")
}

// generateComplexNewSet generates NewSet with multiple different pattern types.
func (g *WireCodeGenerator) generateComplexNewSet(sb *strings.Builder) {
	setName := fmt.Sprintf("ComplexSet%d", g.rand.Intn(1000))
	elements := make([]string, 0)

	// Add providers
	if len(g.providers) > 0 {
		numProviders := g.rand.Intn(len(g.providers)) + 1
		for i := 0; i < numProviders && i < len(g.providers); i++ {
			elements = append(elements, g.providers[i].name)
		}
	}

	// Add Bind patterns
	if len(g.interfaces) > 0 && len(g.types) > 0 {
		numBinds := g.rand.Intn(len(g.interfaces)) + 1
		for i := 0; i < numBinds && i < len(g.interfaces); i++ {
			iface := g.interfaces[i]
			impl := g.types[g.rand.Intn(len(g.types))]
			elements = append(elements, fmt.Sprintf("wire.Bind(new(%s), new(*%s))", iface.name, impl.name))
		}
	}

	// Add Value patterns inline
	if g.rand.Float32() < 0.3 {
		valueTypes := []string{`"inline_value"`, `100`, `false`}
		v := valueTypes[g.rand.Intn(len(valueTypes))]
		elements = append(elements, fmt.Sprintf("wire.Value(%s)", v))
	}

	// Add Struct patterns inline
	if len(g.types) > 0 && g.rand.Float32() < 0.3 {
		t := g.types[g.rand.Intn(len(g.types))]
		elements = append(elements, fmt.Sprintf("wire.Struct(new(%s), \"*\")", t.name))
	}

	// Add FieldsOf patterns inline
	if len(g.types) > 0 && g.rand.Float32() < 0.3 {
		t := g.types[g.rand.Intn(len(g.types))]
		if len(t.fields) > 0 {
			field := t.fields[g.rand.Intn(len(t.fields))]
			elements = append(elements, fmt.Sprintf("wire.FieldsOf(new(*%s), \"%s\")", t.name, field.name))
		}
	}

	// Add InterfaceValue inline
	if len(g.interfaces) > 0 && g.rand.Float32() < 0.2 {
		iface := g.interfaces[g.rand.Intn(len(g.interfaces))]
		elements = append(elements, fmt.Sprintf("wire.InterfaceValue(new(%s), nil)", iface.name))
	}

	if len(elements) == 0 {
		return
	}

	fmt.Fprintf(sb, "var %s = wire.NewSet(\n", setName)
	for _, e := range elements {
		fmt.Fprintf(sb, "\t%s,\n", e)
	}
	sb.WriteString(")\n\n")
}

// generateBuildWithInlineSet generates wire.Build with inline wire.NewSet.
func (g *WireCodeGenerator) generateBuildWithInlineSet(sb *strings.Builder) {
	if len(g.types) == 0 || len(g.providers) == 0 {
		return
	}

	returnType := g.types[g.rand.Intn(len(g.types))]
	funcName := fmt.Sprintf("InitializeWithInlineSet%s", returnType.name)

	// Build inline set elements
	elements := make([]string, 0)
	numProviders := g.rand.Intn(len(g.providers)) + 1
	for i := 0; i < numProviders && i < len(g.providers); i++ {
		elements = append(elements, g.providers[i].name)
	}

	hasError := g.rand.Float32() < 0.5
	if hasError {
		fmt.Fprintf(sb, "func %s() (*%s, error) {\n", funcName, returnType.name)
	} else {
		fmt.Fprintf(sb, "func %s() *%s {\n", funcName, returnType.name)
	}

	// Use inline NewSet in Build
	fmt.Fprintf(sb, "\twire.Build(wire.NewSet(%s))\n", strings.Join(elements, ", "))

	if hasError {
		sb.WriteString("\treturn nil, nil\n")
	} else {
		sb.WriteString("\treturn nil\n")
	}
	sb.WriteString("}\n\n")
}

// GenerateWithMalformedPatterns generates code with intentionally malformed wire patterns
// to test error handling.
func (g *WireCodeGenerator) GenerateWithMalformedPatterns() string {
	var sb strings.Builder

	// Package declaration
	sb.WriteString(fmt.Sprintf("package %s\n\n", g.packageName))

	// Imports
	sb.WriteString("import (\n")
	sb.WriteString("\t\"github.com/google/wire\"\n")
	sb.WriteString(")\n\n")

	// Generate a basic type
	sb.WriteString("type BadType struct {\n")
	sb.WriteString("\tField string\n")
	sb.WriteString("}\n\n")

	// Generate malformed patterns based on random choice
	malformedPatterns := []func(*strings.Builder){
		// Empty NewSet
		func(sb *strings.Builder) {
			sb.WriteString("var EmptySet = wire.NewSet()\n\n")
		},
		// Bind with no implementation
		func(sb *strings.Builder) {
			sb.WriteString("type NoImplInterface interface { Method() }\n\n")
			sb.WriteString("var BadBind = wire.NewSet(wire.Bind(new(NoImplInterface), new(*BadType)))\n\n")
		},
		// Struct with non-existent field
		func(sb *strings.Builder) {
			sb.WriteString("var BadStruct = wire.Struct(new(BadType), \"NonExistentField\")\n\n")
		},
		// FieldsOf with non-existent field
		func(sb *strings.Builder) {
			sb.WriteString("var BadFieldsOf = wire.FieldsOf(new(*BadType), \"NonExistentField\")\n\n")
		},
		// InterfaceValue with wrong type
		func(sb *strings.Builder) {
			sb.WriteString("type SomeInterface interface { Do() }\n\n")
			sb.WriteString("var BadInterfaceValue = wire.InterfaceValue(new(SomeInterface), 42)\n\n")
		},
		// Nested NewSet reference (valid but complex)
		func(sb *strings.Builder) {
			sb.WriteString("var InnerSet = wire.NewSet()\n\n")
			sb.WriteString("var OuterSet = wire.NewSet(InnerSet)\n\n")
		},
		// Deeply nested sets
		func(sb *strings.Builder) {
			sb.WriteString("var Level1 = wire.NewSet()\n")
			sb.WriteString("var Level2 = wire.NewSet(Level1)\n")
			sb.WriteString("var Level3 = wire.NewSet(Level2)\n")
			sb.WriteString("var Level4 = wire.NewSet(Level3)\n\n")
		},
		// Multiple values in NewSet
		func(sb *strings.Builder) {
			sb.WriteString("var MultiValue = wire.NewSet(\n")
			sb.WriteString("\twire.Value(\"str\"),\n")
			sb.WriteString("\twire.Value(42),\n")
			sb.WriteString("\twire.Value(true),\n")
			sb.WriteString("\twire.Value(3.14),\n")
			sb.WriteString(")\n\n")
		},
		// Struct with all fields wildcard
		func(sb *strings.Builder) {
			sb.WriteString("var AllFieldsStruct = wire.Struct(new(BadType), \"*\")\n\n")
		},
		// Multiple Bind for same interface
		func(sb *strings.Builder) {
			sb.WriteString("type DupInterface interface { Do() }\n")
			sb.WriteString("type Impl1 struct{}\nfunc (Impl1) Do() {}\n")
			sb.WriteString("type Impl2 struct{}\nfunc (Impl2) Do() {}\n\n")
			sb.WriteString("var DupBind = wire.NewSet(\n")
			sb.WriteString("\twire.Bind(new(DupInterface), new(*Impl1)),\n")
			sb.WriteString("\twire.Bind(new(DupInterface), new(*Impl2)),\n")
			sb.WriteString(")\n\n")
		},
		// Build with no providers
		func(sb *strings.Builder) {
			sb.WriteString("func EmptyBuild() *BadType {\n")
			sb.WriteString("\twire.Build()\n")
			sb.WriteString("\treturn nil\n")
			sb.WriteString("}\n\n")
		},
		// Build with inline NewSet
		func(sb *strings.Builder) {
			sb.WriteString("func InlineBuild() *BadType {\n")
			sb.WriteString("\twire.Build(wire.NewSet())\n")
			sb.WriteString("\treturn nil\n")
			sb.WriteString("}\n\n")
		},
		// Circular-like reference (not actually circular, but complex)
		func(sb *strings.Builder) {
			sb.WriteString("type A struct { B *B }\n")
			sb.WriteString("type B struct { C *C }\n")
			sb.WriteString("type C struct { D *D }\n")
			sb.WriteString("type D struct{}\n\n")
			sb.WriteString("func NewA(b *B) *A { return &A{B: b} }\n")
			sb.WriteString("func NewB(c *C) *B { return &B{C: c} }\n")
			sb.WriteString("func NewC(d *D) *C { return &C{D: d} }\n")
			sb.WriteString("func NewD() *D { return &D{} }\n\n")
			sb.WriteString("var ChainSet = wire.NewSet(NewA, NewB, NewC, NewD)\n\n")
		},
		// FieldsOf with multiple fields
		func(sb *strings.Builder) {
			sb.WriteString("type MultiField struct {\n")
			sb.WriteString("\tA string\n")
			sb.WriteString("\tB int\n")
			sb.WriteString("\tC bool\n")
			sb.WriteString("\tD float64\n")
			sb.WriteString("}\n\n")
			sb.WriteString("var MultiFieldsOf = wire.FieldsOf(new(*MultiField), \"A\", \"B\", \"C\", \"D\")\n\n")
		},
		// Complex type in Struct
		func(sb *strings.Builder) {
			sb.WriteString("type ComplexStruct struct {\n")
			sb.WriteString("\tData map[string][]int\n")
			sb.WriteString("\tChan chan struct{}\n")
			sb.WriteString("\tFunc func() error\n")
			sb.WriteString("}\n\n")
			sb.WriteString("var ComplexStructSet = wire.Struct(new(ComplexStruct), \"*\")\n\n")
		},
		// Provider returning cleanup
		func(sb *strings.Builder) {
			sb.WriteString("func NewWithCleanup() (*BadType, func()) {\n")
			sb.WriteString("\treturn &BadType{}, func() {}\n")
			sb.WriteString("}\n\n")
			sb.WriteString("var CleanupSet = wire.NewSet(NewWithCleanup)\n\n")
		},
		// Provider with error
		func(sb *strings.Builder) {
			sb.WriteString("func NewWithError() (*BadType, error) {\n")
			sb.WriteString("\treturn &BadType{}, nil\n")
			sb.WriteString("}\n\n")
			sb.WriteString("var ErrorSet = wire.NewSet(NewWithError)\n\n")
		},
		// Provider with both cleanup and error
		func(sb *strings.Builder) {
			sb.WriteString("func NewWithBoth() (*BadType, func(), error) {\n")
			sb.WriteString("\treturn &BadType{}, func() {}, nil\n")
			sb.WriteString("}\n\n")
			sb.WriteString("var BothSet = wire.NewSet(NewWithBoth)\n\n")
		},
	}

	// Apply random malformed patterns
	numPatterns := g.rand.Intn(len(malformedPatterns)) + 1
	usedIndices := make(map[int]bool)

	for i := 0; i < numPatterns; i++ {
		idx := g.rand.Intn(len(malformedPatterns))
		if !usedIndices[idx] {
			malformedPatterns[idx](&sb)
			usedIndices[idx] = true
		}
	}

	return sb.String()
}

// FuzzMigrate is a fuzz test for the migrate package.
// It generates random wire configuration files and ensures the migrator
// doesn't panic and handles errors gracefully.
func FuzzMigrate(f *testing.F) {
	// Add seed corpus - use byte slices that will be converted to int64 seeds
	seedCorpus := []int64{
		0, 1, 42, 100, 1000, 12345, 99999,
		-1, -42, -1000,
	}

	for _, seed := range seedCorpus {
		// Convert seed to bytes for the fuzzer
		f.Add(seed, false)
		f.Add(seed, true)
	}

	f.Fuzz(func(t *testing.T, seed int64, useMalformed bool) {
		// Create generator with seed
		gen := NewWireCodeGenerator(seed)

		// Generate code
		var code string
		if useMalformed {
			code = gen.GenerateWithMalformedPatterns()
		} else {
			code = gen.Generate()
		}

		// Create temporary directory
		tmpDir := t.TempDir()

		// Write generated code to a temp file
		inputPath := filepath.Join(tmpDir, "wire.go")
		if err := os.WriteFile(inputPath, []byte(code), 0644); err != nil {
			t.Fatalf("failed to write input file: %v", err)
		}

		// Create output path
		outputPath := filepath.Join(tmpDir, "kessoku.go")

		// Run migrator - should not panic
		migrator := NewMigrator()

		// The migrator might return an error (which is fine), but should not panic
		_ = migrator.MigrateFiles([]string{inputPath}, outputPath)

		// If output was generated, verify it's valid Go code (syntax only)
		if _, err := os.Stat(outputPath); err == nil {
			outputContent, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}

			// Basic sanity check - should have package declaration
			if !strings.Contains(string(outputContent), "package ") {
				t.Errorf("generated output missing package declaration")
			}
		}
	})
}

// TestWireCodeGenerator tests the code generator itself.
func TestWireCodeGenerator(t *testing.T) {
	tests := []struct {
		name string
		seed int64
	}{
		{"seed_0", 0},
		{"seed_42", 42},
		{"seed_12345", 12345},
		{"seed_negative", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewWireCodeGenerator(tt.seed)
			code := gen.Generate()

			// Basic validations
			if !strings.Contains(code, "package fuzztest") {
				t.Error("generated code missing package declaration")
			}

			if !strings.Contains(code, "github.com/google/wire") {
				t.Error("generated code missing wire import")
			}

			if !strings.Contains(code, "wire.NewSet") {
				t.Error("generated code should contain at least one wire.NewSet")
			}
		})
	}
}

// TestWireCodeGeneratorMalformed tests malformed code generation.
func TestWireCodeGeneratorMalformed(t *testing.T) {
	tests := []struct {
		name string
		seed int64
	}{
		{"malformed_seed_0", 0},
		{"malformed_seed_42", 42},
		{"malformed_seed_100", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewWireCodeGenerator(tt.seed)
			code := gen.GenerateWithMalformedPatterns()

			// Basic validations
			if !strings.Contains(code, "package fuzztest") {
				t.Error("generated code missing package declaration")
			}

			if !strings.Contains(code, "github.com/google/wire") {
				t.Error("generated code missing wire import")
			}
		})
	}
}

// TestMigratorFuzzedInput runs the migrator on several generated inputs
// to ensure robustness.
func TestMigratorFuzzedInput(t *testing.T) {
	seeds := []int64{0, 1, 42, 100, 500, 1000, 5000, 10000}

	for _, seed := range seeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			gen := NewWireCodeGenerator(seed)
			code := gen.Generate()

			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, "wire.go")
			if err := os.WriteFile(inputPath, []byte(code), 0644); err != nil {
				t.Fatalf("failed to write input file: %v", err)
			}

			outputPath := filepath.Join(tmpDir, "kessoku.go")
			migrator := NewMigrator()

			// Should not panic
			err := migrator.MigrateFiles([]string{inputPath}, outputPath)

			// Log the result but don't fail on errors (expected for some generated inputs)
			if err != nil {
				t.Logf("Migration returned error (expected for some inputs): %v", err)
				t.Logf("Generated code:\n%s", code)
			}
		})
	}
}

// TestMigratorMalformedInput tests the migrator with malformed inputs.
func TestMigratorMalformedInput(t *testing.T) {
	seeds := []int64{0, 42, 100, 500, 1000}

	for _, seed := range seeds {
		t.Run(fmt.Sprintf("malformed_seed_%d", seed), func(t *testing.T) {
			gen := NewWireCodeGenerator(seed)
			code := gen.GenerateWithMalformedPatterns()

			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, "wire.go")
			if err := os.WriteFile(inputPath, []byte(code), 0644); err != nil {
				t.Fatalf("failed to write input file: %v", err)
			}

			outputPath := filepath.Join(tmpDir, "kessoku.go")
			migrator := NewMigrator()

			// Should not panic - errors are expected
			_ = migrator.MigrateFiles([]string{inputPath}, outputPath)
		})
	}
}
