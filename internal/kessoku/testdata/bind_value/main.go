package main

import "os"

type Service struct {
	writer *os.File
}

func NewService(w *os.File) *Service {
	return &Service{writer: w}
}

func main() {
}
