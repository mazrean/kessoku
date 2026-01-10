package main

// Custom named types
type APIKey string
type DatabaseURL string
type Timeout int

func NewAPIKey() APIKey {
	return APIKey("secret-api-key")
}

func NewDatabaseURL() DatabaseURL {
	return DatabaseURL("postgres://localhost/db")
}

type Service struct {
	apiKey      APIKey
	databaseURL DatabaseURL
}

func NewService(apiKey APIKey, dbURL DatabaseURL) *Service {
	return &Service{apiKey: apiKey, databaseURL: dbURL}
}

func main() {
}
