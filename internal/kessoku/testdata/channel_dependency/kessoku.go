//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

// EventChan is a channel for events.
type EventChan chan string

// NewEventChannel creates an event channel.
func NewEventChannel() EventChan {
	return make(chan string, 100)
}

// EventHandler handles events from a channel.
type EventHandler struct {
	events EventChan
}

// NewEventHandler creates a new event handler.
func NewEventHandler(events EventChan) *EventHandler {
	return &EventHandler{events: events}
}

// Test channel type as dependency
var _ = kessoku.Inject[*EventHandler](
	"InitializeEventHandler",
	kessoku.Provide(NewEventChannel),
	kessoku.Provide(NewEventHandler),
)
