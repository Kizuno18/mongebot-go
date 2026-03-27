// Package twitch - main platform provider implementing platform.Platform interface.
package twitch

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/platform"
)

// Provider implements platform.Platform for Twitch.
type Provider struct {
	logger *slog.Logger
}

// NewProvider creates a new Twitch platform provider.
func NewProvider(logger *slog.Logger) *Provider {
	return &Provider{
		logger: logger.With("platform", "twitch"),
	}
}

// Name returns "twitch".
func (p *Provider) Name() string {
	return "twitch"
}

// SupportedFeatures returns all features Twitch supports.
func (p *Provider) SupportedFeatures() []platform.Feature {
	return []platform.Feature{
		platform.FeatureAds,
		platform.FeatureChat,
		platform.FeaturePubSub,
		platform.FeatureSegments,
		platform.FeatureGQL,
		platform.FeatureSpade,
		platform.FeatureRestream,
	}
}

// Connect creates a new TwitchViewer for the given configuration.
func (p *Provider) Connect(ctx context.Context, cfg *platform.ViewerConfig) (platform.Viewer, error) {
	viewer := NewViewer(cfg, p.logger)
	return viewer, nil
}

// ValidateToken checks if a Twitch OAuth token is still valid.
func (p *Provider) ValidateToken(ctx context.Context, token string, proxyURL string) (platform.TokenStatus, error) {
	client := p.httpClient(proxyURL)

	payload := `{"query": "query { currentUser { id } }"}`
	req, err := http.NewRequestWithContext(ctx, "POST", GQLURL, stringReader(payload))
	if err != nil {
		return platform.TokenInvalid, err
	}

	setGQLHeaders(req, token, "", "")

	resp, err := client.Do(req)
	if err != nil {
		return platform.TokenInvalid, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		return platform.TokenValid, nil
	case 401:
		return platform.TokenExpired, nil
	case 429:
		return platform.TokenRateLimited, nil
	default:
		return platform.TokenInvalid, nil
	}
}

// GetStreamStatus checks whether a channel is live.
func (p *Provider) GetStreamStatus(ctx context.Context, channel string) (platform.StreamStatus, error) {
	meta, err := p.GetStreamMetadata(ctx, channel, "", "")
	if err != nil {
		return platform.StreamUnknown, err
	}
	if meta.BroadcastID != "" {
		return platform.StreamOnline, nil
	}
	return platform.StreamOffline, nil
}

// GetStreamMetadata fetches broadcast ID, channel ID, and other metadata.
func (p *Provider) GetStreamMetadata(ctx context.Context, channel string, token string, proxyURL string) (*platform.StreamMetadata, error) {
	client := p.httpClient(proxyURL)

	gql := newGQLClient(client, token, "", "")
	broadcastID, channelID, err := gql.getStreamMetadata(ctx, channel)
	if err != nil {
		return nil, err
	}

	return &platform.StreamMetadata{
		BroadcastID: broadcastID,
		ChannelID:   channelID,
	}, nil
}

// httpClient creates an HTTP client with optional proxy support.
func (p *Provider) httpClient(proxyURL string) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		MaxConnsPerHost:     10,
	}

	if proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}
}
