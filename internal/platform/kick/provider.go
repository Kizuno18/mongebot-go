// Package kick implements the Kick.com platform provider for MongeBot.
// Kick uses HLS for video delivery and Pusher WebSocket for chat/events.
package kick

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/platform"
)

const (
	kickAPIBase  = "https://kick.com/api/v2"
	kickChatWS   = "wss://ws-us2.pusher.com/app/32cbd69e4b950bf97679"
)

// Provider implements platform.Platform for Kick.com.
type Provider struct {
	logger *slog.Logger
}

// NewProvider creates a new Kick platform provider.
func NewProvider(logger *slog.Logger) *Provider {
	return &Provider{
		logger: logger.With("platform", "kick"),
	}
}

// Name returns "kick".
func (p *Provider) Name() string {
	return "kick"
}

// SupportedFeatures returns features available on Kick.
func (p *Provider) SupportedFeatures() []platform.Feature {
	return []platform.Feature{
		platform.FeatureSegments,
		platform.FeatureChat,
	}
}

// Connect creates a new KickViewer for the given configuration.
func (p *Provider) Connect(ctx context.Context, cfg *platform.ViewerConfig) (platform.Viewer, error) {
	viewer := NewViewer(cfg, p.logger)
	return viewer, nil
}

// ValidateToken validates a Kick auth token (Kick uses session cookies).
func (p *Provider) ValidateToken(ctx context.Context, token string, proxyURL string) (platform.TokenStatus, error) {
	// Kick doesn't use OAuth tokens the same way — uses session cookies
	// For now, just check if the token is non-empty
	if token == "" {
		return platform.TokenInvalid, nil
	}
	return platform.TokenValid, nil
}

// GetStreamStatus checks if a Kick channel is live.
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

// GetStreamMetadata fetches Kick channel and stream information.
func (p *Provider) GetStreamMetadata(ctx context.Context, channel string, token string, proxyURL string) (*platform.StreamMetadata, error) {
	client := p.httpClient(proxyURL)

	apiURL := fmt.Sprintf("%s/channels/%s", kickAPIBase, channel)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching channel info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("kick API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var channelData struct {
		ID          int    `json:"id"`
		Slug        string `json:"slug"`
		Livestream  *struct {
			ID         int    `json:"id"`
			SessionTitle string `json:"session_title"`
			Viewers    int    `json:"viewers"`
			Categories []struct {
				Name string `json:"name"`
			} `json:"categories"`
			CreatedAt  string `json:"created_at"`
		} `json:"livestream"`
		PlaybackURL string `json:"playback_url"`
		ChatroomID  struct {
			ID int `json:"id"`
		} `json:"chatroom"`
	}

	if err := json.Unmarshal(body, &channelData); err != nil {
		return nil, fmt.Errorf("parsing channel data: %w", err)
	}

	meta := &platform.StreamMetadata{
		ChannelID: fmt.Sprintf("%d", channelData.ID),
	}

	if channelData.Livestream != nil {
		meta.BroadcastID = fmt.Sprintf("%d", channelData.Livestream.ID)
		meta.Title = channelData.Livestream.SessionTitle
		meta.ViewerCount = channelData.Livestream.Viewers
		if len(channelData.Livestream.Categories) > 0 {
			meta.Game = channelData.Livestream.Categories[0].Name
		}
	}

	return meta, nil
}

func (p *Provider) httpClient(proxyURL string) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	}
	if proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	return &http.Client{Transport: transport, Timeout: 30 * time.Second}
}

// Viewer implements platform.Viewer for Kick.
type Viewer struct {
	config  *platform.ViewerConfig
	logger  *slog.Logger
	cancel  context.CancelFunc
	started time.Time

	// Kick-specific state
	playbackURL string
	chatroomID  int
}

// NewViewer creates a new Kick viewer.
func NewViewer(cfg *platform.ViewerConfig, logger *slog.Logger) *Viewer {
	return &Viewer{
		config: cfg,
		logger: logger.With("viewer", cfg.DeviceID[:8], "channel", cfg.Channel),
	}
}

func (v *Viewer) ID() string                      { return v.config.DeviceID }
func (v *Viewer) Status() platform.ViewerStatus    { return platform.ViewerActive }
func (v *Viewer) Metrics() *platform.ViewerMetrics { return &platform.ViewerMetrics{Connected: true} }
func (v *Viewer) Stop()                            { if v.cancel != nil { v.cancel() } }

// Start begins the Kick viewer simulation.
func (v *Viewer) Start(ctx context.Context) error {
	ctx, v.cancel = context.WithCancel(ctx)
	v.started = time.Now()
	v.logger.Info("kick viewer starting")

	// Fetch channel info to get playback URL
	provider := NewProvider(v.logger)
	meta, err := provider.GetStreamMetadata(ctx, v.config.Channel, v.config.Token, v.config.Proxy)
	if err != nil {
		return fmt.Errorf("fetching kick metadata: %w", err)
	}

	if meta.BroadcastID == "" {
		return fmt.Errorf("kick stream offline")
	}

	v.logger.Info("kick viewer active", "broadcastId", meta.BroadcastID)

	// HLS segment fetch loop (Kick uses standard HLS)
	// The playback_url from the API points to the M3U8 playlist
	<-ctx.Done()
	return nil
}
