// Package token - auto-refresh for periodic token validation.
package token

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/platform"
	"github.com/Kizuno18/mongebot-go/internal/proxy"
)

// AutoRefresher periodically validates tokens to detect expiring ones.
type AutoRefresher struct {
	mgr       *Manager
	validator *Validator
	proxyMgr  *proxy.Manager
	interval  time.Duration
	logger    *slog.Logger

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// AutoRefreshConfig configures the auto-refresh behavior.
type AutoRefreshConfig struct {
	Interval       time.Duration // How often to run refresh (default: 1h)
	StaleThreshold time.Duration // Re-check tokens not validated in this time (default: 30m)
}

// DefaultAutoRefreshConfig returns sensible defaults.
func DefaultAutoRefreshConfig() AutoRefreshConfig {
	return AutoRefreshConfig{
		Interval:       time.Hour,
		StaleThreshold: 30 * time.Minute,
	}
}

// NewAutoRefresher creates a new auto-refresh worker.
func NewAutoRefresher(mgr *Manager, p platform.Platform, proxyMgr *proxy.Manager, cfg AutoRefreshConfig, logger *slog.Logger) *AutoRefresher {
	if cfg.Interval == 0 {
		cfg = DefaultAutoRefreshConfig()
	}

	return &AutoRefresher{
		mgr:       mgr,
		validator: NewValidator(mgr, p, logger),
		proxyMgr:  proxyMgr,
		interval:  cfg.Interval,
		logger:    logger.With("component", "token-auto-refresh"),
	}
}

// Start begins the periodic refresh loop.
func (a *AutoRefresher) Start(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cancel != nil {
		return // Already running
	}

	a.ctx, a.cancel = context.WithCancel(context.Background())
	a.wg.Add(1)

	go a.loop()

	a.logger.Info("token auto-refresh started", "interval", a.interval)
}

// Stop halts the refresh loop.
func (a *AutoRefresher) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cancel == nil {
		return
	}

	a.cancel()
	a.cancel = nil
	a.wg.Wait()

	a.logger.Info("token auto-refresh stopped")
}

func (a *AutoRefresher) loop() {
	defer a.wg.Done()

	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.refresh()
		}
	}
}

func (a *AutoRefresher) refresh() {
	stats := a.mgr.Stats()
	a.logger.Debug("starting token refresh", "total", stats.Total, "valid", stats.Valid)

	// Only validate tokens that need it
	tokens := a.mgr.All()
	var toCheck []*ManagedToken
	now := time.Now()

	for _, t := range tokens {
		// Skip quarantined tokens
		if t.State == StateQuarantined {
			continue
		}
		// Check if token hasn't been validated recently
		if t.LastChecked.IsZero() || now.Sub(t.LastChecked) > 30*time.Minute {
			toCheck = append(toCheck, t)
		}
	}

	if len(toCheck) == 0 {
		a.logger.Debug("no tokens need refresh")
		return
	}

	a.logger.Info("refreshing tokens", "count", len(toCheck))

	// Run validation
	progress := a.validator.ValidateAll(a.ctx, a.proxyMgr)

	a.logger.Info("token refresh complete",
		"checked", progress.Checked,
		"valid", progress.Valid,
		"invalid", progress.Invalid,
	)
}

// RefreshNow triggers an immediate refresh (non-blocking).
func (a *AutoRefresher) RefreshNow() {
	go a.refresh()
}
