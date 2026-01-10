package main

import (
	"log"

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
	"golang.org/x/tools/go/analysis/passes/modernize"
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

	// modernize analyzers
	analyzers = append(analyzers, modernize.Suite...)

	staticcheckAnalyzers := make([]*lint.Analyzer, 0, len(simple.Analyzers)+len(staticcheck.Analyzers)+len(stylecheck.Analyzers))
	staticcheckAnalyzers = append(staticcheckAnalyzers, simple.Analyzers...)
	staticcheckAnalyzers = append(staticcheckAnalyzers, staticcheck.Analyzers...)
	staticcheckAnalyzers = append(staticcheckAnalyzers, stylecheck.Analyzers...)

	for _, analyzer := range staticcheckAnalyzers {
		analyzers = append(analyzers, analyzer.Analyzer)
	}

	multichecker.Main(analyzers...)
}
