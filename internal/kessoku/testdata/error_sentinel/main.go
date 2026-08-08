package main

type App struct {
	name string
}

func NewApp(name string) *App {
	return &App{name: name}
}

func main() {}
