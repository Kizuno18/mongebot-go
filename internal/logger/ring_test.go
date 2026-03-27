package logger

import (
	"testing"
	"time"
)

func TestRingBufferPushAndAll(t *testing.T) {
	rb := NewRingBuffer(5)

	for i := range 3 {
		rb.Push(LogEntry{Message: string(rune('A' + i)), Timestamp: time.Now()})
	}

	entries := rb.All()
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestRingBufferOverflow(t *testing.T) {
	rb := NewRingBuffer(3)

	for i := range 5 {
		rb.Push(LogEntry{Message: string(rune('A' + i)), Timestamp: time.Now()})
	}

	entries := rb.All()
	if len(entries) != 3 {
		t.Errorf("expected 3 entries (buffer capacity), got %d", len(entries))
	}

	// Should have C, D, E (oldest two dropped)
	if entries[0].Message != "C" {
		t.Errorf("expected first entry 'C', got %q", entries[0].Message)
	}
	if entries[2].Message != "E" {
		t.Errorf("expected last entry 'E', got %q", entries[2].Message)
	}
}

func TestRingBufferSubscribe(t *testing.T) {
	rb := NewRingBuffer(10)

	id, ch := rb.Subscribe(5)
	defer rb.Unsubscribe(id)

	entry := LogEntry{Message: "test", Timestamp: time.Now()}
	rb.Push(entry)

	select {
	case received := <-ch:
		if received.Message != "test" {
			t.Errorf("expected message 'test', got %q", received.Message)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for subscriber notification")
	}
}
