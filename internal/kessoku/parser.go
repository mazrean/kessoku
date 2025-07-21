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
		Package: Package{
			Name: pkg.Name,
			Path: pkg.PkgPath,
		},
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

	builds, err := p.findInjectDirectives(targetFile, pkg, kessokuPackageScope, metaData.Imports, astFile.Imports, varPool)
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
func (p *Parser) findInjectDirectives(file *ast.File, pkg *packages.Package, kessokuPackageScope *types.Scope, imports map[string]*ast.ImportSpec, fileImports []*ast.ImportSpec, varPool *VarPool) ([]*BuildDirective, error) {
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

	// Register imported package names to prevent shadowing
	for _, imp := range fileImports {
		if imp.Name != nil {
			// Register alias name
			varPool.Register(imp.Name.Name)
		} else {
			// Register package name from path
			impPath := imp.Path.Value[1 : len(imp.Path.Value)-1] // Remove quotes
			if pkgObj, exists := pkg.Imports[impPath]; exists {
				varPool.Register(pkgObj.Name)
			}
		}
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

		calleeType := pkg.TypesInfo.TypeOf(baseFunc)
		if calleeType == nil {
			slog.Debug("calleeType is nil", "callExpr", callExpr, "baseFunc", baseFunc)
			return true
		}

		if !types.Identical(calleeType, injectorType) {
			slog.Debug("calleeType is not injectorType", "callExpr", callExpr, "calleeType", calleeType, "injectorType", injectorType)
			return true
		}

		build, err := p.parseInjectCall(pkg, kessokuPackageScope, callExpr, imports, fileImports, varPool)
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
func (p *Parser) parseInjectCall(pkg *packages.Package, kessokuPackageScope *types.Scope, call *ast.CallExpr, imports map[string]*ast.ImportSpec, fileImports []*ast.ImportSpec, varPool *VarPool) (*BuildDirective, error) {
	build := &BuildDirective{
		Providers: make([]*ProviderSpec, 0),
	}

	// Extract return type from generic parameter
	switch fun := call.Fun.(type) {
	case *ast.IndexExpr:
		returnType := pkg.TypesInfo.TypeOf(fun.Index)
		build.Return = &Return{
			Type:        returnType,
			ASTTypeExpr: fun.Index,
		}
		// Collect dependencies from return type expression
		p.collectDependencies(fun.Index, pkg.TypesInfo, imports, fileImports, varPool)
	case *ast.IndexListExpr:
		if len(call.Fun.(*ast.IndexListExpr).Indices) == 0 {
			return nil, fmt.Errorf("kessoku.Inject requires at least 1 type argument")
		}
		returnType := pkg.TypesInfo.TypeOf(fun.Indices[0])
		build.Return = &Return{
			Type:        returnType,
			ASTTypeExpr: fun.Indices[0],
		}
		// Collect dependencies from return type expression
		p.collectDependencies(fun.Indices[0], pkg.TypesInfo, imports, fileImports, varPool)
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

	// Parse provider arguments (starting from index 1)
	for _, arg := range call.Args[1:] {
		if err := p.parseProviderArgument(pkg, kessokuPackageScope, arg, build, imports, fileImports, varPool); err != nil {
			return nil, fmt.Errorf("parse provider argument: %w", err)
		}
	}

	return build, nil
}

// parseProviderArgument parses a provider argument in kessoku.Inject call.
func (p *Parser) parseProviderArgument(pkg *packages.Package, kessokuPackageScope *types.Scope, arg ast.Expr, build *BuildDirective, imports map[string]*ast.ImportSpec, fileImports []*ast.ImportSpec, varPool *VarPool) error {
	providerType := pkg.TypesInfo.TypeOf(arg)
	if providerType == nil {
		return fmt.Errorf("get type of argument")
	}

	setObj := kessokuPackageScope.Lookup("Set")
	if setObj == nil || setObj.Type() == nil {
		slog.Warn("kessoku package is imported, but kessoku.Set function is not found")
		return nil
	}

	setFuncType := setObj.Type()
	if setFuncType == nil {
		slog.Warn("kessoku package is imported, but kessoku.Set function is not found")
		return nil
	}

	// Get the return type of the Set function
	sig, sigOk := setFuncType.(*types.Signature)
	if !sigOk || sig.Results().Len() != 1 {
		slog.Warn("kessoku.Set function has unexpected signature")
		return nil
	}
	setType := sig.Results().At(0).Type()

	if types.Identical(providerType, setType) {
		var (
			callExpr   *ast.CallExpr
			currentArg = arg
		)
		for callExpr == nil && currentArg != nil {
			switch v := currentArg.(type) {
			case *ast.CallExpr:
				callExpr = v
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
					slog.Warn("Set call expression is not in the same package. This is not supported.", "object package", varObj.Pkg().Path(), "pkg", pkg.PkgPath)
					return nil
				}

				currentArg = p.getVarDecl(pkg, varObj)
				if currentArg == nil {
					slog.Warn("var declaration not found. Ignoring this Set call.", "obj", varObj)
					return nil
				}
				continue
			}
		}

		if callExpr == nil {
			return fmt.Errorf("invalid Set call expression")
		}

		for _, setArg := range callExpr.Args {
			if err := p.parseProviderArgument(pkg, kessokuPackageScope, setArg, build, imports, fileImports, varPool); err != nil {
				return fmt.Errorf("parse Set provider argument: %w", err)
			}
		}

		return nil
	}

	// Check if this is a Set call or Set variable first
	if callExpr, callOk := arg.(*ast.CallExpr); callOk {
		if selExpr, selOk := callExpr.Fun.(*ast.SelectorExpr); selOk {
			if ident, identOk := selExpr.X.(*ast.Ident); identOk {
				if obj := pkg.TypesInfo.ObjectOf(ident); obj != nil {
					if pkgName, pkgOk := obj.(*types.PkgName); pkgOk && pkgName.Imported().Path() == kessokuPackage {
						if selExpr.Sel.Name == "Set" {
							// This is a kessoku.Set(...) call, parse its arguments as providers
							for _, setArg := range callExpr.Args {
								if err := p.parseProviderArgument(pkg, kessokuPackageScope, setArg, build, imports, fileImports, varPool); err != nil {
									return fmt.Errorf("parse Set provider argument: %w", err)
								}
							}
							// Collect dependencies from the Set call expression
							p.collectDependencies(arg, pkg.TypesInfo, imports, fileImports, varPool)
							return nil
						}
					}
				}
			}
		}
	}

	requires, provides, isReturnError, isAsync, err := p.parseProviderType(pkg, providerType, varPool)
	if err != nil {
		return fmt.Errorf("parse provider type: %w", err)
	}

	build.Providers = append(build.Providers, &ProviderSpec{
		ASTExpr:       arg,
		Type:          ProviderTypeFunction,
		Provides:      provides,
		Requires:      requires,
		IsReturnError: isReturnError,
		IsAsync:       isAsync,
	})

	// Collect dependencies from provider expression
	p.collectDependencies(arg, pkg.TypesInfo, imports, fileImports, varPool)

	return nil
}

func (p *Parser) parseProviderType(pkg *packages.Package, providerType types.Type, varPool *VarPool) ([]types.Type, [][]types.Type, bool, bool, error) {
	named, ok := providerType.(*types.Named)
	if !ok {
		slog.Debug("providerType is not a named type", "providerType", providerType)
		return nil, nil, false, false, fmt.Errorf("provider type is not a named type")
	}

	typeArgs := named.TypeArgs()
	if typeArgs == nil {
		return nil, nil, false, false, fmt.Errorf("provider type has no type arguments")
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
			return nil, nil, false, false, fmt.Errorf("bind type argument is not an interface: %s", interfaceType)
		}

		requires, provides, isReturnError, isAsync, err := p.parseProviderType(pkg, internalProviderType, varPool)
		if err != nil {
			return nil, nil, false, false, fmt.Errorf("parse internal provider type: %w", err)
		}

		for i, provide := range provides {
			for _, providedType := range provide {
				if types.Implements(providedType, intrfcType) {
					// If the provided type is the interface type, we can skip it
					provides[i] = append(provides[i], interfaceType)
					break
				}
			}
		}

		return requires, provides, isReturnError, isAsync, nil
	case "asyncProvider":
		if typeArgs.Len() < asyncProviderMinTypeArgs {
			return nil, nil, false, false, fmt.Errorf("asyncProvider requires at least 2 type arguments")
		}
		internalProviderType := typeArgs.At(1)

		requires, provides, isReturnError, _, err := p.parseProviderType(pkg, internalProviderType, varPool)
		if err != nil {
			return nil, nil, false, false, fmt.Errorf("parse internal provider type: %w", err)
		}

		return requires, provides, isReturnError, true, nil
	case "fnProvider":
		if typeArgs.Len() < 1 {
			return nil, nil, false, false, fmt.Errorf("fnProvider requires at least 1 type argument")
		}

		providerFnSig, ok := typeArgs.At(0).(*types.Signature)
		if !ok || providerFnSig == nil {
			slog.Debug("fnType is nil", "providerType", providerType)
			return nil, nil, false, false, fmt.Errorf("fnProvider type argument is not a function signature")
		}

		requires := make([]types.Type, 0, providerFnSig.Params().Len())
		for i := range providerFnSig.Params().Len() {
			requires = append(requires, providerFnSig.Params().At(i).Type())
		}

		isReturnError := false
		provides := make([][]types.Type, 0, providerFnSig.Results().Len())
		for i := range providerFnSig.Results().Len() {
			if types.Identical(providerFnSig.Results().At(i).Type(), types.Universe.Lookup("error").Type()) {
				isReturnError = true
				continue
			}

			provides = append(provides, []types.Type{providerFnSig.Results().At(i).Type()})
		}

		// Check if this is a bindProvider or asyncProvider
		isAsync := false

		return requires, provides, isReturnError, isAsync, nil
	}

	return nil, nil, false, false, errors.New("no valid provider function found")
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

// collectDependencies extracts package dependencies from an AST expression
func (p *Parser) collectDependencies(expr ast.Expr, typeInfo *types.Info, imports map[string]*ast.ImportSpec, fileImports []*ast.ImportSpec, varPool *VarPool) {
	ast.Inspect(expr, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SelectorExpr:
			// Check if this is a package selector (e.g., fmt.Println)
			if ident, ok := node.X.(*ast.Ident); ok {
				if obj := typeInfo.ObjectOf(ident); obj != nil {
					if pkgName, ok := obj.(*types.PkgName); ok {
						pkgPath := pkgName.Imported().Path()
						// Register package name to prevent shadowing
						varPool.Register(pkgName.Name())
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
			// Register selector name
			if node.Sel != nil {
				varPool.Register(node.Sel.Name)
			}
		case *ast.Ident:
			// Register identifier name to prevent shadowing
			varPool.Register(node.Name)

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
