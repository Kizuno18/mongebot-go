package engine

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestEventBusSubscribePublish(t *testing.T) {
	bus := NewEventBus()

	var received atomic.Int32
	unsub := bus.Subscribe(EventStreamOnline, func(e Event) {
		received.Add(1)
	})

	bus.Publish(Event{Type: EventStreamOnline, Payload: map[string]any{"channel": "test"}})

	time.Sleep(50 * time.Millisecond)
	if received.Load() != 1 {
		t.Errorf("expected 1 event received, got %d", received.Load())
	}

	// Unsubscribe
	unsub()
	bus.Publish(Event{Type: EventStreamOnline})
	time.Sleep(50 * time.Millisecond)
	if received.Load() != 1 {
		t.Errorf("expected still 1 after unsub, got %d", received.Load())
	}
}

func TestEventBusTopicIsolation(t *testing.T) {
	bus := NewEventBus()

	var onlineCount, offlineCount atomic.Int32

	bus.Subscribe(EventStreamOnline, func(e Event) {
		onlineCount.Add(1)
	})
	bus.Subscribe(EventStreamOffline, func(e Event) {
		offlineCount.Add(1)
	})

	bus.Publish(Event{Type: EventStreamOnline})
	bus.Publish(Event{Type: EventStreamOnline})
	bus.Publish(Event{Type: EventStreamOffline})

	time.Sleep(50 * time.Millisecond)
	if onlineCount.Load() != 2 {
		t.Errorf("expected 2 online events, got %d", onlineCount.Load())
	}
	if offlineCount.Load() != 1 {
		t.Errorf("expected 1 offline event, got %d", offlineCount.Load())
	}
}

func TestEventBusSubscribeAll(t *testing.T) {
	bus := NewEventBus()

	var total atomic.Int32
	bus.SubscribeAll(func(e Event) {
		total.Add(1)
	})

	bus.Publish(Event{Type: EventStreamOnline})
	bus.Publish(Event{Type: EventViewerStarted})
	bus.Publish(Event{Type: EventAdDetected})

	time.Sleep(50 * time.Millisecond)
	if total.Load() != 3 {
		t.Errorf("expected 3 total events, got %d", total.Load())
	}
}

func TestPublishSimple(t *testing.T) {
	bus := NewEventBus()

	var receivedPayload map[string]any
	bus.Subscribe(EventTokenExpired, func(e Event) {
		receivedPayload = e.Payload
	})

	bus.PublishSimple(EventTokenExpired, "token", "abc123", "reason", "401")

	time.Sleep(50 * time.Millisecond)
	if receivedPayload["token"] != "abc123" {
		t.Errorf("expected token=abc123, got %v", receivedPayload["token"])
	}
	if receivedPayload["reason"] != "401" {
		t.Errorf("expected reason=401, got %v", receivedPayload["reason"])
	}
}
