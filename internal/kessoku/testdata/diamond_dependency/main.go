package main

type Database struct{}

func NewDatabase() *Database {
	return &Database{}
}

type ServiceA struct {
	db *Database
}

func NewServiceA(db *Database) *ServiceA {
	return &ServiceA{db: db}
}

type ServiceB struct {
	db *Database
}

func NewServiceB(db *Database) *ServiceB {
	return &ServiceB{db: db}
}

type App struct {
	serviceA *ServiceA
	serviceB *ServiceB
}

func NewApp(serviceA *ServiceA, serviceB *ServiceB) *App {
	return &App{serviceA: serviceA, serviceB: serviceB}
}

func main() {
}
