package main

import "context"

type APIKey string

type Config struct {
	key APIKey
}

// NewConfig is declared before any ctx-requiring provider, so declaration-relative
// ordering alone would place apikey before ctx in the injector signature.
func NewConfig(key APIKey) *Config {
	return &Config{key: key}
}

type Service struct {
	config *Config
}

func NewService(ctx context.Context, config *Config) *Service {
	_ = ctx
	return &Service{config: config}
}

func main() {}
