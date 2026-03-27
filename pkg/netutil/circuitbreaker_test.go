package netutil

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:      "test",
		Threshold: 3,
	})

	if cb.State() != CircuitClosed {
		t.Error("initial state should be closed")
	}

	// Successful operations should keep it closed
	for range 10 {
		err := cb.Execute(func() error { return nil })
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}

	if cb.State() != CircuitClosed {
		t.Error("should remain closed after successes")
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:         "test",
		Threshold:    3,
		ResetTimeout: 1 * time.Second,
	})

	fail := errors.New("fail")

	// 3 failures should trip the circuit
	for range 3 {
		cb.Execute(func() error { return fail })
	}

	if cb.State() != CircuitOpen {
		t.Errorf("expected open after %d failures, got %s", 3, cb.State())
	}

	// Should block requests
	err := cb.Execute(func() error { return nil })
	if err == nil {
		t.Error("expected error when circuit is open")
	}
}

func TestCircuitBreaker_HalfOpenRecovery(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:         "test",
		Threshold:    2,
		ResetTimeout: 50 * time.Millisecond,
		HalfOpenMax:  2,
	})

	// Trip the circuit
	cb.Execute(func() error { return errors.New("fail") })
	cb.Execute(func() error { return errors.New("fail") })

	if cb.State() != CircuitOpen {
		t.Fatal("should be open")
	}

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open on next request
	if !cb.AllowRequest() {
		t.Fatal("should allow request in half-open")
	}

	// Two successes should close the circuit
	cb.RecordSuccess()
	cb.RecordSuccess()

	if cb.State() != CircuitClosed {
		t.Errorf("expected closed after recovery, got %s", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:         "test",
		Threshold:    2,
		ResetTimeout: 50 * time.Millisecond,
	})

	// Trip
	cb.Execute(func() error { return errors.New("fail") })
	cb.Execute(func() error { return errors.New("fail") })

	time.Sleep(60 * time.Millisecond)
	cb.AllowRequest() // Transitions to half-open

	// Failure in half-open immediately re-opens
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Errorf("expected re-opened, got %s", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:      "test",
		Threshold: 1,
	})

	cb.Execute(func() error { return errors.New("fail") })
	if cb.State() != CircuitOpen {
		t.Fatal("should be open")
	}

	cb.Reset()
	if cb.State() != CircuitClosed {
		t.Error("should be closed after reset")
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:      "myservice",
		Threshold: 5,
	})

	stats := cb.Stats()
	if stats["name"] != "myservice" {
		t.Errorf("expected name=myservice, got %v", stats["name"])
	}
	if stats["state"] != "closed" {
		t.Errorf("expected state=closed, got %v", stats["state"])
	}
}
