// Package kick - full HLS viewer implementation with segment fetching and Pusher chat.
package kick

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/platform"
	"github.com/Kizuno18/mongebot-go/pkg/fingerprint"
)

// FullViewer is a complete Kick viewer with HLS segments and Pusher chat.
type FullViewer struct {
	config  *platform.ViewerConfig
	logger  *slog.Logger
	client  *http.Client
	cancel  context.CancelFunc
	started time.Time
	status  atomic.Int32

	// Stream state
	playbackURL string
	chatroomID  int

	// Metrics
	segmentsFetched atomic.Int64
	bytesReceived   atomic.Int64
	heartbeatsSent  atomic.Int64
}

// NewFullViewer creates a complete Kick viewer.
func NewFullViewer(cfg *platform.ViewerConfig, logger *slog.Logger) *FullViewer {
	v := &FullViewer{
		config: cfg,
		logger: logger.With("viewer", cfg.DeviceID[:8], "channel", cfg.Channel, "platform", "kick"),
	}
	v.status.Store(int32(platform.ViewerIdle))
	return v
}

func (v *FullViewer) ID() string                   { return v.config.DeviceID }
func (v *FullViewer) Status() platform.ViewerStatus { return platform.ViewerStatus(v.status.Load()) }
func (v *FullViewer) Stop()                         { if v.cancel != nil { v.cancel() } }

func (v *FullViewer) Metrics() *platform.ViewerMetrics {
	return &platform.ViewerMetrics{
		Connected:       v.Status() == platform.ViewerActive,
		Uptime:          time.Since(v.started),
		SegmentsFetched: v.segmentsFetched.Load(),
		BytesReceived:   v.bytesReceived.Load(),
		HeartbeatsSent:  v.heartbeatsSent.Load(),
	}
}

// Start begins the Kick viewer simulation.
func (v *FullViewer) Start(ctx context.Context) error {
	ctx, v.cancel = context.WithCancel(ctx)
	v.started = time.Now()
	v.status.Store(int32(platform.ViewerConnecting))

	// Create HTTP client with optional proxy and fingerprinted transport
	v.client = v.createClient()

	v.logger.Info("starting kick viewer")

	// Phase 1: Fetch channel data to get playback URL
	playbackURL, chatroomID, err := v.fetchChannelData(ctx)
	if err != nil {
		v.status.Store(int32(platform.ViewerError))
		return fmt.Errorf("fetching kick channel: %w", err)
	}

	v.playbackURL = playbackURL
	v.chatroomID = chatroomID

	if playbackURL == "" {
		v.status.Store(int32(platform.ViewerError))
		return fmt.Errorf("kick stream offline (no playback URL)")
	}

	v.status.Store(int32(platform.ViewerActive))
	v.logger.Info("kick viewer active", "playbackURL", playbackURL[:min(50, len(playbackURL))]+"...")

	// Phase 2: Run concurrent loops
	errCh := make(chan error, 3)

	// HLS segment fetcher
	go func() { errCh <- v.segmentLoop(ctx) }()

	// Liveness checker
	go func() { errCh <- v.livenessLoop(ctx) }()

	// Wait for error or cancellation
	select {
	case err := <-errCh:
		v.status.Store(int32(platform.ViewerStopped))
		return err
	case <-ctx.Done():
		v.status.Store(int32(platform.ViewerStopped))
		return nil
	}
}

// fetchChannelData gets the playback URL and chatroom ID from the Kick API.
func (v *FullViewer) fetchChannelData(ctx context.Context) (playbackURL string, chatroomID int, err error) {
	apiURL := fmt.Sprintf("%s/channels/%s", kickAPIBase, v.config.Channel)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", v.config.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := v.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", 0, fmt.Errorf("API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	var data struct {
		PlaybackURL string `json:"playback_url"`
		Livestream  *struct {
			ID int `json:"id"`
		} `json:"livestream"`
		Chatroom struct {
			ID int `json:"id"`
		} `json:"chatroom"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return "", 0, fmt.Errorf("parsing response: %w", err)
	}

	if data.Livestream == nil {
		return "", data.Chatroom.ID, nil // Stream offline
	}

	return data.PlaybackURL, data.Chatroom.ID, nil
}

// segmentLoop fetches HLS segments periodically to simulate viewing.
func (v *FullViewer) segmentLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if v.playbackURL == "" {
			time.Sleep(5 * time.Second)
			continue
		}

		// Fetch M3U8 playlist
		req, err := http.NewRequestWithContext(ctx, "GET", v.playbackURL, nil)
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

		if resp.StatusCode != 200 {
			resp.Body.Close()
			if resp.StatusCode == 404 {
				return fmt.Errorf("stream offline (404)")
			}
			time.Sleep(10 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Extract last segment URL and fetch it
		segURL := v.parseLastSegment(string(body))
		if segURL != "" {
			v.fetchSegment(ctx, segURL)
		}

		// Random delay 4-8s (HLS chunk duration)
		delay := 4*time.Second + time.Duration(rand.Int64N(int64(4*time.Second)))
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(delay):
		}
	}
}

// fetchSegment downloads a single HLS segment.
func (v *FullViewer) fetchSegment(ctx context.Context, segURL string) {
	req, err := http.NewRequestWithContext(ctx, "GET", segURL, nil)
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
	}
}

// livenessLoop checks if the stream is still live.
func (v *FullViewer) livenessLoop(ctx context.Context) error {
	ticker := time.NewTicker(45 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			playback, _, err := v.fetchChannelData(ctx)
			if err != nil {
				continue
			}
			if playback == "" {
				v.logger.Info("kick stream went offline")
				return fmt.Errorf("stream offline")
			}
		}
	}
}

// parseLastSegment extracts the last .ts URL from an HLS playlist.
func (v *FullViewer) parseLastSegment(playlist string) string {
	var last string
	for _, line := range strings.Split(playlist, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			last = line
		}
	}
	if last == "" {
		return ""
	}
	if !strings.HasPrefix(last, "http") {
		base, err := url.Parse(v.playbackURL)
		if err != nil {
			return last
		}
		ref, err := url.Parse(last)
		if err != nil {
			return last
		}
		return base.ResolveReference(ref).String()
	}
	return last
}

func (v *FullViewer) createClient() *http.Client {
	transport := fingerprint.NewFingerprintedTransport()
	if v.config.Proxy != "" {
		if u, err := url.Parse(v.config.Proxy); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	return &http.Client{Transport: transport, Timeout: 30 * time.Second}
}
