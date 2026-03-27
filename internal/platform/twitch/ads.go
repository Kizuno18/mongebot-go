// Package twitch - ad detection and watching via HLS playlist parsing.
package twitch

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

// AdWatcher monitors HLS playlists for stitched ads and fetches their segments.
type AdWatcher struct {
	client    *http.Client
	token     string
	weaverURL string
	userAgent string
	logger    *slog.Logger

	// Metrics
	totalAds     atomic.Int64
	totalGets    atomic.Int64
	processedIDs map[string]bool
}

// NewAdWatcher creates a new ad watcher for the given stream.
func NewAdWatcher(client *http.Client, token, weaverURL, userAgent string, logger *slog.Logger) *AdWatcher {
	return &AdWatcher{
		client:       client,
		token:        token,
		weaverURL:    weaverURL,
		userAgent:    userAgent,
		logger:       logger.With("subsystem", "ads"),
		processedIDs: make(map[string]bool),
	}
}

// Watch monitors the HLS playlist for ads for up to maxDuration.
func (a *AdWatcher) Watch(ctx context.Context, maxDuration time.Duration) (adsWatched int, segmentsFetched int) {
	deadline := time.Now().Add(maxDuration)
	var previousSequence string
	var currentAdID string
	var adStartTime time.Time

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Fetch the playlist
		playlist, err := a.fetchPlaylist(ctx)
		if err != nil {
			a.logger.Debug("ad playlist fetch failed", "error", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Check if playlist has changed
		sequence := extractMediaSequence(playlist)
		if sequence == previousSequence {
			time.Sleep(2 * time.Second)
			continue
		}
		previousSequence = sequence

		// Scan for stitched ad markers
		adID, rollType := detectStitchedAd(playlist)
		if adID == "" {
			// No ad in current playlist
			if currentAdID != "" {
				// Ad just ended
				a.totalAds.Add(1)
				a.logger.Info("ad completed",
					"adId", currentAdID,
					"totalAds", a.totalAds.Load(),
				)
				currentAdID = ""
				return adsWatched + 1, segmentsFetched
			}
			time.Sleep(2 * time.Second)
			continue
		}

		// New ad detected
		if !a.processedIDs[adID] {
			a.processedIDs[adID] = true
			currentAdID = adID
			adStartTime = time.Now()
			adsWatched++
			a.logger.Debug("new ad detected", "adId", adID, "rollType", rollType)
		}

		// Fetch ad segments
		segments := extractSegmentURLs(playlist, a.weaverURL)
		for _, segURL := range segments {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if err := a.fetchSegment(ctx, segURL); err != nil {
				a.logger.Debug("ad segment fetch failed", "error", err)
				continue
			}
			segmentsFetched++
			a.totalGets.Add(1)

			elapsed := time.Since(adStartTime).Seconds()
			a.logger.Debug("watching ad",
				"rollType", rollType,
				"elapsed", fmt.Sprintf("%.0fs", elapsed),
				"adId", adID,
			)
		}

		time.Sleep(1 * time.Second)
	}

	return
}

// TotalAds returns the total number of completed ads.
func (a *AdWatcher) TotalAds() int64 {
	return a.totalAds.Load()
}

// fetchPlaylist retrieves the current HLS playlist.
func (a *AdWatcher) fetchPlaylist(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", a.weaverURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", a.userAgent)
	req.Header.Set("Authorization", "OAuth "+a.token)

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("playlist returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	return string(body), err
}

// fetchSegment downloads a single ad segment.
func (a *AdWatcher) fetchSegment(ctx context.Context, segURL string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", segURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", a.userAgent)

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(io.Discard, resp.Body)
	return err
}

// detectStitchedAd scans playlist for EXT-X-DATERANGE with CLASS=twitch-stitched-ad.
func detectStitchedAd(playlist string) (adID, rollType string) {
	for _, line := range strings.Split(playlist, "\n") {
		if !strings.HasPrefix(line, "#EXT-X-DATERANGE:") {
			continue
		}

		attrs := parseHLSAttributes(line[len("#EXT-X-DATERANGE:"):])
		if attrs["CLASS"] == "twitch-stitched-ad" {
			return attrs["ID"], attrs["X-TV-TWITCH-AD-ROLL-TYPE"]
		}
	}
	return "", ""
}

// extractMediaSequence gets the EXT-X-MEDIA-SEQUENCE value.
func extractMediaSequence(playlist string) string {
	for _, line := range strings.Split(playlist, "\n") {
		if strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:") {
			return strings.TrimPrefix(line, "#EXT-X-MEDIA-SEQUENCE:")
		}
	}
	return ""
}

// extractSegmentURLs extracts .ts segment URLs from the playlist.
func extractSegmentURLs(playlist, baseURL string) []string {
	var segments []string
	base, _ := url.Parse(baseURL)

	for _, line := range strings.Split(playlist, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if !strings.HasPrefix(line, "http") && base != nil {
			ref, err := url.Parse(line)
			if err != nil {
				continue
			}
			line = base.ResolveReference(ref).String()
		}
		segments = append(segments, line)
	}
	return segments
}

// parseHLSAttributes parses comma-separated key=value pairs from HLS tags.
func parseHLSAttributes(raw string) map[string]string {
	attrs := make(map[string]string)
	// Simple parser: split on commas not inside quotes
	var key, value string
	inQuote := false
	inValue := false

	for _, ch := range raw {
		switch {
		case ch == '"':
			inQuote = !inQuote
		case ch == '=' && !inQuote:
			inValue = true
		case ch == ',' && !inQuote:
			attrs[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
			key, value = "", ""
			inValue = false
		default:
			if inValue {
				value += string(ch)
			} else {
				key += string(ch)
			}
		}
	}
	if key != "" {
		attrs[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return attrs
}
