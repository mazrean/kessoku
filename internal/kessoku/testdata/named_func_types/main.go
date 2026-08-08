package main

// Named func types — these are ordinary business types, not wire-style cleanups.
// A provider returning ShutdownFunc or CommitFunc must NOT be rejected.
type ShutdownFunc func()

type CommitFunc func() error

type App struct {
	shutdown ShutdownFunc
	commit   CommitFunc
}

func NewShutdown() ShutdownFunc {
	return func() {}
}

func NewCommit() CommitFunc {
	return func() error { return nil }
}

func NewApp(s ShutdownFunc, c CommitFunc) *App {
	return &App{shutdown: s, commit: c}
}

func main() {}
