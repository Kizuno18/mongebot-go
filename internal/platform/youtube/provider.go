// Package youtube implements a basic YouTube platform provider for MongeBot.
// Uses YouTube's Innertube API for stream status and metadata.
package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/platform"
)

const (
	innertubeAPI    = "https://www.youtube.com/youtubei/v1/browse"
	innertubeKey    = "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
	youtubeVideoURL = "https://www.youtube.com/watch"
)

// Provider implements platform.Platform for YouTube.
type Provider struct {
	logger *slog.Logger
}

// NewProvider creates a new YouTube platform provider.
func NewProvider(logger *slog.Logger) *Provider {
	return &Provider{
		logger: logger.With("platform", "youtube"),
	}
}

func (p *Provider) Name() string { return "youtube" }

func (p *Provider) SupportedFeatures() []platform.Feature {
	return []platform.Feature{
		platform.FeatureSegments,
	}
}

func (p *Provider) Connect(ctx context.Context, cfg *platform.ViewerConfig) (platform.Viewer, error) {
	return &Viewer{
		config:  cfg,
		logger:  p.logger.With("viewer", cfg.DeviceID[:8]),
		started: time.Now(),
	}, nil
}

func (p *Provider) ValidateToken(_ context.Context, token string, _ string) (platform.TokenStatus, error) {
	if token == "" {
		return platform.TokenInvalid, nil
	}
	// YouTube uses Google OAuth — basic check only
	return platform.TokenValid, nil
}

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

// GetStreamMetadata fetches channel livestream info via page scraping.
func (p *Provider) GetStreamMetadata(ctx context.Context, channel string, token string, proxyURL string) (*platform.StreamMetadata, error) {
	client := p.httpClient(proxyURL)

	// Fetch channel live page
	channelURL := fmt.Sprintf("https://www.youtube.com/@%s/live", channel)
	req, err := http.NewRequestWithContext(ctx, "GET", channelURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching channel page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	pageContent := string(body)
	meta := &platform.StreamMetadata{}

	// Extract video ID from canonical URL or og:url
	if idx := strings.Index(pageContent, `"videoId":"`); idx != -1 {
		start := idx + len(`"videoId":"`)
		end := strings.Index(pageContent[start:], `"`)
		if end > 0 {
			meta.BroadcastID = pageContent[start : start+end]
		}
	}

	// Extract channel ID
	if idx := strings.Index(pageContent, `"channelId":"`); idx != -1 {
		start := idx + len(`"channelId":"`)
		end := strings.Index(pageContent[start:], `"`)
		if end > 0 {
			meta.ChannelID = pageContent[start : start+end]
		}
	}

	// Extract title
	if idx := strings.Index(pageContent, `"title":{"runs":[{"text":"`); idx != -1 {
		start := idx + len(`"title":{"runs":[{"text":"`)
		end := strings.Index(pageContent[start:], `"`)
		if end > 0 && end < 200 {
			meta.Title = pageContent[start : start+end]
		}
	}

	// Check if actually live (look for isLive marker)
	if !strings.Contains(pageContent, `"isLive":true`) && !strings.Contains(pageContent, `{"iconType":"LIVE"}`) {
		meta.BroadcastID = "" // Not actually live
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

// Viewer is a basic YouTube viewer stub.
type Viewer struct {
	config  *platform.ViewerConfig
	logger  *slog.Logger
	started time.Time
	cancel  context.CancelFunc
}

func (v *Viewer) ID() string                   { return v.config.DeviceID }
func (v *Viewer) Status() platform.ViewerStatus { return platform.ViewerActive }
func (v *Viewer) Stop()                         { if v.cancel != nil { v.cancel() } }

func (v *Viewer) Metrics() *platform.ViewerMetrics {
	return &platform.ViewerMetrics{
		Connected: true,
		Uptime:    time.Since(v.started),
	}
}

func (v *Viewer) Start(ctx context.Context) error {
	ctx, v.cancel = context.WithCancel(ctx)
	v.logger.Info("youtube viewer starting", "channel", v.config.Channel)

	// YouTube viewer is a stub — HLS segment fetching would be implemented here
	// YouTube uses DASH/HLS with signature-protected URLs
	<-ctx.Done()
	return nil
}

// InnertubeRequest is the payload format for YouTube's internal API.
type InnertubeRequest struct {
	Context struct {
		Client struct {
			ClientName    string `json:"clientName"`
			ClientVersion string `json:"clientVersion"`
			HL            string `json:"hl"`
			GL            string `json:"gl"`
		} `json:"client"`
	} `json:"context"`
	BrowseID string `json:"browseId,omitempty"`
	Params   string `json:"params,omitempty"`
}

// NewInnertubeRequest creates a standard Innertube API request payload.
func NewInnertubeRequest() *InnertubeRequest {
	req := &InnertubeRequest{}
	req.Context.Client.ClientName = "WEB"
	req.Context.Client.ClientVersion = "2.20260101.01.00"
	req.Context.Client.HL = "en"
	req.Context.Client.GL = "US"
	return req
}

// FetchLiveChat uses Innertube API to get live chat messages (future use).
func FetchLiveChat(ctx context.Context, client *http.Client, videoID string) ([]byte, error) {
	apiURL := fmt.Sprintf("https://www.youtube.com/youtubei/v1/live_chat/get_live_chat?key=%s", innertubeKey)

	payload := NewInnertubeRequest()
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
