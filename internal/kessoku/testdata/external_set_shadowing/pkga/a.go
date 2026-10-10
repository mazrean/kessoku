package pkga

import "github.com/mazrean/kessoku"

// Config is provided by Set.
type Config struct {
	Name string
}

// DefaultName is referenced by a bare identifier inside Set.
const DefaultName = "a"

// Label is derived from Config.
type Label string

// Set declares locals named after this package and its imports.
var Set = kessoku.Set(
	kessoku.Provide(func() *Config {
		pkga := DefaultName
		return &Config{Name: pkga}
	}),
	kessoku.Provide(func(kessoku *Config) Label {
		return Label(kessoku.Name + DefaultName)
	}),
)
