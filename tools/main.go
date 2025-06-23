package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alingse/asasalint"
	"github.com/breml/bidichk/pkg/bidichk"
	"github.com/charithe/durationcheck"
	"github.com/go-critic/go-critic/checkers/analyzer"
	"github.com/gordonklaus/ineffassign/pkg/ineffassign"
	"github.com/kisielk/errcheck/errcheck"
	"github.com/kyoh86/exportloopref"
	"github.com/lufeee/execinquery"
	"github.com/nishanths/exhaustive"
	"github.com/sanposhiho/wastedassign/v2"
	"github.com/sonatard/noctx"
	"github.com/tdakkota/asciicheck"
	"github.com/timakin/bodyclose/passes/bodyclose"
	gomnd "github.com/tommy-muehle/go-mnd/v2"
	"github.com/uudashr/iface/unused"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

func main() {
	if len(os.Args) < 2 {
		log.Println("No arguments provided")
		return
	}

	subcommand := os.Args[1]
	os.Args = append([]string{os.Args[0]}, os.Args[2:]...)

	switch subcommand {
	case "lint":
		if err := runLint(); err != nil {
			log.Fatalf("Failed to run lint: %v", err)
		}
	case "apicompat":
		if err := runAPICompat(); err != nil {
			log.Fatalf("Failed to run API compatibility check: %v", err)
		}
	default:
		log.Printf("Invalid subcommand: %s\n", subcommand)
		log.Println("Available subcommands: lint, apicompat")
	}
}

func runLint() error {
	asasalintAnalyzer, err := asasalint.NewAnalyzer(asasalint.LinterSetting{})
	if err != nil {
		log.Fatalf("Failed to create asasalint analyzer: %v", err)
	}

	analyzers := []*analysis.Analyzer{
		// govet default analyzers
		assign.Analyzer,
		atomic.Analyzer,
		atomicalign.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		ctrlflow.Analyzer,
		deepequalerrors.Analyzer,
		errorsas.Analyzer,
		fieldalignment.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		nilness.Analyzer,
		printf.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		sortslice.Analyzer,
		stdmethods.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,

		// golangci-lint default analyzers
		errcheck.Analyzer,
		ineffassign.Analyzer,
		unused.Analyzer,

		// golangci-lint optional analyzers
		asasalintAnalyzer,
		asciicheck.NewAnalyzer(),
		bidichk.NewAnalyzer(),
		bodyclose.Analyzer,
		analyzer.Analyzer,
		noctx.Analyzer,
		gomnd.Analyzer,
		durationcheck.Analyzer,
		exportloopref.Analyzer,
		execinquery.Analyzer,
		exhaustive.Analyzer,
		wastedassign.Analyzer,
	}

	staticcheckAnalyzers := make([]*lint.Analyzer, 0, len(simple.Analyzers)+len(staticcheck.Analyzers)+len(stylecheck.Analyzers))
	staticcheckAnalyzers = append(staticcheckAnalyzers, simple.Analyzers...)
	staticcheckAnalyzers = append(staticcheckAnalyzers, staticcheck.Analyzers...)
	staticcheckAnalyzers = append(staticcheckAnalyzers, stylecheck.Analyzers...)

	for _, analyzer := range staticcheckAnalyzers {
		analyzers = append(analyzers, analyzer.Analyzer)
	}

	multichecker.Main(analyzers...)
	return nil
}

// runAPICompat runs API compatibility checks between the current version and a base version
func runAPICompat() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: apicompat <base_version> [target_version]")
	}

	baseVersion := os.Args[1]
	targetVersion := "."
	if len(os.Args) > 2 {
		targetVersion = os.Args[2]
	}

	log.Printf("Checking API compatibility between %s and %s", baseVersion, targetVersion)

	// Get the module path
	modulePath, err := getModulePath()
	if err != nil {
		return fmt.Errorf("failed to get module path: %w", err)
	}

	// Create temporary directory for base version
	tempDir, err := os.MkdirTemp("", "apicompat-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone base version
	baseDir := filepath.Join(tempDir, "base")
	if err := cloneVersion(modulePath, baseVersion, baseDir); err != nil {
		return fmt.Errorf("failed to clone base version: %w", err)
	}

	// Run apidiff comparison
	return runApidiffComparison(baseDir, targetVersion)
}

// runApidiffComparison runs apidiff to compare APIs between base and target versions
func runApidiffComparison(baseDir, targetDir string) error {
	// Build command to run apidiff
	cmd := exec.Command("go", "run", "golang.org/x/exp/cmd/apidiff", baseDir, targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Running: %s", strings.Join(cmd.Args, " "))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apidiff failed: %w", err)
	}

	return nil
}

// getModulePath returns the module path from go.mod
func getModulePath() (string, error) {
	cmd := exec.Command("go", "list", "-m")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// cloneVersion clones a specific version/tag to a directory
func cloneVersion(modulePath, version, targetDir string) error {
	// Try to get the repository URL from the module path
	repoURL := fmt.Sprintf("https://%s.git", modulePath)

	// Clone the repository
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", version, repoURL, targetDir)
	if err := cmd.Run(); err != nil {
		// If tag doesn't exist, try cloning and checking out
		cmd = exec.Command("git", "clone", repoURL, targetDir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}

		// Checkout the specific version
		cmd = exec.Command("git", "-C", targetDir, "checkout", version)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to checkout version %s: %w", version, err)
		}
	}

	return nil
}
