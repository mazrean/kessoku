package kessoku

import (
	"bufio"
	"errors"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

const (
	// bindProviderMinTypeArgs is the minimum number of type arguments required for bindProvider
	bindProviderMinTypeArgs = 3
	// bindProviderInternalTypeIndex is the index of the internal provider type in bindProvider type arguments
	bindProviderInternalTypeIndex = 2
	// asyncProviderMinTypeArgs is the minimum number of type arguments required for asyncProvider
	asyncProviderMinTypeArgs = 2
)

// injectorValidationError represents a fatal validation error in a kessoku.Inject call,
// such as an invalid or empty injector name. Unlike soft parse errors, these cause the
// tool to exit with an error rather than skipping the directive.
type injectorValidationError struct {
	msg string
}

func (e *injectorValidationError) Error() string {
	return e.msg
}

// Parser analyzes Go source code to find wire build directives and providers.
type Parser struct {
	fset     *token.FileSet
	packages map[string]*types.Package
}

// NewParser creates a new parser instance.
func NewParser() *Parser {
	return &Parser{
		fset:     token.NewFileSet(),
		packages: make(map[string]*types.Package),
	}
}

// ParseFile parses a Go file and extracts wire build directives.
func (p *Parser) ParseFile(filename string, varPool *VarPool) (*MetaData, []*BuildDirective, error) {
	// Return the raw error unwrapped: processor.go's processFile is the single
	// place that adds the "parse file %s:" prefix (avoids a duplicated prefix).
	if _, err := parser.ParseFile(p.fset, filename, nil, parser.ParseComments); err != nil {
		return nil, nil, err
	}

	pkg, err := p.initializePackages(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("initialize packages: %w", err)
	}

	slog.Debug("package", "pkg", pkg, "filename", filename)

	kessokuPkg, ok := pkg.Imports[kessokuPkgPath]
	if !ok || kessokuPkg == nil {
		slog.Warn("kessoku package is not imported", "filename", filename)
		return nil, nil, nil
	}

	// Find the syntax file that matches our target filename.
	// pkg.Syntax corresponds to pkg.CompiledGoFiles (not pkg.GoFiles);
	// the two slices differ when cgo is present because cgo-generated Go files
	// are inserted into CompiledGoFiles.  Using GoFiles to index Syntax would
	// pick the wrong AST node for any cgo-containing package.
	var targetFile *ast.File
	absFilename, _ := filepath.Abs(filename)
	for i, f := range pkg.Syntax {
		if f != nil && i < len(pkg.CompiledGoFiles) {
			absGoFile, _ := filepath.Abs(pkg.CompiledGoFiles[i])
			if absGoFile == absFilename {
				targetFile = f
				break
			}
		}
	}

	// Dot imports hide the package qualifier the directive scanner relies on,
	// so Inject calls would be silently ignored. Reject them explicitly.
	if targetFile != nil {
		for _, imp := range targetFile.Imports {
			if imp.Name == nil || imp.Name.Name != "." {
				continue
			}
			if path, unquoteErr := strconv.Unquote(imp.Path.Value); unquoteErr == nil && path == kessokuPkgPath {
				return nil, nil, fmt.Errorf("%s: dot import of %s is not supported; import it with a package qualifier", filename, kessokuPkgPath)
			}
		}
	}

	if kessokuPkg.Types == nil {
		slog.Warn("kessoku package is imported, but kessoku.Inject function is not found", "filename", filename)
		return nil, nil, nil
	}

	kessokuPackageScope := kessokuPkg.Types.Scope()
	if kessokuPackageScope == nil {
		slog.Warn("kessoku package is imported, but kessoku.Inject function is not found", "filename", filename)
		return nil, nil, nil
	}

	metaData := &MetaData{
		Package: Package{
			Name: pkg.Name,
			Path: pkg.PkgPath,
		},
		Imports: make(map[string]*Import, len(pkg.Imports)),
	}

	slog.Debug("kessoku package", "kessokuPkg", kessokuPkg)

	for _, f := range pkg.Syntax {
		if f == nil {
			continue
		}

		for _, decl := range f.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					switch spec := spec.(type) {
					case *ast.ValueSpec:
						for _, name := range spec.Names {
							if name == nil {
								continue
							}

							_ = varPool.GetName(name.Name)
						}
					case *ast.TypeSpec:
						if spec.Name == nil {
							continue
						}

						_ = varPool.GetName(spec.Name.Name)
					}
				}
			case *ast.FuncDecl:
				if decl.Name == nil {
					continue
				}

				_ = varPool.GetName(decl.Name.Name)
			}
		}
	}

	for _, f := range pkg.Syntax {
		if f == nil {
			continue
		}

		for _, imp := range f.Imports {
			path, unquoteErr := strconv.Unquote(imp.Path.Value)
			if unquoteErr != nil {
				slog.Warn("failed to unquote import path", "error", unquoteErr, "import", imp.Path.Value)
				continue
			}

			if _, ok := metaData.Imports[path]; ok {
				slog.Debug("import already exists", "path", path)
				continue
			}

			pkgObj, ok := pkg.Imports[path]
			if !ok || pkgObj == nil {
				slog.Warn("imported package not found", "path", path)
				continue
			}
			baseName := pkgObj.Name

			name := varPool.GetName(baseName)

			metaData.Imports[path] = &Import{
				Name:          name,
				IsDefaultName: name == baseName,
				IsUsed:        false, // Will be set to true only when actually used in code generation
			}
		}
	}

	if targetFile == nil {
		return nil, nil, fmt.Errorf("target file not found in package syntax")
	}

	builds, err := p.findInjectDirectives(targetFile, pkg, kessokuPackageScope, metaData.Imports, varPool)
	if err != nil {
		return nil, nil, fmt.Errorf("find inject directives: %w", err)
	}

	return metaData, builds, nil
}

// moduleRootForFile walks parent directories of dir looking for a go.mod file
// and returns the directory that contains it. If no go.mod is found the empty
// string is returned.
func moduleRootForFile(filename string) string {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return ""
	}
	dir := filepath.Dir(absPath)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// reached filesystem root without finding go.mod
			return ""
		}
		dir = parent
	}
}

// fileHasWireinjectTag reports whether the named file has a //go:build wireinject
// (or the legacy // +build wireinject) build constraint.  Only the leading
// comment block before the package clause is scanned, which is sufficient for
// build constraints and avoids reading the entire file.
func fileHasWireinjectTag(filename string) bool {
	f, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Stop once we pass the leading comment block (package clause reached).
		if strings.HasPrefix(trimmed, "package ") {
			break
		}

		// Skip blank lines between comment blocks.
		if trimmed == "" {
			continue
		}

		// Only process lines that look like build constraints.
		if !constraint.IsGoBuild(trimmed) && !constraint.IsPlusBuild(trimmed) {
			continue
		}

		expr, err := constraint.Parse(trimmed)
		if err != nil {
			continue
		}

		if expr.Eval(func(tag string) bool { return tag == "wireinject" }) {
			return true
		}
	}

	return false
}

// initializeSSA initializes SSA analysis for a file.
func (p *Parser) initializePackages(filename string) (*packages.Package, error) {
	// Resolve the filename to an absolute path so that the file= query and
	// module root detection both work regardless of the process CWD.
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("get absolute path: %w", err)
	}

	// If the source file is guarded by //go:build wireinject we must pass
	// -tags=wireinject to packages.Load.  Without it the build tool excludes
	// the file from the package, so the membership check below never finds it
	// and returns "file is not in the same package".  This matters both when a
	// non-wireinject sibling file exists (first run) and after the generated
	// *_band.go (tagged //go:build !wireinject) is present (second run).
	var buildFlags []string
	if fileHasWireinjectTag(absFilename) {
		buildFlags = []string{"-tags=wireinject"}
	}

	// Load packages using the new packages API.
	// Set Dir to the module root that contains the target file so that
	// go/packages can locate go.mod regardless of the process CWD.  This
	// allows `kessoku /abs/path/to/file.go` to work even when the CWD is
	// outside the target module.  When no go.mod is found above the file we
	// leave Dir empty, preserving the existing behaviour of relying on the
	// process CWD.
	cfg := &packages.Config{
		Dir: moduleRootForFile(absFilename),
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes |
			packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedDeps,
		BuildFlags: buildFlags,
		Fset:       p.fset,
	}

	// Use the absolute filename in the file= query so that go/packages
	// can locate the file regardless of which directory Dir is set to.
	pkgs, err := packages.Load(cfg, "file="+absFilename)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	// Any package loading or type-checking errors are fatal: continuing with
	// partially-typed packages lets types.Invalid flow into codegen and produces
	// syntactically broken *_band.go files (e.g. "func GetFoo(invalid invalid type)").
	errorCount := packages.PrintErrors(pkgs)
	if errorCount > 0 {
		return nil, fmt.Errorf("package loading errors occurred (%d error(s)); fix them before running kessoku", errorCount)
	}

	for _, pkg := range pkgs {
		for _, goFile := range pkg.GoFiles {
			absGoFile, err := filepath.Abs(goFile)
			if err != nil {
				slog.Debug("failed to get absolute filename", "error", err, "filename", goFile)
				continue
			}

			if absGoFile == absFilename {
				return pkg, nil
			}
		}
	}

	return nil, errors.New("file is not in the same package")
}

// FindInjectDirectives finds all kessoku.Inject calls in the AST.
func (p *Parser) findInjectDirectives(file *ast.File, pkg *packages.Package, kessokuPackageScope *types.Scope, imports map[string]*Import, varPool *VarPool) ([]*BuildDirective, error) {
	injectorObj := kessokuPackageScope.Lookup("Inject")
	if injectorObj == nil || injectorObj.Type() == nil {
		slog.Warn("kessoku package is imported, but kessoku.Inject function is not found")
		return nil, nil
	}

	injectorType := injectorObj.Type()
	if injectorType == nil {
		slog.Warn("kessoku package is imported, but kessoku.Inject function is not found")
		return nil, nil
	}

	var builds []*BuildDirective

	// Only inspect top-level var declarations. Using ast.Inspect over the entire
	// file would also walk function bodies, causing kessoku.Inject calls inside
	// functions (e.g. init()) to be treated as valid injection points. (QA-9)
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, val := range valSpec.Values {
				callExpr, ok := val.(*ast.CallExpr)
				if !ok {
					continue
				}

				var baseFunc *ast.SelectorExpr

				callFun := callExpr.Fun
				for {
					paren, ok := callFun.(*ast.ParenExpr)
					if !ok {
						break
					}
					callFun = paren.X
				}

				switch fun := callFun.(type) {
				case *ast.IndexExpr:
					if sel, ok := fun.X.(*ast.SelectorExpr); ok {
						baseFunc = sel
					}
				case *ast.IndexListExpr:
					if sel, ok := fun.X.(*ast.SelectorExpr); ok {
						baseFunc = sel
					}
				case *ast.SelectorExpr:
					baseFunc = fun
				}

				if baseFunc == nil {
					// callFun is neither *ast.IndexExpr, *ast.IndexListExpr, nor *ast.SelectorExpr.
					// It may be an *ast.Ident when the user stores kessoku.Inject[*T] in a variable
					// and calls it indirectly (e.g. var injectFn = kessoku.Inject[*T]; injectFn(...)).
					// Detect this pattern by checking TypesInfo.Instances: any generic function
					// instantiation whose origin is kessoku.Inject and whose instantiated type
					// matches the type of callFun is a variable holding an injector.
					ident, isIdent := callFun.(*ast.Ident)
					if !isIdent {
						slog.Debug("baseFunc is nil and callFun is not an identifier", "callExpr", callExpr)
						continue
					}
					identType := pkg.TypesInfo.TypeOf(ident)
					if identType == nil {
						slog.Debug("baseFunc is nil, callFun type unknown", "callExpr", callExpr)
						continue
					}
					foundInjector := false
					for instIdent, inst := range pkg.TypesInfo.Instances {
						useObj := pkg.TypesInfo.Uses[instIdent]
						if useObj == nil || !types.Identical(useObj.Type(), injectorType) {
							continue
						}
						if types.Identical(inst.Type, identType) {
							foundInjector = true
							break
						}
					}
					if !foundInjector {
						slog.Debug("callFun identifier does not hold a kessoku.Inject instantiation", "callExpr", callExpr)
						continue
					}
					// Fall through to parseInjectCall; the *ast.Ident case there will
					// recover the return type from TypesInfo.Instances.
				} else {
					calleeType := pkg.TypesInfo.TypeOf(baseFunc)
					if calleeType == nil {
						slog.Debug("calleeType is nil", "callExpr", callExpr, "baseFunc", baseFunc)
						continue
					}

					if !types.Identical(calleeType, injectorType) {
						slog.Debug("calleeType is not injectorType", "callExpr", callExpr, "calleeType", calleeType, "injectorType", injectorType)
						continue
					}
				}

				build, err := p.parseInjectCall(pkg, kessokuPackageScope, callExpr, imports, varPool)
				if err != nil {
					pos := p.fset.Position(callExpr.Pos())
					return nil, fmt.Errorf("%s: parse kessoku.Inject call: %w", pos, err)
				}

				builds = append(builds, build)
			}
		}
	}

	return builds, nil
}

// parseInjectCall parses a kessoku.Inject call expression.
func (p *Parser) parseInjectCall(pkg *packages.Package, kessokuPackageScope *types.Scope, call *ast.CallExpr, imports map[string]*Import, varPool *VarPool) (*BuildDirective, error) {
	build := &BuildDirective{
		Providers: make([]*ProviderSpec, 0),
	}

	// Extract return type from generic parameter.
	// Unwrap any parentheses around the function expression (e.g. (kessoku.Inject[*T])(...)).
	callFun := call.Fun
	for {
		paren, ok := callFun.(*ast.ParenExpr)
		if !ok {
			break
		}
		callFun = paren.X
	}

	switch fun := callFun.(type) {
	case *ast.IndexExpr:
		returnType := pkg.TypesInfo.TypeOf(fun.Index)
		build.Return = &Return{
			Type:        returnType,
			ASTTypeExpr: fun.Index,
		}
		// Collect dependencies from return type expression
		fun.Index, _ = p.collectDependencies(fun.Index, pkg.TypesInfo, imports, varPool)
	case *ast.IndexListExpr:
		if len(fun.Indices) == 0 {
			return nil, fmt.Errorf("kessoku.Inject requires at least 1 type argument")
		}
		returnType := pkg.TypesInfo.TypeOf(fun.Indices[0])
		build.Return = &Return{
			Type:        returnType,
			ASTTypeExpr: fun.Indices[0],
		}
		// Collect dependencies from return type expression
		fun.Indices[0], _ = p.collectDependencies(fun.Indices[0], pkg.TypesInfo, imports, varPool)
	case *ast.Ident:
		// Variable indirection: the user stored kessoku.Inject[*T] in a variable and
		// calls it as injectFn("Name", providers...).  There is no type-argument AST
		// node in the call expression itself, so we recover the type argument from
		// TypesInfo.Instances, which records every instantiation of a generic function.
		injectorObj := kessokuPackageScope.Lookup("Inject")
		if injectorObj == nil {
			return nil, fmt.Errorf("kessoku.Inject not found in kessoku package scope")
		}
		injType := injectorObj.Type()
		identType := pkg.TypesInfo.TypeOf(fun)
		if identType == nil {
			return nil, fmt.Errorf("kessoku.Inject requires at least 1 type argument")
		}
		var typeArg types.Type
		for instIdent, inst := range pkg.TypesInfo.Instances {
			useObj := pkg.TypesInfo.Uses[instIdent]
			if useObj == nil || !types.Identical(useObj.Type(), injType) {
				continue
			}
			if types.Identical(inst.Type, identType) && inst.TypeArgs != nil && inst.TypeArgs.Len() > 0 {
				typeArg = inst.TypeArgs.At(0)
				break
			}
		}
		if typeArg == nil {
			return nil, fmt.Errorf("kessoku.Inject requires at least 1 type argument")
		}
		astExpr, err := createASTTypeExpr(pkg.Types.Path(), typeArg, varPool, imports)
		if err != nil {
			return nil, fmt.Errorf("create AST type expression for return type: %w", err)
		}
		build.Return = &Return{
			Type:        typeArg,
			ASTTypeExpr: astExpr,
		}
	default:
		return nil, fmt.Errorf("kessoku.Inject requires at least 1 type argument")
	}

	if len(call.Args) == 0 {
		return nil, fmt.Errorf("kessoku.Inject requires at least 1 argument")
	}

	// First argument is the function name (string literal)
	tv, ok := pkg.TypesInfo.Types[call.Args[0]]
	if !ok {
		return nil, fmt.Errorf("get type of first argument")
	}

	if tv.Value == nil || tv.Value.Kind() != constant.String {
		return nil, fmt.Errorf("first argument is not a string literal")
	}

	build.InjectorName = constant.StringVal(tv.Value)

	// Validate injector name: must be a valid Go identifier and not a keyword.
	if !token.IsIdentifier(build.InjectorName) {
		return nil, &injectorValidationError{
			msg: fmt.Sprintf("injector name %q is not a valid Go identifier", build.InjectorName),
		}
	}
	if token.IsKeyword(build.InjectorName) {
		return nil, &injectorValidationError{
			msg: fmt.Sprintf("injector name %q is a Go keyword and cannot be used as a function name", build.InjectorName),
		}
	}
	// "init" is a predeclared identifier (not a keyword), but the Go spec requires
	// every func init to have no arguments and no return values. Generating
	// func init() *T { ... } would therefore fail to compile (QA-25).
	if build.InjectorName == "init" {
		return nil, &injectorValidationError{
			msg: fmt.Sprintf("injector name %q is reserved: func init must have no arguments and no return values", build.InjectorName),
		}
	}
	// "main" in package main: the Go spec requires func main to have no arguments
	// and no return values. Generating func main() *T { ... } in package main
	// would therefore fail to compile.
	if build.InjectorName == "main" && pkg.Name == "main" {
		return nil, &injectorValidationError{
			msg: fmt.Sprintf("injector name %q is reserved in package main: func main must have no arguments and no return values", build.InjectorName),
		}
	}

	// Parse provider arguments (starting from index 1)
	for _, arg := range call.Args[1:] {
		if err := p.parseProviderArgument(pkg, kessokuPackageScope, arg, build, imports, varPool); err != nil {
			return nil, fmt.Errorf("parse provider argument: %w", err)
		}
	}

	return build, nil
}

// parseProviderArgument parses a provider argument in kessoku.Inject call.
func (p *Parser) parseProviderArgument(pkg *packages.Package, kessokuPackageScope *types.Scope, arg ast.Expr, build *BuildDirective, imports map[string]*Import, varPool *VarPool) error {
	providerType := pkg.TypesInfo.TypeOf(arg)
	if providerType == nil {
		return fmt.Errorf("get type of argument")
	}

	setObj := kessokuPackageScope.Lookup("set")
	if setObj == nil {
		slog.Warn("kessoku package is imported, but kessoku.set type is not found")
		return nil
	}

	setType := setObj.Type()
	if setType == nil {
		slog.Warn("kessoku package is imported, but kessoku.set function is not found")
		return nil
	}

	if types.Identical(providerType, setType) {
		var (
			callExpr   *ast.CallExpr
			currentArg = arg
		)
		for callExpr == nil && currentArg != nil {
			switch v := currentArg.(type) {
			case *ast.CallExpr:
				callExpr = v
			case *ast.ParenExpr:
				currentArg = v.X
				continue
			case *ast.Ident:
				obj := pkg.TypesInfo.ObjectOf(v)
				if obj == nil {
					return fmt.Errorf("invalid Set call expression")
				}

				varObj, varOk := obj.(*types.Var)
				if !varOk || varObj == nil {
					return fmt.Errorf("invalid Set call expression")
				}

				if varObj.Pkg().Path() != pkg.PkgPath {
					// The Set variable is from a different package, which happens when the
					// package is dot-imported (import . "other").  Because NeedDeps is set
					// in the packages.Load config, pkg.Imports entries have Syntax and
					// TypesInfo populated, so we can look up the declaration directly.
					importedPkgPath := varObj.Pkg().Path()
					importedPkg, ok := pkg.Imports[importedPkgPath]
					if !ok || importedPkg == nil {
						return fmt.Errorf("dot-imported package not found: %s", importedPkgPath)
					}
					varDeclExpr := p.getVarDecl(importedPkg, varObj)
					if varDeclExpr == nil {
						return fmt.Errorf("var declaration not found in dot-imported package %s: %s", importedPkgPath, varObj.Name())
					}
					if err := p.parseProviderArgument(importedPkg, kessokuPackageScope, varDeclExpr, build, imports, varPool); err != nil {
						return fmt.Errorf("parse dot-imported Set %s.%s: %w", importedPkgPath, varObj.Name(), err)
					}
					return nil
				}

				currentArg = p.getVarDecl(pkg, varObj)
				if currentArg == nil {
					slog.Warn("var declaration not found. Ignoring this Set call.", "obj", varObj)
					return nil
				}
				continue
			case *ast.SelectorExpr:
				//lint:ignore ST1005 is ignored because Set is a kessoku-specific proper noun, so capitalizing it in the error string is intentional and not a generic sentence case issue.
				return fmt.Errorf("Set call expression from another package is not supported: %s", v.Sel.Name)
			default:
				return fmt.Errorf("unsupported Set call expression: %T", currentArg)
			}
		}

		if callExpr == nil {
			return fmt.Errorf("invalid Set call expression")
		}

		for _, setArg := range callExpr.Args {
			if err := p.parseProviderArgument(pkg, kessokuPackageScope, setArg, build, imports, varPool); err != nil {
				return fmt.Errorf("parse Set provider argument: %w", err)
			}
		}

		return nil
	}

	result, err := p.parseProviderType(pkg, providerType, varPool)
	if err != nil {
		return fmt.Errorf("parse provider type: %w", err)
	}

	// Collect dependencies from provider expression and get referenced imports
	var referencedImports map[string]*Import
	arg, referencedImports = p.collectDependencies(arg, pkg.TypesInfo, imports, varPool)

	// Check if this is a struct provider (even if wrapped in Async/Bind)
	if result.IsStruct {
		// Handle struct provider
		if result.StructType == nil {
			return fmt.Errorf("structProvider requires a struct type argument")
		}

		// Extract exported fields from the struct type - fail fast on error
		fields, err := extractExportedFields(result.StructType)
		if err != nil {
			return fmt.Errorf("failed to extract fields from struct %s: %w", result.StructType, err)
		}

		build.Providers = append(build.Providers, &ProviderSpec{
			ASTExpr:           arg,
			Type:              ProviderTypeStruct,
			StructType:        result.StructType,
			StructFields:      fields,
			Provides:          result.Provides,
			Requires:          result.Requires,
			IsReturnError:     result.IsReturnError,
			ErrorType:         result.ErrorType,
			IsAsync:           result.IsAsync,
			ReferencedImports: referencedImports,
		})
	} else {
		build.Providers = append(build.Providers, &ProviderSpec{
			ASTExpr:           arg,
			Type:              ProviderTypeFunction,
			Provides:          result.Provides,
			Requires:          result.Requires,
			IsReturnError:     result.IsReturnError,
			ErrorType:         result.ErrorType,
			IsAsync:           result.IsAsync,
			IsVariadic:        result.IsVariadic,
			ReferencedImports: referencedImports,
		})
	}

	return nil
}

// isCleanupFunc reports whether t is a wire-style cleanup function: func() or
// func() error, with no parameters. Wire supports both forms as cleanup returns.
// kessoku has no way to hand a cleanup back to the injector's caller, so such
// providers are rejected with an explicit error instead of silently discarding
// the cleanup or (worse) deferring it inside the injector, which would tear the
// resource down before the caller ever uses it.
func isCleanupFunc(t types.Type) bool {
	// Named types are user-defined business types (e.g. type ShutdownFunc func()).
	// Even when their underlying type is func() or func() error they are NOT
	// wire-style anonymous cleanup functions and must pass through unchanged.
	if _, isNamed := t.(*types.Named); isNamed {
		return false
	}
	sig, ok := t.Underlying().(*types.Signature)
	if !ok || sig.Params().Len() != 0 {
		return false
	}
	// bare func() — no return values
	if sig.Results().Len() == 0 {
		return true
	}
	// func() error — single error return value
	if sig.Results().Len() == 1 {
		errorIface, _ := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
		return errorIface != nil && types.Implements(sig.Results().At(0).Type(), errorIface)
	}
	return false
}

// parseProviderTypeResult holds the result of parsing a provider type.
type parseProviderTypeResult struct {
	StructType    types.Type
	ErrorType     types.Type
	Requires      []types.Type
	Provides      [][]types.Type
	IsReturnError bool
	IsAsync       bool
	IsStruct      bool
	IsVariadic    bool
}

func (p *Parser) parseProviderType(pkg *packages.Package, providerType types.Type, varPool *VarPool) (*parseProviderTypeResult, error) {
	named, ok := providerType.(*types.Named)
	if !ok {
		slog.Debug("providerType is not a named type", "providerType", providerType)
		return nil, fmt.Errorf("provider type is not a named type")
	}

	typeArgs := named.TypeArgs()
	if typeArgs == nil {
		return nil, fmt.Errorf("provider type has no type arguments")
	}

	switch named.Obj().Name() {
	case "bindProvider":
		if typeArgs.Len() < bindProviderMinTypeArgs {
			break
		}

		interfaceType := typeArgs.At(0)
		internalProviderType := typeArgs.At(bindProviderInternalTypeIndex)

		intrfcType, ok := interfaceType.Underlying().(*types.Interface)
		if !ok {
			return nil, fmt.Errorf("bind type argument is not an interface: %s", interfaceType)
		}

		result, err := p.parseProviderType(pkg, internalProviderType, varPool)
		if err != nil {
			return nil, fmt.Errorf("parse internal provider type: %w", err)
		}

		if len(result.Provides) == 0 {
			return nil, fmt.Errorf("bind requires a provider that returns at least one non-error value; the given provider returns nothing")
		}

		implementingType := false
		for i, provide := range result.Provides {
			for _, providedType := range provide {
				if types.Implements(providedType, intrfcType) {
					result.Provides[i] = append(result.Provides[i], interfaceType)
					implementingType = true
					break
				}
			}
		}

		if !implementingType {
			// Find the first provided type for a useful error message
			var providedTypeName string
			for _, provide := range result.Provides {
				if len(provide) > 0 {
					providedTypeName = provide[0].String()
					break
				}
			}
			return nil, fmt.Errorf("provided type %s does not implement interface %s", providedTypeName, interfaceType)
		}

		// Propagate struct info through bind wrapper
		return result, nil
	case "asyncProvider":
		if typeArgs.Len() < asyncProviderMinTypeArgs {
			return nil, fmt.Errorf("asyncProvider requires at least 2 type arguments")
		}
		internalProviderType := typeArgs.At(1)

		result, err := p.parseProviderType(pkg, internalProviderType, varPool)
		if err != nil {
			return nil, fmt.Errorf("parse internal provider type: %w", err)
		}

		// Mark as async but propagate struct info
		result.IsAsync = true
		return result, nil
	case "fnProvider":
		if typeArgs.Len() < 1 {
			return nil, fmt.Errorf("fnProvider requires at least 1 type argument")
		}

		providerFnSig, ok := typeArgs.At(0).(*types.Signature)
		if !ok || providerFnSig == nil {
			slog.Debug("fnType is nil", "providerType", providerType)
			return nil, fmt.Errorf("fnProvider type argument is not a function signature")
		}

		requires := make([]types.Type, 0, providerFnSig.Params().Len())
		for v := range providerFnSig.Params().Variables() {
			t := v.Type()
			// Guard against types.Invalid, which occurs when the source has type errors
			// (e.g. undefined types).  The initializePackages call should have already
			// returned an error in this case, but we check here as a defence-in-depth
			// measure to avoid emitting syntactically broken generated code.
			if basic, ok := t.(*types.Basic); ok && basic.Kind() == types.Invalid {
				return nil, fmt.Errorf("provider parameter %q has an invalid (unresolved) type; fix type errors in the source first", v.Name())
			}
			requires = append(requires, t)
		}

		errorIface, _ := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
		isReturnError := false
		var errorType types.Type
		results := providerFnSig.Results()
		provides := make([][]types.Type, 0, results.Len())
		for i := range results.Len() {
			v := results.At(i)
			if errorIface != nil && types.Implements(v.Type(), errorIface) {
				if i != results.Len()-1 {
					return nil, fmt.Errorf("provider function has error return in non-last position (index %d of %d); error must be the last return value", i, results.Len()-1)
				}
				isReturnError = true
				errorType = v.Type()
				continue
			}

			// A bare func() return is wire's cleanup-function pattern. kessoku cannot
			// return the cleanup to the injector's caller, so reject it loudly rather
			// than silently leaking the resource or closing it before the caller runs.
			//
			// However, a func() or func() error is also a perfectly valid injectable
			// value (e.g. an event handler, factory closure, or strategy function).
			// The distinction: if the provider's ONLY non-error return value is a
			// func(), that func() IS the value being provided — it cannot be a cleanup
			// side-effect because there is nothing else being provided. Wire-style
			// cleanup only makes sense when the provider also returns at least one
			// concrete value (e.g. func() (*DB, func(), error)).  So we skip the
			// cleanup check for the first non-error return value.
			if len(provides) > 0 && isCleanupFunc(v.Type()) {
				return nil, fmt.Errorf("provider returns a cleanup func(); kessoku does not support wire-style cleanup functions — release the resource explicitly (e.g. expose a Close method on the provided type)")
			}

			provides = append(provides, []types.Type{v.Type()})
		}

		return &parseProviderTypeResult{
			Requires:      requires,
			Provides:      provides,
			ErrorType:     errorType,
			IsReturnError: isReturnError,
			IsAsync:       false,
			IsStruct:      false,
			IsVariadic:    providerFnSig.Variadic(),
		}, nil
	case "structProvider":
		if typeArgs.Len() < 1 {
			return nil, fmt.Errorf("structProvider requires 1 type argument")
		}

		structType := typeArgs.At(0)

		// The struct provider requires the struct type and provides the struct type
		return &parseProviderTypeResult{
			Requires:      []types.Type{structType},
			Provides:      [][]types.Type{{structType}},
			IsReturnError: false,
			IsAsync:       false,
			IsStruct:      true,
			StructType:    structType,
		}, nil
	}

	return nil, errors.New("no valid provider function found")
}

// extractExportedFields extracts exported fields from a struct type.
// Fields are returned in alphabetical order by name for deterministic output.
// Unexported fields are ignored.
func extractExportedFields(t types.Type) ([]*StructFieldSpec, error) {
	// Dereference pointer type if needed
	underlying := t
	if ptr, ok := t.(*types.Pointer); ok {
		underlying = ptr.Elem()
	}

	// Get underlying struct type
	var structType *types.Struct
	switch u := underlying.Underlying().(type) {
	case *types.Struct:
		structType = u
	default:
		return nil, fmt.Errorf("not a struct type: %s", t)
	}

	// Collect exported fields
	var fields []*StructFieldSpec
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		// Skip unexported fields
		if !field.Exported() {
			continue
		}

		fields = append(fields, &StructFieldSpec{
			Type:      field.Type(),
			Name:      field.Name(),
			Index:     i,
			Anonymous: field.Anonymous(),
		})
	}

	// Sort fields alphabetically by name for deterministic output
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})

	return fields, nil
}

func (p *Parser) getVarDecl(pkg *packages.Package, obj *types.Var) ast.Expr {
	objPos := obj.Pos()

	for _, file := range pkg.Syntax {
		if file == nil {
			continue
		}

		path, _ := astutil.PathEnclosingInterval(file, objPos, objPos)

		for _, node := range path {
			if valSpec, ok := node.(*ast.ValueSpec); ok {
				for i, ident := range valSpec.Names {
					if ident.Name == obj.Name() {
						return valSpec.Values[i]
					}
				}
			}
		}
	}

	return nil
}

// collectDependencies extracts package dependencies from an AST expression and
// returns both the modified expression and referenced imports.
//
// Two classes of identifiers are handled:
//
//  1. *types.PkgName — a bare package qualifier that appears as the X of a
//     SelectorExpr (e.g. the "pkg" in "pkg.Foo").  Its Name is rewritten to
//     match the alias recorded in imports.
//
//  2. Any object (*types.Func, *types.TypeName, *types.Var, *types.Const)
//     whose owning package differs from the package that declares the
//     identifier in the AST.  This happens exclusively when a package is
//     dot-imported: the identifier appears as a bare Ident in the source but
//     the object's package is different.  In this case the identifier is
//     rewritten to a SelectorExpr (pkgAlias.Name) so that the generated code,
//     which does not carry the dot-import, can still resolve the symbol.
func (p *Parser) collectDependencies(expr ast.Expr, typeInfo *types.Info, imports map[string]*Import, varPool *VarPool) (ast.Expr, map[string]*Import) {
	referencedImports := make(map[string]*Import)

	// resolveImport finds or registers the Import entry for pkgPath and pkgBaseName,
	// records it in referencedImports, and returns the alias string to use in code.
	resolveImport := func(pkgPath, pkgBaseName string) string {
		if imp, ok := imports[pkgPath]; ok {
			referencedImports[pkgPath] = imp
			return imp.Name
		}
		// Package not yet registered — add it.
		name := varPool.GetName(pkgBaseName)
		newImp := &Import{
			Name:          name,
			IsDefaultName: name == pkgBaseName,
			IsUsed:        false,
		}
		imports[pkgPath] = newImp
		referencedImports[pkgPath] = newImp
		return name
	}

	newExpr := astutil.Apply(expr, func(cursor *astutil.Cursor) bool {
		ident, ok := cursor.Node().(*ast.Ident)
		if !ok {
			return true
		}

		obj := typeInfo.ObjectOf(ident)
		if obj == nil {
			slog.Debug("object of identifier is nil", "identifier", ident.Name)
			return true
		}

		switch typedObj := obj.(type) {
		case *types.PkgName:
			// Case 1: bare package qualifier (X of SelectorExpr).
			imported := typedObj.Imported()
			if imported == nil {
				slog.Warn("imported package is nil", "identifier", ident.Name, "object", obj)
				return true
			}
			alias := resolveImport(imported.Path(), typedObj.Name())
			ident.Name = alias

		default:
			// Case 2: dot-imported symbol — the owning package differs from the
			// package in which the identifier syntactically appears.
			objPkg := obj.Pkg()
			if objPkg == nil {
				return true
			}
			// Determine the package path that the AST node lives in by looking
			// up the identifier's position in typeInfo.  If typeInfo has no
			// package for this position we cannot determine the source package,
			// so we fall back to the object's package.
			identPkg := typeInfo.ObjectOf(ident)
			if identPkg == nil {
				return true
			}
			// The identifier is a dot-imported symbol when its declaring package
			// (objPkg) is NOT the same as the package that contains the
			// SelectorExpr's Sel or a non-PkgName ident at the top of the AST.
			// A simpler heuristic: if the ident's parent is a SelectorExpr and
			// the ident is the Sel (right-hand side), the package qualifier is
			// already present — skip.
			if sel, parentIsSel := cursor.Parent().(*ast.SelectorExpr); parentIsSel && sel.Sel == ident {
				return true
			}
			// If the identifier is also a PkgName it was handled above.
			if _, isPkg := obj.(*types.PkgName); isPkg {
				return true
			}
			// Check whether the owning package is in our imports map.  If it is
			// not, the symbol is from the current package and needs no qualifier.
			if _, inImports := imports[objPkg.Path()]; !inImports {
				// Not in imports — could be a current-package symbol; skip.
				return true
			}
			alias := resolveImport(objPkg.Path(), objPkg.Name())
			// Replace the bare Ident with a SelectorExpr: alias.Name
			cursor.Replace(&ast.SelectorExpr{
				X:   &ast.Ident{Name: alias},
				Sel: &ast.Ident{Name: ident.Name},
			})
		}

		return true
	}, nil)

	if newExpr, ok := newExpr.(ast.Expr); ok {
		return newExpr, referencedImports
	}
	return expr, referencedImports
}
