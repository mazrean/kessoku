package kessoku

import (
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"log/slog"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

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
func (p *Parser) ParseFile(filename string) (*MetaData, []*BuildDirective, error) {
	astFile, err := parser.ParseFile(p.fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse file %s: %w", filename, err)
	}

	pkg, err := p.initializePackages(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("initialize packages: %w", err)
	}

	slog.Debug("package", "pkg", pkg, "filename", filename)

	kessokuPkg, ok := pkg.Imports[kessokuPackage]
	if !ok || kessokuPkg == nil {
		slog.Warn("kessoku package is not imported", "filename", filename)
		return nil, nil, nil
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
		Package: pkg.Name,
		Imports: make(map[string]*ast.ImportSpec),
	}

	slog.Debug("kessoku package", "kessokuPkg", kessokuPkg)

	// Find the syntax file that matches our target filename
	var targetFile *ast.File
	absFilename, _ := filepath.Abs(filename)
	for i, f := range pkg.Syntax {
		if f != nil && i < len(pkg.GoFiles) {
			absGoFile, _ := filepath.Abs(pkg.GoFiles[i])
			if absGoFile == absFilename {
				targetFile = f
				break
			}
		}
	}

	if targetFile == nil {
		return nil, nil, fmt.Errorf("target file not found in package syntax")
	}

	builds, err := p.findInjectDirectives(targetFile, pkg.TypesInfo, kessokuPackageScope, metaData.Imports, astFile.Imports)
	if err != nil {
		return nil, nil, fmt.Errorf("find inject directives: %w", err)
	}

	return metaData, builds, nil
}

// initializeSSA initializes SSA analysis for a file.
func (p *Parser) initializePackages(filename string) (*packages.Package, error) {
	// Load packages using the new packages API
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes |
			packages.NeedSyntax | packages.NeedTypesInfo,
		Fset: p.fset,
	}

	// Load the specific file and its dependencies
	pkgs, err := packages.Load(cfg, "file="+filename)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	// Allow some errors but continue if we have valid packages
	errorCount := packages.PrintErrors(pkgs)
	if errorCount > 0 && len(pkgs) == 0 {
		return nil, fmt.Errorf("package loading errors occurred and no packages loaded")
	}

	for _, pkg := range pkgs {
		absFilename, err := filepath.Abs(filename)
		if err != nil {
			slog.Debug("failed to get absolute filename", "error", err, "filename", filename)
			continue
		}

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
func (p *Parser) findInjectDirectives(file *ast.File, typeInfo *types.Info, kessokuPackageScope *types.Scope, imports map[string]*ast.ImportSpec, fileImports []*ast.ImportSpec) ([]*BuildDirective, error) {
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

	ast.Inspect(file, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		var baseFunc *ast.SelectorExpr

		switch fun := callExpr.Fun.(type) {
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
			slog.Debug("baseFunc is nil", "callExpr", callExpr)
			return true
		}

		calleeType := typeInfo.TypeOf(baseFunc)
		if calleeType == nil {
			slog.Debug("calleeType is nil", "callExpr", callExpr, "baseFunc", baseFunc)
			return true
		}

		if !types.Identical(calleeType, injectorType) {
			slog.Debug("calleeType is not injectorType", "callExpr", callExpr, "calleeType", calleeType, "injectorType", injectorType)
			return true
		}

		build, err := p.parseInjectCall(typeInfo, kessokuPackageScope, callExpr, imports, fileImports)
		if err != nil {
			slog.Warn("parseInjectCall failed", "callExpr", callExpr, "error", err)
			return true
		}

		builds = append(builds, build)
		return false
	})

	return builds, nil
}

// parseInjectCall parses a kessoku.Inject call expression.
func (p *Parser) parseInjectCall(typeInfo *types.Info, kessokuPackageScope *types.Scope, call *ast.CallExpr, imports map[string]*ast.ImportSpec, fileImports []*ast.ImportSpec) (*BuildDirective, error) {
	build := &BuildDirective{
		Providers: make([]*ProviderSpec, 0),
	}

	// Extract return type from generic parameter
	switch fun := call.Fun.(type) {
	case *ast.IndexExpr:
		returnType := typeInfo.TypeOf(fun.Index)
		build.Return = &Return{
			Type:        returnType,
			ASTTypeExpr: fun.Index,
		}
		// Collect dependencies from return type expression
		p.collectDependencies(fun.Index, typeInfo, imports, fileImports)
	case *ast.IndexListExpr:
		if len(call.Fun.(*ast.IndexListExpr).Indices) == 0 {
			return nil, fmt.Errorf("kessoku.Inject requires at least 1 type argument")
		}
		returnType := typeInfo.TypeOf(fun.Indices[0])
		build.Return = &Return{
			Type:        returnType,
			ASTTypeExpr: fun.Indices[0],
		}
		// Collect dependencies from return type expression
		p.collectDependencies(fun.Indices[0], typeInfo, imports, fileImports)
	default:
		return nil, fmt.Errorf("kessoku.Inject requires at least 1 type argument")
	}

	if len(call.Args) == 0 {
		return nil, fmt.Errorf("kessoku.Inject requires at least 1 argument")
	}

	// First argument is the function name (string literal)
	tv, ok := typeInfo.Types[call.Args[0]]
	if !ok {
		return nil, fmt.Errorf("get type of first argument")
	}

	if tv.Value == nil || tv.Value.Kind() != constant.String {
		return nil, fmt.Errorf("first argument is not a string literal")
	}

	build.InjectorName = constant.StringVal(tv.Value)

	// Parse provider arguments (starting from index 1)
	for _, arg := range call.Args[1:] {
		if err := p.parseProviderArgument(typeInfo, kessokuPackageScope, arg, build, imports, fileImports); err != nil {
			return nil, fmt.Errorf("parse provider argument: %w", err)
		}
	}

	return build, nil
}

// parseProviderArgument parses a provider argument in kessoku.Inject call.
func (p *Parser) parseProviderArgument(typeInfo *types.Info, kessokuPackageScope *types.Scope, arg ast.Expr, build *BuildDirective, imports map[string]*ast.ImportSpec, fileImports []*ast.ImportSpec) error {
	providerType := typeInfo.TypeOf(arg)
	if providerType == nil {
		return fmt.Errorf("get type of argument")
	}

	// Check if this is a Set call first
	if callExpr, ok := arg.(*ast.CallExpr); ok {
		if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := selExpr.X.(*ast.Ident); ok {
				if obj := typeInfo.ObjectOf(ident); obj != nil {
					if pkgName, ok := obj.(*types.PkgName); ok && pkgName.Imported().Path() == kessokuPackage {
						if selExpr.Sel.Name == "Set" {
							// This is a kessoku.Set(...) call, parse its arguments as providers
							for _, setArg := range callExpr.Args {
								if err := p.parseProviderArgument(typeInfo, kessokuPackageScope, setArg, build, imports, fileImports); err != nil {
									return fmt.Errorf("parse Set provider argument: %w", err)
								}
							}
							// Collect dependencies from the Set call expression
							p.collectDependencies(arg, typeInfo, imports, fileImports)
							return nil
						}
					}
				}
			}
		}
	}

	methodSet := types.NewMethodSet(providerType)
	for method := range methodSet.Methods() {
		if method == nil {
			slog.Debug("method is nil", "arg", arg)
			continue
		}

		methodObj := method.Obj()
		if methodObj == nil {
			slog.Debug("methodObj is nil", "arg", arg)
			continue
		}

		methodName := methodObj.Name()
		if methodName == "Fn" {
			fnType := methodObj.Type().(*types.Signature)
			if fnType.Params().Len() != 0 || fnType.Results().Len() != 1 {
				slog.Debug("fnType is not a function", "arg", arg)
				continue
			}

			providerFnType := fnType.Results().At(0).Type()
			if providerFnType == nil {
				slog.Warn("get provider function type", "method", methodObj.Name(), "arg", arg)
				continue
			}

			providerFnSig, ok := providerFnType.(*types.Signature)
			if !ok {
				slog.Warn("provider function is not a function", "method", methodObj.Name(), "arg", arg)
				continue
			}

			requires := make([]types.Type, 0, providerFnSig.Params().Len())
			for i := 0; i < providerFnSig.Params().Len(); i++ {
				requires = append(requires, providerFnSig.Params().At(i).Type())
			}

			isReturnError := false
			provides := make([]types.Type, 0, providerFnSig.Results().Len())
			for i := 0; i < providerFnSig.Results().Len(); i++ {
				if types.Identical(providerFnSig.Results().At(i).Type(), types.Universe.Lookup("error").Type()) {
					isReturnError = true
					continue
				}

				provides = append(provides, providerFnSig.Results().At(i).Type())
			}

			// Check if this is a bindProvider - it should provide the interface type instead of concrete type
			if named, ok := providerType.(*types.Named); ok {
				typeName := named.Obj().Name()
				if typeName == "bindProvider" {
					// For bindProvider[S, T], we want to provide type S (the interface)
					// but keep the original requires from the wrapped provider
					if typeArgs := named.TypeArgs(); typeArgs != nil && typeArgs.Len() >= 1 {
						interfaceType := typeArgs.At(0)
						provides = []types.Type{interfaceType}
					}
				}
			}

			build.Providers = append(build.Providers, &ProviderSpec{
				Type:          ProviderTypeFunction,
				Requires:      requires,
				Provides:      provides,
				IsReturnError: isReturnError,
				ASTExpr:       arg,
			})

			// Collect dependencies from provider expression
			p.collectDependencies(arg, typeInfo, imports, fileImports)

			return nil
		}
	}

	callExpr, ok := arg.(*ast.CallExpr)
	if !ok || callExpr.Fun == nil || len(callExpr.Args) != 1 {
		return errors.New("invalid provider expression. kessoku.Arg(name name) must be used directly")
	}

	argNameType, ok := typeInfo.Types[callExpr.Args[0]]
	if !ok || argNameType.Type == nil {
		return errors.New("invalid provider expression. kessoku.Arg(name name) must be used directly")
	}

	if argNameType.Value == nil || argNameType.Value.Kind() != constant.String {
		return errors.New("invalid provider expression. kessoku.Arg requires a string literal")
	}

	argName := constant.StringVal(argNameType.Value)

	var argTypeExpr ast.Expr
	switch fun := callExpr.Fun.(type) {
	case *ast.IndexExpr:
		argTypeExpr = fun.Index
	case *ast.IndexListExpr:
		argTypeExpr = fun.Indices[0]
	default:
		return errors.New("invalid provider expression. kessoku.Arg(name name) must be used directly")
	}

	argType := typeInfo.TypeOf(argTypeExpr)
	if argType == nil {
		return fmt.Errorf("get type of argument")
	}

	build.Arguments = append(build.Arguments, &Argument{
		Name:        argName,
		Type:        argType,
		ASTTypeExpr: argTypeExpr,
	})

	// Collect dependencies from argument type expression
	p.collectDependencies(argTypeExpr, typeInfo, imports, fileImports)

	return nil
}

// collectDependencies extracts package dependencies from an AST expression
func (p *Parser) collectDependencies(expr ast.Expr, typeInfo *types.Info, imports map[string]*ast.ImportSpec, fileImports []*ast.ImportSpec) {
	ast.Inspect(expr, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SelectorExpr:
			// Check if this is a package selector (e.g., fmt.Println)
			if ident, ok := node.X.(*ast.Ident); ok {
				if obj := typeInfo.ObjectOf(ident); obj != nil {
					if pkgName, ok := obj.(*types.PkgName); ok {
						pkgPath := pkgName.Imported().Path()
						// Find the corresponding import spec from the original file
						for _, imp := range fileImports {
							impPath := imp.Path.Value[1 : len(imp.Path.Value)-1] // Remove quotes
							if impPath == pkgPath {
								imports[pkgPath] = imp
								break
							}
						}
					}
				}
			}
		case *ast.Ident:
			// Check if this identifier refers to a type from another package
			if obj := typeInfo.ObjectOf(node); obj != nil {
				if pkg := obj.Pkg(); pkg != nil && pkg.Path() != "" {
					// Find the corresponding import spec from the original file
					for _, imp := range fileImports {
						impPath := imp.Path.Value[1 : len(imp.Path.Value)-1] // Remove quotes
						if impPath == pkg.Path() {
							imports[pkg.Path()] = imp
							break
						}
					}
				}
			}
		}
		return true
	})
}
