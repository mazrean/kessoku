package kessoku_test

import (
	"fmt"

	"github.com/mazrean/kessoku"
)

// ExampleInject demonstrates basic dependency injection setup.
func ExampleInject() {
	// Define your service constructors
	NewConfig := func() *Config { return &Config{} }
	NewDatabase := func(cfg *Config) (*Database, error) { return &Database{}, nil }
	NewUserService := func(db *Database) *UserService { return &UserService{} }
	NewApp := func(svc *UserService) *App { return &App{} }

	// Declare the dependency injection
	var _ = kessoku.Inject[*App](
		"InitializeApp",
		kessoku.Provide(NewConfig),
		kessoku.Provide(NewDatabase),
		kessoku.Provide(NewUserService),
		kessoku.Provide(NewApp),
	)

	// This generates a function:
	// func InitializeApp() (*App, error)
	fmt.Println("Generated InitializeApp function")
	// Output: Generated InitializeApp function
}

// ExampleAsync demonstrates parallel execution of slow providers.
func ExampleAsync() {
	// Slow service constructors
	NewDatabaseService := func() (*DatabaseService, error) { return &DatabaseService{}, nil }
	NewCacheService := func() *CacheService { return &CacheService{} }
	NewApp := func(db *DatabaseService, cache *CacheService) *App { return &App{} }

	// Declare async providers for parallel execution
	var _ = kessoku.Inject[*App](
		"InitializeApp",
		kessoku.Async(kessoku.Provide(NewDatabaseService)), // runs in parallel
		kessoku.Async(kessoku.Provide(NewCacheService)),    // runs in parallel
		kessoku.Provide(NewApp),                            // waits for both
	)

	// This generates a function:
	// func InitializeApp(ctx context.Context) (*App, error)
	fmt.Println("Generated InitializeApp with async providers")
	// Output: Generated InitializeApp with async providers
}

// ExampleBind demonstrates interface binding to concrete implementations.
func ExampleBind() {
	// Interface and implementation
	type UserRepository interface {
		GetUser(id string) (*User, error)
	}
	type DatabaseUserRepo struct{}

	NewDatabaseUserRepo := func() *DatabaseUserRepo { return &DatabaseUserRepo{} }
	NewUserService := func(repo UserRepository) *UserService { return &UserService{} }

	// Bind interface to implementation
	var _ = kessoku.Inject[*UserService](
		"InitializeUserService",
		kessoku.Bind[UserRepository](kessoku.Provide(NewDatabaseUserRepo)),
		kessoku.Provide(NewUserService),
	)

	fmt.Println("Generated InitializeUserService with interface binding")
	// Output: Generated InitializeUserService with interface binding
}

// ExampleValue demonstrates providing constant values.
func ExampleValue() {
	// Service constructor that uses configuration values
	NewServer := func(port int, dbURL string, debug bool) *Server {
		return &Server{Port: port, DatabaseURL: dbURL, Debug: debug}
	}

	// Use Value in Inject to provide constant values
	var _ = kessoku.Inject[*Server](
		"InitializeServer",
		kessoku.Value(8080),             // provides int
		kessoku.Value("localhost:5432"), // provides string
		kessoku.Value(true),             // provides bool
		kessoku.Provide(NewServer),
	)

	// Show value provider types
	fmt.Println("Generated InitializeServer function using Value")
	// Output: Generated InitializeServer function using Value
}

// ExampleSet demonstrates grouping providers into reusable sets.
func ExampleSet() {
	// Database-related providers
	NewConnection := func() *Connection { return &Connection{} }
	NewUserRepo := func(conn *Connection) *UserRepository { return &UserRepository{} }
	NewOrderRepo := func(conn *Connection) *OrderRepository { return &OrderRepository{} }

	// Create a reusable set
	var DatabaseSet = kessoku.Set(
		kessoku.Provide(NewConnection),
		kessoku.Provide(NewUserRepo),
		kessoku.Provide(NewOrderRepo),
	)

	// Use the set in multiple injectors
	var _ = kessoku.Inject[*UserService](
		"InitializeUserService",
		DatabaseSet,
		kessoku.Provide(func(repo *UserRepository) *UserService { return &UserService{} }),
	)

	fmt.Println("Generated InitializeUserService using DatabaseSet")
	// Output: Generated InitializeUserService using DatabaseSet
}

// Example types for documentation
type (
	Config          struct{}
	Database        struct{}
	Logger          struct{}
	UserService     struct{}
	App             struct{}
	DatabaseService struct{}
	CacheService    struct{}
	User            struct{}
	Connection      struct{}
	UserRepository  struct{}
	OrderRepository struct{}
	Server          struct {
		DatabaseURL string
		Port        int
		Debug       bool
	}
)
