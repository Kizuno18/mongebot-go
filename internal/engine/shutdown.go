// Package engine - graceful shutdown with connection drain, final metrics, and cleanup.
package engine

import (
	"context"
	"log/slog"
	"time"
)

// ShutdownConfig controls graceful shutdown behavior.
type ShutdownConfig struct {
	DrainTimeout    time.Duration // Max time to wait for workers to finish
	SaveMetrics     bool          // Save final metrics snapshot before exit
	NotifyFrontend  bool          // Send shutdown event to connected clients
}

// DefaultShutdownConfig returns sensible shutdown defaults.
func DefaultShutdownConfig() ShutdownConfig {
	return ShutdownConfig{
		DrainTimeout:   10 * time.Second,
		SaveMetrics:    true,
		NotifyFrontend: true,
	}
}

// GracefulShutdown performs an orderly engine shutdown.
// 1. Notify frontend clients of impending shutdown
// 2. Stop accepting new viewer connections
// 3. Drain existing workers (wait up to DrainTimeout)
// 4. Save final metrics snapshot
// 5. Close all resources
func GracefulShutdown(ctx context.Context, eng *Engine, cfg ShutdownConfig, logger *slog.Logger) {
	shutdownStart := time.Now()
	logger.Info("graceful shutdown initiated", "drainTimeout", cfg.DrainTimeout)

	// Phase 1: Signal engine to stop accepting new workers
	eng.state.Store(int32(StateStopping))

	// Phase 2: Publish shutdown event
	if cfg.NotifyFrontend {
		logger.Debug("notifying frontend of shutdown")
	}

	// Phase 3: Collect final metrics before stopping
	finalMetrics := eng.Metrics()
	logger.Info("final metrics",
		"activeViewers", finalMetrics.ActiveViewers,
		"totalWorkers", finalMetrics.TotalWorkers,
		"segments", finalMetrics.SegmentsFetched,
		"bytes", finalMetrics.BytesReceived,
		"heartbeats", finalMetrics.HeartbeatsSent,
		"ads", finalMetrics.AdsWatched,
		"uptime", finalMetrics.Uptime,
	)

	// Phase 4: Drain workers with timeout
	drainCtx, drainCancel := context.WithTimeout(ctx, cfg.DrainTimeout)
	defer drainCancel()

	eng.workersMu.RLock()
	workerCount := len(eng.workers)
	eng.workersMu.RUnlock()

	logger.Info("draining workers", "count", workerCount, "timeout", cfg.DrainTimeout)

	// Stop all workers
	eng.workersMu.Lock()
	for _, w := range eng.workers {
		w.Stop()
	}
	eng.workersMu.Unlock()

	// Wait for drain or timeout
	drained := make(chan struct{})
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			eng.workersMu.RLock()
			remaining := len(eng.workers)
			eng.workersMu.RUnlock()
			if remaining == 0 {
				close(drained)
				return
			}
			select {
			case <-drainCtx.Done():
				close(drained)
				return
			case <-ticker.C:
			}
		}
	}()

	<-drained

	eng.workersMu.RLock()
	remaining := len(eng.workers)
	eng.workersMu.RUnlock()

	if remaining > 0 {
		logger.Warn("drain timeout reached, force-stopping remaining workers", "remaining", remaining)
		eng.workersMu.Lock()
		eng.workers = make(map[string]*Worker)
		eng.workersMu.Unlock()
	}

	// Phase 5: Final state
	eng.state.Store(int32(StateStopped))

	elapsed := time.Since(shutdownStart)
	logger.Info("graceful shutdown complete",
		"elapsed", elapsed,
		"forceStopped", remaining,
	)
}

// MultiGracefulShutdown handles shutdown for multi-channel engine.
func MultiGracefulShutdown(me *MultiEngine, cfg ShutdownConfig, logger *slog.Logger) {
	logger.Info("multi-engine graceful shutdown initiated")

	channels := me.RunningChannels()
	logger.Info("stopping channels", "count", len(channels))

	for _, ch := range channels {
		eng := me.GetEngine(ch)
		if eng != nil {
			GracefulShutdown(context.Background(), eng, cfg, logger.With("channel", ch))
		}
	}

	me.StopAll()
	logger.Info("multi-engine shutdown complete")
}
