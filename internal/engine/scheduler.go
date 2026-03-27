// Package engine - timed scheduler for auto-start/stop of bot sessions.
// Supports stream-live triggers, time-based schedules, and duration limits.
package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/platform"
)

// ScheduleRule defines when the bot should auto-start/stop.
type ScheduleRule struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Channel     string        `json:"channel"`
	Platform    string        `json:"platform"`
	Trigger     TriggerType   `json:"trigger"`
	Workers     int           `json:"workers"`
	MaxDuration time.Duration `json:"maxDuration,omitempty"` // Auto-stop after duration
	Enabled     bool          `json:"enabled"`

	// Time-based triggers
	StartTime string   `json:"startTime,omitempty"` // "14:00" (24h format)
	StopTime  string   `json:"stopTime,omitempty"`  // "22:00"
	Weekdays  []int    `json:"weekdays,omitempty"`  // 0=Sun, 1=Mon, ..., 6=Sat
}

// TriggerType defines what starts the bot.
type TriggerType string

const (
	TriggerStreamLive TriggerType = "stream_live"   // Start when streamer goes live
	TriggerScheduled  TriggerType = "scheduled"     // Start at specific time
	TriggerManual     TriggerType = "manual"        // Manual only
)

// Scheduler manages auto-start/stop rules.
type Scheduler struct {
	mu       sync.RWMutex
	rules    []ScheduleRule
	running  map[string]context.CancelFunc // active rule ID -> cancel
	engine   *MultiEngine
	platform platform.Platform
	logger   *slog.Logger
}

// NewScheduler creates a new scheduler.
func NewScheduler(engine *MultiEngine, p platform.Platform, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		rules:    make([]ScheduleRule, 0),
		running:  make(map[string]context.CancelFunc),
		engine:   engine,
		platform: p,
		logger:   logger.With("component", "scheduler"),
	}
}

// AddRule adds a schedule rule.
func (s *Scheduler) AddRule(rule ScheduleRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = append(s.rules, rule)
	s.logger.Info("schedule rule added", "name", rule.Name, "trigger", rule.Trigger)
}

// RemoveRule removes a schedule rule by ID.
func (s *Scheduler) RemoveRule(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cancel if running
	if cancel, ok := s.running[id]; ok {
		cancel()
		delete(s.running, id)
	}

	filtered := make([]ScheduleRule, 0, len(s.rules))
	for _, r := range s.rules {
		if r.ID != id {
			filtered = append(filtered, r)
		}
	}
	s.rules = filtered
}

// Start begins monitoring all enabled rules.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.RLock()
	rules := make([]ScheduleRule, len(s.rules))
	copy(rules, s.rules)
	s.mu.RUnlock()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		ruleCtx, cancel := context.WithCancel(ctx)
		s.mu.Lock()
		s.running[rule.ID] = cancel
		s.mu.Unlock()

		switch rule.Trigger {
		case TriggerStreamLive:
			go s.watchStreamLive(ruleCtx, rule)
		case TriggerScheduled:
			go s.watchSchedule(ruleCtx, rule)
		}
	}

	s.logger.Info("scheduler started", "rules", len(rules))
}

// Stop cancels all running schedule monitors.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, cancel := range s.running {
		cancel()
		delete(s.running, id)
	}
	s.logger.Info("scheduler stopped")
}

// ListRules returns all schedule rules.
func (s *Scheduler) ListRules() []ScheduleRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ScheduleRule, len(s.rules))
	copy(result, s.rules)
	return result
}

// watchStreamLive polls stream status and auto-starts when live.
func (s *Scheduler) watchStreamLive(ctx context.Context, rule ScheduleRule) {
	s.logger.Info("watching for stream", "channel", rule.Channel)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	wasLive := false

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			status, err := s.platform.GetStreamStatus(ctx, rule.Channel)
			if err != nil {
				continue
			}

			isLive := status == platform.StreamOnline

			if isLive && !wasLive {
				// Stream just went live — start engine
				s.logger.Info("stream went live, auto-starting", "channel", rule.Channel)
				if err := s.engine.StartChannel(ctx, rule.Channel, rule.Workers); err != nil {
					s.logger.Warn("auto-start failed", "error", err)
				}

				// Auto-stop after max duration
				if rule.MaxDuration > 0 {
					go func() {
						select {
						case <-ctx.Done():
						case <-time.After(rule.MaxDuration):
							s.logger.Info("max duration reached, stopping", "channel", rule.Channel)
							s.engine.StopChannel(rule.Channel)
						}
					}()
				}
			} else if !isLive && wasLive {
				// Stream went offline — stop engine
				s.logger.Info("stream went offline, stopping", "channel", rule.Channel)
				s.engine.StopChannel(rule.Channel)
			}

			wasLive = isLive
		}
	}
}

// watchSchedule starts/stops based on time-of-day rules.
func (s *Scheduler) watchSchedule(ctx context.Context, rule ScheduleRule) {
	s.logger.Info("time schedule active", "channel", rule.Channel, "start", rule.StartTime, "stop", rule.StopTime)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	engineRunning := false

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			currentTime := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
			weekday := int(now.Weekday())

			// Check weekday filter
			if len(rule.Weekdays) > 0 {
				allowed := false
				for _, d := range rule.Weekdays {
					if d == weekday {
						allowed = true
						break
					}
				}
				if !allowed {
					continue
				}
			}

			if currentTime == rule.StartTime && !engineRunning {
				s.logger.Info("scheduled start time reached", "channel", rule.Channel)
				if err := s.engine.StartChannel(ctx, rule.Channel, rule.Workers); err == nil {
					engineRunning = true
				}
			}

			if currentTime == rule.StopTime && engineRunning {
				s.logger.Info("scheduled stop time reached", "channel", rule.Channel)
				s.engine.StopChannel(rule.Channel)
				engineRunning = false
			}
		}
	}
}
