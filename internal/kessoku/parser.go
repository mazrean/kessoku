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
		Imports: astFile.Imports,
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

	builds, err := p.findInjectDirectives(targetFile, pkg.TypesInfo, kessokuPackageScope)
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
func (p *Parser) findInjectDirectives(file *ast.File, typeInfo *types.Info, kessokuPackageScope *types.Scope) ([]*BuildDirective, error) {
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

		build, err := p.parseInjectCall(typeInfo, kessokuPackageScope, callExpr)
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
func (p *Parser) parseInjectCall(typeInfo *types.Info, kessokuPackageScope *types.Scope, call *ast.CallExpr) (*BuildDirective, error) {
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
	case *ast.IndexListExpr:
		if len(call.Fun.(*ast.IndexListExpr).Indices) == 0 {
			return nil, fmt.Errorf("kessoku.Inject requires at least 1 type argument")
		}
		returnType := typeInfo.TypeOf(fun.Indices[0])
		build.Return = &Return{
			Type:        returnType,
			ASTTypeExpr: fun.Indices[0],
		}
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
		if err := p.parseProviderArgument(typeInfo, kessokuPackageScope, arg, build); err != nil {
			return nil, fmt.Errorf("parse provider argument: %w", err)
		}
	}

	return build, nil
}

// parseProviderArgument parses a provider argument in kessoku.Inject call.
func (p *Parser) parseProviderArgument(typeInfo *types.Info, kessokuPackageScope *types.Scope, arg ast.Expr, build *BuildDirective) error {
	providerType := typeInfo.TypeOf(arg)
	if providerType == nil {
		return fmt.Errorf("get type of argument")
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

			build.Providers = append(build.Providers, &ProviderSpec{
				Type:          ProviderTypeFunction,
				Requires:      requires,
				Provides:      provides,
				IsReturnError: isReturnError,
				ASTExpr:       arg,
			})

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

	return nil
}
