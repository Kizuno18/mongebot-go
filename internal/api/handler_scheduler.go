// Package api - handlers for scheduler, channel search, and multi-channel engine.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/engine"
	"github.com/Kizuno18/mongebot-go/internal/platform/twitch"
)

// SchedulerDeps holds scheduler and multi-engine dependencies.
type SchedulerDeps struct {
	MultiEngine *engine.MultiEngine
	Scheduler   *engine.Scheduler
}

var globalSchedulerDeps *SchedulerDeps

// SetSchedulerDeps sets the scheduler dependencies globally.
func SetSchedulerDeps(deps *SchedulerDeps) {
	globalSchedulerDeps = deps
}

// getSchedulerHandler returns handlers for scheduler/search/multi methods.
func getSchedulerHandler(method string) (handlerFunc, bool) {
	handlers := map[string]handlerFunc{
		// Multi-channel engine
		"multi.start":    handleMultiStart,
		"multi.stop":     handleMultiStop,
		"multi.stopAll":  handleMultiStopAll,
		"multi.status":   handleMultiStatus,
		"multi.channels": handleMultiChannels,
		"multi.workers":  handleMultiWorkers,

		// Scheduler
		"scheduler.list":   handleSchedulerList,
		"scheduler.add":    handleSchedulerAdd,
		"scheduler.remove": handleSchedulerRemove,
		"scheduler.start":  handleSchedulerStart,
		"scheduler.stop":   handleSchedulerStop,

		// Channel search
		"channel.search": handleChannelSearch,

		// Behavior profiles
		"behavior.list": handleBehaviorList,

		// Drops tracking
		"drops.progress": handleDropsProgress,
		"drops.points":   handleDropsPoints,
	}

	h, ok := handlers[method]
	return h, ok
}

// --- Multi-channel handlers ---

type multiStartParams struct {
	Channel string `json:"channel"`
	Workers int    `json:"workers"`
}

func handleMultiStart(ctx context.Context, params json.RawMessage) (any, error) {
	if globalSchedulerDeps == nil || globalSchedulerDeps.MultiEngine == nil {
		return nil, fmt.Errorf("multi-engine not initialized")
	}
	var p multiStartParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Channel == "" {
		return nil, fmt.Errorf("channel is required")
	}
	if p.Workers <= 0 {
		p.Workers = 50
	}
	if err := globalSchedulerDeps.MultiEngine.StartChannel(ctx, p.Channel, p.Workers); err != nil {
		return nil, err
	}
	return map[string]any{"status": "started", "channel": p.Channel, "workers": p.Workers}, nil
}

type multiStopParams struct {
	Channel string `json:"channel"`
}

func handleMultiStop(_ context.Context, params json.RawMessage) (any, error) {
	if globalSchedulerDeps == nil || globalSchedulerDeps.MultiEngine == nil {
		return nil, fmt.Errorf("multi-engine not initialized")
	}
	var p multiStopParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	return map[string]string{"status": "stopped"}, globalSchedulerDeps.MultiEngine.StopChannel(p.Channel)
}

func handleMultiStopAll(_ context.Context, _ json.RawMessage) (any, error) {
	if globalSchedulerDeps == nil || globalSchedulerDeps.MultiEngine == nil {
		return nil, fmt.Errorf("multi-engine not initialized")
	}
	globalSchedulerDeps.MultiEngine.StopAll()
	return map[string]string{"status": "all stopped"}, nil
}

func handleMultiStatus(_ context.Context, _ json.RawMessage) (any, error) {
	if globalSchedulerDeps == nil || globalSchedulerDeps.MultiEngine == nil {
		return nil, fmt.Errorf("multi-engine not initialized")
	}
	return map[string]any{
		"channels":   globalSchedulerDeps.MultiEngine.Status(),
		"count":      globalSchedulerDeps.MultiEngine.Count(),
		"aggregated": globalSchedulerDeps.MultiEngine.AggregatedStatus(),
	}, nil
}

func handleMultiChannels(_ context.Context, _ json.RawMessage) (any, error) {
	if globalSchedulerDeps == nil || globalSchedulerDeps.MultiEngine == nil {
		return nil, fmt.Errorf("multi-engine not initialized")
	}
	return globalSchedulerDeps.MultiEngine.RunningChannels(), nil
}

type multiWorkersParams struct {
	Channel string `json:"channel"`
	Count   int    `json:"count"`
}

func handleMultiWorkers(ctx context.Context, params json.RawMessage) (any, error) {
	if globalSchedulerDeps == nil || globalSchedulerDeps.MultiEngine == nil {
		return nil, fmt.Errorf("multi-engine not initialized")
	}
	var p multiWorkersParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	return map[string]string{"status": "updated"}, globalSchedulerDeps.MultiEngine.SetChannelWorkers(ctx, p.Channel, p.Count)
}

// --- Scheduler handlers ---

func handleSchedulerList(_ context.Context, _ json.RawMessage) (any, error) {
	if globalSchedulerDeps == nil || globalSchedulerDeps.Scheduler == nil {
		return nil, fmt.Errorf("scheduler not initialized")
	}
	return globalSchedulerDeps.Scheduler.ListRules(), nil
}

func handleSchedulerAdd(_ context.Context, params json.RawMessage) (any, error) {
	if globalSchedulerDeps == nil || globalSchedulerDeps.Scheduler == nil {
		return nil, fmt.Errorf("scheduler not initialized")
	}
	var rule engine.ScheduleRule
	if err := json.Unmarshal(params, &rule); err != nil {
		return nil, err
	}
	globalSchedulerDeps.Scheduler.AddRule(rule)
	return map[string]string{"status": "added", "id": rule.ID}, nil
}

type schedulerRemoveParams struct {
	ID string `json:"id"`
}

func handleSchedulerRemove(_ context.Context, params json.RawMessage) (any, error) {
	if globalSchedulerDeps == nil || globalSchedulerDeps.Scheduler == nil {
		return nil, fmt.Errorf("scheduler not initialized")
	}
	var p schedulerRemoveParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	globalSchedulerDeps.Scheduler.RemoveRule(p.ID)
	return map[string]string{"status": "removed"}, nil
}

func handleSchedulerStart(ctx context.Context, _ json.RawMessage) (any, error) {
	if globalSchedulerDeps == nil || globalSchedulerDeps.Scheduler == nil {
		return nil, fmt.Errorf("scheduler not initialized")
	}
	globalSchedulerDeps.Scheduler.Start(ctx)
	return map[string]string{"status": "scheduler started"}, nil
}

func handleSchedulerStop(_ context.Context, _ json.RawMessage) (any, error) {
	if globalSchedulerDeps == nil || globalSchedulerDeps.Scheduler == nil {
		return nil, fmt.Errorf("scheduler not initialized")
	}
	globalSchedulerDeps.Scheduler.Stop()
	return map[string]string{"status": "scheduler stopped"}, nil
}

// --- Channel search handler ---

type channelSearchParams struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

func handleChannelSearch(ctx context.Context, params json.RawMessage) (any, error) {
	var p channelSearchParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Query == "" {
		return []any{}, nil
	}
	if p.Limit <= 0 {
		p.Limit = 8
	}

	// Get a valid token for the search
	var token string
	if globalExtDeps.TokenMgr != nil {
		tokens := globalExtDeps.TokenMgr.GetValidValues()
		if len(tokens) > 0 {
			token = tokens[0]
		}
	}

	// Use a default HTTP client for search
	client := &http.Client{Timeout: 15 * time.Second}
	results, err := twitch.SearchChannels(ctx, client, p.Query, token, p.Limit)
	if err != nil {
		return nil, fmt.Errorf("channel search failed: %w", err)
	}
	return results, nil
}

// --- Behavior profile handlers ---

func handleBehaviorList(_ context.Context, _ json.RawMessage) (any, error) {
	profiles := make(map[string]any)
	for name, p := range engine.Profiles {
		profiles[name] = map[string]any{
			"name":        p.Name,
			"description": p.Desc,
			"chatChance":  p.ChatJoinChance,
			"adChance":    p.AdWatchChance,
		}
	}
	return profiles, nil
}

// --- Drops handlers ---

type dropsPointsParams struct {
	ChannelID string `json:"channelId"`
}

func handleDropsPoints(ctx context.Context, params json.RawMessage) (any, error) {
	var p dropsPointsParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.ChannelID == "" {
		return nil, fmt.Errorf("channelId required")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	tracker := twitch.NewDropsTracker(client, "") // Token from extended deps
	return tracker.GetChannelPoints(ctx, p.ChannelID)
}

func handleDropsProgress(ctx context.Context, _ json.RawMessage) (any, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	tracker := twitch.NewDropsTracker(client, "")
	return tracker.GetDropsProgress(ctx)
}
