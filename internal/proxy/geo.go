// Package proxy - IP geolocation enrichment for proxy country/region tagging.
package proxy

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

// GeoResult holds geolocation data for a proxy IP.
type GeoResult struct {
	IP       string `json:"ip"`
	Country  string `json:"country"`
	CountryCode string `json:"countryCode"`
	Region   string `json:"regionName"`
	City     string `json:"city"`
	ISP      string `json:"isp"`
}

// GeoEnricher fetches geolocation data for proxies and enriches them with country info.
type GeoEnricher struct {
	logger      *slog.Logger
	client      *http.Client
	concurrency int
	cache       map[string]*GeoResult
	cacheMu     sync.RWMutex
}

// NewGeoEnricher creates a geo-enricher.
func NewGeoEnricher(logger *slog.Logger) *GeoEnricher {
	return &GeoEnricher{
		logger:      logger.With("component", "proxy-geo"),
		client:      &http.Client{Timeout: 10 * time.Second},
		concurrency: 10,
		cache:       make(map[string]*GeoResult),
	}
}

// EnrichAll fetches country data for all proxies in the manager.
func (g *GeoEnricher) EnrichAll(ctx context.Context, mgr *Manager) {
	proxies := mgr.All()
	g.logger.Info("enriching proxy locations", "count", len(proxies))

	var wg sync.WaitGroup
	sem := make(chan struct{}, g.concurrency)

	for _, p := range proxies {
		if p.Country != "" {
			continue // Already enriched
		}

		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(proxy *Proxy) {
			defer func() {
				<-sem
				wg.Done()
			}()

			geo, err := g.lookup(ctx, proxy.Host)
			if err != nil {
				return
			}

			mgr.mu.Lock()
			proxy.Country = geo.CountryCode
			mgr.mu.Unlock()

			// Rate limit: free API allows 45 requests per minute
			time.Sleep(1400 * time.Millisecond)
		}(p)
	}

	wg.Wait()
	g.logger.Info("proxy geo-enrichment complete")
}

// Lookup fetches geolocation for a single IP address.
func (g *GeoEnricher) Lookup(ctx context.Context, ip string) (*GeoResult, error) {
	return g.lookup(ctx, ip)
}

func (g *GeoEnricher) lookup(ctx context.Context, ip string) (*GeoResult, error) {
	// Check cache
	g.cacheMu.RLock()
	if cached, ok := g.cache[ip]; ok {
		g.cacheMu.RUnlock()
		return cached, nil
	}
	g.cacheMu.RUnlock()

	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=ip,country,countryCode,regionName,city,isp", ip)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var geo GeoResult
	if err := json.Unmarshal(body, &geo); err != nil {
		return nil, err
	}

	// Cache result
	g.cacheMu.Lock()
	g.cache[ip] = &geo
	g.cacheMu.Unlock()

	return &geo, nil
}

// CountryCodeToFlag converts a 2-letter country code to a flag emoji.
func CountryCodeToFlag(code string) string {
	if len(code) != 2 {
		return "🌍"
	}
	// Unicode regional indicator symbols: A=0x1F1E6, B=0x1F1E7, etc.
	first := rune(code[0]-'A') + 0x1F1E6
	second := rune(code[1]-'A') + 0x1F1E6
	return string([]rune{first, second})
}

// GetCountryStats returns a map of country codes to proxy counts.
func GetCountryStats(mgr *Manager) map[string]int {
	counts := make(map[string]int)
	for _, p := range mgr.All() {
		if p.Country != "" {
			counts[p.Country]++
		} else {
			counts["??"]++
		}
	}
	return counts
}
