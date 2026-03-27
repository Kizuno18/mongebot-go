// Package engine - global rate limit detection and throttling.
// Monitors 429 responses across all viewers and applies coordinated cooldowns.
package engine

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// RateLimitTracker monitors rate limit events and applies global cooldowns.
type RateLimitTracker struct {
	mu           sync.RWMutex
	logger       *slog.Logger
	events       []rateLimitEvent
	cooldownEnd  time.Time
	totalHits    atomic.Int64
	windowSize   time.Duration // Time window for counting hits
	threshold    int           // Hits within window to trigger cooldown
	cooldownTime time.Duration // How long to cool down after threshold
}

type rateLimitEvent struct {
	timestamp time.Time
	source    string // "token:xxx" or "proxy:xxx"
	component string // "gql", "spade", "hls"
}

// NewRateLimitTracker creates a rate limit tracker.
func NewRateLimitTracker(logger *slog.Logger) *RateLimitTracker {
	return &RateLimitTracker{
		logger:       logger.With("component", "rate-limiter"),
		events:       make([]rateLimitEvent, 0),
		windowSize:   1 * time.Minute,
		threshold:    5,  // 5 rate limits per minute triggers global cooldown
		cooldownTime: 30 * time.Second,
	}
}

// SetThreshold configures when the global cooldown triggers.
func (rlt *RateLimitTracker) SetThreshold(hitsPerWindow int, window, cooldown time.Duration) {
	rlt.mu.Lock()
	defer rlt.mu.Unlock()
	rlt.threshold = hitsPerWindow
	rlt.windowSize = window
	rlt.cooldownTime = cooldown
}

// RecordHit records a rate limit (429) response.
// Returns true if a global cooldown was triggered.
func (rlt *RateLimitTracker) RecordHit(source, component string) bool {
	rlt.mu.Lock()
	defer rlt.mu.Unlock()

	rlt.totalHits.Add(1)
	now := time.Now()

	rlt.events = append(rlt.events, rateLimitEvent{
		timestamp: now,
		source:    source,
		component: component,
	})

	// Prune old events outside the window
	cutoff := now.Add(-rlt.windowSize)
	pruned := make([]rateLimitEvent, 0, len(rlt.events))
	for _, e := range rlt.events {
		if e.timestamp.After(cutoff) {
			pruned = append(pruned, e)
		}
	}
	rlt.events = pruned

	// Check if we've exceeded the threshold
	if len(rlt.events) >= rlt.threshold {
		rlt.cooldownEnd = now.Add(rlt.cooldownTime)
		rlt.logger.Warn("global rate limit cooldown triggered",
			"hits", len(rlt.events),
			"window", rlt.windowSize,
			"cooldown", rlt.cooldownTime,
		)
		// Clear events after triggering cooldown
		rlt.events = rlt.events[:0]
		return true
	}

	rlt.logger.Debug("rate limit hit recorded",
		"source", source,
		"component", component,
		"windowHits", len(rlt.events),
		"threshold", rlt.threshold,
	)

	return false
}

// ShouldThrottle returns true if we're currently in a global cooldown period.
func (rlt *RateLimitTracker) ShouldThrottle() bool {
	rlt.mu.RLock()
	defer rlt.mu.RUnlock()
	return time.Now().Before(rlt.cooldownEnd)
}

// CooldownRemaining returns how much time is left in the current cooldown.
func (rlt *RateLimitTracker) CooldownRemaining() time.Duration {
	rlt.mu.RLock()
	defer rlt.mu.RUnlock()
	remaining := time.Until(rlt.cooldownEnd)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// WaitForCooldown blocks until the cooldown period ends or context is cancelled.
func (rlt *RateLimitTracker) WaitForCooldown(done <-chan struct{}) {
	remaining := rlt.CooldownRemaining()
	if remaining <= 0 {
		return
	}

	rlt.logger.Info("waiting for rate limit cooldown", "remaining", remaining)
	select {
	case <-done:
	case <-time.After(remaining):
	}
}

// Stats returns current rate limit statistics.
func (rlt *RateLimitTracker) Stats() map[string]any {
	rlt.mu.RLock()
	defer rlt.mu.RUnlock()

	return map[string]any{
		"totalHits":    rlt.totalHits.Load(),
		"windowHits":   len(rlt.events),
		"threshold":    rlt.threshold,
		"inCooldown":   time.Now().Before(rlt.cooldownEnd),
		"cooldownLeft": rlt.CooldownRemaining().String(),
		"windowSize":   rlt.windowSize.String(),
	}
}

// MostFrequentSource returns which token or proxy is getting rate-limited most.
func (rlt *RateLimitTracker) MostFrequentSource() (string, int) {
	rlt.mu.RLock()
	defer rlt.mu.RUnlock()

	counts := make(map[string]int)
	for _, e := range rlt.events {
		counts[e.source]++
	}

	var maxSource string
	var maxCount int
	for source, count := range counts {
		if count > maxCount {
			maxSource = source
			maxCount = count
		}
	}
	return maxSource, maxCount
}
