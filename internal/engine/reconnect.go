// Package engine - viewer auto-reconnection with exponential backoff.
// Uses the FSM to manage state transitions during reconnection.
package engine

import (
	"context"
	"log/slog"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/platform"
	"github.com/Kizuno18/mongebot-go/pkg/netutil"
)

// ReconnectConfig controls auto-reconnection behavior.
type ReconnectConfig struct {
	Enabled     bool          `json:"enabled"`
	MaxAttempts int           `json:"maxAttempts"` // 0 = unlimited
	BaseDelay   time.Duration `json:"baseDelay"`
	MaxDelay    time.Duration `json:"maxDelay"`
	Jitter      bool          `json:"jitter"`
}

// DefaultReconnectConfig returns sensible reconnection defaults.
func DefaultReconnectConfig() ReconnectConfig {
	return ReconnectConfig{
		Enabled:     true,
		MaxAttempts: 10,
		BaseDelay:   3 * time.Second,
		MaxDelay:    60 * time.Second,
		Jitter:      true,
	}
}

// ReconnectingViewer wraps a platform.Viewer with auto-reconnection logic.
type ReconnectingViewer struct {
	inner    platform.Viewer
	platform platform.Platform
	config   *platform.ViewerConfig
	reconfig ReconnectConfig
	fsm      *ViewerFSM
	logger   *slog.Logger
}

// NewReconnectingViewer wraps a viewer with auto-reconnection.
func NewReconnectingViewer(
	p platform.Platform,
	cfg *platform.ViewerConfig,
	rcfg ReconnectConfig,
	logger *slog.Logger,
) *ReconnectingViewer {
	return &ReconnectingViewer{
		platform: p,
		config:   cfg,
		reconfig: rcfg,
		fsm:      NewViewerFSM(),
		logger:   logger.With("viewer", cfg.DeviceID[:8], "component", "reconnect"),
	}
}

// Start begins the viewer with auto-reconnection on failure.
func (rv *ReconnectingViewer) Start(ctx context.Context) error {
	if !rv.reconfig.Enabled {
		// No reconnection — run viewer directly
		viewer, err := rv.platform.Connect(ctx, rv.config)
		if err != nil {
			return err
		}
		rv.inner = viewer
		return viewer.Start(ctx)
	}

	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Transition: idle/error → connecting
		rv.fsm.ForceState(platform.ViewerConnecting)

		rv.logger.Info("connecting", "attempt", attempt+1)

		viewer, err := rv.platform.Connect(ctx, rv.config)
		if err != nil {
			rv.fsm.ForceState(platform.ViewerError)
			rv.logger.Warn("connect failed", "error", err, "attempt", attempt+1)

			if rv.shouldGiveUp(attempt) {
				return err
			}
			rv.waitBackoff(ctx, attempt)
			attempt++
			continue
		}

		rv.inner = viewer
		rv.fsm.ForceState(platform.ViewerActive)
		attempt = 0 // Reset on successful connection

		// Run the viewer (blocks until error/disconnect)
		viewerErr := viewer.Start(ctx)

		if ctx.Err() != nil {
			// Context cancelled — intentional shutdown
			rv.fsm.ForceState(platform.ViewerStopped)
			return nil
		}

		// Viewer died — attempt reconnection
		rv.fsm.ForceState(platform.ViewerReconnecting)
		rv.logger.Warn("viewer disconnected, will reconnect",
			"error", viewerErr,
			"attempt", attempt+1,
		)

		if rv.shouldGiveUp(attempt) {
			rv.fsm.ForceState(platform.ViewerError)
			return viewerErr
		}

		rv.waitBackoff(ctx, attempt)
		attempt++
	}
}

// Stop stops the inner viewer.
func (rv *ReconnectingViewer) Stop() {
	rv.fsm.ForceState(platform.ViewerStopped)
	if rv.inner != nil {
		rv.inner.Stop()
	}
}

// ID returns the viewer ID.
func (rv *ReconnectingViewer) ID() string {
	return rv.config.DeviceID
}

// Status returns the FSM state.
func (rv *ReconnectingViewer) Status() platform.ViewerStatus {
	return rv.fsm.State()
}

// Metrics returns metrics from the inner viewer (if connected).
func (rv *ReconnectingViewer) Metrics() *platform.ViewerMetrics {
	if rv.inner != nil {
		return rv.inner.Metrics()
	}
	return &platform.ViewerMetrics{}
}

// shouldGiveUp returns true if max attempts have been exceeded.
func (rv *ReconnectingViewer) shouldGiveUp(attempt int) bool {
	return rv.reconfig.MaxAttempts > 0 && attempt >= rv.reconfig.MaxAttempts
}

// waitBackoff waits with exponential backoff before the next reconnection attempt.
func (rv *ReconnectingViewer) waitBackoff(ctx context.Context, attempt int) {
	delay := netutil.DefaultRetry().BaseDelay
	if rv.reconfig.BaseDelay > 0 {
		delay = rv.reconfig.BaseDelay
	}

	// Exponential backoff: delay * 2^attempt, capped at maxDelay
	for range attempt {
		delay *= 2
		if rv.reconfig.MaxDelay > 0 && delay > rv.reconfig.MaxDelay {
			delay = rv.reconfig.MaxDelay
			break
		}
	}

	// Add jitter
	if rv.reconfig.Jitter {
		delay = netutil.RandomDuration(delay/2, delay)
	}

	rv.logger.Debug("backoff wait", "delay", delay, "attempt", attempt+1)

	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}
}
