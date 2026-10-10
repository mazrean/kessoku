package pkga

import "github.com/mazrean/kessoku"

// Base is embedded into an anonymous struct.
type Base struct {
	ID int
}

// Result is provided by Set.
type Result struct {
	Sum int
}

// Set declares types with unexported fields inside a closure.
var Set = kessoku.Set(
	kessoku.Provide(func() *Result {
		type pair struct{ a, b int }
		p := pair{1, 2}
		anon := struct {
			Base
			n int
		}{Base{ID: 3}, 4}
		return &Result{Sum: p.a + p.b + anon.ID + anon.n}
	}),
)
