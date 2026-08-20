module github.com/mazrean/kessoku/tools

go 1.26.0

require (
	github.com/alingse/asasalint v0.0.11
	github.com/breml/bidichk v0.3.3
	github.com/charithe/durationcheck v0.0.11
	github.com/go-critic/go-critic v0.14.4
	github.com/gordonklaus/ineffassign v0.2.0
	github.com/kisielk/errcheck v1.20.0
	github.com/kyoh86/exportloopref v0.1.11
	github.com/lufeee/execinquery v1.2.1
	github.com/nishanths/exhaustive v0.12.0
	github.com/sanposhiho/wastedassign/v2 v2.1.0
	github.com/sonatard/noctx v0.5.1
	github.com/tdakkota/asciicheck v0.4.1
	github.com/timakin/bodyclose v0.0.0-20260723120731-857993a2939c
	github.com/tommy-muehle/go-mnd/v2 v2.5.1
	github.com/uudashr/iface v1.5.0
	golang.org/x/exp v0.0.0-20260820142414-ca536658362e
	golang.org/x/tools v0.49.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-toolsmith/astcast v1.1.0 // indirect
	github.com/go-toolsmith/astcopy v1.1.0 // indirect
	github.com/go-toolsmith/astequal v1.2.0 // indirect
	github.com/go-toolsmith/astfmt v1.1.0 // indirect
	github.com/go-toolsmith/astp v1.1.0 // indirect
	github.com/go-toolsmith/strparse v1.1.0 // indirect
	github.com/go-toolsmith/typep v1.1.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/gostaticanalysis/analysisutil v0.7.1 // indirect
	github.com/gostaticanalysis/comment v1.5.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/quasilyte/go-ruleguard v0.4.5 // indirect
	github.com/quasilyte/gogrep v0.5.0 // indirect
	github.com/quasilyte/regex/syntax v0.0.0-20210819130434-b3f0c404a727 // indirect
	github.com/quasilyte/stdinfo v0.0.0-20220114132959-f7386bf02567 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/exp/typeparams v0.0.0-20251023183803-a4bb9ffd2546 // indirect
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	honnef.co/go/tools v0.7.0
)

tool (
	github.com/mazrean/kessoku/tools/apicompat
	github.com/mazrean/kessoku/tools/lint
)
