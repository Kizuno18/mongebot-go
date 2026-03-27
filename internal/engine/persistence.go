// Package engine - metrics persistence to SQLite at regular intervals.
package engine

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/storage"
)

// MetricsPersister periodically saves engine metrics snapshots to SQLite.
type MetricsPersister struct {
	db        *storage.DB
	engine    *Engine
	logger    *slog.Logger
	interval  time.Duration
	sessionID atomic.Int64
}

// NewMetricsPersister creates a metrics persister.
func NewMetricsPersister(db *storage.DB, eng *Engine, logger *slog.Logger, interval time.Duration) *MetricsPersister {
	return &MetricsPersister{
		db:       db,
		engine:   eng,
		logger:   logger.With("component", "metrics-persister"),
		interval: interval,
	}
}

// StartSession creates a new session record and begins periodic snapshots.
func (mp *MetricsPersister) StartSession(ctx context.Context, profileID, channel, platform string) error {
	if mp.db == nil {
		return nil // No database, skip persistence
	}

	sessionID, err := mp.db.InsertSession(ctx, profileID, channel, platform)
	if err != nil {
		mp.logger.Error("failed to create session", "error", err)
		return err
	}

	mp.sessionID.Store(sessionID)
	mp.logger.Info("session started", "sessionId", sessionID, "channel", channel)

	go mp.snapshotLoop(ctx)
	return nil
}

// EndSession marks the current session as finished with final metrics.
func (mp *MetricsPersister) EndSession(ctx context.Context, reason string) {
	if mp.db == nil {
		return
	}

	sessionID := mp.sessionID.Load()
	if sessionID == 0 {
		return
	}

	m := mp.engine.Metrics()
	err := mp.db.EndSession(ctx, sessionID, reason,
		m.ActiveViewers,
		int(m.SegmentsFetched),
		m.BytesReceived,
		int(m.AdsWatched),
		int(m.HeartbeatsSent),
	)
	if err != nil {
		mp.logger.Error("failed to end session", "error", err)
	} else {
		mp.logger.Info("session ended", "sessionId", sessionID, "reason", reason)
	}
}

// snapshotLoop periodically saves metrics snapshots.
func (mp *MetricsPersister) snapshotLoop(ctx context.Context) {
	ticker := time.NewTicker(mp.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mp.saveSnapshot(ctx)
		}
	}
}

// saveSnapshot records a single metrics snapshot.
func (mp *MetricsPersister) saveSnapshot(ctx context.Context) {
	sessionID := mp.sessionID.Load()
	if sessionID == 0 {
		return
	}

	m := mp.engine.Metrics()
	err := mp.db.InsertMetricsSnapshot(ctx, sessionID,
		m.ActiveViewers,
		m.TotalWorkers,
		int(m.SegmentsFetched),
		m.BytesReceived,
		int(m.HeartbeatsSent),
		int(m.AdsWatched),
	)
	if err != nil {
		mp.logger.Debug("snapshot save failed", "error", err)
	}
}
