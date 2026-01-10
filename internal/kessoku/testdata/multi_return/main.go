package main

type DatabaseConfig struct {
	DSN string
}

type CacheConfig struct {
	URL string
}

// NewConfigs returns multiple values
func NewConfigs() (*DatabaseConfig, *CacheConfig) {
	return &DatabaseConfig{DSN: "test-dsn"}, &CacheConfig{URL: "redis://localhost"}
}

type Service struct {
	dbConfig    *DatabaseConfig
	cacheConfig *CacheConfig
}

func NewService(dbConfig *DatabaseConfig, cacheConfig *CacheConfig) *Service {
	return &Service{dbConfig: dbConfig, cacheConfig: cacheConfig}
}

func main() {
}
