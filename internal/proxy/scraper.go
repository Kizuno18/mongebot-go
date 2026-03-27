// Package proxy - automated proxy scraping from public free proxy APIs.
package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Scraper fetches free proxies from public API sources.
type Scraper struct {
	logger  *slog.Logger
	client  *http.Client
	sources []ProxySource
}

// ProxySource defines a public proxy API endpoint.
type ProxySource struct {
	Name   string
	URL    string
	Parser func(body []byte) ([]*Proxy, error)
}

// NewScraper creates a proxy scraper with default sources.
func NewScraper(logger *slog.Logger) *Scraper {
	return &Scraper{
		logger: logger.With("component", "proxy-scraper"),
		client: &http.Client{Timeout: 30 * time.Second},
		sources: defaultSources(),
	}
}

// Scrape fetches proxies from all sources and returns the combined results.
func (s *Scraper) Scrape(ctx context.Context) ([]*Proxy, error) {
	var allProxies []*Proxy
	seen := make(map[string]bool)

	for _, source := range s.sources {
		s.logger.Info("scraping proxy source", "source", source.Name)

		proxies, err := s.scrapeSource(ctx, source)
		if err != nil {
			s.logger.Warn("source scrape failed", "source", source.Name, "error", err)
			continue
		}

		for _, p := range proxies {
			key := p.Host + ":" + p.Port
			if !seen[key] {
				seen[key] = true
				allProxies = append(allProxies, p)
			}
		}

		s.logger.Info("source scraped", "source", source.Name, "found", len(proxies))
	}

	s.logger.Info("scraping complete", "total", len(allProxies))
	return allProxies, nil
}

// ScrapeAndImport fetches proxies and adds them to the manager.
func (s *Scraper) ScrapeAndImport(ctx context.Context, mgr *Manager) (added int, err error) {
	proxies, err := s.Scrape(ctx)
	if err != nil {
		return 0, err
	}

	var rawProxies []string
	for _, p := range proxies {
		rawProxies = append(rawProxies, p.Raw())
	}

	added, _ = mgr.AddBulk(rawProxies)
	return added, nil
}

// scrapeSource fetches and parses proxies from a single source.
func (s *Scraper) scrapeSource(ctx context.Context, source ProxySource) ([]*Proxy, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", source.URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return source.Parser(body)
}

// defaultSources returns the built-in public proxy API list.
func defaultSources() []ProxySource {
	return []ProxySource{
		{
			Name: "ProxyScrape (HTTP)",
			URL:  "https://api.proxyscrape.com/v4/free-proxy-list/get?request=display_proxies&proxy_format=protocolipport&format=text&protocol=http&timeout=10000",
			Parser: parseTextList,
		},
		{
			Name: "ProxyScrape (SOCKS5)",
			URL:  "https://api.proxyscrape.com/v4/free-proxy-list/get?request=display_proxies&proxy_format=protocolipport&format=text&protocol=socks5&timeout=10000",
			Parser: parseTextList,
		},
		{
			Name: "GeoNode",
			URL:  "https://proxylist.geonode.com/api/proxy-list?limit=200&page=1&sort_by=lastChecked&sort_type=desc",
			Parser: parseGeoNode,
		},
	}
}

// parseTextList parses a plain text proxy list (one per line, format: protocol://ip:port).
func parseTextList(body []byte) ([]*Proxy, error) {
	var proxies []*Proxy
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		p, err := ParseProxy(line)
		if err != nil {
			continue
		}
		proxies = append(proxies, p)
	}
	return proxies, nil
}

// parseGeoNode parses the GeoNode API JSON response.
func parseGeoNode(body []byte) ([]*Proxy, error) {
	var resp struct {
		Data []struct {
			IP        string   `json:"ip"`
			Port      string   `json:"port"`
			Protocols []string `json:"protocols"`
			Country   string   `json:"country"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var proxies []*Proxy
	for _, item := range resp.Data {
		proxyType := ProxyHTTP
		if len(item.Protocols) > 0 {
			switch item.Protocols[0] {
			case "socks4":
				proxyType = ProxySOCKS4
			case "socks5":
				proxyType = ProxySOCKS5
			}
		}

		proxies = append(proxies, &Proxy{
			Host:    item.IP,
			Port:    item.Port,
			Type:    proxyType,
			Health:  HealthUnknown,
			Country: item.Country,
		})
	}
	return proxies, nil
}
