package main

import (
	"fmt"
	"time"

	"github.com/mazrean/kessoku"
)

const (
	dbSleepDuration    = 200 * time.Millisecond
	cacheSleepDuration = 150 * time.Millisecond
)

type DB struct{ Addr string }
type Cache struct{ Addr string }

func SlowDB() *DB {
	time.Sleep(dbSleepDuration)
	return &DB{Addr: "db:5432"}
}

func SlowCache() *Cache {
	time.Sleep(cacheSleepDuration)
	return &Cache{Addr: "cache:6379"}
}

//go:generate go tool kessoku $GOFILE

var _ = kessoku.Inject[string]("InitApp",
	kessoku.Async(kessoku.Provide(SlowDB)),
	kessoku.Async(kessoku.Provide(SlowCache)),
	kessoku.Provide(func(db *DB, cache *Cache) string {
		return fmt.Sprintf("App running with %s and %s", db.Addr, cache.Addr)
	}),
)
