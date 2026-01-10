package main

//go:generate go tool kessoku $GOFILE

import "github.com/mazrean/kessoku"

// Test Value with complex types (slice, map)
var config = map[string]string{
	"key1": "value1",
	"key2": "value2",
}

var tags = []string{"tag1", "tag2", "tag3"}

var _ = kessoku.Inject[*Service](
	"InitializeService",
	kessoku.Value(config),
	kessoku.Value(tags),
	kessoku.Provide(NewService),
)
