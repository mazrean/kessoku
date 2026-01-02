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

// Generate generates a random wire configuration file.
func (g *WireCodeGenerator) Generate() string {
	var sb strings.Builder

	// Package declaration
	fmt.Fprintf(&sb, "package %s\n\n", g.packageName)

	// Imports
	sb.WriteString("import (\n")
	sb.WriteString("\t\"github.com/google/wire\"\n")
	sb.WriteString(")\n\n")

	// Generate types
	numTypes := g.rand.Intn(5) + 1
	for i := range numTypes {
		g.generateType(&sb, i)
	}

	// Generate interfaces
	numInterfaces := g.rand.Intn(3)
	for i := range numInterfaces {
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

	fieldTypes := []string{"string", "int", "bool", "float64"}
	for i := range numFields {
		fieldName := fmt.Sprintf("Field%d", i)
		fieldType := fieldTypes[g.rand.Intn(len(fieldTypes))]
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
	fmt.Fprintf(sb, "type %s interface {\n", name)

	for i := range numMethods {
		methodName := fmt.Sprintf("Method%d", i)
		methods = append(methods, methodName)
		fmt.Fprintf(sb, "\t%s() string\n", methodName)
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

	// Randomly decide if provider takes parameters
	var params []fuzzParamInfo
	if g.rand.Float32() < 0.3 && len(g.types) > 1 {
		// Add a parameter from another type
		otherTypes := make([]fuzzTypeInfo, 0)
		for _, ot := range g.types {
			if ot.name != t.name {
				otherTypes = append(otherTypes, ot)
			}
		}
		if len(otherTypes) > 0 {
			paramType := otherTypes[g.rand.Intn(len(otherTypes))]
			params = append(params, fuzzParamInfo{
				name:     strings.ToLower(paramType.name[:1]) + paramType.name[1:],
				typeName: "*" + paramType.name,
			})
		}
	}

	// Build function signature
	paramStrs := make([]string, 0, len(params))
	for _, p := range params {
		paramStrs = append(paramStrs, fmt.Sprintf("%s %s", p.name, p.typeName))
	}

	// Randomly decide if provider returns error
	hasError := g.rand.Float32() < 0.3
	returnType := fmt.Sprintf("*%s", t.name)
	if hasError {
		returnType = fmt.Sprintf("(*%s, error)", t.name)
	}

	fmt.Fprintf(sb, "func %s(%s) %s {\n", providerName, strings.Join(paramStrs, ", "), returnType)
	if hasError {
		fmt.Fprintf(sb, "\treturn &%s{}, nil\n", t.name)
	} else {
		fmt.Fprintf(sb, "\treturn &%s{}\n", t.name)
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
	}

	// Always generate at least one NewSet
	g.generateNewSet(sb)

	// Randomly add more patterns
	numAdditional := g.rand.Intn(3)
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

	fmt.Fprintf(sb, "var %s = wire.NewSet(\n", setName)
	for _, p := range selectedProviders {
		fmt.Fprintf(sb, "\t%s,\n", p)
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
	fmt.Fprintf(sb, "var %s = wire.Value(%s)\n\n", varName, v.expr)
}

func (g *WireCodeGenerator) generateStruct(sb *strings.Builder) {
	if len(g.types) == 0 {
		return
	}

	t := g.types[g.rand.Intn(len(g.types))]
	varName := fmt.Sprintf("StructProvider%d", g.rand.Intn(1000))

	if g.rand.Float32() < 0.5 {
		// All fields
		fmt.Fprintf(sb, "var %s = wire.Struct(new(%s), \"*\")\n\n", varName, t.name)
	} else if len(t.fields) > 0 {
		// Specific fields
		numFields := g.rand.Intn(len(t.fields)) + 1
		fieldNames := make([]string, 0, numFields)
		for i := 0; i < numFields && i < len(t.fields); i++ {
			fieldNames = append(fieldNames, fmt.Sprintf("\"%s\"", t.fields[i].name))
		}
		fmt.Fprintf(sb, "var %s = wire.Struct(new(%s), %s)\n\n",
			varName, t.name, strings.Join(fieldNames, ", "))
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

	fmt.Fprintf(sb, "var %s = wire.FieldsOf(new(*%s), %s)\n\n",
		varName, t.name, strings.Join(fieldNames, ", "))
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
		fmt.Fprintf(sb, "func %s() (*%s, error) {\n", funcName, returnType.name)
		fmt.Fprintf(sb, "\twire.Build(%s)\n", strings.Join(selectedProviders, ", "))
		sb.WriteString("\treturn nil, nil\n")
	} else {
		fmt.Fprintf(sb, "func %s() *%s {\n", funcName, returnType.name)
		fmt.Fprintf(sb, "\twire.Build(%s)\n", strings.Join(selectedProviders, ", "))
		sb.WriteString("\treturn nil\n")
	}
	sb.WriteString("}\n\n")
}

// GenerateWithMalformedPatterns generates code with intentionally malformed wire patterns
// to test error handling.
func (g *WireCodeGenerator) GenerateWithMalformedPatterns() string {
	var sb strings.Builder

	// Package declaration
	fmt.Fprintf(&sb, "package %s\n\n", g.packageName)

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
	}

	// Apply random malformed patterns
	numPatterns := g.rand.Intn(len(malformedPatterns)) + 1
	usedIndices := make(map[int]bool)

	for range numPatterns {
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
