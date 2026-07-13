package main

import (
	"net/http"
	"sync/atomic"
)

// NewHTTPClientPtr creates an atomic pointer to an HTTP client.
func NewHTTPClientPtr() *atomic.Pointer[*http.Client] {
	p := &atomic.Pointer[*http.Client]{}
	client := &http.Client{}
	p.Store(&client)
	return p
}

func main() {}
