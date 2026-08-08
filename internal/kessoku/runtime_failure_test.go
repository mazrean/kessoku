package kessoku

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Failure-mode tests for the CLI: invalid inputs must fail loudly with a
// non-zero exit code instead of silently generating nothing.

// runGenerationExpectFailure runs code generation on the given files and
// returns the combined output. It fails the test when generation SUCCEEDS.
func runGenerationExpectFailure(t *testing.T, files map[string]string, genArgs ...string) string {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping runtime codegen test in short mode")
	}

	bin := buildKessokuBin(t)
	dir := setupRuntimeModule(t, files)

	genCmd := exec.CommandContext(context.Background(), bin, genArgs...)
	genCmd.Dir = dir
	genCmd.Env = runtimeModuleEnv()
	out, err := genCmd.CombinedOutput()
	if err == nil {
		t.Fatalf("code generation succeeded, want non-zero exit\n%s", out)
	}

	return string(out)
}

// TestGenerationFailsOnInvalidDirective verifies that an Inject directive the
// parser cannot process aborts generation with a non-zero exit code rather
// than being skipped with only a warning log.
func TestGenerationFailsOnInvalidDirective(t *testing.T) {
	files := map[string]string{
		"kessoku.go": `package main

import "github.com/mazrean/kessoku"

type Concrete struct{}

func NewConcrete() *Concrete { return &Concrete{} }

// Bind's first type argument must be an interface; *Concrete is invalid.
var _ = kessoku.Inject[*Concrete](
	"InitConcrete",
	kessoku.Bind[*Concrete](kessoku.Provide(NewConcrete)),
)

func main() {}
`,
	}

	out := runGenerationExpectFailure(t, files, "kessoku.go")
	if !strings.Contains(out, "interface") {
		t.Errorf("error output should mention the invalid interface binding, got:\n%s", out)
	}
}

// TestGenerationFailsOnDotImport verifies that dot-importing kessoku produces
// an explicit error instead of silently generating nothing.
func TestGenerationFailsOnDotImport(t *testing.T) {
	files := map[string]string{
		"kessoku.go": `package main

import . "github.com/mazrean/kessoku"

type App struct{}

func NewApp() *App { return &App{} }

var _ = Inject[*App](
	"InitApp",
	Provide(NewApp),
)

func main() {}
`,
	}

	out := runGenerationExpectFailure(t, files, "kessoku.go")
	if !strings.Contains(out, "dot import") {
		t.Errorf("error output should mention unsupported dot import, got:\n%s", out)
	}
}

// TestGenerationFailsOnDuplicateInjectorNames verifies that two Inject
// directives generating the same function name in one package are rejected
// instead of emitting colliding declarations.
func TestGenerationFailsOnDuplicateInjectorNames(t *testing.T) {
	files := map[string]string{
		"kessoku.go": `package main

import "github.com/mazrean/kessoku"

type A struct{}

func NewA() *A { return &A{} }

var _ = kessoku.Inject[*A](
	"InitApp",
	kessoku.Provide(NewA),
)

func main() {}
`,
		"second.go": `package main

import "github.com/mazrean/kessoku"

type B struct{}

func NewB() *B { return &B{} }

var _ = kessoku.Inject[*B](
	"InitApp",
	kessoku.Provide(NewB),
)
`,
	}

	dir := setupRuntimeModule(t, files)
	out := func() string {
		t.Helper()
		bin := buildKessokuBin(t)
		genCmd := exec.CommandContext(context.Background(), bin, "kessoku.go", "second.go")
		genCmd.Dir = dir
		genCmd.Env = runtimeModuleEnv()
		outBytes, err := genCmd.CombinedOutput()
		if err == nil {
			t.Fatalf("code generation succeeded, want duplicate-name error\n%s", outBytes)
		}
		return string(outBytes)
	}()

	if !strings.Contains(out, "duplicate") {
		t.Errorf("error output should mention the duplicate injector name, got:\n%s", out)
	}

	// No partial output should be left behind.
	for _, band := range []string{"kessoku_band.go", "second_band.go"} {
		if _, err := os.Stat(filepath.Join(dir, band)); err == nil {
			t.Errorf("partial generated file %s left behind after failure", band)
		}
	}
}
