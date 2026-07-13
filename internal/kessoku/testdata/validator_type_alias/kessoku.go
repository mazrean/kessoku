//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// Verify that a validator (error-only provider) whose parameter is a type alias
// for a type provided by a regular provider connects to that provider rather than
// generating a spurious extra injector argument (QA-3).
var _ = kessoku.Inject[*Service](
	"GetService",
	kessoku.Provide(NewDB),
	kessoku.Provide(ValidateDB),
	kessoku.Provide(NewService),
)
