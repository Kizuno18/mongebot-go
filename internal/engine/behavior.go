// Package engine - viewer behavior randomization profiles.
// Controls how each viewer behaves to simulate different user types.
package engine

import (
	"math/rand/v2"
	"time"
)

// BehaviorProfile defines timing and feature parameters for a viewer.
type BehaviorProfile struct {
	Name string `json:"name"`
	Desc string `json:"description"`

	// Timing ranges (randomized within these bounds)
	HeartbeatInterval  MinMaxDuration `json:"heartbeatInterval"`
	SegmentFetchDelay  MinMaxDuration `json:"segmentFetchDelay"`
	GQLPulseInterval   MinMaxDuration `json:"gqlPulseInterval"`
	LivenessCheckDelay MinMaxDuration `json:"livenessCheckDelay"`

	// Feature weights (probability 0.0-1.0 that feature is enabled per viewer)
	ChatJoinChance     float64 `json:"chatJoinChance"`
	PubSubChance       float64 `json:"pubSubChance"`
	AdWatchChance      float64 `json:"adWatchChance"`
	SegmentFetchChance float64 `json:"segmentFetchChance"`

	// Session behavior
	MaxSessionDuration MinMaxDuration `json:"maxSessionDuration"` // 0 = unlimited
	ReconnectDelay     MinMaxDuration `json:"reconnectDelay"`
}

// MinMaxDuration represents a randomizable duration range.
type MinMaxDuration struct {
	Min time.Duration `json:"min"`
	Max time.Duration `json:"max"`
}

// Random returns a random duration within the range.
func (r MinMaxDuration) Random() time.Duration {
	if r.Min >= r.Max {
		return r.Min
	}
	return r.Min + time.Duration(rand.Int64N(int64(r.Max-r.Min)))
}

// Profiles contains the built-in behavior profiles.
var Profiles = map[string]BehaviorProfile{
	"lurker": {
		Name: "Lurker",
		Desc: "Passive viewer — minimal interaction, just watches stream",
		HeartbeatInterval:  MinMaxDuration{55 * time.Second, 65 * time.Second},
		SegmentFetchDelay:  MinMaxDuration{6 * time.Second, 12 * time.Second},
		GQLPulseInterval:   MinMaxDuration{5 * time.Minute, 10 * time.Minute},
		LivenessCheckDelay: MinMaxDuration{30 * time.Second, 60 * time.Second},
		ChatJoinChance:     0.1,
		PubSubChance:       0.8,
		AdWatchChance:      0.9,
		SegmentFetchChance: 1.0,
		MaxSessionDuration: MinMaxDuration{0, 0}, // Unlimited
		ReconnectDelay:     MinMaxDuration{5 * time.Second, 15 * time.Second},
	},
	"active": {
		Name: "Active Viewer",
		Desc: "Regular viewer — watches stream, joins chat, receives events",
		HeartbeatInterval:  MinMaxDuration{58 * time.Second, 62 * time.Second},
		SegmentFetchDelay:  MinMaxDuration{4 * time.Second, 8 * time.Second},
		GQLPulseInterval:   MinMaxDuration{3 * time.Minute, 7 * time.Minute},
		LivenessCheckDelay: MinMaxDuration{35 * time.Second, 45 * time.Second},
		ChatJoinChance:     0.7,
		PubSubChance:       1.0,
		AdWatchChance:      1.0,
		SegmentFetchChance: 1.0,
		MaxSessionDuration: MinMaxDuration{0, 0},
		ReconnectDelay:     MinMaxDuration{3 * time.Second, 10 * time.Second},
	},
	"engaged": {
		Name: "Engaged Viewer",
		Desc: "Highly active — fast heartbeats, aggressive segment fetch, always in chat",
		HeartbeatInterval:  MinMaxDuration{55 * time.Second, 60 * time.Second},
		SegmentFetchDelay:  MinMaxDuration{2 * time.Second, 5 * time.Second},
		GQLPulseInterval:   MinMaxDuration{2 * time.Minute, 4 * time.Minute},
		LivenessCheckDelay: MinMaxDuration{20 * time.Second, 35 * time.Second},
		ChatJoinChance:     1.0,
		PubSubChance:       1.0,
		AdWatchChance:      1.0,
		SegmentFetchChance: 1.0,
		MaxSessionDuration: MinMaxDuration{0, 0},
		ReconnectDelay:     MinMaxDuration{1 * time.Second, 5 * time.Second},
	},
	"stealth": {
		Name: "Stealth",
		Desc: "Minimal footprint — slow timing, fewer features, harder to detect",
		HeartbeatInterval:  MinMaxDuration{58 * time.Second, 70 * time.Second},
		SegmentFetchDelay:  MinMaxDuration{8 * time.Second, 20 * time.Second},
		GQLPulseInterval:   MinMaxDuration{8 * time.Minute, 15 * time.Minute},
		LivenessCheckDelay: MinMaxDuration{60 * time.Second, 120 * time.Second},
		ChatJoinChance:     0.0,
		PubSubChance:       0.5,
		AdWatchChance:      0.3,
		SegmentFetchChance: 0.7,
		MaxSessionDuration: MinMaxDuration{30 * time.Minute, 2 * time.Hour},
		ReconnectDelay:     MinMaxDuration{10 * time.Second, 30 * time.Second},
	},
	"rotating": {
		Name: "Rotating",
		Desc: "Short sessions with frequent reconnects — simulates tab switching",
		HeartbeatInterval:  MinMaxDuration{58 * time.Second, 62 * time.Second},
		SegmentFetchDelay:  MinMaxDuration{4 * time.Second, 8 * time.Second},
		GQLPulseInterval:   MinMaxDuration{3 * time.Minute, 5 * time.Minute},
		LivenessCheckDelay: MinMaxDuration{30 * time.Second, 45 * time.Second},
		ChatJoinChance:     0.3,
		PubSubChance:       0.8,
		AdWatchChance:      0.5,
		SegmentFetchChance: 1.0,
		MaxSessionDuration: MinMaxDuration{5 * time.Minute, 20 * time.Minute},
		ReconnectDelay:     MinMaxDuration{2 * time.Second, 8 * time.Second},
	},
}

// RandomProfile returns a weighted random behavior profile.
// Distribution: 40% lurker, 30% active, 15% engaged, 10% stealth, 5% rotating
func RandomProfile() BehaviorProfile {
	r := rand.Float64()
	switch {
	case r < 0.40:
		return Profiles["lurker"]
	case r < 0.70:
		return Profiles["active"]
	case r < 0.85:
		return Profiles["engaged"]
	case r < 0.95:
		return Profiles["stealth"]
	default:
		return Profiles["rotating"]
	}
}

// ShouldEnable returns true based on a probability (0.0-1.0).
func ShouldEnable(chance float64) bool {
	return rand.Float64() < chance
}

// GetProfile returns a named behavior profile or falls back to "active".
func GetProfile(name string) BehaviorProfile {
	if p, ok := Profiles[name]; ok {
		return p
	}
	return Profiles["active"]
}

// ListProfiles returns all available profile names.
func ListProfiles() []string {
	names := make([]string, 0, len(Profiles))
	for name := range Profiles {
		names = append(names, name)
	}
	return names
}
