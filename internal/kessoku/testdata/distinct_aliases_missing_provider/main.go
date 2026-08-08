package main

// DBConnectionString and CacheConnectionString are distinct type aliases for string.
type DBConnectionString = string
type CacheConnectionString = string

// NewDBConnStr provides only a DBConnectionString.
// There is intentionally no provider for CacheConnectionString.
func NewDBConnStr() DBConnectionString {
	return "db://localhost"
}

// DBClient uses a DBConnectionString.
type DBClient struct{ ConnStr string }

// NewDBClient creates a DBClient.
func NewDBClient(s DBConnectionString) *DBClient {
	return &DBClient{ConnStr: s}
}

// CacheClient uses a CacheConnectionString.
type CacheClient struct{ ConnStr string }

// NewCacheClient creates a CacheClient — requires CacheConnectionString,
// which has no provider in the injector below.
func NewCacheClient(s CacheConnectionString) *CacheClient {
	return &CacheClient{ConnStr: s}
}

// App aggregates both clients.
type App struct {
	DB    *DBClient
	Cache *CacheClient
}

// NewApp creates an App.
func NewApp(db *DBClient, cache *CacheClient) *App {
	return &App{DB: db, Cache: cache}
}

func main() {}
