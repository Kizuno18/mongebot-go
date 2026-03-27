package engine

import (
	"testing"
	"time"
)

func TestRateLimitTracker_NoTrigger(t *testing.T) {
	rlt := NewRateLimitTracker(testLogger())

	// Single hit should not trigger cooldown
	triggered := rlt.RecordHit("token:abc", "gql")
	if triggered {
		t.Error("single hit should not trigger cooldown")
	}

	if rlt.ShouldThrottle() {
		t.Error("should not throttle after single hit")
	}
}

func TestRateLimitTracker_TriggerCooldown(t *testing.T) {
	rlt := NewRateLimitTracker(testLogger())
	rlt.SetThreshold(3, 1*time.Minute, 100*time.Millisecond)

	// 3 hits should trigger cooldown
	rlt.RecordHit("token:1", "gql")
	rlt.RecordHit("token:2", "spade")
	triggered := rlt.RecordHit("token:3", "hls")

	if !triggered {
		t.Error("3rd hit should trigger cooldown")
	}

	if !rlt.ShouldThrottle() {
		t.Error("should be throttling after cooldown triggered")
	}

	// Wait for cooldown to expire
	time.Sleep(150 * time.Millisecond)

	if rlt.ShouldThrottle() {
		t.Error("should no longer throttle after cooldown expired")
	}
}

func TestRateLimitTracker_MostFrequentSource(t *testing.T) {
	rlt := NewRateLimitTracker(testLogger())

	rlt.RecordHit("token:abc", "gql")
	rlt.RecordHit("token:abc", "spade")
	rlt.RecordHit("token:xyz", "gql")
	rlt.RecordHit("token:abc", "hls")

	source, count := rlt.MostFrequentSource()
	if source != "token:abc" {
		t.Errorf("expected most frequent=token:abc, got %s", source)
	}
	if count != 3 {
		t.Errorf("expected count=3, got %d", count)
	}
}

func TestRateLimitTracker_Stats(t *testing.T) {
	rlt := NewRateLimitTracker(testLogger())
	rlt.RecordHit("proxy:1", "gql")

	stats := rlt.Stats()
	if stats["totalHits"].(int64) != 1 {
		t.Errorf("expected totalHits=1, got %v", stats["totalHits"])
	}
	if stats["inCooldown"].(bool) {
		t.Error("should not be in cooldown")
	}
}
