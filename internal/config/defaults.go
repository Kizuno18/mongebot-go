// Package config - default configuration values.
package config

import "time"

// DefaultConfig returns a new AppConfig populated with sensible defaults.
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Version: 1,
		Engine: EngineConfig{
			MaxWorkers:        50,
			RestartInterval:   Duration{10 * time.Second},
			HeartbeatInterval: Duration{60 * time.Second},
			SegmentFetchDelay: RangeConfig{
				Min: Duration{4 * time.Second},
				Max: Duration{8 * time.Second},
			},
			GQLPulseInterval: RangeConfig{
				Min: Duration{3 * time.Minute},
				Max: Duration{7 * time.Minute},
			},
			ProxyTimeout: Duration{60 * time.Second},
			MaxRetries:      3,
			StickyProxy:     true,
			BehaviorProfile: "random",
			MultiChannel:    false,
			Features: FeatureFlags{
				Ads:      true,
				Chat:     true,
				PubSub:   true,
				Segments: true,
				GQLPulse: true,
				Spade:    true,
			},
		},
		API: APIConfig{
			Port: 9800,
			Host: "127.0.0.1",
		},
		Logging: LogConfig{
			Level:      "info",
			File:       "mongebot.log",
			MaxSizeMB:  50,
			RingBuffer: 1000,
		},
		Profiles: []ProfileConfig{},
	}
}
