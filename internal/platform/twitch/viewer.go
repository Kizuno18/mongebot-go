// Package twitch - TwitchViewer implements the platform.Viewer interface.
// Simulates a real Twitch viewer with HLS segments, Spade events, GQL pulses,
// PubSub WebSocket, and IRC Chat.
package twitch

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/platform"
	"github.com/Kizuno18/mongebot-go/pkg/fingerprint"
)

// Viewer simulates a single Twitch viewer connection.
type Viewer struct {
	mu     sync.RWMutex
	config *platform.ViewerConfig
	logger *slog.Logger

	// Connection state
	client      *http.Client
	gql         *gqlClient
	status      atomic.Int32
	cancel      context.CancelFunc
	startedAt   time.Time
	broadcastID string
	channelID   string
	viewerID    string
	weaverURL   string
	spadeURL    string

	// Metrics (atomic for lock-free reads)
	segmentsFetched atomic.Int64
	bytesReceived   atomic.Int64
	heartbeatsSent  atomic.Int64
	adsWatched      atomic.Int64
	lastActivity    atomic.Value // time.Time
	lastError       atomic.Value // string
}

// NewViewer creates a new TwitchViewer.
func NewViewer(cfg *platform.ViewerConfig, logger *slog.Logger) *Viewer {
	v := &Viewer{
		config: cfg,
		logger: logger.With("viewer", cfg.DeviceID[:8], "channel", cfg.Channel),
	}
	v.status.Store(int32(platform.ViewerIdle))
	v.lastActivity.Store(time.Now())
	return v
}

// ID returns the viewer's device ID as its unique identifier.
func (v *Viewer) ID() string {
	return v.config.DeviceID
}

// Status returns the current viewer status.
func (v *Viewer) Status() platform.ViewerStatus {
	return platform.ViewerStatus(v.status.Load())
}

// Metrics returns a snapshot of the viewer's metrics.
func (v *Viewer) Metrics() *platform.ViewerMetrics {
	var uptime time.Duration
	if !v.startedAt.IsZero() {
		uptime = time.Since(v.startedAt)
	}

	lastErr := ""
	if e, ok := v.lastError.Load().(string); ok {
		lastErr = e
	}

	lastAct := time.Now()
	if t, ok := v.lastActivity.Load().(time.Time); ok {
		lastAct = t
	}

	return &platform.ViewerMetrics{
		Connected:       v.Status() == platform.ViewerActive,
		Uptime:          uptime,
		SegmentsFetched: v.segmentsFetched.Load(),
		BytesReceived:   v.bytesReceived.Load(),
		HeartbeatsSent:  v.heartbeatsSent.Load(),
		AdsWatched:      v.adsWatched.Load(),
		LastError:       lastErr,
		LastActivity:    lastAct,
	}
}

// Start begins the viewer simulation. Blocks until context is cancelled or error.
func (v *Viewer) Start(ctx context.Context) error {
	ctx, v.cancel = context.WithCancel(ctx)
	v.startedAt = time.Now()
	v.setStatus(platform.ViewerConnecting)

	// Setup HTTP client with proxy
	v.client = v.createHTTPClient()
	v.gql = newGQLClient(v.client, v.config.Token, v.config.UserAgent, v.config.DeviceID)

	v.logger.Info("starting viewer connection")

	// Phase 1: Fetch metadata
	if err := v.fetchMetadata(ctx); err != nil {
		v.setError(err)
		return fmt.Errorf("metadata fetch failed: %w", err)
	}

	// Phase 2: Get stream token and HLS playlist
	if err := v.setupHLS(ctx); err != nil {
		v.setError(err)
		return fmt.Errorf("HLS setup failed: %w", err)
	}

	v.setStatus(platform.ViewerActive)
	v.logger.Info("viewer active", "broadcastId", v.broadcastID, "channelId", v.channelID)

	// Phase 3: Send initial Spade event
	if err := v.sendSpadeEvent(ctx, "video-play"); err != nil {
		v.logger.Warn("initial spade event failed", "error", err)
	}

	// Phase 4: Run concurrent loops
	errCh := make(chan error, 8)

	// Heartbeat loop (minute-watched events)
	go func() { errCh <- v.heartbeatLoop(ctx) }()

	// HLS segment fetcher
	go func() { errCh <- v.segmentFetcherLoop(ctx) }()

	// GQL pulse loop
	go func() { errCh <- v.gqlPulseLoop(ctx) }()

	// Stream liveness checker (HEAD requests)
	go func() { errCh <- v.livenessLoop(ctx) }()

	// PubSub WebSocket (ad detection + stream events)
	go func() {
		pubsub := NewPubSubClient(v.config.Token, v.channelID, v.logger)
		pubsub.OnAd(func() {
			v.logger.Debug("ad event received, starting ad watcher")
			v.mu.RLock()
			weaverURL := v.weaverURL
			v.mu.RUnlock()
			watcher := NewAdWatcher(v.client, v.config.Token, weaverURL, v.config.UserAgent, v.logger)
			ads, segs := watcher.Watch(ctx, 5*time.Minute)
			if ads > 0 {
				v.adsWatched.Add(int64(ads))
				v.segmentsFetched.Add(int64(segs))
				v.logger.Info("ad watching complete", "adsWatched", ads, "segments", segs)
			}
		})
		errCh <- pubsub.Connect(ctx, v.config.Proxy)
	}()

	// IRC Chat WebSocket (channel presence)
	go func() {
		chat := NewChatClient(v.config.Token, v.config.Channel, v.logger)
		errCh <- chat.Connect(ctx)
	}()

	// Auto-claim channel points bonus (runs in background)
	go func() {
		claimer := NewPointsClaimer(v.client, v.config.Token, v.channelID, PointsAutoClaimConfig{
			Enabled:  true,
			Interval: 5 * time.Minute,
		}, v.logger)
		errCh <- claimer.Run(ctx)
	}()

	// Wait for any loop to return an error or context cancellation
	select {
	case err := <-errCh:
		v.setStatus(platform.ViewerStopped)
		if err != nil {
			v.setError(err)
			return err
		}
		return nil
	case <-ctx.Done():
		v.setStatus(platform.ViewerStopped)
		return ctx.Err()
	}
}

// Stop gracefully stops the viewer.
func (v *Viewer) Stop() {
	v.logger.Info("stopping viewer")
	if v.cancel != nil {
		v.cancel()
	}
	v.setStatus(platform.ViewerStopped)
}

// fetchMetadata gets broadcast ID, channel ID, and viewer ID.
func (v *Viewer) fetchMetadata(ctx context.Context) error {
	broadcastID, channelID, err := v.gql.getStreamMetadata(ctx, v.config.Channel)
	if err != nil {
		return fmt.Errorf("stream metadata: %w", err)
	}

	v.mu.Lock()
	v.broadcastID = broadcastID
	v.channelID = channelID
	v.mu.Unlock()

	if broadcastID == "" {
		return fmt.Errorf("stream appears to be offline")
	}

	viewerID, status, err := v.gql.getAuthenticatedUserID(ctx)
	if err != nil {
		return fmt.Errorf("auth user ID: %w", err)
	}
	if status == 401 {
		return fmt.Errorf("token expired (401)")
	}

	v.mu.Lock()
	v.viewerID = viewerID
	v.mu.Unlock()

	return nil
}

// setupHLS fetches the stream token, M3U8 playlist, and selects the best quality stream.
func (v *Viewer) setupHLS(ctx context.Context) error {
	token, sig, status, err := v.gql.getStreamToken(ctx, v.config.Channel)
	if err != nil || token == "" || sig == "" {
		return fmt.Errorf("stream token failed (status=%d): %w", status, err)
	}

	// Fetch M3U8 master playlist
	m3u8URL := fmt.Sprintf("%s/%s.m3u8", UsherURL, v.config.Channel)
	params := url.Values{
		"player_type":                  {PlayerType},
		"player_backend":               {PlayerBackend},
		"playlist_include_framerate":   {"true"},
		"allow_source":                 {"true"},
		"transcode_mode":              {"cbr_v1"},
		"token":                        {token},
		"sig":                          {sig},
		"player_version":              {PlayerVersion},
	}

	fullURL := m3u8URL + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", v.config.UserAgent)

	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("M3U8 fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("M3U8 returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading M3U8: %w", err)
	}

	// Parse M3U8 and find best quality stream URL
	weaverURL := parseBestStreamURL(string(body))
	if weaverURL == "" {
		return fmt.Errorf("no streams found in M3U8 playlist")
	}

	v.mu.Lock()
	v.weaverURL = weaverURL
	v.mu.Unlock()

	v.logger.Debug("HLS setup complete", "weaverURL", weaverURL[:50]+"...")
	return nil
}

// heartbeatLoop sends minute-watched Spade events every 60 seconds.
func (v *Viewer) heartbeatLoop(ctx context.Context) error {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := v.sendSpadeEvent(ctx, "minute-watched"); err != nil {
				v.logger.Debug("heartbeat failed", "error", err)
			} else {
				v.heartbeatsSent.Add(1)
				v.lastActivity.Store(time.Now())
			}
		}
	}
}

// segmentFetcherLoop periodically fetches HLS segments to simulate video consumption.
func (v *Viewer) segmentFetcherLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		v.mu.RLock()
		weaverURL := v.weaverURL
		v.mu.RUnlock()

		if weaverURL == "" {
			time.Sleep(5 * time.Second)
			continue
		}

		req, err := http.NewRequestWithContext(ctx, "GET", weaverURL, nil)
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}
		req.Header.Set("User-Agent", v.config.UserAgent)

		resp, err := v.client.Do(req)
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}

		if resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err == nil {
				segmentURL := parseLastSegmentURL(string(body), weaverURL)
				if segmentURL != "" {
					v.fetchSegment(ctx, segmentURL)
				}
			}
		} else {
			resp.Body.Close()
		}

		// Random delay between 4-8 seconds (simulating real HLS chunk duration)
		delay := 4*time.Second + time.Duration(rand.Int64N(int64(4*time.Second)))
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(delay):
		}
	}
}

// fetchSegment downloads a single HLS segment.
func (v *Viewer) fetchSegment(ctx context.Context, segmentURL string) {
	req, err := http.NewRequestWithContext(ctx, "GET", segmentURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", v.config.UserAgent)

	resp, err := v.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		n, _ := io.Copy(io.Discard, resp.Body)
		v.segmentsFetched.Add(1)
		v.bytesReceived.Add(n)
		v.lastActivity.Store(time.Now())
	}
}

// gqlPulseLoop sends periodic WatchTrackQuery GQL calls.
func (v *Viewer) gqlPulseLoop(ctx context.Context) error {
	for {
		// Random delay between 3-7 minutes
		delay := 3*time.Minute + time.Duration(rand.Int64N(int64(4*time.Minute)))
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(delay):
		}

		if err := v.gql.sendWatchTrackQuery(ctx, v.config.Channel); err != nil {
			v.logger.Debug("GQL pulse failed", "error", err)
		} else {
			v.logger.Debug("GQL pulse sent")
		}
	}
}

// livenessLoop checks if the stream is still live via HEAD requests.
func (v *Viewer) livenessLoop(ctx context.Context) error {
	ticker := time.NewTicker(40 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			v.mu.RLock()
			weaverURL := v.weaverURL
			v.mu.RUnlock()

			if weaverURL == "" {
				continue
			}

			req, err := http.NewRequestWithContext(ctx, "HEAD", weaverURL, nil)
			if err != nil {
				continue
			}
			req.Header.Set("User-Agent", v.config.UserAgent)

			resp, err := v.client.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()

			if resp.StatusCode == 404 {
				v.logger.Info("stream appears offline (404)")
				return fmt.Errorf("stream offline")
			}
		}
	}
}

// sendSpadeEvent sends a Spade analytics event (video-play or minute-watched).
func (v *Viewer) sendSpadeEvent(ctx context.Context, eventType string) error {
	v.mu.RLock()
	broadcastID := v.broadcastID
	channelID := v.channelID
	viewerID := v.viewerID
	v.mu.RUnlock()

	event := map[string]any{
		"event": eventType,
		"properties": map[string]any{
			"broadcast_id": broadcastID,
			"channel_id":   channelID,
			"channel":      v.config.Channel,
			"device_id":    v.config.DeviceID,
			"hidden":       false,
			"live":         true,
			"location":     "channel",
			"logged_in":    viewerID != "",
			"muted":        false,
			"player":       "site",
			"user_id":      viewerID,
			"user_agent":   v.config.UserAgent,
		},
	}

	payload, _ := json.Marshal([]any{event})
	encoded := base64.StdEncoding.EncodeToString(payload)

	// Get spade URL if not cached
	v.mu.RLock()
	spadeURL := v.spadeURL
	v.mu.RUnlock()

	if spadeURL == "" {
		fetched, err := fetchSpadeURL(ctx, v.client, v.config.Channel, v.config.UserAgent)
		if err != nil {
			return fmt.Errorf("fetching spade URL: %w", err)
		}
		v.mu.Lock()
		v.spadeURL = fetched
		spadeURL = fetched
		v.mu.Unlock()
	}

	body := "data=" + url.QueryEscape(encoded)
	req, err := http.NewRequestWithContext(ctx, "POST", spadeURL, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", v.config.UserAgent)

	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// createHTTPClient creates an http.Client with proxy and randomized TLS fingerprint.
func (v *Viewer) createHTTPClient() *http.Client {
	// Use fingerprinted transport for anti-detection
	transport := fingerprint.NewFingerprintedTransport()

	if v.config.Proxy != "" {
		if u, err := url.Parse(v.config.Proxy); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}
}

// setStatus atomically sets the viewer status.
func (v *Viewer) setStatus(s platform.ViewerStatus) {
	v.status.Store(int32(s))
}

// setError stores the last error message.
func (v *Viewer) setError(err error) {
	v.setStatus(platform.ViewerError)
	if err != nil {
		v.lastError.Store(err.Error())
		v.logger.Error("viewer error", "error", err)
	}
}

// parseBestStreamURL extracts the highest bandwidth stream URL from M3U8 content.
func parseBestStreamURL(m3u8Content string) string {
	lines := strings.Split(m3u8Content, "\n")
	var bestURL string
	var bestBandwidth int

	for i, line := range lines {
		if strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {
			// Extract BANDWIDTH value
			for _, part := range strings.Split(line, ",") {
				if strings.HasPrefix(part, "BANDWIDTH=") {
					var bw int
					fmt.Sscanf(part, "BANDWIDTH=%d", &bw)
					if bw > bestBandwidth {
						bestBandwidth = bw
						if i+1 < len(lines) {
							bestURL = strings.TrimSpace(lines[i+1])
						}
					}
				}
			}
		}
	}
	return bestURL
}

// parseLastSegmentURL extracts the last .ts segment URL from an HLS playlist.
func parseLastSegmentURL(playlistContent, baseURL string) string {
	lines := strings.Split(playlistContent, "\n")
	var lastSegment string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			lastSegment = line
		}
	}

	if lastSegment == "" {
		return ""
	}

	// Resolve relative URLs
	if !strings.HasPrefix(lastSegment, "http") {
		base, err := url.Parse(baseURL)
		if err != nil {
			return lastSegment
		}
		ref, err := url.Parse(lastSegment)
		if err != nil {
			return lastSegment
		}
		return base.ResolveReference(ref).String()
	}
	return lastSegment
}

// fetchSpadeURL gets the Twitch Spade analytics URL from the channel page.
func fetchSpadeURL(ctx context.Context, client *http.Client, channel, userAgent string) (string, error) {
	twitchURL := fmt.Sprintf("%s/%s", WebURL, channel)
	req, err := http.NewRequestWithContext(ctx, "GET", twitchURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("twitch page returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	pageContent := string(body)

	// Find settings.js URL
	settingsIdx := strings.Index(pageContent, "https://static.twitchcdn.net/config/settings")
	if settingsIdx == -1 {
		settingsIdx = strings.Index(pageContent, "https://assets.twitch.tv/config/settings")
	}
	if settingsIdx == -1 {
		return "", fmt.Errorf("settings URL not found")
	}

	// Extract the full settings URL
	endIdx := strings.Index(pageContent[settingsIdx:], ".js")
	if endIdx == -1 {
		return "", fmt.Errorf("settings URL end not found")
	}
	settingsURL := pageContent[settingsIdx : settingsIdx+endIdx+3]

	// Fetch settings.js
	req2, err := http.NewRequestWithContext(ctx, "GET", settingsURL, nil)
	if err != nil {
		return "", err
	}
	req2.Header.Set("User-Agent", userAgent)

	resp2, err := client.Do(req2)
	if err != nil {
		return "", err
	}
	defer resp2.Body.Close()

	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return "", err
	}

	// Extract spade_url
	settingsContent := string(body2)
	spadeIdx := strings.Index(settingsContent, `"spade_url":"`)
	if spadeIdx == -1 {
		return "", fmt.Errorf("spade_url not found in settings")
	}
	spadeStart := spadeIdx + len(`"spade_url":"`)
	spadeEnd := strings.Index(settingsContent[spadeStart:], `"`)
	if spadeEnd == -1 {
		return "", fmt.Errorf("spade_url end not found")
	}

	return settingsContent[spadeStart : spadeStart+spadeEnd], nil
}

// init ensures fingerprint package is used (will be expanded later).
var _ = fingerprint.GenerateDeviceID
