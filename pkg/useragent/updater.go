// Package useragent - automatic user-agent string updater.
// Fetches latest browser version numbers and generates realistic UA strings.
package useragent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Updater fetches the latest browser versions and updates the UA pool.
type Updater struct {
	pool   *Pool
	client *http.Client
	logger *slog.Logger
	mu     sync.Mutex
}

// BrowserVersions holds the latest version numbers for major browsers.
type BrowserVersions struct {
	Chrome  string `json:"chrome"`
	Firefox string `json:"firefox"`
	Safari  string `json:"safari"`
	Edge    string `json:"edge"`
	Updated time.Time `json:"updated"`
}

// NewUpdater creates a user-agent updater.
func NewUpdater(pool *Pool, logger *slog.Logger) *Updater {
	return &Updater{
		pool:   pool,
		client: &http.Client{Timeout: 15 * time.Second},
		logger: logger,
	}
}

// Update fetches latest browser versions and regenerates the UA pool.
func (u *Updater) Update(ctx context.Context) (*BrowserVersions, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	versions, err := u.fetchLatestVersions(ctx)
	if err != nil {
		// Fall back to hardcoded recent versions
		versions = &BrowserVersions{
			Chrome:  "131.0.0.0",
			Firefox: "134.0",
			Safari:  "18.2",
			Edge:    "131.0.0.0",
			Updated: time.Now(),
		}
		u.logger.Warn("using fallback browser versions", "error", err)
	}

	agents := generateFromVersions(versions)

	u.pool.mu.Lock()
	u.pool.agents = agents
	u.pool.mu.Unlock()

	u.logger.Info("user agents updated",
		"count", len(agents),
		"chrome", versions.Chrome,
		"firefox", versions.Firefox,
	)

	return versions, nil
}

// AutoUpdate runs the updater periodically.
func (u *Updater) AutoUpdate(ctx context.Context, interval time.Duration) {
	// Update immediately
	u.Update(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			u.Update(ctx)
		}
	}
}

// fetchLatestVersions fetches current browser version numbers from public APIs.
func (u *Updater) fetchLatestVersions(ctx context.Context) (*BrowserVersions, error) {
	// Use the Chrome for Testing API for Chrome version
	chromeVersion, err := u.fetchChromeVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("chrome version: %w", err)
	}

	versions := &BrowserVersions{
		Chrome:  chromeVersion,
		Firefox: estimateFirefoxVersion(chromeVersion),
		Safari:  "18.2",
		Edge:    chromeVersion, // Edge follows Chrome versioning
		Updated: time.Now(),
	}

	return versions, nil
}

// fetchChromeVersion gets the latest stable Chrome version.
func (u *Updater) fetchChromeVersion(ctx context.Context) (string, error) {
	apiURL := "https://googlechromelabs.github.io/chrome-for-testing/last-known-good-versions.json"

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data struct {
		Channels struct {
			Stable struct {
				Version string `json:"version"`
			} `json:"Stable"`
		} `json:"channels"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	return data.Channels.Stable.Version, nil
}

// estimateFirefoxVersion estimates Firefox version from Chrome version.
// Firefox typically trails Chrome by ~3 version numbers.
func estimateFirefoxVersion(chromeVersion string) string {
	var major int
	fmt.Sscanf(chromeVersion, "%d.", &major)
	// Firefox major ≈ Chrome major + 3
	return fmt.Sprintf("%d.0", major+3)
}

// generateFromVersions creates a full set of UA strings from browser versions.
func generateFromVersions(v *BrowserVersions) []string {
	osVariants := []struct {
		os     string
		detail string
	}{
		{"Windows NT 10.0; Win64; x64", "Windows"},
		{"Macintosh; Intel Mac OS X 10_15_7", "macOS"},
		{"X11; Linux x86_64", "Linux"},
		{"X11; Ubuntu; Linux x86_64", "Ubuntu"},
	}

	var agents []string

	for _, os := range osVariants {
		// Chrome
		agents = append(agents,
			fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", os.os, v.Chrome),
		)

		// Firefox
		agents = append(agents,
			fmt.Sprintf("Mozilla/5.0 (%s; rv:%s) Gecko/20100101 Firefox/%s", os.os, v.Firefox, v.Firefox),
		)

		// Edge (Windows only)
		if os.detail == "Windows" {
			agents = append(agents,
				fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36 Edg/%s", os.os, v.Chrome, v.Edge),
			)
		}
	}

	// Safari (macOS only)
	agents = append(agents,
		fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%s Safari/605.1.15", v.Safari),
	)

	return agents
}
