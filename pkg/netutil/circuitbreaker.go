// Package netutil - circuit breaker pattern for graceful degradation.
// Prevents cascading failures by temporarily disabling failing operations.
package netutil

import (
	"fmt"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // Normal operation
	CircuitOpen                        // Failing, blocking requests
	CircuitHalfOpen                    // Testing if service recovered
)

// String returns a human-readable circuit state.
func (s CircuitState) String() string {
	names := [...]string{"closed", "open", "half-open"}
	if int(s) < len(names) {
		return names[s]
	}
	return "unknown"
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu           sync.Mutex
	name         string
	state        CircuitState
	failureCount int
	successCount int
	threshold    int           // Failures before opening
	resetTimeout time.Duration // How long to wait before half-open
	lastFailure  time.Time
	halfOpenMax  int // Max test requests in half-open
}

// CircuitBreakerConfig configures a circuit breaker.
type CircuitBreakerConfig struct {
	Name         string
	Threshold    int           // Number of consecutive failures to trip (default: 5)
	ResetTimeout time.Duration // Time to wait before testing again (default: 30s)
	HalfOpenMax  int           // Successes needed to close from half-open (default: 2)
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.Threshold <= 0 {
		cfg.Threshold = 5
	}
	if cfg.ResetTimeout <= 0 {
		cfg.ResetTimeout = 30 * time.Second
	}
	if cfg.HalfOpenMax <= 0 {
		cfg.HalfOpenMax = 2
	}
	return &CircuitBreaker{
		name:         cfg.Name,
		state:        CircuitClosed,
		threshold:    cfg.Threshold,
		resetTimeout: cfg.ResetTimeout,
		halfOpenMax:  cfg.HalfOpenMax,
	}
}

// Execute runs fn if the circuit allows it.
// Returns ErrCircuitOpen if the circuit is tripped.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.AllowRequest() {
		return fmt.Errorf("circuit breaker %q is open", cb.name)
	}

	err := fn()
	if err != nil {
		cb.RecordFailure()
	} else {
		cb.RecordSuccess()
	}
	return err
}

// AllowRequest checks if a request should be allowed through.
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if reset timeout has elapsed
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = CircuitHalfOpen
			cb.successCount = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return true
	}
}

// RecordSuccess records a successful operation.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0

	if cb.state == CircuitHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.halfOpenMax {
			cb.state = CircuitClosed
		}
	}
}

// RecordFailure records a failed operation.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailure = time.Now()

	if cb.state == CircuitHalfOpen {
		// Any failure in half-open immediately re-opens
		cb.state = CircuitOpen
		return
	}

	if cb.failureCount >= cb.threshold {
		cb.state = CircuitOpen
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Reset manually resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = CircuitClosed
	cb.failureCount = 0
	cb.successCount = 0
}

// Stats returns current circuit breaker statistics.
func (cb *CircuitBreaker) Stats() map[string]any {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return map[string]any{
		"name":         cb.name,
		"state":        cb.state.String(),
		"failures":     cb.failureCount,
		"threshold":    cb.threshold,
		"lastFailure":  cb.lastFailure,
		"resetTimeout": cb.resetTimeout.String(),
	}
}
