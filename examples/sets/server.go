package main

import "fmt"

// Server represents an HTTP server
type Server struct {
	config *Config
}

// NewServer creates a new server instance
func NewServer(config *Config) *Server {
	return &Server{
		config: config,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	fmt.Printf("Server starting on port %s\n", s.config.Port)
	return nil
}

// Stop stops the HTTP server
func (s *Server) Stop() error {
	fmt.Println("Server stopped")
	return nil
}