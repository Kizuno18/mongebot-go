// Package engine orchestrates the viewer worker pool, managing lifecycle,
// proxy/token assignment, metrics aggregation, and auto-restart of dead workers.
package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/config"
	"github.com/Kizuno18/mongebot-go/internal/platform"
	"github.com/Kizuno18/mongebot-go/internal/proxy"
	"github.com/Kizuno18/mongebot-go/pkg/fingerprint"
	"github.com/Kizuno18/mongebot-go/pkg/useragent"
)

// State represents the engine's lifecycle state.
type State int

const (
	StateStopped State = iota
	StateStarting
	StateRunning
	StatePaused
	StateStopping
)

// String returns a human-readable state.
func (s State) String() string {
	names := [...]string{"stopped", "starting", "running", "paused", "stopping"}
	if int(s) < len(names) {
		return names[s]
	}
	return "unknown"
}

// Engine is the main orchestrator for viewer workers.
type Engine struct {
	mu       sync.RWMutex
	state    atomic.Int32
	platform platform.Platform
	proxyMgr *proxy.Manager
	tokens   []string
	uaPool   *useragent.Pool
	cfg      config.EngineConfig
	logger   *slog.Logger

	// Worker management
	workers    map[string]*Worker
	workersMu  sync.RWMutex
	cancel     context.CancelFunc
	channel    string

	// Rate limit tracking
	rateLimiter *RateLimitTracker

	// Event callbacks (for API layer)
	onMetrics func(*AggregatedMetrics)
	onLog     func(string, string) // level, message
}

// AggregatedMetrics holds the combined metrics from all active workers.
type AggregatedMetrics struct {
	ActiveViewers   int           `json:"activeViewers"`
	TotalWorkers    int           `json:"totalWorkers"`
	SegmentsFetched int64         `json:"segmentsFetched"`
	BytesReceived   int64         `json:"bytesReceived"`
	HeartbeatsSent  int64         `json:"heartbeatsSent"`
	AdsWatched      int64         `json:"adsWatched"`
	Uptime          time.Duration `json:"uptime"`
	EngineState     string        `json:"engineState"`
	Channel         string        `json:"channel"`
}

// New creates a new Engine.
func New(p platform.Platform, proxyMgr *proxy.Manager, tokens []string, uaPool *useragent.Pool, cfg config.EngineConfig, logger *slog.Logger) *Engine {
	e := &Engine{
		platform:    p,
		proxyMgr:    proxyMgr,
		tokens:      tokens,
		uaPool:      uaPool,
		cfg:         cfg,
		logger:      logger.With("component", "engine"),
		workers:     make(map[string]*Worker),
		rateLimiter: NewRateLimitTracker(logger),
	}
	e.state.Store(int32(StateStopped))
	return e
}

// Start begins the engine for the given channel with the configured number of workers.
func (e *Engine) Start(ctx context.Context, channel string, numWorkers int) error {
	if State(e.state.Load()) == StateRunning {
		return fmt.Errorf("engine already running")
	}

	e.state.Store(int32(StateStarting))
	e.channel = channel

	ctx, e.cancel = context.WithCancel(ctx)

	e.logger.Info("starting engine", "channel", channel, "workers", numWorkers)

	// Spawn initial workers
	for i := range numWorkers {
		if err := e.spawnWorker(ctx, i); err != nil {
			e.logger.Warn("failed to spawn worker", "index", i, "error", err)
		}
		time.Sleep(100 * time.Millisecond) // Stagger starts
	}

	e.state.Store(int32(StateRunning))

	// Start the worker monitor (restarts dead workers)
	go e.monitorLoop(ctx, numWorkers)

	// Start metrics aggregation loop
	go e.metricsLoop(ctx)

	return nil
}

// Stop gracefully stops all workers and the engine.
func (e *Engine) Stop() {
	e.state.Store(int32(StateStopping))
	e.logger.Info("stopping engine")

	if e.cancel != nil {
		e.cancel()
	}

	e.workersMu.Lock()
	for _, w := range e.workers {
		w.Stop()
	}
	e.workers = make(map[string]*Worker)
	e.workersMu.Unlock()

	e.state.Store(int32(StateStopped))
	e.logger.Info("engine stopped")
}

// SetWorkerCount dynamically adjusts the number of active workers.
func (e *Engine) SetWorkerCount(ctx context.Context, count int) {
	e.workersMu.Lock()
	current := len(e.workers)
	e.workersMu.Unlock()

	if count > current {
		// Scale up
		for i := current; i < count; i++ {
			if err := e.spawnWorker(ctx, i); err != nil {
				e.logger.Warn("failed to spawn worker during scale-up", "error", err)
			}
		}
	} else if count < current {
		// Scale down: stop excess workers
		e.workersMu.Lock()
		removed := 0
		for id, w := range e.workers {
			if removed >= current-count {
				break
			}
			w.Stop()
			delete(e.workers, id)
			removed++
		}
		e.workersMu.Unlock()
	}
}

// Metrics returns aggregated metrics from all workers.
func (e *Engine) Metrics() *AggregatedMetrics {
	e.workersMu.RLock()
	defer e.workersMu.RUnlock()

	m := &AggregatedMetrics{
		TotalWorkers: len(e.workers),
		EngineState:  State(e.state.Load()).String(),
		Channel:      e.channel,
	}

	for _, w := range e.workers {
		wm := w.viewer.Metrics()
		if wm.Connected {
			m.ActiveViewers++
		}
		m.SegmentsFetched += wm.SegmentsFetched
		m.BytesReceived += wm.BytesReceived
		m.HeartbeatsSent += wm.HeartbeatsSent
		m.AdsWatched += wm.AdsWatched
		if wm.Uptime > m.Uptime {
			m.Uptime = wm.Uptime
		}
	}
	return m
}

// State returns the current engine state.
func (e *Engine) GetState() State {
	return State(e.state.Load())
}

// OnMetrics sets a callback for periodic metrics updates.
func (e *Engine) OnMetrics(fn func(*AggregatedMetrics)) {
	e.onMetrics = fn
}

// spawnWorker creates and starts a new viewer worker.
func (e *Engine) spawnWorker(ctx context.Context, index int) error {
	// Acquire proxy and token
	p := e.proxyMgr.Acquire()
	if p == nil {
		return fmt.Errorf("no available proxies")
	}

	if index >= len(e.tokens) {
		return fmt.Errorf("no available tokens (index %d, have %d)", index, len(e.tokens))
	}
	token := e.tokens[index%len(e.tokens)]

	viewerCfg := &platform.ViewerConfig{
		Channel:   e.channel,
		Token:     token,
		Proxy:     p.URL(),
		UserAgent: e.uaPool.Random(),
		DeviceID:  fingerprint.GenerateDeviceID(),
	}

	// Wrap viewer with auto-reconnection
	reconnectViewer := NewReconnectingViewer(
		e.platform, viewerCfg, DefaultReconnectConfig(), e.logger,
	)

	w := &Worker{
		id:       viewerCfg.DeviceID,
		viewer:   reconnectViewer,
		proxy:    p,
		proxyMgr: e.proxyMgr,
	}

	e.workersMu.Lock()
	e.workers[w.id] = w
	e.workersMu.Unlock()

	// Start worker in a goroutine
	go func() {
		defer func() {
			e.proxyMgr.Release(p)
			e.workersMu.Lock()
			delete(e.workers, w.id)
			e.workersMu.Unlock()
		}()

		if err := viewer.Start(ctx); err != nil {
			e.logger.Debug("worker finished", "id", w.id[:8], "error", err)
		}
	}()

	return nil
}

// monitorLoop periodically checks for dead workers and restarts them.
func (e *Engine) monitorLoop(ctx context.Context, targetWorkers int) {
	ticker := time.NewTicker(e.cfg.RestartInterval.Duration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if State(e.state.Load()) != StateRunning {
				continue
			}

			e.workersMu.RLock()
			current := len(e.workers)
			e.workersMu.RUnlock()

			if current < targetWorkers {
				deficit := targetWorkers - current
				e.logger.Info("restarting dead workers", "deficit", deficit)
				for i := range deficit {
					if err := e.spawnWorker(ctx, current+i); err != nil {
						e.logger.Warn("failed to restart worker", "error", err)
					}
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	}
}

// metricsLoop periodically aggregates and emits metrics.
func (e *Engine) metricsLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if e.onMetrics != nil {
				e.onMetrics(e.Metrics())
			}
		}
	}
}
