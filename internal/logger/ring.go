// Package logger - ring buffer for storing recent log entries for the UI log viewer.
package logger

import (
	"sync"
	"time"
)

// LogEntry represents a single log entry stored in the ring buffer.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Component string    `json:"component,omitempty"`
	Worker    string    `json:"worker,omitempty"`
	Fields    map[string]any `json:"fields,omitempty"`
}

// RingBuffer is a fixed-size circular buffer for log entries.
type RingBuffer struct {
	mu      sync.RWMutex
	entries []LogEntry
	head    int
	size    int
	cap     int

	// subscribers receive new entries in real-time
	subMu       sync.RWMutex
	subscribers map[int]chan LogEntry
	nextSubID   int
}

// NewRingBuffer creates a ring buffer with the given capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		entries:     make([]LogEntry, capacity),
		cap:         capacity,
		subscribers: make(map[int]chan LogEntry),
	}
}

// Push adds a new log entry to the buffer and notifies subscribers.
func (rb *RingBuffer) Push(entry LogEntry) {
	rb.mu.Lock()
	rb.entries[rb.head] = entry
	rb.head = (rb.head + 1) % rb.cap
	if rb.size < rb.cap {
		rb.size++
	}
	rb.mu.Unlock()

	// Notify subscribers (non-blocking)
	rb.subMu.RLock()
	for _, ch := range rb.subscribers {
		select {
		case ch <- entry:
		default:
			// Drop if subscriber is slow
		}
	}
	rb.subMu.RUnlock()
}

// All returns all entries in chronological order.
func (rb *RingBuffer) All() []LogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	result := make([]LogEntry, rb.size)
	if rb.size < rb.cap {
		copy(result, rb.entries[:rb.size])
	} else {
		start := rb.head
		copy(result, rb.entries[start:])
		copy(result[rb.cap-start:], rb.entries[:start])
	}
	return result
}

// Subscribe returns a channel that receives new log entries in real-time.
// Call Unsubscribe with the returned ID when done.
func (rb *RingBuffer) Subscribe(bufferSize int) (int, <-chan LogEntry) {
	rb.subMu.Lock()
	defer rb.subMu.Unlock()

	id := rb.nextSubID
	rb.nextSubID++
	ch := make(chan LogEntry, bufferSize)
	rb.subscribers[id] = ch
	return id, ch
}

// Unsubscribe removes a subscriber by ID.
func (rb *RingBuffer) Unsubscribe(id int) {
	rb.subMu.Lock()
	defer rb.subMu.Unlock()

	if ch, ok := rb.subscribers[id]; ok {
		close(ch)
		delete(rb.subscribers, id)
	}
}
