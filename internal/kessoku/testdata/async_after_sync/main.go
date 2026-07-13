package main

// SyncService is a synchronous service with no dependencies.
type SyncService struct{}

func NewSyncService() *SyncService {
	return &SyncService{}
}

// AsyncService is an asynchronous service with no dependencies.
// It should run in a goroutine (not on the main goroutine).
type AsyncService struct{}

func NewAsyncService() *AsyncService {
	return &AsyncService{}
}

// App depends on both services.
type App struct {
	sync  *SyncService
	async *AsyncService
}

func NewApp(s *SyncService, a *AsyncService) *App {
	return &App{sync: s, async: a}
}

func main() {}
