package main

import "context"

// Ctx is a user-defined alias of context.Context.
type Ctx = context.Context

type Config struct {
	DSN string
}

func NewConfig(ctx Ctx) (*Config, error) {
	return &Config{DSN: "test-dsn"}, nil
}

type Cache struct{}

func NewCache(ctx Ctx) (*Cache, error) {
	return &Cache{}, nil
}

type Service struct {
	config *Config
	cache  *Cache
}

func NewService(config *Config, cache *Cache) *Service {
	return &Service{config: config, cache: cache}
}

func main() {
}
