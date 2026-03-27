// Package engine - typed event bus for decoupled inter-module communication.
// Supports publish/subscribe with topic-based routing and concurrent-safe operations.
package engine

import (
	"sync"
)

// EventType identifies the category of an event.
type EventType string

const (
	EventStreamOnline   EventType = "stream.online"
	EventStreamOffline  EventType = "stream.offline"
	EventViewerStarted  EventType = "viewer.started"
	EventViewerStopped  EventType = "viewer.stopped"
	EventViewerError    EventType = "viewer.error"
	EventAdDetected     EventType = "ad.detected"
	EventAdCompleted    EventType = "ad.completed"
	EventTokenExpired   EventType = "token.expired"
	EventProxyDead      EventType = "proxy.dead"
	EventMetricsUpdate  EventType = "metrics.update"
	EventEngineStarted  EventType = "engine.started"
	EventEngineStopped  EventType = "engine.stopped"
	EventConfigChanged  EventType = "config.changed"
)

// Event is a generic event with typed payload.
type Event struct {
	Type    EventType      `json:"type"`
	Payload map[string]any `json:"payload,omitempty"`
}

// EventHandler is a callback for handling events.
type EventHandler func(Event)

// EventBus provides publish/subscribe messaging between modules.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[EventType][]subscriberEntry
	allHandlers []EventHandler // handlers that receive ALL events
	nextID      int
}

type subscriberEntry struct {
	id      int
	handler EventHandler
}

// NewEventBus creates a new event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]subscriberEntry),
	}
}

// Subscribe registers a handler for a specific event type.
// Returns an unsubscribe function.
func (bus *EventBus) Subscribe(eventType EventType, handler EventHandler) func() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	id := bus.nextID
	bus.nextID++

	bus.subscribers[eventType] = append(bus.subscribers[eventType], subscriberEntry{
		id:      id,
		handler: handler,
	})

	return func() {
		bus.mu.Lock()
		defer bus.mu.Unlock()
		entries := bus.subscribers[eventType]
		for i, e := range entries {
			if e.id == id {
				bus.subscribers[eventType] = append(entries[:i], entries[i+1:]...)
				break
			}
		}
	}
}

// SubscribeAll registers a handler that receives every event.
func (bus *EventBus) SubscribeAll(handler EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.allHandlers = append(bus.allHandlers, handler)
}

// Publish sends an event to all matching subscribers (non-blocking).
func (bus *EventBus) Publish(event Event) {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// Topic-specific handlers
	for _, entry := range bus.subscribers[event.Type] {
		go entry.handler(event)
	}

	// Global handlers
	for _, handler := range bus.allHandlers {
		go handler(event)
	}
}

// PublishSimple is a convenience method for events without complex payloads.
func (bus *EventBus) PublishSimple(eventType EventType, kv ...string) {
	payload := make(map[string]any)
	for i := 0; i+1 < len(kv); i += 2 {
		payload[kv[i]] = kv[i+1]
	}
	bus.Publish(Event{Type: eventType, Payload: payload})
}
