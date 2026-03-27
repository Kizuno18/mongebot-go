package engine

import (
	"context"
	"testing"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/platform"
)

func TestReconnectingViewer_DirectMode(t *testing.T) {
	// With reconnection disabled, should behave like a normal viewer
	cfg := &platform.ViewerConfig{
		Channel:  "test",
		Token:    "token123",
		DeviceID: "abcdef1234567890abcdef1234567890",
	}

	rv := NewReconnectingViewer(
		&mockPlatform{},
		cfg,
		ReconnectConfig{Enabled: false},
		testLogger(),
	)

	if rv.ID() != cfg.DeviceID {
		t.Errorf("expected ID=%s, got %s", cfg.DeviceID, rv.ID())
	}

	if rv.Status() != platform.ViewerIdle {
		t.Errorf("expected idle status, got %s", rv.Status())
	}
}

func TestReconnectingViewer_StopBeforeStart(t *testing.T) {
	cfg := &platform.ViewerConfig{
		Channel:  "test",
		Token:    "token",
		DeviceID: "abcdef1234567890abcdef1234567890",
	}

	rv := NewReconnectingViewer(
		&mockPlatform{},
		cfg,
		DefaultReconnectConfig(),
		testLogger(),
	)

	// Stop should not panic even if not started
	rv.Stop()

	if rv.Status() != platform.ViewerStopped {
		t.Errorf("expected stopped, got %s", rv.Status())
	}
}

func TestReconnectingViewer_ContextCancel(t *testing.T) {
	cfg := &platform.ViewerConfig{
		Channel:  "test",
		Token:    "token",
		DeviceID: "abcdef1234567890abcdef1234567890",
	}

	rv := NewReconnectingViewer(
		&mockPlatform{},
		cfg,
		ReconnectConfig{
			Enabled:     true,
			MaxAttempts: 3,
			BaseDelay:   10 * time.Millisecond,
			MaxDelay:    50 * time.Millisecond,
		},
		testLogger(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Should exit when context is cancelled
	err := rv.Start(ctx)
	if err != nil && err != context.DeadlineExceeded {
		// Mock viewer just blocks on ctx.Done(), so this should succeed
		t.Logf("start returned: %v (expected nil or deadline)", err)
	}
}

func TestDefaultReconnectConfig(t *testing.T) {
	cfg := DefaultReconnectConfig()

	if !cfg.Enabled {
		t.Error("should be enabled by default")
	}
	if cfg.MaxAttempts != 10 {
		t.Errorf("expected maxAttempts=10, got %d", cfg.MaxAttempts)
	}
	if cfg.BaseDelay != 3*time.Second {
		t.Errorf("expected baseDelay=3s, got %v", cfg.BaseDelay)
	}
	if !cfg.Jitter {
		t.Error("jitter should be enabled by default")
	}
}
