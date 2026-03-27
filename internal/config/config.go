// Package config handles application configuration loading, validation and persistence.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AppConfig is the root configuration for the entire application.
type AppConfig struct {
	mu       sync.RWMutex
	filePath string

	Version  int           `json:"version"`
	Engine   EngineConfig  `json:"engine"`
	API      APIConfig     `json:"api"`
	Logging  LogConfig     `json:"logging"`
	Profiles []ProfileConfig `json:"profiles"`
}

// EngineConfig controls the viewer engine behavior.
type EngineConfig struct {
	MaxWorkers        int           `json:"maxWorkers"`
	RestartInterval   Duration      `json:"restartInterval"`
	HeartbeatInterval Duration      `json:"heartbeatInterval"`
	SegmentFetchDelay RangeConfig   `json:"segmentFetchDelay"`
	GQLPulseInterval  RangeConfig   `json:"gqlPulseInterval"`
	ProxyTimeout      Duration      `json:"proxyTimeout"`
	MaxRetries        int           `json:"maxRetries"`
	Features          FeatureFlags  `json:"features"`
}

// FeatureFlags allows toggling individual viewer behaviors.
type FeatureFlags struct {
	Ads      bool `json:"ads"`
	Chat     bool `json:"chat"`
	PubSub   bool `json:"pubsub"`
	Segments bool `json:"segments"`
	GQLPulse bool `json:"gqlPulse"`
	Spade    bool `json:"spade"`
}

// RangeConfig represents a min/max duration range for randomized intervals.
type RangeConfig struct {
	Min Duration `json:"min"`
	Max Duration `json:"max"`
}

// APIConfig controls the IPC API server.
type APIConfig struct {
	Port      int    `json:"port"`
	Host      string `json:"host"`
	AuthToken string `json:"authToken,omitempty"`
}

// LogConfig controls logging behavior.
type LogConfig struct {
	Level      string `json:"level"`
	File       string `json:"file"`
	MaxSizeMB  int    `json:"maxSizeMb"`
	RingBuffer int    `json:"ringBuffer"`
}

// ProfileConfig represents a saved multi-account profile.
type ProfileConfig struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Channel  string `json:"channel"`
	Active   bool   `json:"active"`

	// Per-profile engine overrides (nil = use global)
	MaxWorkers *int          `json:"maxWorkers,omitempty"`
	Features   *FeatureFlags `json:"features,omitempty"`
}

// Duration wraps time.Duration for JSON serialization as string.
type Duration struct {
	time.Duration
}

// MarshalJSON serializes Duration as a string like "10s", "5m".
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON deserializes Duration from a string like "10s", "5m".
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = parsed
	return nil
}

// Load reads config from the given file path, or creates defaults if missing.
func Load(path string) (*AppConfig, error) {
	cfg := DefaultConfig()
	cfg.filePath = path

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config file
			if saveErr := cfg.Save(); saveErr != nil {
				return nil, fmt.Errorf("creating default config: %w", saveErr)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

// Save writes the current config to disk.
func (c *AppConfig) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	dir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(c.filePath, data, 0o644)
}

// Update applies a mutation function to the config and saves it.
func (c *AppConfig) Update(fn func(*AppConfig)) error {
	c.mu.Lock()
	fn(c)
	c.mu.Unlock()
	return c.Save()
}

// GetEngine returns a copy of the engine config (thread-safe).
func (c *AppConfig) GetEngine() EngineConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Engine
}

// GetActiveProfile returns the currently active profile, if any.
func (c *AppConfig) GetActiveProfile() *ProfileConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for i := range c.Profiles {
		if c.Profiles[i].Active {
			return &c.Profiles[i]
		}
	}
	return nil
}
