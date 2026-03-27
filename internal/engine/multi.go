// Package engine - multi-channel support running independent engines concurrently.
package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/Kizuno18/mongebot-go/internal/config"
	"github.com/Kizuno18/mongebot-go/internal/platform"
	"github.com/Kizuno18/mongebot-go/internal/proxy"
	"github.com/Kizuno18/mongebot-go/pkg/useragent"
)

// MultiEngine manages multiple independent channel engines simultaneously.
type MultiEngine struct {
	mu       sync.RWMutex
	engines  map[string]*Engine // keyed by channel name
	platform platform.Platform
	proxyMgr *proxy.Manager
	tokens   []string
	uaPool   *useragent.Pool
	cfg      config.EngineConfig
	logger   *slog.Logger
	eventBus *EventBus
}

// ChannelStatus represents the status of a single channel engine.
type ChannelStatus struct {
	Channel       string              `json:"channel"`
	State         string              `json:"state"`
	ActiveViewers int                 `json:"activeViewers"`
	TotalWorkers  int                 `json:"totalWorkers"`
	Metrics       *AggregatedMetrics  `json:"metrics"`
}

// NewMultiEngine creates a multi-channel engine manager.
func NewMultiEngine(p platform.Platform, proxyMgr *proxy.Manager, tokens []string, uaPool *useragent.Pool, cfg config.EngineConfig, logger *slog.Logger) *MultiEngine {
	return &MultiEngine{
		engines:  make(map[string]*Engine),
		platform: p,
		proxyMgr: proxyMgr,
		tokens:   tokens,
		uaPool:   uaPool,
		cfg:      cfg,
		logger:   logger.With("component", "multi-engine"),
		eventBus: NewEventBus(),
	}
}

// StartChannel starts a new engine for the given channel.
func (me *MultiEngine) StartChannel(ctx context.Context, channel string, workers int) error {
	me.mu.Lock()
	defer me.mu.Unlock()

	if _, exists := me.engines[channel]; exists {
		return fmt.Errorf("channel %q is already running", channel)
	}

	eng := New(me.platform, me.proxyMgr, me.tokens, me.uaPool, me.cfg, me.logger)

	if err := eng.Start(ctx, channel, workers); err != nil {
		return fmt.Errorf("starting %q: %w", channel, err)
	}

	me.engines[channel] = eng
	me.logger.Info("channel engine started", "channel", channel, "workers", workers)

	me.eventBus.PublishSimple(EventEngineStarted, "channel", channel)
	return nil
}

// StopChannel stops the engine for a specific channel.
func (me *MultiEngine) StopChannel(channel string) error {
	me.mu.Lock()
	defer me.mu.Unlock()

	eng, exists := me.engines[channel]
	if !exists {
		return fmt.Errorf("channel %q is not running", channel)
	}

	eng.Stop()
	delete(me.engines, channel)
	me.logger.Info("channel engine stopped", "channel", channel)

	me.eventBus.PublishSimple(EventEngineStopped, "channel", channel)
	return nil
}

// StopAll stops all running channel engines.
func (me *MultiEngine) StopAll() {
	me.mu.Lock()
	defer me.mu.Unlock()

	for channel, eng := range me.engines {
		eng.Stop()
		me.logger.Info("channel engine stopped", "channel", channel)
	}
	me.engines = make(map[string]*Engine)
}

// SetChannelWorkers adjusts worker count for a specific channel.
func (me *MultiEngine) SetChannelWorkers(ctx context.Context, channel string, count int) error {
	me.mu.RLock()
	defer me.mu.RUnlock()

	eng, exists := me.engines[channel]
	if !exists {
		return fmt.Errorf("channel %q is not running", channel)
	}

	eng.SetWorkerCount(ctx, count)
	return nil
}

// Status returns the status of all running channels.
func (me *MultiEngine) Status() []ChannelStatus {
	me.mu.RLock()
	defer me.mu.RUnlock()

	var statuses []ChannelStatus
	for channel, eng := range me.engines {
		m := eng.Metrics()
		statuses = append(statuses, ChannelStatus{
			Channel:       channel,
			State:         eng.GetState().String(),
			ActiveViewers: m.ActiveViewers,
			TotalWorkers:  m.TotalWorkers,
			Metrics:       m,
		})
	}
	return statuses
}

// GetEngine returns the engine for a specific channel.
func (me *MultiEngine) GetEngine(channel string) *Engine {
	me.mu.RLock()
	defer me.mu.RUnlock()
	return me.engines[channel]
}

// RunningChannels returns a list of all running channel names.
func (me *MultiEngine) RunningChannels() []string {
	me.mu.RLock()
	defer me.mu.RUnlock()

	channels := make([]string, 0, len(me.engines))
	for ch := range me.engines {
		channels = append(channels, ch)
	}
	return channels
}

// Count returns the number of running channel engines.
func (me *MultiEngine) Count() int {
	me.mu.RLock()
	defer me.mu.RUnlock()
	return len(me.engines)
}

// AggregatedStatus returns combined metrics across all channels.
func (me *MultiEngine) AggregatedStatus() *AggregatedMetrics {
	me.mu.RLock()
	defer me.mu.RUnlock()

	total := &AggregatedMetrics{EngineState: "running"}
	for _, eng := range me.engines {
		m := eng.Metrics()
		total.ActiveViewers += m.ActiveViewers
		total.TotalWorkers += m.TotalWorkers
		total.SegmentsFetched += m.SegmentsFetched
		total.BytesReceived += m.BytesReceived
		total.HeartbeatsSent += m.HeartbeatsSent
		total.AdsWatched += m.AdsWatched
	}
	if len(me.engines) == 0 {
		total.EngineState = "stopped"
	}
	return total
}

// EventBus returns the multi-engine event bus.
func (me *MultiEngine) EventBus() *EventBus {
	return me.eventBus
}
