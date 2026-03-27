// Package account manages multi-account profiles with per-channel configurations.
package account

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Profile represents a saved bot configuration for a specific channel.
type Profile struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Platform  string    `json:"platform"`
	Channel   string    `json:"channel"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Engine overrides (nil = use global default)
	MaxWorkers *int             `json:"maxWorkers,omitempty"`
	Features   *FeatureOverride `json:"features,omitempty"`

	// Associated resources
	TokenIDs []string `json:"tokenIds,omitempty"`
	ProxyTag string   `json:"proxyTag,omitempty"` // Filter proxies by tag/group
	Notes    string   `json:"notes,omitempty"`
}

// FeatureOverride allows per-profile feature flag overrides.
type FeatureOverride struct {
	Ads      *bool `json:"ads,omitempty"`
	Chat     *bool `json:"chat,omitempty"`
	PubSub   *bool `json:"pubsub,omitempty"`
	Segments *bool `json:"segments,omitempty"`
	GQLPulse *bool `json:"gqlPulse,omitempty"`
	Spade    *bool `json:"spade,omitempty"`
}

// NewProfile creates a new profile with a generated ID.
func NewProfile(name, platform, channel string) *Profile {
	return &Profile{
		ID:        generateID(),
		Name:      name,
		Platform:  platform,
		Channel:   channel,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Clone creates a copy of the profile with a new ID and name.
func (p *Profile) Clone(newName string) *Profile {
	clone := *p
	clone.ID = generateID()
	clone.Name = newName
	clone.Active = false
	clone.CreatedAt = time.Now()
	clone.UpdatedAt = time.Now()
	return &clone
}

// EffectiveWorkers returns the worker count for this profile,
// falling back to the provided global default.
func (p *Profile) EffectiveWorkers(globalDefault int) int {
	if p.MaxWorkers != nil {
		return *p.MaxWorkers
	}
	return globalDefault
}

// EffectiveFeature returns whether a feature is enabled,
// falling back to the global default.
func (f *FeatureOverride) EffectiveFeature(feature string, globalDefault bool) bool {
	if f == nil {
		return globalDefault
	}
	switch feature {
	case "ads":
		if f.Ads != nil {
			return *f.Ads
		}
	case "chat":
		if f.Chat != nil {
			return *f.Chat
		}
	case "pubsub":
		if f.PubSub != nil {
			return *f.PubSub
		}
	case "segments":
		if f.Segments != nil {
			return *f.Segments
		}
	case "gqlPulse":
		if f.GQLPulse != nil {
			return *f.GQLPulse
		}
	case "spade":
		if f.Spade != nil {
			return *f.Spade
		}
	}
	return globalDefault
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
