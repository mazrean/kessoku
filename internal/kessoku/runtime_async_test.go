package kessoku

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Runtime tests for generated async injector code.
// Each case writes a small module into a temp dir, runs code generation
// in-process, then compiles and executes the generated injector with -race
// to validate its concurrency semantics (no goroutine leaks, no deadlocks,
// correct error propagation).

var (
	kessokuBinOnce sync.Once
	kessokuBinPath string
	kessokuBinErr  error
)

func buildKessokuBin(t *testing.T) string {
	t.Helper()

	kessokuBinOnce.Do(func() {
		repoRoot, err := filepath.Abs("../..")
		if err != nil {
			kessokuBinErr = err
			return
		}

		binDir, err := os.MkdirTemp("", "kessoku-bin-*")
		if err != nil {
			kessokuBinErr = err
			return
		}

		kessokuBinPath = filepath.Join(binDir, "kessoku")
		cmd := exec.CommandContext(context.Background(), "go", "build", "-o", kessokuBinPath, "./cmd/kessoku")
		cmd.Dir = repoRoot
		if out, buildErr := cmd.CombinedOutput(); buildErr != nil {
			kessokuBinErr = &execError{msg: "build kessoku CLI: " + buildErr.Error() + "\n" + string(out)}
		}
	})
	if kessokuBinErr != nil {
		t.Fatalf("failed to build kessoku binary: %v", kessokuBinErr)
	}

	return kessokuBinPath
}

type execError struct{ msg string }

func (e *execError) Error() string { return e.msg }

// runGenerated writes kessoku.go into a fresh module, generates DI code for
// it, then runs `go run -race .` with a hard timeout.
// It returns combined output, whether the run finished before the timeout,
// and the exec error (nil on exit status 0).
func runGenerated(t *testing.T, src string, timeout time.Duration) (output string, finished bool, runErr error) {
	t.Helper()

	return runGeneratedFiles(t, map[string]string{"kessoku.go": src}, timeout)
}

// setupRuntimeModule creates a fresh module in a temp dir with the given
// files (relative path -> content) and returns the module directory.
func setupRuntimeModule(t *testing.T, files map[string]string) string {
	t.Helper()

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	dir := t.TempDir()

	goMod := `module kessokuruntimetest

go 1.25.0

require (
	github.com/mazrean/kessoku v0.0.0-00010101000000-000000000000
	golang.org/x/sync v0.20.0
)

replace github.com/mazrean/kessoku => ` + repoRoot + "\n"
	if writeErr := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600); writeErr != nil {
		t.Fatalf("write go.mod: %v", writeErr)
	}

	goSum, readErr := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if readErr != nil {
		t.Fatalf("read repo go.sum: %v", readErr)
	}
	if writeErr := os.WriteFile(filepath.Join(dir, "go.sum"), goSum, 0o600); writeErr != nil {
		t.Fatalf("write go.sum: %v", writeErr)
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if mkdirErr := os.MkdirAll(filepath.Dir(path), 0o750); mkdirErr != nil {
			t.Fatalf("mkdir for %s: %v", name, mkdirErr)
		}
		if writeErr := os.WriteFile(path, []byte(content), 0o600); writeErr != nil {
			t.Fatalf("write %s: %v", name, writeErr)
		}
	}

	return dir
}

// runtimeModuleEnv returns the environment for commands run inside a module
// created by setupRuntimeModule.
func runtimeModuleEnv() []string {
	return append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
}

// runGeneratedFiles is like runGenerated but accepts multiple files
// (relative path -> content). Code generation runs on kessoku.go.
func runGeneratedFiles(t *testing.T, files map[string]string, timeout time.Duration) (output string, finished bool, runErr error) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping runtime codegen test in short mode")
	}

	bin := buildKessokuBin(t)
	dir := setupRuntimeModule(t, files)
	env := runtimeModuleEnv()

	genCmd := exec.CommandContext(context.Background(), bin, "kessoku.go")
	genCmd.Dir = dir
	genCmd.Env = env
	if out, genErr := genCmd.CombinedOutput(); genErr != nil {
		t.Fatalf("code generation failed: %v\n%s", genErr, out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	runCmd := exec.CommandContext(ctx, "go", "run", "-race", ".")
	runCmd.Dir = dir
	runCmd.Env = env
	out, cmdErr := runCmd.CombinedOutput()
	if ctx.Err() != nil {
		return string(out), false, cmdErr
	}

	return string(out), true, cmdErr
}

// TestAsyncRuntimeGoroutineLeakOnSyncError verifies that when a synchronous
// provider fails while async providers are still running, the injector does
// not leak goroutines: it must cancel and wait for in-flight async providers
// before returning.
func TestAsyncRuntimeGoroutineLeakOnSyncError(t *testing.T) {
	src := `package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/mazrean/kessoku"
)

type DB struct{}
type Cache struct{}
type Messaging struct{}
type App struct{}

var errBoom = errors.New("boom")

// NewDB runs on the main goroutine (first async provider) and fails fast
// while NewCache/NewMessaging are still running in errgroup goroutines.
func NewDB() (*DB, error) { return nil, errBoom }

func NewCache() (*Cache, error) {
	time.Sleep(300 * time.Millisecond)
	return &Cache{}, nil
}

func NewMessaging() (*Messaging, error) {
	time.Sleep(300 * time.Millisecond)
	return &Messaging{}, nil
}

func NewApp(db *DB, c *Cache, m *Messaging) *App { return &App{} }

var _ = kessoku.Inject[*App](
	"InitApp",
	kessoku.Async(kessoku.Provide(NewDB)),
	kessoku.Async(kessoku.Provide(NewCache)),
	kessoku.Async(kessoku.Provide(NewMessaging)),
	kessoku.Provide(NewApp),
)

func main() {
	base := runtime.NumGoroutine()
	_, err := InitApp(context.Background())
	if !errors.Is(err, errBoom) {
		fmt.Printf("FAIL: err = %v, want errBoom\n", err)
		os.Exit(1)
	}
	// The injector must not return while provider goroutines are running.
	deadline := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= base {
			fmt.Println("OK")
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	fmt.Printf("FAIL: goroutine leak: base=%d now=%d\n", base, runtime.NumGoroutine())
	os.Exit(1)
}
`
	out, finished, err := runGenerated(t, src, 60*time.Second)
	if !finished {
		t.Fatalf("generated injector timed out (deadlock):\n%s", out)
	}
	if err != nil || !strings.Contains(out, "OK") {
		t.Fatalf("goroutine leak check failed: %v\n%s", err, out)
	}
}

// TestAsyncRuntimeErrorFidelity verifies that when an async provider fails
// inside an errgroup goroutine, the injector returns the provider's actual
// error rather than the derived context.Canceled.
func TestAsyncRuntimeErrorFidelity(t *testing.T) {
	src := `package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/mazrean/kessoku"
)

type DB struct{}
type Cache struct{}
type Messaging struct{}
type App struct{}

var errCache = errors.New("cache connection failed")

// NewDB succeeds on the main goroutine; NewCache fails inside an errgroup
// goroutine while the main goroutine is waiting on completion channels.
func NewDB() (*DB, error) {
	time.Sleep(50 * time.Millisecond)
	return &DB{}, nil
}

func NewCache() (*Cache, error) {
	time.Sleep(10 * time.Millisecond)
	return nil, errCache
}

func NewMessaging() (*Messaging, error) {
	time.Sleep(100 * time.Millisecond)
	return &Messaging{}, nil
}

func NewApp(db *DB, c *Cache, m *Messaging) (*App, error) { return &App{}, nil }

var _ = kessoku.Inject[*App](
	"InitApp",
	kessoku.Async(kessoku.Provide(NewDB)),
	kessoku.Async(kessoku.Provide(NewCache)),
	kessoku.Async(kessoku.Provide(NewMessaging)),
	kessoku.Provide(NewApp),
)

func main() {
	_, err := InitApp(context.Background())
	if err == nil {
		fmt.Println("FAIL: expected error, got nil")
		os.Exit(1)
	}
	if !errors.Is(err, errCache) {
		fmt.Printf("FAIL: err = %v, want errCache\n", err)
		os.Exit(1)
	}
	fmt.Println("OK")
}
`
	out, finished, err := runGenerated(t, src, 60*time.Second)
	if !finished {
		t.Fatalf("generated injector timed out (deadlock):\n%s", out)
	}
	if err != nil || !strings.Contains(out, "OK") {
		t.Fatalf("error fidelity check failed: %v\n%s", err, out)
	}
}

// TestAsyncRuntimeNoErrorInjectorCancelledContext verifies that an injector
// without an error return completes with correct values even when called
// with an already-cancelled context (it must not deadlock and must not
// silently return zero values).
func TestAsyncRuntimeNoErrorInjectorCancelledContext(t *testing.T) {
	src := `package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mazrean/kessoku"
)

type Config struct{ Name string }
type SvcA struct{ Name string }
type SvcB struct{ Name string }
type App struct{ Name string }

func NewConfig() *Config { return &Config{Name: "cfg"} }

func NewSvcA(c *Config) *SvcA {
	time.Sleep(20 * time.Millisecond)
	return &SvcA{Name: c.Name + "/a"}
}

func NewSvcB(c *Config) *SvcB {
	time.Sleep(20 * time.Millisecond)
	return &SvcB{Name: c.Name + "/b"}
}

func NewApp(a *SvcA, b *SvcB) *App { return &App{Name: a.Name + "+" + b.Name} }

var _ = kessoku.Inject[*App](
	"InitApp",
	kessoku.Provide(NewConfig),
	kessoku.Async(kessoku.Provide(NewSvcA)),
	kessoku.Async(kessoku.Provide(NewSvcB)),
	kessoku.Provide(NewApp),
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled

	done := make(chan *App, 1)
	go func() { done <- InitApp(ctx) }()

	select {
	case app := <-done:
		if app == nil || app.Name != "cfg/a+cfg/b" {
			fmt.Printf("FAIL: unexpected result: %+v\n", app)
			os.Exit(1)
		}
		fmt.Println("OK")
	case <-time.After(10 * time.Second):
		fmt.Println("FAIL: deadlock")
		os.Exit(1)
	}
}
`
	out, finished, err := runGenerated(t, src, 60*time.Second)
	if !finished {
		t.Fatalf("generated injector timed out (deadlock):\n%s", out)
	}
	if err != nil || !strings.Contains(out, "OK") {
		t.Fatalf("cancelled-context check failed: %v\n%s", err, out)
	}
}

// TestAsyncRuntimeValueReturnTypeCompiles verifies that an async injector
// whose return type is a value type (not a pointer) generates compiling
// code on the eg.Wait() error path.
func TestAsyncRuntimeValueReturnTypeCompiles(t *testing.T) {
	src := `package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mazrean/kessoku"
)

type Config struct{ Name string }
type App struct{ Cfg Config }

func NewConfig() (Config, error) {
	time.Sleep(10 * time.Millisecond)
	return Config{Name: "cfg"}, nil
}

func NewSub() (int, error) {
	time.Sleep(10 * time.Millisecond)
	return 42, nil
}

func NewApp(c Config, n int) (App, error) {
	if n != 42 {
		return App{}, fmt.Errorf("bad n: %d", n)
	}
	return App{Cfg: c}, nil
}

var _ = kessoku.Inject[App](
	"InitApp",
	kessoku.Async(kessoku.Provide(NewConfig)),
	kessoku.Async(kessoku.Provide(NewSub)),
	kessoku.Provide(NewApp),
)

func main() {
	app, err := InitApp(context.Background())
	if err != nil || app.Cfg.Name != "cfg" {
		fmt.Printf("FAIL: app=%+v err=%v\n", app, err)
		os.Exit(1)
	}
	fmt.Println("OK")
}
`
	out, finished, err := runGenerated(t, src, 60*time.Second)
	if !finished {
		t.Fatalf("generated injector timed out (deadlock):\n%s", out)
	}
	if err != nil || !strings.Contains(out, "OK") {
		t.Fatalf("value return type check failed: %v\n%s", err, out)
	}
}

// TestAsyncRuntimeHappyPathRace runs a diamond-shaped async graph under the
// race detector as a regression guard for data races in generated code.
func TestAsyncRuntimeHappyPathRace(t *testing.T) {
	src := `package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mazrean/kessoku"
)

type Config struct{ N int }
type L struct{ N int }
type R struct{ N int }
type App struct{ N int }

func NewConfig() (*Config, error) { return &Config{N: 1}, nil }

func NewL(c *Config) (*L, error) {
	time.Sleep(10 * time.Millisecond)
	return &L{N: c.N + 1}, nil
}

func NewR(c *Config) (*R, error) {
	time.Sleep(15 * time.Millisecond)
	return &R{N: c.N + 2}, nil
}

func NewApp(l *L, r *R) (*App, error) { return &App{N: l.N + r.N}, nil }

var _ = kessoku.Inject[*App](
	"InitApp",
	kessoku.Provide(NewConfig),
	kessoku.Async(kessoku.Provide(NewL)),
	kessoku.Async(kessoku.Provide(NewR)),
	kessoku.Provide(NewApp),
)

func main() {
	for i := 0; i < 50; i++ {
		app, err := InitApp(context.Background())
		if err != nil || app.N != 5 {
			fmt.Printf("FAIL: app=%+v err=%v\n", app, err)
			os.Exit(1)
		}
	}
	fmt.Println("OK")
}
`
	out, finished, err := runGenerated(t, src, 120*time.Second)
	if !finished {
		t.Fatalf("generated injector timed out (deadlock):\n%s", out)
	}
	if err != nil || !strings.Contains(out, "OK") {
		t.Fatalf("race regression check failed: %v\n%s", err, out)
	}
}

// TestAsyncRuntimeGenericProviderTypes verifies that generic provider return
// types keep their type arguments in the generated var declarations of async
// injectors.
func TestAsyncRuntimeGenericProviderTypes(t *testing.T) {
	src := `package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mazrean/kessoku"
)

type Container[T any] struct{ Value T }

type App struct{ S string }

func NewStrContainer() (*Container[string], error) {
	time.Sleep(10 * time.Millisecond)
	return &Container[string]{Value: "hello"}, nil
}

func NewIntContainer() (*Container[int], error) {
	time.Sleep(10 * time.Millisecond)
	return &Container[int]{Value: 42}, nil
}

func NewApp(s *Container[string], n *Container[int]) (*App, error) {
	return &App{S: fmt.Sprintf("%s/%d", s.Value, n.Value)}, nil
}

var _ = kessoku.Inject[*App](
	"InitApp",
	kessoku.Async(kessoku.Provide(NewStrContainer)),
	kessoku.Async(kessoku.Provide(NewIntContainer)),
	kessoku.Provide(NewApp),
)

func main() {
	app, err := InitApp(context.Background())
	if err != nil || app.S != "hello/42" {
		fmt.Printf("FAIL: app=%+v err=%v\n", app, err)
		os.Exit(1)
	}
	fmt.Println("OK")
}
`
	out, finished, err := runGenerated(t, src, 60*time.Second)
	if !finished {
		t.Fatalf("generated injector timed out:\n%s", out)
	}
	if err != nil || !strings.Contains(out, "OK") {
		t.Fatalf("generic provider check failed: %v\n%s", err, out)
	}
}

// TestRuntimeVariadicProvider verifies that variadic provider functions are
// called with slice expansion (`args...`) in generated code.
func TestRuntimeVariadicProvider(t *testing.T) {
	src := `package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mazrean/kessoku"
)

type Service struct{ Opts string }

func NewService(opts ...string) *Service {
	return &Service{Opts: strings.Join(opts, ",")}
}

var _ = kessoku.Inject[*Service](
	"InitService",
	kessoku.Provide(NewService),
)

func main() {
	svc := InitService([]string{"a", "b"})
	if svc.Opts != "a,b" {
		fmt.Printf("FAIL: svc=%+v\n", svc)
		os.Exit(1)
	}
	fmt.Println("OK")
}
`
	out, finished, err := runGenerated(t, src, 60*time.Second)
	if !finished {
		t.Fatalf("generated injector timed out:\n%s", out)
	}
	if err != nil || !strings.Contains(out, "OK") {
		t.Fatalf("variadic provider check failed: %v\n%s", err, out)
	}
}

// TestAsyncRuntimeUserIdentifierCollisions verifies that user types whose
// lower-camel names collide with generated identifiers (eg, cancel, ctx)
// do not break the generated code.
func TestAsyncRuntimeUserIdentifierCollisions(t *testing.T) {
	src := `package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mazrean/kessoku"
)

type Eg struct{ N int }
type Cancel struct{ N int }
type Ctx struct{ N int }
type App struct{ N int }

func NewEg() (*Eg, error) {
	time.Sleep(10 * time.Millisecond)
	return &Eg{N: 1}, nil
}

func NewCancel() (*Cancel, error) {
	time.Sleep(10 * time.Millisecond)
	return &Cancel{N: 2}, nil
}

func NewCtx() (*Ctx, error) {
	time.Sleep(10 * time.Millisecond)
	return &Ctx{N: 4}, nil
}

func NewApp(e *Eg, c *Cancel, x *Ctx) (*App, error) {
	return &App{N: e.N + c.N + x.N}, nil
}

var _ = kessoku.Inject[*App](
	"InitApp",
	kessoku.Async(kessoku.Provide(NewEg)),
	kessoku.Async(kessoku.Provide(NewCancel)),
	kessoku.Async(kessoku.Provide(NewCtx)),
	kessoku.Provide(NewApp),
)

func main() {
	app, err := InitApp(context.Background())
	if err != nil || app.N != 7 {
		fmt.Printf("FAIL: app=%+v err=%v\n", app, err)
		os.Exit(1)
	}
	fmt.Println("OK")
}
`
	out, finished, err := runGenerated(t, src, 60*time.Second)
	if !finished {
		t.Fatalf("generated injector timed out:\n%s", out)
	}
	if err != nil || !strings.Contains(out, "OK") {
		t.Fatalf("identifier collision check failed: %v\n%s", err, out)
	}
}

// TestAsyncRuntimeErrgroupImportCollision verifies that a user package whose
// base name is "errgroup" does not break the generated x/sync/errgroup usage.
func TestAsyncRuntimeErrgroupImportCollision(t *testing.T) {
	files := map[string]string{
		"errgroup/errgroup.go": `package errgroup

type Pool struct{ N int }

func NewPool() *Pool { return &Pool{N: 3} }
`,
		"kessoku.go": `package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mazrean/kessoku"

	"kessokuruntimetest/errgroup"
)

type Cache struct{ N int }
type App struct{ N int }

func NewPoolWrapper() (*errgroup.Pool, error) {
	time.Sleep(10 * time.Millisecond)
	return errgroup.NewPool(), nil
}

func NewCache() (*Cache, error) {
	time.Sleep(10 * time.Millisecond)
	return &Cache{N: 4}, nil
}

func NewApp(p *errgroup.Pool, c *Cache) (*App, error) {
	return &App{N: p.N + c.N}, nil
}

var _ = kessoku.Inject[*App](
	"InitApp",
	kessoku.Async(kessoku.Provide(NewPoolWrapper)),
	kessoku.Async(kessoku.Provide(NewCache)),
	kessoku.Provide(NewApp),
)

func main() {
	app, err := InitApp(context.Background())
	if err != nil || app.N != 7 {
		fmt.Printf("FAIL: app=%+v err=%v\n", app, err)
		os.Exit(1)
	}
	fmt.Println("OK")
}
`,
	}
	out, finished, err := runGeneratedFiles(t, files, 60*time.Second)
	if !finished {
		t.Fatalf("generated injector timed out:\n%s", out)
	}
	if err != nil || !strings.Contains(out, "OK") {
		t.Fatalf("errgroup import collision check failed: %v\n%s", err, out)
	}
}

// TestAsyncRuntimeStructFieldAccessWithAsync verifies that struct field
// expansion composes with async providers (predeclared variables must not be
// shadowed by := inside goroutines).
func TestAsyncRuntimeStructFieldAccessWithAsync(t *testing.T) {
	src := `package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mazrean/kessoku"
)

type Config struct {
	Host string
	Port int
}

type Conn struct{ Addr string }
type Cache struct{ N int }
type App struct {
	Addr string
	N    int
}

func NewConfig() *Config { return &Config{Host: "h", Port: 80} }

func NewConn(host string, port int) (*Conn, error) {
	time.Sleep(10 * time.Millisecond)
	return &Conn{Addr: fmt.Sprintf("%s:%d", host, port)}, nil
}

func NewCache() (*Cache, error) {
	time.Sleep(10 * time.Millisecond)
	return &Cache{N: 9}, nil
}

func NewApp(conn *Conn, cache *Cache) (*App, error) {
	return &App{Addr: conn.Addr, N: cache.N}, nil
}

var _ = kessoku.Inject[*App](
	"InitApp",
	kessoku.Provide(NewConfig),
	kessoku.Struct[*Config](),
	kessoku.Async(kessoku.Provide(NewConn)),
	kessoku.Async(kessoku.Provide(NewCache)),
	kessoku.Provide(NewApp),
)

func main() {
	app, err := InitApp(context.Background())
	if err != nil || app.Addr != "h:80" || app.N != 9 {
		fmt.Printf("FAIL: app=%+v err=%v\n", app, err)
		os.Exit(1)
	}
	fmt.Println("OK")
}
`
	out, finished, err := runGenerated(t, src, 60*time.Second)
	if !finished {
		t.Fatalf("generated injector timed out:\n%s", out)
	}
	if err != nil || !strings.Contains(out, "OK") {
		t.Fatalf("struct field access with async check failed: %v\n%s", err, out)
	}
}
