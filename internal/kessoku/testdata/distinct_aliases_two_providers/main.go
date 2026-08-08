package main

// DBConnectionString and CacheConnectionString are distinct type aliases for
// string. In Go's type system they are identical to string, but kessoku treats
// each alias as a separate dependency key so that the correct connection string
// is routed to each client constructor.
type DBConnectionString = string
type CacheConnectionString = string

// NewDBConnStr provides the database connection string.
func NewDBConnStr() DBConnectionString {
	return "db://localhost"
}

// NewCacheConnStr provides the cache connection string.
func NewCacheConnStr() CacheConnectionString {
	return "cache://localhost"
}

// DBClient uses a DBConnectionString.
type DBClient struct{ ConnStr string }

// NewDBClient creates a DBClient.
func NewDBClient(s DBConnectionString) *DBClient {
	return &DBClient{ConnStr: s}
}

// CacheClient uses a CacheConnectionString.
type CacheClient struct{ ConnStr string }

// NewCacheClient creates a CacheClient.
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
