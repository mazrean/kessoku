package main

import "fmt"

func main() {
	db := InitializeDatabase()
	fmt.Printf("Database connected to %s:%d (debug: %v)\n", db.host, db.port, db.debug)
}
