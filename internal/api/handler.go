// Package api - JSON-RPC method handlers.
package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Kizuno18/mongebot-go/internal/proxy"
)

// handlerFunc is the signature for RPC method handlers.
type handlerFunc func(ctx context.Context, params json.RawMessage) (any, error)

// getHandler returns the handler for a given method name.
func (s *Server) getHandler(method string) (handlerFunc, bool) {
	handlers := map[string]handlerFunc{
		// Engine methods
		"engine.start":      s.handleEngineStart,
		"engine.stop":       s.handleEngineStop,
		"engine.status":     s.handleEngineStatus,
		"engine.setWorkers": s.handleEngineSetWorkers,

		// Proxy methods
		"proxy.list":    s.handleProxyList,
		"proxy.import":  s.handleProxyImport,
		"proxy.check":   s.handleProxyCheck,

		// Config methods
		"config.get": s.handleConfigGet,
		"config.set": s.handleConfigSet,

		// Log methods
		"logs.history": s.handleLogsHistory,
	}

	h, ok := handlers[method]
	if ok {
		return h, true
	}

	// Check extended handlers (profiles, tokens, stream, scraper)
	if h, ok := getExtendedHandler(method); ok {
		return h, true
	}

	// Check scheduler/search/multi-engine handlers
	if h, ok := getSchedulerHandler(method); ok {
		return h, true
	}

	// Check system handlers
	if h, ok := getSystemHandler(method); ok {
		return h, true
	}

	// Check webhook handlers
	return getWebhookHandler(method)
}

// --- Engine Handlers ---

type engineStartParams struct {
	Channel string `json:"channel"`
	Workers int    `json:"workers"`
}

func (s *Server) handleEngineStart(ctx context.Context, params json.RawMessage) (any, error) {
	var p engineStartParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if p.Channel == "" {
		return nil, fmt.Errorf("channel is required")
	}
	if p.Workers <= 0 {
		p.Workers = s.appCfg.GetEngine().MaxWorkers
	}

	if err := s.engine.Start(ctx, p.Channel, p.Workers); err != nil {
		return nil, err
	}

	return map[string]any{
		"status":  "started",
		"channel": p.Channel,
		"workers": p.Workers,
	}, nil
}

func (s *Server) handleEngineStop(_ context.Context, _ json.RawMessage) (any, error) {
	s.engine.Stop()
	return map[string]string{"status": "stopped"}, nil
}

func (s *Server) handleEngineStatus(_ context.Context, _ json.RawMessage) (any, error) {
	return s.engine.Metrics(), nil
}

type setWorkersParams struct {
	Count int `json:"count"`
}

func (s *Server) handleEngineSetWorkers(ctx context.Context, params json.RawMessage) (any, error) {
	var p setWorkersParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.Count <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}

	s.engine.SetWorkerCount(ctx, p.Count)
	return map[string]any{"workers": p.Count}, nil
}

// --- Proxy Handlers ---

func (s *Server) handleProxyList(_ context.Context, _ json.RawMessage) (any, error) {
	proxies := s.proxyMgr.All()
	total, available, inUse := s.proxyMgr.Count()
	return map[string]any{
		"proxies":   proxies,
		"total":     total,
		"available": available,
		"inUse":     inUse,
	}, nil
}

type proxyImportParams struct {
	Proxies []string `json:"proxies"`
}

func (s *Server) handleProxyImport(_ context.Context, params json.RawMessage) (any, error) {
	var p proxyImportParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	added, errors := s.proxyMgr.AddBulk(p.Proxies)
	return map[string]any{
		"added":  added,
		"errors": errors,
	}, nil
}

func (s *Server) handleProxyCheck(_ context.Context, _ json.RawMessage) (any, error) {
	go func() {
		checker := proxy.NewChecker(s.logger)
		checker.CheckAll(context.Background(), s.proxyMgr, nil)
	}()
	return map[string]string{"status": "check started"}, nil
}

// --- Config Handlers ---

func (s *Server) handleConfigGet(_ context.Context, _ json.RawMessage) (any, error) {
	return s.appCfg, nil
}

func (s *Server) handleConfigSet(_ context.Context, params json.RawMessage) (any, error) {
	// Partial config update
	var updates map[string]any
	if err := json.Unmarshal(params, &updates); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Apply updates via JSON merge
	currentJSON, _ := json.Marshal(s.appCfg)
	merged := make(map[string]any)
	json.Unmarshal(currentJSON, &merged)

	for k, v := range updates {
		merged[k] = v
	}

	mergedJSON, _ := json.Marshal(merged)
	if err := json.Unmarshal(mergedJSON, s.appCfg); err != nil {
		return nil, fmt.Errorf("applying config: %w", err)
	}

	if err := s.appCfg.Save(); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}

	return map[string]string{"status": "updated"}, nil
}

// --- Log Handlers ---

func (s *Server) handleLogsHistory(_ context.Context, _ json.RawMessage) (any, error) {
	return s.logRing.All(), nil
}
