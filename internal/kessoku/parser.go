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
	"strconv"

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

	kessokuPkg, ok := pkg.Imports[kessokuPkgPath]
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
		Imports: make(map[string]*Import, len(pkg.Imports)),
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
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				slog.Warn("failed to unquote import path", "error", err, "import", imp.Path.Value)
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
func (p *Parser) findInjectDirectives(file *ast.File, pkg *packages.Package, kessokuPackageScope *types.Scope, imports map[string]*Import, fileImports []*ast.ImportSpec, varPool *VarPool) ([]*BuildDirective, error) {
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
func (p *Parser) parseInjectCall(pkg *packages.Package, kessokuPackageScope *types.Scope, call *ast.CallExpr, imports map[string]*Import, fileImports []*ast.ImportSpec, varPool *VarPool) (*BuildDirective, error) {
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
		fun.Index = p.collectDependencies(fun.Index, pkg.TypesInfo, imports, varPool)
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
		fun.Indices[0] = p.collectDependencies(fun.Indices[0], pkg.TypesInfo, imports, varPool)
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
func (p *Parser) parseProviderArgument(pkg *packages.Package, kessokuPackageScope *types.Scope, arg ast.Expr, build *BuildDirective, imports map[string]*Import, fileImports []*ast.ImportSpec, varPool *VarPool) error {
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

	requires, provides, isReturnError, isAsync, err := p.parseProviderType(pkg, providerType, varPool)
	if err != nil {
		return fmt.Errorf("parse provider type: %w", err)
	}

	// Collect dependencies from provider expression
	arg = p.collectDependencies(arg, pkg.TypesInfo, imports, varPool)

	build.Providers = append(build.Providers, &ProviderSpec{
		ASTExpr:       arg,
		Type:          ProviderTypeFunction,
		Provides:      provides,
		Requires:      requires,
		IsReturnError: isReturnError,
		IsAsync:       isAsync,
	})

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
func (p *Parser) collectDependencies(expr ast.Expr, typeInfo *types.Info, imports map[string]*Import, varPool *VarPool) ast.Expr {
	ast.Inspect(expr, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}

		obj := typeInfo.ObjectOf(ident)
		if obj == nil {
			slog.Debug("object of identifier is nil", "identifier", ident.Name)
			return true
		}

		pkgName, ok := obj.(*types.PkgName)
		if !ok {
			slog.Debug("object is not a package name", "identifier", ident.Name, "object", obj)
			return true
		}

		imported := pkgName.Imported()
		if imported == nil {
			slog.Warn("imported package is nil", "identifier", ident.Name, "object", obj)
			return true
		}

		pkgPath := imported.Path()
		if imp, ok := imports[pkgPath]; ok {
			ident.Name = imp.Name
		} else {
			slog.Warn("import not found for package", "package", pkgPath, "identifier", ident.Name)

			// Register the package name to prevent shadowing
			name := varPool.GetName(ident.Name)
			imports[pkgPath] = &Import{
				Name:          name,
				IsDefaultName: name == pkgName.Name(),
			}
		}

		return true
	})

	return expr
}
