package main

import (
	"errors"
	"fmt"
	"go/types"
	"log"
	"os"

	"golang.org/x/exp/apidiff"
	"golang.org/x/tools/go/packages"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("usage: apicompat <base_package_path> <target_package_path>")
	}

	if err := runAPICompat(os.Args[1], os.Args[2]); err != nil {
		log.Fatalf("Failed to run API compatibility check: %v", err)
	}
}

// runAPICompat runs API compatibility checks between the current version and a base version
func runAPICompat(basePackagePath, targetPackagePath string) error {
	// Load packages for comparison
	basePackages, err := loadPackages(basePackagePath)
	if err != nil {
		return fmt.Errorf("failed to load base packages: %w", err)
	}

	targetPackages, err := loadPackages(targetPackagePath)
	if err != nil {
		return fmt.Errorf("failed to load target packages: %w", err)
	}

	// Compare APIs
	return compareAPIs(basePackages, targetPackages)
}

func loadPackages(packagePath string) (*types.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedTypesInfo | packages.NeedDeps,
	}

	pkgs, err := packages.Load(cfg, packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	for _, pkg := range pkgs {
		if pkg.PkgPath == packagePath {
			return pkg.Types, nil
		}
	}

	return nil, errors.New("not implemented")
}

func compareAPIs(basePackage, targetPackage *types.Package) error {
	report := apidiff.Changes(basePackage, targetPackage)

	// Print only incompatible changes to stdout (empty output means no breaking changes)
	if err := report.TextIncompatible(os.Stdout, false); err != nil {
		return fmt.Errorf("failed to print incompatible changes: %w", err)
	}

	return nil
}
