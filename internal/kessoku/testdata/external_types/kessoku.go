//go:generate go tool kessoku $GOFILE

package main

import (
	"net/http"
	"time"

	"github.com/mazrean/kessoku"
)

// NewTimeout creates a timeout duration.
func NewTimeout() time.Duration {
	return 30 * time.Second
}

// NewHTTPClient creates an HTTP client with timeout.
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
	}
}

// APIClient wraps HTTP client.
type APIClient struct {
	client *http.Client
}

// NewAPIClient creates an API client.
func NewAPIClient(client *http.Client) *APIClient {
	return &APIClient{client: client}
}

// Test using standard library types
var _ = kessoku.Inject[*APIClient](
	"InitializeAPIClient",
	kessoku.Provide(NewTimeout),
	kessoku.Provide(NewHTTPClient),
	kessoku.Provide(NewAPIClient),
)
