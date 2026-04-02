// Package platform defines the abstraction layer for multi-platform support.
// Each streaming platform (Twitch, Kick, YouTube) implements this interface.
package platform

import (
	"context"
	"time"
)

// Platform is the main interface that each streaming service must implement.
type Platform interface {
	// Name returns the platform identifier (e.g., "twitch", "kick", "youtube").
	Name() string

	// Connect creates a new Viewer instance for the given configuration.
	Connect(ctx context.Context, cfg *ViewerConfig) (Viewer, error)

	// ValidateToken checks if an auth token is still valid.
	ValidateToken(ctx context.Context, token string, proxy string) (TokenStatus, error)

	// GetStreamStatus checks whether a channel is currently live.
	GetStreamStatus(ctx context.Context, channel string) (StreamStatus, error)

	// GetStreamMetadata fetches detailed stream information.
	GetStreamMetadata(ctx context.Context, channel string, token string, proxy string) (*StreamMetadata, error)

	// SupportedFeatures returns which features this platform supports.
	SupportedFeatures() []Feature
}

// Viewer represents an active simulated viewer connection.
type Viewer interface {
	// Start begins the viewer simulation (blocking until context is cancelled or error).
	Start(ctx context.Context) error

	// Stop gracefully stops the viewer.
	Stop()

	// Status returns the current viewer state.
	Status() ViewerStatus

	// Metrics returns real-time metrics for this viewer.
	Metrics() *ViewerMetrics

	// ID returns the unique identifier for this viewer instance.
	ID() string
}

// ViewerConfig holds all parameters needed to create a viewer.
type ViewerConfig struct {
	Channel             string
	Token               string
	Proxy               string
	ProxyChain          []string        // Ordered list of proxy URLs for chain routing
	UserAgent           string
	DeviceID            string
	BehaviorProfileName string            // Name of behavior profile (lurker, active, engaged, stealth, rotating)
	Options             map[string]any

	// Behavior timing ranges (populated from engine.BehaviorProfile)
	HeartbeatInterval   MinMax `json:"heartbeatInterval,omitempty"`
	SegmentFetchDelay   MinMax `json:"segmentFetchDelay,omitempty"`
	GQLPulseInterval    MinMax `json:"gqlPulseInterval,omitempty"`
	LivenessCheckDelay  MinMax `json:"livenessCheckDelay,omitempty"`
	MaxSessionDuration  MinMax `json:"maxSessionDuration,omitempty"`
	ReconnectDelay      MinMax `json:"reconnectDelay,omitempty"`
}

// MinMax represents a min/max range for randomized durations.
type MinMax struct {
	Min time.Duration `json:"min"`
	Max time.Duration `json:"max"`
}

// StreamStatus represents the current state of a stream.
type StreamStatus int

const (
	StreamOffline StreamStatus = iota
	StreamOnline
	StreamUnknown
)

// String returns a human-readable stream status.
func (s StreamStatus) String() string {
	switch s {
	case StreamOffline:
		return "offline"
	case StreamOnline:
		return "online"
	default:
		return "unknown"
	}
}

// StreamMetadata holds information about an active stream.
type StreamMetadata struct {
	BroadcastID string    `json:"broadcastId"`
	ChannelID   string    `json:"channelId"`
	ViewerCount int       `json:"viewerCount"`
	Title       string    `json:"title"`
	Game        string    `json:"game"`
	StartedAt   time.Time `json:"startedAt"`
}

// ViewerMetrics holds real-time performance counters for a viewer.
type ViewerMetrics struct {
	Connected       bool          `json:"connected"`
	Uptime          time.Duration `json:"uptime"`
	SegmentsFetched int64         `json:"segmentsFetched"`
	BytesReceived   int64         `json:"bytesReceived"`
	HeartbeatsSent  int64         `json:"heartbeatsSent"`
	AdsWatched      int64         `json:"adsWatched"`
	LastError       string        `json:"lastError,omitempty"`
	LastActivity    time.Time     `json:"lastActivity"`
}

// ViewerStatus represents the lifecycle state of a viewer.
type ViewerStatus int

const (
	ViewerIdle ViewerStatus = iota
	ViewerConnecting
	ViewerActive
	ViewerReconnecting
	ViewerStopped
	ViewerError
)

// String returns a human-readable viewer status.
func (s ViewerStatus) String() string {
	names := [...]string{"idle", "connecting", "active", "reconnecting", "stopped", "error"}
	if int(s) < len(names) {
		return names[s]
	}
	return "unknown"
}

// TokenStatus represents the result of a token validation check.
type TokenStatus int

const (
	TokenValid TokenStatus = iota
	TokenExpired
	TokenInvalid
	TokenRateLimited
)

// Feature represents a capability that a platform may or may not support.
type Feature string

const (
	FeatureAds      Feature = "ads"
	FeatureChat     Feature = "chat"
	FeaturePubSub   Feature = "pubsub"
	FeatureSegments Feature = "segments"
	FeatureGQL      Feature = "gql"
	FeatureSpade    Feature = "spade"
	FeatureRestream Feature = "restream"
)
