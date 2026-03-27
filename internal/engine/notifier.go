// Package engine - stream monitoring and notification system.
// Watches target channels for online/offline transitions and triggers actions.
package engine

import (
	"context"
	"log/slog"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/platform"
)

// StreamEvent represents a stream status change.
type StreamEvent struct {
	Channel   string                  `json:"channel"`
	Platform  string                  `json:"platform"`
	Status    platform.StreamStatus   `json:"status"`
	Metadata  *platform.StreamMetadata `json:"metadata,omitempty"`
	Timestamp time.Time               `json:"timestamp"`
}

// NotifyFunc is called when a stream status changes.
type NotifyFunc func(event StreamEvent)

// StreamMonitor watches channels for online/offline transitions.
type StreamMonitor struct {
	platform platform.Platform
	logger   *slog.Logger
	interval time.Duration
	channels map[string]platform.StreamStatus // last known status
	onEvent  NotifyFunc
}

// NewStreamMonitor creates a stream monitor.
func NewStreamMonitor(p platform.Platform, logger *slog.Logger, interval time.Duration) *StreamMonitor {
	return &StreamMonitor{
		platform: p,
		logger:   logger.With("component", "stream-monitor"),
		interval: interval,
		channels: make(map[string]platform.StreamStatus),
	}
}

// OnEvent sets the notification callback.
func (m *StreamMonitor) OnEvent(fn NotifyFunc) {
	m.onEvent = fn
}

// Watch starts monitoring a channel for status changes.
func (m *StreamMonitor) Watch(ctx context.Context, channel string) {
	m.logger.Info("monitoring channel", "channel", channel)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Check immediately
	m.check(ctx, channel)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.check(ctx, channel)
		}
	}
}

// WatchMultiple monitors multiple channels concurrently.
func (m *StreamMonitor) WatchMultiple(ctx context.Context, channels []string) {
	for _, ch := range channels {
		go m.Watch(ctx, ch)
	}
}

// check polls the current stream status and fires events on transitions.
func (m *StreamMonitor) check(ctx context.Context, channel string) {
	status, err := m.platform.GetStreamStatus(ctx, channel)
	if err != nil {
		m.logger.Debug("status check failed", "channel", channel, "error", err)
		return
	}

	prevStatus, exists := m.channels[channel]
	m.channels[channel] = status

	// Only fire events on state transitions (or first check)
	if !exists || prevStatus != status {
		event := StreamEvent{
			Channel:   channel,
			Platform:  m.platform.Name(),
			Status:    status,
			Timestamp: time.Now(),
		}

		// Fetch metadata if stream just went online
		if status == platform.StreamOnline {
			meta, err := m.platform.GetStreamMetadata(ctx, channel, "", "")
			if err == nil {
				event.Metadata = meta
			}
		}

		m.logger.Info("stream status changed",
			"channel", channel,
			"from", prevStatus.String(),
			"to", status.String(),
		)

		if m.onEvent != nil {
			m.onEvent(event)
		}
	}
}

// IsOnline returns whether a channel is currently known to be online.
func (m *StreamMonitor) IsOnline(channel string) bool {
	status, exists := m.channels[channel]
	return exists && status == platform.StreamOnline
}
